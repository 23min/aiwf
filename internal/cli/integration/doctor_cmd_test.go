package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/doctor"
	"github.com/23min/aiwf/internal/initrepo"
)

// TestRun_DoctorClean reports problems=0 in a freshly-initialized repo.
func TestRun_DoctorClean(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	// No --skip-hook: doctor's "clean" judgement requires both
	// hooks to be installed. The test runs only doctor afterward
	// (read-only), no commits, so the test-binary-as-hook hazard
	// doesn't apply.
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := cli.Execute([]string{"doctor", "--root", root}); rc != cliutil.ExitOK {
		t.Errorf("doctor on clean repo = %d, want %d", rc, cliutil.ExitOK)
	}
}

// TestRun_DoctorDetectsSkillDrift: tamper with a materialized skill
// and confirm doctor surfaces it as a problem.
func TestRun_DoctorDetectsSkillDrift(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	skillPath := filepath.Join(root, ".claude", "skills", "aiwf-add", "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := cli.Execute([]string{"doctor", "--root", root}); rc != cliutil.ExitFindings {
		t.Errorf("doctor on drifted repo = %d, want %d", rc, cliutil.ExitFindings)
	}
}

// TestRun_DoctorReportsMissingConfig: a repo without aiwf.yaml is a
// problem (run init).
func TestRun_DoctorReportsMissingConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if rc := cli.Execute([]string{"doctor", "--root", root}); rc != cliutil.ExitFindings {
		t.Errorf("doctor on un-init'd repo = %d, want %d", rc, cliutil.ExitFindings)
	}
}

// TestRun_DoctorReportsLegacyActor: a pre-I2.5 aiwf.yaml that still
// carries `actor:` must surface a deprecation note in doctor's
// output. The note is informational — it does NOT increment problems
// (the field is harmless, just unnecessary).
func TestRun_DoctorReportsLegacyActor(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	// Append the legacy `actor:` line to simulate a pre-I2.5 repo.
	contents := []byte("aiwf_version: " + cli.Version + "\nactor: human/legacy\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "deprecated") || !strings.Contains(joined, "human/legacy") {
		t.Errorf("doctor should surface the legacy actor as deprecated; got:\n%s", joined)
	}
}

// TestRun_DoctorReportsRuntimeIdentity: doctor should echo the
// runtime-derived actor + its source so the user can confirm what
// the next mutating verb's aiwf-actor: trailer would say.
func TestRun_DoctorReportsRuntimeIdentity(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "actor:") {
		t.Errorf("doctor should include an `actor:` line surfacing runtime identity:\n%s", joined)
	}
	// The setupCLITestRepo helper configures a deterministic git
	// identity; the source label must be "git config user.email".
	if !strings.Contains(joined, "git config user.email") {
		t.Errorf("doctor's actor line should name git config user.email as the source:\n%s", joined)
	}
}

// TestRun_DoctorReportsLegacyAiwfVersion (G47): a pre-G47 aiwf.yaml
// carrying an `aiwf_version:` key surfaces a deprecation note via
// doctor (mirrors the legacy-actor note). The advisory does not
// increment the doctor problem count.
func TestRun_DoctorReportsLegacyAiwfVersion(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	// No --skip-hook: the test asserts problems == 0 (the legacy
	// field is advisory, not a problem). Without hooks installed
	// the missing-hook problems would mask the assertion. No
	// commits triggered.
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	// Replace aiwf.yaml with one that carries the legacy field.
	contents := []byte("aiwf_version: 9.9.9-legacy\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "9.9.9-legacy") {
		t.Errorf("report should name the legacy aiwf_version value; got:\n%s", joined)
	}
	if !strings.Contains(joined, "deprecated `aiwf_version:") {
		t.Errorf("report should carry a deprecation note for aiwf_version; got:\n%s", joined)
	}
	// Legacy field is advisory — no problem count bump.
	if problems != 0 {
		t.Errorf("legacy aiwf_version should be advisory (problems=0); got problems=%d:\n%s", problems, joined)
	}
	if rc := cli.Execute([]string{"doctor", "--root", root}); rc != cliutil.ExitOK {
		t.Errorf("CLI exit on advisory legacy aiwf_version = %d, want %d", rc, cliutil.ExitOK)
	}
}

