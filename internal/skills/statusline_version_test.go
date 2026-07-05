package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/version"
)

// TestRenderStatusline asserts the version sentinel is substituted at
// render time (mirroring the guidance fragment), that the rendered
// marker carries the given version, and that the raw embed still holds
// the unsubstituted sentinel (so the single source of truth is the
// embed, not a pre-rendered copy). G-0344.
func TestRenderStatusline(t *testing.T) {
	t.Parallel()
	if !bytes.Contains(statuslineEmbed, []byte(statuslineVersionSentinel)) {
		t.Fatalf("embed must carry the %q sentinel for RenderStatusline to substitute", statuslineVersionSentinel)
	}
	rendered := RenderStatusline("v1.2.3")
	if bytes.Contains(rendered, []byte(statuslineVersionSentinel)) {
		t.Errorf("rendered script must not still contain the sentinel %q", statuslineVersionSentinel)
	}
	got, ok := InstalledStatuslineVersion(rendered)
	if !ok {
		t.Fatal("rendered script must carry a parseable aiwf version marker")
	}
	if got != "v1.2.3" {
		t.Errorf("rendered marker version = %q, want %q", got, "v1.2.3")
	}
}

// TestInstalledStatuslineVersion covers the marker parser: a marked copy
// yields its token; content without the marker yields ok=false (a
// legacy/foreign copy the upgrade-only refresh must not touch). G-0344.
func TestInstalledStatuslineVersion(t *testing.T) {
	t.Parallel()
	t.Run("marked → version token", func(t *testing.T) {
		t.Parallel()
		got, ok := InstalledStatuslineVersion(RenderStatusline("v0.21.0"))
		if !ok || got != "v0.21.0" {
			t.Errorf("InstalledStatuslineVersion = (%q, %v), want (%q, true)", got, ok, "v0.21.0")
		}
	})
	t.Run("unmarked → ok=false", func(t *testing.T) {
		t.Parallel()
		got, ok := InstalledStatuslineVersion([]byte("#!/usr/bin/env bash\n# a hand-written statusline\necho hi\n"))
		if ok {
			t.Errorf("InstalledStatuslineVersion of an unmarked script must return ok=false, got version %q", got)
		}
	})
}

// TestStatuslineBodyDrifted asserts body-drift ignores the version-marker
// line: two copies that differ only in their stamped version are NOT
// drifted, while a genuine body edit IS. G-0344.
func TestStatuslineBodyDrifted(t *testing.T) {
	t.Parallel()
	t.Run("only version differs → not drifted", func(t *testing.T) {
		t.Parallel()
		if StatuslineBodyDrifted(RenderStatusline("v9.9.9")) {
			t.Error("a copy differing from the embed only in its stamped version must not be reported as body-drifted")
		}
	})
	t.Run("body edited → drifted", func(t *testing.T) {
		t.Parallel()
		edited := append(RenderStatusline("v1.0.0"), []byte("\n# a local edit\n")...)
		if !StatuslineBodyDrifted(edited) {
			t.Error("a copy with an edited body must be reported as drifted")
		}
	})
}

// TestStatuslineWriteNeedsConfirmation pins G-0367's version gate for the
// explicit `--statusline` write: a tagged (release) binary needs no
// confirmation — the deliberate-operator-forcing-a-refresh case the
// explicit path exists for — while an untagged binary (a dev/worktree
// build, or any pseudo/devel value version.Compare can't order) does, since
// that write lands unconditionally in the shared, cross-project user scope
// with no version marker distinguishing it as non-release.
func TestStatuslineWriteNeedsConfirmation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		binary version.Info
		want   bool
	}{
		{"tagged release binary needs no confirmation", version.Parse("v1.2.3"), false},
		{"devel binary needs confirmation", version.Parse(version.DevelVersion), true},
		{"dirty worktree stamp needs confirmation", version.Parse("main@abc1234-dirty"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := StatuslineWriteNeedsConfirmation(tc.binary); got != tc.want {
				t.Errorf("StatuslineWriteNeedsConfirmation(%+v) = %v, want %v", tc.binary, got, tc.want)
			}
		})
	}
}

// TestDecideStatuslineRefresh pins the pure upgrade-only decision matrix
// (G-0344): never downgrade, never act on an unorderable pair, heal an
// equal-version body edit, and stay silent when already current.
func TestDecideStatuslineRefresh(t *testing.T) {
	t.Parallel()
	v1 := version.Parse("v1.0.0")
	v2 := version.Parse("v2.0.0")
	devel := version.Parse(version.DevelVersion)

	cases := []struct {
		name        string
		binary      version.Info
		installed   version.Info
		marked      bool
		bodyDrifted bool
		want        StatuslineRefreshAction
	}{
		{"unmarked copy is left alone", v2, v1, false, false, RefreshActionSkipped},
		{"binary ahead → upgrade", v2, v1, true, false, RefreshActionUpgraded},
		{"binary behind → skip (no downgrade)", v1, v2, true, false, RefreshActionSkipped},
		{"unorderable binary (devel) → skip", devel, v1, true, false, RefreshActionSkipped},
		{"unorderable installed (devel) → skip", v1, devel, true, false, RefreshActionSkipped},
		{"equal + body drift → heal", v1, v1, true, true, RefreshActionHealed},
		{"equal + no drift → current", v1, v1, true, false, RefreshActionCurrent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, detail := decideStatuslineRefresh(tc.binary, tc.installed, tc.marked, tc.bodyDrifted)
			if got != tc.want {
				t.Errorf("decideStatuslineRefresh(%+v, %+v, marked=%v, drift=%v) = %q (%q), want %q",
					tc.binary, tc.installed, tc.marked, tc.bodyDrifted, got, detail, tc.want)
			}
		})
	}
}

