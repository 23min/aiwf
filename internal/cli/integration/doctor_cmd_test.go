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
		// M-070/AC-7: end-to-end coverage of the recommended-plugin
		// check, both the warning-fires path and the install-silences
		// path. Adding these labels keeps the seam test honest about
		// what the production self-check actually exercises.
		"ok    doctor recommended-plugins fixture: declare in aiwf.yaml",
		"ok    doctor recommended-plugins fixture: warning silent after enable in settings.json",
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
	if strings.Contains(joined, "hook:      missing") {
		t.Errorf("doctor reports pre-push hook missing despite install at configured path:\n%s", joined)
	}
	if strings.Contains(joined, "pre-commit: missing") {
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
	if !strings.Contains(joined, "hook:      ok") {
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
	if !strings.Contains(joined, "pre-commit: ok") {
		t.Errorf("pre-commit line should report ok on a fresh init:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("fresh init should produce no problems; got %d:\n%s", problems, joined)
	}
}

// TestDoctorReport_PreCommitHookGateOnly (G42 + G-0112): with
// status_md.auto_update false, the pre-commit hook is installed
// gate-only. Per G-0112 gate-only is now the *only* shape of the
// pre-commit body, so doctor reports plain "pre-commit: ok" (no
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
	if !strings.Contains(joined, "pre-commit: ok") {
		t.Errorf("expected 'pre-commit: ok' line under G-0112:\n%s", joined)
	}
	// Post-commit should be absent under opt-out — that's the new
	// surface where auto_update flips behavior.
	if !strings.Contains(joined, "post-commit: not installed") {
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
	if !strings.Contains(joined, "pre-commit: missing") {
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
	if !strings.Contains(joined, "post-commit: present") || !strings.Contains(joined, "config says off") {
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
	if !strings.Contains(joined, "pre-commit: present but not aiwf-managed") {
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
	if !strings.Contains(joined, "pre-commit: stale path") {
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
	if !strings.Contains(joined, "validator: cue-missing missing") {
		t.Errorf("missing validator should be reported:\n%s", joined)
	}
	if !strings.Contains(joined, "validator: echo-ok ok") {
		t.Errorf("present validator should be reported:\n%s", joined)
	}
	if problems != 0 {
		t.Errorf("missing validator should NOT increment problems in default mode; got %d\n%s", problems, joined)
	}
}

// TestDoctorReport_RecommendedPlugins_EmptyConfig_NoOutputNoFileRead:
// M-070/AC-4 — when `doctor.recommended_plugins` is absent or empty,
// the new check makes zero observations: no `recommended-plugin-not-installed`
// line in the output and no problems++. Verified with two configs:
// the field absent entirely + an explicit `[]`. To make the
// "no file read" half observable without process tracing, $HOME is
// set to a directory we did NOT populate; if the check incorrectly
// tried to read `installed_plugins.json` it would still find nothing
// and return empty — so this test pairs with the fixture-injected
// AC-5/AC-6 tests below where the file presence matters.
func TestDoctorReport_RecommendedPlugins_EmptyConfig_NoOutputNoFileRead(t *testing.T) {
	cases := []struct {
		name      string
		yamlExtra string
	}{
		{name: "field absent", yamlExtra: ""},
		{name: "empty list", yamlExtra: "doctor:\n  recommended_plugins: []\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := setupCLITestRepo(t)
			if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
				t.Fatalf("init: %d", rc)
			}
			// Append (or rewrite) aiwf.yaml with the test's extra block.
			contents := []byte("hosts: [claude-code]\n" + tc.yamlExtra)
			if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("HOME", t.TempDir())
			lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
			joined := strings.Join(lines, "\n")
			if strings.Contains(joined, "recommended-plugin-not-installed") {
				t.Errorf("empty config emitted recommended-plugin warnings:\n%s", joined)
			}
		})
	}
}

// TestDoctorReport_RecommendedPlugins_OneMissing_OneWarningWithInstall:
// M-070/AC-3 — one declared, none installed. Exactly one warning line
// per missing plugin; the warning text carries (a) the
// `recommended-plugin-not-installed` finding code so a script can grep,
// (b) the plugin id name@marketplace, and (c) the canonical install
// command `claude /plugin install <id>`.
func TestDoctorReport_RecommendedPlugins_OneMissing_OneWarningWithInstall(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	contents := []byte("hosts: [claude-code]\ndoctor:\n  recommended_plugins:\n    - aiwf-extensions@ai-workflow-rituals\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if c := strings.Count(joined, "recommended-plugin-not-installed"); c != 1 {
		t.Errorf("count(recommended-plugin-not-installed) = %d, want 1; output:\n%s", c, joined)
	}
	if !strings.Contains(joined, "aiwf-extensions@ai-workflow-rituals") {
		t.Errorf("warning missing plugin id; output:\n%s", joined)
	}
	if !strings.Contains(joined, "PROJECT scope") {
		t.Errorf("warning missing install advice with PROJECT scope guidance; output:\n%s", joined)
	}
}

// TestDoctorReport_RecommendedPlugins_NoneInstalled_NWarnings: M-070/AC-3
// — N declared, none installed produces exactly N warnings (never
// deduped, never skipped). Order matches declaration order so the user
// can correlate with their aiwf.yaml.
func TestDoctorReport_RecommendedPlugins_NoneInstalled_NWarnings(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	contents := []byte(`hosts: [claude-code]
doctor:
  recommended_plugins:
    - aiwf-extensions@ai-workflow-rituals
    - wf-rituals@ai-workflow-rituals
    - some-third@somewhere
`)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if c := strings.Count(joined, "recommended-plugin-not-installed"); c != 3 {
		t.Errorf("count(recommended-plugin-not-installed) = %d, want 3; output:\n%s", c, joined)
	}
	for _, plugin := range []string{
		"aiwf-extensions@ai-workflow-rituals",
		"wf-rituals@ai-workflow-rituals",
		"some-third@somewhere",
	} {
		if !strings.Contains(joined, plugin) {
			t.Errorf("warning missing plugin %q; output:\n%s", plugin, joined)
		}
	}
}

// TestDoctorReport_RecommendedPlugins_AllInstalledForProject_NoWarning
// and TestDoctorReport_RecommendedPlugins_InstalledElsewhereStillWarns
// were removed in G-0138 / M-0133 / AC-3: their setup used
// installed_plugins.json's projectPath equality (the path-strict
// check that produced the false-positive class fixed by AC-3). The
// new source of truth is <rootDir>/.claude/settings.json's
// enabledPlugins map (path-independent), tested by
// TestDoctorReport_RecommendedPlugins_EnabledInSettings_NoWarning
// below.

// TestDoctorReport_RecommendedPlugins_AreSoftWarning_DoNotIncrementProblems:
// per M-070 spec: "Severity: warning. Plugins are advisory; refusing on
// absence is too strong." Even when the warning fires, doctor's exit
// code stays 0 (problems unchanged). This decoupling is what lets a
// consumer declare recommended plugins without breaking CI.
func TestDoctorReport_RecommendedPlugins_AreSoftWarning_DoNotIncrementProblems(t *testing.T) {
	root := setupCLITestRepo(t)
	// --skip-hook keeps the test independent of hook installation; but
	// with skip-hook, doctor reports a missing-hook problem. The
	// AC-targeted assertion is "the recommended-plugin warning does
	// not contribute to the problem count" — measured by comparing the
	// problem count with vs. without the recommended-plugin block.
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	t.Setenv("HOME", t.TempDir())
	// Baseline problems count with no recommended_plugins declared.
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, baseProblems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	// Add a missing recommended plugin; problems must not increase.
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\ndoctor:\n  recommended_plugins:\n    - aiwf-extensions@ai-workflow-rituals\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, withProblems := doctor.DoctorReport(root, doctor.DoctorOptions{})
	if withProblems != baseProblems {
		t.Errorf("recommended-plugin warning incremented problems: base=%d with=%d (warning must be soft)", baseProblems, withProblems)
	}
}

// TestAppendRecommendedPluginsReport_NilCfg_NoOp moved with the doctor
// verb to internal/cli/doctor/internal_test.go (M-0117/AC-3) — the
// helper is unexported in its new home, so the test rebases as a
// same-package test there.

// TestDoctorReport_RecommendedPlugins_CorruptedIndex_EmitsAdvisory was
// removed in G-0138 / M-0133 / AC-3: doctor no longer reads
// installed_plugins.json, so a corrupt index can't surface here. The
// equivalent test for the new source of truth is
// TestDoctorReport_RecommendedPlugins_MalformedSettings_Error
// (malformed .claude/settings.json surfaces as a clear plugins: line
// naming the file). writeInstalledPluginsFixture was deleted with
// its last caller.

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

// G-0138 / M-0133 / AC-3: the doctor's recommended-plugins check
// sources truth from the project's committed `.claude/settings.json`
// `enabledPlugins` map (path-independent), not the machine-local
// `~/.claude/plugins/installed_plugins.json` (path-strict; false-
// positives across worktrees, devcontainers, and re-clones).

// TestDoctorReport_RecommendedPlugins_EnabledInSettings_NoWarning:
// a plugin declared in <rootDir>/.claude/settings.json's
// `enabledPlugins` map silences the warning, regardless of
// installed_plugins.json state.
func TestDoctorReport_RecommendedPlugins_EnabledInSettings_NoWarning(t *testing.T) {
	// t.Setenv below precludes t.Parallel.
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	yaml := []byte("hosts: [claude-code]\ndoctor:\n  recommended_plugins:\n    - aiwf-extensions@ai-workflow-rituals\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	settings := []byte(`{"enabledPlugins":{"aiwf-extensions@ai-workflow-rituals":true}}`)
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".claude", "settings.json"), settings, 0o644); err != nil {
		t.Fatal(err)
	}
	// Deliberately set HOME to an empty dir to prove the check
	// does NOT depend on installed_plugins.json — settings.json
	// alone silences the warning.
	t.Setenv("HOME", t.TempDir())
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "recommended-plugin-not-installed") {
		t.Errorf("enabledPlugins=true should silence the warning regardless of installed_plugins.json state; got:\n%s", joined)
	}
}

// TestDoctorReport_RecommendedPlugins_AdviceMentionsProjectScope:
// the warning's install advice tells the operator how to install
// at PROJECT scope (the bare CLI form defaults to user scope per
// Claude Code docs; the doctor's advice must steer the user toward
// the right scope).
func TestDoctorReport_RecommendedPlugins_AdviceMentionsProjectScope(t *testing.T) {
	// t.Setenv below precludes t.Parallel.
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	yaml := []byte("hosts: [claude-code]\ndoctor:\n  recommended_plugins:\n    - aiwf-extensions@ai-workflow-rituals\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "recommended-plugin-not-installed") {
		t.Fatalf("warning should fire (no enabledPlugins); got:\n%s", joined)
	}
	if !strings.Contains(joined, "PROJECT scope") {
		t.Errorf("install advice should mention 'PROJECT scope' (vs default user scope); got:\n%s", joined)
	}
}

// TestDoctorReport_RecommendedPlugins_MalformedSettings_Error:
// invalid JSON in .claude/settings.json surfaces as a clear error
// in the report, not a silent skip.
func TestDoctorReport_RecommendedPlugins_MalformedSettings_Error(t *testing.T) {
	// t.Setenv below precludes t.Parallel.
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	yaml := []byte("hosts: [claude-code]\ndoctor:\n  recommended_plugins:\n    - aiwf-extensions@ai-workflow-rituals\n")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".claude", "settings.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	lines, _ := doctor.DoctorReport(root, doctor.DoctorOptions{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, ".claude/settings.json") {
		t.Errorf("malformed settings.json should surface in plugins: line with filename; got:\n%s", joined)
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
	if !strings.Contains(joined, "hook:      aiwf binary not found on PATH") {
		t.Errorf("expected pre-push 'aiwf binary not found on PATH' diagnostic:\n%s", joined)
	}
	if !strings.Contains(joined, "pre-commit: aiwf binary not found on PATH") {
		t.Errorf("expected pre-commit 'aiwf binary not found on PATH' diagnostic:\n%s", joined)
	}
	if !strings.Contains(joined, "post-commit: aiwf binary not found on PATH") {
		t.Errorf("expected post-commit 'aiwf binary not found on PATH' diagnostic:\n%s", joined)
	}
	if problems == 0 {
		t.Errorf("not-found-on-PATH should increment problems for all three hooks; got 0:\n%s", joined)
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
		"hook:      ok (/bin/sh; pre-G-0135 shape, run `aiwf update`",
		"pre-commit: ok (/bin/sh; pre-G-0135 shape, run `aiwf update`",
		"post-commit: ok (/bin/sh; pre-G-0135 shape, run `aiwf update`",
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
