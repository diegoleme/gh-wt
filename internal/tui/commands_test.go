package tui

import (
	"os/exec"
	"testing"

	"github.com/diegoleme/gh-wt/internal/listutil"
)

func TestShellQuote(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{``, `''`},
		{`hello`, `'hello'`},
		{`with spaces`, `'with spaces'`},
		{`it's`, `'it'\''s'`},
		{`a'b'c`, `'a'\''b'\''c'`},
		{`"quotes"`, `'"quotes"'`},
		{`$VAR ` + "`cmd`" + ` \n`, `'$VAR ` + "`cmd`" + ` \n'`},
	}
	for _, c := range cases {
		got := shellQuote(c.in)
		if got != c.want {
			t.Errorf("shellQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestShellQuoteRoundtripWithSh feeds the quoted output to /bin/sh and verifies
// it tokenizes back to the original string as a single argument.
func TestShellQuoteRoundtripWithSh(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	inputs := []string{
		"simple",
		"with spaces",
		"it's complicated",
		`Inbox sidebar: faturas em aberto não aparecem para devedor 'OPTICALIA 207 - SP ALPHA SQUARE MALL'`,
		`mix "double" and 'single' quotes`,
		`$IFS ` + "`evil`" + ` ; rm -rf /`,
	}
	for _, in := range inputs {
		quoted := shellQuote(in)
		out, err := exec.Command("sh", "-c", "printf %s "+quoted).Output()
		if err != nil {
			t.Fatalf("sh failed for %q: %v", in, err)
		}
		if string(out) != in {
			t.Errorf("roundtrip for %q via %q produced %q", in, quoted, string(out))
		}
	}
}

// TestResolveCommandAutoQuote verifies that string template variables are
// auto-quoted, so users don't need to add quotes or call escape helpers.
func TestResolveCommandAutoQuote(t *testing.T) {
	entry := &listutil.Entry{
		IssueNumber: 447,
		IssueTitle:  `Inbox sidebar: faturas em aberto não aparecem para devedor 'OPTICALIA 207 - SP ALPHA SQUARE MALL'`,
		Branch:      "447-inbox",
		Path:        "/tmp/wt with space/447",
	}

	cases := []struct {
		name string
		tmpl string
		want string
	}{
		{
			name: "title and path are quoted automatically",
			tmpl: `script.sh {{.IssueTitle}} {{.Path}}`,
			want: `script.sh 'Inbox sidebar: faturas em aberto não aparecem para devedor '\''OPTICALIA 207 - SP ALPHA SQUARE MALL'\''' '/tmp/wt with space/447'`,
		},
		{
			name: "branch is quoted",
			tmpl: `git branch -D {{.Branch}}`,
			want: `git branch -D '447-inbox'`,
		},
		{
			name: "issue number is interpolated raw",
			tmpl: `gh issue view {{.IssueNumber}}`,
			want: `gh issue view 447`,
		},
		{
			name: "concatenation with prefix produces single shell word",
			tmpl: `script.sh '#'{{.IssueNumber}}-{{.IssueTitle}}`,
			want: `script.sh '#'447-'Inbox sidebar: faturas em aberto não aparecem para devedor '\''OPTICALIA 207 - SP ALPHA SQUARE MALL'\'''`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := resolveCommand(c.tmpl, entry, "")
			if err != nil {
				t.Fatalf("resolveCommand: %v", err)
			}
			if got != c.want {
				t.Errorf("\n  got: %s\n want: %s", got, c.want)
			}
		})
	}
}

// TestResolveCommandShellTokenization is the integration-level guarantee: after
// resolution, the command must be tokenizable by sh into the expected args.
func TestResolveCommandShellTokenization(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	entry := &listutil.Entry{
		IssueNumber: 447,
		IssueTitle:  `Inbox sidebar: faturas em aberto não aparecem para devedor 'OPTICALIA 207 - SP ALPHA SQUARE MALL'`,
		Branch:      "447-inbox",
		Path:        "/tmp/wt with space/447",
	}

	// User template: a name (issue prefix + title concatenated) and a cwd.
	tmpl := `script '#'{{.IssueNumber}}-{{.IssueTitle}} {{.Path}}`
	resolved, err := resolveCommand(tmpl, entry, "")
	if err != nil {
		t.Fatalf("resolveCommand: %v", err)
	}

	// Use `set --` to push the rendered tokens (after the script name) into $@,
	// then print arg count and each arg on its own line.
	probe := `set -- ` + resolved[len("script "):] + `; printf '%s\n' "$#"; for a in "$@"; do printf '<%s>\n' "$a"; done`
	out, err := exec.Command("sh", "-c", probe).Output()
	if err != nil {
		t.Fatalf("sh failed: %v\nresolved=%s", err, resolved)
	}

	want := "2\n<#447-" + entry.IssueTitle + ">\n<" + entry.Path + ">\n"
	if string(out) != want {
		t.Errorf("tokenization mismatch.\n got: %q\nwant: %q\nresolved: %s", string(out), want, resolved)
	}
}

// TestResolveCommandNumericRaw guards the contract that integer template
// variables are interpolated unquoted, so things like `[ {{.PRNumber}} -gt 0 ]`
// keep working as plain shell arithmetic comparisons.
func TestResolveCommandNumericRaw(t *testing.T) {
	entry := &listutil.Entry{IssueNumber: 1, PRNumber: 42}
	got, err := resolveCommand(`if [ {{.PRNumber}} -gt 0 ]; then echo {{.IssueNumber}}; fi`, entry, "")
	if err != nil {
		t.Fatalf("resolveCommand: %v", err)
	}
	want := `if [ 42 -gt 0 ]; then echo 1; fi`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
