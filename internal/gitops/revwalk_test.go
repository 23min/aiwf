package gitops_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/gitops"
)

// TestBulkRevwalk_PlainDir confirms the no-op path: a directory that
// is not a git repo emits no records and returns no error.
func TestBulkRevwalk_PlainDir(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir() // plain dir, no git init

	var got []gitops.CommitRecord
	err := gitops.BulkRevwalk(ctx, root, func(rec gitops.CommitRecord) error {
		got = append(got, rec)
		return nil
	})
	if err != nil {
		t.Fatalf("BulkRevwalk on plain dir returned err=%v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("BulkRevwalk on plain dir emitted %d records, want 0", len(got))
	}
}

// TestBulkRevwalk_RepoNoCommits confirms the no-op path for an
// init'd-but-empty repo: HEAD is unborn, no commits, no callbacks.
func TestBulkRevwalk_RepoNoCommits(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("Init: %v", err)
	}

	var got []gitops.CommitRecord
	err := gitops.BulkRevwalk(ctx, root, func(rec gitops.CommitRecord) error {
		got = append(got, rec)
		return nil
	})
	if err != nil {
		t.Fatalf("BulkRevwalk on init'd repo returned err=%v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("BulkRevwalk on init'd repo emitted %d records, want 0", len(got))
	}
}

// TestBulkRevwalk_SingleRootCommit pins the root-commit shape: one
// record with empty Parents, one PathTouch with Status="A", and
// trailers parsed from the commit message.
func TestBulkRevwalk_SingleRootCommit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := initRepoWithCommits(t, []commitSpec{
		{
			files: map[string]string{"alpha.md": "alpha v1\n"},
			subj:  "add alpha",
			trailers: []gitops.Trailer{
				{Key: "aiwf-verb", Value: "add"},
				{Key: "aiwf-entity", Value: "E-0001"},
				{Key: "aiwf-actor", Value: "human/peter"},
			},
		},
	})

	got := collectRecords(t, ctx, root)

	if len(got) != 1 {
		t.Fatalf("BulkRevwalk emitted %d records, want 1", len(got))
	}
	rec := got[0]
	if rec.Commit == "" {
		t.Errorf("Commit empty, want a full SHA")
	}
	if len(rec.Parents) != 0 {
		t.Errorf("Parents = %v, want empty (root commit)", rec.Parents)
	}
	// M-0216 AC-2: --raw carries blob object ids. An add has an
	// all-zero pre-image and a post-image equal to the committed blob.
	wantPaths := []gitops.PathTouch{{
		Status:  "A",
		Path:    "alpha.md",
		PreSHA:  "0000000000000000000000000000000000000000",
		PostSHA: blobID(t, ctx, root, rec.Commit, "alpha.md"),
	}}
	if diff := cmp.Diff(wantPaths, rec.Paths); diff != "" {
		t.Errorf("Paths mismatch (-want +got):\n%s", diff)
	}
	if gitops.BlobAllZero(rec.Paths[0].PostSHA) {
		t.Errorf("PostSHA = %q, want a real blob id", rec.Paths[0].PostSHA)
	}
	wantTrailers := map[string]string{
		"aiwf-verb":   "add",
		"aiwf-entity": "E-0001",
		"aiwf-actor":  "human/peter",
	}
	if diff := cmp.Diff(wantTrailers, rec.Trailers); diff != "" {
		t.Errorf("Trailers mismatch (-want +got):\n%s", diff)
	}
}

