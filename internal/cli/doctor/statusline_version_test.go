package doctor

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/skills"
	"github.com/23min/aiwf/internal/version"
)

// TestStatuslineVersionLines covers the version-relationship + body-drift
// advisory branches with an injected binary version, since a `go test`
// binary always reports `(devel)` and could never reach the ahead/behind
// arms through version.Current(). G-0344.
func TestStatuslineVersionLines(t *testing.T) {
	t.Parallel()

	firstLine := func(lines []string, needle string) bool {
		for _, l := range lines {
			if strings.Contains(l, needle) {
				return true
			}
		}
		return false
	}

	t.Run("unmarked copy → no-marker advisory", func(t *testing.T) {
		t.Parallel()
		lines := statuslineVersionLines([]byte("#!/usr/bin/env bash\necho hi\n"), version.Parse("v1.0.0"))
		if len(lines) != 1 || !firstLine(lines, "no aiwf version marker") {
			t.Errorf("unmarked copy must emit exactly the no-marker advisory, got %v", lines)
		}
	})

	t.Run("binary ahead → refresh-available advisory", func(t *testing.T) {
		t.Parallel()
		lines := statuslineVersionLines(skills.RenderStatusline("v1.0.0"), version.Parse("v2.0.0"))
		if !firstLine(lines, "version:") || !firstLine(lines, "run `aiwf update` to refresh") {
			t.Errorf("binary-ahead must advise a refresh, got %v", lines)
		}
		if firstLine(lines, "drift:") {
			t.Errorf("a version difference alone must not report body drift, got %v", lines)
		}
	})

	t.Run("binary behind → not-downgraded advisory", func(t *testing.T) {
		t.Parallel()
		lines := statuslineVersionLines(skills.RenderStatusline("v2.0.0"), version.Parse("v1.0.0"))
		if !firstLine(lines, "newer than this aiwf binary") || !firstLine(lines, "not downgraded") {
			t.Errorf("binary-behind must report the installed copy is newer and not downgraded, got %v", lines)
		}
	})

	t.Run("binary behind + newer body → no downgrade-inducing drift hint", func(t *testing.T) {
		t.Parallel()
		// A newer installed version whose body also changed: the drift
		// hint's remediation would downgrade it, so it must be suppressed
		// — only the not-downgraded version line survives.
		newerEdited := append(skills.RenderStatusline("v2.0.0"), []byte("\n# body changed in the newer release\n")...)
		lines := statuslineVersionLines(newerEdited, version.Parse("v1.0.0"))
		if !firstLine(lines, "not downgraded") {
			t.Errorf("binary-behind must still report not-downgraded, got %v", lines)
		}
		if firstLine(lines, "drift:") {
			t.Errorf("binary-behind must NOT emit a drift hint (its remediation would downgrade), got %v", lines)
		}
	})

	t.Run("equal version, no body edit → silent", func(t *testing.T) {
		t.Parallel()
		lines := statuslineVersionLines(skills.RenderStatusline("v1.0.0"), version.Parse("v1.0.0"))
		if len(lines) != 0 {
			t.Errorf("a current, unedited copy must emit no advisory, got %v", lines)
		}
	})

	t.Run("equal version, body edited → drift only", func(t *testing.T) {
		t.Parallel()
		edited := append(skills.RenderStatusline("v1.0.0"), []byte("\n# local edit\n")...)
		lines := statuslineVersionLines(edited, version.Parse("v1.0.0"))
		if !firstLine(lines, "drift:") {
			t.Errorf("an equal-version body edit must report drift, got %v", lines)
		}
		if firstLine(lines, "version:") {
			t.Errorf("an equal-version copy must not report a version advisory, got %v", lines)
		}
	})
}
