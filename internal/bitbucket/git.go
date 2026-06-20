package bitbucket

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// hostPattern matches a Bitbucket Cloud host, anchored to avoid over-matching
// look-alike hosts (evilbitbucket.org, notbitbucket.org, bitbucket.org.attacker.com).
//
// The host segment must be exactly "bitbucket.org" or an SSH config host alias
// of the form "bitbucket.org-<suffix>" (the common multi-account pattern, e.g.
// "bitbucket.org-work"). The alias suffix is dash-introduced and dotless, so
// a dotted look-alike like "bitbucket.org.attacker.com" OR a dash-then-dotted
// one like "bitbucket.org-foo.attacker.com" is NOT accepted (the suffix class
// excludes ".", so the host pattern stops before any dot and the required ":" or
// "/" separator then fails to match the foreign domain).
//
// A left boundary (start of host, "@", or "//") is required so that
// "evilbitbucket.org" / "notbitbucket.org" cannot match.
const hostPattern = `bitbucket\.org(?:-[A-Za-z0-9_-]+)?`

var (
	// scp-like form: [user@]<host>:workspace/repo[.git]
	//   git@bitbucket.org:ws/repo.git
	//   git@bitbucket.org-work:ws/repo
	//
	// workspace and repo are each a single path segment (no "/", "?" or "#"),
	// so extra path/query/fragment junk is rejected rather than swallowed into
	// the repo slug.
	scpRemoteRegex = regexp.MustCompile(
		`^(?:[^@/]+@)?` + hostPattern + `:([^/?#]+)/([^/?#]+?)(?:\.git)?/?$`,
	)

	// URL-scheme form: <scheme>://[user@]<host>[:port]/workspace/repo[.git]
	//   ssh://git@bitbucket.org-work/myws/myrepo.git
	//   ssh://git@bitbucket.org:22/ws/repo.git
	//   https://bitbucket.org/ws/repo.git
	//   https://user@bitbucket.org/ws/repo
	//
	// An optional :<port> is allowed after the host (valid SSH/HTTPS URL
	// syntax); workspace and repo are single segments as above.
	urlRemoteRegex = regexp.MustCompile(
		`^(?:ssh|https?)://(?:[^@/]+@)?` + hostPattern + `(?::[0-9]+)?/([^/?#]+)/([^/?#]+?)(?:\.git)?/?$`,
	)
)

// parseBitbucketRemote parses a single git remote URL and returns the Bitbucket
// workspace and repo slug if the URL points at Bitbucket Cloud. ok is false for
// any URL that is not a recognized Bitbucket remote (including look-alike hosts).
//
// Supported forms (all with or without a trailing .git):
//   - scp-like:    git@bitbucket.org:ws/repo
//   - https:       https://[user@]bitbucket.org/ws/repo
//   - ssh scheme:  ssh://git@bitbucket.org/ws/repo
//   - host alias:  git@bitbucket.org-work:ws/repo (and the ssh:// form)
func parseBitbucketRemote(url string) (workspace, repoSlug string, ok bool) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", "", false
	}

	if m := urlRemoteRegex.FindStringSubmatch(url); len(m) == 3 {
		return m[1], m[2], true
	}
	if m := scpRemoteRegex.FindStringSubmatch(url); len(m) == 3 {
		return m[1], m[2], true
	}
	return "", "", false
}

// GetLocalRepoInfo attempts to parse the Bitbucket workspace and repo slug
// from the local git repository's remotes. It checks all remotes, prioritizing Bitbucket.
func GetLocalRepoInfo() (workspace, repoSlug string, err error) {
	cmd := exec.Command("git", "remote", "-v")
	output, err := cmd.Output()
	if err != nil {
		return "", "", errors.New("not a git repository or no remotes configured")
	}

	lines := strings.Split(string(output), "\n")
	var foundOtherHosts []string

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		url := fields[1]

		if ws, repo, ok := parseBitbucketRemote(url); ok {
			return ws, repo, nil
		}

		if strings.Contains(url, "github.com") {
			foundOtherHosts = append(foundOtherHosts, "GitHub")
		} else if strings.Contains(url, "gitlab.com") {
			foundOtherHosts = append(foundOtherHosts, "GitLab")
		}
	}

	if len(foundOtherHosts) > 0 {
		return "", "", fmt.Errorf("detected a %s repository. bbkt only supports Bitbucket Cloud repositories", foundOtherHosts[0])
	}

	return "", "", errors.New("no Bitbucket remotes found in the local repository")
}
