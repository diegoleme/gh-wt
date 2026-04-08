package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/diegoleme/gh-wt/internal/listutil"
	"github.com/diegoleme/gh-wt/internal/style"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active worktrees with issue and PR context",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")

		entries, err := listutil.BuildEntries()
		if err != nil {
			return err
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(entries)
		}

		printTable(entries)
		return nil
	},
}

func printTable(entries []listutil.Entry) {
	headers := []string{"BRANCH", "ISSUE", "PR", "CI", "REVIEW", "MERGED", "PATH"}
	gap := 3

	type tableRow struct{ cols []string }
	var rows []tableRow
	for _, e := range entries {
		branch := e.Branch
		if branch == "" {
			branch = "(detached)"
		}
		if e.Current {
			branch = "* " + branch
		} else {
			branch = "  " + branch
		}

		merged := ""
		if e.Merged {
			merged = "✓"
		}

		path := style.ShortenPath(e.Path)
		if e.Merged {
			branch = style.Dim.Render(branch)
			path = style.Dim.Render(path)
		}

		rows = append(rows, tableRow{cols: []string{
			branch,
			style.ColorIssue(e),
			style.ColorPR(e),
			style.ColorChecks(e.ChecksState),
			style.ColorReview(e.ReviewDecision),
			merged,
			path,
		}})
	}

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, r := range rows {
		for i, col := range r.cols {
			w := lipgloss.Width(col)
			if w > widths[i] {
				widths[i] = w
			}
		}
	}

	var headerParts []string
	for i, h := range headers {
		headerParts = append(headerParts, style.PadRight(h, widths[i]+gap))
	}
	fmt.Println(strings.Join(headerParts, ""))

	for _, r := range rows {
		var parts []string
		for i, col := range r.cols {
			parts = append(parts, style.PadRight(col, widths[i]+gap))
		}
		fmt.Println(strings.Join(parts, ""))
	}
}

func init() {
	listCmd.Flags().Bool("json", false, "output as JSON")
	rootCmd.AddCommand(listCmd)
}
