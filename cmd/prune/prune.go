package prune

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "prune",
	Short: "Delete stale branches",
	// Example: "",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("prune called")
	},
}

func init() {

}
