package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/aiwfyaml"
	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/config"
	"github.com/23min/ai-workflow-v2/tools/internal/initrepo"
	"github.com/23min/ai-workflow-v2/tools/internal/render"
	"github.com/23min/ai-workflow-v2/tools/internal/skills"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
	"github.com/23min/ai-workflow-v2/tools/internal/version"
)

// runInit handles `aiwf init`: writes aiwf.yaml, scaffolds entity
// directories, materializes skills, appends to .gitignore, writes a
// CLAUDE.md template, and installs the pre-push hook. No commit.
//
// --dry-run reports the would-be ledger without touching disk.
// --skip-hook performs every other step but omits hook installation.
func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root (default: cwd)")
	actor := fs.String("actor", "", "default actor for the commit trailer (overrides git config derivation)")
	dryRun := fs.Bool("dry-run", false, "report what init would do without writing anything")
	skipHook := fs.Bool("skip-hook", false, "skip installing the pre-push hook (every other step still runs)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	rootDir, err := resolveInitRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf init: %v\n", err)
		return exitUsage
	}

	if !*dryRun {
		release, rc := acquireRepoLock(rootDir, "aiwf init")
		if release == nil {
			return rc
		}
		defer release()
	}

	res, err := initrepo.Init(context.Background(), rootDir, initrepo.Options{
		ActorOverride: *actor,
		AiwfVersion:   Version,
		DryRun:        *dryRun,
		SkipHook:      *skipHook,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf init: %v\n", err)
		return exitInternal
	}

	if res.DryRun {
		fmt.Println("aiwf init: dry-run — nothing was written.")
	}
	for _, s := range res.Steps {
		if s.Detail != "" {
			fmt.Printf("  %-9s  %s  (%s)\n", s.Action, s.What, s.Detail)
		} else {
			fmt.Printf("  %-9s  %s\n", s.Action, s.What)
		}
	}

	if res.HookConflict {
		fmt.Println()
		fmt.Println("aiwf init: setup landed except the pre-push hook.")
		fmt.Println("A non-aiwf hook is already at .git/hooks/pre-push and was left untouched.")
		fmt.Println("To finish wiring validation into your push flow, do one of:")
		fmt.Println("  1. Add this line inside the existing hook:    aiwf check || exit 1")
		fmt.Println("  2. Use a hook manager (husky/lefthook/etc.) to compose hooks; have it run `aiwf check`.")
		fmt.Println("Then drop the marker comment `# aiwf:pre-push` somewhere in the hook so future")
		fmt.Println("`aiwf init` runs recognise it as managed and refresh it on binary upgrades.")
		fmt.Println()
		fmt.Println("Without this, `aiwf check` won't run automatically on `git push`.")
		fmt.Println("You can still validate manually any time with `aiwf check`.")
		return exitFindings
	}

	switch {
	case res.DryRun:
		fmt.Println("\naiwf init: dry-run complete. Re-run without --dry-run to apply.")
	case *skipHook:
		fmt.Println("\naiwf init: done (pre-push hook skipped). Commit aiwf.yaml when you're ready.")
		fmt.Println("Run `aiwf init` again later to install the hook, or wire `aiwf check` into your push flow manually.")
		if !ritualsPluginInstalled(rootDir) {
			printRitualsSuggestion()
		}
	default:
		fmt.Println("\naiwf init: done. Commit aiwf.yaml when you're ready.")
		if !ritualsPluginInstalled(rootDir) {
			printRitualsSuggestion()
		}
	}
	return exitOK
}

