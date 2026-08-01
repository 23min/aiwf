package gitops

import (
	"context"
	"os"
	"path/filepath"
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
