package bitbucket

import "testing"

// fakeSSHConfig maps SSH host aliases to the hostname `ssh -G` would resolve
// them to. Anything not present echoes back the input, mirroring how ssh
// treats a host with no matching config entry.
var fakeSSHConfig = map[string]string{
	"bb-cloud":          "bitbucket.org", // arbitrarily-named alias → Bitbucket
	"work":              "bitbucket.org",
	"bitbucket.org-two": "bitbucket.org", // dashed alias resolves (not by spelling)
	"bad-alias":         "github.com",    // alias that points elsewhere
	"gh-personal":       "github.com",
}

// withFakeResolver installs a hermetic SSH resolver for the duration of a test
// so results never depend on the machine's real ~/.ssh/config.
func withFakeResolver(t *testing.T) {
	t.Helper()
	prev := resolveSSHHostname
	resolveSSHHostname = func(host string) string {
		if h, ok := fakeSSHConfig[host]; ok {
			return h
		}
		return host
	}
	t.Cleanup(func() { resolveSSHHostname = prev })
}

func TestParseBitbucketRemote(t *testing.T) {
	withFakeResolver(t)

	tests := []struct {
		name     string
		url      string
		wantWS   string
		wantRepo string
		wantOK   bool
	}{
		// --- Literal host (no alias resolution needed) ---
		{
			name:     "scp literal with .git",
			url:      "git@bitbucket.org:myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		{
			name:     "scp literal without .git",
			url:      "git@bitbucket.org:myws/myrepo",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		{
			name:     "https with .git",
			url:      "https://bitbucket.org/myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		{
			name:     "https without .git",
			url:      "https://bitbucket.org/myws/myrepo",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		{
			name:     "https with user@ and .git",
			url:      "https://someuser@bitbucket.org/myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		{
			name:     "ssh scheme with .git",
			url:      "ssh://git@bitbucket.org/myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		{
			name:     "ssh scheme with default port",
			url:      "ssh://git@bitbucket.org:22/myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},

		// --- SSH aliases resolved via ssh config (the real feature) ---
		{
			name:     "scp arbitrarily-named alias resolves to Bitbucket",
			url:      "git@bb-cloud:myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		{
			name:     "ssh scheme arbitrarily-named alias resolves to Bitbucket",
			url:      "ssh://git@work/teamx/appx.git",
			wantWS:   "teamx",
			wantRepo: "appx",
			wantOK:   true,
		},
		{
			name:     "dashed alias is accepted via resolution, not spelling",
			url:      "git@bitbucket.org-two:myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		{
			name:     "alias with explicit port resolves",
			url:      "ssh://git@bb-cloud:22/myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},

		// --- Aliases that resolve elsewhere must be rejected ---
		{
			// An alias literally spelled "bitbucket.org-*" but pointing at
			// another host is NOT trusted by name — it resolves to github.com.
			name:   "alias resolving elsewhere is rejected",
			url:    "git@bad-alias:myws/myrepo.git",
			wantOK: false,
		},
		{
			// A dashed name with no config entry resolves to itself, so it is
			// no longer blindly trusted the way a spelling match would be.
			name:   "unconfigured dashed alias is rejected",
			url:    "git@bitbucket.org-unknown:myws/myrepo.git",
			wantOK: false,
		},

		// --- https never resolves SSH aliases (literal DNS only) ---
		{
			name:   "https alias name is not resolved",
			url:    "https://work/myws/myrepo.git",
			wantOK: false,
		},

		// --- Look-alike hosts must never be treated as the target ---
		{
			name:   "evil prefix host",
			url:    "git@evilbitbucket.org:ws/repo.git",
			wantOK: false,
		},
		{
			name:   "dotted attacker suffix scp",
			url:    "git@bitbucket.org.attacker.com:ws/repo.git",
			wantOK: false,
		},
		{
			name:   "dotted attacker suffix https",
			url:    "https://bitbucket.org.attacker.com/ws/repo.git",
			wantOK: false,
		},

		// --- Non-target hosts ---
		{
			name:   "github scp is not the target host",
			url:    "git@github.com:ws/repo.git",
			wantOK: false,
		},
		{
			name:   "github https is not the target host",
			url:    "https://github.com/ws/repo.git",
			wantOK: false,
		},
		{
			name:   "configured github alias is not the target host",
			url:    "git@gh-personal:ws/repo.git",
			wantOK: false,
		},

		// --- Extra path/query/fragment must not be swallowed into the slug ---
		{
			name:   "https deep path is rejected",
			url:    "https://bitbucket.org/ws/repo/src/main/README.md",
			wantOK: false,
		},
		{
			name:   "https repo with query string is rejected",
			url:    "https://bitbucket.org/ws/repo.git?foo=bar",
			wantOK: false,
		},
		{
			name:   "https repo with fragment is rejected",
			url:    "https://bitbucket.org/ws/repo.git#frag",
			wantOK: false,
		},
		{
			name:   "scp deep path is rejected",
			url:    "git@bitbucket.org:ws/repo/extra",
			wantOK: false,
		},

		{
			name:   "empty url",
			url:    "",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ws, repo, ok := parseBitbucketRemote(tc.url)
			if ok != tc.wantOK {
				t.Fatalf("parseBitbucketRemote(%q) ok = %v, want %v (ws=%q repo=%q)",
					tc.url, ok, tc.wantOK, ws, repo)
			}
			if !tc.wantOK {
				return
			}
			if ws != tc.wantWS || repo != tc.wantRepo {
				t.Fatalf("parseBitbucketRemote(%q) = (%q, %q), want (%q, %q)",
					tc.url, ws, repo, tc.wantWS, tc.wantRepo)
			}
		})
	}
}

// TestGitHostOverride verifies BBKT_HOST swaps the recognized host, including
// resolving an SSH alias to that custom host.
func TestGitHostOverride(t *testing.T) {
	t.Setenv("BBKT_HOST", "git.example.com")

	prev := resolveSSHHostname
	resolveSSHHostname = func(host string) string {
		if host == "selfhost" {
			return "git.example.com"
		}
		return host
	}
	t.Cleanup(func() { resolveSSHHostname = prev })

	// Literal custom host is recognized; the default host no longer is.
	if _, _, ok := parseBitbucketRemote("git@git.example.com:ws/repo.git"); !ok {
		t.Errorf("expected custom host to be recognized under BBKT_HOST override")
	}
	if _, _, ok := parseBitbucketRemote("git@bitbucket.org:ws/repo.git"); ok {
		t.Errorf("expected default bitbucket.org to be rejected under BBKT_HOST override")
	}
	// An alias resolving to the custom host is recognized.
	if _, _, ok := parseBitbucketRemote("git@selfhost:ws/repo.git"); !ok {
		t.Errorf("expected alias resolving to custom host to be recognized")
	}
}
