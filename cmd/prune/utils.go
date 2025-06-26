package prune

import (
	"fmt"

	"github.com/byFrederick/branch-sweeper/pkg/sweeper"
	"github.com/charmbracelet/log"
)

func pruneBranches(options cmdOptions) {
	prunedBranches, err := sweeper.Sweeper(
		sweeper.SweeperOptions{
			Path:       options.path,
			StaleDays:  options.staleDays,
			Merged:     options.merged,
			BaseBranch: options.baseBranch,
			Prune:      true,
			Remote:     options.remote,
			RemoteName: options.remoteName,
		},
	)

	if len(prunedBranches) > 0 {
		for _, entries := range prunedBranches {
			fmt.Printf("%s/%s deleted\n", entries[0], entries[1])
		}
	} else {
		log.Error("No branches found, nothing to delete")
	}

	if err != nil {
		log.Warn(err)
	}
}
