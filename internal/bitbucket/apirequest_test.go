package bitbucket

import "testing"

func TestNormalizeAPIPath(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"bare no slash", "repositories/ws/repo", "/repositories/ws/repo", false},
		{"leading slash", "/repositories/ws/repo", "/repositories/ws/repo", false},
		{"with 2.0 prefix", "/2.0/repositories/ws/repo", "/repositories/ws/repo", false},
		{"bare 2.0 prefix", "2.0/repositories/ws/repo", "/repositories/ws/repo", false},
		{"full api url", "https://api.bitbucket.org/2.0/repositories/ws/repo", "/repositories/ws/repo", false},
		{"full api url with query", "https://api.bitbucket.org/2.0/repositories/ws/repo?fields=uuid&page=2", "/repositories/ws/repo?fields=uuid&page=2", false},
		{"query preserved on bare path", "/repositories/ws/repo?q=1", "/repositories/ws/repo?q=1", false},
		{"trims whitespace", "  /2.0/user  ", "/user", false},
		{"empty is error", "", "", true},
		{"whitespace only is error", "   ", "", true},
		{"foreign host rejected", "https://evil.example.com/2.0/x", "", true},
		{"lookalike host rejected", "https://api.bitbucket.org.evil.com/2.0/x", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeAPIPath(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("NormalizeAPIPath(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if got != tc.want {
				t.Fatalf("NormalizeAPIPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