// TestBulkRevwalk_TwoLinearCommits pins the linear parent chain: the
// second commit's Parents contains the first commit's SHA, and a
// path-modify emits Status="M".
func TestBulkRevwalk_TwoLinearCommits(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := initRepoWithCommits(t, []commitSpec{
		{
			files: map[string]string{"alpha.md": "alpha v1\n"},
			subj:  "add alpha",
			trailers: []gitops.Trailer{
				{Key: "aiwf-verb", Value: "add"},
				{Key: "aiwf-entity", Value: "E-0001"},
			},
		},
		{
			files: map[string]string{"alpha.md": "alpha v2\n"},
			subj:  "modify alpha",
			trailers: []gitops.Trailer{
				{Key: "aiwf-verb", Value: "edit-body"},
				{Key: "aiwf-entity", Value: "E-0001"},
			},
		},
	})

	got := collectRecords(t, ctx, root)
	if len(got) != 2 {
		t.Fatalf("BulkRevwalk emitted %d records, want 2", len(got))
	}

	// Walk order: git log --all defaults to reverse-chronological
	// (newest first). The newest commit is the modify, the oldest is
	// the root add. Sort by Paths[0].Status so this assertion does
	// not depend on git's walk order:
	byStatus := map[string]gitops.CommitRecord{}
	for _, rec := range got {
		if len(rec.Paths) == 0 {
			t.Fatalf("record %+v has no Paths", rec)
		}
		byStatus[rec.Paths[0].Status] = rec
	}
	add, ok := byStatus["A"]
	if !ok {
		t.Fatalf("no record with Paths[0].Status=A; got statuses=%v", statusKeys(byStatus))
	}
	mod, ok := byStatus["M"]
	if !ok {
		t.Fatalf("no record with Paths[0].Status=M; got statuses=%v", statusKeys(byStatus))
	}

	if len(add.Parents) != 0 {
		t.Errorf("root commit Parents = %v, want empty", add.Parents)
	}
	if len(mod.Parents) != 1 {
		t.Fatalf("modify commit Parents = %v, want one entry", mod.Parents)
	}
	if mod.Parents[0] != add.Commit {
		t.Errorf("modify commit Parents[0] = %s, want %s", mod.Parents[0], add.Commit)
	}
}

// TestBulkRevwalk_Rename pins the rename shape: the rename commit's
// PathTouch carries Status="R" with SrcPath set to the pre-rename path.
func TestBulkRevwalk_Rename(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "alpha.md"), []byte("alpha v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "alpha.md"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := gitops.Commit(ctx, root, "add alpha", "", []gitops.Trailer{
		{Key: "aiwf-verb", Value: "add"},
	}); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := gitops.Mv(ctx, root, "alpha.md", "beta.md"); err != nil {
		t.Fatalf("Mv: %v", err)
	}
	if err := gitops.Commit(ctx, root, "rename alpha to beta", "", []gitops.Trailer{
		{Key: "aiwf-verb", Value: "rename"},
	}); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got := collectRecords(t, ctx, root)
	if len(got) != 2 {
		t.Fatalf("BulkRevwalk emitted %d records, want 2", len(got))
	}

	var renameRec *gitops.CommitRecord
	for i := range got {
		for _, p := range got[i].Paths {
			if p.Status == "R" {
				rec := got[i]
				renameRec = &rec
			}
		}
	}
	if renameRec == nil {
		t.Fatalf("no record with Status=R; got=%+v", got)
	}
	if len(renameRec.Paths) != 1 {
		t.Fatalf("rename record Paths len = %d, want 1", len(renameRec.Paths))
	}
	got0 := renameRec.Paths[0]
	if got0.Status != "R" || got0.SrcPath != "alpha.md" || got0.Path != "beta.md" {
		t.Errorf("rename PathTouch = %+v, want {R alpha.md beta.md}", got0)
	}
}

