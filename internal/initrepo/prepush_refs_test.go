package initrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/testsupport"
)

// pushFixture is a clone with the rendered pre-push hook installed, an
// `origin` bare remote already holding `main`, and a stand-in `aiwf` on
// PATH that records each invocation in checkLog.
type pushFixture struct {
	work     string
	env      []string
	checkLog string
}

func newPushFixture(t *testing.T) *pushFixture {
	t.Helper()
	bare := t.TempDir()
	runGit(t, bare, "init", "-q", "--bare")
	work := freshGitRepo(t)
	if err := os.WriteFile(filepath.Join(work, "aiwf.yaml"), []byte("aiwf_version: 0.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", "aiwf.yaml")
	runGit(t, work, "commit", "-q", "-m", "init")
	runGit(t, work, "branch", "-M", "main")
	runGit(t, work, "remote", "add", "origin", bare)
	runGit(t, work, "push", "-q", "origin", "main")

	if err := testsupport.WriteExecutable(filepath.Join(work, ".git", "hooks", "pre-push"), []byte(preHookScript())); err != nil {
		t.Fatal(err)
	}
	shimDir := t.TempDir()
	checkLog := filepath.Join(t.TempDir(), "check.log")
	shim := "#!/bin/sh\necho \"$*\" >> \"" + checkLog + "\"\nexit 0\n"
	if err := testsupport.WriteExecutable(filepath.Join(shimDir, "aiwf"), []byte(shim)); err != nil {
		t.Fatal(err)
	}
	return &pushFixture{
		work:     work,
		env:      append(os.Environ(), "PATH="+shimDir+":"+os.Getenv("PATH")),
		checkLog: checkLog,
	}
}

// commitOn creates branch off the current HEAD, commits one file on it,
// and returns to the branch that was checked out before.
func (f *pushFixture) commitOn(t *testing.T, branch string) {
	t.Helper()
	prev := strings.TrimSpace(runGit(t, f.work, "rev-parse", "--abbrev-ref", "HEAD"))
	runGit(t, f.work, "checkout", "-q", "-b", branch)
	if err := os.WriteFile(filepath.Join(f.work, branch+".txt"), []byte(branch), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, f.work, "add", branch+".txt")
	runGit(t, f.work, "commit", "-q", "-m", branch)
	runGit(t, f.work, "checkout", "-q", prev)
}

// push runs `git push origin <args>` through the installed hook and
// reports whether it succeeded, with its combined output.
func (f *pushFixture) push(t *testing.T, args ...string) (ok bool, output string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"push", "origin"}, args...)...)
	cmd.Dir = f.work
	cmd.Env = f.env
	out, err := cmd.CombinedOutput()
	return err == nil, string(out)
}

func (f *pushFixture) checkRuns(t *testing.T) int {
	t.Helper()
	data, err := os.ReadFile(f.checkLog)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(data), "\n")
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// TestPrePushHook_RefusesRefNotAtHEAD pins G-0685: `aiwf check` judges
// the checked-out branch and working tree, so the hook refuses a pushed
// ref whose commit is not HEAD rather than letting it pass unexamined,
// and runs the check for a ref that is HEAD.
func TestPrePushHook_RefusesRefNotAtHEAD(t *testing.T) {
	t.Parallel()
	f := newPushFixture(t)
	f.commitOn(t, "feature")

	ok, out := f.push(t, "feature")
	if ok {
		t.Fatalf("push of feature from the main checkout succeeded; want refused:\n%s", out)
	}
	if !strings.Contains(out, "refs/heads/feature") {
		t.Errorf("refusal does not name the refused ref:\n%s", out)
	}
	if n := f.checkRuns(t); n != 0 {
		t.Errorf("aiwf check ran %d times for a refused push; want 0", n)
	}

	runGit(t, f.work, "checkout", "-q", "feature")
	if ok, out := f.push(t, "feature"); !ok {
		t.Fatalf("push of the checked-out branch refused:\n%s", out)
	}
	if n := f.checkRuns(t); n != 1 {
		t.Errorf("aiwf check ran %d times for the checked-out branch; want 1", n)
	}
}

