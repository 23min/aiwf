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

// TestConfiguredAreas pins the M-0175 accessor the grouping renderers
// read: it returns the declared members + default label, and (nil, "")
// when no aiwf.yaml is present (the graceful path that yields flat
// rendering).
func TestConfiguredAreas(t *testing.T) {
	t.Parallel()
	t.Run("returns members and default label", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
			[]byte("areas:\n  members:\n    - platform\n    - billing\n  default: Backlog\n"), 0o644); err != nil {
			t.Fatalf("write aiwf.yaml: %v", err)
		}
		members, def := cliutil.ConfiguredAreas(root)
		if len(members) != 2 || members[0] != "platform" || members[1] != "billing" {
			t.Errorf("members = %v, want [platform billing]", members)
		}
		if def != "Backlog" {
			t.Errorf("default = %q, want Backlog", def)
		}
	})
	t.Run("nil and empty when no aiwf.yaml", func(t *testing.T) {
		t.Parallel()
		members, def := cliutil.ConfiguredAreas(t.TempDir())
		if members != nil || def != "" {
			t.Errorf("ConfiguredAreas(empty) = (%v, %q), want (nil, \"\")", members, def)
		}
	})
}

// TestConfiguredAreaMembersFull pins the E-0044/M-0179 accessor the
// rename-area writer reads: it returns the full label+location member shape
// (so paths survive a rename), and nil when no aiwf.yaml is present.
func TestConfiguredAreaMembersFull(t *testing.T) {
	t.Parallel()
	t.Run("returns full members with paths", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
			[]byte("areas:\n  members:\n    - name: app-a\n      paths:\n        - projects/app-a/**\n    - billing\n"), 0o644); err != nil {
			t.Fatalf("write aiwf.yaml: %v", err)
		}
		members := cliutil.ConfiguredAreaMembersFull(root)
		if len(members) != 2 {
			t.Fatalf("members = %v, want 2", members)
		}
		if members[0].Name != "app-a" || len(members[0].Paths) != 1 || members[0].Paths[0] != "projects/app-a/**" {
			t.Errorf("members[0] = %+v, want app-a with [projects/app-a/**]", members[0])
		}
		if members[1].Name != "billing" || members[1].Paths != nil {
			t.Errorf("members[1] = %+v, want billing with nil paths", members[1])
		}
	})
	t.Run("nil when no aiwf.yaml", func(t *testing.T) {
		t.Parallel()
		if got := cliutil.ConfiguredAreaMembersFull(t.TempDir()); got != nil {
			t.Errorf("ConfiguredAreaMembersFull(empty) = %v, want nil", got)
		}
	})
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

	t.Run("reserved global sentinel is silent even with a block", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeAreaConfig(t, root, "platform", "billing")
		if note := cliutil.UndeclaredAreaNote(root, "global"); note != "" {
			t.Errorf("global note = %q, want empty (global is a recognized cross-cutting sentinel)", note)
		}
	})

	t.Run("reserved global sentinel gets the no-block note with no block at all", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir() // no aiwf.yaml
		note := cliutil.UndeclaredAreaNote(root, "global")
		// Position A — global is feature-gated: with no areas block, every
		// value (including global) is not a declared area, so the advisory
		// note fires and names the missing block.
		if !strings.Contains(note, "global") {
			t.Errorf("global note (no block) = %q, should name the requested value", note)
		}
		if !strings.Contains(note, "areas") {
			t.Errorf("global note (no block) = %q, should mention the missing areas block", note)
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