// runUpdate handles `aiwf update`: refreshes every marker-managed
// framework artifact the consumer is opted into. The pipeline is the
// same one `aiwf init` runs after first-time scaffolding —
// `initrepo.RefreshArtifacts` — so init and update converge to the
// same state for a given binary version + aiwf.yaml.
//
// Concretely the verb refreshes:
//   - the embedded skills under .claude/skills/aiwf-*
//   - the .gitignore patterns covering them
//   - the marker-managed pre-push hook
//   - the marker-managed pre-commit hook (gated by
//     aiwf.yaml's status_md.auto_update; default-on)
//
// Hook conflicts (a non-marker hook already in place) are reported
// in the per-step ledger and surface a remediation block, mirroring
// `aiwf init`'s conflict path.
func runUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf update: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf update")
	if release == nil {
		return rc
	}
	defer release()

	cfg, err := config.Load(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf update: %v\n", err)
		return exitInternal
	}

	steps, conflict, err := initrepo.RefreshArtifacts(context.Background(), rootDir, initrepo.RefreshOptions{
		StatusMdAutoUpdate: cfg.StatusMdAutoUpdate(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf update: %v\n", err)
		return exitInternal
	}

	for _, s := range steps {
		if s.Detail != "" {
			fmt.Printf("  %-9s  %s  (%s)\n", s.Action, s.What, s.Detail)
		} else {
			fmt.Printf("  %-9s  %s\n", s.Action, s.What)
		}
	}

	if conflict {
		fmt.Println()
		fmt.Println("aiwf update: artifacts refreshed except a hook with no aiwf marker.")
		fmt.Println("A non-aiwf hook is at one of .git/hooks/pre-push or .git/hooks/pre-commit and was left untouched.")
		fmt.Println("To finish wiring, either:")
		fmt.Println("  1. Add the relevant aiwf invocation inside your existing hook")
		fmt.Println("       pre-push:    aiwf check || exit 1")
		fmt.Println("       pre-commit:  aiwf status --root \"$(git rev-parse --show-toplevel)\" --format=md > STATUS.md && git add STATUS.md")
		fmt.Println("  2. Use a hook manager (husky/lefthook/etc.) to compose hooks.")
		fmt.Println("Then drop the marker comment somewhere in the hook (`# aiwf:pre-push` or `# aiwf:pre-commit`)")
		fmt.Println("so future `aiwf init`/`aiwf update` runs recognise it as managed.")
		return exitFindings
	}

	fmt.Println("\naiwf update: done.")
	return exitOK
}

// runHistory handles `aiwf history <id>`: filters git log for the
// entity's structured trailers and prints one line per event.
func runHistory(args []string) int {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root")
	format := fs.String("format", "text", "output format: text or json")
	pretty := fs.Bool("pretty", false, "indent JSON output (only with --format=json)")
	showAuth := fs.Bool("show-authorization", false, "include the full aiwf-authorized-by SHA on scope-authorized rows (text format only)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf history: usage: aiwf history <id>")
		return exitUsage
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf history: --format must be text or json, got %q\n", *format)
		return exitUsage
	}
	id := rest[0]

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf history: %v\n", err)
		return exitUsage
	}

	events, err := readHistory(context.Background(), rootDir, id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf history: %v\n", err)
		return exitInternal
	}

	switch *format {
	case "text":
		if len(events) == 0 {
			fmt.Printf("no history for %s\n", id)
			return exitOK
		}
		// Resolve authorize-SHA → scope-entity once; chip rendering
		// reads from the map. Pre-I2.5 commits and pure entity-only
		// histories produce an empty map (no chips).
		scopeEntities := buildScopeEntityMap(context.Background(), rootDir, events)
		for i := range events {
			e := &events[i]
			fmt.Printf("%s  %-16s  %-10s  %-12s  %s  %s%s\n",
				e.Date, renderActor(*e), e.Verb, renderTo(e.To), e.Detail, e.Commit,
				renderScopeChips(*e, scopeEntities, *showAuth))
			if e.Force != "" {
				fmt.Printf("    [forced: %s]\n", e.Force)
			}
			if e.AuditOnly != "" {
				fmt.Printf("    [audit-only: %s]\n", e.AuditOnly)
			}
			if e.Reason != "" {
				fmt.Printf("    [reason: %s]\n", e.Reason)
			}
			if e.Body != "" {
				for _, line := range strings.Split(e.Body, "\n") {
					fmt.Printf("    %s\n", line)
				}
			}
		}
	case "json":
		env := render.Envelope{
			Tool:    "aiwf",
			Version: Version,
			Status:  "ok",
			Result:  map[string]any{"id": id, "events": events},
			Metadata: map[string]any{
				"root":   rootDir,
				"events": len(events),
			},
		}
		if err := render.JSON(os.Stdout, env, *pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf history: %v\n", err)
			return exitInternal
		}
	}
	return exitOK
}

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
	Date         string   `json:"date"`
	Actor        string   `json:"actor"`
	Verb         string   `json:"verb"`
	Detail       string   `json:"detail"`
	Commit       string   `json:"commit"`
	Body         string   `json:"body,omitempty"`
	To           string   `json:"to,omitempty"`
	Force        string   `json:"force,omitempty"`
	AuditOnly    string   `json:"audit_only,omitempty"`
	Principal    string   `json:"principal,omitempty"`
	OnBehalfOf   string   `json:"on_behalf_of,omitempty"`
	AuthorizedBy string   `json:"authorized_by,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	ScopeEnds    []string `json:"scope_ends,omitempty"`
	Reason       string   `json:"reason,omitempty"`
}

// readHistory shells out to `git log` and returns one HistoryEvent per
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
func readHistory(ctx context.Context, root, id string) ([]HistoryEvent, error) {
	if !hasCommits(ctx, root) {
		return nil, nil
	}
	const sep = "\x1f"
	const recSep = "\x1e\n"
	args := []string{
		"log",
		"--reverse",
		"-E",
		"--grep", "^aiwf-entity: " + regexp.QuoteMeta(id) + "$",
		"--grep", "^aiwf-prior-entity: " + regexp.QuoteMeta(id) + "$",
	}
	if isBareMilestoneID(id) {
		// Path-prefix match anchored on the literal `/` boundary so
		// M-007/ cannot match M-070/. Includes M-NNN/AC-N events.
		args = append(args,
			"--grep", "^aiwf-entity: "+regexp.QuoteMeta(id)+"/AC-[0-9]+$",
			"--grep", "^aiwf-prior-entity: "+regexp.QuoteMeta(id)+"/AC-[0-9]+$",
		)
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
	const fieldCount = 15
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, sep, fieldCount)
		if len(parts) < fieldCount {
			continue
		}
		events = append(events, HistoryEvent{
			Commit:       shortHash(parts[0]),
			Date:         parts[1],
			Detail:       strings.TrimSpace(parts[2]),
			Verb:         strings.TrimSpace(parts[3]),
			Actor:        strings.TrimSpace(parts[4]),
			To:           strings.TrimSpace(parts[5]),
			Force:        strings.TrimSpace(parts[6]),
			AuditOnly:    strings.TrimSpace(parts[7]),
			Principal:    strings.TrimSpace(parts[8]),
			OnBehalfOf:   strings.TrimSpace(parts[9]),
			AuthorizedBy: strings.TrimSpace(parts[10]),
			Scope:        strings.TrimSpace(parts[11]),
			ScopeEnds:    splitMultiValueTrailer(parts[12]),
			Reason:       strings.TrimSpace(parts[13]),
			Body:         stripTrailers(strings.TrimSpace(parts[14])),
		})
	}
	return events, nil
}

// splitMultiValueTrailer splits a `git log %(trailers:key=...,
// valueonly=true,unfold=true)` cell into one entry per repeated
// trailer. Multi-value trailers (notably aiwf-scope-ends) are
// rendered newline-separated by git; we split, trim, and drop empty
// entries.
func splitMultiValueTrailer(raw string) []string {
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

// stripTrailers removes the trailing trailer block from a commit body.
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
func stripTrailers(body string) string {
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

// shortHash returns the first 7 hex digits of a SHA, the conventional
// short form. Falls back to the full hash if it is shorter.
func shortHash(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}

// renderTo formats the target-status column in `aiwf history` text
// output. Empty (the absent-trailer case for non-promote events and
// pre-I2 promote commits) renders as "-"; a populated value is shown
// with a leading arrow so the column reads as a transition target.
func renderTo(to string) string {
	if to == "" {
		return "-"
	}
	return "→ " + to
}

// renderActor formats the actor column. When a non-human principal
// is present and differs from the actor (the agent-acts-for-human
// case from I2.5), the column reads `principal via agent` so the
// human is visually attributed first. Direct human acts (no
// principal) render the actor verbatim.
func renderActor(e HistoryEvent) string {
	if e.Principal == "" || e.Principal == e.Actor {
		return e.Actor
	}
	return e.Principal + " via " + e.Actor
}

// renderScopeChips assembles the trailing chip block for one history
// row. For `aiwf authorize` rows, a `[<scope> <event>]` chip names
// the lifecycle event (`opened` / `paused` / `resumed`). For
// scope-authorized rows, a `[<scope-entity> <auth-short>]` chip
// names the authorizing scope. For terminal-promote rows that ended
// one or more scopes, one `[<scope-entity> ended]` chip per ended
// scope.
//
// scopeEntities maps full auth-SHA to scope-entity id. showAuth
// flips on the full SHA inline (the --show-authorization flag).
//
// The output begins with a leading "  " when non-empty so it sits
// flush against the Commit column the caller already printed.
func renderScopeChips(e HistoryEvent, scopeEntities map[string]string, showAuth bool) string {
	var chips []string
	if e.Verb == "authorize" && e.Scope != "" {
		chips = append(chips, fmt.Sprintf("[scope: %s]", e.Scope))
	}
	if e.AuthorizedBy != "" {
		scopeEntity := scopeEntities[e.AuthorizedBy]
		if scopeEntity == "" {
			scopeEntity = "?"
		}
		sha := shortHash(e.AuthorizedBy)
		if showAuth {
			sha = e.AuthorizedBy
		}
		chips = append(chips, fmt.Sprintf("[%s %s]", scopeEntity, sha))
	}
	for _, sha := range e.ScopeEnds {
		scopeEntity := scopeEntities[sha]
		if scopeEntity == "" {
			scopeEntity = shortHash(sha)
		}
		chips = append(chips, fmt.Sprintf("[%s ended]", scopeEntity))
	}
	if len(chips) == 0 {
		return ""
	}
	return "  " + strings.Join(chips, " ")
}

// buildScopeEntityMap walks every authorize-opener commit visible
// from HEAD once and returns auth-SHA → scope-entity. Used by
// renderScopeChips to label the [<entity> <sha>] chip without a
// per-row git lookup. Pre-I2.5 repos with no authorize commits
// produce an empty map; the renderer falls back gracefully.
//
// The walk is bounded by the existing event set: any auth SHA the
// rendered events reference is looked up via this map; SHAs absent
// from the map render as "?", which is benign.
func buildScopeEntityMap(ctx context.Context, root string, events []HistoryEvent) map[string]string {
	out := map[string]string{}
	if !hasCommits(ctx, root) {
		return out
	}
	cmd := exec.CommandContext(ctx, "git", "log",
		"-E",
		"--grep", "^aiwf-verb: authorize$",
		"--grep", "^aiwf-scope: opened$",
		"--all-match",
		"--pretty=tformat:%H\x1f%(trailers:key=aiwf-entity,valueonly=true,unfold=true)\x1e")
	cmd.Dir = root
	outBytes, err := cmd.Output()
	if err != nil {
		// Treat lookup failure as a missing map: chips render with "?"
		// rather than blocking the verb on a metadata read.
		return out
	}
	for _, rec := range strings.Split(string(outBytes), "\x1e") {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, "\x1f", 2)
		if len(parts) < 2 {
			continue
		}
		sha := strings.TrimSpace(parts[0])
		entity := strings.TrimSpace(parts[1])
		if sha == "" || entity == "" {
			continue
		}
		out[sha] = entity
	}
	return out
}

// bareMilestoneIDPattern recognizes a top-level milestone id (`M-NNN`).
// Used by readHistory to decide whether to also match composite-id
// trailers under the milestone (the path-prefix shape promised by the
// design).
var bareMilestoneIDPattern = regexp.MustCompile(`^M-\d{3,}$`)

// isBareMilestoneID reports whether id is a bare milestone id that
// should match its AC events too (path-prefix match).
func isBareMilestoneID(id string) bool {
	return bareMilestoneIDPattern.MatchString(id)
}

// hasCommits reports whether root's HEAD points at a real commit.
// `git log` on an empty repo errors with "your current branch X does
// not have any commits yet"; this guard converts that into "no events".
func hasCommits(ctx context.Context, root string) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = root
	return cmd.Run() == nil
}

// runDoctor handles `aiwf doctor`: version check, materialized-skill
// drift check, id-collision check. With --self-check, instead drives
// every mutating verb against a throwaway repo to prove the binary
// works end-to-end.
func runDoctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root")
	selfCheck := fs.Bool("self-check", false, "run every verb against a temp repo and report pass/fail")
	checkLatest := fs.Bool("check-latest", false, "look up the latest published aiwf version on the Go module proxy (one HTTP call; honors GOPROXY=off)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	if *selfCheck {
		return runSelfCheck()
	}

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor: %v\n", err)
		return exitUsage
	}

	report, problems := doctorReport(rootDir, doctorOptions{CheckLatest: *checkLatest})
	for _, line := range report {
		fmt.Println(line)
	}
	if problems > 0 {
		return exitFindings
	}
	return exitOK
}

