package naming

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
var leadingTrailingDash = regexp.MustCompile(`^-+|-+$`)

// BranchName generates a branch name from an issue number and title.
// Format: <issue_number>-<sanitized_title>
const maxBranchLength = 50

func BranchName(issueNumber int, issueTitle string) string {
	prefix := fmt.Sprintf("%d-", issueNumber)
	maxSlug := maxBranchLength - len(prefix)
	slug := Sanitize(issueTitle)
	if slug == "" {
		return fmt.Sprintf("%d", issueNumber)
	}
	if len(slug) > maxSlug {
		slug = slug[:maxSlug]
		slug = leadingTrailingDash.ReplaceAllString(slug, "")
	}
	return prefix + slug
}

// Sanitize converts a string to a URL/branch-safe slug.
// Handles accented characters by stripping diacritics (é→e, ã→a, ç→c).
func Sanitize(s string) string {
	s = strings.ToLower(s)
	s = stripAccents(s)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = leadingTrailingDash.ReplaceAllString(s, "")
	if len(s) > 60 {
		s = s[:60]
		s = leadingTrailingDash.ReplaceAllString(s, "")
	}
	return s
}

// ParseIssueNumber extracts the issue number from a branch name.
// Returns the issue number and true if found, 0 and false otherwise.
func ParseIssueNumber(branchName string) (int, bool) {
	parts := strings.SplitN(branchName, "-", 2)
	if len(parts) < 1 {
		return 0, false
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, false
	}
	return n, true
}

// stripAccents removes diacritical marks from characters.
// "não" → "nao", "situação" → "situacao"
func stripAccents(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range norm.NFD.String(s) {
		if unicode.Is(unicode.Mn, r) {
			continue // skip combining marks
		}
		b.WriteRune(r)
	}
	return b.String()
}
