package bitbucket

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// defaultGitHost is the canonical Bitbucket Cloud hostname. It is the default
// host a git remote must point at to be recognized as a bbkt remote.
const defaultGitHost = "bitbucket.org"

// gitHost returns the host that git remotes must resolve to in order to be
// recognized. It defaults to Bitbucket Cloud but can be overridden with
// BBKT_HOST for a self-hosted or alternate domain.
//
// Note: this only governs local git-remote recognition. The API base URL and
// OAuth endpoints are configured separately; pointing bbkt at a genuinely
// different provider needs those too.
func gitHost() string {
	if h := strings.TrimSpace(os.Getenv("BBKT_HOST")); h != "" {
		return h
	}
	return defaultGitHost
}

var (
	// scp-like form: [user@]<host>:workspace/repo[.git]
	//   git@<host>:ws/repo.git
	//   git@<alias>:ws/repo          (any-named SSH config host alias)
	//
	// host, workspace and repo are each a single segment; extra
	// path/query/fragment junk is rejected rather than swallowed into the slug.
	scpRemoteRegex = regexp.MustCompile(
		`^(?:[^@/]+@)?([^@/:]+):([^/?#]+)/([^/?#]+?)(?:\.git)?/?$`,
	)

	// URL-scheme form: <scheme>://[user@]<host>[:port]/workspace/repo[.git]
	//   ssh://git@<alias>/ws/repo.git
	//   ssh://git@<host>:22/ws/repo.git
	//   https://<host>/ws/repo.git
	//   https://user@<host>/ws/repo
	urlRemoteRegex = regexp.MustCompile(
		`^(ssh|https?)://(?:[^@/]+@)?([^@/:]+)(?::[0-9]+)?/([^/?#]+)/([^/?#]+?)(?:\.git)?/?$`,
	)
)

// resolveSSHHostname resolves an SSH host (which may be a ~/.ssh/config Host
// alias) to its effective hostname via `ssh -G`. `ssh -G` only parses config;
// it makes no network connection. For a host with no alias entry, ssh echoes
// the input back as the hostname, so an unaliased look-alike does not resolve
// to the target host. It is a package var so tests can inject a fake resolver.
//
// On any failure (ssh missing, timeout, unparseable output) it returns the
// input unchanged — callers then compare against the literal host, so a real
// alias simply won't be recognized rather than being mis-resolved.
var resolveSSHHostname = func(host string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ssh", "-G", host).Output()
	if err != nil {
		return host
	}
	// `ssh -G` prints one "keyword value" pair per line; we want "hostname".
	for _, line := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(line, "hostname "); ok {
			if h := strings.TrimSpace(v); h != "" {
				return h
			}
		}
	}
	return host
}

// hostMatches reports whether host points at the recognized git host.
// sshAliasable is true for the scp-like and ssh:// forms, where host may be an
// SSH config alias worth resolving; it is false for https, whose host is always
// a literal DNS name (no SSH aliasing) and so must match directly.
func hostMatches(host string, sshAliasable bool) bool {
	target := gitHost()
	if strings.EqualFold(host, target) {
		return true
	}
	if sshAliasable {
		return strings.EqualFold(resolveSSHHostname(host), target)
	}
	return false
}

// parseBitbucketRemote parses a single git remote URL and returns the workspace
// and repo slug if the URL points at the recognized git host. ok is false for
// any URL that does not.
//
// Recognition is by the effective host, not by spelling: the literal target
// host, or an SSH config alias (any name) whose resolved hostname is the target
// host, is accepted; anything else — including a look-alike host or an alias
// that resolves elsewhere — is rejected.
//
// Supported forms (all with or without a trailing .git):
//   - scp-like:    git@<host>:ws/repo
//   - https:       https://[user@]<host>/ws/repo
//   - ssh scheme:  ssh://git@<host>[:port]/ws/repo
//   - host alias:  git@<alias>:ws/repo (and the ssh:// form), any alias name
func parseBitbucketRemote(url string) (workspace, repoSlug string, ok bool) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", "", false
	}

	if m := urlRemoteRegex.FindStringSubmatch(url); len(m) == 5 {
		scheme, host, ws, repo := m[1], m[2], m[3], m[4]
		if hostMatches(host, scheme == "ssh") {
			return ws, repo, true
		}
		return "", "", false
	}
	if m := scpRemoteRegex.FindStringSubmatch(url); len(m) == 4 {
		host, ws, repo := m[1], m[2], m[3]
		if hostMatches(host, true) {
			return ws, repo, true
		}
		return "", "", false
	}
	return "", "", false
}

// GetLocalRepoInfo attempts to parse the workspace and repo slug from the local
// git repository's remotes. It checks all remotes and returns the first that
// points at the recognized git host.
func GetLocalRepoInfo() (workspace, repoSlug string, err error) {
	cmd := exec.Command("git", "remote", "-v")
	output, err := cmd.Output()
	if err != nil {
		return "", "", errors.New("not a git repository or no remotes configured")
	}

	lines := strings.Split(string(output), "\n")
	sawRemote := false

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sawRemote = true

		if ws, repo, ok := parseBitbucketRemote(fields[1]); ok {
			return ws, repo, nil
		}
	}

	if sawRemote {
		return "", "", errors.New("no remote pointing at the Bitbucket host was found in this repository")
	}
	return "", "", errors.New("no git remotes configured in this repository")
}
