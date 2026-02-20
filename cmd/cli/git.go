package cli

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

// ParseArgs resolves workspace and repoSlug from the command line arguments,
// and returns any remaining trailing arguments.
// It expects either:
// - exact trailingArgsCount arguments (infers workspace and repoSlug from git)
// - trailingArgsCount + 2 arguments (explicit workspace and repoSlug provided first)
func ParseArgs(args []string, trailingArgsCount int) (workspace, repoSlug string, trailing []string, err error) {
	if len(args) == trailingArgsCount+2 {
		return args[0], args[1], args[2:], nil
	}

	if len(args) == trailingArgsCount {
		ws, rs, err := GetLocalRepoInfo()
		if err != nil {
			return "", "", nil, fmt.Errorf("could not infer workspace/repo from git: %v", err)
		}
		return ws, rs, args, nil
	}

	return "", "", nil, fmt.Errorf("expected either %d arguments (infer repo from git) or %d arguments (explicit workspace and repo)", trailingArgsCount, trailingArgsCount+2)
}
