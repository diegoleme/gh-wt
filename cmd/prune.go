package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/diegoleme/gh-wt/internal/listutil"
	"github.com/diegoleme/gh-wt/internal/style"
	"github.com/diegoleme/gh-wt/internal/worktree"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune [issue-or-pr-number]",
	Short: "Remove worktrees for merged/closed branches, or a specific one",
	Long: `Without arguments: identifies worktrees whose branch has been merged or PR closed, and removes them after confirmation.
With an argument: removes the worktree matching the given issue or PR number.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		entries, err := listutil.BuildEntries()
		if err != nil {
			return err
		}

		// Specific issue/PR number
		if len(args) == 1 {
			num, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("argument must be a number: %s", args[0])
			}
			return pruneByNumber(entries, num, force)
		}

		// Auto-prune merged/closed
		return pruneAuto(entries, cmd)
	},
}

func pruneByNumber(entries []listutil.Entry, num int, force bool) error {
	// Find by issue number or PR number
	var target *listutil.Entry
	for _, e := range entries {
		if e.IssueNumber == num || e.PRNumber == num {
			if e.Branch != "" && e.Path != "" {
				entry := e
				target = &entry
				break
			}
		}
	}

	if target == nil {
		return fmt.Errorf("no worktree found for #%d", num)
	}

	if target.Current {
		return fmt.Errorf("cannot remove current worktree (%s)", target.Branch)
	}

	fmt.Printf("  %s  %s\n", target.Branch, style.Dim.Render(style.ShortenPath(target.Path)))

	if !force {
		fmt.Printf("\nRemove this worktree? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := worktree.Remove(target.Path, true); err != nil {
		return fmt.Errorf("failed to remove %s: %w", target.Branch, err)
	}

	delCmd := exec.Command("git", "branch", "-D", target.Branch)
	delCmd.CombinedOutput()

	fmt.Printf("\n✓ Removed %s\n", target.Branch)
	return nil
}

func pruneAuto(entries []listutil.Entry, cmd *cobra.Command) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	var candidates []listutil.Entry
	var kept []listutil.Entry
	for _, e := range entries {
		if e.Current {
			kept = append(kept, e)
			continue
		}
		if e.Merged || e.PRState == "closed" {
			candidates = append(candidates, e)
		} else {
			kept = append(kept, e)
		}
	}

	if len(candidates) == 0 {
		fmt.Println("No worktrees to prune.")
		return nil
	}

	fmt.Printf("Remove (%d):\n\n", len(candidates))
	for _, c := range candidates {
		fmt.Printf("  %s  %s  %s\n",
			style.Dim.Render(c.Branch),
			pruneReason(c),
			style.Dim.Render(style.ShortenPath(c.Path)),
		)
	}

	if len(kept) > 0 {
		fmt.Printf("\nKeep (%d):\n\n", len(kept))
		for _, k := range kept {
			fmt.Printf("  %s  %s\n",
				k.Branch,
				keepReason(k),
			)
		}
	}

	fmt.Println()

	if dryRun {
		fmt.Println("Dry run — no changes made.")
		return nil
	}

	if !force {
		fmt.Print("Remove these worktrees? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	fmt.Println()
	removed := 0
	for _, c := range candidates {
		if err := worktree.Remove(c.Path, true); err != nil {
			fmt.Printf("  ✗ %s: %s\n", c.Branch, err)
			continue
		}

		delCmd := exec.Command("git", "branch", "-D", c.Branch)
		delCmd.CombinedOutput()

		fmt.Printf("  ✓ Removed %s\n", c.Branch)
		removed++
	}

	fmt.Printf("\nPruned %d worktree(s).\n", removed)
	return nil
}

func pruneReason(e listutil.Entry) string {
	if e.PRState == "merged" {
		return style.Purple.Render(fmt.Sprintf("PR #%d merged", e.PRNumber))
	}
	if e.PRState == "closed" {
		return style.Red.Render(fmt.Sprintf("PR #%d closed", e.PRNumber))
	}
	if e.Merged {
		return style.Purple.Render("branch merged")
	}
	return style.Dim.Render("unknown")
}

func keepReason(e listutil.Entry) string {
	if e.Current {
		return style.Dim.Render("(current)")
	}
	if e.PRNumber > 0 {
		return style.Green.Render(fmt.Sprintf("PR #%d %s", e.PRNumber, e.PRState))
	}
	if e.IssueNumber > 0 {
		return style.Dim.Render(fmt.Sprintf("issue #%d", e.IssueNumber))
	}
	return style.Dim.Render("no PR")
}

func init() {
	pruneCmd.Flags().Bool("dry-run", false, "show what would be removed without removing")
	pruneCmd.Flags().Bool("force", false, "skip confirmation prompt")
	rootCmd.AddCommand(pruneCmd)
}
