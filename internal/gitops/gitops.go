// Package gitops is a thin wrapper around the git CLI for the operations
// `aiwf` needs: rename a tracked file, stage paths, and create a commit
// carrying structured trailers.
//
// We shell out to git rather than embedding go-git for two reasons:
// the host's git config (signing keys, hook installation, identity)
// is what users expect to apply, and our needs are small enough that
// a subprocess is the boring choice.
package gitops

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Trailer is a single key=value line emitted in the commit body.
// The key conventionally uses the `aiwf-*` prefix.
type Trailer struct {
	Key   string
	Value string
}

// CommitMessage assembles a subject, optional body, and trailers into
// the conventional commit-message form: subject, blank line, body
// (when non-empty) blank line, trailers (one per line). Exposed so
// callers (and tests) can construct messages without invoking git.
//
// The body is free-form prose. Whitespace is trimmed from both ends;
// an empty body produces no body section.
func CommitMessage(subject, body string, trailers []Trailer) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(subject, "\n"))
	if trimmed := strings.TrimSpace(body); trimmed != "" {
		b.WriteString("\n\n")
		b.WriteString(trimmed)
	}
	if len(trailers) > 0 {
		b.WriteString("\n\n")
		for i, tr := range trailers {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(tr.Key)
			b.WriteString(": ")
			b.WriteString(tr.Value)
		}
	}
	b.WriteString("\n")
	return b.String()
}

// Init initializes a git repository at workdir with `main` as the
// default branch. Used by tests; not invoked by `aiwf` verbs at
// runtime. The explicit `-b main` is what makes the test set
// env-independent — without it, `git init` honours the runner's
// `init.defaultBranch` config and tests that later `git checkout main`
// fail on runners that default to `master` (or anything else).
func Init(ctx context.Context, workdir string) error {
	return run(ctx, workdir, "init", "-q", "-b", "main")
}

// Mv runs `git mv` to relocate a tracked file or directory. from and to
// are paths relative to workdir.
//
// Test/porcelain-only (F7, docs/initiatives/verb-layer-cleanup.md): no
// `aiwf` verb calls this at runtime — verb.Apply/gitops.CommitVerbChange
// writes through CommitTree's plumbing path (commit-tree + update-ref),
// not `git mv`. Mv is one of the "forbidden APIs"
// internal/policies/verbs_validate_then_write.go's AST scan bans any
// exported internal/verb function from calling directly; it stays for
// tests that want a real `git mv` fixture.
func Mv(ctx context.Context, workdir, from, to string) error {
	return run(ctx, workdir, "mv", from, to)
}

// Add stages paths in workdir.
//
// Test/porcelain-only (F7): no `aiwf` verb calls this at runtime, for
// the same reason as Mv — see its doc comment.
func Add(ctx context.Context, workdir string, paths ...string) error {
	if len(paths) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, paths...)
	return run(ctx, workdir, args...)
}

// Commit creates a commit with the given subject line, optional body,
// and trailers. The commit's index is whatever has been staged with
// Add prior to this call; this is intentionally low-level — verbs
// control staging.
//
// Test/porcelain-only (F7): no `aiwf` verb calls this at runtime, for
// the same reason as Mv — see its doc comment.
func Commit(ctx context.Context, workdir, subject, body string, trailers []Trailer) error {
	msg := CommitMessage(subject, body, trailers)
	return run(ctx, workdir, "commit", "-m", msg)
}

// CommitAllowEmpty creates a commit even when the index has no staged
// changes.
//
// Test/porcelain-only (F7): no `aiwf` verb calls this at runtime today.
// `aiwf authorize`'s scope-only commits and `aiwf <verb> --audit-only`'s
// backfilled audit trail both route through
// gitops.CommitVerbChange/CommitTree — the plumbing path (commit-tree +
// update-ref), which fires no hooks and needs no staged changes to
// produce an empty-diff commit — not this porcelain `git commit
// --allow-empty` wrapper. It stays for tests that want a real
// allow-empty commit fixture.
func CommitAllowEmpty(ctx context.Context, workdir, subject, body string, trailers []Trailer) error {
	msg := CommitMessage(subject, body, trailers)
	return run(ctx, workdir, "commit", "--allow-empty", "-m", msg)
}

