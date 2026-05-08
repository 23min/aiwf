package contractverify

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/ai-workflow-v2/internal/aiwfyaml"
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

// TestRun_ValidatorUnavailable: the load-bearing test for G3. A
// configured validator whose binary is not on PATH must produce a
// `validator-unavailable` Result, not a hard `environment` error.
// The downstream wiring then renders this as a warning (not an
// error) unless strict_validators is set.
func TestRun_ValidatorUnavailable(t *testing.T) {
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
	if got[0].Code != CodeValidatorUnavailable {
		t.Errorf("code = %q, want %q", got[0].Code, CodeValidatorUnavailable)
	}
	if got[0].EntityID != "C-001" {
		t.Errorf("entity id = %q, want C-001", got[0].EntityID)
	}
	if !strings.Contains(got[0].Message, "fake") {
		t.Errorf("message should name the validator; got %q", got[0].Message)
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

// --- Edge case coverage (added during the I1 hardening pass) ---

func TestSubstitute_TokenAlone(t *testing.T) {
	got := substitute([]string{"{{schema}}"},
		aiwfyaml.Entry{ID: "C-001", Schema: "s.cue"}, "v1", "fix.json")
	if got[0] != "s.cue" {
		t.Errorf("token-alone substitution = %q, want %q", got[0], "s.cue")
	}
}

func TestSubstitute_ConcatenatedAndRepeated(t *testing.T) {
	args := []string{
		"prefix={{schema}}.suffix",
		"--schema={{schema}} --fixture={{fixture}}",
		"{{schema}}{{fixture}}",
		"--id={{contract_id}}-v={{version}}",
	}
	got := substitute(args,
		aiwfyaml.Entry{ID: "C-001", Schema: "s"}, "v1", "f")
	want := []string{
		"prefix=s.suffix",
		"--schema=s --fixture=f",
		"sf",
		"--id=C-001-v=v1",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSubstitute_LiteralBracesAreNotPlaceholders(t *testing.T) {
	// `{{}}` is not a documented placeholder; the replacer must leave
	// it alone. (strings.NewReplacer matches longest-token-first, so
	// this is essentially testing the replacer doesn't get confused
	// by adjacent braces.)
	got := substitute([]string{"echo {{}}"},
		aiwfyaml.Entry{ID: "C-001", Schema: "s"}, "v1", "f")
	if got[0] != "echo {{}}" {
		t.Errorf("literal `{{}}` mutated to %q", got[0])
	}
}

func TestSubstitute_EmptyArgsSlice(t *testing.T) {
	got := substitute(nil, aiwfyaml.Entry{ID: "C-001"}, "v1", "f")
	if got == nil || len(got) != 0 {
		t.Errorf("substitute(nil) = %v, want empty slice", got)
	}
}

func TestRun_NoReclassificationWhenZeroValidFixtures(t *testing.T) {
	// Only invalid fixtures, none valid. The reclassification rule
	// requires at least one valid fixture; with zero valids, no
	// fixture-rejected can fire and therefore no validator-error.
	repo := t.TempDir()
	script := fakeValidatorScript(t, repo)
	writeFixture(t, repo, "fixtures", "v1", "invalid", "a.json", "FAIL") // ok

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

func TestRun_NoReclassificationWhenSomeValidsPass(t *testing.T) {
	// One valid passes, one valid fails. Reclassification requires
	// *every* valid to fail, so this stays as one fixture-rejected.
	repo := t.TempDir()
	script := fakeValidatorScript(t, repo)
	writeFixture(t, repo, "fixtures", "v1", "valid", "good.json", "PASS")
	writeFixture(t, repo, "fixtures", "v1", "valid", "bad.json", "FAIL")

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
		t.Errorf("code = %q, want %q (no reclassification with mixed results)", got[0].Code, CodeFixtureRejected)
	}
}

func TestRun_MultipleVersions_VerifyAndEvolve(t *testing.T) {
	repo := t.TempDir()
	script := fakeValidatorScript(t, repo)
	// v3 = current; one PASS valid, one PASS invalid... wait, invalid
	// expects FAIL.  Make it correct:
	writeFixture(t, repo, "fixtures", "v3", "valid", "ok.json", "PASS")
	writeFixture(t, repo, "fixtures", "v3", "invalid", "bad.json", "FAIL")
	// v2 historical: one valid that fails HEAD → evolution-regression.
	writeFixture(t, repo, "fixtures", "v2", "valid", "old.json", "FAIL")
	// v1 historical: one valid that passes HEAD → no finding.
	writeFixture(t, repo, "fixtures", "v1", "valid", "ancient.json", "PASS")

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
		t.Fatalf("expected exactly one finding (v2 evolution regression); got %d: %+v", len(got), got)
	}
	if got[0].Code != CodeEvolutionRegression {
		t.Errorf("code = %q, want %q", got[0].Code, CodeEvolutionRegression)
	}
	if got[0].Version != "v2" {
		t.Errorf("version = %q, want v2", got[0].Version)
	}
}

func TestRun_VersionSubstitutionFlowsThrough(t *testing.T) {
	// Validator that records its argv into a sentinel file in the repo,
	// then exits 0. Verifies that {{version}} is substituted with the
	// directory name (not, say, an empty string).
	repo := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh script")
	}
	logFile := filepath.Join(repo, "validator.log")
	script := filepath.Join(repo, "log-validator.sh")
	body := `#!/bin/sh
echo "$1 $2" >> ` + logFile + `
exit 0
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, repo, "fixtures", "v1", "valid", "a.json", "PASS")
	writeFixture(t, repo, "fixtures", "v2", "valid", "b.json", "PASS")

	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}", "{{version}}"}},
		},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "fake", Schema: "s", Fixtures: "fixtures",
		}},
	}
	_ = Run(context.Background(), Options{RepoRoot: repo, Contracts: contracts})
	logged, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("reading validator log: %v", err)
	}
	got := string(logged)
	if !strings.Contains(got, " v1") || !strings.Contains(got, " v2") {
		t.Errorf("validator log did not record both version substitutions:\n%s", got)
	}
}

func TestRun_NonexistentFixturesDirSilent(t *testing.T) {
	repo := t.TempDir()
	script := fakeValidatorScript(t, repo)
	// fixtures: points at a directory that doesn't exist.
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "fake", Schema: "s", Fixtures: "ghost",
		}},
	}
	got := Run(context.Background(), Options{RepoRoot: repo, Contracts: contracts})
	if len(got) != 0 {
		t.Errorf("expected silence for missing fixtures dir (contract-config covers it); got %+v", got)
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	repo := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh script")
	}
	// Validator that sleeps so we have time to cancel.
	script := filepath.Join(repo, "slow.sh")
	body := `#!/bin/sh
sleep 5
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, repo, "fixtures", "v1", "valid", "a.json", "PASS")

	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "fake", Schema: "s", Fixtures: "fixtures",
		}},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Run starts
	got := Run(ctx, Options{RepoRoot: repo, Contracts: contracts})
	// We expect the run to complete quickly and yield a finding (the
	// canceled exec turns into an exec error, classified as
	// fixture-rejected per runValidator's fallback).
	if len(got) == 0 {
		t.Errorf("expected at least one finding from the canceled run")
	}
}

