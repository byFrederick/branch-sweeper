package prune

import (
	"github.com/spf13/cobra"
)

type cmdOptions struct {
	path       string
	staleDays  int
	merged     bool
	baseBranch string
	include    string
	exclude    string
	remote     bool
	remoteName string
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
	include, _ := cmd.Flags().GetString("include")
	exclude, _ := cmd.Flags().GetString("exclude")
	remote, _ := cmd.Flags().GetBool("remote")
	remoteName, _ := cmd.Flags().GetString("remote-name")

	return cmdOptions{
		path:       path,
		staleDays:  days,
		merged:     merged,
		baseBranch: base,
		include:    include,
		exclude:    exclude,
		remote:     remote,
		remoteName: remoteName,
	}
}

func init() {
	Cmd.Flags().BoolP(
		"remote",
		"r",
		false,
		"Delete matching branch on the remote repository (requires your SSH public key loaded in ssh-agent for auth)",
	)

	Cmd.Flags().String(
		"remote-name",
		"origin",
		"Name of Git remote",
	)
}
