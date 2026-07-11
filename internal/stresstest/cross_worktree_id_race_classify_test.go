package stresstest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// cross_worktree_id_race_classify_test.go pins
// classifyCrossWorktreeRace and findEntityFile — the pure/file-only
// decision logic behind CrossWorktreeIDRaceScenario (M-0241/AC-3) —
// deterministically, since a real collision only happens when the
// two sibling worktrees' `aiwf add` calls happen to race (an
// accepted, not-guaranteed outcome per the AC).

func TestClassifyCrossWorktreeRace(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		collided            bool
		checkFindings       []verbEnvelopeFinding
		attemptedReallocate bool
		reallocateStatus    string
		postCheckFindings   []verbEnvelopeFinding
		wantViolations      int
	}{
		{
			name:           "no collision this attempt — a benign no-op, zero violations",
			collided:       false,
			wantViolations: 0,
		},
		{
			name:                "collision caught by check, cleanly resolved by reallocate, post-check clean",
			collided:            true,
			checkFindings:       []verbEnvelopeFinding{{Code: check.CodeIDsUnique, Severity: "error"}},
			attemptedReallocate: true,
			reallocateStatus:    "ok",
			postCheckFindings:   []verbEnvelopeFinding{{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "warning"}},
			wantViolations:      0,
		},
		{
			name:                "collision occurred but check did not surface ids-unique — a violation",
			collided:            true,
			checkFindings:       []verbEnvelopeFinding{{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "warning"}},
			attemptedReallocate: true,
			reallocateStatus:    "ok",
			postCheckFindings:   nil,
			wantViolations:      1,
		},
		{
			name:                "collision occurred but no reallocate was attempted — a violation",
			collided:            true,
			checkFindings:       []verbEnvelopeFinding{{Code: check.CodeIDsUnique, Severity: "error"}},
			attemptedReallocate: false,
			wantViolations:      1,
		},
		{
			name:                "reallocate was attempted but did not report ok — a violation",
			collided:            true,
			checkFindings:       []verbEnvelopeFinding{{Code: check.CodeIDsUnique, Severity: "error"}},
			attemptedReallocate: true,
			reallocateStatus:    "error",
			postCheckFindings:   nil,
			wantViolations:      1,
		},
		{
			name:                "ids-unique finding still present after reallocate — a violation",
			collided:            true,
			checkFindings:       []verbEnvelopeFinding{{Code: check.CodeIDsUnique, Severity: "error"}},
			attemptedReallocate: true,
			reallocateStatus:    "ok",
			postCheckFindings:   []verbEnvelopeFinding{{Code: check.CodeIDsUnique, Severity: "error"}},
			wantViolations:      1,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			violations := classifyCrossWorktreeRace(tc.collided, tc.checkFindings, tc.attemptedReallocate, tc.reallocateStatus, tc.postCheckFindings)
			if len(violations) != tc.wantViolations {
				t.Errorf("violations = %d (%+v), want %d", len(violations), violations, tc.wantViolations)
			}
		})
	}
}

func TestFindEntityFile_FindsAMatchingFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "work", "gaps")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "G-0001-actorb.md"), []byte("body"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// A decoy with the same id but a different slug must not match.
	if err := os.WriteFile(filepath.Join(dir, "G-0001-actora.md"), []byte("body"), 0o644); err != nil {
		t.Fatalf("write decoy: %v", err)
	}

	got, err := findEntityFile(root, "G-0001", "actorb")
	if err != nil {
		t.Fatalf("findEntityFile: %v", err)
	}
	want := filepath.Join("work", "gaps", "G-0001-actorb.md")
	if got != want {
		t.Fatalf("findEntityFile = %q, want %q (relative to root, not absolute — aiwf reallocate resolves its path arg against the repo root)", got, want)
	}
}

func TestFindEntityFile_ErrorsWhenNoMatch(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	if _, err := findEntityFile(root, "G-0001", "actorb"); err == nil {
		t.Fatal("expected an error when no entity file matches")
	}
}

// TestFindEntityFile_RequiresExactNameNotSubstring pins that a
// filename merely CONTAINING the id (but not exactly "<id>-<slug>.md")
// is not a match. A single file, alphabetically after where the
// (absent) exact match would sort, so a substring-matching mutant
// would find it regardless of filepath.WalkDir's directory-entry
// enumeration order — TestFindEntityFile_FindsAMatchingFile's
// same-id/different-slug decoy alone doesn't pin this deterministically,
// since alphabetical walk order can coincidentally leave the correct
// file as the last (and therefore winning) match either way.
func TestFindEntityFile_RequiresExactNameNotSubstring(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "work", "gaps")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Contains "G-0001" as a substring and sorts after any exact
	// "G-0001-actorb.md" would, but is not that exact name.
	if err := os.WriteFile(filepath.Join(dir, "G-0001-zzz-not-the-match.md"), []byte("body"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := findEntityFile(root, "G-0001", "actorb"); err == nil {
		t.Fatal("expected findEntityFile to require an exact name match, not merely a containing id substring")
	}
}
