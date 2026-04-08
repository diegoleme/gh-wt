package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/diegoleme/gh-wt/internal/config"
	"github.com/diegoleme/gh-wt/internal/naming"
	"github.com/diegoleme/gh-wt/internal/open"
	"github.com/diegoleme/gh-wt/internal/worktree"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <issue|branch>",
	Short: "Open a worktree in a new terminal session",
	Long:  `Executes the configured open.command for the worktree matching the given issue number or branch name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if cfg.Open.Command == "" {
			return fmt.Errorf("open.command not configured in .gh-wt.yml")
		}

		worktrees, err := worktree.List()
		if err != nil {
			return err
		}

		target := args[0]
		wt, err := resolveWorktree(worktrees, target)
		if err != nil {
			return err
		}

		absPath, _ := filepath.Abs(wt.Path)
		issueNumber, _ := naming.ParseIssueNumber(wt.Branch)

		fmt.Printf("Opening %s...\n", wt.Branch)
		return open.Run(open.Opts{
			Command:      cfg.Open.Command,
			WorktreePath: absPath,
			Branch:       wt.Branch,
			IssueNumber:  issueNumber,
		})
	},
}

func resolveWorktree(worktrees []worktree.Info, target string) (*worktree.Info, error) {
	// Try as issue number first
	if issueNum, err := strconv.Atoi(target); err == nil {
		prefix := fmt.Sprintf("%d-", issueNum)
		for _, wt := range worktrees {
			if n, ok := naming.ParseIssueNumber(wt.Branch); ok && n == issueNum {
				return &wt, nil
			}
			// Also check prefix match
			if len(wt.Branch) > len(prefix) && wt.Branch[:len(prefix)] == prefix {
				return &wt, nil
			}
		}
		return nil, fmt.Errorf("no worktree found for issue #%d", issueNum)
	}

	// Try as branch name
	for _, wt := range worktrees {
		if wt.Branch == target {
			return &wt, nil
		}
	}

	return nil, fmt.Errorf("no worktree found for %q", target)
}

func init() {
	rootCmd.AddCommand(openCmd)
}
