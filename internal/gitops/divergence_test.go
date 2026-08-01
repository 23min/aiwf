package gitops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// seedCommittedFiles initialises a repo, writes each file, and commits
// them — so every path named is present in HEAD and equal to it on disk.
func seedCommittedFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	names := make([]string, 0, len(files))
	for name, content := range files {
		writeAt(t, root, name, content)
		names = append(names, name)
	}
	if err := Add(ctx, root, names...); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return root
}

// addAndCommit stages everything in root and commits it, for a fixture
// step that lands after the initial seed.
func addAndCommit(t *testing.T, root, subject string) {
	t.Helper()
	ctx := context.Background()
	if err := Add(ctx, root, "."); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := Commit(ctx, root, subject, "", nil); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

// writeAt writes content at a repo-relative, forward-slash path,
// creating parent directories as needed.
func writeAt(t *testing.T, root, relPath, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestDivergentPaths_ClassifiesEveryWayAPathCanDisagreeWithHEAD is the
// primitive's contract. For each requested path it answers whether the
// working copy still equals the record, and when it does not, in which
// of the three possible ways.
//
// The nested path is not decoration: every entity file lives under
// work/<kind>/, so a comparison that only worked at the repo root would
// be inert at every call site that matters.
func TestDivergentPaths_ClassifiesEveryWayAPathCanDisagreeWithHEAD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{
		"clean.md":             "unchanged\n",
		"work/gaps/edited.md":  "committed\n",
		"work/gaps/deleted.md": "committed\n",
	})
	writeAt(t, root, "work/gaps/edited.md", "hand-edited\n")
	if err := os.Remove(filepath.Join(root, "work", "gaps", "deleted.md")); err != nil {
		t.Fatal(err)
	}
	writeAt(t, root, "work/gaps/new.md", "never committed\n")

	got, err := DivergentPaths(ctx, root, []string{
		"clean.md",
		"work/gaps/edited.md",
		"work/gaps/deleted.md",
		"work/gaps/new.md",
	})
	if err != nil {
		t.Fatalf("DivergentPaths: %v", err)
	}
	want := []Divergence{
		{Path: "work/gaps/deleted.md", Kind: DivergenceAbsentFromDisk},
		{Path: "work/gaps/edited.md", Kind: DivergenceModified},
		{Path: "work/gaps/new.md", Kind: DivergenceAbsentFromHEAD},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DivergentPaths (-want +got):\n%s", diff)
	}
}

// TestDivergentPaths_ReportsOnlyTheRequestedPaths pins the scoping every
// claim-side call site depends on. A verb asking whether one entity is
// already at the requested state must not be refused because the operator
// is midway through editing an unrelated file.
func TestDivergentPaths_ReportsOnlyTheRequestedPaths(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{
		"asked-about.md": "committed\n",
		"unrelated.md":   "committed\n",
	})
	writeAt(t, root, "asked-about.md", "edited\n")
	writeAt(t, root, "unrelated.md", "also edited\n")

	got, err := DivergentPaths(ctx, root, []string{"asked-about.md"})
	if err != nil {
		t.Fatalf("DivergentPaths: %v", err)
	}
	want := []Divergence{{Path: "asked-about.md", Kind: DivergenceModified}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DivergentPaths (-want +got):\n%s", diff)
	}
}

// TestDivergentPaths_CleanTreeReportsNothing is the negative control: a
// primitive that reported divergence for an untouched path would refuse
// every verb in a clean repo.
func TestDivergentPaths_CleanTreeReportsNothing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{"clean.md": "unchanged\n"})

	got, err := DivergentPaths(ctx, root, []string{"clean.md"})
	if err != nil {
		t.Fatalf("DivergentPaths: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("clean path reported as divergent: %+v", got)
	}
}

// TestDivergentPaths_EmptyRequestReportsNothing covers the guard every
// call site hits when a verb's claim is scoped to no path at all.
func TestDivergentPaths_EmptyRequestReportsNothing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{"clean.md": "unchanged\n"})

	got, err := DivergentPaths(ctx, root, nil)
	if err != nil {
		t.Fatalf("DivergentPaths: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty request reported %+v, want nothing", got)
	}
}

