// Package entityview holds the read-side projection helpers shared by
// aiwf's read verbs (show, history, render, check, status): parsing an
// entity's lifecycle out of git log trailers, assembling the scope table
// that describes which authorization grants touched it, and reading an
// entity file's body prose. The package is free of internal/cli
// dependencies (no Cobra, no cliutil) so it can be unit-tested and reused
// without pulling in a verb's command-wiring package.
package entityview

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// HistoryEvent is one line of `aiwf history`. The JSON representation
// is the structured form callers consume.
//
// Body carries the commit's free-form body — typically the human's
// `--reason` for a status transition, or empty when the verb wasn't
// invoked with one. Trailers are stripped before storage so Body is
// pure prose.
//
// To is the target status of a `promote` event, extracted from the
// `aiwf-to:` trailer (added in I2). Empty for non-promote events and
// for pre-I2 promote commits that were written before the trailer
// schema landed; the renderer shows a dash for those rows.
//
// Force is the reason value of an `aiwf-force:` trailer. Empty for
// non-forced transitions; non-empty marks the event as having
// bypassed the FSM's transition-legality rule.
//
// AuditOnly is the reason value of an `aiwf-audit-only:` trailer
// (I2.5 G24 recovery mode). Empty for normal verb commits; non-empty
// marks the event as a backfilled audit trail for state that was
// reached via a manual commit. Renders as a `[audit-only: <reason>]`
// chip in text output, mirroring the `[forced: ...]` rendering.
//
// Principal, OnBehalfOf, AuthorizedBy, Scope, ScopeEnds, Reason
// expose the I2.5 provenance trailer set. Principal is the human on
// whose authority the actor ran (always `human/<id>` when set);
// OnBehalfOf names the human inside whose scope the act lands;
// AuthorizedBy is the SHA of the authorize commit that opened the
// scope. Scope carries the lifecycle event for `aiwf authorize`
// commits (`opened` / `paused` / `resumed`); ScopeEnds is the slice
// of authorize-SHAs whose scopes the commit terminated (multiple
// ends per commit are allowed). Reason carries the free-text
// rationale from `aiwf-reason:`. All fields are empty for pre-I2.5
// commits — the renderer treats absence as "no chip".
type HistoryEvent struct {
	Date         string              `json:"date"`
	Actor        string              `json:"actor"`
	Verb         string              `json:"verb"`
	Detail       string              `json:"detail"`
	Commit       string              `json:"commit"`
	Body         string              `json:"body,omitempty"`
	To           string              `json:"to,omitempty"`
	Force        string              `json:"force,omitempty"`
	AuditOnly    string              `json:"audit_only,omitempty"`
	Principal    string              `json:"principal,omitempty"`
	OnBehalfOf   string              `json:"on_behalf_of,omitempty"`
	AuthorizedBy string              `json:"authorized_by,omitempty"`
	Scope        string              `json:"scope,omitempty"`
	ScopeEnds    []string            `json:"scope_ends,omitempty"`
	Reason       string              `json:"reason,omitempty"`
	Tests        *gitops.TestMetrics `json:"tests,omitempty"`
}

// hasCommits reports whether root's HEAD points at a real commit.
// `git log` on an empty repo errors with "your current branch X does
// not have any commits yet"; this guard converts that into "no events".
//
// Deliberately duplicated from cliutil.HasCommits (D-0045) rather than
// imported: entityview must stay free of internal/cli/* so a future
// non-CLI consumer never drags in Cobra transitively through cliutil.
func hasCommits(ctx context.Context, root string) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = root
	return cmd.Run() == nil
}

// ReadHistory shells out to `git log` and returns one HistoryEvent per
// commit whose `aiwf-entity:` or `aiwf-prior-entity:` trailer matches
// id. Events are returned oldest-first.
//
// The git format string carries seven fields per record separated by
// the ASCII unit separator (\x1f), with the ASCII record separator
// (\x1e) between commits — none of these appear in subjects or
// trailers, so a single split suffices. Pre-I2 commits without
// `aiwf-to:` or `aiwf-force:` trailers produce empty strings for
// those fields; the renderer treats empty as "absent" and emits a
// dash, which is the load-bearing backwards-compat behavior.
//
// For a bare milestone id (e.g. `M-007`), the query also matches
// composite-id trailers under that milestone (`M-007/AC-N`) so the
// milestone view shows its AC events alongside its own. The match is
// anchored on the literal `/` boundary so `M-007/` cannot prefix-
// match `M-070/`. A composite id queried directly (`M-007/AC-1`)
// matches only that AC's events.
func ReadHistory(ctx context.Context, root, id string) ([]HistoryEvent, error) {
	return ReadHistoryChain(ctx, root, []string{id})
}

