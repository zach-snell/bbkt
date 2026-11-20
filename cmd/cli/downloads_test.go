package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFetch(s string, fail bool) func(*os.File) (int64, error) {
	return func(f *os.File) (int64, error) {
		n, _ := f.WriteString(s)
		if fail {
			return int64(n), errors.New("network down")
		}
		return int64(n), nil
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSaveDownload(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "a.zip")

	if _, err := saveDownload(out, false, writeFetch("v1", false)); err != nil || readFile(t, out) != "v1" {
		t.Fatalf("new file: err=%v", err)
	}
	// Existing file without --force: refused, untouched, nothing downloaded.
	called := false
	_, err := saveDownload(out, false, func(*os.File) (int64, error) { called = true; return 0, nil })
	if err == nil || !strings.Contains(err.Error(), "already exists") || called || readFile(t, out) != "v1" {
		t.Fatalf("no-force overwrite: err=%v called=%v", err, called)
	}
	// --force failing mid-transfer keeps the old file and leaves no temp file.
	if _, err := saveDownload(out, true, writeFetch("partial", true)); err == nil || readFile(t, out) != "v1" {
		t.Fatalf("failed force: err=%v content=%q", err, readFile(t, out))
	}
	if _, err := saveDownload(out, true, writeFetch("v2", false)); err != nil || readFile(t, out) != "v2" {
		t.Fatalf("force: err=%v", err)
	}
	// Failed fresh download leaves nothing behind.
	fresh := filepath.Join(dir, "b.zip")
	if _, err := saveDownload(fresh, false, writeFetch("partial", true)); err == nil {
		t.Fatal("expected error")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Fatalf("leftover files: %v", entries)
	}
}

func TestFormatBytes(t *testing.T) {
	cases := map[int64]string{
		0:         "0 B",
		1023:      "1023 B",
		1024:      "1.0 KiB",
		15178474:  "14.5 MiB",
		154416927: "147.3 MiB",
		1 << 30:   "1.0 GiB",
	}
	for in, want := range cases {
		if got := formatBytes(in); got != want {
			t.Errorf("formatBytes(%d) = %q, want %q", in, got, want)
		}
	}
}
