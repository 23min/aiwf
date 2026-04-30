package contractverify

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/ai-workflow-v2/tools/internal/aiwfyaml"
)

// fakeValidatorScript writes a small shell script to dir that exits
// 0 when the (final) argument's file content starts with "PASS" and
// non-zero otherwise. Returns the absolute path to the script.
//
// macOS and Linux only; aiwf does not run on Windows so the test
// suite skips when GOOS is "windows".
func fakeValidatorScript(t *testing.T, dir string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("contractverify tests use a /bin/sh script; skipping on Windows")
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

// writeFixture writes content to <root>/<fixturesRel>/<version>/<bucket>/<name>.
// bucket is "valid" or "invalid".
func writeFixture(t *testing.T, root, fixturesRel, version, bucket, name, content string) {
	t.Helper()
	dir := filepath.Join(root, fixturesRel, version, bucket)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSubstitute_AppliesAllFourVariables(t *testing.T) {
	args := []string{
		"--schema={{schema}}",
		"--fixture={{fixture}}",
		"--id={{contract_id}}",
		"--version={{version}}",
		"plain",
	}
	got := substitute(args,
		aiwfyaml.Entry{ID: "C-001", Schema: "s.cue"},
		"v3", "fix.json")
	want := []string{
		"--schema=s.cue",
		"--fixture=fix.json",
		"--id=C-001",
		"--version=v3",
		"plain",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("substitute mismatch (-want +got):\n%s", diff)
	}
}

func TestEnumerateVersions_PicksHighest(t *testing.T) {
	root := t.TempDir()
	fixDir := filepath.Join(root, "fixtures")
	for _, v := range []string{"v1", "v2", "v3"} {
		if err := os.MkdirAll(filepath.Join(fixDir, v), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A non-directory entry must be skipped.
	if err := os.WriteFile(filepath.Join(fixDir, "README.md"), []byte("ignored"), 0o644); err != nil {
		t.Fatal(err)
	}
	versions, err := enumerateVersions(fixDir)
	if err != nil {
		t.Fatalf("enumerateVersions: %v", err)
	}
	if diff := cmp.Diff([]string{"v1", "v2", "v3"}, versions); diff != "" {
		t.Errorf("versions (-want +got):\n%s", diff)
	}
}

func TestEnumerateVersions_MissingDir(t *testing.T) {
	versions, err := enumerateVersions(filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("expected error for missing dir")
	}
	if versions != nil {
		t.Errorf("versions should be nil on error; got %+v", versions)
	}
}

func TestWalkFixtures_RegularFilesOnly(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.json", "b.json", "c.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("PASS"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A subdirectory must be ignored (not recursed).
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := walkFixtures(dir)
	if len(got) != 3 {
		t.Errorf("got %d files, want 3: %+v", len(got), got)
	}
}

func TestRun_VerifyPassClean(t *testing.T) {
	repo := t.TempDir()
	script := fakeValidatorScript(t, repo)

	// One version v1 with two valid (PASS) and one invalid (FAIL).
	writeFixture(t, repo, "fixtures", "v1", "valid", "a.json", "PASS")
	writeFixture(t, repo, "fixtures", "v1", "valid", "b.json", "PASS")
	writeFixture(t, repo, "fixtures", "v1", "invalid", "c.json", "FAIL")

	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "fake", Schema: "s", Fixtures: "fixtures",
		}},
	}
	got := Run(context.Background(), Options{RepoRoot: repo, Contracts: contracts})
	if len(got) != 0 {
		t.Errorf("expected no findings; got %+v", got)
	}
}

func TestRun_FixtureRejected_OneFailingValid(t *testing.T) {
	repo := t.TempDir()
	script := fakeValidatorScript(t, repo)
	writeFixture(t, repo, "fixtures", "v1", "valid", "a.json", "PASS")
	writeFixture(t, repo, "fixtures", "v1", "valid", "b.json", "FAIL")
	writeFixture(t, repo, "fixtures", "v1", "invalid", "c.json", "FAIL")

	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "fake", Schema: "s", Fixtures: "fixtures",
		}},
	}
	got := Run(context.Background(), Options{RepoRoot: repo, Contracts: contracts})
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(got), got)
	}
	if got[0].Code != CodeFixtureRejected {
		t.Errorf("code = %q, want %q", got[0].Code, CodeFixtureRejected)
	}
	if !strings.Contains(got[0].FixturePath, "b.json") {
		t.Errorf("FixturePath = %q, want b.json", got[0].FixturePath)
	}
	if !strings.Contains(got[0].Detail, "rejected") {
		t.Errorf("Detail did not capture validator stderr: %q", got[0].Detail)
	}
}

func TestRun_FixtureAccepted_OneInvalidPasses(t *testing.T) {
	repo := t.TempDir()
	script := fakeValidatorScript(t, repo)
	writeFixture(t, repo, "fixtures", "v1", "valid", "a.json", "PASS")
	// invalid fixture *should* fail but content "PASS" makes it pass.
	writeFixture(t, repo, "fixtures", "v1", "invalid", "c.json", "PASS")

	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "fake", Schema: "s", Fixtures: "fixtures",
		}},
	}
	got := Run(context.Background(), Options{RepoRoot: repo, Contracts: contracts})
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(got), got)
	}
	if got[0].Code != CodeFixtureAccepted {
		t.Errorf("code = %q, want %q", got[0].Code, CodeFixtureAccepted)
	}
}

