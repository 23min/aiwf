package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// This file exercises the defensive / drift branches of DoctorReport and
// its hook-report helpers that the `[]Problem` refactor (M-0224) touched.
// Each test constructs the reachable repo state directly in a t.TempDir()
// and calls the specific helper — mirroring the (nil, nil, root) pattern
// in commit_msg_report_test.go and internal_test.go. Two branches are
// genuinely unreachable (compiled-in embed reads) and carry a
// `//coverage:ignore` in doctor.go instead.

// newHooks creates <root>/.git/hooks and returns its path. resolveHooksDir
// falls back to this location because a bare t.TempDir() is not a git repo.
func newHooks(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// writeHookFile writes a hook body at <hooks>/<name>, executable.
func writeHookFile(t *testing.T, hooks, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(hooks, name), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

// nonExecLocal drops a present-but-non-executable <name>.local sibling —
// the state localChainSuffix reports as a chain problem.
func nonExecLocal(t *testing.T, hooks, name string) {
	t.Helper()
	p := filepath.Join(hooks, name+".local")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o644); err != nil { // guarantee no exec bits regardless of umask
		t.Fatal(err)
	}
}

// existingBin creates a real file under root and returns its absolute
// path, standing in for a pre-G-0135 baked exec path that os.Stat
// resolves (so the stale-path check passes).
func existingBin(t *testing.T, root string) string {
	t.Helper()
	p := filepath.Join(root, "fake-aiwf")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// mkHookDir places a *directory* at the hook path so os.ReadFile fails
// with a non-ErrNotExist error (EISDIR) — the read-error branch.
func mkHookDir(t *testing.T, hooks, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(hooks, name), 0o755); err != nil {
		t.Fatal(err)
	}
}

// ---- pre-push (appendHookReport) --------------------------------------

// TestAppendHookReport_ReadError: a directory at .git/hooks/pre-push makes
// os.ReadFile fail with a non-ErrNotExist error — doctor.go:400.
func TestAppendHookReport_ReadError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	mkHookDir(t, hooks, "pre-push")
	_, problems := appendHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
}

// TestAppendHookReport_Malformed: marker present, no `command -v aiwf`,
// no `exec` line — extractHookExecPath returns "" — doctor.go:437.
func TestAppendHookReport_Malformed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	writeHookFile(t, hooks, "pre-push", "#!/bin/sh\n# aiwf:pre-push\nexit 0\n")
	lines, problems := appendHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
	if out := strings.Join(lines, "\n"); !strings.Contains(out, "malformed") {
		t.Errorf("want malformed diagnostic; got:\n%s", out)
	}
}

// TestAppendHookReport_PreG0135ChainProblem: a pre-G-0135 baked path that
// exists, plus a non-executable pre-push.local sibling — the chainProblem
// arm of the pre-G-0135 ok path — doctor.go:452.
func TestAppendHookReport_PreG0135ChainProblem(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	bin := existingBin(t, root)
	writeHookFile(t, hooks, "pre-push", fmt.Sprintf("#!/bin/sh\n# aiwf:pre-push\nexec '%s' check\n", bin))
	nonExecLocal(t, hooks, "pre-push")
	lines, problems := appendHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
	if out := strings.Join(lines, "\n"); !strings.Contains(out, "not executable") {
		t.Errorf("want non-executable .local diagnostic; got:\n%s", out)
	}
}

// ---- pre-commit (appendPreCommitHookReport) ---------------------------

// TestAppendPreCommitHookReport_ReadError: directory at the hook path —
// doctor.go:517.
func TestAppendPreCommitHookReport_ReadError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	mkHookDir(t, hooks, "pre-commit")
	_, problems := appendPreCommitHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
}

