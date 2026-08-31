package check

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/skills"
)

// runCommitMsg validates aiwf-verb trailers in a commit-message
// file against the running binary's Cobra verb tree ∪ the
// ritualVerbs allowlist. Used by the `.git/hooks/commit-msg` hook
// installed by aiwf init/update. Closes G-0218's primary chokepoint.
//
// Exit codes: ExitOK pass; ExitFindings refused value(s); ExitUsage
// bad path / missing file; ExitInternal permission or IO error,
// ritualVerbs derivation failure.
func runCommitMsg(path string, registeredVerbs map[string]struct{}, stderr io.Writer) int {
	if path == "" {
		_, _ = fmt.Fprintln(stderr, "aiwf check: --commit-msg requires a path")
		return cliutil.ExitUsage
	}
	msg, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			_, _ = fmt.Fprintf(stderr, "aiwf check: commit-msg file does not exist: %s\n", path)
			return cliutil.ExitUsage
		}
		// Permission, EISDIR, EIO — not an operator typo; an
		// environment problem the operator wants surfaced clearly.
		_, _ = fmt.Fprintf(stderr, "aiwf check: reading commit-msg %q: %v\n", path, err)
		return cliutil.ExitInternal
	}

	ritualVerbs, err := skills.RitualTrailerVerbs()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "aiwf check: %v\n", err)
		return cliutil.ExitInternal
	}

	// Extract the trailer block via git's canonical heuristic; raw
	// ParseTrailers on the file body would yield false positives on
	// `Key: value` lines that appear in body prose (e.g. a commit
	// message DISCUSSING aiwf-verb: implement as an example).
	block, err := extractTrailerBlock(context.Background(), msg)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "aiwf check: extracting trailers from %q: %v\n", path, err)
		return cliutil.ExitInternal
	}
	if code := checkACScopedSubject(msg, block, stderr); code != cliutil.ExitOK {
		return code
	}

	if len(block) == 0 {
		return cliutil.ExitOK
	}
	// Trailer keys are case-sensitive (`Aiwf-Verb` is silently
	// ignored here; the trailer-keys policy polices casing elsewhere).
	var bad []string
	for _, tr := range gitops.ParseTrailers(string(block)) {
		if tr.Key != gitops.TrailerVerb {
			continue
		}
		if _, ok := registeredVerbs[tr.Value]; ok {
			continue
		}
		if _, ok := ritualVerbs[tr.Value]; ok {
			continue
		}
		bad = append(bad, tr.Value)
	}
	if len(bad) == 0 {
		return cliutil.ExitOK
	}
	sort.Strings(bad)
	_, _ = fmt.Fprintf(stderr,
		"aiwf check: commit-msg refuses aiwf-verb trailer value(s): %q\n"+
			"  Allowed: the Cobra verb tree (`aiwf <verb> --help`) ∪ ritualVerbs (wrap-milestone, wrap-epic).\n"+
			"  An empty value (`aiwf-verb:` with nothing after) is a malformed trailer — name a verb or remove the line.\n"+
			"  For epic-integration merges use `aiwf-verb: wrap-milestone` per aiwfx-wrap-milestone.\n",
		bad)
	return cliutil.ExitFindings
}

// extractTrailerBlock pipes the commit message through
// `git interpret-trailers --parse` to honor git's canonical
// trailer-block heuristic (last contiguous paragraph of
// trailer-shaped lines, 75% threshold, etc.). Re-implementing the
// heuristic in Go would drift; deferring to git itself matches
// the "framework correctness must not depend on the LLM's behavior"
// principle in miniature. Empty output means no trailer block.
func extractTrailerBlock(ctx context.Context, msg []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", "interpret-trailers", "--parse")
	cmd.Stdin = bytes.NewReader(msg)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git interpret-trailers --parse: %w\n%s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// acScopedSubject matches the trailing `(M-NNNN/AC-N)` scope the per-AC commit
// convention puts on an implementation commit's subject. The id shape is
// deliberately loose on width — Canonicalize settles narrow against canonical —
// and anchored at end of line so a scope mentioned mid-sentence is not a claim.
var acScopedSubject = regexp.MustCompile(`\(([A-Za-z]-\d+/AC-\d+)\)\s*$`)

// checkACScopedSubject refuses a commit whose subject claims to implement an
// acceptance criterion while its aiwf-entity trailer names something else, or
// nothing.
//
// The subject is the commit's own claim; the trailer is what makes the claim
// reachable, because `aiwf history <id>/AC-N` selects by trailer and never reads
// a subject. Without the trailer the link from a criterion to the commit that
// implemented it exists only in the milestone spec's prose.
//
// A subject naming no AC has made no claim and is not judged, which is what
// keeps the rule silent for a project that does not scope subjects by AC, and
// for an acceptance criterion met by an observation rather than by code.
func checkACScopedSubject(msg, block []byte, stderr io.Writer) int {
	subject, _, _ := bytes.Cut(msg, []byte("\n"))
	m := acScopedSubject.FindSubmatch(bytes.TrimSpace(subject))
	if m == nil {
		return cliutil.ExitOK
	}
	claimed := entity.Canonicalize(string(m[1]))

	var carried string
	for _, tr := range gitops.ParseTrailers(string(block)) {
		if tr.Key == gitops.TrailerEntity {
			carried = strings.TrimSpace(tr.Value)
		}
	}
	if entity.Canonicalize(carried) == claimed {
		return cliutil.ExitOK
	}

	got := carried
	if got == "" {
		got = "no aiwf-entity trailer"
	} else {
		got = "aiwf-entity: " + got
	}
	_, _ = fmt.Fprintf(stderr,
		"aiwf check: commit-msg refuses a subject claiming %s while carrying %s\n"+
			"  The subject says this commit implements %s; the trailer is what makes it\n"+
			"  reachable from that criterion — `aiwf history` selects by trailer, never by subject.\n"+
			"  Add: --trailer \"aiwf-entity: %s\"\n"+
			"  Or drop the (%s) scope from the subject if this commit does not implement it.\n",
		claimed, got, claimed, claimed, claimed)
	return cliutil.ExitFindings
}