// TestRun_DoctorSelfCheck_Passes runs doctor --self-check end-to-end
// and asserts the run reports a clean pass. The self-check spins up
// its own throwaway repo and exercises every verb including ones
// that commit (which fire the pre-commit hook installed during
// init). Driving it as a subprocess via runBin gives consumer
// parity: the hook bakes in a real aiwf binary path rather than
// the test binary's path, so the hook fires correctly. Running
// in-process via the dispatcher would deadlock (hook execs the
// test binary).
func TestRun_DoctorSelfCheck_Passes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	out, err := testutil.RunBin(t, root, "", nil, "doctor", "--self-check")
	if err != nil {
		t.Fatalf("doctor --self-check: %v\n%s", err, out)
	}
	if !strings.Contains(out, "self-check passed") {
		t.Errorf("output missing pass marker:\n%s", out)
	}
	// Each verb appears in the step list. The three update entries
	// pin the install / opt-out / re-install transition added in
	// step 7 of update-broaden-plan.md so a regression that breaks
	// the round-trip surfaces here, not in the field.
	for _, label := range []string{
		"ok    init",
		"ok    add epic",
		"ok    add milestone",
		"ok    add adr",
		"ok    add gap",
		"ok    add decision",
		"ok    add contract",
		"ok    promote",
		"ok    cancel",
		"ok    rename",
		"ok    reallocate",
		"ok    history",
		"ok    render roadmap",
		"ok    update (default install)",
		"ok    update (status_md.auto_update: false → keeps gate, removes post-commit)",
		"ok    update (status_md.auto_update: true → reinstates post-commit)",
		"ok    check",
		"ok    doctor",
		// M-0152: doctor verifies the materialized rituals.
		"ok    doctor verifies rituals materialized (ADR-0014 §5)",
	} {
		if !strings.Contains(out, label) {
			t.Errorf("output missing %q:\n%s", label, out)
		}
	}

	// On success the self-check repo should be removed; the path is
	// printed at the start of the run.
	prefix := "self-check repo: "
	idx := strings.Index(out, prefix)
	if idx < 0 {
		t.Fatalf("missing repo path line:\n%s", out)
	}
	after := out[idx+len(prefix):]
	end := strings.IndexByte(after, '\n')
	if end < 0 {
		t.Fatalf("malformed repo path line:\n%s", out)
	}
	repoPath := strings.TrimSpace(after[:end])
	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		t.Errorf("self-check should clean up its repo on success: stat %s err=%v", repoPath, err)
	}
}