// markerValidatorScript writes a script that creates a marker file
// every time it is invoked. Tests use this to assert the validator
// is NOT invoked for entries with escaping paths — the marker file
// must not exist after Run returns.
func markerValidatorScript(t *testing.T, dir, markerPath string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("posix script")
	}
	path := filepath.Join(dir, "marker-validator.sh")
	body := "#!/bin/sh\n" +
		"echo invoked >> " + markerPath + "\n" +
		"exit 0\n"
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestRun_DoesNotInvokeValidator_ForEscapedSchema is the load-bearing
// test for G1: when an entry's schema path escapes the repo root, the
// validator binary must never be executed against that entry.
func TestRun_DoesNotInvokeValidator_ForEscapedSchema(t *testing.T) {
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(repo, "marker.txt")
	script := markerValidatorScript(t, repo, marker)
	writeFixture(t, repo, "fixtures", "v1", "valid", "a.json", "PASS")

	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "fake", Schema: "../../etc/passwd", Fixtures: "fixtures",
		}},
	}
	_ = Run(context.Background(), Options{RepoRoot: repo, Contracts: contracts})
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("validator was invoked for an entry with an escaping schema; marker.txt exists")
	}
}

// TestRun_DoesNotInvokeValidator_ForEscapedFixtures: same load-bearing
// guarantee, but for the fixtures path (which is what flows through to
// fixture enumeration and would invoke the validator on each file).
func TestRun_DoesNotInvokeValidator_ForEscapedFixtures(t *testing.T) {
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Populate the outside directory with fixtures that look real, so
	// without the guard, runOne would happily enumerate and invoke
	// the validator on them.
	if err := os.MkdirAll(filepath.Join(outside, "v1", "valid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "v1", "valid", "leak.json"), []byte("PASS"), 0o644); err != nil {
		t.Fatal(err)
	}

	marker := filepath.Join(repo, "marker.txt")
	script := markerValidatorScript(t, repo, marker)

	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "fake", Schema: "schema.cue", Fixtures: outside,
		}},
	}
	_ = Run(context.Background(), Options{RepoRoot: repo, Contracts: contracts})
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("validator was invoked for an entry with escaping fixtures; marker.txt exists")
	}
}

// TestRun_MixedEntries_OnlyCleanOneVerifies: with three entries where
// the middle one has clean paths, only the middle one should produce
// validator activity. Confirms the guard skips at the per-entry level.
func TestRun_MixedEntries_OnlyCleanOneVerifies(t *testing.T) {
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	script := fakeValidatorScript(t, repo)
	writeFixture(t, repo, "fixtures", "v1", "valid", "a.json", "PASS")
	writeFixture(t, repo, "fixtures", "v1", "valid", "b.json", "FAIL")

	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"fake": {Command: script, Args: []string{"{{fixture}}"}},
		},
		Entries: []aiwfyaml.Entry{
			{ID: "C-001", Validator: "fake", Schema: "../escape.cue", Fixtures: "fixtures"},
			{ID: "C-002", Validator: "fake", Schema: "schema.cue", Fixtures: "fixtures"},
			{ID: "C-003", Validator: "fake", Schema: "schema.cue", Fixtures: "../escape-fix"},
		},
	}
	got := Run(context.Background(), Options{RepoRoot: repo, Contracts: contracts})
	for _, r := range got {
		if r.EntityID != "C-002" {
			t.Errorf("only C-002 should produce findings; got %+v", r)
		}
	}
	// And C-002 should produce its expected fixture-rejected for b.json.
	found := false
	for _, r := range got {
		if r.EntityID == "C-002" && r.Code == CodeFixtureRejected {
			found = true
		}
	}
	if !found {
		t.Errorf("C-002's fixture-rejected finding missing; got %+v", got)
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