// TestPrePushHook_AllowsDeletion pins that deleting a remote branch
// pushes no commit, so no ref is refused.
func TestPrePushHook_AllowsDeletion(t *testing.T) {
	t.Parallel()
	f := newPushFixture(t)
	runGit(t, f.work, "push", "-q", "--no-verify", "origin", "main:refs/heads/doomed")

	if ok, out := f.push(t, "--delete", "doomed"); !ok {
		t.Fatalf("remote branch deletion refused:\n%s", out)
	}
}

// TestPrePushHook_JudgesAnnotatedTagByItsCommit pins that a tag object
// is compared by the commit it points at: a tag of HEAD passes, a tag
// of an older commit is refused.
func TestPrePushHook_JudgesAnnotatedTagByItsCommit(t *testing.T) {
	t.Parallel()
	f := newPushFixture(t)
	runGit(t, f.work, "tag", "-a", "v-old", "-m", "old")
	runGit(t, f.work, "commit", "-q", "--allow-empty", "-m", "next")
	runGit(t, f.work, "tag", "-a", "v-head", "-m", "head")

	if ok, out := f.push(t, "v-head"); !ok {
		t.Fatalf("push of a tag at HEAD refused:\n%s", out)
	}
	ok, out := f.push(t, "v-old")
	if ok {
		t.Fatalf("push of a tag at an older commit succeeded; want refused:\n%s", out)
	}
	if !strings.Contains(out, "refs/tags/v-old") {
		t.Errorf("refusal does not name the refused tag:\n%s", out)
	}
}

// TestPrePushHook_JudgesEveryRefInAPush pins that the refusal reads
// every line git hands the hook: a push carrying HEAD's branch and two
// others is refused, naming each ref not at HEAD and not the one that
// is.
func TestPrePushHook_JudgesEveryRefInAPush(t *testing.T) {
	t.Parallel()
	f := newPushFixture(t)
	runGit(t, f.work, "commit", "-q", "--allow-empty", "-m", "ahead")
	f.commitOn(t, "feature")
	f.commitOn(t, "other")

	ok, out := f.push(t, "main", "feature", "other")
	if ok {
		t.Fatalf("mixed push succeeded; want refused:\n%s", out)
	}
	refusal := refusalLine(out)
	for _, ref := range []string{"refs/heads/feature", "refs/heads/other"} {
		if !strings.Contains(refusal, ref) {
			t.Errorf("refusal %q does not name %s", refusal, ref)
		}
	}
	if strings.Contains(refusal, "refs/heads/main") {
		t.Errorf("refusal %q names refs/heads/main, which is at HEAD", refusal)
	}
	if n := f.checkRuns(t); n != 0 {
		t.Errorf("aiwf check ran %d times for a refused push; want 0", n)
	}
}

// refusalLine returns the hook's refusal line from push output.
func refusalLine(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "refusing to push") {
			return line
		}
	}
	return ""
}

// TestPrePushHook_LocalHookReadsEveryRefLine pins that a pre-push.local
// sibling receives git's ref lines unchanged, every field of every line,
// and that reading them leaves the list intact for the refusal that
// follows it.
func TestPrePushHook_LocalHookReadsEveryRefLine(t *testing.T) {
	t.Parallel()
	f := newPushFixture(t)
	oldMain := strings.TrimSpace(runGit(t, f.work, "rev-parse", "HEAD"))
	runGit(t, f.work, "commit", "-q", "--allow-empty", "-m", "ahead")
	f.commitOn(t, "feature")
	newMain := strings.TrimSpace(runGit(t, f.work, "rev-parse", "main"))
	feature := strings.TrimSpace(runGit(t, f.work, "rev-parse", "feature"))
	zero := strings.Repeat("0", len(feature))
	localLog := filepath.Join(t.TempDir(), "local.log")
	local := "#!/bin/sh\ncat > \"" + localLog + "\"\n"
	if err := testsupport.WriteExecutable(filepath.Join(f.work, ".git", "hooks", "pre-push.local"), []byte(local)); err != nil {
		t.Fatal(err)
	}

	ok, out := f.push(t, "main", "feature")
	if ok {
		t.Fatalf("push of feature from the main checkout succeeded behind a stdin-reading .local hook:\n%s", out)
	}
	if !strings.Contains(refusalLine(out), "refs/heads/feature") {
		t.Errorf("push was not refused by the ref check:\n%s", out)
	}
	seen, err := os.ReadFile(localLog)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSuffix(string(seen), "\n"), "\n")
	sort.Strings(got)
	want := []string{
		"refs/heads/feature " + feature + " refs/heads/feature " + zero,
		"refs/heads/main " + newMain + " refs/heads/main " + oldMain,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("lines pre-push.local read (-want +got):\n%s", diff)
	}
	if !strings.HasSuffix(string(seen), "\n") {
		t.Errorf("pre-push.local stdin %q does not end in a newline", seen)
	}
}