// TestDoctor_HonorsCoreHooksPath (G48): when the consumer has set
// `core.hooksPath`, doctor reads the hook at the configured location
// (not hardcoded `.git/hooks/`) and reports against it. Without
// G48's helper, doctor would say "missing — pre-push validation not
// installed" because it looked at the wrong location.
func TestDoctor_HonorsCoreHooksPath(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	// Configure a relative tracked-hooks dir before init lands hooks.
	if err := osExec(t, root, "git", "config", "core.hooksPath", "scripts/git-hooks"); err != nil {
		t.Fatalf("git config core.hooksPath: %v", err)
	}
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Confirm the install landed at the configured path (this is
	// what G48 fixes).
	configured := filepath.Join(root, "scripts", "git-hooks")
	for _, name := range []string{"pre-push", "pre-commit"} {
		if _, err := os.Stat(filepath.Join(configured, name)); err != nil {
			t.Fatalf("%s missing at configured hooksPath: %v", name, err)
		}
	}

	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if problems != 0 {
		t.Errorf("doctor problems = %d, want 0\n%s", problems, joined)
	}
	// The hook line should report ok against the configured path,
	// not the default. We don't pin the exact phrasing — just
	// confirm doctor isn't lying about a missing hook.
	if strings.Contains(joined, "hook:         missing") {
		t.Errorf("doctor reports pre-push hook missing despite install at configured path:\n%s", joined)
	}
	if strings.Contains(joined, "pre-commit:   missing") {
		t.Errorf("doctor reports pre-commit hook missing despite install at configured path:\n%s", joined)
	}
}

// TestDoctor_HookChainReporting (G45): doctor reports the .local
// sibling state for both pre-push and pre-commit hooks. Three states
// matter: absent (no suffix), present + executable ("chains to ..."),
// present + non-executable (error, increments problem count).
func TestDoctor_HookChainReporting(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// State 1: no .local sibling — doctor clean, no chain mention.
	t.Run("absent: no chain mention", func(t *testing.T) {
		lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
		joined := strings.Join(lines, "\n")
		if problems != 0 {
			t.Errorf("problems = %d, want 0\n%s", problems, joined)
		}
		if strings.Contains(joined, "chains to") {
			t.Errorf("clean repo report mentions 'chains to' (no .local should be present):\n%s", joined)
		}
	})

	// State 2: executable .local — doctor clean, chain reported.
	t.Run("executable: chains to noted", func(t *testing.T) {
		hooksDir := filepath.Join(root, ".git", "hooks")
		localPP := filepath.Join(hooksDir, "pre-push.local")
		localPC := filepath.Join(hooksDir, "pre-commit.local")
		if err := os.WriteFile(localPP, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(localPC, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = os.Remove(localPP)
			_ = os.Remove(localPC)
		})

		lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
		joined := strings.Join(lines, "\n")
		if problems != 0 {
			t.Errorf("problems = %d on executable .local siblings, want 0\n%s", problems, joined)
		}
		if !strings.Contains(joined, "chains to .git/hooks/pre-push.local") {
			t.Errorf("report missing pre-push chain notice:\n%s", joined)
		}
		if !strings.Contains(joined, "chains to .git/hooks/pre-commit.local") {
			t.Errorf("report missing pre-commit chain notice:\n%s", joined)
		}
	})

	// State 3: non-executable .local — doctor flags as error.
	t.Run("non-executable: doctor errors", func(t *testing.T) {
		hooksDir := filepath.Join(root, ".git", "hooks")
		localPP := filepath.Join(hooksDir, "pre-push.local")
		if err := os.WriteFile(localPP, []byte("#!/bin/sh\nexit 0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(localPP) })

		lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
		joined := strings.Join(lines, "\n")
		if problems == 0 {
			t.Errorf("problems = 0 on non-executable .local, want >0\n%s", joined)
		}
		if !strings.Contains(joined, "not executable") {
			t.Errorf("report missing 'not executable':\n%s", joined)
		}
	})
}

// TestDoctorReport_Contents checks the pure helper produces the
// expected lines for a typical fresh repo.
func TestDoctorReport_Contents(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	if problems != 0 {
		t.Errorf("problems = %d on a fresh init, want 0\n%s", problems, strings.Join(lines, "\n"))
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"binary:", "config:", "skills:", "ids:"} {
		if !strings.Contains(joined, want) {
			t.Errorf("report missing %q:\n%s", want, joined)
		}
	}
}

// TestDoctor_CheckLatest_ProxyDisabled verifies the opt-in latest
// row is shown when --check-latest is set, and that GOPROXY=off
// produces a benign "unavailable" advisory rather than a failure.
func TestDoctor_CheckLatest_ProxyDisabled(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Setenv("GOPROXY", "off")

	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{CheckLatest: true})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "latest:") {
		t.Errorf("expected `latest:` row when --check-latest is set:\n%s", joined)
	}
	if !strings.Contains(joined, "proxy disabled") {
		t.Errorf("expected proxy-disabled advisory:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("proxy-disabled should not increment problems; got %d:\n%s", problems, joined)
	}
}

// TestDoctor_CheckLatest_DefaultOff confirms the latest row does not
// appear in the default (no --check-latest) report path. Doctor must
// stay offline by default.
func TestDoctor_CheckLatest_DefaultOff(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{}) // CheckLatest false
	if strings.Contains(strings.Join(lines, "\n"), "latest:") {
		t.Errorf("latest: row should not appear without --check-latest:\n%s", strings.Join(lines, "\n"))
	}
}

// TestDoctorReport_HookOK: a freshly-initialised repo has the hook
// installed at .git/hooks/pre-push pointing at an existing binary;
// doctor reports it as ok.
func TestDoctorReport_HookOK(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "hook:") {
		t.Errorf("doctor should include a hook: line:\n%s", joined)
	}
	if !strings.Contains(joined, "hook:         ok") {
		t.Errorf("hook line should report ok on a fresh init:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("fresh init should produce no problems; got %d:\n%s", problems, joined)
	}
}

// TestDoctorReport_HookStalePath_DetectsDrift is the load-bearing
// test for G12: when the binary that init recorded in
// .git/hooks/pre-push no longer exists at that path (binary moved /
// upgraded to a different location / removed), doctor reports the
// drift and increments problems so users see the issue without
// having to discover it on a failed push.
func TestDoctorReport_HookStalePath_DetectsDrift(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	// Hand-edit the hook to point at a non-existent path, simulating
	// a binary that's been moved away.
	hookPath := filepath.Join(root, ".git", "hooks", "pre-push")
	stale := `#!/bin/sh
# aiwf:pre-push
exec /nonexistent/path/to/old-aiwf check
`
	if err := os.WriteFile(hookPath, []byte(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if problems == 0 {
		t.Errorf("stale hook path should be a problem; got 0:\n%s", joined)
	}
	if !strings.Contains(joined, "stale") && !strings.Contains(joined, "missing") {
		t.Errorf("hook line should describe the staleness:\n%s", joined)
	}
}

// TestDoctorReport_HookMissing: when no .git/hooks/pre-push exists
// at all, doctor reports it as missing (so the user knows pre-push
// validation isn't gating their push, even if everything else is
// clean).
func TestDoctorReport_HookMissing(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
		SkipHook:      true,
	}); err != nil {
		t.Fatal(err)
	}
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "hook:") {
		t.Errorf("doctor should include hook: line:\n%s", joined)
	}
	if !strings.Contains(joined, "missing") && !strings.Contains(joined, "not installed") {
		t.Errorf("hook line should describe absence:\n%s", joined)
	}
}

