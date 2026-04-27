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
	"strings"
)

// Trailer is a single key=value line emitted in the commit body.
// The key conventionally uses the `aiwf-*` prefix.
type Trailer struct {
	Key   string
	Value string
}

// CommitMessage assembles a subject + trailers into the conventional
// commit-message form: subject, blank line, trailers (one per line).
// Exposed so callers (and tests) can construct messages without
// invoking git.
func CommitMessage(subject string, trailers []Trailer) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(subject, "\n"))
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

// Init initializes a git repository at workdir. Used by tests; not
// invoked by `aiwf` verbs at runtime.
func Init(ctx context.Context, workdir string) error {
	return run(ctx, workdir, "init", "-q")
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

// Commit creates a commit with the given subject line and trailers.
// The commit's index is whatever has been staged with Add prior to this
// call; this is intentionally low-level — verbs control staging.
func Commit(ctx context.Context, workdir, subject string, trailers []Trailer) error {
	msg := CommitMessage(subject, trailers)
	return run(ctx, workdir, "commit", "-m", msg)
}

// IsRepo reports whether workdir is inside a git working tree.
func IsRepo(ctx context.Context, workdir string) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = workdir
	return cmd.Run() == nil
}

// HeadSubject returns the subject line of HEAD's commit. Used by tests
// to verify a commit landed; not used at runtime.
func HeadSubject(ctx context.Context, workdir string) (string, error) {
	out, err := output(ctx, workdir, "log", "-1", "--pretty=%s")
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
