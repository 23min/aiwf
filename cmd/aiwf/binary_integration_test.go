package main

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Binary integration tests close G27's bug class: they build the
// actual cmd binary and run verbs as subprocesses so the production
// path — `runtime/debug.ReadBuildInfo` plus the ldflags-stamped
// Version global — is exercised the way a user's installed binary
// would be.
//
// `go test` on its own cannot catch the v0.1.0 bug class
// (`aiwf version` returning "dev" while doctor read buildinfo
// correctly); the test binary's buildinfo always reads as "(devel)"
// and there is no runtime way to spoof a different value. Subprocess-
// ing a freshly-built binary is the only path.
//
// Tests are gated under `-short` because each shells out to
// `go build` (~3-5s on a warm cache); CI's default `go test` opts
// in, faster local iterations skip via `-short`.

// TestBinary_VersionVerb_RespectsLdflags pins the ldflags-stamped
// path: a binary built with `-ldflags="-X main.Version=v0.99.0-…"`
// must report that exact value from `aiwf version`. This is the
// `make install` path the kernel-dev repo uses today.
func TestBinary_VersionVerb_RespectsLdflags(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	const stampedVersion = "v0.99.0-binary-integration-test"
	bin := buildBinary(t, tmp, "-ldflags=-X main.Version="+stampedVersion)

	out, err := runBinary(bin, "version")
	if err != nil {
		t.Fatalf("aiwf version: %v\n%s", err, out)
	}
	got := strings.TrimSpace(out)
	if got != stampedVersion {
		t.Errorf("aiwf version = %q, want %q\n(G27: ldflags-stamped value did not reach the verb)", got, stampedVersion)
	}
}

// TestBinary_VersionVerb_FallsBackToBuildInfo pins the seam between
// `aiwf version` and `aiwf doctor`'s `binary:` row when no ldflags
// stamp is present. Both surfaces must report the *same* underlying
// value — a regression that updates one source of truth without the
// other (the v0.1.0 shape) fails this test even when both surfaces
// individually look "right" in isolation.
func TestBinary_VersionVerb_FallsBackToBuildInfo(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp /* no ldflags */)

	verOut, err := runBinary(bin, "version")
	if err != nil {
		t.Fatalf("aiwf version: %v\n%s", err, verOut)
	}
	verVer := strings.TrimSpace(verOut)
	if verVer == "" {
		t.Fatal("aiwf version printed empty output")
	}
	if verVer == "dev" {
		t.Errorf("aiwf version returned literal sentinel %q — G27 regression: the no-ldflags path should defer to runtime/debug.ReadBuildInfo", verVer)
	}

	// doctor's binary: row carries `<version> (<state-label>)`. Pull
	// the version token and assert it matches `aiwf version`'s output.
	doctorOut, err := runBinary(bin, "doctor", "--root", tmp)
	if err != nil && !exitedWithCode(err, 1) {
		// doctor exits 1 ("findings") when aiwf.yaml is missing in
		// --root; that's expected here. Anything else is a real fail.
		t.Fatalf("aiwf doctor: %v\n%s", err, doctorOut)
	}
	row := extractRow(doctorOut, "binary:")
	if row == "" {
		t.Fatalf("aiwf doctor missing 'binary:' row\n%s", doctorOut)
	}
	docVer := versionTokenFromBinaryRow(row)
	if docVer == "" {
		t.Fatalf("could not extract version token from doctor row %q", row)
	}
	if docVer != verVer {
		t.Errorf("seam mismatch (G27): aiwf version = %q, doctor binary: row version = %q\nrow: %s", verVer, docVer, strings.TrimSpace(row))
	}
}

// TestBinary_DoctorSelfCheck_Passes pins M-051 AC-6: every mutating
// verb (add, promote, cancel, rename, reallocate, import, plus
// edit-body and move) runs cleanly inside a real binary subprocess.
// `doctor --self-check` is the canonical end-to-end matrix — it
// scaffolds a throwaway repo, drives every verb through it, and
// asserts the resulting state. Exit 0 here means the entire post-
// Cobra dispatch tree (root command + each verb's flag binding +
// repo lock acquisition + decorateAndFinish path + verb.Apply) works
// the way a user's installed binary does, not just the way go test's
// in-process run() works.
func TestBinary_DoctorSelfCheck_Passes(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp /* no ldflags */)

	out, err := runBinary(bin, "doctor", "--self-check")
	if err != nil {
		t.Fatalf("aiwf doctor --self-check: %v\n%s", err, out)
	}
	if !strings.Contains(out, "self-check passed") {
		t.Errorf("expected 'self-check passed' in output:\n%s", out)
	}
}