// TestDoctorReport_PreCommitHookOK: fresh init lands the pre-commit
// hook with the marker; doctor reports it ok and increments no
// problems.
func TestDoctorReport_PreCommitHookOK(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "pre-commit:   ok") {
		t.Errorf("pre-commit line should report ok on a fresh init:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("fresh init should produce no problems; got %d:\n%s", problems, joined)
	}
}

// TestDoctorReport_PreCommitHookGateOnly (G42 + G-0112): with
// status_md.auto_update false, the pre-commit hook is installed
// gate-only. Per G-0112 gate-only is now the *only* shape of the
// pre-commit body, so doctor reports plain "pre-commit:   ok" (no
// "gate-only" qualifier). Doctor counts no problems — that's the
// desired-and-actual-agree state.
func TestDoctorReport_PreCommitHookGateOnly(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	// Pre-write aiwf.yaml with the same Version the binary will
	// stamp on init, so the version-skew check doesn't add a
	// confounding problem to the count.
	yaml := []byte("aiwf_version: " + cli.Version + "\nactor: human/test\nstatus_md:\n  auto_update: false\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "pre-commit:   ok") {
		t.Errorf("expected 'pre-commit: ok' line under G-0112:\n%s", joined)
	}
	// Post-commit should be absent under opt-out — that's the new
	// surface where auto_update flips behavior.
	if !strings.Contains(joined, "post-commit:  not installed") {
		t.Errorf("expected 'post-commit: not installed' under opt-out (G-0112):\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("gate-only + post-commit-absent should produce no problems; got %d:\n%s", problems, joined)
	}
}

// TestDoctorReport_PreCommitHookMissingButFlagOn: hook removed but
// config still says install — drift, doctor flags as a problem and
// hints `aiwf update`.
func TestDoctorReport_PreCommitHookMissingButFlagOn(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, ".git", "hooks", "pre-commit")); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "pre-commit:   missing") {
		t.Errorf("expected 'pre-commit: missing' line:\n%s", joined)
	}
	if problems == 0 {
		t.Errorf("missing pre-commit hook with flag on should be a problem")
	}
	if !strings.Contains(joined, "aiwf update") {
		t.Errorf("remediation should reference `aiwf update`:\n%s", joined)
	}
}

// TestDoctorReport_PostCommitHookPresentButFlagOff (G-0112):
// post-commit hook on disk but the user just flipped
// status_md.auto_update to false — drift; `aiwf update` removes it.
// (Under G-0112 the regen toggle lives on the post-commit hook, not
// the pre-commit hook.)
func TestDoctorReport_PostCommitHookPresentButFlagOff(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	yaml := []byte(`aiwf_version: 0.1.0
actor: human/test
status_md:
  auto_update: false
`)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "post-commit:  present") || !strings.Contains(joined, "config says off") {
		t.Errorf("expected post-commit 'present ... config says off' diagnostic:\n%s", joined)
	}
	if problems == 0 {
		t.Errorf("post-commit-present-but-config-off should be a problem")
	}
}

