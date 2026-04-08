package worktree

import (
	"fmt"
	"os/exec"
	"strings"
	"text/template"
)

type CreateOpts struct {
	Branch       string
	BaseBranch   string
	PathTemplate string
	RepoName     string
	RepoPath     string
	IssueNumber  int
}

type Info struct {
	Path   string
	Branch string
	Head   string
}

// Create creates a new git worktree. If the branch already exists (e.g. fetched
// from remote), it uses it directly. Otherwise creates a new branch from BaseBranch.
func Create(opts CreateOpts) (string, error) {
	wtPath, err := resolvePath(opts)
	if err != nil {
		return "", fmt.Errorf("failed to resolve worktree path: %w", err)
	}

	var cmd *exec.Cmd
	if branchExists(opts.Branch) {
		// Branch already exists (created by gh issue develop + fetch)
		cmd = exec.Command("git", "worktree", "add", wtPath, opts.Branch)
	} else {
		// Create new branch from base
		cmd = exec.Command("git", "worktree", "add", "-b", opts.Branch, wtPath, opts.BaseBranch)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree add failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return wtPath, nil
}

// FetchBranch fetches a specific branch from origin and creates a local tracking branch.
func FetchBranch(branch string) error {
	cmd := exec.Command("git", "fetch", "origin", fmt.Sprintf("%s:%s", branch, branch))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// LastCommitTime returns the unix timestamp of the last commit in a worktree.
func LastCommitTime(wtPath string) int64 {
	cmd := exec.Command("git", "log", "-1", "--format=%ct")
	cmd.Dir = wtPath
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	var t int64
	fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &t)
	return t
}

func branchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	return cmd.Run() == nil
}

// MergedBranches returns a set of branch names that have been merged into the given base branch.
// This is a local git operation, no API calls.
func MergedBranches(base string) (map[string]bool, error) {
	cmd := exec.Command("git", "branch", "--merged", base, "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git branch --merged failed: %w", err)
	}

	merged := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			merged[line] = true
		}
	}
	return merged, nil
}

// List returns all worktrees.
func List() ([]Info, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list failed: %w", err)
	}

	return parseWorktreeList(string(output)), nil
}

// Remove removes a worktree. If the worktree directory is already gone or
// corrupted, falls back to `git worktree prune` to clean up the reference.
func Remove(path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = append(args, "--force")
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	// Fallback: if the directory is gone/corrupted, prune cleans up orphan refs
	pruneCmd := exec.Command("git", "worktree", "prune")
	if pruneOut, pruneErr := pruneCmd.CombinedOutput(); pruneErr != nil {
		return fmt.Errorf("git worktree remove failed: %s (prune also failed: %s)", strings.TrimSpace(string(output)), strings.TrimSpace(string(pruneOut)))
	}

	return nil
}

func resolvePath(opts CreateOpts) (string, error) {
	tmpl, err := template.New("path").Parse(opts.PathTemplate)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"RepoName":    opts.RepoName,
		"RepoPath":    opts.RepoPath,
		"Branch":      opts.Branch,
		"IssueNumber": opts.IssueNumber,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func parseWorktreeList(output string) []Info {
	var worktrees []Info
	var current Info

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "worktree "):
			current.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "":
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = Info{}
			}
		}
	}

	return worktrees
}
