package pipeline

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/diegoleme/gh-wt/internal/config"
	"github.com/diegoleme/gh-wt/internal/copyignored"
	ghclient "github.com/diegoleme/gh-wt/internal/gh"
	"github.com/diegoleme/gh-wt/internal/hooks"
	"github.com/diegoleme/gh-wt/internal/naming"
	"github.com/diegoleme/gh-wt/internal/open"
	"github.com/diegoleme/gh-wt/internal/worktree"
)

type StartOpts struct {
	IssueNumber int
	Base        string
	NoLink      bool
	NoAssign    bool
	NoHooks     bool
}

// Start executes the full pipeline for creating a worktree from an issue.
//
// Pipeline order:
//  1. Fetch issue (GitHub API)
//  2. gh issue develop (creates remote branch + links to issue)
//  3. git fetch + git worktree add
//  4. gh issue edit --add-assignee @me
//  5. Execute pre-start hooks
//  6. Copy ignored files
//  7. Execute post-start hooks
//  8. Open (if on_start)
//  9. Print path
func Start(opts StartOpts) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 1. Fetch issue
	issue, err := ghclient.FetchIssue(opts.IssueNumber)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "✓ Issue #%d: %s\n", issue.Number, issue.Title)

	// 2. Generate branch name and create remote branch linked to issue
	branchName := naming.BranchName(issue.Number, issue.Title)

	baseBranch := opts.Base
	if baseBranch == "" {
		baseBranch = cfg.Branch.Base
	}
	if baseBranch == "" {
		baseBranch, err = ghclient.DefaultBranch()
		if err != nil {
			return fmt.Errorf("failed to determine default branch: %w", err)
		}
	}

	if !opts.NoLink {
		if err := ghclient.DevelopBranch(issue.Number, branchName, baseBranch); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Failed to link branch to issue: %s\n", err)
			// gh issue develop with --base may write a [branch ""] section
			// to .git/config when the branch name is empty (e.g. stale link).
			// Clean it up so it doesn't corrupt subsequent git operations.
			cleanEmptyBranchConfig()
		} else {
			fmt.Fprintf(os.Stderr, "✓ Branch linked to issue #%d\n", issue.Number)
		}
	}

	// 3. Fetch branch and create worktree
	if err := worktree.FetchBranch(branchName); err != nil {
		// Branch might not exist on remote (if --no-link), that's ok
		fmt.Fprintf(os.Stderr, "⚠ Fetch failed (will create local branch): %s\n", err)
	}

	repo, err := ghclient.Repo()
	if err != nil {
		return err
	}

	repoPath, err := os.Getwd()
	if err != nil {
		return err
	}

	wtPath, err := worktree.Create(worktree.CreateOpts{
		Branch:       branchName,
		BaseBranch:   baseBranch,
		PathTemplate: cfg.Worktree.Path,
		RepoName:     repo.Name,
		RepoPath:     repoPath,
		IssueNumber:  issue.Number,
	})
	if err != nil {
		return err
	}

	absWtPath, _ := filepath.Abs(wtPath)
	fmt.Fprintf(os.Stderr, "✓ Branch: %s\n", branchName)
	fmt.Fprintf(os.Stderr, "✓ Worktree: %s\n", wtPath)

	// 4. Assign issue to current user
	if !opts.NoAssign {
		if err := ghclient.AssignIssue(issue.Number); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Failed to assign issue: %s\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "✓ Assigned issue #%d to you\n", issue.Number)
		}
	}

	if opts.NoHooks {
		fmt.Println(absWtPath)
		return nil
	}

	// 5. Pre-start hooks
	if len(cfg.Hooks.PreStart) > 0 {
		fmt.Fprintln(os.Stderr, "Running pre-start hooks...")
		if err := hooks.Run(cfg.Hooks.PreStart, absWtPath); err != nil {
			return fmt.Errorf("pre-start hook failed: %w", err)
		}
	}

	// 6. Copy ignored files
	if cfg.Worktree.CopyIgnored.Enabled {
		if err := copyignored.Copy(cfg.Worktree.CopyIgnored, repoPath, absWtPath); err != nil {
			return fmt.Errorf("copy-ignored failed: %w", err)
		}
	}

	// 7. Post-start hooks
	if len(cfg.Hooks.PostStart) > 0 {
		fmt.Fprintln(os.Stderr, "Running post-start hooks...")
		if err := hooks.Run(cfg.Hooks.PostStart, absWtPath); err != nil {
			return fmt.Errorf("post-start hook failed: %w", err)
		}
	}

	// 8. Open (if configured)
	if cfg.Open.OnStart && cfg.Open.Command != "" {
		fmt.Fprintln(os.Stderr, "Opening worktree...")
		if err := open.Run(open.Opts{
			Command:      cfg.Open.Command,
			WorktreePath: absWtPath,
			Branch:       branchName,
			IssueNumber:  issue.Number,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Open failed: %s\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "✓ Opened worktree")
		}
	}

	// 9. Print path
	fmt.Println(absWtPath)

	return nil
}

// cleanEmptyBranchConfig removes a [branch ""] section from .git/config.
// gh issue develop --base writes branch.<name>.gh-merge-base; when the branch
// name is empty this corrupts the config and breaks all subsequent git commands.
func cleanEmptyBranchConfig() {
	exec.Command("git", "config", "--remove-section", "branch.").CombinedOutput()
}
