package sweeper

import (
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

type SweeperOptions struct {
	Path       string
	StaleDays  int
	Merged     bool
	BaseBranch string
	Prune      bool
}

// Sweeper scans repositories in the given path and identifies branches that match the specified criteria
// It can optionally delete (prune) identified branches
func Sweeper(options SweeperOptions) map[string][]string {
	repoBranches := make(map[string][]string)

	filepath.WalkDir(options.Path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && d.Name() == ".git" {
			path := filepath.Join(path, "..")
			repo, err := git.PlainOpen(path)

			if err != nil {
				log.Fatalf("Could not open repository %q", path)
			}

			repoName := filepath.Base(path)
			branches, _ := repo.Branches()
			baseBranch, err := baseBranch(repoName, branches, options.BaseBranch)

			if err != nil {
				log.Fatal(err.Error())
			}

			// Get a new branches iterator
			branches, _ = repo.Branches()

			branches.ForEach(func(branch *plumbing.Reference) error {
				if branch.Name().Short() == options.BaseBranch {
					return nil
				}

				if !isStale(repo, branch.Hash(), options.StaleDays) {
					return nil
				}

				if options.Merged {
					if !isMerged(repo, baseBranch.Hash(), branch.Hash()) {
						return nil
					}
				}

				if options.Prune {
					// Delete branch .git/config
					err = repo.DeleteBranch(branch.Name().Short())

					// Delete branch .git/refs
					repo.Storer.RemoveReference(branch.Name())
				}

				repoBranches[repoName] = append(repoBranches[repoName], branch.Name().Short())
				return nil
			})
		}

		return nil
	})

	return repoBranches
}

// baseBranch iterates through the repository branches to find and validate the specified base branch.
func baseBranch(repoName string, branches storer.ReferenceIter, optionsBaseBranch string) (*plumbing.Reference, error) {
	var baseBranch *plumbing.Reference

	branches.ForEach(func(branch *plumbing.Reference) error {
		if branch.Name().Short() == optionsBaseBranch {
			baseBranch = branch
		}
		return nil
	})

	if baseBranch != nil {
		return baseBranch, nil
	}

	return nil, fmt.Errorf("%s/%s is not a valid base branch", repoName, optionsBaseBranch)
}

// isStale checks if a branch's latest commit is older than the specified number of days
func isStale(repo *git.Repository, hash plumbing.Hash, staleDays int) bool {
	commits, _ := repo.Log(&git.LogOptions{From: hash})

	// Get last commit
	commit, _ := commits.Next()

	return time.Since(commit.Author.When) >= time.Duration(staleDays)*24*time.Hour
}

// isMerged checks if a branch latest commit exists in the base branch commit history
// It compares the last commit of the branch against all commits in the base branch
func isMerged(repo *git.Repository, baseBranchHash plumbing.Hash, branchHash plumbing.Hash) bool {
	baseBranchCommits, _ := repo.Log(&git.LogOptions{From: baseBranchHash})
	branchCommits, _ := repo.Log(&git.LogOptions{From: branchHash})

	branchLastCommit, _ := branchCommits.Next()
	branchLastCommitHash := branchLastCommit.Hash

	isMerged := false

	baseBranchCommits.ForEach(func(commit *object.Commit) error {
		if commit.Hash == branchLastCommitHash {
			isMerged = true
		}
		return nil
	})

	return isMerged
}