// ReadHistoryChain is ReadHistory's lineage-aware variant: it greps
// git log for any aiwf-entity / aiwf-prior-entity trailer matching
// any id in chain, dedupes by commit SHA, and returns a single
// oldest-first chronological slice. Used by `aiwf history <id>`
// after the cmd dispatcher has expanded id through prior_ids
// lineage. A single-element chain is the pre-G37 behavior; longer
// chains weave pre-rename and post-rename history into one timeline.
func ReadHistoryChain(ctx context.Context, root string, chain []string) ([]HistoryEvent, error) {
	if !hasCommits(ctx, root) {
		return nil, nil
	}
	if len(chain) == 0 {
		return nil, nil
	}
	const sep = "\x1f"
	const recSep = "\x1e\n"
	args := []string{
		"log",
		"--reverse",
		"-E",
	}
	for _, id := range chain {
		// Width-tolerant per AC-2/AC-4 in M-081: a query for E-22
		// matches both legacy `E-22` trailers and canonical `E-0022`
		// trailers (and vice versa) via entity.IDGrepAlternation.
		alt := entity.IDGrepAlternation(id)
		args = append(args,
			"--grep", "^aiwf-entity: "+alt+"$",
			"--grep", "^aiwf-prior-entity: "+alt+"$",
		)
		if isBareMilestoneID(id) {
			// Path-prefix match anchored on the literal `/` boundary
			// so M-007/ cannot match M-070/. Includes M-NNN/AC-N
			// events. The bare-id alternation handles width tolerance;
			// the AC-N portion stays free-form (any digits).
			args = append(args,
				"--grep", "^aiwf-entity: "+alt+"/AC-[0-9]+$",
				"--grep", "^aiwf-prior-entity: "+alt+"/AC-[0-9]+$",
			)
		}
	}
	args = append(args,
		"--pretty=tformat:%H"+sep+"%aI"+sep+"%s"+
			sep+"%(trailers:key=aiwf-verb,valueonly=true,unfold=true)"+
			sep+"%(trailers:key=aiwf-actor,valueonly=true,unfold=true)"+
			sep+"%(trailers:key=aiwf-to,valueonly=true,unfold=true)"+
			sep+"%(trailers:key=aiwf-force,valueonly=true,unfold=true)"+
			sep+"%(trailers:key=aiwf-audit-only,valueonly=true,unfold=true)"+
			sep+"%(trailers:key=aiwf-principal,valueonly=true,unfold=true)"+
			sep+"%(trailers:key=aiwf-on-behalf-of,valueonly=true,unfold=true)"+
			sep+"%(trailers:key=aiwf-authorized-by,valueonly=true,unfold=true)"+
			sep+"%(trailers:key=aiwf-scope,valueonly=true,unfold=true)"+
			sep+"%(trailers:key=aiwf-scope-ends,valueonly=true,unfold=true)"+
			sep+"%(trailers:key=aiwf-reason,valueonly=true,unfold=true)"+
			sep+"%(trailers:key=aiwf-tests,valueonly=true,unfold=true)"+
			sep+"%b\x1e",
	)
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

	var events []HistoryEvent
	const fieldCount = 16
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, sep, fieldCount)
		if len(parts) < fieldCount {
			continue
		}
		verb := strings.TrimSpace(parts[3])
		actor := strings.TrimSpace(parts[4])
		// Skip prose-mention false-positives (G30): `--grep` matched a
		// wrapped line that starts with `aiwf-entity: <id>` but Git's
		// trailer parser found no real aiwf-verb / aiwf-actor pair.
		// A genuine entity event always carries both.
		if verb == "" && actor == "" {
			continue
		}
		ev := HistoryEvent{
			Commit:       ShortHash(parts[0]),
			Date:         parts[1],
			Detail:       strings.TrimSpace(parts[2]),
			Verb:         verb,
			Actor:        actor,
			To:           strings.TrimSpace(parts[5]),
			Force:        strings.TrimSpace(parts[6]),
			AuditOnly:    strings.TrimSpace(parts[7]),
			Principal:    strings.TrimSpace(parts[8]),
			OnBehalfOf:   strings.TrimSpace(parts[9]),
			AuthorizedBy: strings.TrimSpace(parts[10]),
			Scope:        strings.TrimSpace(parts[11]),
			ScopeEnds:    SplitMultiValueTrailer(parts[12]),
			Reason:       strings.TrimSpace(parts[13]),
			Body:         StripTrailers(strings.TrimSpace(parts[15])),
		}
		if metrics, ok := gitops.ParseTestMetrics(parts[14]); ok {
			m := metrics
			ev.Tests = &m
		}
		events = append(events, ev)
	}
	return events, nil
}

