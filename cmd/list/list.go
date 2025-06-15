package list

import (
	"github.com/spf13/cobra"
)

type cmdOptions struct {
	path       string
	staleDays  int
	merged     bool
	noRemote   bool
	baseBranch string
	json       bool
}

var Cmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List stale branches",
	Example: "branch-sweeper list --days 30 --base master --path ~/",
	Run: func(cmd *cobra.Command, args []string) {
		options := getOptions(cmd)
		listBranches(options)
	},
}

func getOptions(cmd *cobra.Command) cmdOptions {
	path, _ := cmd.Flags().GetString("path")
	days, _ := cmd.Flags().GetInt("days")
	merged, _ := cmd.Flags().GetBool("merged")
	noRemote, _ := cmd.Flags().GetBool("no-remote")
	base, _ := cmd.Flags().GetString("base")
	json, _ := cmd.Flags().GetBool("json")

	return cmdOptions{
		path:       path,
		staleDays:  days,
		merged:     merged,
		noRemote:   noRemote,
		baseBranch: base,
		json:       json,
	}
}

func init() {
	Cmd.Flags().Bool(
		"json",
		false,
		"Output the list as a JSON",
	)
}
