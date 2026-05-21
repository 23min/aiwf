package gitops_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// TestBlobReader_PlainDirErrors confirms NewBlobReader rejects a
// non-repo directory before spawning the subprocess.
func TestBlobReader_PlainDirErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	br, err := gitops.NewBlobReader(ctx, t.TempDir())
	if err == nil {
		_ = br.Close()
		t.Fatal("NewBlobReader on plain dir returned nil err; want an error")
	}
}

// TestBlobReader_EmptyRootErrors confirms NewBlobReader rejects an
// empty root string before any subprocess spawn.
func TestBlobReader_EmptyRootErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	br, err := gitops.NewBlobReader(ctx, "")
	if err == nil {
		_ = br.Close()
		t.Fatal("NewBlobReader with empty root returned nil err; want an error")
	}
}

// TestBlobReader_ReadAtHEAD pins the basic read contract: opening a
// reader on a repo and reading HEAD:<path> returns the file's content.
func TestBlobReader_ReadAtHEAD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := initRepoWithCommits(t, []commitSpec{
		{files: map[string]string{"alpha.md": "alpha v1 content\n"}, subj: "c1"},
	})
	head := headSHA(t, ctx, root)

	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil {
		t.Fatalf("NewBlobReader: %v", err)
	}
	defer br.Close()

	got, err := br.Read(head, "alpha.md")
	if err != nil {
		t.Fatalf("Read(%s, alpha.md): %v", head, err)
	}
	want := "alpha v1 content\n"
	if string(got) != want {
		t.Errorf("Read returned %q, want %q", string(got), want)
	}
}

// TestBlobReader_ReadAtPriorCommit pins commit-pinned reads: after a
// modify, reading the OLD SHA returns the OLD content. This is the
// load-bearing case for fsm-history-consistent's status-at-parent
// reads.
func TestBlobReader_ReadAtPriorCommit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := initRepoWithCommits(t, []commitSpec{
		{files: map[string]string{"alpha.md": "v1\n"}, subj: "c1"},
		{files: map[string]string{"alpha.md": "v2\n"}, subj: "c2"},
	})
	headBeforeLast, err := runGitOutput(ctx, root, "rev-parse", "HEAD~1")
	if err != nil {
		t.Fatalf("rev-parse HEAD~1: %v", err)
	}

	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil {
		t.Fatalf("NewBlobReader: %v", err)
	}
	defer br.Close()

	got, err := br.Read(headBeforeLast, "alpha.md")
	if err != nil {
		t.Fatalf("Read(%s, alpha.md): %v", headBeforeLast, err)
	}
	if string(got) != "v1\n" {
		t.Errorf("Read returned %q, want v1\\n", string(got))
	}
}

// TestBlobReader_MissingPath pins the missing-blob signal: a path
// that doesn't exist at the commit returns ErrBlobMissing (not a
// subprocess crash, not a generic error).
func TestBlobReader_MissingPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := initRepoWithCommits(t, []commitSpec{
		{files: map[string]string{"alpha.md": "v1\n"}, subj: "c1"},
	})
	head := headSHA(t, ctx, root)

	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil {
		t.Fatalf("NewBlobReader: %v", err)
	}
	defer br.Close()

	_, err = br.Read(head, "nonexistent.md")
	if !errors.Is(err, gitops.ErrBlobMissing) {
		t.Errorf("Read(%s, nonexistent.md) err = %v, want ErrBlobMissing", head, err)
	}
}

// TestBlobReader_SequentialReadsReuseSubprocess proves the pump
// shape: two reads on the same BlobReader both succeed. (Subprocess
// crash between reads would surface as a protocol-state error on the
// second call.)
func TestBlobReader_SequentialReadsReuseSubprocess(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := initRepoWithCommits(t, []commitSpec{
		{
			files: map[string]string{"alpha.md": "alpha\n", "beta.md": "beta\n"},
			subj:  "c1",
		},
	})
	head := headSHA(t, ctx, root)

	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil {
		t.Fatalf("NewBlobReader: %v", err)
	}
	defer br.Close()

	got1, err := br.Read(head, "alpha.md")
	if err != nil {
		t.Fatalf("Read alpha.md: %v", err)
	}
	got2, err := br.Read(head, "beta.md")
	if err != nil {
		t.Fatalf("Read beta.md: %v", err)
	}
	if string(got1) != "alpha\n" {
		t.Errorf("alpha.md = %q, want alpha\\n", string(got1))
	}
	if string(got2) != "beta\n" {
		t.Errorf("beta.md = %q, want beta\\n", string(got2))
	}
}

// TestBlobReader_PostCloseReadFails confirms Read after Close
// returns an error (not a panic, not a silent partial result).
func TestBlobReader_PostCloseReadFails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := initRepoWithCommits(t, []commitSpec{
		{files: map[string]string{"alpha.md": "v1\n"}, subj: "c1"},
	})
	head := headSHA(t, ctx, root)

	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil {
		t.Fatalf("NewBlobReader: %v", err)
	}
	if err := br.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err = br.Read(head, "alpha.md")
	if err == nil {
		t.Error("Read after Close returned nil err; want an error")
	}
}

// TestBlobReader_CloseIdempotent confirms calling Close twice is a
// no-op (no panic, no double-reap, no error on the second call).
func TestBlobReader_CloseIdempotent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := initRepoWithCommits(t, []commitSpec{
		{files: map[string]string{"alpha.md": "v1\n"}, subj: "c1"},
	})

	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil {
		t.Fatalf("NewBlobReader: %v", err)
	}
	if err := br.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := br.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

// TestBlobReader_BinaryBlobContentPreserved confirms blob content
// passes through unchanged including embedded NUL bytes — the parser
// must not be line-oriented for the content payload.
func TestBlobReader_BinaryBlobContentPreserved(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	// Embedded NUL + newline-flavored binary-ish content.
	payload := []byte("first line\n\x00binary\x00content\nlast line\n")
	if err := os.WriteFile(filepath.Join(root, "bin.dat"), payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "bin.dat"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := gitops.Commit(ctx, root, "add bin.dat", "", nil); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	head := headSHA(t, ctx, root)

	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil {
		t.Fatalf("NewBlobReader: %v", err)
	}
	defer br.Close()

	got, err := br.Read(head, "bin.dat")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("blob content roundtrip mismatch:\n got %q\nwant %q", string(got), string(payload))
	}
}