// SplitMultiValueTrailer splits a `git log %(trailers:key=...,
// valueonly=true,unfold=true)` cell into one entry per repeated
// trailer. Multi-value trailers (notably aiwf-scope-ends) are
// rendered newline-separated by git; we split, trim, and drop empty
// entries.
func SplitMultiValueTrailer(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// StripTrailers removes the trailing trailer block from a commit body.
// `git log %(body)` includes everything after the subject and the
// separating blank line, including trailers; we only want the prose.
//
// The heuristic walks backward through a contiguous run of
// trailer-shape `<Token>: <value>` lines at the end of the body. The
// run is only treated as a trailer block when (a) the run is preceded
// by a blank line or is the entire body, and (b) the run contains at
// least one `aiwf-*` trailer. The aiwf-* marker is what distinguishes
// real trailers (which we always emit) from body prose that happens to
// look like a trailer (e.g. "decided: 30 days" written by a human).
func StripTrailers(body string) string {
	if body == "" {
		return ""
	}
	lines := strings.Split(body, "\n")

	// Walk backward, eating trailing blank lines.
	end := len(lines)
	for end > 0 && lines[end-1] == "" {
		end--
	}
	// Walk backward through the contiguous trailer-shape block.
	trailerStart := end
	for trailerStart > 0 && isTrailerLine(lines[trailerStart-1]) {
		trailerStart--
	}
	hasTrailer := trailerStart < end
	precededByBlank := trailerStart == 0 || lines[trailerStart-1] == ""
	hasAiwfMarker := false
	for i := trailerStart; i < end; i++ {
		if strings.HasPrefix(lines[i], "aiwf-") {
			hasAiwfMarker = true
			break
		}
	}
	if !hasTrailer || !precededByBlank || !hasAiwfMarker {
		return strings.TrimSpace(body)
	}
	// Strip the trailer block plus the blank line separating it.
	cut := trailerStart
	for cut > 0 && lines[cut-1] == "" {
		cut--
	}
	return strings.TrimSpace(strings.Join(lines[:cut], "\n"))
}

// isTrailerLine reports whether s looks like a git commit trailer:
// a `Key: value` line where Key matches the conventional shape
// (alphanumerics, hyphens, no whitespace before the colon).
func isTrailerLine(s string) bool {
	idx := strings.Index(s, ": ")
	if idx <= 0 {
		return false
	}
	for _, r := range s[:idx] {
		switch {
		case r == '-':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

// ShortHash returns the first 7 hex digits of a SHA, the conventional
// short form. Falls back to the full hash if it is shorter.
func ShortHash(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}

// bareMilestoneIDPattern recognizes a top-level milestone id (`M-NNN`).
// Used by ReadHistoryChain to decide whether to also match composite-id
// trailers under the milestone (the path-prefix shape promised by the
// design).
var bareMilestoneIDPattern = regexp.MustCompile(`^M-\d{3,}$`)

// isBareMilestoneID reports whether id is a bare milestone id that
// should match its AC events too (path-prefix match).
func isBareMilestoneID(id string) bool {
	return bareMilestoneIDPattern.MatchString(id)
}