// TestPrePushHook_UpToDatePushGivesLocalHookNoInput pins that a push
// with nothing to send hands pre-push.local an empty stdin, as git
// does, and still runs the check.
func TestPrePushHook_UpToDatePushGivesLocalHookNoInput(t *testing.T) {
	t.Parallel()
	f := newPushFixture(t)
	localLog := filepath.Join(t.TempDir(), "local.log")
	local := "#!/bin/sh\ncat > \"" + localLog + "\"\n"
	if err := testsupport.WriteExecutable(filepath.Join(f.work, ".git", "hooks", "pre-push.local"), []byte(local)); err != nil {
		t.Fatal(err)
	}

	if ok, out := f.push(t, "main"); !ok {
		t.Fatalf("up-to-date push refused:\n%s", out)
	}
	seen, err := os.ReadFile(localLog)
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 0 {
		t.Errorf("pre-push.local stdin = %q; want empty", seen)
	}
	if n := f.checkRuns(t); n != 1 {
		t.Errorf("aiwf check ran %d times; want 1", n)
	}
}

// TestPrePushHook_LocalHookExitDecidesThePush pins that the stdin
// replay leaves pre-push.local's exit status in charge: a failing
// sibling aborts the push before the check, and a sibling that ignores
// stdin and succeeds lets the push through to the check.
func TestPrePushHook_LocalHookExitDecidesThePush(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		local     string
		wantOK    bool
		wantCheck int
	}{
		{"reads stdin and fails", "#!/bin/sh\ncat >/dev/null\nexit 3\n", false, 0},
		{"ignores stdin and succeeds", "#!/bin/sh\nexit 0\n", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newPushFixture(t)
			runGit(t, f.work, "commit", "-q", "--allow-empty", "-m", "ahead")
			if err := testsupport.WriteExecutable(filepath.Join(f.work, ".git", "hooks", "pre-push.local"), []byte(tc.local)); err != nil {
				t.Fatal(err)
			}
			if ok, out := f.push(t, "main"); ok != tc.wantOK {
				t.Fatalf("push ok = %v, want %v:\n%s", ok, tc.wantOK, out)
			}
			if n := f.checkRuns(t); n != tc.wantCheck {
				t.Errorf("aiwf check ran %d times; want %d", n, tc.wantCheck)
			}
		})
	}
}

// TestPrePushHook_RefusesRefWhenHEADIsUnborn pins that a ref naming no
// commit is refused even when HEAD has none either: an unborn branch
// has nothing for the check to judge, so nothing passes as checked.
func TestPrePushHook_RefusesRefWhenHEADIsUnborn(t *testing.T) {
	t.Parallel()
	f := newPushFixture(t)
	runGit(t, f.work, "tag", "tree-tag", "HEAD^{tree}")
	runGit(t, f.work, "checkout", "-q", "--orphan", "fresh")

	ok, out := f.push(t, "tree-tag")
	if ok {
		t.Fatalf("push of a tree tag from an unborn branch succeeded; want refused:\n%s", out)
	}
	if strings.Contains(out, "expected commit type") {
		t.Errorf("refusal carries git's peel error instead of only the hook's message:\n%s", out)
	}
}
