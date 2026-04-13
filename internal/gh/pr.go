package gh

import "fmt"

type PullRequest struct {
	Number           int
	Title            string
	State            string // OPEN, CLOSED, MERGED
	IsDraft          bool
	MergedAt         string
	URL              string
	Additions        int
	Deletions        int
	ReviewDecision   string // APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED, ""
	Mergeable        string // MERGEABLE, CONFLICTING, UNKNOWN
	MergeStateStatus string // BEHIND, BLOCKED, CLEAN, DIRTY, DRAFT, HAS_HOOKS, UNKNOWN, UNSTABLE
	ChecksState      string // SUCCESS, FAILURE, PENDING, EXPECTED, ERROR, ""
	LinkedIssue      *LinkedIssue
}

type LinkedIssue struct {
	Number      int
	Title       string
	State       string
	StateReason string
	Labels      []string
}

// DisplayState returns a human-readable state for display.
func (pr *PullRequest) DisplayState() string {
	if pr.MergedAt != "" {
		return "merged"
	}
	if pr.IsDraft {
		return "draft"
	}
	switch pr.State {
	case "OPEN":
		return "open"
	case "CLOSED":
		return "closed"
	case "MERGED":
		return "merged"
	default:
		return pr.State
	}
}

const prQuery = `
query($owner: String!, $repo: String!, $branch: String!) {
  repository(owner: $owner, name: $repo) {
    pullRequests(
      headRefName: $branch
      first: 1
      states: [OPEN, CLOSED, MERGED]
      orderBy: {field: CREATED_AT, direction: DESC}
    ) {
      nodes {
        number
        title
        state
        isDraft
        mergedAt
        url
        additions
        deletions
        reviewDecision
        mergeable
        mergeStateStatus
        closingIssuesReferences(first: 1) {
          nodes {
            number
            title
            state
            stateReason
            labels(first: 20) {
              nodes {
                name
              }
            }
          }
        }
        commits(last: 1) {
          nodes {
            commit {
              statusCheckRollup {
                state
              }
            }
          }
        }
      }
    }
  }
}`

// FindPRForBranch returns the most recent PR for a given head branch (any state)
// with rich data: checks, review decision, linked issues.
func FindPRForBranch(branch string) (*PullRequest, error) {
	repo, err := Repo()
	if err != nil {
		return nil, err
	}

	client, err := GraphQLClient()
	if err != nil {
		return nil, err
	}

	var result struct {
		Repository struct {
			PullRequests struct {
				Nodes []struct {
					Number           int    `json:"number"`
					Title            string `json:"title"`
					State            string `json:"state"`
					IsDraft          bool   `json:"isDraft"`
					MergedAt         string `json:"mergedAt"`
					URL              string `json:"url"`
					Additions        int    `json:"additions"`
					Deletions        int    `json:"deletions"`
					ReviewDecision   string `json:"reviewDecision"`
					Mergeable        string `json:"mergeable"`
					MergeStateStatus string `json:"mergeStateStatus"`
					ClosingIssuesReferences struct {
						Nodes []struct {
							Number      int    `json:"number"`
							Title       string `json:"title"`
							State       string `json:"state"`
							StateReason string `json:"stateReason"`
							Labels      struct {
								Nodes []struct {
									Name string `json:"name"`
								} `json:"nodes"`
							} `json:"labels"`
						} `json:"nodes"`
					} `json:"closingIssuesReferences"`
					Commits struct {
						Nodes []struct {
							Commit struct {
								StatusCheckRollup *struct {
									State string `json:"state"`
								} `json:"statusCheckRollup"`
							} `json:"commit"`
						} `json:"nodes"`
					} `json:"commits"`
				} `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	}

	vars := map[string]interface{}{
		"owner":  repo.Owner,
		"repo":   repo.Name,
		"branch": branch,
	}

	if err := client.Do(prQuery, vars, &result); err != nil {
		return nil, fmt.Errorf("failed to query PR for branch %s: %w", branch, err)
	}

	nodes := result.Repository.PullRequests.Nodes
	if len(nodes) == 0 {
		return nil, nil
	}

	node := nodes[0]
	pr := &PullRequest{
		Number:           node.Number,
		Title:            node.Title,
		State:            node.State,
		IsDraft:          node.IsDraft,
		MergedAt:         node.MergedAt,
		URL:              node.URL,
		Additions:        node.Additions,
		Deletions:        node.Deletions,
		ReviewDecision:   node.ReviewDecision,
		Mergeable:        node.Mergeable,
		MergeStateStatus: node.MergeStateStatus,
	}

	// Extract checks state
	if len(node.Commits.Nodes) > 0 {
		if rollup := node.Commits.Nodes[0].Commit.StatusCheckRollup; rollup != nil {
			pr.ChecksState = rollup.State
		}
	}

	// Extract linked issue
	if len(node.ClosingIssuesReferences.Nodes) > 0 {
		issue := node.ClosingIssuesReferences.Nodes[0]
		labels := make([]string, len(issue.Labels.Nodes))
		for i, l := range issue.Labels.Nodes {
			labels[i] = l.Name
		}
		pr.LinkedIssue = &LinkedIssue{
			Number:      issue.Number,
			Title:       issue.Title,
			State:       issue.State,
			StateReason: issue.StateReason,
			Labels:      labels,
		}
	}

	return pr, nil
}