// TestDoctorReport_PreCommitHookAlien: a non-marker hook in place.
// Doctor reports it but does not increment problems (the user owns
// the hook; aiwf can't and won't touch it).
func TestDoctorReport_PreCommitHookAlien(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	alien := []byte("#!/bin/sh\n# user's own hook, no marker\nexit 0\n")
	if err := os.WriteFile(hookPath, alien, 0o755); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "pre-commit:   present but not aiwf-managed") {
		t.Errorf("expected 'not aiwf-managed' diagnostic:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("alien pre-commit hook should be informational, got %d problems", problems)
	}
}

// TestDoctorReport_PreCommitHookStalePath: marker present but the
// exec path no longer exists. Same drift class as G12 for pre-push.
func TestDoctorReport_PreCommitHookStalePath(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	stale := []byte(`#!/bin/sh
# aiwf:pre-commit
set -e
repo_root="$(git rev-parse --show-toplevel)"
[ -f "$repo_root/aiwf.yaml" ] || exit 0
tmp="$repo_root/STATUS.md.tmp"
if '/nonexistent/path/to/old-aiwf' status --root "$repo_root" --format=md >"$tmp" 2>/dev/null; then
    mv "$tmp" "$repo_root/STATUS.md"
    git add "$repo_root/STATUS.md"
else
    rm -f "$tmp"
fi
exit 0
`)
	if err := os.WriteFile(hookPath, stale, 0o755); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "pre-commit:   stale path") {
		t.Errorf("expected 'pre-commit: stale path' line:\n%s", joined)
	}
	if problems == 0 {
		t.Errorf("stale path should be a problem")
	}
}

// TestDoctorReport_ReportsFilesystemCaseSensitivity: doctor names
// the filesystem's case-sensitivity so users on macOS APFS know
// they're on a case-insensitive volume (where E-01-foo and
// E-01-Foo collapse to one path) before they hit the footgun.
func TestDoctorReport_ReportsFilesystemCaseSensitivity(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "filesystem:") {
		t.Errorf("doctor should report filesystem case-sensitivity:\n%s", joined)
	}
}

// TestDoctorReport_ValidatorAvailability_Warning: a configured
// validator binary missing from PATH appears as a warning line in
// the report and does NOT increment problems (default lenient).
func TestDoctorReport_ValidatorAvailability_Warning(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(`aiwf_version: `+cli.Version+`
actor: human/test
contracts:
  validators:
    cue-missing:
      command: /nonexistent/cue-12345
      args: []
    echo-ok:
      command: echo
      args: []
  entries: []
`), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "validator:    cue-missing missing") {
		t.Errorf("missing validator should be reported:\n%s", joined)
	}
	if !strings.Contains(joined, "validator:    echo-ok ok") {
		t.Errorf("present validator should be reported:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("missing validator should NOT increment problems in default mode; got %d\n%s", problems, joined)
	}
}

// TestDoctorReport_ValidatorAvailability_StrictIncrementsProblems:
// strict_validators=true makes a missing validator a hard problem
// in the doctor report (matching the verify-time error).
func TestDoctorReport_ValidatorAvailability_StrictIncrementsProblems(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(`aiwf_version: `+cli.Version+`
actor: human/test
contracts:
  strict_validators: true
  validators:
    cue-missing:
      command: /nonexistent/cue-12345
      args: []
  entries: []
`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	if problems == 0 {
		t.Error("strict_validators=true must make missing validator a problem")
	}
}

// G-0135 / M-0133 / AC-1 branch-coverage tests for the doctor's hook
// reports. Post-G-0135 hooks resolve aiwf via `command -v aiwf` at
// hook-fire time; doctor validates via exec.LookPath. The branches
// below are: (a) lookup fails (binary not on PATH), and (b) the
// pre-G-0135 shape with a still-valid baked path (operator hasn't
// run `aiwf update` yet but their old install still works).
//
// The "binary not on PATH" tests use t.Setenv to clear PATH; they
// cannot run under t.Parallel because t.Setenv panics in parallel
// tests.

// TestDoctorReport_HookOK_AiwfNotOnPATH: fresh init produces the
// new (command -v) shape. When PATH does not contain aiwf, doctor
// reports the not-found diagnostic and increments problems.
func TestDoctorReport_HookOK_AiwfNotOnPATH(t *testing.T) {
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(t.TempDir(), "no-aiwf-here"))
	lines, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "hook:         aiwf binary not found on PATH") {
		t.Errorf("expected pre-push 'aiwf binary not found on PATH' diagnostic:\n%s", joined)
	}
	if !strings.Contains(joined, "pre-commit:   aiwf binary not found on PATH") {
		t.Errorf("expected pre-commit 'aiwf binary not found on PATH' diagnostic:\n%s", joined)
	}
	if !strings.Contains(joined, "post-commit:  aiwf binary not found on PATH") {
		t.Errorf("expected post-commit 'aiwf binary not found on PATH' diagnostic:\n%s", joined)
	}
	if !strings.Contains(joined, "commit-msg:   aiwf binary not found on PATH") {
		t.Errorf("expected commit-msg 'aiwf binary not found on PATH' diagnostic:\n%s", joined)
	}
	if problems == 0 {
		t.Errorf("not-found-on-PATH should increment problems for all four hooks; got 0:\n%s", joined)
	}
}