// doctorOptions carries flag-derived knobs into doctorReport. Kept
// separate from runDoctor's flag.FlagSet so doctorReport stays
// flag-package-free and unit-testable. Add fields here when new
// doctor flags arrive.
type doctorOptions struct {
	// CheckLatest, when true, performs a Go module proxy lookup for
	// the latest published aiwf version and adds a `latest:` row to
	// the report. Default false (offline).
	CheckLatest bool
}

// doctorReport collects every doctor finding into a slice of human
// strings and returns the count of problems. Pure for testability.
func doctorReport(rootDir string, opts doctorOptions) (lines []string, problems int) {
	// 1. Binary version (advisory). Always shown; reads from
	//    runtime/debug.ReadBuildInfo via version.Current().
	current := version.Current()
	lines = append(lines, fmt.Sprintf("binary:    %s", renderBinaryVersion(current)))

	// 1b. Latest published (advisory, opt-in). One HTTP call to the
	//     Go module proxy. Skipped unless --check-latest is set so
	//     `aiwf doctor` stays fast and offline by default.
	if opts.CheckLatest {
		lines = append(lines, "latest:    "+renderLatestPublished(current))
	}

	// 2. aiwf.yaml presence + pin coherence (advisory). Pin coherence
	//    compares the aiwf_version: field against the running binary
	//    via version.Compare; mismatches surface as advisory rows
	//    rather than incrementing the problem count (the pin records
	//    intent, not enforcement). Load-error states still increment
	//    problems — those are real config faults.
	cfg, err := config.Load(rootDir)
	switch {
	case errors.Is(err, config.ErrNotFound):
		lines = append(lines, "config:    aiwf.yaml not found (run `aiwf init`)")
		problems++
	case err != nil:
		lines = append(lines, "config:    "+err.Error())
		problems++
	default:
		lines = append(lines, fmt.Sprintf("config:    ok (aiwf_version=%s)", cfg.AiwfVersion))
		if cfg.AiwfVersion != "" {
			lines = append(lines, "pin:       "+renderPinCoherence(current, cfg.AiwfVersion))
		}
		if cfg.LegacyActor != "" {
			// Pre-I2.5 `actor:` field. Identity is now runtime-derived
			// (per provenance-model.md); the file's value is ignored.
			// Surface as a one-line deprecation hint so the user knows
			// the field no longer does anything and can remove it.
			lines = append(lines,
				fmt.Sprintf("           note: aiwf.yaml carries a deprecated `actor: %s` key — identity is now runtime-derived from git config user.email; the field is ignored and can be removed", cfg.LegacyActor))
		}
	}

	// 1b. Runtime-identity resolution. Echoes what the next mutating
	//     verb's aiwf-actor: trailer would say, plus the source the
	//     value came from (--actor flag is absent here, so the source
	//     is git config user.email).
	if actor, source, actorErr := resolveActorWithSource("", rootDir); actorErr != nil {
		lines = append(lines, "actor:     "+actorErr.Error())
		problems++
	} else {
		lines = append(lines, fmt.Sprintf("actor:     %s (from %s)", actor, source))
	}

	// 2. Materialized-skill drift.
	embedded, err := skills.List()
	if err != nil {
		lines = append(lines, "skills:    "+err.Error())
		problems++
	} else {
		drift, missing := skillDrift(rootDir, embedded)
		switch {
		case len(missing) > 0:
			lines = append(lines, fmt.Sprintf("skills:    %d missing — run `aiwf init` or `aiwf update`", len(missing)))
			for _, m := range missing {
				lines = append(lines, "             - "+m)
			}
			problems++
		case len(drift) > 0:
			lines = append(lines, fmt.Sprintf("skills:    %d drifted — run `aiwf update` to refresh", len(drift)))
			for _, d := range drift {
				lines = append(lines, "             - "+d)
			}
			problems++
		default:
			lines = append(lines, fmt.Sprintf("skills:    ok (%d skills, byte-equal to embed)", len(embedded)))
		}
	}

	// 3. id-collision check (only ids-unique findings; all other
	// errors are reported by `aiwf check`).
	tr, loadErrs, err := tree.Load(context.Background(), rootDir)
	if err != nil {
		lines = append(lines, "ids:       "+err.Error())
		problems++
	} else {
		findings := check.Run(tr, loadErrs)
		collisions := 0
		for i := range findings {
			f := &findings[i]
			if f.Code == "ids-unique" {
				collisions++
				lines = append(lines, fmt.Sprintf("ids:       collision %s @ %s", f.EntityID, f.Path))
			}
		}
		if collisions == 0 {
			lines = append(lines, "ids:       ok (no collisions)")
		} else {
			problems++
		}
	}

	// 4. Configured contract validators: list each one and whether
	//    the binary is on PATH. A missing validator is reported but
	//    does not increment problems unless `strict_validators: true`
	//    is set — matches the contract verify rendering.
	lines, problems = appendValidatorReport(lines, problems, rootDir)

	// 5. Filesystem case-sensitivity. Informational; case-insensitive
	//    is the default on macOS APFS and Windows NTFS, and on those
	//    volumes E-01-foo and E-01-Foo collapse to the same dir.
	//    Users should know which they're on before they hit the
	//    footgun. The check.casePaths validator catches actual
	//    collisions; this line just surfaces the platform fact.
	lines = append(lines, fmt.Sprintf("filesystem: %s (%s)", filesystemCaseLabel(rootDir), rootDir))

	// 6. Pre-push hook: present, marker-tagged, and pointing at a
	//    binary that still exists. Catches the G12 drift case where
	//    `aiwf init` baked in an absolute path that's since moved.
	lines, problems = appendHookReport(lines, problems, rootDir)

	// 6b. Pre-commit hook: same drift detection, plus the config-
	//     driven opt-out — when status_md.auto_update is false, the
	//     desired state is "no marker-managed hook on disk".
	lines, problems = appendPreCommitHookReport(lines, problems, rootDir)

	// 5. Rituals-plugin presence (soft note — does not increment
	// problems). Best-effort heuristic: greps project/local settings
	// for `aiwf-extensions`. User-scope installs are invisible here,
	// so a "not detected" result is a hint, not a finding.
	if ritualsPluginInstalled(rootDir) {
		lines = append(lines, "plugin:    rituals plugin detected (aiwf-extensions in .claude/settings)")
	} else {
		lines = append(lines,
			"plugin:    rituals plugin not detected in .claude/settings.{json,local.json}",
			"             aiwf works alone, but the workflow skills and role agents that turn it",
			"             into an end-to-end loop ship via the companion plugin. To install:",
			"               /plugin marketplace add "+ritualsMarketplaceSlug,
			"               /plugin install aiwf-extensions@"+ritualsMarketplaceName,
			"             User-scope plugin installs aren't visible to this check; ignore if installed.",
		)
	}

	return lines, problems
}

