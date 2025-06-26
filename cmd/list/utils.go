package list

import (
	"fmt"

	"github.com/byFrederick/branch-sweeper/pkg/sweeper"
	"github.com/charmbracelet/log"
)

func listBranches(options cmdOptions) {
	repoBranches, err := sweeper.Sweeper(
		sweeper.SweeperOptions{
			Path:       options.path,
			StaleDays:  options.staleDays,
			Merged:     options.merged,
			BaseBranch: options.baseBranch,
		},
	)

	if len(repoBranches) > 0 {
		fmt.Printf("%-40s %-40s\n", "Repository", "Branch")

		for _, entries := range repoBranches {
			for _, entry := range entries {
				fmt.Printf("%-41s", entry)
			}
			fmt.Print("\n")
		}
	} else {
		fmt.Println("No branches found")
	}

	if err != nil {
		log.Warn(err)
	}
}