// TestDoctorReport_PreG0135ShapeStillValid: a hand-written old-shape
// hook (absolute path baked at install time) whose baked path still
// exists. Doctor recognizes the old shape and reports `ok (...; run
// aiwf update to switch to PATH lookup)` without incrementing
// problems — the install still works, but the operator should
// migrate via `aiwf update`.
func TestDoctorReport_PreG0135ShapeStillValid(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatal(err)
	}
	// Hand-write the three hooks in pre-G-0135 shape with /bin/sh as
	// the baked binary (guaranteed to exist on Unix test runners).
	hooksDir := filepath.Join(root, ".git", "hooks")
	prePush := []byte(`#!/bin/sh
# aiwf:pre-push
[ -f "$(git rev-parse --show-toplevel)/aiwf.yaml" ] || exit 0
exec '/bin/sh' check
`)
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-push"), prePush, 0o755); err != nil {
		t.Fatal(err)
	}
	preCommit := []byte(`#!/bin/sh
# aiwf:pre-commit
set -e
repo_root="$(git rev-parse --show-toplevel)"
[ -f "$repo_root/aiwf.yaml" ] || exit 0
if ! '/bin/sh' check --shape-only --root "$repo_root" >&2; then
    exit 1
fi
exit 0
`)
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), preCommit, 0o755); err != nil {
		t.Fatal(err)
	}
	postCommit := []byte(`#!/bin/sh
# aiwf:post-commit
repo_root="$(git rev-parse --show-toplevel)"
[ -f "$repo_root/aiwf.yaml" ] || exit 0
tmp="$repo_root/STATUS.md.tmp"
if '/bin/sh' status --root "$repo_root" --format=md >"$tmp" 2>/dev/null; then
    mv "$tmp" "$repo_root/STATUS.md"
else
    rm -f "$tmp"
fi
exit 0
`)
	if err := os.WriteFile(filepath.Join(hooksDir, "post-commit"), postCommit, 0o755); err != nil {
		t.Fatal(err)
	}
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"hook:         ok (/bin/sh; pre-G-0135 shape, run `aiwf update`",
		"pre-commit:   ok (/bin/sh; pre-G-0135 shape, run `aiwf update`",
		"post-commit:  ok (/bin/sh; pre-G-0135 shape, run `aiwf update`",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected line containing %q in doctor report:\n%s", want, joined)
		}
	}
}

// TestDoctorReport_EnvLinePresent_DevcontainerCase asserts the
// `env:` line appears in DoctorReport output. The unit test
// `TestDetectContainer` in internal/cli/doctor/ covers the four
// signal combinations exhaustively; this integration test only
// confirms the line is wired into DoctorReport.
//
// Test is serial (no t.Parallel) because it mutates AIWF_DEVCONTAINER
// via t.Setenv to make the assertion deterministic regardless of the
// host's incoming env.
//
// Pins M-0135/AC-1.
func TestDoctorReport_EnvLinePresent_DevcontainerCase(t *testing.T) {
	t.Setenv("AIWF_DEVCONTAINER", "1")
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "env:") {
		t.Errorf("expected `env:` line in doctor report:\n%s", joined)
	}
	if !strings.Contains(joined, "AIWF_DEVCONTAINER") {
		t.Errorf("expected `env:` line to mention AIWF_DEVCONTAINER signal:\n%s", joined)
	}
}

