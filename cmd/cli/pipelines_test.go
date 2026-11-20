package cli

import (
	"strings"
	"testing"

	"github.com/zach-snell/bbkt/internal/bitbucket"
)

func TestParsePipelineVars(t *testing.T) {
	got, err := parsePipelineVars([]string{"FLAVOUR=release", " URL =https://x?a=b", "EMPTY="}, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []bitbucket.PipelineVariable{
		{Key: "FLAVOUR", Value: "release", Secured: true},
		{Key: "URL", Value: "https://x?a=b", Secured: true},
		{Key: "EMPTY", Value: "", Secured: true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	for _, bad := range []string{"NOEQUALS", "=value", " =x"} {
		if _, err := parsePipelineVars([]string{bad}, false); err == nil {
			t.Errorf("parsePipelineVars(%q) should fail", bad)
		}
	}

	_, err = parsePipelineVars([]string{"OK=1", "=s3cr3t-token"}, true)
	if err == nil || strings.Contains(err.Error(), "s3cr3t") {
		t.Fatalf("secured error must not echo the value: %v", err)
	}
	if !strings.Contains(err.Error(), "#2") {
		t.Errorf("error should name the argument position: %v", err)
	}
}
