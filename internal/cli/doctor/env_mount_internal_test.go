package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestShadowMountStatus covers the three observable states (ok /
// empty / missing), the regular-file-not-directory edge case, the
// hidden-entries filter, and the 100+ count cap.
//
// Pins M-0135/AC-2.
func TestShadowMountStatus(t *testing.T) {
	t.Parallel()

	t.Run("ok with two plugin entries", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		mustMkdirAll(t, filepath.Join(home, ".claude", "plugins", "plugin-a"))
		mustMkdirAll(t, filepath.Join(home, ".claude", "plugins", "plugin-b"))
		state, count, err := shadowMountStatus(home)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if state != mountStateOK {
			t.Errorf("state = %v, want %v", state, mountStateOK)
		}
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		mustMkdirAll(t, filepath.Join(home, ".claude", "plugins"))
		state, count, err := shadowMountStatus(home)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if state != mountStateEmpty {
			t.Errorf("state = %v, want %v", state, mountStateEmpty)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
	})

	t.Run("missing target directory", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		state, _, err := shadowMountStatus(home)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if state != mountStateMissing {
			t.Errorf("state = %v, want %v", state, mountStateMissing)
		}
	})

	t.Run("target is a regular file not a directory", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		mustMkdirAll(t, filepath.Join(home, ".claude"))
		// plugins is a regular file, not a directory.
		if err := os.WriteFile(filepath.Join(home, ".claude", "plugins"), []byte(""), 0o644); err != nil {
			t.Fatalf("seed regular file: %v", err)
		}
		state, _, err := shadowMountStatus(home)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if state != mountStateMissing {
			t.Errorf("state = %v, want %v (regular file at target should be missing)", state, mountStateMissing)
		}
	})

	t.Run("hidden entries excluded from count", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		mustMkdirAll(t, filepath.Join(home, ".claude", "plugins", "plugin-a"))
		mustMkdirAll(t, filepath.Join(home, ".claude", "plugins", ".lock"))
		if err := os.WriteFile(filepath.Join(home, ".claude", "plugins", ".tmp"), []byte(""), 0o644); err != nil {
			t.Fatalf("seed hidden file: %v", err)
		}
		state, count, err := shadowMountStatus(home)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if state != mountStateOK {
			t.Errorf("state = %v, want %v", state, mountStateOK)
		}
		if count != 1 {
			t.Errorf("count = %d, want 1 (hidden entries `.lock` and `.tmp` must not count)", count)
		}
	})

	t.Run("count caps at 100", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		plugins := filepath.Join(home, ".claude", "plugins")
		for i := 0; i < 150; i++ {
			mustMkdirAll(t, filepath.Join(plugins, fmt.Sprintf("plugin-%03d", i)))
		}
		state, count, err := shadowMountStatus(home)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if state != mountStateOK {
			t.Errorf("state = %v, want %v", state, mountStateOK)
		}
		// Count is capped at 100 — value above 100 collapses to
		// 100 for the operator-facing render.
		if count != 100 {
			t.Errorf("count = %d, want 100 (cap)", count)
		}
	})
}

// TestRenderMountLine pins the formatted render shape for each state
// + the 100+ display marker for the capped count.
func TestRenderMountLine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		state  mountState
		count  int
		errMsg string
		want   string
	}{
		{name: "ok 1", state: mountStateOK, count: 1, want: "plugin-mount: ok (1 plugin entries cached)"},
		{name: "ok 7", state: mountStateOK, count: 7, want: "plugin-mount: ok (7 plugin entries cached)"},
		{name: "ok capped", state: mountStateOK, count: 100, want: "plugin-mount: ok (100+ plugin entries cached)"},
		{name: "empty", state: mountStateEmpty, count: 0, want: "plugin-mount: empty (mount target exists but no plugin entries — first rebuild before initialize.sh, or shadow-mount not yet seeded)"},
		{name: "missing", state: mountStateMissing, count: 0, want: "plugin-mount: missing (mount target does not exist — devcontainer.json mount entry stripped or container rebuild failed mid-postcreate)"},
		{name: "errmsg", state: mountStateError, errMsg: "boom", want: "plugin-mount: boom"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := renderMountLine(tc.state, tc.count, tc.errMsg)
			if got != tc.want {
				t.Errorf("renderMountLine = %q\n want %q", got, tc.want)
			}
		})
	}
}

func mustMkdirAll(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", p, err)
	}
}
