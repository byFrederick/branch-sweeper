package sweeper

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

type SweeperOptions struct {
	Path       string
	StaleDays  int
	Merged     bool
	BaseBranch string
	Prune      bool
	Remote     bool
	RemoteName string
}

// Sweeper scans repositories in the given path and identifies branches that match the specified criteria
// It can optionally delete (prune) identified branches
func Sweeper(options SweeperOptions) ([][]string, error) {
	repoBranches := [][]string{}

	err := filepath.WalkDir(options.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		if d.Name() == ".git" {
			return fs.SkipDir
		}

		_, err = os.Stat(filepath.Join(path, ".git"))

		if err == nil {
			path := filepath.Join(path)

			repo, err := git.PlainOpen(path)

			if err != nil {
				log.Errorf("Could not open repository %s", path)
				return fs.SkipDir
			}

			repoName := filepath.Base(path)
			branches, err := repo.Branches()

			if err != nil {
				return fmt.Errorf("%s failed to get list of branches: %v", repo, err)
			}

			baseBranch, err := baseBranch(repoName, branches, options.BaseBranch)

			if err != nil {
				log.Fatal(err.Error())
			}

			// Get a new branches iterator
			branches, err = repo.Branches()

			if err != nil {
				return fmt.Errorf("%s failed to get list of branches: %v", repo, err)
			}

			err = branches.ForEach(func(branch *plumbing.Reference) error {
				if branch.Name().Short() == options.BaseBranch {
					return nil
				}

				staled, err := isStale(repoName, repo, branch, options.StaleDays)

				if err != nil {
					return err
				}

				if !staled {
					return nil
				}

				if options.Merged {
					merged, err := isMerged(repoName, repo, baseBranch, branch)

					if err != nil {
						return err
					}

					if !merged {
						return nil
					}
				}

				if options.Prune {
					err := deleteBranch(repoName, repo, branch)

					if err != nil {
						return err
					}

					if options.Remote {
						deleteRemoteBranch(repoName, repo, options.RemoteName, branch.Name().Short())
					}
				}

				repoBranches = append(repoBranches, []string{repoName, branch.Name().Short()})

				return nil
			})

			if err != nil {
				return err
			}

			return fs.SkipDir
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan repositories on path: %v", err)
	}

	return repoBranches, nil
}

// baseBranch iterates through the repository branches to find and validate the specified base branch.
func baseBranch(repoName string, branches storer.ReferenceIter, optionsBaseBranch string) (*plumbing.Reference, error) {
	var baseBranch *plumbing.Reference

	err := branches.ForEach(func(branch *plumbing.Reference) error {
		if branch.Name().Short() == optionsBaseBranch {
			baseBranch = branch
			return storer.ErrStop
		}
		return nil
	})

	if err != nil && err != storer.ErrStop {
		return nil, fmt.Errorf("%s branch lookup failed: %w", repoName, err)
	}

	if baseBranch == nil {
		return nil, fmt.Errorf("%s base branch %q not found", repoName, optionsBaseBranch)
	}

	return baseBranch, nil
}

// isStale checks if a branch's latest commit is older than the specified number of days
func isStale(repoName string, repo *git.Repository, branch *plumbing.Reference, staleDays int) (bool, error) {
	commits, err := repo.Log(&git.LogOptions{From: branch.Hash()})

	if err != nil {
		return false, fmt.Errorf("%s error getting branch commits log: %v", repoName, err)
	}

	// Get last commit
	commit, err := commits.Next()

	if err != nil {
		return false, fmt.Errorf("%s error getting branch last commit: %v", repoName, err)
	}

	return time.Since(commit.Author.When) >= time.Duration(staleDays)*24*time.Hour, nil
}

// isMerged checks if a branch latest commit exists in the base branch commit history
// It compares the last commit of the branch against all commits in the base branch
func isMerged(repoName string, repo *git.Repository, baseBranch *plumbing.Reference, branch *plumbing.Reference) (bool, error) {
	baseBranchCommits, err := repo.Log(&git.LogOptions{From: baseBranch.Hash()})

	if err != nil {
		return false, fmt.Errorf("%s error getting base branch commits log: %v", repoName, err)
	}

	branchCommits, err := repo.Log(&git.LogOptions{From: branch.Hash()})
	if err != nil {
		return false, fmt.Errorf("%s error getting branch commits log: %v", repoName, err)
	}

	branchLastCommit, err := branchCommits.Next()

	if err != nil {
		return false, fmt.Errorf("%s error getting branch last commit: %v", repoName, err)
	}

	isMerged := false

	err = baseBranchCommits.ForEach(func(commit *object.Commit) error {
		if commit.Hash == branchLastCommit.Hash {
			isMerged = true
			return storer.ErrStop
		}
		return nil
	})

	if err != nil && err != storer.ErrStop {
		return false, fmt.Errorf("%s base branch commits lookup failed: %w", repoName, err)
	}

	return isMerged, nil
}

// deleteBranch deletes a local branch from the repository, removing both its config and reference
func deleteBranch(repoName string, repo *git.Repository, branch *plumbing.Reference) error {
	// Delete branch .git/config
	_ = repo.DeleteBranch(branch.Name().Short())

	// Delete branch .git/refs
	err := repo.Storer.RemoveReference(branch.Name())

	if err != nil {
		return fmt.Errorf("%s failed to delete branch %s: %v", repoName, branch, err)
	}

	return nil
}

// deleteRemoteBranch deletes a branch from the remote repository using SSH authentication via ssh-agent
func deleteRemoteBranch(repoName string, repo *git.Repository, remoteName string, branchName string) {
	remote, err := repo.Remote(remoteName)

	if err != nil {
		log.Fatalf("%s failed to get remote %s: %v", repoName, remoteName, err)
	}

	auth, err := ssh.NewSSHAgentAuth("git")

	if err != nil {
		log.Fatalf("%s failed to get public key from ssh-agent: %v", repoName, err)
	}

	pushOptions := &git.PushOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec(":refs/heads/" + branchName),
		},
		Auth: auth,
	}

	err = remote.Push(pushOptions)

	if err != nil {
		log.Printf("%s failed to delete remote branch: %v", repoName, err)
	}
}
