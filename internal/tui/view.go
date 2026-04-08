package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/diegoleme/gh-wt/internal/listutil"
	"github.com/diegoleme/gh-wt/internal/style"
)

// Nerd Font icons
const (
	iconBranch      = "\ue0a0" //
	iconIssueOpen   = "\uf41b" //
	iconPR          = "\ue728" //
	iconCheckPass   = "\uf00c" //
	iconCheckFail   = "\uf00d" //
	iconMerged      = "\ue727" //
	iconReview      = "\uf4a6" //
	iconFolder      = "\uf07b" //
	iconPending     = "\uf252" //
)

var (
	// Card styles with left border only
	leftBorder = lipgloss.Border{
		Left: "▌",
	}

	cardStyle = lipgloss.NewStyle().
			Border(leftBorder, false, false, false, true).
			BorderForeground(lipgloss.Color("#30363d")).
			PaddingLeft(1).
			MarginLeft(2).
			MarginBottom(1)

	selectedCardStyle = lipgloss.NewStyle().
				Border(leftBorder, false, false, false, true).
				BorderForeground(lipgloss.Color("#ffffff")).
				PaddingLeft(1).
				MarginLeft(2).
				MarginBottom(1)

	mergedCardStyle = lipgloss.NewStyle().
			Border(leftBorder, false, false, false, true).
			BorderForeground(lipgloss.Color("#30363d")).
			PaddingLeft(1).
			MarginLeft(2).
			MarginBottom(1)

	titleBold      = lipgloss.NewStyle().Bold(true)
	statusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#848d97"))
	inputStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#58a6ff"))
	cursorBg       = lipgloss.NewStyle().Background(lipgloss.Color("#58a6ff")).Foreground(lipgloss.Color("#0d1117"))
	labelStyle     = lipgloss.NewStyle().Background(lipgloss.Color("#30363d")).Foreground(lipgloss.Color("#c9d1d9")).Padding(0, 1)
)

func (m Model) View() string {
	switch m.state {
	case stateLoading, stateRunning:
		return m.viewLoading()
	default:
		return m.viewMain()
	}
}

func (m Model) viewLoading() string {
	msg := "Fetching issues..."
	if m.state == stateRunning {
		msg = "Running..."
	}
	return fmt.Sprintf("\n  %s %s\n", m.spinner.View(), msg)
}

func (m Model) viewMain() string {
	statusBar := m.viewStatusBar()
	footer := m.viewFooter()

	// Count lines used by status + footer
	bottomLines := strings.Count(statusBar, "\n") + strings.Count(footer, "\n")
	cardsHeight := m.height - bottomLines - 2 // -1 top margin, -1 bottom padding
	if cardsHeight < 5 {
		cardsHeight = 5
	}

	cards := m.viewCardsWithHeight(cardsHeight)

	base := cards + statusBar + footer

	switch m.state {
	case stateInput:
		return m.overlay(base, m.viewInput())
	case stateConfirmDelete:
		return m.overlay(base, m.viewConfirm())
	case stateOutput, stateOutputDone:
		return m.overlay(base, m.viewOutput())
	}

	return base
}

func (m Model) overlay(base, dialog string) string {
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#0d1117")),
	)
}

var (
	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#c9d1d9")).
				MarginLeft(3).
				MarginTop(1).
				MarginBottom(1)
)

func sectionLabel(section string, count int) string {
	switch section {
	case "active":
		return fmt.Sprintf("IN PROGRESS (%d)", count)
	case "stale":
		return fmt.Sprintf("DONE (%d)", count)
	case "open":
		return fmt.Sprintf("TODO (%d)", count)
	default:
		return section
	}
}

func (m Model) viewCardsWithHeight(maxHeight int) string {
	if len(m.entries) == 0 {
		return "\n  No open issues found.\n\n"
	}

	// Count entries per section
	sectionCounts := make(map[string]int)
	for _, e := range m.entries {
		sectionCounts[e.Section]++
	}

	// Build all content and track which line each entry starts at
	type renderedBlock struct {
		content  string
		entryIdx int // -1 for section headers
	}
	var blocks []renderedBlock

	lastSection := ""
	for idx, e := range m.entries {
		if e.Section != lastSection {
			header := sectionHeaderStyle.Render(sectionLabel(e.Section, sectionCounts[e.Section]))
			blocks = append(blocks, renderedBlock{content: header + "\n", entryIdx: -1})
			lastSection = e.Section
		}
		sel := idx == m.cursor
		blocks = append(blocks, renderedBlock{content: m.renderCard(e, sel), entryIdx: idx})
	}

	// Flatten to lines
	var allLines []string
	for _, block := range blocks {
		lines := strings.Split(block.content, "\n")
		allLines = append(allLines, lines...)
	}

	// Apply scroll
	vpHeight := maxHeight
	start := m.scroll
	if start < 0 {
		start = 0
	}
	if start > len(allLines) {
		start = len(allLines)
	}
	end := start + vpHeight
	if end > len(allLines) {
		end = len(allLines)
	}

	visible := allLines[start:end]

	// Pad to exactly maxHeight lines
	for len(visible) < vpHeight {
		visible = append(visible, "")
	}

	return "\n" + strings.Join(visible, "\n") + "\n"
}

