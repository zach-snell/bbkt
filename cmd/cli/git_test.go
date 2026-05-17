package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newScopeCmd returns a cobra.Command pre-wired with the same -W/-R
// persistent flags RootCmd registers. Tests use this to exercise ParseArgs
// without booting the full root tree.
func newScopeCmd(t *testing.T) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: "test"}
	c.Flags().StringP("workspace", "W", "", "")
	c.Flags().StringP("repo", "R", "", "")
	return c
}

func TestParseArgs_FlagCompoundRepoOnly(t *testing.T) {
	cmd := newScopeCmd(t)
	_ = cmd.Flags().Set("repo", "myws/myrepo")

	ws, rs, trailing, err := ParseArgs(cmd, nil, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ws != "myws" || rs != "myrepo" {
		t.Errorf("got ws=%q rs=%q, want myws/myrepo", ws, rs)
	}
	if len(trailing) != 0 {
		t.Errorf("trailing should be empty, got %v", trailing)
	}
}

func TestParseArgs_FlagWorkspacePlusFlagRepoSlug(t *testing.T) {
	cmd := newScopeCmd(t)
	_ = cmd.Flags().Set("workspace", "myws")
	_ = cmd.Flags().Set("repo", "myrepo")

	ws, rs, _, err := ParseArgs(cmd, nil, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ws != "myws" || rs != "myrepo" {
		t.Errorf("got ws=%q rs=%q, want myws/myrepo", ws, rs)
	}
}

func TestParseArgs_FlagWorkspacePlusPositionalRepo(t *testing.T) {
	cmd := newScopeCmd(t)
	_ = cmd.Flags().Set("workspace", "myws")

	// trailingArgsCount=0: layout becomes [repo-slug] when workspace is pinned.
	ws, rs, _, err := ParseArgs(cmd, []string{"myrepo"}, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ws != "myws" || rs != "myrepo" {
		t.Errorf("got ws=%q rs=%q, want myws/myrepo", ws, rs)
	}
}

func TestParseArgs_EnvFallback(t *testing.T) {
	t.Setenv("BBKT_WORKSPACE", "env-ws")
	t.Setenv("BBKT_REPO", "env-ws/env-repo")
	cmd := newScopeCmd(t)

	ws, rs, _, err := ParseArgs(cmd, nil, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ws != "env-ws" || rs != "env-repo" {
		t.Errorf("got ws=%q rs=%q, want env-ws/env-repo", ws, rs)
	}
}

func TestParseArgs_FlagBeatsEnv(t *testing.T) {
	t.Setenv("BBKT_REPO", "env-ws/env-repo")
	cmd := newScopeCmd(t)
	_ = cmd.Flags().Set("repo", "flag-ws/flag-repo")

	ws, rs, _, err := ParseArgs(cmd, nil, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ws != "flag-ws" || rs != "flag-repo" {
		t.Errorf("got ws=%q rs=%q, want flag-ws/flag-repo", ws, rs)
	}
}

func TestParseArgs_PositionalWorkspaceAndRepo(t *testing.T) {
	cmd := newScopeCmd(t)

	ws, rs, trailing, err := ParseArgs(cmd, []string{"posws", "posrepo", "42"}, 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ws != "posws" || rs != "posrepo" {
		t.Errorf("got ws=%q rs=%q, want posws/posrepo", ws, rs)
	}
	if len(trailing) != 1 || trailing[0] != "42" {
		t.Errorf("trailing = %v, want [42]", trailing)
	}
}

func TestParseArgs_WorkspaceOnlyMinusOne(t *testing.T) {
	// repos-list style: trailingArgsCount=-1 — workspace from positional or flag.
	cmd := newScopeCmd(t)
	_ = cmd.Flags().Set("workspace", "flagws")

	ws, _, _, err := ParseArgs(cmd, nil, -1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ws != "flagws" {
		t.Errorf("got ws=%q, want flagws", ws)
	}
}

func TestParseArgs_MissingWorkspaceErrorMentionsFlag(t *testing.T) {
	// No flag, no env, no positionals, no git inference (cwd isn't a bitbucket clone).
	t.Setenv("BBKT_WORKSPACE", "")
	t.Setenv("BBKT_REPO", "")
	cmd := newScopeCmd(t)

	_, _, _, err := ParseArgs(cmd, nil, 0)
	if err == nil {
		t.Fatal("expected error when no scope is resolvable")
	}
	if !strings.Contains(err.Error(), "-R") || !strings.Contains(err.Error(), "-W") {
		t.Errorf("error should mention -R and -W; got: %v", err)
	}
}

func TestSplitRepoSpec(t *testing.T) {
	cases := []struct {
		in        string
		wantWS    string
		wantSlug  string
		wantOK    bool
	}{
		{"ws/slug", "ws", "slug", true},
		{"ws/with/slash", "ws", "with/slash", true}, // SplitN keeps the remainder intact
		{"slug-only", "", "", false},
		{"", "", "", false},
		{"/slug", "", "", false},
		{"ws/", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			ws, s, ok := splitRepoSpec(tc.in)
			if ws != tc.wantWS || s != tc.wantSlug || ok != tc.wantOK {
				t.Errorf("splitRepoSpec(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tc.in, ws, s, ok, tc.wantWS, tc.wantSlug, tc.wantOK)
			}
		})
	}
}
