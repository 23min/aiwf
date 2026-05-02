package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/scope"
)

// loadEntityScopes returns every scope ever opened on entity id, in
// open-order (oldest first), with each scope's current state derived
// from the entity's commit history. Empty / no-commit repos return
// (nil, nil).
//
// The walker scans every commit whose `aiwf-entity:` trailer matches
// id (the same filter `aiwf history` uses) once, applying transitions
// in commit order:
//
//   - aiwf-verb=authorize, aiwf-scope=opened → append a new scope
//     starting in state active.
//   - aiwf-verb=authorize, aiwf-scope=paused → flip the most-recently-
//     opened still-active scope to paused.
//   - aiwf-verb=authorize, aiwf-scope=resumed → flip the most-recently-
//     paused scope to active.
//   - aiwf-scope-ends: <auth-sha> on any commit → mark the matching
//     scope ended (terminal). Auto-end fires when a terminal-promote
//     of the scope-entity carries this trailer (added by step 6); for
//     the initial step 5 cut, hand-crafted fixtures use the same path.
//
// The "most recently X" rule mirrors the verb's pause/resume picker:
// scopes are appended in chronological open-order, so a backwards
// walk finds the latest match. Multiple parallel scopes on the same
// entity are supported; transitions land on the freshest matching
// state.
func loadEntityScopes(ctx context.Context, root, id string) ([]*scope.Scope, error) {
	if !hasCommits(ctx, root) {
		return nil, nil
	}
	commits, err := readEntityScopeCommits(ctx, root, id)
	if err != nil {
		return nil, err
	}
	var scopes []*scope.Scope
	byAuth := map[string]*scope.Scope{}
	for _, c := range commits {
		idx := indexCommitTrailers(c.Trailers)
		switch {
		case idx[gitops.TrailerVerb] == "authorize" && idx[gitops.TrailerScope] == "opened":
			s := &scope.Scope{
				AuthSHA:   c.SHA,
				Entity:    idx[gitops.TrailerEntity],
				Agent:     idx[gitops.TrailerTo],
				Principal: idx[gitops.TrailerActor],
				State:     scope.StateActive,
				Events: []scope.Event{
					{SHA: c.SHA, State: scope.StateActive, Reason: idx[gitops.TrailerReason]},
				},
			}
			scopes = append(scopes, s)
			byAuth[c.SHA] = s
		case idx[gitops.TrailerVerb] == "authorize" && idx[gitops.TrailerScope] == "paused":
			if s := mostRecent(scopes, scope.StateActive); s != nil {
				s.State = scope.StatePaused
				s.Events = append(s.Events, scope.Event{SHA: c.SHA, State: scope.StatePaused, Reason: idx[gitops.TrailerReason]})
			}
		case idx[gitops.TrailerVerb] == "authorize" && idx[gitops.TrailerScope] == "resumed":
			if s := mostRecent(scopes, scope.StatePaused); s != nil {
				s.State = scope.StateActive
				s.Events = append(s.Events, scope.Event{SHA: c.SHA, State: scope.StateActive, Reason: idx[gitops.TrailerReason]})
			}
		}
		// aiwf-scope-ends can repeat (one trailer per ended scope);
		// walk every trailer rather than the indexed map.
		for _, tr := range c.Trailers {
			if tr.Key != gitops.TrailerScopeEnds {
				continue
			}
			s := byAuth[tr.Value]
			if s == nil || s.State == scope.StateEnded {
				continue
			}
			s.State = scope.StateEnded
			s.Events = append(s.Events, scope.Event{SHA: c.SHA, State: scope.StateEnded, Reason: idx[gitops.TrailerReason]})
		}
	}
	return scopes, nil
}

// mostRecent returns the most-recently-opened scope whose current
// state matches `state`, or nil. Scopes are in open-order; we walk
// backward.
func mostRecent(scopes []*scope.Scope, state scope.State) *scope.Scope {
	for i := len(scopes) - 1; i >= 0; i-- {
		if scopes[i].State == state {
			return scopes[i]
		}
	}
	return nil
}

// indexCommitTrailers builds a key→value lookup for a commit's
// trailers. When a key repeats (notably aiwf-scope-ends), the last
// occurrence wins; the only loader site that cares about every value
// of a repeating key is the scope-ends loop, which iterates the slice
// directly.
func indexCommitTrailers(trailers []gitops.Trailer) map[string]string {
	out := make(map[string]string, len(trailers))
	for _, tr := range trailers {
		out[tr.Key] = tr.Value
	}
	return out
}