func TestRun_ValidatorError_AllValidsFail(t *testing.T) {
	repo := t.TempDir()
	script := fakeValidatorScript(t, repo)
	// Every valid fixture fails — reclassify to validator-error.
	writeFixture(t, repo, "fixtures", "v1", "valid", "a.json", "FAIL")
	writeFixture(t, repo, "fixtures", "v1", "valid", "b.json", "FAIL")
	writeFixture(t, repo, "fixtures", "v1", "invalid", "c.json", "FAIL") // ok

	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "fake", Schema: "s", Fixtures: "fixtures",
		}},
	}
	got := Run(context.Background(), Options{RepoRoot: repo, Contracts: contracts})
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1 collapsed: %+v", len(got), got)
	}
	if got[0].Code != CodeValidatorError {
		t.Errorf("code = %q, want %q (collapsed from per-fixture rejects)", got[0].Code, CodeValidatorError)
	}
	// Detail must carry the per-fixture stderr so the user has
	// something to act on.
	if !strings.Contains(got[0].Detail, "a.json") || !strings.Contains(got[0].Detail, "b.json") {
		t.Errorf("validator-error detail missing per-fixture context: %q", got[0].Detail)
	}
}

func TestRun_EvolutionRegression(t *testing.T) {
	repo := t.TempDir()
	script := fakeValidatorScript(t, repo)
	// Current version v2 is clean.
	writeFixture(t, repo, "fixtures", "v2", "valid", "current.json", "PASS")
	// Historical version v1 valid fixture fails HEAD schema.
	writeFixture(t, repo, "fixtures", "v1", "valid", "old.json", "FAIL")

	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "fake", Schema: "s", Fixtures: "fixtures",
		}},
	}
	got := Run(context.Background(), Options{RepoRoot: repo, Contracts: contracts})
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(got), got)
	}
	if got[0].Code != CodeEvolutionRegression {
		t.Errorf("code = %q, want %q", got[0].Code, CodeEvolutionRegression)
	}
	if got[0].Version != "v1" {
		t.Errorf("version = %q, want v1 (the historical version)", got[0].Version)
	}
}

func TestRun_EnvironmentMissingBinary(t *testing.T) {
	repo := t.TempDir()
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: "/nonexistent/path/to/validator-binary", Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "fake", Schema: "s", Fixtures: "fixtures",
		}},
	}
	got := Run(context.Background(), Options{RepoRoot: repo, Contracts: contracts})
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(got), got)
	}
	if got[0].Code != CodeEnvironment {
		t.Errorf("code = %q, want %q", got[0].Code, CodeEnvironment)
	}
}

func TestRun_SkipsTerminalContracts(t *testing.T) {
	repo := t.TempDir()
	script := fakeValidatorScript(t, repo)
	// C-001 has a failing valid fixture but is skipped.
	writeFixture(t, repo, "f1", "v1", "valid", "a.json", "FAIL")
	// C-002 has clean fixtures.
	writeFixture(t, repo, "f2", "v1", "valid", "b.json", "PASS")

	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{
			{ID: "C-001", Validator: "fake", Schema: "s", Fixtures: "f1"},
			{ID: "C-002", Validator: "fake", Schema: "s", Fixtures: "f2"},
		},
	}
	got := Run(context.Background(), Options{
		RepoRoot:  repo,
		Contracts: contracts,
		SkipIDs:   map[string]bool{"C-001": true},
	})
	if len(got) != 0 {
		t.Errorf("expected no findings (C-001 skipped, C-002 clean); got %+v", got)
	}
}

func TestRun_EmptyFixturesDirSkippedSilently(t *testing.T) {
	repo := t.TempDir()
	script := fakeValidatorScript(t, repo)
	// Create the fixtures dir but leave it empty.
	if err := os.MkdirAll(filepath.Join(repo, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}

	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "fake", Schema: "s", Fixtures: "fixtures",
		}},
	}
	got := Run(context.Background(), Options{RepoRoot: repo, Contracts: contracts})
	if len(got) != 0 {
		t.Errorf("expected no findings for empty fixtures dir; got %+v", got)
	}
}

func TestRun_NilContractsReturnsEmpty(t *testing.T) {
	got := Run(context.Background(), Options{RepoRoot: t.TempDir(), Contracts: nil})
	if got != nil {
		t.Errorf("expected nil; got %+v", got)
	}
}

func TestCombineStdStreams(t *testing.T) {
	if got := combineStdStreams(nil, nil); got != "" {
		t.Errorf("empty case = %q", got)
	}
	if got := combineStdStreams([]byte("hi\n"), nil); got != "hi" {
		t.Errorf("stdout-only = %q", got)
	}
	if got := combineStdStreams(nil, []byte("err\n")); got != "err" {
		t.Errorf("stderr-only = %q", got)
	}
	got := combineStdStreams([]byte("a\n"), []byte("b\n"))
	if !strings.Contains(got, "[stdout]") || !strings.Contains(got, "[stderr]") {
		t.Errorf("combined missing labels: %q", got)
	}
}