// StagedPaths returns every path currently staged in the index whose
// content differs from HEAD. Order is git's order; duplicates are not
// produced by `git diff --cached --name-only`. Used by verb.Apply to
// detect overlap between the user's pre-existing staged changes and a
// verb's about-to-write paths (G34 conflict guard / stash isolation).
//
// `-z` null-delimits the output so paths containing spaces, newlines,
// or other shell-hostile bytes round-trip safely. Empty output (clean
// index) returns a nil slice.
func StagedPaths(ctx context.Context, workdir string) ([]string, error) {
	out, err := output(ctx, workdir, "diff", "--cached", "--name-only", "-z")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	parts := strings.Split(strings.TrimRight(out, "\x00"), "\x00")
	paths := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// DirtyPaths returns the repo-relative paths that differ from HEAD in the
// working tree — unstaged and staged modifications, additions, and deletions
// of tracked files (via `git diff --name-only HEAD -z`), plus untracked,
// non-ignored files (via `git ls-files --others --exclude-standard -z`). It is
// the raw material the red/green diff-shape gate classifies (M-0276). Paths are
// '/'-separated and repo-relative; the result is sorted and deduplicated. A
// clean tree returns an empty slice.
//
// `-z` null-delimits both listings so paths containing spaces, newlines, or
// other shell-hostile bytes round-trip safely.
func DirtyPaths(ctx context.Context, workdir string) ([]string, error) {
	set := make(map[string]struct{})
	for _, args := range [][]string{
		{"diff", "--name-only", "HEAD", "-z"},
		{"ls-files", "--others", "--exclude-standard", "-z"},
	} {
		out, err := output(ctx, workdir, args...)
		if err != nil {
			return nil, err
		}
		for _, p := range strings.Split(strings.TrimRight(out, "\x00"), "\x00") {
			if p != "" {
				set[p] = struct{}{}
			}
		}
	}
	if len(set) == 0 {
		return nil, nil
	}
	paths := make([]string, 0, len(set))
	for p := range set {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}

// IsRepo reports whether workdir is inside a git working tree.
func IsRepo(ctx context.Context, workdir string) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = workdir
	return cmd.Run() == nil
}

// GitDir returns the absolute path to the git directory for workdir.
// Handles worktrees (where `.git` is a file, not a directory) and
// submodules transparently. Returns an error when workdir is not in a
// git repo.
func GitDir(ctx context.Context, workdir string) (string, error) {
	out, err := output(ctx, workdir, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// HooksDir returns the effective hooks directory for workdir. When
// `core.hooksPath` is set in the repo's git config, it is honored
// (relative paths resolve against workdir); otherwise the default
// `<commonGitDir>/hooks` is returned — the *shared* git dir, not the
// per-worktree one. From a linked worktree, git only fires hooks from
// the shared dir (per G-0136 / M-0133 / AC-2: writes to
// `.git/worktrees/<id>/hooks/` are inert).
//
// `git config --get` exits 1 when the key is unset, which `output`
// surfaces as an error; we treat any error as "fall back to default."
//
// Symlink resolution matters on macOS: `t.TempDir()` returns
// `/var/folders/...` but git resolves it to `/private/var/folders/...`.
// Without canonicalizing the relative-path branch, callers that
// compare the returned value against git-derived paths get
// long-up-and-back relative results that aren't useful for
// human-facing reports.
func HooksDir(ctx context.Context, workdir string) (string, error) {
	if out, err := output(ctx, workdir, "config", "--get", "core.hooksPath"); err == nil {
		if configured := strings.TrimSpace(out); configured != "" {
			if filepath.IsAbs(configured) {
				return configured, nil
			}
			canonical, evalErr := filepath.EvalSymlinks(workdir)
			if evalErr != nil {
				canonical = workdir
			}
			return filepath.Join(canonical, configured), nil
		}
	}
	commonDir, err := commonGitDir(ctx, workdir)
	if err != nil {
		return "", err
	}
	return filepath.Join(commonDir, "hooks"), nil
}

// RunPostCommitHook invokes the repo's post-commit hook, if one is
// installed and executable, mirroring what `git commit`'s porcelain
// layer does automatically after landing a commit. CommitTree-based
// commits (M-0186) bypass git's hook machinery entirely — commit-tree
// and update-ref are plumbing, and plumbing never fires hooks — so a
// caller that wants commit-tree-based commits to behave like a normal
// `git commit` for hook purposes (the STATUS.md regeneration hook,
// G-0112, or any hook a user has chained into post-commit.local) must
// invoke this explicitly after a successful commit.
//
// Matches git's own tolerance for this specific hook: per githooks(5),
// post-commit's exit status is informational only and never affects
// the outcome of the commit that already landed, so it is not
// surfaced here either. The only error this returns is a genuine
// environment problem resolving the hooks directory; a missing or
// non-executable hook file is a silent no-op, exactly as git itself
// treats it.
func RunPostCommitHook(ctx context.Context, workdir string) error {
	hooksDir, err := HooksDir(ctx, workdir)
	if err != nil {
		return fmt.Errorf("resolving hooks dir: %w", err)
	}
	hookPath := filepath.Join(hooksDir, "post-commit")
	info, statErr := os.Stat(hookPath)
	if statErr != nil || info.Mode()&0o111 == 0 {
		return nil
	}
	cmd := exec.CommandContext(ctx, hookPath)
	cmd.Dir = workdir
	_ = cmd.Run()
	return nil
}

// commonGitDir returns the absolute path to the *shared* git
// directory for workdir. In a main checkout this matches GitDir.
// In a linked worktree this is the main repo's `.git/`, not the
// worktree's `.git/worktrees/<id>/` — the directory git consults
// for hooks, packed refs, and other shared state.
//
// Uses `--path-format=absolute` (git 2.31+) to skip our own
// relative-path resolution; the modern flag is present on every
// supported toolchain in this repo's CI matrix.
func commonGitDir(ctx context.Context, workdir string) (string, error) {
	out, err := output(ctx, workdir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// MainCheckoutRoot returns the absolute path to the main working tree's
// root for the repo at workdir — the parent of the shared git dir. When
// workdir is a linked worktree, this is the main checkout, not the
// worktree, so aiwf writes shared per-repo artifacts (the health file)
// there and a single copy serves every worktree.
func MainCheckoutRoot(ctx context.Context, workdir string) (string, error) {
	commonDir, err := commonGitDir(ctx, workdir)
	if err != nil {
		return "", err
	}
	return filepath.Dir(commonDir), nil
}

// InWorktree reports whether workdir is inside a linked git worktree
// (vs. the main checkout). True when the per-worktree git dir
// differs from the shared common dir. Useful for operator-facing
// messages that want to flag "this action affects all worktrees of
// the repo" when run from a linked worktree.
//
// Returns false (no error) when workdir is the main checkout, and a
// non-nil error when git is not reachable from workdir.
func InWorktree(ctx context.Context, workdir string) (bool, error) {
	gitDir, err := GitDir(ctx, workdir)
	if err != nil {
		return false, err
	}
	commonDir, err := commonGitDir(ctx, workdir)
	if err != nil {
		return false, err
	}
	return gitDir != commonDir, nil
}

// HeadSubject returns the subject line of HEAD's commit. Used by tests
// to verify a commit landed; not used at runtime.
func HeadSubject(ctx context.Context, workdir string) (string, error) {
	out, err := output(ctx, workdir, "log", "-1", "--pretty=%s")
	return strings.TrimSpace(out), err
}

// HeadBody returns the body of HEAD's commit (the part between the
// subject and any trailers). Used by tests to verify a `--reason` text
// landed in the commit; not used at runtime.
func HeadBody(ctx context.Context, workdir string) (string, error) {
	out, err := output(ctx, workdir, "log", "-1", "--pretty=%b")
	return strings.TrimSpace(out), err
}

// HeadTrailers returns HEAD's trailer key/value pairs (via
// `git log -1 --pretty=%(trailers...)`). Tests use this to assert
// aiwf's structured trailers landed correctly.
func HeadTrailers(ctx context.Context, workdir string) ([]Trailer, error) {
	out, err := output(ctx, workdir, "log", "-1", "--pretty=%(trailers:only=true,unfold=true)")
	if err != nil {
		return nil, err
	}
	return ParseTrailers(out), nil
}

// ReadFromHEAD returns the bytes of relPath as it exists in the
// HEAD commit. Returns (nil, nil) when the path is not present at
// HEAD (e.g., the file is new in the working tree but not yet
// committed) so callers can branch on "exists at HEAD" cleanly
// without parsing stderr. Real git errors (no HEAD, repo-not-found,
// transport failure) are wrapped and returned.
//
// relPath must be repo-relative and forward-slashed; git's
// HEAD:<path> grammar requires that shape.
//
// Used by `aiwf edit-body` (M-060 bless mode) to compare working-
// copy bytes against HEAD bytes for the no-diff and frontmatter-
// changed refusal paths. Two-step (exists check then content read)
// avoids parsing localized git stderr text — the existence probe is
// the canonical pattern for this question.
func ReadFromHEAD(ctx context.Context, workdir, relPath string) ([]byte, error) {
	probe := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "--quiet", "HEAD:"+relPath)
	probe.Dir = workdir
	probe.Env = gitEnv()
	if err := probe.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			// exit 1 with --quiet means the path does not exist at HEAD.
			return nil, nil
		}
		return nil, fmt.Errorf("git rev-parse HEAD:%s: %w", relPath, err)
	}
	cmd := exec.CommandContext(ctx, "git", "show", "HEAD:"+relPath)
	cmd.Dir = workdir
	cmd.Env = gitEnv()
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git show HEAD:%s: %w\n%s", relPath, err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git show HEAD:%s: %w", relPath, err)
	}
	return out, nil
}

