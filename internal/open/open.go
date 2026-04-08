package open

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

type Opts struct {
	Command      string
	WorktreePath string
	Branch       string
	IssueNumber  int
}

// Run executes the open command with template variables resolved.
func Run(opts Opts) error {
	if opts.Command == "" {
		return fmt.Errorf("open.command not configured in .gh-wt.yml")
	}

	resolved, err := resolveTemplate(opts)
	if err != nil {
		return fmt.Errorf("failed to resolve open command template: %w", err)
	}

	cmd := exec.Command("sh", "-c", resolved)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open command failed: %w", err)
	}

	return nil
}

func resolveTemplate(opts Opts) (string, error) {
	tmpl, err := template.New("open").Parse(opts.Command)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"WorktreePath": opts.WorktreePath,
		"Branch":       opts.Branch,
		"IssueNumber":  opts.IssueNumber,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
