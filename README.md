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
- **Scope-aware MCP** — At startup the MCP server introspects your token's granted scopes and hides tools whose read scope the token lacks. This is a usability filter, not a write guard: limit what an agent can change with a least-privilege credential (see [Scope-gated tools](#scope-gated-tools)).
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

# no terminal (scripts, CI): pipe the token in instead of typing it
printf '%s\n' "$TOKEN" | bbkt auth --profile ci --email you@example.com --with-token
```

Profiles live side by side; the first one you create is active. Use another with `bbkt --profile work …`, `bbkt profile use work`, or `BBKT_PROFILE=work` in an MCP server's environment.

> **Important:** Bitbucket Cloud REST API requires *scoped* API tokens since the September 2025 phase-2 of app-password deprecation. At [id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens), use the **"Create API token with scopes"** button — not plain "Create API token". Unscoped tokens authenticate to Atlassian but Bitbucket rejects them; bbkt will detect this and point you here.

Read scopes (browse repos, PRs and pipelines; nothing can change):

```
read:account                 read:user:bitbucket
read:workspace:bitbucket     read:repository:bitbucket
read:pullrequest:bitbucket   read:pipeline:bitbucket
```

Add these to open and merge PRs, comment, commit files, delete branches and run pipelines (what most agent setups use):

```
write:repository:bitbucket   write:pullrequest:bitbucket   write:pipeline:bitbucket
```

Leave `admin:*` and `delete:*` scopes off unless you need them. A write-scoped token can merge PRs and commit to any unprotected branch, so protect `main`/`develop` with Bitbucket branch restrictions: see [Securing agent access](https://zach-snell.github.io/bbkt/guides/securing-agents/).

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
bbkt pipelines trigger --ref-name main --pattern build --var FLAVOUR=release [--secured-var KEY=VALUE]
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

### HTTP Streamable (for clients that can't spawn a process)

```bash
bbkt mcp --port 8080
```

This serves the MCP Streamable Transport (SSE-based) on `127.0.0.1:8080`.

- **Bearer token required.** Every request must send `Authorization: Bearer <token>`; anything else gets `401`. Set the token with `BBKT_MCP_HTTP_TOKEN` (at least 16 characters). If it is unset on loopback, bbkt generates a random token and prints it once to stderr.
- **Loopback only by default.** `--host <addr>` changes the bind address. A non-loopback address is refused unless you pass `--allow-remote` **and** set `BBKT_MCP_HTTP_TOKEN`:

  ```bash
  export BBKT_MCP_HTTP_TOKEN="$(openssl rand -hex 32)"
  bbkt mcp --port 8080 --host 0.0.0.0 --allow-remote
  ```

- **Cleartext.** There is no TLS; the token and all Bitbucket data travel unencrypted. Put a remote bind behind a TLS-terminating reverse proxy.

### Scope-gated tools

The MCP server calls Bitbucket's `/user` endpoint at startup to introspect your token's granted scopes, then drops any tool whose **read** scope is missing (e.g. `manage_pipelines` without `pipeline`). If introspection fails, the server refuses to start; set `BBKT_SKIP_SCOPE_CHECK=1` to start anyway, in which case no scope filtering happens and every tool is registered.

> [!IMPORTANT]
> Scope filtering does not hide write actions. A read-only token still exposes `create`, `delete`, `merge`, `write_file`, `trigger` and the other write actions inside the multiplexed `manage_*` tools; they fail at the API instead. The credential is what limits an agent, so use **least-privilege credentials** for agent setups, ideally a Bitbucket [repository or workspace access token](https://support.atlassian.com/bitbucket-cloud/docs/access-tokens/) passed via `BITBUCKET_ACCESS_TOKEN`. Repository and workspace access tokens are not tied to a user; if introspection fails for one and startup is refused, set `BBKT_SKIP_SCOPE_CHECK=1`. Every tool is then registered, and Bitbucket still enforces the token's own permissions. For scope choices and Bitbucket branch protection, see [Securing agent access](https://zach-snell.github.io/bbkt/guides/securing-agents/).

To remove tools outright, regardless of scopes:

```bash
export BITBUCKET_DISABLED_TOOLS="manage_repositories,manage_pipelines,bitbucket_api"
```

Valid names: `manage_workspaces`, `manage_repositories`, `manage_refs`, `manage_commits`, `manage_pull_requests`, `manage_pr_comments`, `manage_source`, `manage_pipelines`, `manage_issues`, `bitbucket_api`. Unknown names are rejected at startup.

### `bitbucket_api` passthrough

`bitbucket_api` gives the agent authenticated access to any Bitbucket v2 endpoint without a typed tool. Only `GET`/`HEAD` are allowed by default; set `BBKT_API_ALLOW_WRITE=1` on the server to allow `POST`/`PUT`/`PATCH`/`DELETE`, which lets the agent call any write endpoint the credential permits.

To skip credential loading entirely (tools will return auth-required errors when invoked — useful for testing the transport):

```bash
bbkt mcp --no-auth
```

## Environment Variables

| Variable | Purpose | Required |
|---|---|---|
| `BITBUCKET_USERNAME` | Atlassian email (despite the legacy name) — used with `BITBUCKET_API_TOKEN` | Only for env-var API-token auth |
| `BITBUCKET_API_TOKEN` | Atlassian scoped API token | Only for env-var API-token auth |
| `BITBUCKET_ACCESS_TOKEN` | Bearer token: OAuth 2.0 access token or repository/workspace access token (overrides stored profile) | Only for env-var bearer auth |
| `BITBUCKET_OAUTH_CLIENT_ID` | OAuth consumer Key | Only for `bbkt auth --oauth` |
| `BITBUCKET_OAUTH_CLIENT_SECRET` | OAuth consumer Secret | Only for `bbkt auth --oauth` |
| `BBKT_PROFILE` | Profile name override (one-shot) | No |
| `BBKT_OAUTH_CALLBACK_PORT` | Local callback port for OAuth flow (default 8976) | No |
| `BITBUCKET_DISABLED_TOOLS` | Comma-separated MCP tools to remove (names above; unknown names rejected at startup) | No |
| `BBKT_API_ALLOW_WRITE` | `1` allows write methods through the `bitbucket_api` MCP passthrough | No |
| `BBKT_SKIP_SCOPE_CHECK` | `1` starts the MCP server even if scope introspection fails | No |
| `BBKT_MCP_HTTP_TOKEN` | Bearer token for `bbkt mcp --port` (generated on loopback if unset) | Required for `--allow-remote` |

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
| `bitbucket_api` | any v2 endpoint; reads only unless `BBKT_API_ALLOW_WRITE=1` | — |

"Required Scope" is the read scope that keeps a tool registered; write actions inside a tool still need the matching write scope at call time. Scopes shown are the OAuth-style names. For Atlassian API tokens, the equivalent granular scopes are `read:<scope>:bitbucket` / `write:<scope>:bitbucket`.

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
