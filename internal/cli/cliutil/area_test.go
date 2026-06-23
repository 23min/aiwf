package cliutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// writeAreaConfig drops a minimal aiwf.yaml carrying an areas.members
// block into root. Used to exercise UndeclaredAreaNote's declared-set
// branches without standing up a full repo.
func writeAreaConfig(t *testing.T, root string, members ...string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("areas:\n  members:\n")
	for _, m := range members {
		b.WriteString("    - " + m + "\n")
	}
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
}

// TestUndeclaredAreaNote pins M-0174/AC-5's advisory-note logic, the
// single source the three read verbs share: an empty area is silent; a
// declared area is silent; an undeclared area names the value and the
// declared set; an undeclared area with no areas block at all names the
// value and points at the missing block. The note is purely advisory —
// it never affects the (mechanical) filter, only tells the operator the
// value they typed is not one they declared.
func TestUndeclaredAreaNote(t *testing.T) {
	t.Parallel()

	t.Run("empty area is silent", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeAreaConfig(t, root, "platform", "billing")
		if note := cliutil.UndeclaredAreaNote(root, ""); note != "" {
			t.Errorf("empty area note = %q, want empty", note)
		}
	})

	t.Run("declared area is silent", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeAreaConfig(t, root, "platform", "billing")
		if note := cliutil.UndeclaredAreaNote(root, "platform"); note != "" {
			t.Errorf("declared area note = %q, want empty", note)
		}
	})

	t.Run("undeclared area names value and declared set", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeAreaConfig(t, root, "platform", "billing")
		note := cliutil.UndeclaredAreaNote(root, "nonsense")
		for _, want := range []string{"nonsense", "platform", "billing"} {
			if !strings.Contains(note, want) {
				t.Errorf("note %q missing %q", note, want)
			}
		}
	})

	t.Run("undeclared area with no areas block names the missing block", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir() // no aiwf.yaml at all
		note := cliutil.UndeclaredAreaNote(root, "platform")
		if !strings.Contains(note, "platform") {
			t.Errorf("note %q should name the requested value", note)
		}
		if !strings.Contains(note, "areas") {
			t.Errorf("note %q should mention the missing areas block", note)
		}
	})
}
