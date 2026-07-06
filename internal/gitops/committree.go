package gitops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PathWrite is a single repo-relative path and its full content, one entry
// in the set of writes CommitTree folds into a constructed commit.
type PathWrite struct {
	Path    string
	Content []byte
}

// CommitTree builds a commit from HEAD's tree plus writes, entirely
// against a throwaway index — HEAD's tree, the temp index, and the
// object database are the only things read or written. The live index
// and worktree are never touched, which is what makes this primitive
// safe to call while the caller (or a concurrent process) has its own
// staged or unstaged changes pending: nothing here can desync them.
//
// Steps: `git read-tree HEAD` into a temp index seeds it with the
// current tree; each write is hashed into a blob and added to that
// index via `update-index --add --cacheinfo`; `write-tree` produces the
// resulting tree; `commit-tree` builds the commit object against HEAD
// as its sole parent; `update-ref` moves HEAD (compare-and-swap against
// the parent SHA captured at the start, so a concurrent HEAD move is
// detected rather than silently overwritten). Returns the new commit's
// SHA.
//
// Reconciling the written paths into the live index (so `git status` is
// clean for them) is a separate concern — see the post-commit
// reconciliation this primitive is paired with.
//
// removes evicts paths from the tree seeded by read-tree — the
// mechanism a rename needs: read-tree carries the parent's tree forward
// unchanged, so a rename's old path stays present unless explicitly
// removed. A remove for a path absent from the parent tree is a no-op,
// not an error.
//
// A repo with no commits yet (unborn HEAD) is not an error: CommitTree
// builds a root commit instead, the same as `git commit` does on a
// fresh repository. This matters for verb.Apply's very first commit
// against a brand-new consumer repo.
func CommitTree(ctx context.Context, workdir string, removes []string, writes []PathWrite, subject, body string, trailers []Trailer) (string, error) {
	if !IsRepo(ctx, workdir) {
		return "", errors.New("resolving HEAD: not a git repository")
	}
	parent, err := output(ctx, workdir, "rev-parse", "HEAD")
	if err != nil {
		// Unborn HEAD (no commits yet) — build a root commit.
		return commitTreeFromParent(ctx, workdir, "", removes, writes, subject, body, trailers)
	}
	parent = strings.TrimSpace(parent)
	return commitTreeFromParent(ctx, workdir, parent, removes, writes, subject, body, trailers)
}

