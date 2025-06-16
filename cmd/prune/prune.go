package prune

import (
	"github.com/spf13/cobra"
)

type cmdOptions struct {
	path       string
	staleDays  int
	merged     bool
	baseBranch string
}

var Cmd = &cobra.Command{
	Use:   "prune",
	Short: "Delete stale branches",
	// Example: "",
	Run: func(cmd *cobra.Command, args []string) {
		options := getOptions(cmd)
		pruneBranches(options)
	},
}

func getOptions(cmd *cobra.Command) cmdOptions {
	path, _ := cmd.Flags().GetString("path")
	days, _ := cmd.Flags().GetInt("days")
	merged, _ := cmd.Flags().GetBool("merged")
	base, _ := cmd.Flags().GetString("base")

	return cmdOptions{
		path:       path,
		staleDays:  days,
		merged:     merged,
		baseBranch: base,
	}
}

func init() {

}
