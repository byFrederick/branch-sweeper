package prune

import (
	"fmt"

	"github.com/byFrederick/branch-sweeper/pkg/sweeper"
)

func pruneBranches(options cmdOptions) {
	prunedBranches := sweeper.Sweeper(
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

	for repo, branches := range prunedBranches {
		for _, branch := range branches {
			fmt.Printf("%s/%s deleted\n", repo, branch)
		}
	}

}
