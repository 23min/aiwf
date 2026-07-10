package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRunList_EnumeratesEveryRegisteredScenario pins M-0249/AC-3:
// an operator can discover what's runnable without reading Go
// source — every one of the 12 catalog names appears, in catalog
// order, one per line.
func TestRunList_EnumeratesEveryRegisteredScenario(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer

	if err := runList(&out); err != nil {
		t.Fatalf("runList: %v", err)
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	wantNames := scenarioNames()
	if len(lines) != len(wantNames) {
		t.Fatalf("runList printed %d lines, want %d:\n%s", len(lines), len(wantNames), out.String())
	}
	for i, name := range wantNames {
		if lines[i] != name {
			t.Errorf("line %d = %q, want %q", i, lines[i], name)
		}
	}
}

// TestRun_ListCommand_Succeeds drives the "list" subcommand through
// the same entry point a real invocation uses.
func TestRun_ListCommand_Succeeds(t *testing.T) {
	t.Parallel()
	if code := run([]string{"list"}); code != 0 {
		t.Fatalf("run([list]) = %d, want 0", code)
	}
}