// TestAppendPreCommitHookReport_PostG0135StatusDrift: PATH-lookup shape
// carrying the stale `status --root` regen step (G-0112) — doctor.go:540.
// Relies on aiwf being on PATH (guaranteed by the test environment, as
// the existing commit-msg OurHook test also assumes).
func TestAppendPreCommitHookReport_PostG0135StatusDrift(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	writeHookFile(t, hooks, "pre-commit", "#!/bin/sh\n# aiwf:pre-commit\ncommand -v aiwf\naiwf status --root .\n")
	lines, problems := appendPreCommitHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
	if out := strings.Join(lines, "\n"); !strings.Contains(out, "STATUS.md regen") {
		t.Errorf("want G-0112 drift diagnostic; got:\n%s", out)
	}
}

// TestAppendPreCommitHookReport_PostG0135ChainProblem: PATH-lookup shape,
// no drift, with a non-executable .local sibling — doctor.go:549.
func TestAppendPreCommitHookReport_PostG0135ChainProblem(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	writeHookFile(t, hooks, "pre-commit", "#!/bin/sh\n# aiwf:pre-commit\ncommand -v aiwf\nexec aiwf check --shape-only\n")
	nonExecLocal(t, hooks, "pre-commit")
	lines, problems := appendPreCommitHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
	if out := strings.Join(lines, "\n"); !strings.Contains(out, "not executable") {
		t.Errorf("want non-executable .local diagnostic; got:\n%s", out)
	}
}

// TestAppendPreCommitHookReport_PreG0135Malformed: marker, no PATH-lookup,
// no `if '<path>' …` line — extractPreCommitExecPath returns "" —
// doctor.go:559.
func TestAppendPreCommitHookReport_PreG0135Malformed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	writeHookFile(t, hooks, "pre-commit", "#!/bin/sh\n# aiwf:pre-commit\nexit 0\n")
	lines, problems := appendPreCommitHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
	if out := strings.Join(lines, "\n"); !strings.Contains(out, "malformed") {
		t.Errorf("want malformed diagnostic; got:\n%s", out)
	}
}

// TestAppendPreCommitHookReport_PreG0135ChainAndDrift: pre-G-0135 baked
// path that exists, a non-executable .local sibling (chainProblem —
// doctor.go:572), AND a stale `status --root` step (drift — doctor.go:575).
// Both fire: chainProblem appends first, the drift check appends and
// returns, so two error problems land.
func TestAppendPreCommitHookReport_PreG0135ChainAndDrift(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	bin := existingBin(t, root)
	writeHookFile(t, hooks, "pre-commit", fmt.Sprintf("#!/bin/sh\n# aiwf:pre-commit\nif '%s' status --root .; then :; fi\n", bin))
	nonExecLocal(t, hooks, "pre-commit")
	lines, problems := appendPreCommitHookReport(nil, nil, root)
	if got := errorCount(problems); got != 2 {
		t.Errorf("error problems = %d, want 2 (chain + drift)", got)
	}
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "STATUS.md regen") {
		t.Errorf("want G-0112 drift diagnostic; got:\n%s", out)
	}
	var sawChain bool
	for i := range problems {
		if strings.Contains(problems[i].Message, "not executable") {
			sawChain = true
		}
	}
	if !sawChain {
		t.Errorf("want a chain problem naming the non-executable .local; got %+v", problems)
	}
}

// ---- commit-msg (appendCommitMsgHookReport) ---------------------------

// TestAppendCommitMsgHookReport_ReadError: directory at the hook path —
// doctor.go:602.
func TestAppendCommitMsgHookReport_ReadError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	mkHookDir(t, hooks, "commit-msg")
	_, problems := appendCommitMsgHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
}

// TestAppendCommitMsgHookReport_ChainProblem: marker + aiwf on PATH + a
// non-executable commit-msg.local sibling — doctor.go:623.
func TestAppendCommitMsgHookReport_ChainProblem(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	writeHookFile(t, hooks, "commit-msg", "#!/bin/sh\n# aiwf:commit-msg\nexec aiwf check --commit-msg \"$1\"\n")
	nonExecLocal(t, hooks, "commit-msg")
	lines, problems := appendCommitMsgHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
	if out := strings.Join(lines, "\n"); !strings.Contains(out, "not executable") {
		t.Errorf("want non-executable .local diagnostic; got:\n%s", out)
	}
}