// TestStatuslineRefreshOutcome_LedgerLine asserts the display decision:
// an already-current copy is silent (show=false), while any acted-on or
// skip-worth-knowing outcome renders a one-line entry naming its scope
// and detail. G-0344.
func TestStatuslineRefreshOutcome_LedgerLine(t *testing.T) {
	t.Parallel()
	t.Run("current → silent", func(t *testing.T) {
		t.Parallel()
		o := StatuslineRefreshOutcome{Scope: StatuslineScopeUser, Action: RefreshActionCurrent, Detail: "v1.0.0"}
		if line, show := o.LedgerLine(); show || line != "" {
			t.Errorf("an already-current outcome must be silent, got (%q, %v)", line, show)
		}
	})
	t.Run("upgraded → shown with scope and detail", func(t *testing.T) {
		t.Parallel()
		o := StatuslineRefreshOutcome{Scope: StatuslineScopeProject, Action: RefreshActionUpgraded, Detail: "v1.0.0 → v2.0.0"}
		line, show := o.LedgerLine()
		if !show {
			t.Fatal("an upgraded outcome must render a ledger line")
		}
		for _, want := range []string{"upgraded", "project", "v1.0.0 → v2.0.0"} {
			if !strings.Contains(line, want) {
				t.Errorf("ledger line %q must contain %q", line, want)
			}
		}
	})
}

// writeStatusline writes content to <dir>/.claude/statusline.sh,
// creating the parent. Returns the file path.
func writeStatusline(t *testing.T, dir string, content []byte) string {
	t.Helper()
	dest := filepath.Join(dir, ".claude", "statusline.sh")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, content, 0o755); err != nil {
		t.Fatal(err)
	}
	return dest
}

