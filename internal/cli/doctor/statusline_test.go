package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

// TestStatuslineReport_AC1_NotEmittedWithoutInstall asserts M-0157/AC-1:
// when no statusline script is installed (neither project nor user scope),
// the block is not emitted at all.
func TestStatuslineReport_AC1_NotEmittedWithoutInstall(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()
	lines := appendStatuslineReportWithHome(nil, root, home, false)
	for _, line := range lines {
		if strings.Contains(line, "statusline:") {
			t.Errorf("AC-1: statusline block must not be emitted when the script is not installed; got line: %s", line)
		}
	}
	if len(lines) != 0 {
		t.Errorf("AC-1: expected 0 lines when statusline is not installed, got %d", len(lines))
	}
}

// TestStatuslineReport_AC1_EmittedWhenInstalled asserts M-0157/AC-1:
// when the script is installed, the statusline: line is emitted.
func TestStatuslineReport_AC1_EmittedWhenInstalled(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()
	installStatusline(t, root)

	lines := appendStatuslineReportWithHome(nil, root, home, false)
	found := false
	for _, line := range lines {
		if strings.Contains(line, "statusline:") && strings.Contains(line, "installed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("AC-1: statusline: installed line must be present when the script exists; lines: %v", lines)
	}
}

// TestStatuslineReport_AC2_InstallHintsPlatformBranched asserts
// M-0157/AC-2: install hints branch on GOOS.
func TestStatuslineReport_AC2_InstallHintsPlatformBranched(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tool, goos, wantContains string
	}{
		{"jq", "darwin", "brew install jq"},
		{"jq", "linux", "apt-get install jq"},
		{"gh", "darwin", "brew install gh"},
		{"gh", "linux", "apt-get install gh"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.tool+"_"+tc.goos, func(t *testing.T) {
			t.Parallel()
			got := installHintFor(tc.tool, tc.goos)
			if !strings.Contains(got, tc.wantContains) {
				t.Errorf("AC-2: installHintFor(%q, %q) = %q, want to contain %q", tc.tool, tc.goos, got, tc.wantContains)
			}
		})
	}
}

// TestStatuslineReport_AC3_NotWiredPrintsSnippet asserts M-0157/AC-3:
// when the script is installed but no settings file contains a
// statusLine key, the wiring hint is emitted.
func TestStatuslineReport_AC3_NotWiredPrintsSnippet(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()
	installStatusline(t, root)

	lines := appendStatuslineReportWithHome(nil, root, home, false)
	foundWiring := false
	for _, line := range lines {
		if strings.Contains(line, "wiring:") && strings.Contains(line, "not found") {
			foundWiring = true
			break
		}
	}
	if !foundWiring {
		t.Errorf("AC-3: wiring hint must be emitted when statusLine key is absent from settings; lines: %v", lines)
	}
}

// TestStatuslineReport_AC3_WiredSuppressesSnippet asserts M-0157/AC-3:
// when the statusLine key is present in settings, no wiring hint.
func TestStatuslineReport_AC3_WiredSuppressesSnippet(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()
	installStatusline(t, root)
	wireSettings(t, root)

	lines := appendStatuslineReportWithHome(nil, root, home, false)
	for _, line := range lines {
		if strings.Contains(line, "wiring:") {
			t.Errorf("AC-3: wiring hint must not be emitted when statusLine is already wired; got line: %s", line)
		}
	}
}