// TestBulkRevwalk_MergeCommit pins the merge-commit shape: Parents
// contains both incoming commit SHAs. G-0372 Fix 1 drops the `-m` flag
// from the underlying `git log`, so --raw's diff output is suppressed
// for merge commits (git's default without -m/-c/--cc) — the merge
// commit's record still appears (via the tformat pretty-print, which
// isn't diff-dependent) with Parents populated, but Paths is empty.
// This is safe because every current consumer of BulkRevwalk's output
// discards merge-commit observations unconditionally (D-0010).
func TestBulkRevwalk_MergeCommit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Root commit on main.
	writeAndCommit(t, ctx, root, "alpha.md", "alpha v1\n", "add alpha", []gitops.Trailer{
		{Key: "aiwf-verb", Value: "add"},
	})
	mainHead := headSHA(t, ctx, root)

	// Create a feature branch off main and add beta.md.
	if err := runGit(ctx, root, "checkout", "-b", "feature"); err != nil {
		t.Fatalf("checkout -b feature: %v", err)
	}
	writeAndCommit(t, ctx, root, "beta.md", "beta v1\n", "add beta on feature", []gitops.Trailer{
		{Key: "aiwf-verb", Value: "add"},
	})
	featHead := headSHA(t, ctx, root)

	// Back to main, add gamma.md.
	if err := runGit(ctx, root, "checkout", "main"); err != nil {
		// Fallback: some git versions default to "master"; try that.
		if err2 := runGit(ctx, root, "checkout", "master"); err2 != nil {
			t.Fatalf("checkout main/master: %v / %v", err, err2)
		}
	}
	writeAndCommit(t, ctx, root, "gamma.md", "gamma v1\n", "add gamma on main", []gitops.Trailer{
		{Key: "aiwf-verb", Value: "add"},
	})
	mainAfterHead := headSHA(t, ctx, root)
	if mainAfterHead == mainHead {
		t.Fatalf("main HEAD didn't advance: %s", mainAfterHead)
	}

	// Merge feature into main, forcing a merge commit with --no-ff.
	if err := runGit(ctx, root, "merge", "--no-ff", "-m", "merge feature\n\naiwf-verb: merge", "feature"); err != nil {
		t.Fatalf("merge feature: %v", err)
	}
	mergeHead := headSHA(t, ctx, root)

	got := collectRecords(t, ctx, root)

	var mergeRec *gitops.CommitRecord
	for i := range got {
		if got[i].Commit == mergeHead {
			rec := got[i]
			mergeRec = &rec
			break
		}
	}
	if mergeRec == nil {
		t.Fatalf("no record for merge commit %s; got commits=%v", mergeHead, commits(got))
	}

	// Merge commit's parents: mainAfterHead (first) and featHead.
	wantParents := []string{mainAfterHead, featHead}
	sortedGot := append([]string(nil), mergeRec.Parents...)
	sortedWant := append([]string(nil), wantParents...)
	sort.Strings(sortedGot)
	sort.Strings(sortedWant)
	if diff := cmp.Diff(sortedWant, sortedGot); diff != "" {
		t.Errorf("merge Parents mismatch (-want +got):\n%s", diff)
	}

	// G-0372 Fix 1: without -m, `git log --raw` emits no diff records
	// for a merge commit — Paths must be empty even though this merge
	// integrated a branch that added beta.md.
	if len(mergeRec.Paths) != 0 {
		t.Errorf("merge record Paths = %+v, want empty (no -m fan-out)", mergeRec.Paths)
	}
}

// TestBulkRevwalk_CallbackErrorHalts confirms a non-nil callback error
// halts the walk and is propagated verbatim — the consumer can
// short-circuit.
func TestBulkRevwalk_CallbackErrorHalts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := initRepoWithCommits(t, []commitSpec{
		{files: map[string]string{"a.md": "1\n"}, subj: "c1"},
		{files: map[string]string{"a.md": "2\n"}, subj: "c2"},
		{files: map[string]string{"a.md": "3\n"}, subj: "c3"},
	})

	sentinel := errors.New("stop here")
	calls := 0
	err := gitops.BulkRevwalk(ctx, root, func(rec gitops.CommitRecord) error {
		calls++
		if calls == 1 {
			return sentinel
		}
		return nil
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want sentinel %v", err, sentinel)
	}
	if calls != 1 {
		t.Errorf("callback invoked %d times after error, want 1", calls)
	}
}

// TestBlobAllZero covers both object formats git can stamp an id in.
// A repository created with `--object-format=sha256` writes the absent
// side of an add or a delete as 64 zeros, so a predicate that knows
// only the 40-zero SHA-1 spelling reads every one of them as a real
// blob id and sends it to be looked up.
//
// The two widths are the enumerated set: `git init --object-format`
// takes sha1 or sha256 and nothing else.
func TestBlobAllZero(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		id   string
		want bool
	}{
		{"empty string", "", true},
		{"sha1 all-zero", strings.Repeat("0", 40), true},
		{"sha256 all-zero", strings.Repeat("0", 64), true},
		{"real sha1 id", "bb41accdf2666775279fec9994f070720b2e22e3", false},
		{"real sha256 id", "96c18f0297e38d01f4b2dacddea4259aea6b2961eb0822bd2c0c3f6029030045", false},
		{"sha1 id ending in zeros", "bb41accdf2666775279fec9994f0707200000000", false},
		{"a lone zero digit", "0", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := gitops.BlobAllZero(c.id); got != c.want {
				t.Errorf("BlobAllZero(%q) = %v, want %v", c.id, got, c.want)
			}
		})
	}
}

