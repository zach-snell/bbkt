package bitbucket

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var (
	// Matches git@bitbucket.org:workspace/repo-slug.git
	sshRegex = regexp.MustCompile(`git@bitbucket\.org:([^/]+)/([^/]+)\.git`)

	// Matches https://[user@]bitbucket.org/workspace/repo-slug.git
	httpsRegex = regexp.MustCompile(`https://.*bitbucket\.org/([^/]+)/([^/]+)\.git`)

	// Matches git@bitbucket.org:workspace/repo-slug
	sshRegexNoGit = regexp.MustCompile(`git@bitbucket\.org:([^/]+)/([^/]+)`)

	// Matches https://[user@]bitbucket.org/workspace/repo-slug
	httpsRegexNoGit = regexp.MustCompile(`https://.*bitbucket\.org/([^/]+)/([^/]+)`)
)

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

		// Try SSH format
		if matches := sshRegex.FindStringSubmatch(url); len(matches) == 3 {
			return matches[1], matches[2], nil
		}

		// Try HTTPS format
		if matches := httpsRegex.FindStringSubmatch(url); len(matches) == 3 {
			return matches[1], matches[2], nil
		}

		// Try without .git
		if matches := sshRegexNoGit.FindStringSubmatch(url); len(matches) == 3 {
			return matches[1], matches[2], nil
		}
		if matches := httpsRegexNoGit.FindStringSubmatch(url); len(matches) == 3 {
			return matches[1], matches[2], nil
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
