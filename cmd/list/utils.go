package list

import (
	"fmt"

	"github.com/byFrederick/branch-sweeper/pkg/sweeper"
)

func listBranches(options cmdOptions) {
	repoBranches := sweeper.Sweeper(
		sweeper.SweeperOptions{
			Path:       options.path,
			StaleDays:  options.staleDays,
			Merged:     options.merged,
			BaseBranch: options.baseBranch,
		},
	)

	if !options.json {
		fmt.Printf("%-40s %-15s\n", "Repository", "Branch")

		for repo, branches := range repoBranches {
			for _, branch := range branches {
				fmt.Printf("%-40s %-15s\n", repo, branch)
			}
		}
	} else {
		fmt.Println("json list")
	}
}