// appendHookReport inspects .git/hooks/pre-push and reports its
// state: missing, present-but-not-aiwf-managed, stale (the embedded
// absolute binary path no longer exists), or ok. A stale or
// missing-from-tracked-managed hook is a problem; a non-aiwf hook
// is a warning surfaced as informational text.
func appendHookReport(in []string, problemsIn int, rootDir string) (lines []string, problems int) {
	lines = in
	problems = problemsIn

	hookPath := filepath.Join(rootDir, ".git", "hooks", "pre-push")
	raw, err := os.ReadFile(hookPath)
	if errors.Is(err, os.ErrNotExist) {
		lines = append(lines, "hook:      missing — pre-push validation not installed; run `aiwf init` to install")
		problems++
		return lines, problems
	}
	if err != nil {
		lines = append(lines, "hook:      "+err.Error())
		problems++
		return lines, problems
	}
	if !strings.Contains(string(raw), "# aiwf:pre-push") {
		lines = append(lines, "hook:      present but not aiwf-managed (no `# aiwf:pre-push` marker); aiwf check is not running pre-push")
		return lines, problems
	}
	// Extract the absolute path from `exec '<path>' check`.
	embedded := extractHookExecPath(string(raw))
	if embedded == "" {
		lines = append(lines, "hook:      aiwf-managed but malformed (no exec line found); run `aiwf init` to refresh")
		problems++
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		lines = append(lines, fmt.Sprintf("hook:      stale path %s — binary moved or removed; run `aiwf init` to refresh", embedded))
		problems++
		return lines, problems
	}
	lines = append(lines, fmt.Sprintf("hook:      ok (%s)", embedded))
	return lines, problems
}

