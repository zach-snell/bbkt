# bbkt (Bitbucket CLI & MCP Server)

[![Documentation](https://img.shields.io/badge/docs-reference-blue)](https://zach-snell.github.io/bbkt/)
[![Go Report Card](https://goreportcard.com/badge/github.com/zach-snell/bbkt)](https://goreportcard.com/report/github.com/zach-snell/bbkt)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A complete command-line interface and Model Context Protocol (MCP) server written in Go that provides programmatic integration with Bitbucket Cloud workspaces and repositories.

<p align="center">
  <img src="demo.gif" alt="bbkt CLI demo" width="700" />
</p>

## Features

- **Dual mode** — Use as an interactive CLI for daily work, or launch as an MCP server (`bbkt mcp`) for AI agents (Claude Desktop, Cursor, etc.).
- **Git-aware** — Most commands auto-detect the current workspace and repo from `.git/config`, so `bbkt prs list` Just Works inside a Bitbucket repo.
- **Multi-profile** — Switch between personal and work Atlassian accounts with `--profile` or `BBKT_PROFILE`; auto-selects the right profile based on your git config email.
- **Two auth modes** — Atlassian API tokens (Basic auth) or interactive OAuth 2.0 with automatic token refresh.
- **Scope-aware MCP** — At startup the MCP server introspects your token's granted scopes and silently hides tools the token can't use, preventing the AI from hallucinating successful writes it doesn't have permission for.
- **Read and write** — Repositories, workspaces, pipelines, issues, pull requests, comments, and source code (read/write/delete) — all via the API.

## Installation

### Recommended — curl | bash

Installs the latest release binary for your OS/arch from [GitHub Releases](https://github.com/zach-snell/bbkt/releases).

```bash
# System-wide (uses sudo if /usr/local/bin isn't writable)
curl -sSL https://raw.githubusercontent.com/zach-snell/bbkt/main/install.sh | bash

# User-local, no sudo (~/.local/bin)
curl -sSL https://raw.githubusercontent.com/zach-snell/bbkt/main/install.sh | bash -s -- --user
```

Fish users on `--user`: run `fish_add_path ~/.local/bin` once if it isn't already on `$PATH`.

### From GitHub Releases

Download a prebuilt binary for your OS/arch directly from the [Releases](https://github.com/zach-snell/bbkt/releases) page (Linux/macOS/Windows, amd64 + arm64).

### From source

```bash
git clone https://github.com/zach-snell/bbkt.git
cd bbkt
go build -o bbkt ./cmd/bbkt
```

Requires Go 1.26+.

## Quickstart

```bash
# 1. Authenticate (interactive — prompts for email + Atlassian API token)
bbkt auth

# 2. Verify
bbkt status

# 3. Use it. Inside a Bitbucket repo, workspace/repo are inferred from git.
bbkt prs list
bbkt pipelines trigger --ref-name main
bbkt source read README.md
```

For OAuth instead of an API token:

```bash
export BITBUCKET_OAUTH_CLIENT_ID=<consumer-key>
export BITBUCKET_OAUTH_CLIENT_SECRET=<consumer-secret>
bbkt auth --oauth
```

## Authentication

bbkt stores credentials at `~/.config/bbkt/credentials.json` and supports multiple profiles.

### Atlassian API tokens (default)

```bash
bbkt auth                  # save to "default" profile
bbkt auth --profile work   # save to a named profile
```

> **Important:** Bitbucket Cloud REST API requires *scoped* API tokens since the September 2025 phase-2 of app-password deprecation. At [id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens), use the **"Create API token with scopes"** button — not plain "Create API token". Unscoped tokens authenticate to Atlassian but Bitbucket rejects them; bbkt will detect this and point you here.

Recommended scope set for full read/write:

```
read:account
read:user:bitbucket          read:workspace:bitbucket
read:repository:bitbucket    write:repository:bitbucket
read:pullrequest:bitbucket   write:pullrequest:bitbucket
read:pipeline:bitbucket      write:pipeline:bitbucket
```

### OAuth 2.0 (browser flow)

Register an OAuth consumer in your Bitbucket workspace settings, then:

```bash
export BITBUCKET_OAUTH_CLIENT_ID=<consumer-key>
export BITBUCKET_OAUTH_CLIENT_SECRET=<consumer-secret>
bbkt auth --oauth
```

bbkt opens a browser, captures the callback on `http://localhost:8976`, exchanges the code for tokens, and stores them. Access tokens auto-refresh on expiry. To override the callback port (must match your registered `redirect_uri`):

```bash
export BBKT_OAUTH_CALLBACK_PORT=9876
```

### Multi-profile switching

```bash
bbkt profile                  # list profiles, mark active
bbkt profile use work         # set active profile
bbkt profile refresh          # refresh cached workspace list per profile
BBKT_PROFILE=work bbkt prs list   # one-shot profile override
```

When `BBKT_PROFILE` is unset, bbkt tries to auto-select a profile whose accessible workspaces match your git config email; otherwise it falls back to `active_profile`.

### One-shot env-var auth (CI, scripts)

Setting any of these env vars bypasses the stored profile entirely:

```bash
# API token
export BITBUCKET_USERNAME=you@example.com   # despite the name, this is your Atlassian email
export BITBUCKET_API_TOKEN=ATATT3xFf...

# OAuth bearer
export BITBUCKET_ACCESS_TOKEN=<oauth-access-token>
```

## CLI Usage

Global flags (all commands): `--json` raw JSON output, `--profile <name>` profile override.

```bash
# Auth & profiles
bbkt auth [--oauth] [--profile <name>]     # set up credentials
bbkt status                                # show active profile + token health
bbkt logout                                # remove stored credentials
bbkt profile [use <name> | refresh]        # manage profiles

# Workspaces / repos
bbkt workspaces [list | get <workspace>]
bbkt repos     [list | get | create | delete]
                 [--query <q>] [--role <owner|admin|contributor|member>]

# Pull requests (workspace/repo inferred from git)
bbkt prs list                              # --state OPEN|MERGED|SUPERSEDED|DECLINED
bbkt prs get <pr-id>
bbkt prs create --title <t> --source <branch> [--destination <branch>]
bbkt prs merge <pr-id> [--strategy merge_commit|squash|fast_forward]
bbkt prs approve <pr-id>
bbkt prs decline <pr-id>
bbkt prs comments [list | add | resolve | unresolve]   # --content, --parent, --file, --to, --from

# Pipelines
bbkt pipelines list                        # --status SUCCESSFUL|FAILED|INPROGRESS
bbkt pipelines get <pipeline-uuid>
bbkt pipelines trigger --ref-name <branch> [--ref-type branch|tag|bookmark]
bbkt pipelines stop <pipeline-uuid>
bbkt pipelines steps <pipeline-uuid>
bbkt pipelines log <pipeline-uuid> <step-uuid>

# Issues
bbkt issues [list | get | create | update]
              [--state] [--kind bug|enhancement|proposal|task] [--priority ...]

# Source code
bbkt source read <path> [--ref <ref>]
bbkt source tree [<path>] [--max-depth <n>]
bbkt source search <query>
bbkt source history <path>
bbkt source write <path> --content <text> [--message <m>] [--branch <b>]
bbkt source delete <path> [--message <m>]
```

Most commands prompt interactively (via `huh`) when required arguments are missing. Run `bbkt <command> --help` for full flag details.

## MCP Server

`bbkt mcp` launches a Model Context Protocol server for AI agents. Two transports:

### Stdio (default — for Claude Desktop, Cursor, etc.)

Add to your MCP client config:

```json
{
  "mcpServers": {
    "bitbucket": {
      "command": "/absolute/path/to/bbkt",
      "args": ["mcp"]
    }
  }
}
```

By default the server reads credentials from `~/.config/bbkt/credentials.json` (the profile set up via `bbkt auth`). To override per-client:

```json
{
  "mcpServers": {
    "bitbucket": {
      "command": "/absolute/path/to/bbkt",
      "args": ["mcp"],
      "env": {
        "BBKT_PROFILE": "work"
      }
    }
  }
}
```

### HTTP Streamable (for remote / network clients)

```bash
bbkt mcp --port 8080
```

This serves the MCP Streamable Transport (SSE-based) on the given port.

### Scope-gated tools

The MCP server calls Bitbucket's `/user` endpoint at startup to introspect your token's granted scopes, then **silently drops any tool whose required scope is missing**. This prevents the AI from confidently calling, say, `manage_pipelines` (`pipeline` scope) on a read-only token and getting a `403` it can't recover from.

To explicitly deny tools even when scopes allow them:

```bash
export BITBUCKET_DISABLED_TOOLS="manage_repositories,manage_pipelines"
```

To skip credential loading entirely (tools will return auth-required errors when invoked — useful for testing the transport):

```bash
bbkt mcp --no-auth
```

## Environment Variables

| Variable | Purpose | Required |
|---|---|---|
| `BITBUCKET_USERNAME` | Atlassian email (despite the legacy name) — used with `BITBUCKET_API_TOKEN` | Only for env-var API-token auth |
| `BITBUCKET_API_TOKEN` | Atlassian scoped API token | Only for env-var API-token auth |
| `BITBUCKET_ACCESS_TOKEN` | OAuth 2.0 bearer token (overrides stored profile) | Only for env-var OAuth |
| `BITBUCKET_OAUTH_CLIENT_ID` | OAuth consumer Key | Only for `bbkt auth --oauth` |
| `BITBUCKET_OAUTH_CLIENT_SECRET` | OAuth consumer Secret | Only for `bbkt auth --oauth` |
| `BBKT_PROFILE` | Profile name override (one-shot) | No |
| `BBKT_OAUTH_CALLBACK_PORT` | Local callback port for OAuth flow (default 8976) | No |
| `BITBUCKET_DISABLED_TOOLS` | Comma-separated MCP tools to disable | No |

When `BITBUCKET_ACCESS_TOKEN` *or* (`BITBUCKET_USERNAME` + `BITBUCKET_API_TOKEN`) is set, the stored profile is bypassed entirely.

## Tools Provided (MCP)

| Tool | Operations | Required Scope |
|---|---|---|
| `manage_workspaces` | list, get | — |
| `manage_repositories` | list, get, create, delete | `repository` |
| `manage_refs` | list, create, delete branches and tags | `repository` |
| `manage_commits` | list, get, diff, diffstat | `repository` |
| `manage_source` | read, list_directory, get_history, search, write, delete | `repository` |
| `manage_pull_requests` | list, get, create, update, merge, approve, unapprove, decline, diff, diffstat, commits | `pullrequest` |
| `manage_pr_comments` | list, create, update, delete, resolve, unresolve | `pullrequest` |
| `manage_pipelines` | list, get, trigger, stop, list-steps, get-step-log | `pipeline` |
| `manage_issues` | list, get, create, update | `issue` |

Scopes shown are the OAuth-style names. For Atlassian API tokens, the equivalent granular scopes are `read:<scope>:bitbucket` / `write:<scope>:bitbucket`.

## Development

Requires Go 1.26+.

```bash
go test ./...                    # unit tests
go test -tags=live ./...         # live integration tests (requires BBKT_LIVE_* secrets)
golangci-lint run ./...
```

The repo also has a docs site (Astro) under `docs/` deployed to <https://zach-snell.github.io/bbkt/>.

## License

Apache 2.0 — see [LICENSE](LICENSE).
