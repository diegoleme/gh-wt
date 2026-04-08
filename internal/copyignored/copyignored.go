package copyignored

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/diegoleme/gh-wt/internal/config"
)

// Copy copies gitignored files from the source worktree to the target,
// respecting exclude patterns and creating symlinks where configured.
func Copy(cfg config.CopyIgnoredConfig, sourceDir, targetDir string) error {
	if !cfg.Enabled {
		return nil
	}

	// Get list of ignored files that exist in source
	files, err := listIgnoredFiles(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to list ignored files: %w", err)
	}

	symlinkSet := make(map[string]bool)
	for _, s := range cfg.Symlink {
		symlinkSet[s] = true
	}

	copied := 0
	symlinked := 0

	for _, file := range files {
		if shouldExclude(file, cfg.Exclude) {
			continue
		}

		srcPath := filepath.Join(sourceDir, file)
		dstPath := filepath.Join(targetDir, file)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}

		// Check if this is a symlink candidate (top-level dir match)
		topLevel := strings.SplitN(file, string(filepath.Separator), 2)[0]
		if symlinkSet[topLevel] {
			// Only symlink the top-level directory itself
			symlinkDst := filepath.Join(targetDir, topLevel)
			symlinkSrc := filepath.Join(sourceDir, topLevel)
			if _, err := os.Lstat(symlinkDst); err == nil {
				continue // Already exists
			}
			if err := os.Symlink(symlinkSrc, symlinkDst); err != nil {
				return fmt.Errorf("failed to symlink %s: %w", topLevel, err)
			}
			symlinked++
			continue
		}

		// Copy file
		data, err := os.ReadFile(srcPath)
		if err != nil {
			continue // Skip files we can't read
		}
		info, err := os.Stat(srcPath)
		if err != nil {
			continue
		}
		if err := os.WriteFile(dstPath, data, info.Mode()); err != nil {
			return fmt.Errorf("failed to copy %s: %w", file, err)
		}
		copied++
	}

	if copied > 0 || symlinked > 0 {
		fmt.Printf("✓ Copied %d ignored files, symlinked %d directories\n", copied, symlinked)
	}

	return nil
}

func listIgnoredFiles(dir string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func shouldExclude(file string, excludes []string) bool {
	for _, pattern := range excludes {
		// Simple prefix match for directory patterns (ending with /)
		if strings.HasSuffix(pattern, "/") {
			if strings.HasPrefix(file, strings.TrimSuffix(pattern, "/")) {
				return true
			}
		}
		matched, _ := filepath.Match(pattern, file)
		if matched {
			return true
		}
	}
	return false
}
