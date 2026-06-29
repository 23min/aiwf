package cliutil

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRemoteTrackingRef(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		ref            string
		remote, branch string
		ok             bool
	}{
		{"default", "refs/remotes/origin/main", "origin", "main", true},
		{"multi-segment branch", "refs/remotes/origin/feature/x", "origin", "feature/x", true},
		{"local ref", "refs/heads/main", "", "", false},
		{"bare remote no branch", "refs/remotes/origin", "", "", false},
		{"trailing slash empty branch", "refs/remotes/origin/", "", "", false},
		{"non-ref garbage", "garbage", "", "", false},
		{"empty", "", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, b, ok := parseRemoteTrackingRef(c.ref)
			if r != c.remote || b != c.branch || ok != c.ok {
				t.Errorf("parseRemoteTrackingRef(%q) = (%q, %q, %v), want (%q, %q, %v)",
					c.ref, r, b, ok, c.remote, c.branch, c.ok)
			}
		})
	}
}

func TestFetchTrunkBestEffort_NonRemoteTrackingTrunk_Errors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A LOCAL trunk ref has no remote-tracking branch to fetch; the
	// function reports that without ever shelling out to git fetch.
	if err := os.WriteFile(filepath.Join(dir, "aiwf.yaml"),
		[]byte("allocate:\n  trunk: refs/heads/main\n"), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	err := FetchTrunkBestEffort(context.Background(), dir)
	if err == nil {
		t.Fatal("FetchTrunkBestEffort with a local trunk ref = nil, want error")
	}
	if !strings.Contains(err.Error(), "not a remote-tracking ref") {
		t.Errorf("error %q should explain the trunk ref is not a remote-tracking ref", err)
	}
}

func TestFetchTrunkBestEffort_MalformedConfig_Errors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A tab in the indentation is invalid YAML, so config.Load returns a
	// non-ErrNotFound error — surfaced as a loading failure (not silently
	// treated as "no config").
	if err := os.WriteFile(filepath.Join(dir, "aiwf.yaml"),
		[]byte("allocate:\n\ttrunk: refs/remotes/origin/main\n"), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	err := FetchTrunkBestEffort(context.Background(), dir)
	if err == nil {
		t.Fatal("FetchTrunkBestEffort with malformed aiwf.yaml = nil, want error")
	}
	if !strings.Contains(err.Error(), "loading aiwf.yaml") {
		t.Errorf("error %q should mention loading aiwf.yaml", err)
	}
}