// TestDivergentPaths_NonRepoErrors pins the fail-loud direction: a
// comparison that cannot be made must not read as "nothing diverges".
//
// The record's side is read through one batch pump for the whole set, so
// a directory that is no repo fails before any individual path is
// reached — and the error names the directory, which is what is actually
// wrong. A per-path failure still names its path
// (TestDivergentPaths_UnreadablePathErrors).
func TestDivergentPaths_NonRepoErrors(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_, err := DivergentPaths(context.Background(), root, []string{"any.md"})
	if err == nil {
		t.Fatal("DivergentPaths on a non-repo dir: want error, got nil")
	}
	if !strings.Contains(err.Error(), root) {
		t.Errorf("error does not name the root it failed on:\n%v", err)
	}
	if !strings.Contains(err.Error(), "not a git repo") {
		t.Errorf("error does not say what is wrong with it:\n%v", err)
	}
}

// TestDivergentPaths_ByteExactAcrossFraming pins that the record's side
// is the file's bytes and nothing else. The comparison is byte equality
// and the blobs arrive over git's batch protocol, whose framing is
// length-prefixed with a trailing newline of its own — so content that
// ends without a newline, or has none at all, is where a framing byte
// would leak in and report an untouched file as modified.
func TestDivergentPaths_ByteExactAcrossFraming(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	files := map[string]string{
		"empty.md":      "",
		"no-newline.md": "no trailing newline",
		"two-blanks.md": "text\n\n\n",
		"binary-ish.md": "\x00\x01\x02 not really text\n",
		"normal.md":     "ordinary content\n",
	}
	root := seedCommittedFiles(t, files)

	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	got, err := DivergentPaths(ctx, root, paths)
	if err != nil {
		t.Fatalf("DivergentPaths: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("committed files reported as divergent from their own record: %+v", got)
	}
}

// TestDivergentPaths_UnreadablePathErrors pins the fail-loud direction
// for a path that exists but cannot be read as a file. A directory where
// a file was expected is the reachable case: a verb whose plan names a
// directory-shaped kind hands one straight through.
func TestDivergentPaths_UnreadablePathErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{"work/epics/E-0001/epic.md": "committed\n"})

	// The directory is present at HEAD only through its contents, so this
	// is a read failure rather than an absent path.
	_, err := DivergentPaths(ctx, root, []string{"work/epics/E-0001"})
	if err == nil {
		t.Fatal("DivergentPaths on a directory: want error, got nil")
	}
	if !strings.Contains(err.Error(), "work/epics/E-0001") {
		t.Errorf("error does not name the unreadable path:\n%v", err)
	}
}

// TestDivergentPaths_AbsentFromBothIsNotDivergence covers the arm a
// statement-level gate cannot distinguish: a path in neither HEAD nor the
// working tree. Nothing disagrees about it — the caller asked about
// something that does not exist, which is a resolution question rather
// than a divergence one.
func TestDivergentPaths_AbsentFromBothIsNotDivergence(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{"clean.md": "unchanged\n"})

	got, err := DivergentPaths(ctx, root, []string{"work/gaps/never-existed.md"})
	if err != nil {
		t.Fatalf("DivergentPaths: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("a path absent from both sides reported as divergent: %+v", got)
	}
}

// gitConfig sets a repo-local config value on a seeded fixture.
func gitConfig(t *testing.T, root, key, value string) {
	t.Helper()
	cmd := exec.Command("git", "config", key, value)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config %s %s: %v\n%s", key, value, err, out)
	}
}

// TestDivergentPaths_MatchesWhatTheCommitPathWouldStore pins the
// transformation applied to the working-copy side, which decides what
// "unchanged" means here.
//
// The verb commit path stores the working copy's bytes verbatim, so a
// path is unchanged exactly when those bytes already equal the blob the
// record holds. Applying git's clean filter to the disk side instead
// compares against a convention these commits do not follow: in a repo
// carrying content filters it reports a path as divergent whose bytes
// HEAD already holds, refusing a verb that would rewrite nothing.
//
// The fixture commits through the same seam the verbs use, so the blob
// under test is one aiwf itself wrote.
func TestDivergentPaths_MatchesWhatTheCommitPathWouldStore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{"seed.md": "seed\n"})
	gitConfig(t, root, "core.autocrlf", "true")

	// Content a clean filter would rewrite, stored verbatim the way a
	// verb's commit stores it.
	body := []byte("first\r\nsecond\r\n")
	writeAt(t, root, "work/gaps/G-0001-a.md", string(body))
	if _, err := CommitVerbChange(ctx, root, nil,
		[]PathWrite{{Path: "work/gaps/G-0001-a.md", Content: body}},
		"fixture: store CRLF bytes verbatim", "", nil); err != nil {
		t.Fatalf("CommitVerbChange: %v", err)
	}

	got, err := DivergentPaths(ctx, root, []string{"work/gaps/G-0001-a.md"})
	if err != nil {
		t.Fatalf("DivergentPaths: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("a path whose bytes the record already holds was called divergent: %+v", got)
	}

	// And a real edit to the same path is still caught.
	writeAt(t, root, "work/gaps/G-0001-a.md", "first\r\nEDITED\r\n")
	got, err = DivergentPaths(ctx, root, []string{"work/gaps/G-0001-a.md"})
	if err != nil {
		t.Fatalf("DivergentPaths: %v", err)
	}
	want := []Divergence{{Path: "work/gaps/G-0001-a.md", Kind: DivergenceModified}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("edited path (-want +got):\n%s", diff)
	}
}

