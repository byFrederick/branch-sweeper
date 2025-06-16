package cmd

import (
	"os"

	"github.com/byFrederick/branch-sweeper/cmd/list"
	"github.com/byFrederick/branch-sweeper/cmd/prune"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "branch-sweeper",
	Short: "Identify and remove stale Git branches across local repositories",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(list.Cmd)
	rootCmd.AddCommand(prune.Cmd)

	rootCmd.PersistentFlags().StringP(
		"path",
		"p",
		".",
		"Directory to scan for Git repos.",
	)

	rootCmd.PersistentFlags().IntP(
		"days",
		"d",
		30,
		"Minimum days since last commit to mark a branch stale.",
	)

	rootCmd.PersistentFlags().BoolP(
		"merged",
		"m",
		false,
		"Include branches already merged into the base branch.",
	)

	rootCmd.PersistentFlags().StringP(
		"base",
		"b",
		"main",
		"Base branch name to check merges against.",
	)
}
