package sweeper

import (
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
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
	NoRemote   bool
	BaseBranch string
}

func Sweeper(options SweeperOptions) map[string][]string {
	repoBranches := make(map[string][]string)

	// Walk specified path
	filepath.WalkDir(options.Path, func(path string, d fs.DirEntry, err error) error {

		// Identify if it's a git repo
		if d.IsDir() && d.Name() == ".git" {
			path := filepath.Join(path, "..")
			repo, err := git.PlainOpen(path)

			// Logs if the repo can't be opened
			if err != nil {
				log.Fatalf("Could not open repository %q", path)
			}

			branches, _ := repo.Branches()
			repoName := filepath.Base(path)

			// Get base branch object
			baseBranch, err := baseBranch(branches, options.BaseBranch)

			// Logs if the base branch is not valid
			if err != nil {
				log.Fatal(err.Error())
			}

			// Iterates through all the branches
			branches.ForEach(func(branch *plumbing.Reference) error {
				branchName := strings.TrimPrefix(branch.Name().String(), "refs/heads/")

				// Skip if it's the base branch
				if branchName == options.BaseBranch {
					return nil
				}

				// Check if branch is stale
				if !isStale(repo, branch.Hash(), options.StaleDays) {
					return nil
				}

				// If Merged option is enabled, check if branch is merged
				if options.Merged {
					if !isMerged(repo, baseBranch.Hash(), branch.Hash()) {
						return nil
					}
				}

				// if options.NoRemote {
				// 	if existOnRemoteRepo()
				// }

				// Add branch to the map if it passes all conditions
				repoBranches[repoName] = append(repoBranches[repoName], branchName)
				return nil
			})
		}

		return nil
	})

	return repoBranches
}

func baseBranch(branches storer.ReferenceIter, userBaseBranchName string) (*plumbing.Reference, error) {
	// Get repo base branch
	repoBaseBranch, _ := branches.Next()
	repoBaseBranchName := strings.TrimPrefix(repoBaseBranch.Name().String(), "refs/heads/")

	// Checks if repo base branch is equal to the base branch specified by the user
	if repoBaseBranchName == userBaseBranchName {
		return repoBaseBranch, nil
	}
	return nil, fmt.Errorf("specified base branch is not valid, branch specified %s and repo base branch %s",
		userBaseBranchName,
		repoBaseBranchName,
	)
}

func isStale(repo *git.Repository, hash plumbing.Hash, staleDays int) bool {
	// Retrieve commits logs
	commits, _ := repo.Log(&git.LogOptions{From: hash})

	// Get first commit from the log
	commit, _ := commits.Next()

	return time.Since(commit.Author.When) >= time.Duration(staleDays)*24*time.Hour
}

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
