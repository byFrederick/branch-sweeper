package list

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List stale branches",
	Example: "branch-sweeper list --days 30 --base master --path ~/",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("list called")
	},
}

func init() {
	Cmd.Flags().Bool(
		"json",
		false,
		"Output the list as a JSON",
	)
}