// TestBinary_ReadOnlyVerbs_ExitOK pins M-050 AC-4: each migrated
// read-only verb (check, history, doctor, schema, template, render)
// runs cleanly as a subprocess against the migrated Cobra binary
// and returns the contracted exit code. This covers the production
// path the in-process `run()` tests cannot — Cobra's flag parsing,
// our exitError unwrap, and the os.Exit translation only become
// visible when a real binary executes.
func TestBinary_ReadOnlyVerbs_ExitOK(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp /* no ldflags */)

	// Empty repo (no aiwf.yaml, no work tree). Doctor returns 1
	// ("findings"); the others run cleanly with exit 0.
	emptyRepo := t.TempDir()

	cases := []struct {
		name string
		args []string
		want int // expected exit code
	}{
		{"check_empty", []string{"check", "--root", emptyRepo}, 0},
		{"history_unknown_id", []string{"history", "E-99", "--root", emptyRepo}, 0},
		{"doctor_empty", []string{"doctor", "--root", emptyRepo}, 1},
		{"schema_all", []string{"schema"}, 0},
		{"schema_one", []string{"schema", "epic"}, 0},
		{"template_all", []string{"template"}, 0},
		{"template_one", []string{"template", "milestone"}, 0},
		{"render_help", []string{"render", "--help"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runBinary(bin, tc.args...)
			got := 0
			if err != nil {
				if !exitedWithCode(err, tc.want) {
					t.Fatalf("aiwf %v: unexpected error %v\n%s", tc.args, err, out)
				}
				got = tc.want
			}
			if got != tc.want {
				t.Errorf("aiwf %v exit = %d, want %d\n%s", tc.args, got, tc.want, out)
			}
		})
	}
}

// skipIfShortOrUnsupported gates the binary integration tests:
// requires `go` on PATH, skipped under `-short`, skipped on Windows
// (aiwf is unix-only).
func skipIfShortOrUnsupported(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping binary integration test (-short); requires go build")
	}
	if runtime.GOOS == "windows" {
		t.Skip("aiwf is unix-only; binary integration test follows suit")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go not on PATH: %v", err)
	}
}

// buildBinary compiles ./cmd/aiwf into tmp/aiwf with the given
// extra `go build` args (typically `-ldflags=…`) and returns the path.
// Builds happen from the repo root so the relative package path
// resolves regardless of which package the test runs in.
func buildBinary(t *testing.T, tmp string, extraArgs ...string) string {
	t.Helper()
	out := filepath.Join(tmp, "aiwf")
	args := append([]string{"build"}, extraArgs...)
	args = append(args, "-o", out, "./cmd/aiwf")
	cmd := exec.Command("go", args...)
	cmd.Dir = repoRootForTest(t)
	if buildOut, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, buildOut)
	}
	return out
}

// repoRootForTest walks up from the test's cwd looking for go.mod
// and returns the absolute directory containing it. The test binary
// runs in the package directory (cmd/aiwf); the repo root is
// two levels up.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	for i := 0; i < 6; i++ {
		if _, err := exec.Command("test", "-f", filepath.Join(dir, "go.mod")).Output(); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate repo root (no go.mod in 6 parents)")
	return ""
}

// runBinary invokes bin with args and returns combined stdout+stderr.
// Combined output is what a user sees, so the assertions read the
// same bytes the user would.
func runBinary(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// exitedWithCode reports whether err is an *exec.ExitError with the
// given exit code. Used to tolerate doctor's `exitFindings` (1) when
// no aiwf.yaml is present in --root.
func exitedWithCode(err error, code int) bool {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode() == code
	}
	return false
}

// extractRow returns the first line of haystack whose prefix (after
// trimming leading whitespace) matches prefix. Empty if not found.
func extractRow(haystack, prefix string) string {
	for _, line := range strings.Split(haystack, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), prefix) {
			return line
		}
	}
	return ""
}

