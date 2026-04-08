package cmd

import (
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of the current worktree",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: implement
		cmd.Println("not implemented yet")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
