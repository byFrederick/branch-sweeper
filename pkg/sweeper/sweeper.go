package sweeper

import (
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"time"

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
func Sweeper(options SweeperOptions) map[string][]string {
	repoBranches := make(map[string][]string)

	err := filepath.WalkDir(options.Path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && d.Name() == ".git" {
			path := filepath.Join(path, "..")
			repo, err := git.PlainOpen(path)

			if err != nil {
				log.Printf("Could not open repository %s", path)
				return nil
			}

			repoName := filepath.Base(path)
			branches, err := repo.Branches()
			checkErrGetBranches(repoName, err)

			baseBranch, err := baseBranch(repoName, branches, options.BaseBranch)

			if err != nil {
				log.Fatal(err.Error())
			}

			// Get a new branches iterator
			branches, err = repo.Branches()
			checkErrGetBranches(repoName, err)

			err = branches.ForEach(func(branch *plumbing.Reference) error {
				if branch.Name().Short() == options.BaseBranch {
					return nil
				}

				if !isStale(repoName, repo, branch, options.StaleDays) {
					return nil
				}

				if options.Merged {
					if !isMerged(repoName, repo, baseBranch, branch) {
						return nil
					}
				}

				if options.Prune {
					deleteBranch(repoName, repo, branch)

					if options.Remote {
						deleteRemoteBranch(repoName, repo, options.RemoteName, branch.Name().Short())
					}
				}

				repoBranches[repoName] = append(repoBranches[repoName], branch.Name().Short())
				return nil
			})
			checkErrGetBranches(repoName, err)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Failed to scan repositories: %v", err)
	}

	return repoBranches
}

// baseBranch iterates through the repository branches to find and validate the specified base branch.
func baseBranch(repoName string, branches storer.ReferenceIter, optionsBaseBranch string) (*plumbing.Reference, error) {
	var baseBranch *plumbing.Reference

	err := branches.ForEach(func(branch *plumbing.Reference) error {
		if branch.Name().Short() == optionsBaseBranch {
			baseBranch = branch
		}
		return nil
	})

	if baseBranch != nil {
		return baseBranch, nil
	}
	checkErrGetBranches(repoName, err)

	return nil, fmt.Errorf("%s failed to get branch: %s", repoName, optionsBaseBranch)
}

// isStale checks if a branch's latest commit is older than the specified number of days
func isStale(repoName string, repo *git.Repository, branch *plumbing.Reference, staleDays int) bool {
	commits, err := repo.Log(&git.LogOptions{From: branch.Hash()})
	checkErrGetCommit(repoName, branch.Name().Short(), err)

	// Get last commit
	commit, err := commits.Next()
	checkErrGetCommit(repoName, branch.Name().Short(), err)

	return time.Since(commit.Author.When) >= time.Duration(staleDays)*24*time.Hour
}

// isMerged checks if a branch latest commit exists in the base branch commit history
// It compares the last commit of the branch against all commits in the base branch
func isMerged(repoName string, repo *git.Repository, baseBranch *plumbing.Reference, branch *plumbing.Reference) bool {
	baseBranchCommits, err := repo.Log(&git.LogOptions{From: baseBranch.Hash()})
	checkErrGetCommit(repoName, branch.Name().Short(), err)

	branchCommits, err := repo.Log(&git.LogOptions{From: branch.Hash()})
	checkErrGetCommit(repoName, branch.Name().Short(), err)

	branchLastCommit, err := branchCommits.Next()
	checkErrGetCommit(repoName, branch.Name().Short(), err)

	isMerged := false

	err = baseBranchCommits.ForEach(func(commit *object.Commit) error {
		if commit.Hash == branchLastCommit.Hash {
			isMerged = true
		}
		return nil
	})
	checkErrGetBranches(repoName, err)

	return isMerged
}

func deleteBranch(repoName string, repo *git.Repository, branch *plumbing.Reference) {
	// Delete branch .git/config
	err := repo.DeleteBranch(branch.Name().Short())

	if err != nil {
		log.Printf("%s failed to delete branch config %s: %v", repoName, branch, err)
	}

	// Delete branch .git/refs
	err = repo.Storer.RemoveReference(branch.Name())

	if err != nil {
		log.Fatalf("%s failed to delete branch %s: %v", repoName, branch, err)
	}
}

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
		log.Printf("%s failed push to %s: %v", repoName, remoteName, err)
	}
}