func relativePath(p string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return p
	}
	rel, err := filepath.Rel(cwd, p)
	if err != nil {
		return p
	}
	return rel
}

func renderLabels(labels []string) string {
	var parts []string
	for _, l := range labels {
		parts = append(parts, labelStyle.Render(l))
	}
	return strings.Join(parts, " ")
}

func (m Model) renderCard(e listutil.Entry, selected bool) string {
	cardWidth := m.width - 4
	if cardWidth < 40 {
		cardWidth = 40
	}

	const cardLines = 3
	lines := make([]string, cardLines)

	// Line 1: issue/title + labels
	lines[0] = m.cardTitle(e, selected)

	if e.HasWorktree {
		// Line 2: worktree path
		lines[1] = fmt.Sprintf("%s %s%s",
			style.Dim.Render(iconFolder),
			style.Dim.Render(relativePath(e.Path)),
			currentBadge(e),
		)
		// Line 3: PR info
		lines[2] = m.cardPRLine(e)
	} else {
		// Line 2: no worktree
		lines[1] = style.Dim.Render("no worktree")
		// Line 3: PR info (might still have a PR if branch exists without worktree — unlikely but safe)
		lines[2] = m.cardPRLine(e)
	}

	// Truncate lines to fit card width (accounting for border + padding)
	maxContentWidth := cardWidth - 3 // 1 border + 1 padding left + 1 safety
	for i, line := range lines {
		lines[i] = truncate(line, maxContentWidth)
	}

	content := strings.Join(lines, "\n")

	var cs lipgloss.Style
	switch {
	case selected:
		cs = selectedCardStyle
	case e.Merged:
		cs = mergedCardStyle
	default:
		cs = cardStyle
	}

	return cs.Width(cardWidth).Render(content) + "\n"
}

// truncate cuts a string (possibly with ANSI codes) to maxWidth visible characters.
func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	visible := lipgloss.Width(s)
	if visible <= maxWidth {
		return s
	}
	// Use ansi-aware truncation: walk runes, track visible width
	result := strings.Builder{}
	w := 0
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		if w >= maxWidth-1 {
			result.WriteString("…")
			// Close any open ANSI sequences
			result.WriteString("\x1b[0m")
			break
		}
		result.WriteRune(r)
		w++
	}
	return result.String()
}

func currentBadge(e listutil.Entry) string {
	if e.Current {
		return style.Dim.Render(" (current)")
	}
	return ""
}

func (m Model) cardTitle(e listutil.Entry, selected bool) string {
	if e.IssueNumber > 0 {
		num := style.ColorIssue(e)
		title := e.IssueTitle
		if title == "" {
			title = e.Branch
		}

		labels := ""
		if len(e.Labels) > 0 {
			labels = "  " + renderLabels(e.Labels)
		}

		if selected {
			return fmt.Sprintf("%s  %s%s", num, titleBold.Render(title), labels)
		}
		if e.Merged {
			return fmt.Sprintf("%s  %s%s", num, style.Dim.Render(title), labels)
		}
		return fmt.Sprintf("%s  %s%s", num, title, labels)
	}

	if selected {
		return titleBold.Render(e.Branch)
	}
	if e.Merged {
		return style.Dim.Render(e.Branch)
	}
	return e.Branch
}

func (m Model) cardPRLine(e listutil.Entry) string {
	if e.PRNumber == 0 {
		return style.Dim.Render("no PR")
	}

	sep := style.Dim.Render(" · ")
	var parts []string

	// PR number (colored) + state (dim)
	num := fmt.Sprintf("#%d", e.PRNumber)
	switch e.PRState {
	case "open":
		parts = append(parts, style.Green.Render(num)+" "+style.Dim.Render("open"))
	case "draft":
		parts = append(parts, style.Dim.Render(num)+" "+style.Dim.Render("draft"))
	case "merged":
		parts = append(parts, style.Purple.Render(num)+" "+style.Dim.Render("merged"))
	case "closed":
		parts = append(parts, style.Red.Render(num)+" "+style.Dim.Render("closed"))
	}

	// For merged/closed, just show the state
	if e.PRState == "merged" || e.PRState == "closed" {
		return strings.Join(parts, sep)
	}

	// Lines changed
	if e.PRAdditions > 0 || e.PRDeletions > 0 {
		changes := fmt.Sprintf("%s %s",
			style.Green.Render(fmt.Sprintf("+%d", e.PRAdditions)),
			style.Red.Render(fmt.Sprintf("-%d", e.PRDeletions)),
		)
		parts = append(parts, changes)
	}

	// CI
	if e.ChecksState != "" {
		switch e.ChecksState {
		case "SUCCESS":
			parts = append(parts, "ci "+style.Green.Render("✓"))
		case "FAILURE", "ERROR":
			parts = append(parts, "ci "+style.Red.Render("✗"))
		case "PENDING", "EXPECTED":
			parts = append(parts, "ci "+style.Orange.Render("●"))
		}
	}

	// Review
	if e.ReviewDecision != "" {
		switch e.ReviewDecision {
		case "APPROVED":
			parts = append(parts, "review "+style.Green.Render("✓"))
		case "CHANGES_REQUESTED":
			parts = append(parts, "review "+style.Orange.Render("✗"))
		case "REVIEW_REQUIRED":
			parts = append(parts, "review "+style.Dim.Render("●"))
		}
	}

	// Conflicts
	if e.Mergeable == "CONFLICTING" {
		parts = append(parts, style.Red.Render("⚠ conflicts"))
	}

	// Behind (needs update)
	if e.MergeStateStatus == "BEHIND" {
		parts = append(parts, style.Orange.Render("↻ behind"))
	}

	return strings.Join(parts, sep)
}