// appendPreCommitHookReport inspects .git/hooks/pre-commit and
// reports its state, with one extra wrinkle vs. pre-push: the
// config flag `status_md.auto_update` controls whether the hook is
// supposed to be installed at all. A "no marker hook on disk and
// flag is false" state is the desired-and-actual-agree case and
// reports as `disabled by config` (no problem). A "flag is true and
// hook missing" state is drift (a problem; remediated by `aiwf
// update`).
func appendPreCommitHookReport(in []string, problemsIn int, rootDir string) (lines []string, problems int) {
	lines = in
	problems = problemsIn

	autoUpdate := true
	if cfg, err := config.Load(rootDir); err == nil {
		autoUpdate = cfg.StatusMdAutoUpdate()
	}

	hookPath := filepath.Join(rootDir, ".git", "hooks", "pre-commit")
	raw, err := os.ReadFile(hookPath)
	if errors.Is(err, os.ErrNotExist) {
		if !autoUpdate {
			lines = append(lines, "pre-commit: disabled by config (status_md.auto_update: false)")
			return lines, problems
		}
		lines = append(lines, "pre-commit: missing — STATUS.md auto-update not installed; run `aiwf update`")
		problems++
		return lines, problems
	}
	if err != nil {
		lines = append(lines, "pre-commit: "+err.Error())
		problems++
		return lines, problems
	}
	if !strings.Contains(string(raw), "# aiwf:pre-commit") {
		lines = append(lines, "pre-commit: present but not aiwf-managed (no `# aiwf:pre-commit` marker); STATUS.md is not being auto-updated")
		return lines, problems
	}
	embedded := extractPreCommitExecPath(string(raw))
	if embedded == "" {
		lines = append(lines, "pre-commit: aiwf-managed but malformed (no aiwf invocation found); run `aiwf update` to refresh")
		problems++
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		lines = append(lines, fmt.Sprintf("pre-commit: stale path %s — binary moved or removed; run `aiwf update` to refresh", embedded))
		problems++
		return lines, problems
	}
	if !autoUpdate {
		// Hook on disk but config says off — drift in the other
		// direction. Remediation is the same: `aiwf update` removes it.
		lines = append(lines, "pre-commit: present but config says off (status_md.auto_update: false); run `aiwf update` to remove")
		problems++
		return lines, problems
	}
	lines = append(lines, fmt.Sprintf("pre-commit: ok (%s)", embedded))
	return lines, problems
}

