# bbkt (Bitbucket CLI & MCP Server)

A complete command-line interface and Model Context Protocol (MCP) server written in Go that provides programmatic integration with Bitbucket workspaces and repositories.

## Features

- **Dual Mode**: Run as a rich, interactive CLI tool for daily developer tasks, or as an MCP server for AI agents.
- **Git Awareness**: Automatically detects your current Bitbucket repository from `.git/config` when run from the terminal.
- **Interactive UI**: Sleek terminal UI wizards trigger automatically when required arguments are omitted.
- **Read/Write Operations**: Seamlessly manage repositories, workspaces, pipelines, issues, and pull requests. Modify or delete repository source code directly from the API.
- **Authentication**: Supports standard App Passwords or an interactive OAuth 2.0 web flow for desktop users.

## Installation

### From Source
```bash
# Clone the repository
git clone https://github.com/zach-snell/bbkt.git
cd bbkt

# Run the install script (builds and moves to ~/.local/bin)
./install.sh
```

Ensure `~/.local/bin` is added to your system `$PATH` for the executable to be universally available.

### From GitHub Releases
Download the appropriate binary for your system (Linux, macOS, Windows) from the [Releases](https://github.com/zach-snell/bbkt/releases) page.

## CLI Usage

`bbkt` provides a robust command-line interface with the following core modules:

```bash
# Manage workspaces
bbkt workspaces [list, get]

# Manage repositories
bbkt repos [list, get, create, delete]

# Manage pull requests and comments
bbkt prs [list, get, create, merge, approve, decline]
bbkt prs comments [list, add, resolve]

# Trigger and view pipelines
bbkt pipelines [list, get, trigger, stop, logs]

# Issue tracking
bbkt issues [list, get, create, update]

# Read, search, and edit source code
bbkt source [read, tree, search, history, write, delete]
```

## MCP Usage

The tool also serves as an MCP server. It supports two protocols: Stdio (default via `bbkt mcp`) and the official Streamable Transport API over HTTP.

### Stdio Transport (Default)
If you intend to use this with an MCP client (such as Claude Desktop or Cursor), add it to your client's configuration file as a local command:

```json
{
  "mcpServers": {
    "bitbucket": {
      "command": "/absolute/path/to/bbkt",
      "args": ["mcp"],
      "env": {
        "BITBUCKET_USERNAME": "your-username",
        "BITBUCKET_API_TOKEN": "your-api-token"
      }
    }
  }
}
```

### Streamable Transport (HTTP)
You can run the server as a long-lived HTTP process serving the Streamable Transport API (which uses Server-Sent Events underneath). This is useful for remote network clients.

```bash
bbkt mcp --port 8080
```

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `BITBUCKET_USERNAME` | Your Bitbucket username | No (but recommended for API Tokens) |
| `BITBUCKET_API_TOKEN` | An Atlassian API Token | No (If omitted, triggers OAuth 2.0 browser flow) |
| `BITBUCKET_CLIENT_ID` | OAuth 2.0 Client ID | Only if using OAuth |
| `BITBUCKET_CLIENT_SECRET` | OAuth 2.0 Client Secret | Only if using OAuth |

### API Token Scopes & Security

When creating your Atlassian API Token, we recommend granting the following scopes for a balanced experience:

`read:account, read:user, read:repository:bitbucket, write:repository:bitbucket, read:pullrequest:bitbucket, write:pullrequest:bitbucket, read:pipeline:bitbucket, write:pipeline:bitbucket`

**Token Introspection:** The `bbkt mcp` server dynamically evaluates your API token's granted scopes at startup. If you omit specific permissions (like `write:pipeline:bitbucket`), the server will completely hide the associated MCP tools (`trigger_pipeline`, `stop_pipeline`) from the AI agent to prevent hallucinated successes.

**Explicit Tool Denial:** Even if your token has full admin privileges, you can explicitly deny the AI agent access to any tool using the `BITBUCKET_DISABLED_TOOLS` environment variable.

```bash
export BITBUCKET_DISABLED_TOOLS="delete_repository,delete_branch,delete_file"
```

## Tools Provided

- `manage_workspaces`: Getting and listing Bitbucket workspaces
- `manage_repositories`: Listing, getting, creating, and deleting repositories
- `manage_refs`: Listing, creating, and deleting branches and tags
- `manage_commits`: Listing and getting commits, diffs, and diffstats
- `manage_source`: Source code operations (read, list_directory, get_history, search, write, delete)
- `manage_pull_requests`: All pull request operations (list, get, create, update, merge, approve, unapprove, decline, diff, diffstat, commits)
- `manage_pr_comments`: Managing pull request comments (list, create, update, delete, resolve, unresolve)
- `manage_pipelines`: Managing Bitbucket Pipelines (list, get, trigger, stop, list-steps, get-step-log)
- `manage_issues`: Managing repository issues (list, get, create, update)

## Development

Requirements:
- Go 1.25+

```bash
# Run tests
go test ./...

# Run the linter
golangci-lint run ./...
```
