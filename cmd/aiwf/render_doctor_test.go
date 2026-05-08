package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun_DoctorReportsRenderConfig: a freshly-init'd repo (default
// settings) shows the render config line with out_dir=site and
// commit_output=false. Pins the I3 step 7 deliverable that doctor
// surfaces the consumer's render setup.
func TestRun_DoctorReportsRenderConfig(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	captured := captureStdout(t, func() {
		// Ignore rc: --skip-hook means hooks are missing, so doctor
		// reports problems and exits non-zero. The test only
		// inspects render-config output, not the exit code.
		_ = run([]string{"doctor", "--root", root})
	})
	if !strings.Contains(string(captured), "render:") {
		t.Errorf("doctor missing render: line:\n%s", captured)
	}
	if !strings.Contains(string(captured), "out_dir=site") {
		t.Errorf("doctor render line missing default out_dir=site:\n%s", captured)
	}
	if !strings.Contains(string(captured), "commit_output=false") {
		t.Errorf("doctor render line missing commit_output=false:\n%s", captured)
	}
}

// TestRun_DoctorDetectsCommitOutputDrift: simulate the
// false→true flip without re-running aiwf update — the gitignore
// still carries `site/`. Doctor must flag it as drift and exit
// non-zero so CI catches the misconfiguration.
func TestRun_DoctorDetectsCommitOutputDrift(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	// Confirm site/ is in the gitignore from the default init.
	gi, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if !strings.Contains(string(gi), "site/") {
		t.Fatalf("expected site/ in .gitignore after init; got:\n%s", gi)
	}

	// Flip aiwf.yaml.html.commit_output to true WITHOUT running
	// aiwf update — the stale gitignore line remains.
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "html:\n  commit_output: true\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	captured := captureStdout(t, func() {
		// rc==1 expected (drift is a problem).
		_ = run([]string{"doctor", "--root", root})
	})
	out := string(captured)
	if !strings.Contains(out, "drift:") || !strings.Contains(out, "site/") {
		t.Errorf("doctor missing drift report for stale gitignore line:\n%s", out)
	}
}
