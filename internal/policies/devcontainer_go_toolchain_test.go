package policies

import (
	"path/filepath"
	"testing"
)

func TestPolicyDevcontainerGoToolchain(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyDevcontainerGoToolchain)
}

// TestPolicyDevcontainerGoToolchain_Fixtures drives each way the container's
// toolchain can disagree with CI's, plus the matching pair that must pass.
func TestPolicyDevcontainerGoToolchain_Fixtures(t *testing.T) {
	t.Parallel()
	const goYML = "env:\n  GO_VERSION: \"1.25.12\"\n"
	pinned := func(v string) string { return `{"containerEnv": {"GOTOOLCHAIN": "` + v + `"}}` }
	cases := []struct {
		name   string
		files  map[string]string
		firing bool
	}{
		{name: "matching-passes", firing: false, files: map[string]string{devcontainerConfigPath: pinned("go1.25.12"), goWorkflowPath: goYML}},
		{name: "no-devcontainer", firing: true, files: map[string]string{goWorkflowPath: goYML}},
		{name: "no-workflow", firing: true, files: map[string]string{devcontainerConfigPath: pinned("go1.25.12")}},
		{name: "no-exact-go-version", firing: true, files: map[string]string{devcontainerConfigPath: pinned("go1.25.12"), goWorkflowPath: "env:\n  GO_VERSION: \"1.25\"\n"}},
		{name: "no-gotoolchain", firing: true, files: map[string]string{devcontainerConfigPath: `{"containerEnv": {}}`, goWorkflowPath: goYML}},
		{name: "different-patch", firing: true, files: map[string]string{devcontainerConfigPath: pinned("go1.25.11"), goWorkflowPath: goYML}},
		{name: "auto-suffix", firing: true, files: map[string]string{devcontainerConfigPath: pinned("go1.25.12+auto"), goWorkflowPath: goYML}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for rel, content := range tc.files {
				mustWrite(t, filepath.Join(root, rel), content)
			}
			vs, err := PolicyDevcontainerGoToolchain(root)
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			if got := len(vs) > 0; got != tc.firing {
				t.Errorf("fired = %v, want %v; violations: %v", got, tc.firing, vs)
			}
		})
	}
}
