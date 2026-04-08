package hooks

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/diegoleme/gh-wt/internal/config"
)

// Run executes a list of hook steps in order in the given directory.
func Run(steps []config.HookStep, dir string) error {
	for _, step := range steps {
		if step.Run == "" {
			continue
		}

		fmt.Printf("  → %s\n", step.Run)

		cmd := exec.Command("sh", "-c", step.Run)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("hook %q failed: %w", step.Run, err)
		}

		fmt.Printf("  ✓ %s\n", step.Run)
	}
	return nil
}
