package listutil

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	ghclient "github.com/diegoleme/gh-wt/internal/gh"
	"github.com/diegoleme/gh-wt/internal/naming"
	"github.com/diegoleme/gh-wt/internal/worktree"
)

type Entry struct {
	Branch           string   `json:"branch"`
	Path             string   `json:"path"`
	IssueNumber      int      `json:"issue_number,omitempty"`
	IssueTitle       string   `json:"issue_title,omitempty"`
	IssueState       string   `json:"issue_state,omitempty"`
	IssueStateReason string   `json:"issue_state_reason,omitempty"`
	Labels           []string `json:"labels,omitempty"`
	PRNumber         int      `json:"pr_number,omitempty"`
	PRState          string   `json:"pr_state,omitempty"`
	PRAdditions      int      `json:"pr_additions,omitempty"`
	PRDeletions      int      `json:"pr_deletions,omitempty"`
	ChecksState      string   `json:"checks_state,omitempty"`
	ReviewDecision   string   `json:"review_decision,omitempty"`
	Mergeable        string   `json:"mergeable,omitempty"`
	MergeStateStatus string   `json:"merge_state_status,omitempty"`
	Merged           bool     `json:"merged"`
	Current          bool     `json:"current"`
	HasWorktree      bool     `json:"has_worktree"`
	IssueUpdatedAt   string   `json:"issue_updated_at,omitempty"`
	LastCommitTime   int64    `json:"last_commit_time,omitempty"` // unix timestamp
	Section          string   `json:"section"`                    // "active", "stale", "open"
}

type BuildOption func(*buildOptions)

type buildOptions struct {
	quiet bool
}

// WithQuiet suppresses the stderr spinner (for TUI usage).
func WithQuiet() BuildOption {
	return func(o *buildOptions) { o.quiet = true }
}