// versionTokenFromBinaryRow extracts the version string from a
// doctor `binary:` row of the shape "binary:    <version> (<state>)".
// Returns the value between the colon-space and the trailing
// state-label parenthetical. Empty when the row doesn't match.
func versionTokenFromBinaryRow(row string) string {
	row = strings.TrimSpace(row)
	const prefix = "binary:"
	if !strings.HasPrefix(row, prefix) {
		return ""
	}
	rest := strings.TrimSpace(row[len(prefix):])
	if i := strings.LastIndex(rest, " ("); i > 0 {
		return rest[:i]
	}
	return rest
}

// TestBinary_RenderHTML_EndToEnd builds the cmd binary, runs it
// against a freshly-init'd repo, exercises every interactive
// surface the renderer has (epic + milestones + ACs + phase
// history + scope), and asserts the resulting site is well-formed
// and contains the expected per-tab content. This is the
// I3-step-7 binary-level safety net — `go test` in-process tests
// can't catch a regression that only appears once the cmd is
// shelled out (e.g., embed.FS resolution from a packed binary,
// flag-parsing differences when args land via os.Args vs. run()).
func TestBinary_RenderHTML_EndToEnd(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp /* no ldflags */)

	// Build the consumer repo via the binary, just like a user
	// would. Each verb is its own subprocess; failure on any of
	// them is a real bug.
	repo := filepath.Join(tmp, "consumer")
	if err := exec.Command("mkdir", "-p", repo).Run(); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := exec.Command("git", "-C", repo, "init", "-q").Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	for _, kv := range []struct{ k, v string }{
		{"user.email", "test@example.com"},
		{"user.name", "test"},
	} {
		if err := exec.Command("git", "-C", repo, "config", kv.k, kv.v).Run(); err != nil {
			t.Fatalf("git config %s: %v", kv.k, err)
		}
	}
	for _, args := range [][]string{
		{"init", "--root", repo, "--actor", "human/test"},
		{"add", "epic", "--root", repo, "--actor", "human/test", "--title", "Foundations"},
		{"add", "milestone", "--root", repo, "--actor", "human/test", "--epic", "E-01", "--title", "Schema parser"},
		{"add", "ac", "--root", repo, "--actor", "human/test", "M-001", "--title", "Engine starts"},
		{"promote", "--root", repo, "--actor", "human/test", "M-001/AC-1", "--phase", "red"},
		{
			"promote", "--root", repo, "--actor", "human/test", "M-001/AC-1", "--phase", "green",
			"--tests", "pass=12 fail=0 skip=1",
		},
		{"authorize", "--root", repo, "--actor", "human/test", "M-001", "--to", "ai/claude"},
	} {
		if out, err := runBinary(bin, args...); err != nil {
			t.Fatalf("aiwf %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	siteDir := filepath.Join(tmp, "site")
	out, err := runBinary(bin, "render", "--root", repo, "--format", "html", "--out", siteDir)
	if err != nil {
		t.Fatalf("aiwf render: %v\n%s", err, out)
	}
	// Envelope reports out_dir + files_written + elapsed_ms.
	// Fixture via cmd-side resolver:
	//   1 index + 1 status + 1 epic (E-01) + 1 milestone (M-001) = 4
	if !strings.Contains(out, `"files_written":4`) {
		t.Errorf("envelope did not report files_written=4 (index + status + epic + milestone): %s", out)
	}

	// Page-level assertions through the binary — the templates
	// must produce the same content via the cmd binary as via
	// in-process run(), pinning the embed.FS resolution path.
	for _, name := range []string{"index.html", "E-01.html", "M-001.html", "assets/style.css"} {
		path := filepath.Join(siteDir, name)
		if _, statErr := exec.Command("test", "-f", path).Output(); statErr != nil {
			t.Errorf("expected %s in site dir; %v", name, statErr)
		}
	}

	mHTML := readFileT(t, filepath.Join(siteDir, "M-001.html"))
	if err := assertWellFormed(mHTML); err != nil {
		t.Errorf("M-001.html (binary render) is not well-formed: %v", err)
	}
	assertContainsIn(t, mHTML, "build", "phase-red", "binary render: Build tab missing red phase")
	assertContainsIn(t, mHTML, "build", "phase-green", "binary render: Build tab missing green phase")
	assertContainsIn(t, mHTML, "build", "pass=12", "binary render: aiwf-tests trailer not surfaced")
	assertContainsIn(t, mHTML, "provenance", "scope-state-active", "binary render: scope row missing")
}
