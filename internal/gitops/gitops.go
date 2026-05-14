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
	"os/exec"
	"path/filepath"
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
func Mv(ctx context.Context, workdir, from, to string) error {
	return run(ctx, workdir, "mv", from, to)
}

// Add stages paths in workdir.
func Add(ctx context.Context, workdir string, paths ...string) error {
	if len(paths) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, paths...)
	return run(ctx, workdir, args...)
}

// Restore resets the index and worktree to HEAD for the given paths.
// Used by Apply to roll back partial verb mutations after a failure.
// Paths that don't exist at HEAD (brand-new files staged but never
// committed) produce a "pathspec did not match" warning that this
// function suppresses — the caller separately removes such files.
func Restore(ctx context.Context, workdir string, paths ...string) error {
	if len(paths) == 0 {
		return nil
	}
	args := append([]string{"restore", "--staged", "--worktree", "--"}, paths...)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workdir
	cmd.Env = gitEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		// `git restore` exits non-zero when ALL pathspecs miss; a
		// mixed run (some hit, some miss) exits zero with a warning.
		// We accept the all-miss case silently — it means the
		// rollback had nothing tracked to restore, which is correct
		// for a verb whose only ops were OpWrite of brand-new files.
		if strings.Contains(string(out), "did not match any file") {
			return nil
		}
		return fmt.Errorf("git restore: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Commit creates a commit with the given subject line, optional body,
// and trailers. The commit's index is whatever has been staged with
// Add prior to this call; this is intentionally low-level — verbs
// control staging. An empty body produces no body section.
func Commit(ctx context.Context, workdir, subject, body string, trailers []Trailer) error {
	msg := CommitMessage(subject, body, trailers)
	return run(ctx, workdir, "commit", "-m", msg)
}

// CommitAllowEmpty creates a commit even when the index has no staged
// changes. Used by verbs that record an event without touching files —
// `aiwf authorize` opens / pauses / resumes a scope by writing only
// trailers, and `aiwf <verb> --audit-only` (G24, plan step 5b) backfills
// an audit trail for state that was reached via a manual commit. Both
// are byte-identical to a normal commit except for the empty diff.
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

// StashStaged sets aside the user's currently-staged changes so the
// verb's commit boundary is exactly the verb's mutation plus any
// hook-added files (notably the pre-commit STATUS.md regeneration).
// Pair with StashPop after the commit lands.
//
// `git stash push --staged` (git ≥ 2.35) stashes only what's in the
// index; the worktree side of those paths is left alone. Untracked
// files and unstaged worktree edits are not affected. The message is
// stamped into the stash entry so a subsequent `git stash list`
// makes the source obvious if recovery becomes manual.
//
// G34 background: switched from `git commit -- <paths>` (--only) to
// stash because pre-commit hooks that `git add` extra files (like
// the aiwf STATUS.md hook) interact poorly with --only — git records
// the hook's addition in HEAD but resets the post-commit index to
// only the explicitly-named paths, leaving a phantom staged-deletion
// behind. Stash gives the verb a clean index to commit against
// without disturbing hook semantics.
func StashStaged(ctx context.Context, workdir, message string) error {
	return run(ctx, workdir, "stash", "push", "--staged", "--quiet", "-m", message)
}

// StashPop restores the most recently stashed entry into the index,
// reversing StashStaged. Errors propagate verbatim — a pop failure
// after the verb's commit landed is recoverable by hand
// (`git stash list` / `git stash pop`); the kernel does not silently
// drop the stash.
func StashPop(ctx context.Context, workdir string) error {
	return run(ctx, workdir, "stash", "pop", "--quiet")
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
// `<gitDir>/hooks` is returned. The result is always an absolute
// path with symlinks resolved, matching git's own canonicalization
// (via `--absolute-git-dir`).
//
// `git config --get` exits 1 when the key is unset, which `output`
// surfaces as an error; we treat any error as "fall back to
// default." If git is genuinely broken, the GitDir call below
// surfaces the same underlying issue with a clearer message.
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
	gitDir, err := GitDir(ctx, workdir)
	if err != nil {
		return "", err
	}
	return filepath.Join(gitDir, "hooks"), nil
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
	return parseTrailers(out), nil
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

func parseTrailers(out string) []Trailer {
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
