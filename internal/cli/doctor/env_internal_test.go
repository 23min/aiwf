package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDetectContainer covers the four signal combinations for the
// container-detection probe. Internal-package test so it can reach
// the unexported helper.
//
// Pins M-0135/AC-1.
func TestDetectContainer(t *testing.T) {
	t.Parallel()
	// A real existing file the helper can stat() when the test wants
	// the dockerenv signal to fire. The test uses a t.TempDir-rooted
	// file as the dockerenv-path stand-in.
	tmpDir := t.TempDir()
	dockerenv := filepath.Join(tmpDir, ".dockerenv")
	if err := os.WriteFile(dockerenv, []byte(""), 0o644); err != nil {
		t.Fatalf("seed dockerenv: %v", err)
	}
	missing := filepath.Join(tmpDir, "nope")

	cases := []struct {
		name          string
		dockerenvPath string
		devEnv        string
		wantIn        bool
		wantLabel     string
	}{
		{
			name:          "both signals",
			dockerenvPath: dockerenv,
			devEnv:        "1",
			wantIn:        true,
			wantLabel:     "devcontainer (/.dockerenv + AIWF_DEVCONTAINER)",
		},
		{
			name:          "only dockerenv",
			dockerenvPath: dockerenv,
			devEnv:        "",
			wantIn:        true,
			wantLabel:     "devcontainer (/.dockerenv)",
		},
		{
			name:          "only env var truthy lowercase",
			dockerenvPath: missing,
			devEnv:        "true",
			wantIn:        true,
			wantLabel:     "devcontainer (AIWF_DEVCONTAINER)",
		},
		{
			name:          "only env var truthy mixed case",
			dockerenvPath: missing,
			devEnv:        "TRUE",
			wantIn:        true,
			wantLabel:     "devcontainer (AIWF_DEVCONTAINER)",
		},
		{
			name:          "neither",
			dockerenvPath: missing,
			devEnv:        "",
			wantIn:        false,
			wantLabel:     "host",
		},
		{
			name:          "env var falsy zero",
			dockerenvPath: missing,
			devEnv:        "0",
			wantIn:        false,
			wantLabel:     "host",
		},
		{
			name:          "env var falsy false",
			dockerenvPath: missing,
			devEnv:        "false",
			wantIn:        false,
			wantLabel:     "host",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotIn, gotLabel := detectContainer(tc.dockerenvPath, tc.devEnv)
			if gotIn != tc.wantIn {
				t.Errorf("inContainer = %v, want %v", gotIn, tc.wantIn)
			}
			if gotLabel != tc.wantLabel {
				t.Errorf("label = %q, want %q", gotLabel, tc.wantLabel)
			}
		})
	}
}

// TestDetectContainer_DockerenvSymlinkCounts: a symlink to a real file
// at the dockerenv path is still treated as present (stat, not lstat).
// AC-1 edge case.
func TestDetectContainer_DockerenvSymlinkCounts(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	if err := os.WriteFile(target, []byte(""), 0o644); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	link := filepath.Join(tmpDir, "linked.dockerenv")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unsupported on this filesystem: %v", err)
	}
	gotIn, gotLabel := detectContainer(link, "")
	if !gotIn {
		t.Errorf("inContainer = false, want true (stat should follow symlink)")
	}
	if gotLabel != "devcontainer (/.dockerenv)" {
		t.Errorf("label = %q, want %q", gotLabel, "devcontainer (/.dockerenv)")
	}
}