// TestBulkRevwalk_AbsentSideIsAllZero_BothObjectFormats derives the
// two all-zero spellings from git itself rather than from the same
// constants the predicate under test uses, so a repository whose
// object format the predicate does not know fails here rather than
// silently reporting an absent blob as a real one.
func TestBulkRevwalk_AbsentSideIsAllZero_BothObjectFormats(t *testing.T) {
	t.Parallel()
	for _, format := range []string{"sha1", "sha256"} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			root := t.TempDir()
			mustGit := func(args ...string) {
				t.Helper()
				if err := runGit(ctx, root, args...); err != nil {
					t.Fatalf("git %v: %v", args, err)
				}
			}
			mustGit("init", "-q", "-b", "main", "--object-format="+format, ".")
			mustGit("config", "user.email", "test@example.com")
			mustGit("config", "user.name", "aiwf-test")
			if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("hello\n"), 0o644); err != nil {
				t.Fatalf("write a.md: %v", err)
			}
			mustGit("add", "-A")
			mustGit("commit", "-q", "-m", "add a.md")
			mustGit("rm", "-q", "a.md")
			mustGit("commit", "-q", "-m", "delete a.md")

			var sides []string
			err := gitops.BulkRevwalk(ctx, root, func(rec gitops.CommitRecord) error {
				for _, p := range rec.Paths {
					switch p.Status {
					case "A":
						sides = append(sides, p.PreSHA) // add: pre-image absent
					case "D":
						sides = append(sides, p.PostSHA) // delete: post-image absent
					}
				}
				return nil
			})
			if err != nil {
				t.Fatalf("BulkRevwalk: %v", err)
			}
			if len(sides) != 2 {
				t.Fatalf("collected %d absent-side ids, want 2 (one add, one delete): %v", len(sides), sides)
			}
			for _, id := range sides {
				if !gitops.BlobAllZero(id) {
					t.Errorf("BlobAllZero(%q) = false for the absent side of an %s-format add/delete; it would be sent to the object store as a real id", id, format)
				}
			}
		})
	}
}

// --- test helpers ---

type commitSpec struct {
	files    map[string]string
	subj     string
	trailers []gitops.Trailer
}

func initRepoWithCommits(t *testing.T, specs []commitSpec) string {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	for i, s := range specs {
		for path, content := range s.files {
			if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o644); err != nil {
				t.Fatalf("write %s: %v", path, err)
			}
			if err := gitops.Add(ctx, root, path); err != nil {
				t.Fatalf("Add %s: %v", path, err)
			}
		}
		if err := gitops.Commit(ctx, root, s.subj, "", s.trailers); err != nil {
			t.Fatalf("Commit #%d (%s): %v", i, s.subj, err)
		}
	}
	return root
}

func writeAndCommit(t *testing.T, ctx context.Context, root, path, content, subj string, trailers []gitops.Trailer) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := gitops.Add(ctx, root, path); err != nil {
		t.Fatalf("Add %s: %v", path, err)
	}
	if err := gitops.Commit(ctx, root, subj, "", trailers); err != nil {
		t.Fatalf("Commit %s: %v", subj, err)
	}
}

func collectRecords(t *testing.T, ctx context.Context, root string) []gitops.CommitRecord {
	t.Helper()
	var got []gitops.CommitRecord
	err := gitops.BulkRevwalk(ctx, root, func(rec gitops.CommitRecord) error {
		got = append(got, rec)
		return nil
	})
	if err != nil {
		t.Fatalf("BulkRevwalk: %v", err)
	}
	return got
}

func statusKeys(m map[string]gitops.CommitRecord) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func commits(recs []gitops.CommitRecord) []string {
	out := make([]string, 0, len(recs))
	for _, r := range recs {
		out = append(out, r.Commit)
	}
	return out
}

func headSHA(t *testing.T, ctx context.Context, root string) string {
	t.Helper()
	out, err := runGitOutput(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return out
}

// blobID returns the full 40-char blob object id of path at commit
// via `git rev-parse <commit>:<path>` — the independent oracle the
// blob-id-population assertion checks BulkRevwalk's PostSHA against.
func blobID(t *testing.T, ctx context.Context, root, commit, path string) string {
	t.Helper()
	out, err := runGitOutput(ctx, root, "rev-parse", commit+":"+path)
	if err != nil {
		t.Fatalf("rev-parse %s:%s: %v", commit, path, err)
	}
	return out
}

func runGit(ctx context.Context, root string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(strings.TrimSpace(string(out)) + ": " + err.Error())
	}
	return nil
}

func runGitOutput(ctx context.Context, root string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
