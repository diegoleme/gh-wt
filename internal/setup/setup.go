// Package setup runs a first-time interactive wizard that seeds
// ~/.config/gh-wt/ with a default configuration tailored for the
// zellij + claude workflow.
package setup

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed templates
var templates embed.FS

// MaybeRun runs the first-time setup wizard if the user's config file does
// not yet exist. It is a no-op otherwise, so it is safe to call on every
// invocation of the TUI entrypoint.
func MaybeRun() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	targetDir := filepath.Join(configDir, "gh-wt")
	configPath := filepath.Join(targetDir, "config.yml")

	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	return runWizard(targetDir)
}

func runWizard(targetDir string) error {
	fmt.Println()
	fmt.Println("Welcome to gh-wt!")
	fmt.Println()
	fmt.Println("It looks like this is your first time running gh-wt. The default")
	fmt.Println("configuration is tailored for a zellij + claude workflow:")
	fmt.Println()
	fmt.Println("  - zellij opens a workspace with gh-wt on the left and claude on the right")
	fmt.Println("  - pressing <enter> on an issue opens (or focuses) a claude pane for its worktree")
	fmt.Println()
	fmt.Println("Required tools:")
	fmt.Println()
	printTool("zellij", "terminal multiplexer for the workspace layout")
	printTool("claude", "Claude Code CLI")
	printTool("jq", "used by the open-or-focus helper script")
	fmt.Println()
	fmt.Println("Files to be created in " + targetDir + ":")
	fmt.Println()
	fmt.Println("  config.yml        keybindings and defaults")
	fmt.Println("  workspace.kdl     zellij layout: gh-wt side-by-side with claude")
	fmt.Println("  open-or-focus.sh  opens or focuses a claude pane for a worktree")
	fmt.Println()
	fmt.Println("You can edit any of these files later to customize your workflow.")
	fmt.Println()

	if !prompt("Create default configuration now?") {
		fmt.Println()
		fmt.Println("Skipped. Create " + filepath.Join(targetDir, "config.yml") + " manually to configure gh-wt.")
		fmt.Println()
		return nil
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", targetDir, err)
	}

	entries, err := fs.ReadDir(templates, "templates")
	if err != nil {
		return fmt.Errorf("read embedded templates: %w", err)
	}

	fmt.Println()
	for _, e := range entries {
		src := filepath.Join("templates", e.Name())
		dst := filepath.Join(targetDir, e.Name())

		data, err := templates.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read template %s: %w", src, err)
		}

		mode := fs.FileMode(0o644)
		if strings.HasSuffix(e.Name(), ".sh") {
			mode = 0o755
		}

		if _, err := os.Stat(dst); err == nil {
			fmt.Printf("  ~ %s (exists, skipped)\n", dst)
			continue
		}

		if err := os.WriteFile(dst, data, mode); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
		fmt.Printf("  + %s\n", dst)
	}

	fmt.Println()
	fmt.Println("Setup complete. Launching gh-wt...")
	fmt.Println()
	return nil
}

func printTool(name, desc string) {
	mark := "✓ installed"
	if _, err := exec.LookPath(name); err != nil {
		mark = "✗ not found"
	}
	fmt.Printf("  %-8s %-14s %s\n", name, mark, desc)
}

func prompt(q string) bool {
	fmt.Printf("%s [Y/n]: ", q)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return false
	}
	ans := strings.TrimSpace(strings.ToLower(line))
	return ans == "" || ans == "y" || ans == "yes"
}