// BuildIssueEntries lists issues and worktrees, returning entries in 3 sections:
// 1. Active Worktrees (open issue + worktree, sorted by last commit time)
// 2. Stale Worktrees (closed issue + worktree still exists, sorted by issue updatedAt)
// 3. Open Issues (open issue, no worktree, sorted by issue updatedAt)
func BuildIssueEntries() ([]Entry, error) {
	// 1. Fetch all open issues
	issues, err := ghclient.ListOpenIssues()
	if err != nil {
		return nil, err
	}

	// 2. List local worktrees
	wts, err := worktree.List()
	if err != nil {
		return nil, err
	}

	cwd, _ := os.Getwd()
	cwd, _ = filepath.Abs(cwd)

	// 3. Build worktree lookup by issue number
	type wtInfo struct {
		Branch  string
		Path    string
		AbsPath string
		Current bool
	}
	wtByIssue := make(map[int]wtInfo)
	for _, wt := range wts {
		if n, ok := naming.ParseIssueNumber(wt.Branch); ok {
			absPath, _ := filepath.Abs(wt.Path)
			wtByIssue[n] = wtInfo{
				Branch:  wt.Branch,
				Path:    wt.Path,
				AbsPath: absPath,
				Current: absPath == cwd,
			}
		}
	}

	// 4. Build set of open issue numbers
	openIssueSet := make(map[int]bool)
	for _, issue := range issues {
		openIssueSet[issue.Number] = true
	}

	// 5. Find worktrees whose issues are NOT in the open list (closed/stale)
	var staleIssueNumbers []int
	for issueNum := range wtByIssue {
		if !openIssueSet[issueNum] {
			staleIssueNumbers = append(staleIssueNumbers, issueNum)
		}
	}

	// 6. Fetch stale issue details concurrently
	staleIssues := make(map[int]*ghclient.IssueState)
	if len(staleIssueNumbers) > 0 {
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, 10)
		for _, n := range staleIssueNumbers {
			wg.Add(1)
			go func(num int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				state, err := ghclient.FetchIssueState(num)
				if err == nil && state != nil {
					mu.Lock()
					staleIssues[num] = state
					mu.Unlock()
				}
			}(n)
		}
		wg.Wait()
	}

	// 7. Build all entries
	var active, stale, open []Entry

	for _, issue := range issues {
		entry := Entry{
			IssueNumber:    issue.Number,
			IssueTitle:     issue.Title,
			IssueState:     issue.State,
			IssueUpdatedAt: issue.UpdatedAt,
			Labels:         issue.Labels,
		}

		if wt, ok := wtByIssue[issue.Number]; ok {
			entry.Branch = wt.Branch
			entry.Path = wt.Path
			entry.Current = wt.Current
			entry.HasWorktree = true
			entry.LastCommitTime = worktree.LastCommitTime(wt.AbsPath)
			entry.Section = "active"
			active = append(active, entry)
		} else {
			entry.Section = "open"
			open = append(open, entry)
		}
	}

	for _, num := range staleIssueNumbers {
		wt := wtByIssue[num]
		entry := Entry{
			IssueNumber: num,
			Branch:      wt.Branch,
			Path:        wt.Path,
			Current:     wt.Current,
			HasWorktree: true,
			Section:     "stale",
		}
		if state, ok := staleIssues[num]; ok {
			entry.IssueTitle = state.Title
			entry.IssueState = state.State
			entry.IssueStateReason = state.StateReason
			entry.IssueUpdatedAt = state.UpdatedAt
		}
		entry.LastCommitTime = worktree.LastCommitTime(wt.AbsPath)
		stale = append(stale, entry)
	}

	// 8. Sort each section
	// Active: by last commit time (most recent first)
	sort.Slice(active, func(i, j int) bool {
		return active[i].LastCommitTime > active[j].LastCommitTime
	})
	// Stale: by issue updatedAt (most recent first)
	sort.Slice(stale, func(i, j int) bool {
		return stale[i].IssueUpdatedAt > stale[j].IssueUpdatedAt
	})
	// Open: by issue updatedAt (most recent first)
	sort.Slice(open, func(i, j int) bool {
		return open[i].IssueUpdatedAt > open[j].IssueUpdatedAt
	})

	// 9. Check merged branches
	defaultBranch, _ := ghclient.DefaultBranch()
	if defaultBranch != "" {
		merged, err := worktree.MergedBranches(defaultBranch)
		if err == nil {
			markMerged := func(entries []Entry) {
				for i, e := range entries {
					if e.Branch != "" {
						entries[i].Merged = merged[e.Branch]
					}
				}
			}
			markMerged(active)
			markMerged(stale)
		}
	}

	// 10. Combine: done → in progress → todo
	var entries []Entry
	entries = append(entries, stale...)
	entries = append(entries, active...)
	entries = append(entries, open...)

	// 11. Enrich worktree entries with PR info
	fetchPRInfo(entries)

	for i, e := range entries {
		if e.PRState == "merged" {
			entries[i].Merged = true
		}
	}

	return entries, nil
}

func fetchPRInfo(entries []Entry) {
	var lookups []int
	for i, e := range entries {
		if e.HasWorktree && e.Branch != "" {
			lookups = append(lookups, i)
		}
	}

	if len(lookups) == 0 {
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, idx := range lookups {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			pr, err := ghclient.FindPRForBranch(entries[i].Branch)
			if err == nil && pr != nil {
				entries[i].PRNumber = pr.Number
				entries[i].PRState = pr.DisplayState()
				entries[i].PRAdditions = pr.Additions
				entries[i].PRDeletions = pr.Deletions
				entries[i].ChecksState = pr.ChecksState
				entries[i].ReviewDecision = pr.ReviewDecision
				entries[i].Mergeable = pr.Mergeable
				entries[i].MergeStateStatus = pr.MergeStateStatus
			}
		}(idx)
	}

	wg.Wait()
}