// ---- post-commit (appendPostCommitHookReport) -------------------------

// TestAppendPostCommitHookReport_ReadError: directory at the hook path
// (autoUpdate defaults to true with no aiwf.yaml) — doctor.go:653.
func TestAppendPostCommitHookReport_ReadError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	mkHookDir(t, hooks, "post-commit")
	_, problems := appendPostCommitHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
}

// TestAppendPostCommitHookReport_AlienNoMarker: present but no
// `# aiwf:post-commit` marker — a warning, not an error — doctor.go:659.
func TestAppendPostCommitHookReport_AlienNoMarker(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	writeHookFile(t, hooks, "post-commit", "#!/bin/sh\n# user's own hook\nexit 0\n")
	lines, problems := appendPostCommitHookReport(nil, nil, root)
	if got := errorCount(problems); got != 0 {
		t.Errorf("error problems = %d, want 0 (alien hook is a warning)", got)
	}
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "not aiwf-managed") {
		t.Errorf("want not-aiwf-managed diagnostic; got:\n%s", out)
	}
	var sawWarn bool
	for i := range problems {
		if problems[i].Severity == SeverityWarn {
			sawWarn = true
		}
	}
	if !sawWarn {
		t.Errorf("want a warn problem; got %+v", problems)
	}
}

// TestAppendPostCommitHookReport_PostG0135ChainProblem: marker + PATH
// lookup + aiwf on PATH + a non-executable .local sibling — doctor.go:684.
func TestAppendPostCommitHookReport_PostG0135ChainProblem(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	writeHookFile(t, hooks, "post-commit", "#!/bin/sh\n# aiwf:post-commit\ncommand -v aiwf\nexec aiwf status --root .\n")
	nonExecLocal(t, hooks, "post-commit")
	lines, problems := appendPostCommitHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
	if out := strings.Join(lines, "\n"); !strings.Contains(out, "not executable") {
		t.Errorf("want non-executable .local diagnostic; got:\n%s", out)
	}
}

// TestAppendPostCommitHookReport_PreG0135Malformed: marker, no PATH
// lookup, no `if '<path>' …` line — extractPreCommitExecPath returns "" —
// doctor.go:692.
func TestAppendPostCommitHookReport_PreG0135Malformed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	writeHookFile(t, hooks, "post-commit", "#!/bin/sh\n# aiwf:post-commit\nexit 0\n")
	lines, problems := appendPostCommitHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
	if out := strings.Join(lines, "\n"); !strings.Contains(out, "malformed") {
		t.Errorf("want malformed diagnostic; got:\n%s", out)
	}
}

// TestAppendPostCommitHookReport_PreG0135StalePath: pre-G-0135 baked path
// that no longer exists — the stale-path branch — doctor.go:698.
func TestAppendPostCommitHookReport_PreG0135StalePath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	writeHookFile(t, hooks, "post-commit", "#!/bin/sh\n# aiwf:post-commit\nif '/nonexistent/path/old-aiwf' status --root .; then :; fi\n")
	lines, problems := appendPostCommitHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
	if out := strings.Join(lines, "\n"); !strings.Contains(out, "stale path") {
		t.Errorf("want stale-path diagnostic; got:\n%s", out)
	}
}

// TestAppendPostCommitHookReport_PreG0135ChainProblem: pre-G-0135 baked
// path that exists, plus a non-executable .local sibling — doctor.go:707.
func TestAppendPostCommitHookReport_PreG0135ChainProblem(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := newHooks(t, root)
	bin := existingBin(t, root)
	writeHookFile(t, hooks, "post-commit", fmt.Sprintf("#!/bin/sh\n# aiwf:post-commit\nif '%s' status --root .; then :; fi\n", bin))
	nonExecLocal(t, hooks, "post-commit")
	lines, problems := appendPostCommitHookReport(nil, nil, root)
	if got := errorCount(problems); got != 1 {
		t.Errorf("error problems = %d, want 1", got)
	}
	if out := strings.Join(lines, "\n"); !strings.Contains(out, "not executable") {
		t.Errorf("want non-executable .local diagnostic; got:\n%s", out)
	}
}

