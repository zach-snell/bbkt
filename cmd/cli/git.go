package cli

import (
	"fmt"

	"github.com/zach-snell/bbkt/internal/bitbucket"
)

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
		ws, rs, err := bitbucket.GetLocalRepoInfo()
		if err != nil {
			return "", "", nil, fmt.Errorf("could not infer workspace/repo from git: %v", err)
		}
		return ws, rs, args, nil
	}

	return "", "", nil, fmt.Errorf("expected either %d arguments (infer repo from git) or %d arguments (explicit workspace and repo)", trailingArgsCount, trailingArgsCount+2)
}
