package cmd

import (
	"fmt"
	"strconv"

	"github.com/diegoleme/gh-wt/internal/pipeline"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <issue-number>",
	Short: "Create a branch and worktree for a GitHub issue",
	Long:  `Creates a branch, worktree, and links the branch to the issue. Runs the full setup pipeline: pre-start hooks, copy-ignored, post-start hooks.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		issueNumber, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("issue number must be an integer: %s", args[0])
		}

		base, _ := cmd.Flags().GetString("base")
		noLink, _ := cmd.Flags().GetBool("no-link")
		noAssign, _ := cmd.Flags().GetBool("no-assign")
		noHooks, _ := cmd.Flags().GetBool("no-hooks")

		opts := pipeline.StartOpts{
			IssueNumber: issueNumber,
			Base:        base,
			NoLink:      noLink,
			NoAssign:    noAssign,
			NoHooks:     noHooks,
		}

		return pipeline.Start(opts)
	},
}

func init() {
	startCmd.Flags().String("base", "", "base branch (default: repo default branch)")
	startCmd.Flags().Bool("no-link", false, "don't link branch to issue")
	startCmd.Flags().Bool("no-assign", false, "don't assign issue to yourself")
	startCmd.Flags().Bool("no-hooks", false, "skip hooks and copy-ignored")
	rootCmd.AddCommand(startCmd)
}