// TestDivergentPaths_AwkwardPathNamesAreCompared pins that a path is
// carried on git's argument vector rather than through a line-oriented
// protocol. A space truncates a whitespace-split response; a newline
// splits one request into two and shifts every later answer onto the
// wrong path — silently, with a nil error.
//
// The unrelated clean paths are the assertion that matters: a
// desynchronised protocol reports them as divergent while they are
// untouched.
func TestDivergentPaths_AwkwardPathNamesAreCompared(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{
		"work/gaps/with space.md": "committed\n",
		"clean-a.md":              "identical\n",
		"clean-b.md":              "identical\n",
	})
	writeAt(t, root, "work/gaps/with space.md", "hand-edited\n")

	// The newline path exists nowhere; it must not disturb its neighbours.
	got, err := DivergentPaths(ctx, root, []string{
		"work/gaps/two\nlines.md", "clean-a.md", "work/gaps/with space.md", "clean-b.md",
	})
	if err != nil {
		t.Fatalf("DivergentPaths: %v", err)
	}
	want := []Divergence{{Path: "work/gaps/with space.md", Kind: DivergenceModified}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DivergentPaths (-want +got):\n%s", diff)
	}
}

// TestDivergentPaths_SymlinkComparesItsTarget pins that a link is
// compared as what git stores for it — the target string — not as the
// bytes of whatever it points at. Reading through the link can never
// equal the record, so an untouched tracked symlink reads as modified
// forever and no remedy clears it.
func TestDivergentPaths_SymlinkComparesItsTarget(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{
		"work/epics/E-0001/M-0001.md": "milestone body\n",
		"work/epics/E-0001/other.md":  "other body\n",
	})
	link := filepath.Join(root, "work", "epics", "E-0001", "latest.md")
	if err := os.Symlink("M-0001.md", link); err != nil {
		t.Skipf("symlinks unavailable on this platform: %v", err)
	}
	addAndCommit(t, root, "fixture: a tracked symlink")

	got, err := DivergentPaths(ctx, root, []string{"work/epics/E-0001/latest.md"})
	if err != nil {
		t.Fatalf("DivergentPaths over an untouched link: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("an untouched symlink was called divergent: %+v", got)
	}

	// Re-point it: the target string changed, so the record disagrees.
	if rmErr := os.Remove(link); rmErr != nil {
		t.Fatal(rmErr)
	}
	if linkErr := os.Symlink("other.md", link); linkErr != nil {
		t.Fatal(linkErr)
	}
	got, err = DivergentPaths(ctx, root, []string{"work/epics/E-0001/latest.md"})
	if err != nil {
		t.Fatalf("DivergentPaths over a re-pointed link: %v", err)
	}
	want := []Divergence{{Path: "work/epics/E-0001/latest.md", Kind: DivergenceModified}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("re-pointed link (-want +got):\n%s", diff)
	}
}

// TestDivergentPaths_PathThatExistsNowhereIsNotDivergence covers the
// shape where a parent component is a file rather than a directory. The
// working tree does not hold the path and neither does the record, so
// there is nothing to disagree about — reporting it as present-on-disk
// would let a phantom decline a real candidate downstream.
func TestDivergentPaths_PathThatExistsNowhereIsNotDivergence(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{"clean.md": "unchanged\n"})

	got, err := DivergentPaths(ctx, root, []string{"clean.md/nested.md", "does/not/exist.md"})
	if err != nil {
		t.Fatalf("DivergentPaths: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("a path present in neither place was reported as divergent: %+v", got)
	}
}

