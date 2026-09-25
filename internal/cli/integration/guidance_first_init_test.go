package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/testsupport"
)

// TestBinary_FirstInitAdoptsApplicablePacks runs init on a repository with no
// aiwf.yaml. The default catalogue URL is redirected to a local fixture through
// the isolated home's Git configuration, so the first run adopts exactly as a
// later one does without reaching the network.
func TestBinary_FirstInitAdoptsApplicablePacks(t *testing.T) {
	t.Parallel()
	root, source := t.TempDir(), testsupport.GuidanceSource(t, "*.xyz")
	if err := gitops.Init(t.Context(), root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.xyz"), []byte("project source"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Only the isolated home the fixture environment provides gets the
	// redirect; the process's own HOME is never written.
	isolated := testsupport.GuidanceEnvironment(t)
	home := ""
	for _, entry := range isolated {
		if value, ok := strings.CutPrefix(entry, "HOME="); ok {
			home = value
		}
	}
	if home == "" || home == os.Getenv("HOME") {
		t.Fatalf("fixture environment provides no isolated HOME: %q", home)
	}
	redirect := "[url \"" + source + "\"]\n\tinsteadOf = " + (&config.Config{}).GuidanceSource() + "\n"
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(redirect), 0o644); err != nil {
		t.Fatal(err)
	}
	// Git reads the isolated home's configuration even when the parent sets
	// GIT_CONFIG_GLOBAL.
	environment := append(os.Environ(), append(isolated, "GIT_CONFIG_GLOBAL="+filepath.Join(home, ".gitconfig"))...)
	cmd := exec.CommandContext(t.Context(), testutil.AiwfBinary(t), "init", "--root", root, "--skip-hook", "--actor", "human/test")
	cmd.Env = environment
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init: %v\n%s", err, output)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Guidance.Packs == nil || !slices.Equal(*cfg.Guidance.Packs, []string{"sample/base"}) {
		t.Fatalf("first init did not adopt the applicable pack: %v\n%s", cfg.Guidance.Packs, output)
	}
	if _, err := os.Stat(filepath.Join(root, ".guidance", "packs", "sample", "base", "guide.md")); err != nil {
		t.Fatalf("adopted pack not installed: %v\n%s", err, output)
	}
}