// ---- DoctorReport inline sections -------------------------------------

// TestDoctorReport_ConfigLoadError: a malformed aiwf.yaml is a load error
// distinct from not-found — the config `err != nil` arm — doctor.go:200.
func TestDoctorReport_ConfigLoadError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// `a: b: c` is a mapping value in a context that forbids it — yaml
	// rejects it, so config.Load returns a parse error (not ErrNotFound).
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("a: b: c\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, problems := DoctorReport(root, DoctorOptions{})
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "config:") || !strings.Contains(out, "parsing aiwf.yaml") {
		t.Errorf("want a config parse-error line; got:\n%s", out)
	}
	var sawErr bool
	for i := range problems {
		if problems[i].Severity == SeverityError && strings.Contains(problems[i].Message, "parsing aiwf.yaml") {
			sawErr = true
		}
	}
	if !sawErr {
		t.Errorf("want an error problem naming the aiwf.yaml parse failure; got %+v", problems)
	}
}

// TestDoctorReport_DetachedHead: a detached HEAD (currentBranch=="" and
// headIsDetached) yields the advisory `head:` line and a warn problem —
// doctor.go:247.
func TestDoctorReport_DetachedHead(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init")
	git("commit", "--allow-empty", "-m", "init")
	git("checkout", "--detach")

	lines, problems := DoctorReport(root, DoctorOptions{})
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "detached-head") {
		t.Errorf("want a detached-head advisory line; got:\n%s", out)
	}
	var sawWarn bool
	for i := range problems {
		if problems[i].Severity == SeverityWarn && strings.Contains(problems[i].Message, "detached-head") {
			sawWarn = true
		}
	}
	if !sawWarn {
		t.Errorf("want a warn problem for the detached HEAD; got %+v", problems)
	}
}

// TestDoctorReport_TreeLoadError: a repo whose `work` entry is a regular
// file makes tree.Load stat work/epics through a non-directory (ENOTDIR,
// not ErrNotExist), so it hard-errors — the ids `err != nil` arm —
// doctor.go:280.
func TestDoctorReport_TreeLoadError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "work"), []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, problems := DoctorReport(root, DoctorOptions{})
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "ids:") || !strings.Contains(out, "statting") {
		t.Errorf("want an ids: line carrying the tree-load stat error; got:\n%s", out)
	}
	var sawErr bool
	for i := range problems {
		if problems[i].Severity == SeverityError && strings.Contains(problems[i].Message, "statting") {
			sawErr = true
		}
	}
	if !sawErr {
		t.Errorf("want an error problem naming the tree-load failure; got %+v", problems)
	}
}

// TestDoctorReport_IDCollision: two entities sharing one id make check.Run
// emit an ids-unique finding, which the ids section surfaces as a
// collision line and an error problem — doctor.go:288.
func TestDoctorReport_IDCollision(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	gaps := filepath.Join(root, "work", "gaps")
	if err := os.MkdirAll(gaps, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: G-0001\ntitle: %s\nstatus: open\n---\n\nbody\n"
	if err := os.WriteFile(filepath.Join(gaps, "G-0001-one.md"), []byte(fmt.Sprintf(body, "one")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gaps, "G-0001-two.md"), []byte(fmt.Sprintf(body, "two")), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, problems := DoctorReport(root, DoctorOptions{})
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "collision G-0001") {
		t.Errorf("want an ids collision line for G-0001; got:\n%s", out)
	}
	var sawErr bool
	for i := range problems {
		if problems[i].Severity == SeverityError && strings.Contains(problems[i].Message, "collision G-0001") {
			sawErr = true
		}
	}
	if !sawErr {
		t.Errorf("want an error problem naming the G-0001 collision; got %+v", problems)
	}
}
