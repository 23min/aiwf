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

// runCommitMsg refuses a commit message at composition time, on four
// grounds, all of which cost a second here and an amend or a rebase once
// the commit exists. Used by the `.git/hooks/commit-msg` hook installed
// by aiwf init/update.
//
//   - an aiwf-verb value outside the running binary's Cobra verb tree ∪
//     the ritualVerbs allowlist;
//   - a subject claiming an acceptance criterion whose aiwf-entity
//     trailer names something else, or nothing;
//   - an aiwf trailer block git will not read, because a blank line
//     leaves it out of the message's final paragraph;
//   - a staged edit to the ritual authoring tree that names no entity.
//
// root is the repo whose index the last of those reads; empty means the
// process working directory, which is where git runs a hook.
//
// Exit codes: ExitOK pass; ExitFindings refused value(s); ExitUsage
// bad path / missing file; ExitInternal permission or IO error,
// ritualVerbs derivation failure.
func runCommitMsg(path, root string, registeredVerbs map[string]struct{}, stderr io.Writer) int {
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

	if code := checkHiddenTrailerBlock(msg, stderr); code != cliutil.ExitOK {
		return code
	}

	if code := checkShippedSurfaceOwner(root, block, stderr); code != cliutil.ExitOK {
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

// trailerLine matches a line of the `Key: value` shape git's trailer parser
// recognizes.
var trailerLine = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]*:\s`)

// checkHiddenTrailerBlock refuses a message carrying an aiwf trailer block git
// will not read.
//
// Git takes trailers from a message's last paragraph only. A blank line between
// the aiwf block and a trailing line of somebody else's convention — a
// `Co-Authored-By:`, typically — leaves the aiwf block as ordinary body prose:
// `aiwf history` never sees the commit, and the verb-value check above has
// nothing to judge, so a value outside the closed set rides through. The author
// sees a correct-looking message, which is why this is caught where it is
// written rather than reported afterwards.
//
// The signature is a whole paragraph of trailer-shaped lines carrying at least
// one aiwf key, sitting before the final paragraph. A message that merely
// mentions a trailer key inside prose is not that, and is left alone — which is
// the case the parser indirection exists for.
func checkHiddenTrailerBlock(msg []byte, stderr io.Writer) int {
	_, body, found := bytes.Cut(msg, []byte("\n"))
	if !found {
		return cliutil.ExitOK
	}
	paras := splitParagraphs(string(body))
	if len(paras) < 2 {
		return cliutil.ExitOK
	}
	// The final paragraph is the one git reads; every earlier one is prose as
	// far as the parser is concerned.
	for _, para := range paras[:len(paras)-1] {
		keys, ok := aiwfTrailerParagraph(para)
		if !ok {
			continue
		}
		_, _ = fmt.Fprintf(stderr,
			"aiwf check: commit-msg refuses an aiwf trailer block git will not read: %s\n"+
				"  Git takes trailers from the last paragraph only, and a blank line above the\n"+
				"  final one leaves this block as body prose — invisible to `aiwf history` and to\n"+
				"  the trailer-value check.\n"+
				"  Move these into the final paragraph (no blank line before it): %s\n",
			strings.Join(keys, ", "), strings.Join(keys, ", "))
		return cliutil.ExitFindings
	}
	return cliutil.ExitOK
}

// splitParagraphs splits on blank lines and drops empty runs.
func splitParagraphs(s string) []string {
	var out []string
	for _, p := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n\n") {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// aiwfTrailerParagraph reports whether every line of para is trailer-shaped and
// at least one carries an aiwf key, returning those keys.
func aiwfTrailerParagraph(para string) ([]string, bool) {
	var keys []string
	for _, line := range strings.Split(strings.TrimSpace(para), "\n") {
		if !trailerLine.MatchString(line) {
			return nil, false
		}
		if key, _, ok := strings.Cut(line, ":"); ok && strings.HasPrefix(key, "aiwf-") {
			keys = append(keys, key)
		}
	}
	return keys, len(keys) > 0
}

// checkShippedSurfaceOwner refuses a staged edit to the ritual authoring tree
// whose message names no entity.
//
// A SKILL.md there materializes into consumer repos, so the edit reaches
// consumers directly and something must own it. The CI backstop judges commits,
// which means a forgotten trailer is found once the commit exists and the repair
// is an amend or a rebase; refusing at composition costs a second instead.
//
// No repo detection is needed. The rule fires on a staged path under the
// authoring tree, and a consumer repo has none, so the path predicate is the
// scope. Presence is all it asks: resolving the id against the tree needs a load
// the hook does not do, and the CI backstop keeps that half.
func checkShippedSurfaceOwner(root string, block []byte, stderr io.Writer) int {
	staged, err := stagedRitualEdits(root)
	if err != nil || len(staged) == 0 {
		// A repo the hook cannot read the index of is not a fault to state
		// here; the CI backstop still judges the commit afterwards.
		return cliutil.ExitOK
	}
	for _, tr := range gitops.ParseTrailers(string(block)) {
		if tr.Key == gitops.TrailerEntity && strings.TrimSpace(tr.Value) != "" {
			return cliutil.ExitOK
		}
	}
	_, _ = fmt.Fprintf(stderr,
		"aiwf check: commit-msg refuses a shipped-ritual edit that names no entity: %s\n"+
			"  A SKILL.md here materializes into consumer repos, so the edit needs an owner.\n"+
			"  Add: --trailer \"aiwf-entity: <id>\" naming the epic, milestone, gap or decision it belongs to.\n"+
			"  No aiwf-verb is wanted — no aiwf verb commits source.\n",
		strings.Join(staged, ", "))
	return cliutil.ExitFindings
}

// ritualAuthoringDir is the embedded-ritual authoring tree whose edits ship to
// consumers. It matches the path the CI-tier backstop watches.
const ritualAuthoringDir = "internal/skills/embedded-rituals/"

// stagedRitualEdits lists staged paths under the ritual authoring tree.
func stagedRitualEdits(root string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var hits []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasPrefix(line, ritualAuthoringDir) {
			hits = append(hits, line)
		}
	}
	return hits, nil
}
