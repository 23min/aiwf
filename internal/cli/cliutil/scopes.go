package cliutil

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
)

// LoadEntityScopes returns every scope ever opened on entity id, in
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
func LoadEntityScopes(ctx context.Context, root, id string) ([]*scope.Scope, error) {
	if !HasCommits(ctx, root) {
		return nil, nil
	}
	commits, err := readEntityScopeCommits(ctx, root, id)
	if err != nil {
		return nil, err
	}
	return ReplayScopes(commits), nil
}

// ReplayScopes replays the scope FSM over commits (oldest-first,
// already filtered to one entity's `aiwf-entity:` trailer) and returns
// every scope ever opened, in open-order, each carrying its current
// state. It is the pure, git-free core lifted out of LoadEntityScopes
// so render's single pass (M-0221) replays scopes from the shared HEAD
// walk through the SAME code path — no second copy of the FSM. The
// transition rules are documented on LoadEntityScopes.
//
// Callers pass a per-entity commit slice: LoadEntityScopes from its
// grep, render from the per-entity bucket of the one HEAD pass. The
// replay itself is identical either way.
func ReplayScopes(commits []CommitTrailers) []*scope.Scope {
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
	return scopes
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

// CommitTrailers is the per-commit shape the pure scope/opener replays
// consume: a commit's SHA plus its parsed trailer set. Exported so
// render's single pass (M-0221) can feed the same ReplayScopes /
// OpenersFrom helpers this package's grep-based loaders use — render
// converts each check.HeadCommit to this shape (SHA + Trailers) and
// replays through one code path rather than a fourth copy.
type CommitTrailers struct {
	SHA      string
	Trailers []gitops.Trailer
}

// readEntityScopeCommits returns commits whose `aiwf-entity:` trailer
// matches id, oldest-first. Each record carries the SHA plus every
// trailer (not just a fixed subset, because aiwf-scope / aiwf-scope-ends
// may appear on any verb's commit). Pre-aiwf commits whose `aiwf-entity:`
// trailer happens to match a future id are not in scope (the regex is
// anchored on `^aiwf-entity: <id>$`).
func readEntityScopeCommits(ctx context.Context, root, id string) ([]CommitTrailers, error) {
	const fieldSep = "\x1f"
	const recSep = "\x1e\n"
	args := []string{
		"log",
		"--reverse",
		"-E",
		// Width-tolerant per AC-2/AC-4 in M-081: pre-migration trailers
		// at narrow width still match a canonical-id query (and vice versa).
		"--grep", "^aiwf-entity: " + entity.IDGrepAlternation(id) + "$",
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
	var commits []CommitTrailers
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, fieldSep, 2)
		if len(parts) < 2 {
			continue
		}
		commits = append(commits, CommitTrailers{
			SHA:      strings.TrimSpace(parts[0]),
			Trailers: gitops.ParseTrailers(parts[1]),
		})
	}
	return commits, nil
}

// ResolveCurrentEntityID walks the aiwf-prior-entity chain forward
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
func ResolveCurrentEntityID(ctx context.Context, root, id string) (string, error) {
	if !HasCommits(ctx, root) {
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
		// Width-tolerant per AC-2/AC-4 in M-081.
		"--grep", "^aiwf-prior-entity: " + entity.IDGrepAlternation(priorID) + "$",
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
		// Compare canonicalized so a narrow trailer value is recognized
		// as "the same entity as priorID" when the lineage is mid-migration.
		if entityID != "" && entity.Canonicalize(entityID) != entity.Canonicalize(priorID) {
			return entityID, nil
		}
	}
	return "", nil
}