// TestAutoRefreshStatuslineForVersion drives the upgrade-only auto-refresh
// end-to-end over the filesystem for one scope at a time: an older copy is
// upgraded, a newer copy is never downgraded, an equal-version body edit is
// healed, an already-current copy is a silent no-op, an unmarked copy is
// left alone, and an absent copy is never created. G-0344.
func TestAutoRefreshStatuslineForVersion(t *testing.T) {
	t.Parallel()

	t.Run("older copy → upgraded to binary version", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		path := writeStatusline(t, home, RenderStatusline("v1.0.0"))
		outcomes, err := AutoRefreshStatuslineForVersion(t.TempDir(), home, version.Parse("v2.0.0"))
		if err != nil {
			t.Fatalf("AutoRefreshStatuslineForVersion: %v", err)
		}
		if len(outcomes) != 1 || outcomes[0].Action != RefreshActionUpgraded {
			t.Fatalf("want one Upgraded outcome, got %+v", outcomes)
		}
		got, _ := os.ReadFile(path)
		if !bytes.Equal(got, RenderStatusline("v2.0.0")) {
			t.Error("older copy must be rewritten to the binary's rendered version")
		}
	})

	t.Run("newer copy → skipped, never downgraded", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		newer := RenderStatusline("v2.0.0")
		path := writeStatusline(t, home, newer)
		outcomes, err := AutoRefreshStatuslineForVersion(t.TempDir(), home, version.Parse("v1.0.0"))
		if err != nil {
			t.Fatalf("AutoRefreshStatuslineForVersion: %v", err)
		}
		if len(outcomes) != 1 || outcomes[0].Action != RefreshActionSkipped {
			t.Fatalf("want one Skipped outcome, got %+v", outcomes)
		}
		got, _ := os.ReadFile(path)
		if !bytes.Equal(got, newer) {
			t.Error("a newer installed copy must not be downgraded")
		}
	})

	t.Run("equal version + body edit → healed", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		edited := append(RenderStatusline("v1.0.0"), []byte("\n# local edit\n")...)
		path := writeStatusline(t, home, edited)
		outcomes, err := AutoRefreshStatuslineForVersion(t.TempDir(), home, version.Parse("v1.0.0"))
		if err != nil {
			t.Fatalf("AutoRefreshStatuslineForVersion: %v", err)
		}
		if len(outcomes) != 1 || outcomes[0].Action != RefreshActionHealed {
			t.Fatalf("want one Healed outcome, got %+v", outcomes)
		}
		got, _ := os.ReadFile(path)
		if !bytes.Equal(got, RenderStatusline("v1.0.0")) {
			t.Error("an equal-version body edit must be healed to the embed")
		}
	})

	t.Run("already current → no write", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		current := RenderStatusline("v1.0.0")
		path := writeStatusline(t, home, current)
		info, _ := os.Stat(path)
		outcomes, err := AutoRefreshStatuslineForVersion(t.TempDir(), home, version.Parse("v1.0.0"))
		if err != nil {
			t.Fatalf("AutoRefreshStatuslineForVersion: %v", err)
		}
		if len(outcomes) != 1 || outcomes[0].Action != RefreshActionCurrent {
			t.Fatalf("want one Current outcome, got %+v", outcomes)
		}
		after, _ := os.Stat(path)
		if !info.ModTime().Equal(after.ModTime()) {
			t.Error("an already-current copy must not be rewritten (no mtime churn)")
		}
	})

	t.Run("unmarked copy → skipped, untouched", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		foreign := []byte("#!/usr/bin/env bash\n# a foreign statusline\necho hi\n")
		path := writeStatusline(t, home, foreign)
		outcomes, err := AutoRefreshStatuslineForVersion(t.TempDir(), home, version.Parse("v2.0.0"))
		if err != nil {
			t.Fatalf("AutoRefreshStatuslineForVersion: %v", err)
		}
		if len(outcomes) != 1 || outcomes[0].Action != RefreshActionSkipped {
			t.Fatalf("want one Skipped outcome, got %+v", outcomes)
		}
		got, _ := os.ReadFile(path)
		if !bytes.Equal(got, foreign) {
			t.Error("an unmarked copy must be left untouched")
		}
	})

	t.Run("absent copy → no outcome, no file created", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		home := t.TempDir()
		outcomes, err := AutoRefreshStatuslineForVersion(root, home, version.Parse("v2.0.0"))
		if err != nil {
			t.Fatalf("AutoRefreshStatuslineForVersion: %v", err)
		}
		if len(outcomes) != 0 {
			t.Fatalf("absent copies must yield no outcomes, got %+v", outcomes)
		}
		for _, dir := range []string{root, home} {
			if _, err := os.Stat(filepath.Join(dir, ".claude", "statusline.sh")); err == nil {
				t.Errorf("auto-refresh must never create a statusline (found one under %s)", dir)
			}
		}
	})

	t.Run("empty home → only project scope considered", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		projPath := writeStatusline(t, root, RenderStatusline("v1.0.0"))
		outcomes, err := AutoRefreshStatuslineForVersion(root, "", version.Parse("v2.0.0"))
		if err != nil {
			t.Fatalf("AutoRefreshStatuslineForVersion: %v", err)
		}
		if len(outcomes) != 1 || outcomes[0].Scope != StatuslineScopeProject {
			t.Fatalf("empty home must skip the user candidate and process only project, got %+v", outcomes)
		}
		if got, _ := os.ReadFile(projPath); !bytes.Equal(got, RenderStatusline("v2.0.0")) {
			t.Error("project copy must still upgrade when home is empty")
		}
	})

	t.Run("unreadable copy (dir at path) → error", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		// A directory at the destination makes os.ReadFile fail with a
		// non-not-exist error, exercising the read-fault return.
		if err := os.MkdirAll(filepath.Join(root, ".claude", "statusline.sh"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := AutoRefreshStatuslineForVersion(root, t.TempDir(), version.Parse("v2.0.0")); err == nil {
			t.Error("a directory at the statusline path must surface a read error, got nil")
		}
	})

	t.Run("both scopes processed independently", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		home := t.TempDir()
		userNewer := RenderStatusline("v2.0.0")
		projPath := writeStatusline(t, root, RenderStatusline("v1.0.0"))
		userPath := writeStatusline(t, home, userNewer)

		outcomes, err := AutoRefreshStatuslineForVersion(root, home, version.Parse("v1.5.0"))
		if err != nil {
			t.Fatalf("AutoRefreshStatuslineForVersion: %v", err)
		}
		if len(outcomes) != 2 {
			t.Fatalf("both installed scopes must yield an outcome each, got %+v", outcomes)
		}
		byScope := map[StatuslineScope]StatuslineRefreshAction{}
		for _, o := range outcomes {
			byScope[o.Scope] = o.Action
		}
		if byScope[StatuslineScopeUser] != RefreshActionSkipped {
			t.Errorf("newer user-scope copy must be skipped, got %q", byScope[StatuslineScopeUser])
		}
		if byScope[StatuslineScopeProject] != RefreshActionUpgraded {
			t.Errorf("older project-scope copy must be upgraded, got %q", byScope[StatuslineScopeProject])
		}
		if got, _ := os.ReadFile(userPath); !bytes.Equal(got, userNewer) {
			t.Error("user-scope newer copy must be preserved")
		}
		if got, _ := os.ReadFile(projPath); !bytes.Equal(got, RenderStatusline("v1.5.0")) {
			t.Error("project-scope older copy must be upgraded to the binary version")
		}
	})
}