// TestDivergentPaths_SymlinkAgainstADifferentRecord covers the two ways
// a link on disk can disagree with the record other than by target.
//
// Both are paths a directory move would carry into a commit: an
// untracked link the record has never held, and a link standing where
// the record holds an ordinary file. The second is a change of entry
// kind that no object-id comparison expresses, so it is decided before
// the ids are compared at all.
func TestDivergentPaths_SymlinkAgainstADifferentRecord(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{
		"work/epics/E-0001/M-0001.md":   "milestone body\n",
		"work/epics/E-0001/was-file.md": "an ordinary committed file\n",
	})

	// An untracked link: on disk, never recorded.
	fresh := filepath.Join(root, "work", "epics", "E-0001", "untracked-link.md")
	if err := os.Symlink("M-0001.md", fresh); err != nil {
		t.Skipf("symlinks unavailable on this platform: %v", err)
	}
	// A link standing where the record holds a file.
	swapped := filepath.Join(root, "work", "epics", "E-0001", "was-file.md")
	if err := os.Remove(swapped); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("M-0001.md", swapped); err != nil {
		t.Fatal(err)
	}

	got, err := DivergentPaths(ctx, root, []string{
		"work/epics/E-0001/untracked-link.md",
		"work/epics/E-0001/was-file.md",
	})
	if err != nil {
		t.Fatalf("DivergentPaths: %v", err)
	}
	want := []Divergence{
		{Path: "work/epics/E-0001/untracked-link.md", Kind: DivergenceAbsentFromHEAD},
		{Path: "work/epics/E-0001/was-file.md", Kind: DivergenceModified},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DivergentPaths (-want +got):\n%s", diff)
	}
}

// TestDivergentPaths_FileWhereTheRecordHoldsALink is the mirror: the
// record holds a link and the working tree an ordinary file. Comparing
// object ids would compare the target string against the file's bytes,
// which is a comparison of two unrelated things.
func TestDivergentPaths_FileWhereTheRecordHoldsALink(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{"work/epics/E-0001/M-0001.md": "milestone body\n"})
	link := filepath.Join(root, "work", "epics", "E-0001", "latest.md")
	if err := os.Symlink("M-0001.md", link); err != nil {
		t.Skipf("symlinks unavailable on this platform: %v", err)
	}
	addAndCommit(t, root, "fixture: a tracked symlink")

	// Replace the link with a real file holding the target's own bytes,
	// which is what a byte comparison would have called equal.
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(link, []byte("milestone body\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := DivergentPaths(ctx, root, []string{"work/epics/E-0001/latest.md"})
	if err != nil {
		t.Fatalf("DivergentPaths: %v", err)
	}
	want := []Divergence{{Path: "work/epics/E-0001/latest.md", Kind: DivergenceModified}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DivergentPaths (-want +got):\n%s", diff)
	}
}

// TestDivergentPaths_UninspectablePathErrors pins the fail-loud
// direction for a path whose presence cannot be established at all — an
// unreadable parent directory, as distinct from a path that is simply
// not there. "Cannot tell" must not resolve to "absent from disk", which
// would let a carried path pass unexamined.
func TestDivergentPaths_UninspectablePathErrors(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	ctx := context.Background()
	root := seedCommittedFiles(t, map[string]string{"work/gaps/G-0001-a.md": "committed\n"})
	blocked := filepath.Join(root, "work", "sealed")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })

	_, err := DivergentPaths(ctx, root, []string{"work/sealed/inside.md"})
	if err == nil {
		t.Fatal("DivergentPaths over an uninspectable path: want an error, got nil")
	}
	if !strings.Contains(err.Error(), "work/sealed/inside.md") {
		t.Errorf("error does not name the path it could not inspect:\n%v", err)
	}
}

// TestDivergentPaths_SpansMoreThanOneArgvChunk pins the chunk loop.
// Paths ride on git's argument vector, so a large set is split across
// several invocations — and a split that dropped or misaligned a chunk
// would answer for the wrong path, or answer for fewer paths than were
// asked about, which is a silent wrong verdict rather than an error.
//
// The edited files sit deliberately in the first chunk, the last, and
// across the boundary.
func TestDivergentPaths_SpansMoreThanOneArgvChunk(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	total := argvChunk*2 + 3
	files := make(map[string]string, total)
	paths := make([]string, 0, total)
	for i := range total {
		p := fmt.Sprintf("work/gaps/G-%04d-x.md", i)
		files[p] = "committed\n"
		paths = append(paths, p)
	}
	root := seedCommittedFiles(t, files)

	edited := []int{0, argvChunk - 1, argvChunk, argvChunk + 1, total - 1}
	want := make([]Divergence, 0, len(edited))
	for _, i := range edited {
		writeAt(t, root, paths[i], "hand-edited\n")
		want = append(want, Divergence{Path: paths[i], Kind: DivergenceModified})
	}
	sort.Slice(want, func(a, b int) bool { return want[a].Path < want[b].Path })

	got, err := DivergentPaths(ctx, root, paths)
	if err != nil {
		t.Fatalf("DivergentPaths over %d paths: %v", total, err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DivergentPaths across chunk boundaries (-want +got):\n%s", diff)
	}
}
