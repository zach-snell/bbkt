package bitbucket

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// IsExpired uses a 5-min buffer: a token is treated as expired if it expires
// within 5 minutes, so callers can refresh before the token dies mid-request.
func TestIsExpired_FiveMinuteBuffer(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name      string
		authType  AuthType
		createdAt time.Time
		expiresIn int
		want      bool
	}{
		{
			name:      "oauth with 4 min remaining is expired (inside buffer)",
			authType:  AuthTypeOAuth,
			createdAt: now.Add(-1 * time.Hour),
			expiresIn: int((time.Hour + 4*time.Minute) / time.Second),
			want:      true,
		},
		{
			name:      "oauth with 6 min remaining is not expired (outside buffer)",
			authType:  AuthTypeOAuth,
			createdAt: now.Add(-1 * time.Hour),
			expiresIn: int((time.Hour + 6*time.Minute) / time.Second),
			want:      false,
		},
		{
			name:      "oauth already past absolute expiry is expired",
			authType:  AuthTypeOAuth,
			createdAt: now.Add(-2 * time.Hour),
			expiresIn: int(time.Hour / time.Second),
			want:      true,
		},
		{
			name:      "api token never expires",
			authType:  AuthTypeAPIToken,
			createdAt: now.Add(-365 * 24 * time.Hour),
			expiresIn: 0,
			want:      false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Credentials{
				AuthType:  tc.authType,
				CreatedAt: tc.createdAt,
				ExpiresIn: tc.expiresIn,
			}
			if got := c.IsExpired(); got != tc.want {
				t.Errorf("IsExpired() = %v, want %v", got, tc.want)
			}
		})
	}
}

// Older versions of bbkt wrote credentials.json as a single Credentials
// object at the root; newer versions use ProfileStore. Migration must be
// transparent or upgrading users lose access to their saved creds.
func TestLoadProfileStore_MigratesLegacyFormat(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	path := filepath.Join(home, ".config", "bbkt", "credentials.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := `{
  "auth_type": "api_token",
  "email": "old@example.com",
  "api_token": "legacy-token",
  "created_at": "2024-01-01T00:00:00Z"
}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}

	store, err := LoadProfileStore()
	if err != nil {
		t.Fatalf("LoadProfileStore: %v", err)
	}
	if store.ActiveProfile != "default" {
		t.Errorf("ActiveProfile = %q, want 'default'", store.ActiveProfile)
	}
	def, ok := store.Profiles["default"]
	if !ok {
		t.Fatal("expected 'default' profile after migration")
	}
	if def.Email != "old@example.com" {
		t.Errorf("migrated email = %q, want old@example.com", def.Email)
	}
	if def.APIToken != "legacy-token" {
		t.Errorf("migrated api token = %q, want legacy-token", def.APIToken)
	}
	if def.ProfileName != "default" {
		t.Errorf("ProfileName on migrated creds = %q, want 'default'", def.ProfileName)
	}

	// Migration should rewrite the file in the new format so we don't re-migrate forever.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var shape map[string]any
	if err := json.Unmarshal(data, &shape); err != nil {
		t.Fatalf("unmarshaling migrated file: %v", err)
	}
	if _, ok := shape["profiles"]; !ok {
		t.Errorf("on-disk file should contain 'profiles' key after migration, got: %s", data)
	}
}

func TestLoadProfileStore_NewFormatRoundTrips(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	orig := &ProfileStore{
		ActiveProfile: "work",
		Profiles: map[string]*Credentials{
			"work": {
				AuthType:    AuthTypeOAuth,
				AccessToken: "a1", RefreshToken: "r1",
				ClientID: "cid", ClientSecret: "csec",
				ExpiresIn: 7200, CreatedAt: time.Now().UTC().Truncate(time.Second),
			},
			"personal": {
				AuthType: AuthTypeAPIToken,
				Email:    "me@example.com", APIToken: "t",
				CreatedAt: time.Now().UTC().Truncate(time.Second),
			},
		},
	}
	if err := SaveProfileStore(orig); err != nil {
		t.Fatal(err)
	}

	got, err := LoadProfileStore()
	if err != nil {
		t.Fatal(err)
	}
	if got.ActiveProfile != "work" {
		t.Errorf("ActiveProfile = %q, want 'work'", got.ActiveProfile)
	}
	if len(got.Profiles) != 2 {
		t.Errorf("profile count = %d, want 2", len(got.Profiles))
	}
	work := got.Profiles["work"]
	if work == nil || work.ProfileName != "work" || work.AccessToken != "a1" {
		t.Errorf("work profile not round-tripped: %+v", work)
	}
	personal := got.Profiles["personal"]
	if personal == nil || personal.ProfileName != "personal" || personal.Email != "me@example.com" {
		t.Errorf("personal profile not round-tripped: %+v", personal)
	}
}
