package bitbucket

import "testing"

func TestParseBitbucketRemote(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantWS   string
		wantRepo string
		wantOK   bool
	}{
		// scp-like literal host
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
		// https with/without user@, with/without .git
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
			name:     "https with user@ without .git",
			url:      "https://someuser@bitbucket.org/myws/myrepo",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		// URL-scheme form using the ssh transport
		{
			name:     "ssh scheme with .git",
			url:      "ssh://git@bitbucket.org/myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		{
			name:     "ssh scheme without .git",
			url:      "ssh://git@bitbucket.org/myws/myrepo",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		// SSH config host alias (the common multi-account pattern)
		{
			name:     "ssh scheme host alias",
			url:      "ssh://git@bitbucket.org-work/myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		{
			name:     "scp host alias",
			url:      "git@bitbucket.org-work:myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		{
			name:     "scp host alias without .git",
			url:      "git@bitbucket.org-work:ws/repo",
			wantWS:   "ws",
			wantRepo: "repo",
			wantOK:   true,
		},
		// Over-match negatives — look-alike hosts must NOT be treated as Bitbucket
		{
			name:   "evil prefix host",
			url:    "git@evilbitbucket.org:ws/repo.git",
			wantOK: false,
		},
		{
			name:   "not prefix host",
			url:    "git@notbitbucket.org:ws/repo.git",
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
		{
			// Dash-then-dotted: the alias-suffix branch must be dotless, so the
			// foreign domain after the dash is rejected (the suffix stops at the
			// dot and the required separator then fails on ".attacker.com").
			name:   "dash-dotted attacker suffix scp",
			url:    "git@bitbucket.org-foo.attacker.com:ws/repo.git",
			wantOK: false,
		},
		{
			name:   "dash-dotted attacker suffix ssh scheme",
			url:    "ssh://git@bitbucket.org-foo.attacker.com/ws/repo.git",
			wantOK: false,
		},
		{
			name:   "dash-dotted attacker suffix https",
			url:    "https://bitbucket.org-evil.attacker.com/ws/repo.git",
			wantOK: false,
		},
		{
			name:   "evil prefix https",
			url:    "https://evilbitbucket.org/ws/repo.git",
			wantOK: false,
		},
		// Other hosts are not Bitbucket
		{
			name:   "github scp not bitbucket",
			url:    "git@github.com:ws/repo.git",
			wantOK: false,
		},
		{
			name:   "github https not bitbucket",
			url:    "https://github.com/ws/repo.git",
			wantOK: false,
		},
		{
			name:   "gitlab scp not bitbucket",
			url:    "git@gitlab.com:ws/repo.git",
			wantOK: false,
		},
		{
			name:   "empty url",
			url:    "",
			wantOK: false,
		},
		// ssh scheme with explicit port (valid SSH URL syntax)
		{
			name:     "ssh scheme with default port",
			url:      "ssh://git@bitbucket.org:22/myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		{
			name:     "ssh scheme host alias with port",
			url:      "ssh://git@bitbucket.org-work:22/myws/myrepo.git",
			wantWS:   "myws",
			wantRepo: "myrepo",
			wantOK:   true,
		},
		// Extra path/query/fragment must not be swallowed into the repo slug.
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