// extractPreCommitExecPath pulls the binary path out of the
// pre-commit hook's `if 'path' status …` line. Returns empty when
// the line cannot be located (malformed hook).
func extractPreCommitExecPath(script string) string {
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "if ") {
			continue
		}
		rest := strings.TrimPrefix(line, "if ")
		if !strings.HasPrefix(rest, "'") {
			continue
		}
		rest = rest[1:]
		end := strings.IndexByte(rest, '\'')
		if end < 0 {
			return ""
		}
		return rest[:end]
	}
	return ""
}

// extractHookExecPath pulls the binary path out of the hook script's
// `exec '<path>' check` line. Returns empty when no such line is
// found (malformed hook).
func extractHookExecPath(script string) string {
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "exec ") {
			continue
		}
		// `exec '/path/to/aiwf' check` — pull the single-quoted segment.
		rest := strings.TrimPrefix(line, "exec ")
		if !strings.HasPrefix(rest, "'") {
			// Bare exec word; take the first token before space.
			if idx := strings.IndexByte(rest, ' '); idx > 0 {
				return rest[:idx]
			}
			return rest
		}
		// Find the closing quote.
		rest = rest[1:]
		end := strings.IndexByte(rest, '\'')
		if end < 0 {
			return ""
		}
		return rest[:end]
	}
	return ""
}