// TestStatuslineReport_AC4_DriftDetected asserts M-0157/AC-4:
// when the on-disk script differs from the embedded copy, drift is reported.
func TestStatuslineReport_AC4_DriftDetected(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()
	installStatusline(t, root)
	path := filepath.Join(root, ".claude", "statusline.sh")
	if err := os.WriteFile(path, []byte("#!/bin/bash\n# modified\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	lines := appendStatuslineReportWithHome(nil, root, home, false)
	foundDrift := false
	for _, line := range lines {
		if strings.Contains(line, "drift:") {
			foundDrift = true
			break
		}
	}
	if !foundDrift {
		t.Errorf("AC-4: drift line must be emitted when on-disk differs from embedded; lines: %v", lines)
	}
}

// TestStatuslineReport_AC4_NoDriftWhenMatching asserts M-0157/AC-4:
// when the on-disk script matches the embedded copy, no drift line.
func TestStatuslineReport_AC4_NoDriftWhenMatching(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()
	installStatusline(t, root)

	lines := appendStatuslineReportWithHome(nil, root, home, false)
	for _, line := range lines {
		if strings.Contains(line, "drift:") {
			t.Errorf("AC-4: drift line must not be emitted when on-disk matches embedded; got line: %s", line)
		}
	}
}

// TestStatuslineReport_AC5_ContainerNudge asserts M-0157/AC-5:
// when inContainer=true and project scope, the nudge is emitted.
func TestStatuslineReport_AC5_ContainerNudge(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()
	installStatusline(t, root)

	lines := appendStatuslineReportWithHome(nil, root, home, true)
	foundNudge := false
	for _, line := range lines {
		if strings.Contains(line, "nudge:") && strings.Contains(line, "--scope user") {
			foundNudge = true
			break
		}
	}
	if !foundNudge {
		t.Errorf("AC-5: container nudge must be emitted when inContainer=true and project scope; lines: %v", lines)
	}
}

// TestStatuslineReport_AC5_NoNudgeOutsideContainer asserts M-0157/AC-5:
// when not in a container, no nudge is emitted.
func TestStatuslineReport_AC5_NoNudgeOutsideContainer(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()
	installStatusline(t, root)

	lines := appendStatuslineReportWithHome(nil, root, home, false)
	for _, line := range lines {
		if strings.Contains(line, "nudge:") {
			t.Errorf("AC-5: nudge must not be emitted outside container; got line: %s", line)
		}
	}
}

// TestStatuslineReport_AC5_ResolveInstalledStatusline verifies scope
// resolution covers project, user, and neither-installed.
func TestStatuslineReport_AC5_ResolveInstalledStatusline(t *testing.T) {
	t.Parallel()
	t.Run("project scope found", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		installStatusline(t, root)
		path, scope := resolveInstalledStatusline(
			filepath.Join(root, ".claude", "statusline.sh"),
			filepath.Join(t.TempDir(), ".claude", "statusline.sh"),
		)
		if scope != "project" {
			t.Errorf("expected scope 'project', got %q", scope)
		}
		if path == "" {
			t.Error("expected non-empty path")
		}
	})
	t.Run("user scope fallback", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		userPath := filepath.Join(home, ".claude", "statusline.sh")
		if err := os.MkdirAll(filepath.Dir(userPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(userPath, skills.StatuslineBytes(), 0o755); err != nil {
			t.Fatal(err)
		}
		path, scope := resolveInstalledStatusline(
			filepath.Join(t.TempDir(), ".claude", "statusline.sh"),
			userPath,
		)
		if scope != "user" {
			t.Errorf("expected scope 'user', got %q", scope)
		}
		if path != userPath {
			t.Errorf("expected path %q, got %q", userPath, path)
		}
	})
	t.Run("neither installed", func(t *testing.T) {
		t.Parallel()
		path, scope := resolveInstalledStatusline(
			filepath.Join(t.TempDir(), ".claude", "statusline.sh"),
			filepath.Join(t.TempDir(), ".claude", "statusline.sh"),
		)
		if path != "" || scope != "" {
			t.Errorf("expected empty path and scope, got %q %q", path, scope)
		}
	})
}

// --- test helpers ---

func installStatusline(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "statusline.sh"), skills.StatuslineBytes(), 0o755); err != nil {
		t.Fatal(err)
	}
}

func wireSettings(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"statusLine":{"type":"command","command":".claude/statusline.sh"}}` + "\n")
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), content, 0o644); err != nil {
		t.Fatal(err)
	}
}
