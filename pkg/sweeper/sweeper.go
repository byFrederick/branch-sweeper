package sweeper

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/gobwas/glob"
)

type SweeperOptions struct {
	Path       string
	StaleDays  int
	Merged     bool
	BaseBranch string
	Prune      bool
	Remote     bool
	RemoteName string
	Include    string
	Exclude    string
}

// Sweeper scans repositories in the given path and identifies branches that match the specified criteria
// It can optionally delete (prune) identified branches
func Sweeper(options SweeperOptions) ([][]string, error) {
	if options.StaleDays < 0 {
		return nil, fmt.Errorf("stale days can't be negative")
	}

	repoBranches := [][]string{}
	errs := []error{}

	err := filepath.WalkDir(options.Path, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			errs = append(errs, walkErr)
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		if d.Name() == ".git" {
			return fs.SkipDir
		}

		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			path := filepath.Join(path)

			repo, err := git.PlainOpen(path)

			if err != nil {
				errs = append(errs, fmt.Errorf("could not open repository on path %s: %w", path, err))
				return fs.SkipDir
			}

			repoName := filepath.Base(path)
			branches, err := repo.Branches()

			if err != nil {
				errs = append(errs, fmt.Errorf("%s failed to get list of branches: %w", repoName, err))
				return fs.SkipDir
			}

			baseBranch, err := findBaseBranch(repoName, branches, options.BaseBranch)

			if err != nil {
				errs = append(errs, err)
				return fs.SkipDir
			}

			// Get a new branches iterator
			branches, err = repo.Branches()

			if err != nil {
				errs = append(errs, fmt.Errorf("%s failed to get list of branches: %w", repoName, err))
				return fs.SkipDir
			}

			err = branches.ForEach(func(branch *plumbing.Reference) error {
				if branch.Name().Short() == options.BaseBranch {
					return nil
				}

				if g := glob.MustCompile(options.Exclude); options.Exclude != "" && g.Match(branch.Name().Short()) {
					return nil
				}

				if g := glob.MustCompile(options.Include); options.Include != "" && !g.Match(branch.Name().Short()) {
					return nil
				}

				staled, err := isStale(repoName, repo, branch, options.StaleDays)

				if err != nil {
					errs = append(errs, err)
					return nil
				}

				if !staled {
					return nil
				}

				if options.Merged {
					merged, err := isMerged(repoName, repo, baseBranch, branch)

					if err != nil {
						errs = append(errs, err)
						return nil
					}

					if !merged {
						return nil
					}
				}

				if options.Prune {
					if err := deleteBranch(repoName, repo, branch); err != nil {
						errs = append(errs, err)
						return nil
					}

					if options.Remote {
						if err := deleteRemoteBranch(repoName, repo, options.RemoteName, branch.Name().Short()); err != nil {
							errs = append(errs, err)
							return nil
						}
					}
				}

				repoBranches = append(repoBranches, []string{repoName, branch.Name().Short()})

				return nil
			})

			if err != nil {
				errs = append(errs, fmt.Errorf("%s failed to get list of branches: %w", repoName, err))
			}

			return fs.SkipDir
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan repositories on path: %w", err)
	}

	return repoBranches, errors.Join(errs...)
}

// baseBranch iterates through the repository branches to find and validate the specified base branch.
func findBaseBranch(repoName string, branches storer.ReferenceIter, optionsBaseBranch string) (*plumbing.Reference, error) {
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
		return false, fmt.Errorf("%s error getting branch commits log: %w", repoName, err)
	}

	// Get last commit
	commit, err := commits.Next()

	if err != nil {
		return false, fmt.Errorf("%s error getting branch last commit: %w", repoName, err)
	}

	return time.Since(commit.Author.When) >= time.Duration(staleDays)*24*time.Hour, nil
}

// isMerged checks if a branch latest commit exists in the base branch commit history
// It compares the last commit of the branch against all commits in the base branch
func isMerged(repoName string, repo *git.Repository, baseBranch *plumbing.Reference, branch *plumbing.Reference) (bool, error) {
	baseBranchCommits, err := repo.Log(&git.LogOptions{From: baseBranch.Hash()})

	if err != nil {
		return false, fmt.Errorf("%s error getting base branch commits log: %w", repoName, err)
	}

	branchCommits, err := repo.Log(&git.LogOptions{From: branch.Hash()})
	if err != nil {
		return false, fmt.Errorf("%s error getting branch commits log: %w", repoName, err)
	}

	branchLastCommit, err := branchCommits.Next()

	if err != nil {
		return false, fmt.Errorf("%s error getting branch last commit: %w", repoName, err)
	}

	var merged bool

	err = baseBranchCommits.ForEach(func(commit *object.Commit) error {
		if commit.Hash == branchLastCommit.Hash {
			merged = true
			return storer.ErrStop
		}
		return nil
	})

	if err != nil && err != storer.ErrStop {
		return false, fmt.Errorf("%s base branch commits lookup failed: %w", repoName, err)
	}

	return merged, nil
}

// deleteBranch deletes a local branch from the repository, removing both its config and reference
func deleteBranch(repoName string, repo *git.Repository, branch *plumbing.Reference) error {
	// Delete branch .git/config, if it doesn't found the branch config it ignores the error and continues
	if err := repo.DeleteBranch(branch.Name().Short()); err != nil && err != git.ErrBranchNotFound {
		return fmt.Errorf("%s failed to delete branch config %s: %w", repoName, branch.Name().Short(), err)
	}

	// Delete branch .git/refs
	if err := repo.Storer.RemoveReference(branch.Name()); err != nil {
		return fmt.Errorf("%s failed to delete branch %s: %w", repoName, branch.Name().Short(), err)
	}

	return nil
}

// deleteRemoteBranch deletes a branch from the remote repository using SSH authentication via ssh-agent
func deleteRemoteBranch(repoName string, repo *git.Repository, remoteName string, branchName string) error {
	remote, err := repo.Remote(remoteName)

	if err != nil {
		return fmt.Errorf("%s failed to get remote %s: %w", repoName, remoteName, err)
	}

	auth, err := ssh.NewSSHAgentAuth("git")

	if err != nil {
		return fmt.Errorf("%s failed to get public key from ssh-agent: %w", repoName, err)
	}

	pushOptions := &git.PushOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec(":refs/heads/" + branchName),
		},
		Auth: auth,
	}

	if err = remote.Push(pushOptions); err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("%s failed to delete remote branch: %w", repoName, err)
	}

	return nil
}