// filesystemCaseLabel returns "case-sensitive" or "case-insensitive"
// based on a probe inside dir: write a temp file, stat its name in
// uppercase, and check whether the filesystem returned the same
// inode. If the probe fails (permissions, no temp space), returns
// "unknown" so the report stays informational rather than blocking.
func filesystemCaseLabel(dir string) string {
	probe, err := os.CreateTemp(dir, ".aiwf-case-probe-")
	if err != nil {
		return "unknown"
	}
	name := probe.Name()
	_ = probe.Close()
	defer func() { _ = os.Remove(name) }()
	upper := filepath.Join(filepath.Dir(name), strings.ToUpper(filepath.Base(name)))
	if _, err := os.Stat(upper); err == nil {
		return "case-insensitive"
	}
	return "case-sensitive"
}

// appendValidatorReport reads aiwf.yaml's contracts block and
// reports each configured validator's binary availability. A
// missing binary is a problem only when strict_validators is set;
// otherwise it's a soft note matching the runtime warning.
func appendValidatorReport(in []string, problemsIn int, rootDir string) (lines []string, problems int) {
	lines = in
	problems = problemsIn
	yamlPath := filepath.Join(rootDir, "aiwf.yaml")
	_, contracts, err := aiwfyaml.Read(yamlPath)
	if err != nil || contracts == nil || len(contracts.Validators) == 0 {
		// No contracts block (or unreadable yaml — already reported
		// by step 1 above). Skip the section silently.
		return lines, problems
	}
	names := make([]string, 0, len(contracts.Validators))
	for n := range contracts.Validators {
		names = append(names, n)
	}
	sort.Strings(names)

	missing := 0
	for _, n := range names {
		v := contracts.Validators[n]
		if _, lpErr := exec.LookPath(v.Command); lpErr == nil {
			lines = append(lines, fmt.Sprintf("validator: %s ok (command=%s)", n, v.Command))
		} else {
			lines = append(lines, fmt.Sprintf("validator: %s missing (command=%s)", n, v.Command))
			missing++
		}
	}
	if missing > 0 && contracts.StrictValidators {
		lines = append(lines, fmt.Sprintf("             %d missing validator(s) and strict_validators=true; pre-push will fail", missing))
		problems += missing
	} else if missing > 0 {
		lines = append(lines,
			"             missing binaries are warnings (strict_validators=false); pushes are not blocked",
			"             install the binary or set strict_validators=true to enforce on every machine",
		)
	}
	return lines, problems
}

// skillDrift compares each embedded skill against its on-disk copy
// and reports two sets: drifted (file exists but differs) and missing
// (file absent).
func skillDrift(rootDir string, embedded []skills.Skill) (drifted, missing []string) {
	for _, s := range embedded {
		on := filepath.Join(rootDir, skills.SkillsDir, s.Name, "SKILL.md")
		got, err := os.ReadFile(on)
		switch {
		case errors.Is(err, os.ErrNotExist):
			missing = append(missing, s.Name)
		case err != nil:
			drifted = append(drifted, s.Name+": "+err.Error())
		case !bytes.Equal(got, s.Content):
			drifted = append(drifted, s.Name)
		}
	}
	return drifted, missing
}