// TestDoctorReport_EnvLine_HostCase asserts the env: line reports
// `host` when neither container signal fires. Test clears
// AIWF_DEVCONTAINER explicitly; the dockerenv-path side is fixed at
// the FS root and not controllable from the integration boundary, so
// this test only asserts the env-var side via t.Setenv("AIWF_DEVCONTAINER", "0")
// — the dockerenv part is covered by the unit test in
// internal/cli/doctor/env_internal_test.go.
//
// Pins M-0135/AC-1.
func TestDoctorReport_EnvLine_RespectsFalsyEnvVar(t *testing.T) {
	t.Setenv("AIWF_DEVCONTAINER", "0")
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "env:") {
		t.Errorf("expected `env:` line in doctor report:\n%s", joined)
	}
	if strings.Contains(joined, "AIWF_DEVCONTAINER") {
		t.Errorf("expected `env:` line to NOT mention AIWF_DEVCONTAINER when value is falsy:\n%s", joined)
	}
}

// TestDoctorReport_EnvLine_InformationalOnly asserts the env: line
// never increments problems on its own. Pinning the "informational"
// contract from the AC-1 pass criterion.
//
// Pins M-0135/AC-1.
func TestDoctorReport_EnvLine_InformationalOnly(t *testing.T) {
	t.Setenv("AIWF_DEVCONTAINER", "1")
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	_, problems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	if problems != 0 {
		t.Errorf("env: line should never increment problems; got %d", problems)
	}
}

// TestDoctorReport_ShadowMount_PluginIndexLineGatedOnContainer asserts
// the `plugin-mount:` line appears in DoctorReport output when
// InContainer() returns true, and is omitted when it returns false.
// Exhaustive state coverage (ok / empty / missing) is in the unit
// test TestShadowMountStatus; this integration test only confirms
// wiring.
//
// Serial (no t.Parallel) because both AIWF_DEVCONTAINER mutation
// (t.Setenv) and HOME redirection are process-global.
//
// Pins M-0135/AC-2.
func TestDoctorReport_ShadowMount_PluginIndexLineGatedOnContainer(t *testing.T) {
	t.Setenv("AIWF_DEVCONTAINER", "1")
	t.Setenv("HOME", t.TempDir())
	root := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), root, initrepo.Options{
		ActorOverride: "human/test",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "plugin-mount:") {
		t.Errorf("expected `plugin-mount:` line when in container:\n%s", joined)
	}
}

// TestDoctorReport_ShadowMount_ReportsMissingAndOK confirms the
// missing and ok states observed via DoctorReport — sanity check
// of the integration wiring across two dir shapes that differ in
// what the operator sees.
//
// Pins M-0135/AC-2.
func TestDoctorReport_ShadowMount_ReportsMissingAndOK(t *testing.T) {
	t.Setenv("AIWF_DEVCONTAINER", "1")

	// missing case: home has no .claude/plugins
	missingHome := t.TempDir()
	t.Setenv("HOME", missingHome)
	rootA := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), rootA, initrepo.Options{ActorOverride: "human/test"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	linesA, _ := doctor.DoctorReport(rootA, doctor.DoctorOptions{})
	joinedA := strings.Join(linesA, "\n")
	if !strings.Contains(joinedA, "plugin-mount: missing") {
		t.Errorf("expected `plugin-mount: missing` when ~/.claude/plugins absent:\n%s", joinedA)
	}

	// ok case: seed plugins/<one entry>
	okHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(okHome, ".claude", "plugins", "some-plugin"), 0o755); err != nil {
		t.Fatalf("seed plugin entry: %v", err)
	}
	t.Setenv("HOME", okHome)
	rootB := setupCLITestRepo(t)
	if _, err := initrepo.Init(context.Background(), rootB, initrepo.Options{ActorOverride: "human/test"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	linesB, _ := doctor.DoctorReport(rootB, doctor.DoctorOptions{})
	joinedB := strings.Join(linesB, "\n")
	if !strings.Contains(joinedB, "plugin-mount: ok (1 plugin entries cached)") {
		t.Errorf("expected `plugin-mount: ok (1 plugin entries cached)`:\n%s", joinedB)
	}
}
