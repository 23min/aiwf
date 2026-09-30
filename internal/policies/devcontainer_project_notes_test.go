package policies

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPolicyDevcontainerProjectNotes(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyDevcontainerProjectNotes)
}

// TestPolicyDevcontainerProjectNotes_Fixtures drives each way the notes can
// disagree with the scripts they describe, plus notes that agree.
func TestPolicyDevcontainerProjectNotes_Fixtures(t *testing.T) {
	t.Parallel()
	const (
		hook = "if [[ \"${AIWF_DEVCONTAINER_E2E:-false}\" == \"true\" ]]; then :; fi\n"
		kit  = "printf '!! Fix the cause, then run: bash a.sh && bash b.sh\\n'\n"
		good = "# Notes\n\n## Playwright tests\n\nSet `AIWF_DEVCONTAINER_E2E=true`, then **Rebuild Container**.\n\n## Recovery\n\n```sh\nbash a.sh && bash b.sh\n```\n"
	)
	files := func(notes, hookBody, kitBody string) map[string]string {
		m := map[string]string{}
		if notes != "" {
			m[projectNotesPath] = notes
		}
		if hookBody != "" {
			m[projectPostCreatePath] = hookBody
		}
		if kitBody != "" {
			m[kitPostCreatePath] = kitBody
		}
		return m
	}
	cases := []struct {
		name   string
		files  map[string]string
		firing bool
	}{
		{name: "agreeing-passes", firing: false, files: files(good, hook, kit)},
		{name: "no-notes", firing: true, files: files("", hook, kit)},
		{name: "hook-without-opt-in", firing: true, files: files(good, "npx playwright install\n", kit)},
		{name: "notes-name-another-variable", firing: true, files: files(strings.Replace(good, "AIWF_DEVCONTAINER_E2E=true", "E2E=true", 1), hook, kit)},
		{name: "notes-omit-rebuild", firing: true, files: files(strings.Replace(good, "Rebuild Container", "restart", 1), hook, kit)},
		{name: "opt-in-outside-its-section", firing: true, files: files(strings.Replace(good, "## Playwright tests", "## Other", 1), hook, kit)},
		{name: "kit-without-recovery", firing: true, files: files(good, hook, "echo done\n")},
		{name: "recovery-command-differs", firing: true, files: files(good, hook, strings.Replace(kit, "b.sh", "c.sh", 1))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for rel, content := range tc.files {
				mustWrite(t, filepath.Join(root, rel), content)
			}
			vs, err := PolicyDevcontainerProjectNotes(root)
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			if got := len(vs) > 0; got != tc.firing {
				t.Errorf("fired = %v, want %v; violations: %v", got, tc.firing, vs)
			}
		})
	}
}
