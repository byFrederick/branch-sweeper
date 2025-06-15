package sweeper

import (
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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

		// Checks if it's a git repo
		if d.IsDir() && d.Name() == ".git" {
			path := filepath.Join(path, "..")
			repo, err := git.PlainOpen(path)

			// Logs if the repo can't be opened
			if err != nil {
				log.Fatalf("Could not open repository %q", path)
			}

			branches, _ := repo.Branches()
			repoName := filepath.Base(path)

			// Iterates through all the branches
			branches.ForEach(func(branch *plumbing.Reference) error {
				branchName := strings.TrimPrefix(branch.Name().String(), "refs/heads/")

				if branchName != options.BaseBranch &&
					isStale(repo, branch.Hash(), options.StaleDays) {
					repoBranches[repoName] = append(repoBranches[repoName], branchName)
				}

				return nil
			})
		}

		return nil
	})

	return repoBranches
}

func isStale(repo *git.Repository, hash plumbing.Hash, staleDays int) bool {
	// Retrieve commits logs
	commits, _ := repo.Log(&git.LogOptions{From: hash})

	// Get first commit from the log
	commit, _ := commits.Next()

	return time.Since(commit.Author.When) >= time.Duration(staleDays)*24*time.Hour
}

// func isMerged(){}