// commitTreeFromParent does the work of CommitTree against an explicit
// parent SHA rather than resolving HEAD itself. Split out so tests can
// drive the real construction-and-update-ref path with a deliberately
// stale parent — reproducing a concurrent-HEAD-move race deterministically,
// without an actual race — while CommitTree's public contract stays
// "build against current HEAD."
func commitTreeFromParent(ctx context.Context, workdir, parent string, removes []string, writes []PathWrite, subject, body string, trailers []Trailer) (string, error) {
	gitDir, err := GitDir(ctx, workdir)
	if err != nil {
		return "", fmt.Errorf("resolving git dir: %w", err)
	}

	// The temp index lives under the repo's own git dir (not system
	// /tmp) so it never crosses a filesystem boundary from the objects
	// it references — the same convention git itself uses for
	// `.git/index.lock`.
	tmpDir, err := os.MkdirTemp(gitDir, "aiwf-commit-tree-*")
	if err != nil {
		return "", fmt.Errorf("creating temp index dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	indexPath := filepath.Join(tmpDir, "index")

	// An empty parent means a root commit (no commits yet) — the temp
	// index starts empty (GIT_INDEX_FILE auto-creates it on first
	// write) rather than seeded from a parent tree that doesn't exist.
	if parent != "" {
		err = runIndexed(ctx, workdir, indexPath, "read-tree", parent)
		if err != nil {
			return "", fmt.Errorf("read-tree %s: %w", parent, err)
		}
	}

	for _, path := range removes {
		err = runIndexed(ctx, workdir, indexPath, "update-index", "--force-remove", path)
		if err != nil {
			return "", fmt.Errorf("removing %s: %w", path, err)
		}
	}

	var blobSHA string
	for _, w := range writes {
		blobSHA, err = hashObject(ctx, workdir, w.Content)
		if err != nil {
			return "", fmt.Errorf("hashing blob for %s: %w", w.Path, err)
		}
		cacheInfo := fmt.Sprintf("100644,%s,%s", blobSHA, w.Path)
		err = runIndexed(ctx, workdir, indexPath, "update-index", "--add", "--cacheinfo", cacheInfo)
		if err != nil {
			return "", fmt.Errorf("update-index %s: %w", w.Path, err)
		}
	}

	treeSHA, err := outputIndexed(ctx, workdir, indexPath, "write-tree")
	if err != nil {
		return "", fmt.Errorf("write-tree: %w", err) //coverage:ignore requires the temp index to reference a blob missing from the object database at write-tree time — every blob it references was just hash-object -w'd into that same database moments earlier
	}
	treeSHA = strings.TrimSpace(treeSHA)

	msg := CommitMessage(subject, body, trailers)
	commitTreeArgs := []string{"commit-tree", treeSHA}
	if parent != "" {
		commitTreeArgs = append(commitTreeArgs, "-p", parent)
	}
	sign, err := gpgSignEnabled(ctx, workdir)
	if err != nil {
		return "", err
	}
	if sign {
		commitTreeArgs = append(commitTreeArgs, "-S")
	}
	commitTreeArgs = append(commitTreeArgs, "-m", msg)
	commitSHA, err := output(ctx, workdir, commitTreeArgs...)
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w", err)
	}
	commitSHA = strings.TrimSpace(commitSHA)

	err = run(ctx, workdir, "update-ref", "HEAD", commitSHA, parent)
	if err != nil {
		return "", fmt.Errorf("update-ref HEAD: %w", err)
	}

	return commitSHA, nil
}

// gpgSignEnabled reports whether commit.gpgsign is set to true for
// workdir. `git commit-tree`, unlike `git commit`, does not consult
// this config on its own — the caller must pass -S explicitly for the
// two to behave the same way. An unset key (git config exits 1 with no
// output) means "not signing," matching commit.gpgsign's own default.
func gpgSignEnabled(ctx context.Context, workdir string) (bool, error) {
	out, err := output(ctx, workdir, "config", "--type=bool", "--get", "commit.gpgsign")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("git config --get commit.gpgsign: %w", err)
	}
	return strings.TrimSpace(out) == "true", nil
}

// hashObject writes content to the object database as a blob (without
// staging it anywhere) and returns its SHA. Does not touch the index.
func hashObject(ctx context.Context, workdir string, content []byte) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "hash-object", "-w", "--stdin")
	cmd.Dir = workdir
	cmd.Env = gitEnv()
	cmd.Stdin = bytes.NewReader(content)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git hash-object: %w\n%s", err, stderr.String())
	}
	return strings.TrimSpace(string(out)), nil
}

// indexedEnv builds the environment for a git index-manipulating
// command: whatever gitEnv() provides, plus GIT_INDEX_FILE pointed at
// indexPath. gitEnv() returning nil means "inherit the parent
// environment" (exec.Cmd's own convention) — appending directly to nil
// would instead mean "only this one variable," silently dropping the
// rest of the environment, so this materializes os.Environ() first
// whenever gitEnv() hasn't overridden it. Keeping this as the one
// composition point (rather than each call site appending
// os.Environ() directly) means a future gitEnv() that starts
// scrubbing/injecting variables is honored here too, not silently
// bypassed.
func indexedEnv(indexPath string) []string {
	env := gitEnv()
	if env == nil {
		env = os.Environ()
	}
	return append(env, "GIT_INDEX_FILE="+indexPath)
}

// runIndexed runs a git index-manipulating command with GIT_INDEX_FILE
// pointed at indexPath, so it reads and writes that file instead of the
// repo's live index.
func runIndexed(ctx context.Context, workdir, indexPath string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workdir
	cmd.Env = indexedEnv(indexPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w\n%s", args[0], err, strings.TrimSpace(string(out)))
	}
	return nil
}

// outputIndexed is runIndexed's output-returning counterpart.
func outputIndexed(ctx context.Context, workdir, indexPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workdir
	cmd.Env = indexedEnv(indexPath)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", args[0], err, stderr.String())
	}
	return string(out), nil
}
