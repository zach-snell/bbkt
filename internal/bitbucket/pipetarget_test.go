package bitbucket

import (
	"encoding/json"
	"testing"
)

func TestPipeTargetPipelineName(t *testing.T) {
	cases := map[string]string{
		`{"selector":{"type":"custom","pattern":"build_bundle"}}`: "build_bundle",
		`{"selector":{"type":"branches","pattern":"develop"}}`:    "branches",
		`{"selector":{"type":"default"}}`:                         "default",
		`{}`:                                                      "",
	}
	for in, want := range cases {
		var tgt PipeTarget
		if err := json.Unmarshal([]byte(in), &tgt); err != nil {
			t.Fatal(err)
		}
		if got := tgt.PipelineName(); got != want {
			t.Errorf("%s: got %q, want %q", in, got, want)
		}
	}
	var nilTarget *PipeTarget
	if nilTarget.PipelineName() != "" {
		t.Error("nil target should have no name")
	}
}