func (m Model) viewInput() string {
	dialogWidth := 50
	if m.width > 60 {
		dialogWidth = m.width / 2
	}

	title := inputStyle.Render(m.inputPrompt)
	input := m.textInput.View()
	hint := style.Dim.Render("enter confirm · esc cancel")

	content := fmt.Sprintf("%s\n\n%s\n\n%s", title, input, hint)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#58a6ff")).
		Padding(1, 2).
		Width(dialogWidth).
		Render(content)
}

func (m Model) viewConfirm() string {
	dialogWidth := 50
	if m.width > 60 {
		dialogWidth = m.width / 2
	}

	entry := m.selectedEntry()
	branch := ""
	if entry != nil {
		branch = entry.Branch
	}

	title := style.Red.Render("Delete worktree")
	body := fmt.Sprintf("Remove %s?", branch)
	hint := style.Dim.Render("y confirm · n cancel")

	content := fmt.Sprintf("%s\n\n%s\n\n%s", title, body, hint)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#da3633")).
		Padding(1, 2).
		Width(dialogWidth).
		Render(content)
}

func (m Model) viewOutput() string {
	dialogWidth := m.width - 10
	if dialogWidth > 80 {
		dialogWidth = 80
	}
	if dialogWidth < 40 {
		dialogWidth = 40
	}

	maxLines := m.height/2 - 4
	if maxLines < 5 {
		maxLines = 5
	}

	// Title
	var title string
	if m.outputDone {
		if m.outputErr != nil {
			title = style.Red.Render(fmt.Sprintf("✗ %s", m.outputLabel))
		} else {
			title = style.Green.Render(fmt.Sprintf("✓ %s", m.outputLabel))
		}
	} else {
		title = fmt.Sprintf("%s %s", m.spinner.View(), m.outputLabel)
	}

	// Log lines (show last N)
	var logLines []string
	start := 0
	if len(m.outputLines) > maxLines {
		start = len(m.outputLines) - maxLines
	}
	for _, line := range m.outputLines[start:] {
		logLines = append(logLines, style.Dim.Render(line))
	}

	log := strings.Join(logLines, "\n")
	if log == "" {
		log = style.Dim.Render("...")
	}

	// Footer hint
	var hint string
	if m.outputDone {
		hint = style.Dim.Render("press any key to close")
	}

	var content string
	if hint != "" {
		content = fmt.Sprintf("%s\n\n%s\n\n%s", title, log, hint)
	} else {
		content = fmt.Sprintf("%s\n\n%s", title, log)
	}

	borderColor := lipgloss.Color("#30363d")
	if m.outputDone && m.outputErr != nil {
		borderColor = lipgloss.Color("#da3633")
	} else if m.outputDone {
		borderColor = lipgloss.Color("#238636")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(dialogWidth).
		Render(content)
}

func (m Model) viewStatusBar() string {
	position := ""
	if len(m.entries) > 0 {
		position = statusBarStyle.Render(fmt.Sprintf("%d / %d issues", m.cursor+1, len(m.entries)))
	}
	var result string

	status := ""
	if m.statusMsg != "" {
		status = statusBarStyle.Render(m.statusMsg)
	}

	if position != "" && status != "" {
		result = fmt.Sprintf("  %s  %s", position, style.Dim.Render("·")+" "+status)
	} else if position != "" {
		result = fmt.Sprintf("  %s", position)
	} else if status != "" {
		result = fmt.Sprintf("  %s", status)
	}
	return "\n" + result + "\n"
}

func (m Model) viewFooter() string {
	var keys []key.Binding
	keys = append(keys,
		builtins.Up,
		builtins.Down,
		builtins.Refresh,
	)

	entry := m.selectedEntry()
	for _, kb := range visibleBindings(m.keybindings, entry, m.cfg) {
		keys = append(keys, key.NewBinding(
			key.WithKeys(kb.Key),
			key.WithHelp(kb.Key, kb.Label),
		))
	}

	keys = append(keys, builtins.Quit)

	var parts []string
	for _, k := range keys {
		h := k.Help()
		parts = append(parts, fmt.Sprintf("%s %s",
			style.Dim.Render(h.Key),
			h.Desc,
		))
	}

	return fmt.Sprintf("  %s\n", strings.Join(parts, style.Dim.Render(" · ")))
}