// ParseTrailers parses a `git log %(trailers:only=true,unfold=true)`
// block into structured Trailer values. The format is one trailer per
// line, `Key: value`, possibly followed by a trailing newline; empty
// lines and malformed lines (missing colon, or starting with colon)
// are skipped. This is the canonical exported home for trailer-line
// parsing across the module.
func ParseTrailers(out string) []Trailer {
	var trailers []Trailer
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.IndexByte(line, ':')
		if idx <= 0 {
			continue
		}
		trailers = append(trailers, Trailer{
			Key:   strings.TrimSpace(line[:idx]),
			Value: strings.TrimSpace(line[idx+1:]),
		})
	}
	return trailers
}

// run invokes git with the given args in workdir and returns any error,
// wrapped with the combined stdout/stderr for diagnostics.
func run(ctx context.Context, workdir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workdir
	cmd.Env = gitEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git %s: %w\n%s", args[0], err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git %s: %w", args[0], err)
	}
	return nil
}

// output runs git and returns stdout. Stderr is included in error wraps.
func output(ctx context.Context, workdir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workdir
	cmd.Env = gitEnv()
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", args[0], err, stderr.String())
	}
	return string(out), nil
}

// gitEnv returns an environment that satisfies git's identity
// requirement under tests where the user's git config might not be set
// (CI, ephemeral containers). Real users invoking `aiwf` from a normal
// shell already have these values; the variables here are silent
// defaults that don't override an existing config.
func gitEnv() []string {
	// Returning nil makes exec.Cmd inherit the parent's environment,
	// which is what we want for normal use. Tests should set
	// GIT_AUTHOR_NAME/EMAIL/GIT_COMMITTER_NAME/EMAIL via
	// t.Setenv before invoking.
	return nil
}
