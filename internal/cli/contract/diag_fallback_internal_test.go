package contract

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/23min/aiwf/internal/cli/add"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/initcmd"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/skills"
)

// diag_fallback_internal_test.go — white-box (package contract, not
// contract_test) so these tests can call the unexported runBind /
// runUnbind / runRecipeInstall / runRecipeRemove directly. cli.Execute
// always mints a real correlation id (NewRootCmd), so the `runID == ""`
// fallback inside each of these four functions is unreachable through
// the CLI surface — only a direct call bypassing NewCmd/Execute (an
// OutputFormat built by hand, zero value CorrelationID) can exercise
// it. Mirrors the established pattern in
// internal/cli/integration/correlation_id_test.go for the other
// wired verbs.

// fallbackTestRepo git-inits root and runs `aiwf init --skip-hook`
// via initcmd.Run directly (internal/cli/contract cannot import the
// internal/cli root package — that would be an import cycle, since
// root wires internal/cli/contract into itself).
func fallbackTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := gitops.Init(context.Background(), root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if rc := initcmd.Run(root, "human/test", false, true, false, "user", false, false, nil, skills.ShippedHooks); rc != cliutil.ExitOK {
		t.Fatalf("aiwf init: rc=%d", rc)
	}
	return root
}

// fallbackFakeValidator writes the same fake validator script
// internal/cli/integration's fakeValidatorCLI uses, returning its
// absolute path. Skips on Windows (shell script).
func fallbackFakeValidator(t *testing.T, dir string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake validator is a /bin/sh script; skipping on Windows")
	}
	path := filepath.Join(dir, "fake-validator.sh")
	body := `#!/bin/sh
fixture="$1"
[ -f "$fixture" ] || { echo "fixture not found: $fixture" >&2; exit 2; }
case "$(head -c 4 "$fixture")" in
  PASS) exit 0 ;;
  *) echo "rejected: $fixture" >&2; exit 1 ;;
esac
`
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("writing fake validator: %v", err)
	}
	return path
}

// fallbackRunID reads path (a diagnostic log file) and returns its
// run_id field, failing the test if the file is missing or not JSON.
func fallbackRunID(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	return rec.RunID
}

func TestRunBind_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := fallbackTestRepo(t)
	script := fallbackFakeValidator(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte("name: fake\ncommand: "+script+"\nargs:\n  - \"{{fixture}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := runRecipeInstall(nil, root, "human/test", customPath, false, cliutil.OutputFormat{}); rc != cliutil.ExitOK {
		t.Fatalf("runRecipeInstall: rc=%d", rc)
	}
	if err := os.WriteFile(filepath.Join(root, "schema.cue"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "fixtures", "v1", "valid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "fixtures", "v1", "valid", "good.json"), []byte("PASS"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := add.Run(entity.KindContract, "Public API", "human/test", "", root,
		"", "", "", "", "", "", "", "",
		"", "", "",
		"", "## Purpose\n\nFixture.\n\n## Stability\n\nFixture.\n", "",
		false, false, cliutil.OutputFormat{}); rc != cliutil.ExitOK {
		t.Fatalf("add contract: rc=%d", rc)
	}

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := runBind("C-0001", root, "human/test", "fake", "schema.cue", "fixtures", false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("runBind: rc=%d", rc)
	}
	if got := fallbackRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

func TestRunUnbind_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := fallbackTestRepo(t)
	script := fallbackFakeValidator(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte("name: fake\ncommand: "+script+"\nargs:\n  - \"{{fixture}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := runRecipeInstall(nil, root, "human/test", customPath, false, cliutil.OutputFormat{}); rc != cliutil.ExitOK {
		t.Fatalf("runRecipeInstall: rc=%d", rc)
	}
	if err := os.WriteFile(filepath.Join(root, "schema.cue"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "fixtures", "v1", "valid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "fixtures", "v1", "valid", "good.json"), []byte("PASS"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := add.Run(entity.KindContract, "Public API", "human/test", "", root,
		"", "", "", "", "", "", "", "",
		"fake", "schema.cue", "fixtures",
		"", "## Purpose\n\nFixture.\n\n## Stability\n\nFixture.\n", "",
		false, false, cliutil.OutputFormat{}); rc != cliutil.ExitOK {
		t.Fatalf("add contract: rc=%d", rc)
	}

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := runUnbind("C-0001", root, "human/test", cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("runUnbind: rc=%d", rc)
	}
	if got := fallbackRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

func TestRunRecipeInstall_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := fallbackTestRepo(t)
	script := fallbackFakeValidator(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte("name: fake\ncommand: "+script+"\nargs:\n  - \"{{fixture}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := runRecipeInstall(nil, root, "human/test", customPath, false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("runRecipeInstall: rc=%d", rc)
	}
	if got := fallbackRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

func TestRunRecipeRemove_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := fallbackTestRepo(t)
	script := fallbackFakeValidator(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte("name: fake\ncommand: "+script+"\nargs:\n  - \"{{fixture}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := runRecipeInstall(nil, root, "human/test", customPath, false, cliutil.OutputFormat{}); rc != cliutil.ExitOK {
		t.Fatalf("runRecipeInstall: rc=%d", rc)
	}

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := runRecipeRemove("fake", root, "human/test", cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("runRecipeRemove: rc=%d", rc)
	}
	if got := fallbackRunID(t, logPath); got == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}
