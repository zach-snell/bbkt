package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

// Scope resolution order, applied identically across every subcommand:
//   1. -R/--repo + -W/--workspace persistent flags
//   2. BBKT_REPO and BBKT_WORKSPACE env vars
//   3. positional args ([workspace] [repo-slug])
//   4. git inference from origin remote in cwd
//   5. error (with a list of accessible workspaces, when available)
//
// -R/--repo accepts either "workspace/slug" (compound, like `gh --repo OWNER/REPO`)
// or a plain "slug" when -W/--workspace (or BBKT_WORKSPACE) supplies the workspace.
// Layers compose: an env workspace + a flag --repo plain-slug is valid.

// ParseArgs resolves workspace and repoSlug for a subcommand and returns any
// remaining positional args after the (optional) workspace/repo prefix.
//
// trailingArgsCount is the number of non-workspace/repo positional args the
// subcommand expects (e.g. 1 for `get <pr-id>`, 0 for `list`). Callers may
// pass -1 to opt out of trailing-arg accounting — used by `repos list` where
// the only positional is the workspace itself.
func ParseArgs(cmd *cobra.Command, args []string, trailingArgsCount int) (workspace, repoSlug string, trailing []string, err error) {
	// 1. flags + env (combined — flag wins over env for the same field)
	wsFlag, repoFlag := scopeFromFlags(cmd)

	if r := repoFlag; r != "" {
		if w, s, ok := splitRepoSpec(r); ok {
			workspace, repoSlug = w, s
		} else {
			repoSlug = r
		}
	}
	if w := wsFlag; w != "" && workspace == "" {
		workspace = w
	}

	// 2. positional args
	// Layout: [workspace] [repo-slug] <trailing...>
	// Only consume from positionals what the flag/env layer did NOT already set.
	n := len(args)
	switch {
	case trailingArgsCount < 0:
		// repos-list style — workspace-only positional, no trailing arg semantics
		if n >= 1 && workspace == "" {
			workspace = args[0]
		}
		trailing = nil
	case n >= 2 && n == trailingArgsCount+2:
		// gosec G602 doesn't trace case guards — n>=2 above proves safety.
		if workspace == "" {
			workspace = args[0]
		}
		if repoSlug == "" {
			repoSlug = args[1] //nolint:gosec // bounded by case guard n>=2
		}
		trailing = args[2:] //nolint:gosec // bounded by case guard n>=2
	case n >= 1 && n == trailingArgsCount+1 && workspace != "" && repoSlug == "":
		// flag/env supplied workspace; positional supplies repo
		repoSlug = args[0]
		trailing = args[1:]
	case n == trailingArgsCount:
		trailing = args
	default:
		return "", "", nil, fmt.Errorf("expected %d positional arg(s); got %d. Pass -R workspace/slug or -W workspace -R slug, or run inside a Bitbucket git clone", trailingArgsCount, n)
	}

	// 3. git inference for whatever still isn't set
	if workspace == "" || (trailingArgsCount >= 0 && repoSlug == "") {
		ws, rs, gerr := bitbucket.GetLocalRepoInfo()
		if gerr == nil {
			if workspace == "" {
				workspace = ws
			}
			if repoSlug == "" {
				repoSlug = rs
			}
		}
	}

	// 4. final validation + helpful error
	if workspace == "" {
		return "", "", nil, missingScopeError("workspace")
	}
	if trailingArgsCount >= 0 && repoSlug == "" {
		return "", "", nil, missingScopeError("repo")
	}

	return workspace, repoSlug, trailing, nil
}

// scopeFromFlags reads -W/--workspace and -R/--repo from the command (which
// inherits them from RootCmd's persistent flags) and falls back to env vars.
// Flag wins over env when both are set.
func scopeFromFlags(cmd *cobra.Command) (workspace, repo string) {
	if cmd != nil {
		workspace, _ = cmd.Flags().GetString("workspace")
		repo, _ = cmd.Flags().GetString("repo")
	}
	if workspace == "" {
		workspace = os.Getenv("BBKT_WORKSPACE")
	}
	if repo == "" {
		repo = os.Getenv("BBKT_REPO")
	}
	return workspace, repo
}

// splitRepoSpec splits a --repo value of the form "workspace/slug" into
// (workspace, slug, true). Returns ok=false for a single-segment value
// (which the caller treats as a bare slug).
func splitRepoSpec(spec string) (workspace, slug string, ok bool) {
	parts := strings.SplitN(spec, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// missingScopeError produces a user-friendly error listing accessible
// workspaces from the cached credentials profile, when available. Falls back
// to a plain message if credentials aren't loaded or the cache is empty.
func missingScopeError(missing string) error {
	hint := "Pass -R workspace/slug (or -W <workspace>), set BBKT_WORKSPACE / BBKT_REPO, or run inside a Bitbucket git clone."

	creds, cerr := bitbucket.LoadCredentials()
	if cerr != nil {
		return fmt.Errorf("could not determine %s.\n%s", missing, hint)
	}

	var entries []string
	for _, w := range creds.AccessibleWorkspaces {
		if w != "" {
			entries = append(entries, w)
		}
	}
	if len(entries) == 0 {
		return fmt.Errorf("could not determine %s.\n%s\n\n(Re-run `bbkt auth` to refresh the cached workspace list for profile %q.)", missing, hint, creds.ProfileName)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("could not determine %s.\n%s\n\nAccessible workspaces for profile %q:\n", missing, hint, creds.ProfileName))
	for _, w := range entries {
		b.WriteString(fmt.Sprintf("  -W %s\n", w))
	}
	return fmt.Errorf("%s", b.String())
}
