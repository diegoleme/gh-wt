package cmd

import (
	"fmt"
	"os"

	"github.com/diegoleme/gh-wt/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh-wt",
	Short: "Worktree-driven development workflow for GitHub",
	Long:  `gh-wt manages the lifecycle of git worktrees linked to GitHub issues. One issue = one branch = one worktree = one PR.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := tui.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