// commitTrailers is the per-commit shape consumed by loadEntityScopes:
// the SHA plus the parsed trailer set.
type commitTrailers struct {
	SHA      string
	Trailers []gitops.Trailer
}

// readEntityScopeCommits returns commits whose `aiwf-entity:` trailer
// matches id, oldest-first. Each record carries the SHA plus every
// trailer (not just a fixed subset, because aiwf-scope / aiwf-scope-ends
// may appear on any verb's commit). Pre-aiwf commits whose `aiwf-entity:`
// trailer happens to match a future id are not in scope (the regex is
// anchored on `^aiwf-entity: <id>$`).
func readEntityScopeCommits(ctx context.Context, root, id string) ([]commitTrailers, error) {
	const fieldSep = "\x1f"
	const recSep = "\x1e\n"
	args := []string{
		"log",
		"--reverse",
		"-E",
		"--grep", "^aiwf-entity: " + regexp.QuoteMeta(id) + "$",
		"--pretty=tformat:%H" + fieldSep + "%(trailers:only=true,unfold=true)\x1e",
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git log: %w", err)
	}
	var commits []commitTrailers
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, fieldSep, 2)
		if len(parts) < 2 {
			continue
		}
		commits = append(commits, commitTrailers{
			SHA:      strings.TrimSpace(parts[0]),
			Trailers: parseTrailerLines(parts[1]),
		})
	}
	return commits, nil
}

// resolveCurrentEntityID walks the aiwf-prior-entity chain forward
// from id to its current id. When id has never been reallocated,
// returns id unchanged. When the entity was renumbered, follows the
// rename chain — each `aiwf reallocate` commit carries
// aiwf-prior-entity: <old> and aiwf-entity: <new> — until no further
// commit names the current id as `prior-entity`.
//
// Used by the I2.5 allow-rule's scope-entity resolution: an
// authorize commit's aiwf-entity trailer is the id at the time of
// authorization. After a reallocate, the historical commit stays
// byte-identical (its SHA remains valid as aiwf-authorized-by:),
// but the live entity now lives under a new id. Forward-walking the
// prior-entity chain produces the current id so tree.Reaches can
// run against the live tree.
//
// Cycle guard: each id may appear at most once. A cycle (which
// would indicate corrupted history) returns the chain's current
// position rather than looping.
func resolveCurrentEntityID(ctx context.Context, root, id string) (string, error) {
	if !hasCommits(ctx, root) {
		return id, nil
	}
	visited := map[string]bool{id: true}
	current := id
	for {
		next, err := readPriorEntityNewID(ctx, root, current)
		if err != nil {
			return current, err
		}
		if next == "" || visited[next] {
			return current, nil
		}
		visited[next] = true
		current = next
	}
}

// readPriorEntityNewID returns the aiwf-entity value of the most
// recent commit whose aiwf-prior-entity trailer matches priorID.
// Returns the empty string when no commit names priorID as prior.
//
// `git log` is queried newest-first so the latest reallocate wins
// when (defensively) multiple commits name the same prior id.
func readPriorEntityNewID(ctx context.Context, root, priorID string) (string, error) {
	const sep = "\x1f"
	const recSep = "\x1e\n"
	args := []string{
		"log",
		"-E",
		"--grep", "^aiwf-prior-entity: " + regexp.QuoteMeta(priorID) + "$",
		"--pretty=tformat:%(trailers:key=aiwf-entity,valueonly=true,unfold=true)" + sep + "\x1e",
	}
	// priorID is regexp-quoted into the --grep argument; exec.Command
	// uses execve (no shell), so there is no injection vector beyond
	// what QuoteMeta neutralizes. Same pattern as readEntityScopeCommits.
	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // regexp.QuoteMeta + execve (no shell) — no injection vector
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git log: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git log: %w", err)
	}
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, sep, 2)
		entityID := strings.TrimSpace(parts[0])
		if entityID != "" && entityID != priorID {
			return entityID, nil
		}
	}
	return "", nil
}

// parseTrailerLines parses a `git log %(trailers:only=true,unfold=true)`
// block into structured Trailer values. The format is one trailer per
// line, `Key: value`, possibly followed by a trailing newline; empty
// lines and malformed lines are skipped.
func parseTrailerLines(s string) []gitops.Trailer {
	var trailers []gitops.Trailer
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.IndexByte(line, ':')
		if idx <= 0 {
			continue
		}
		trailers = append(trailers, gitops.Trailer{
			Key:   strings.TrimSpace(line[:idx]),
			Value: strings.TrimSpace(line[idx+1:]),
		})
	}
	return trailers
}