// AuthorizeOpeners returns a map from each authorize-opener commit's full
// SHA to its scope-entity id (canonicalized). It walks every commit whose
// trailers carry both `aiwf-verb: authorize` and `aiwf-scope: opened` once.
//
// This is the single source of truth for the repo-wide authorize-opener
// map: `aiwf show`'s scope table (foreign source (a)) and `aiwf history`'s
// scope chips both consume it — replacing the two byte-identical private
// copies (show.readAllAuthorizeOpeners and history.BuildScopeEntityMap) that
// predated it, and available for E-0054's render single-pass (M-0221) to
// reuse rather than add a third copy. Both read verbs now guard the call
// behind the loaded-event predicates (history.HasScopeData /
// history.HasAuthorizedBy), so the walk runs only when an entity's events
// actually reference a scope.
//
// An empty / pre-aiwf repo returns (empty, nil). A genuine `git log` failure
// on a repo with commits returns (nil, error); the history caller renders
// unresolved chips as "?" rather than blocking, while show propagates it.
func AuthorizeOpeners(ctx context.Context, root string) (map[string]string, error) {
	if !HasCommits(ctx, root) {
		return map[string]string{}, nil
	}
	commits, err := readAuthorizeOpenerCommits(ctx, root)
	if err != nil {
		return nil, err //coverage:ignore propagates the readAuthorizeOpenerCommits git error, itself unreachable after HasCommits on a valid repo
	}
	return OpenersFrom(commits), nil
}

// readAuthorizeOpenerCommits greps HEAD for commits carrying BOTH
// aiwf-verb: authorize and aiwf-scope: opened (the --all-match narrows
// the walk to opener commits), returning each with its full parsed
// trailer set. OpenersFrom re-applies the predicate, so it is equally
// correct over this pre-filtered slice or render's unfiltered whole-HEAD
// pass (M-0221).
func readAuthorizeOpenerCommits(ctx context.Context, root string) ([]CommitTrailers, error) {
	const fieldSep = "\x1f"
	const recSep = "\x1e"
	cmd := exec.CommandContext(ctx, "git", "log",
		"-E",
		"--grep", "^"+gitops.TrailerVerb+": authorize$",
		"--grep", "^"+gitops.TrailerScope+": opened$",
		"--all-match",
		"--pretty=tformat:%H"+fieldSep+"%(trailers:unfold=true)"+recSep)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		// Defensive: with fixed args and HasCommits already true, `git log`
		// fails only on a corrupt / partial clone between the two calls — a
		// tempdir-based test can't reproduce it. The show caller surfaces
		// this error; the history caller swallows it and renders "?" chips.
		return nil, fmt.Errorf("git log authorize-openers in %s: %w", root, err) //coverage:ignore
	}
	var commits []CommitTrailers
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, fieldSep, 2)
		if len(parts) < 2 { //coverage:ignore unreachable: the \x1f is a literal in the pretty-format, so every non-empty record splits into ≥2 parts
			continue
		}
		commits = append(commits, CommitTrailers{
			SHA:      strings.TrimSpace(parts[0]),
			Trailers: gitops.ParseTrailers(parts[1]),
		})
	}
	return commits, nil
}

// OpenersFrom maps each authorize-opener commit's SHA to its
// scope-entity id (canonicalized). It filters commits to those carrying
// aiwf-verb: authorize AND aiwf-scope: opened, so it is correct over
// either a pre-filtered opener slice (AuthorizeOpeners' grep) or an
// unfiltered whole-HEAD slice (render's single pass, M-0221). Blank
// SHAs and blank/absent entity ids are skipped — defends against
// hand-edited history and matches AuthorizeOpeners' prior behavior.
//
// Canonicalizing the trailer-stored entity id means callers comparing
// against tree-loaded ids never disambiguate widths (M-081 AC-2: the
// read side owns width tolerance).
func OpenersFrom(commits []CommitTrailers) map[string]string {
	result := map[string]string{}
	for _, c := range commits {
		idx := indexCommitTrailers(c.Trailers)
		if idx[gitops.TrailerVerb] != "authorize" || idx[gitops.TrailerScope] != "opened" {
			continue
		}
		ent := strings.TrimSpace(idx[gitops.TrailerEntity])
		if c.SHA == "" || ent == "" {
			continue
		}
		result[c.SHA] = entity.Canonicalize(ent)
	}
	return result
}