// renderLatestPublished formats the doctor latest: row. Calls
// version.Latest with a fresh context, classifying the result:
//
//	v0.2.0 (up to date)
//	v0.2.1 (binary at v0.2.0; run `aiwf upgrade`)
//	v0.1.0 (binary newer at v0.2.0; rolled back?)
//	unavailable (proxy disabled — set GOPROXY to https://proxy.golang.org or override)
//	unavailable (timeout / network error)
//	skew unknown (latest is a pseudo-version; module has no tags yet)
//
// Network errors and proxy-disabled never increment the doctor
// problem count: the row is informational, and absent connectivity
// is not a fault of the running aiwf install.
func renderLatestPublished(current version.Info) string {
	latest, err := version.Latest(context.Background())
	switch {
	case errors.Is(err, version.ErrProxyDisabled):
		return "unavailable (proxy disabled — set GOPROXY to https://proxy.golang.org or remove `off` from the chain)"
	case err != nil:
		return fmt.Sprintf("unavailable (%v)", err)
	}
	switch version.Compare(current, latest) {
	case version.SkewEqual:
		return latest.Version + " (up to date)"
	case version.SkewBehind:
		return fmt.Sprintf("%s (binary at %s; run `aiwf upgrade`)", latest.Version, current.Version)
	case version.SkewAhead:
		return fmt.Sprintf("%s (binary newer at %s; rolled back?)", latest.Version, current.Version)
	default:
		// Either side non-tagged. Most common case in the early-PoC
		// world: the module has no semver tags yet, so the proxy
		// returns a pseudo-version. Surface it honestly.
		return fmt.Sprintf("%s (binary at %s; skew unknown — devel or pseudo-version on either side)",
			latest.Version, current.Version)
	}
}

// renderBinaryVersion formats a version.Info for the doctor binary
// row: the version string plus a parenthetical state ("tagged",
// "working-tree build", "pseudo-version"). Mirrors the upgrade
// verb's renderVersionLabel; kept separate to avoid an admin →
// upgrade dependency.
func renderBinaryVersion(info version.Info) string {
	switch {
	case info.Version == version.DevelVersion:
		return info.Version + " (working-tree build)"
	case strings.HasSuffix(info.Version, "+dirty"):
		return info.Version + " (working-tree build)"
	case info.Tagged:
		return info.Version + " (tagged)"
	default:
		return info.Version + " (pseudo-version)"
	}
}

// renderPinCoherence formats the doctor pin: row. Compares the
// aiwf.yaml `aiwf_version:` value against the running binary and
// returns one of:
//
//	matches binary
//	pinned X, binary newer (Y) — update pin or roll back binary
//	pinned X, binary older (Y) — run aiwf upgrade
//	pinned X, binary at Y — skew unknown (devel or pre-release)
//
// Advisory only: the verb does not increment the doctor problem
// count regardless of skew. Hardening the pin into a refusal is a
// deliberate decision filed for later.
func renderPinCoherence(current version.Info, pinRaw string) string {
	pin := version.Parse(pinValueWithVPrefix(pinRaw))
	switch version.Compare(current, pin) {
	case version.SkewEqual:
		return "matches binary (" + pinRaw + ")"
	case version.SkewAhead:
		return fmt.Sprintf("pinned %s, binary newer (%s) — update pin or roll back binary", pinRaw, current.Version)
	case version.SkewBehind:
		return fmt.Sprintf("pinned %s, binary older (%s) — run `aiwf upgrade`", pinRaw, current.Version)
	default:
		return fmt.Sprintf("pinned %s, binary at %s — skew unknown (devel or pre-release on either side)", pinRaw, current.Version)
	}
}

// pinValueWithVPrefix normalizes the aiwf.yaml pin value for
// version.Parse: aiwf.yaml has historically shipped pins as bare
// "0.1.0" (no leading 'v'), while semver tooling (and the proxy)
// expects "v0.1.0". Add the prefix when missing.
func pinValueWithVPrefix(raw string) string {
	if raw == "" {
		return raw
	}
	if strings.HasPrefix(raw, "v") {
		return raw
	}
	return "v" + raw
}

// resolveInitRoot picks the root directory for `aiwf init`. Unlike
// resolveRoot, it does not error when aiwf.yaml is missing — that's
// the normal case for init.
func resolveInitRoot(explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolving --root: %w", err)
		}
		return abs, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting cwd: %w", err)
	}
	return cwd, nil
}