// BuildEntries lists all worktrees and enriches them with GitHub data.
// Used by CLI commands (list, prune).
func BuildEntries(opts ...BuildOption) ([]Entry, error) {
	var bo buildOptions
	for _, o := range opts {
		o(&bo)
	}
	worktrees, err := worktree.List()
	if err != nil {
		return nil, err
	}

	cwd, _ := os.Getwd()
	cwd, _ = filepath.Abs(cwd)

	entries := make([]Entry, len(worktrees))
	for i, wt := range worktrees {
		entries[i] = Entry{
			Branch:  wt.Branch,
			Path:    wt.Path,
			Current: wt.Path == cwd,
		}
		if n, ok := naming.ParseIssueNumber(wt.Branch); ok {
			entries[i].IssueNumber = n
		}
	}

	// Check which branches are merged (local git, fast)
	defaultBranch, err := ghclient.DefaultBranch()
	if err == nil {
		merged, err := worktree.MergedBranches(defaultBranch)
		if err == nil {
			for i, e := range entries {
				if e.Branch != "" && e.Branch != defaultBranch {
					entries[i].Merged = merged[e.Branch]
				}
			}
		}
	}

	// Fetch PR info via GraphQL
	fetchRemoteInfo(entries, bo.quiet)

	// Mark as merged if PR state is merged
	for i, e := range entries {
		if e.PRState == "merged" {
			entries[i].Merged = true
		}
	}

	return entries, nil
}

func fetchRemoteInfo(entries []Entry, quiet bool) {
	var prLookups []int
	var knownIssueLookups int
	for i, e := range entries {
		if e.Branch != "" {
			prLookups = append(prLookups, i)
		}
		if e.IssueNumber > 0 {
			knownIssueLookups++
		}
	}

	if len(prLookups) == 0 {
		return
	}

	var done atomic.Int32
	var total atomic.Int32
	total.Store(int32(len(prLookups) + knownIssueLookups))

	stop := make(chan struct{})
	stopped := make(chan struct{})
	if !quiet {
		go func() {
			defer close(stopped)
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			ticker := time.NewTicker(80 * time.Millisecond)
			defer ticker.Stop()
			i := 0
			for {
				select {
				case <-stop:
					fmt.Fprintf(os.Stderr, "\r\033[K")
					return
				case <-ticker.C:
					n := done.Load()
					fmt.Fprintf(os.Stderr, "\r%s Fetching info... (%d/%d)", frames[i%len(frames)], n, total.Load())
					i++
				}
			}
		}()
	} else {
		close(stopped)
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, idx := range prLookups {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			pr, err := ghclient.FindPRForBranch(entries[i].Branch)
			if err == nil && pr != nil {
				entries[i].PRNumber = pr.Number
				entries[i].PRState = pr.DisplayState()
				entries[i].PRAdditions = pr.Additions
				entries[i].PRDeletions = pr.Deletions
				entries[i].ChecksState = pr.ChecksState
				entries[i].ReviewDecision = pr.ReviewDecision
				entries[i].Mergeable = pr.Mergeable
				entries[i].MergeStateStatus = pr.MergeStateStatus

				if pr.LinkedIssue != nil {
					if entries[i].IssueNumber == 0 {
						entries[i].IssueNumber = pr.LinkedIssue.Number
					}
					entries[i].IssueTitle = pr.LinkedIssue.Title
					entries[i].IssueState = pr.LinkedIssue.State
					entries[i].IssueStateReason = pr.LinkedIssue.StateReason
				}
			}
			done.Add(1)
		}(idx)
	}

	wg.Wait()

	// Fetch issue state for entries that have issue numbers but no state yet.
	// Some were discovered from PR bodies (not counted in initial total).
	var issueLookups []int
	for i, e := range entries {
		if e.IssueNumber > 0 && e.IssueState == "" {
			issueLookups = append(issueLookups, i)
		}
	}

	if len(issueLookups) > 0 {
		// Only add the delta from PR-discovered issues
		extras := len(issueLookups) - knownIssueLookups
		if extras > 0 {
			total.Add(int32(extras))
		}
		for _, idx := range issueLookups {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				state, err := ghclient.FetchIssueState(entries[i].IssueNumber)
				if err == nil && state != nil {
					entries[i].IssueTitle = state.Title
					entries[i].IssueState = state.State
					entries[i].IssueStateReason = state.StateReason
				}
				done.Add(1)
			}(idx)
		}
		wg.Wait()
	}

	close(stop)
	<-stopped
}
