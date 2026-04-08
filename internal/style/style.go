package style

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/diegoleme/gh-wt/internal/listutil"
)

// GitHub UI colors
var (
	Green  = lipgloss.NewStyle().Foreground(lipgloss.Color("#238636"))
	Purple = lipgloss.NewStyle().Foreground(lipgloss.Color("#8957e5"))
	Red    = lipgloss.NewStyle().Foreground(lipgloss.Color("#da3633"))
	Orange = lipgloss.NewStyle().Foreground(lipgloss.Color("#d29922"))
	Dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("#848d97"))
)

func ColorIssue(e listutil.Entry) string {
	if e.IssueNumber == 0 {
		return Dim.Render("—")
	}
	num := fmt.Sprintf("#%d", e.IssueNumber)
	switch {
	case e.IssueState == "OPEN" || e.IssueState == "open":
		return Green.Render(num)
	case (e.IssueState == "CLOSED" || e.IssueState == "closed") && e.IssueStateReason == "NOT_PLANNED":
		return Dim.Render(num)
	case e.IssueState == "CLOSED" || e.IssueState == "closed":
		return Purple.Render(num)
	default:
		return Green.Render(num)
	}
}

func ColorPR(e listutil.Entry) string {
	if e.PRNumber == 0 {
		return Dim.Render("—")
	}
	num := fmt.Sprintf("#%d", e.PRNumber)
	switch e.PRState {
	case "open":
		return Green.Render(num)
	case "draft":
		return Dim.Render(num)
	case "closed":
		return Red.Render(num)
	default:
		return Purple.Render(num)
	}
}

func ColorChecks(state string) string {
	switch state {
	case "SUCCESS":
		return Green.Render("✓")
	case "FAILURE", "ERROR":
		return Red.Render("✗")
	case "PENDING", "EXPECTED":
		return Orange.Render("⏳")
	default:
		return Dim.Render("—")
	}
}

func ColorReview(decision string) string {
	switch decision {
	case "APPROVED":
		return Green.Render("✓")
	case "CHANGES_REQUESTED":
		return Orange.Render("△")
	case "REVIEW_REQUIRED":
		return Dim.Render("⏳")
	default:
		return Dim.Render("—")
	}
}

func PadRight(display string, targetWidth int) string {
	visible := lipgloss.Width(display)
	if visible >= targetWidth {
		return display
	}
	return display + strings.Repeat(" ", targetWidth-visible)
}

func ShortenPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}
