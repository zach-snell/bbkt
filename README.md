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

## Tools Provided

- `list_repositories`: List repositories in a workspace.
- `list_pull_requests`: List pull requests for a repository.
- `create_pull_request`: Open a new pull request.
- `merge_pull_request`: Merge an existing pull request.
- `approve_pull_request`: Approve a pull request.
- `create_pr_comment`: Reply to or add an inline comment on a pull request.
- `list_pr_commits`: See exactly which commits are included in a PR.
- `list_pipelines`: View build/deployment pipelines across a repository.
- `get_pipeline`: Fetch specific details about a pipeline run.
- `list_branches`: View existing branches.
- `list_commits`: View recent commits to a repository.
- `get_diffstat`: See file additions and deletions for a specific commit or PR.

## Development

Requirements:
- Go 1.25+

```bash
# Run tests
go test ./...

# Run the linter
golangci-lint run ./...
```
