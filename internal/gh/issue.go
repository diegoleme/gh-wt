package gh

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Issue struct {
	Number      int
	Title       string
	Body        string
	Labels      []string
	State       string
	StateReason string
	UpdatedAt   string
}

type IssueState struct {
	Title       string `json:"title"`
	State       string `json:"state"`
	StateReason string `json:"state_reason"`
	UpdatedAt   string `json:"updated_at"`
}

// ListOpenIssues returns all open issues from the current repository.
func ListOpenIssues() ([]Issue, error) {
	cmd := exec.Command("gh", "issue", "list", "--state", "open", "--json", "number,title,labels,state,updatedAt", "--limit", "200")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh issue list failed: %w", err)
	}

	var results []struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		UpdatedAt string `json:"updatedAt"`
		Labels    []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}

	if err := json.Unmarshal(output, &results); err != nil {
		return nil, err
	}

	issues := make([]Issue, len(results))
	for i, r := range results {
		labels := make([]string, len(r.Labels))
		for j, l := range r.Labels {
			labels[j] = l.Name
		}
		issues[i] = Issue{
			Number:    r.Number,
			Title:     r.Title,
			State:     r.State,
			UpdatedAt: r.UpdatedAt,
			Labels:    labels,
		}
	}

	return issues, nil
}

// FetchIssue fetches an issue by number from the current repository.
func FetchIssue(number int) (*Issue, error) {
	repo, err := Repo()
	if err != nil {
		return nil, err
	}

	client, err := RESTClient()
	if err != nil {
		return nil, err
	}

	var result struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		Body        string `json:"body"`
		State       string `json:"state"`
		StateReason string `json:"state_reason"`
		Labels      []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}

	err = client.Get(fmt.Sprintf("repos/%s/%s/issues/%d", repo.Owner, repo.Name, number), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue #%d: %w", number, err)
	}

	labels := make([]string, len(result.Labels))
	for i, l := range result.Labels {
		labels[i] = l.Name
	}

	return &Issue{
		Number:      result.Number,
		Title:       result.Title,
		Body:        result.Body,
		Labels:      labels,
		State:       result.State,
		StateReason: result.StateReason,
	}, nil
}

// FetchIssueState fetches only the state of an issue (lightweight).
func FetchIssueState(number int) (*IssueState, error) {
	repo, err := Repo()
	if err != nil {
		return nil, err
	}

	client, err := RESTClient()
	if err != nil {
		return nil, err
	}

	var result IssueState
	err = client.Get(fmt.Sprintf("repos/%s/%s/issues/%d", repo.Owner, repo.Name, number), &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// DevelopBranch creates a branch linked to an issue using gh issue develop.
// This creates the branch on the remote and links it to the issue.
func DevelopBranch(issueNumber int, branchName string, baseBranch string) error {
	args := []string{"issue", "develop", fmt.Sprintf("%d", issueNumber), "--name", branchName}
	if baseBranch != "" {
		args = append(args, "--base", baseBranch)
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh issue develop failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// AssignIssue assigns the current user to an issue.
func AssignIssue(issueNumber int) error {
	cmd := exec.Command("gh", "issue", "edit", fmt.Sprintf("%d", issueNumber), "--add-assignee", "@me")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh issue edit failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}
