package policies

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPolicyDevcontainerPathsResolve(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyDevcontainerPathsResolve)
}

// TestPolicyDevcontainerPathsResolve_Fixtures drives the rules the policy
// holds: a named path must exist or, as a glob, match; trailing punctuation
// is not part of a path; archived ADRs, tests and testdata are out of scope.
func TestPolicyDevcontainerPathsResolve_Fixtures(t *testing.T) {
	t.Parallel()
	present := map[string]string{
		".devcontainer/devcontainer.json":      "{}",
		".devcontainer/project/post-create.sh": "#!/usr/bin/env bash\n",
	}
	with := func(extra map[string]string) map[string]string {
		m := map[string]string{}
		for k, v := range present {
			m[k] = v
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	cases := []struct {
		name   string
		files  map[string]string
		firing bool
	}{
		{name: "existing-path-and-glob-pass", firing: false, files: with(map[string]string{"CLAUDE.md": "See .devcontainer/devcontainer.json and .devcontainer/project/*.sh.\n"})},
		{name: "removed-path-in-guide", firing: true, files: with(map[string]string{"CLAUDE.md": "Mechanics live in .devcontainer/init.sh\n"})},
		{name: "glob-matching-nothing", firing: true, files: with(map[string]string{"README.md": "Hooks: .devcontainer/hooks/*.sh\n"})},
		{name: "removed-path-in-adr", firing: true, files: with(map[string]string{"docs/adr/ADR-0001-x.md": "Uses .devcontainer/init.sh\n"})},
		{name: "archived-adr-ignored", firing: false, files: with(map[string]string{"docs/adr/archive/ADR-0001-x.md": "Uses .devcontainer/init.sh\n"})},
		{name: "removed-path-in-workflow", firing: true, files: with(map[string]string{".github/workflows/x.yml": "# must match .devcontainer/init.sh\n"})},
		{name: "removed-path-in-go-source", firing: true, files: with(map[string]string{"internal/x/x.go": "// seeded by .devcontainer/init.sh\n"})},
		{name: "go-tests-and-testdata-ignored", firing: false, files: with(map[string]string{
			"internal/x/x_test.go":        "// fixture .devcontainer/init.sh\n",
			"internal/x/testdata/fixture": ".devcontainer/init.sh\n",
		})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for rel, content := range tc.files {
				mustWrite(t, filepath.Join(root, rel), content)
			}
			vs, err := PolicyDevcontainerPathsResolve(root)
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			if got := len(vs) > 0; got != tc.firing {
				t.Errorf("fired = %v, want %v; violations: %v", got, tc.firing, vs)
			}
		})
	}
}

// TestPolicyDevcontainerPathsResolve_UnreadableDirectory pins that a walk
// failure other than a missing directory is returned, not skipped: a skipped
// directory would let its references go unchecked.
func TestPolicyDevcontainerPathsResolve_UnreadableDirectory(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits do not deny the walk")
	}
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "scripts", "locked", "x.sh"), "echo\n")
	denied := filepath.Join(root, "scripts", "locked")
	if err := os.Chmod(denied, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(denied, 0o755) })
	if _, err := PolicyDevcontainerPathsResolve(root); err == nil {
		t.Fatal("want an error when a scanned directory cannot be read, got nil")
	}
}
