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
	"sort"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/aiwfyaml"
	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/config"
	"github.com/23min/ai-workflow-v2/tools/internal/initrepo"
	"github.com/23min/ai-workflow-v2/tools/internal/render"
	"github.com/23min/ai-workflow-v2/tools/internal/skills"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
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

// runUpdate handles `aiwf update`: re-materializes skills only.
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

	if err := skills.Materialize(rootDir); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf update: %v\n", err)
		return exitInternal
	}
	fmt.Println("aiwf update: skills re-materialized.")
	return exitOK
}

// runHistory handles `aiwf history <id>`: filters git log for the
// entity's structured trailers and prints one line per event.
func runHistory(args []string) int {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root")
	format := fs.String("format", "text", "output format: text or json")
	pretty := fs.Bool("pretty", false, "indent JSON output (only with --format=json)")
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
		for _, e := range events {
			fmt.Printf("%s  %-16s  %-10s  %s  %s\n", e.Date, e.Actor, e.Verb, e.Detail, e.Commit)
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
type HistoryEvent struct {
	Date   string `json:"date"`
	Actor  string `json:"actor"`
	Verb   string `json:"verb"`
	Detail string `json:"detail"`
	Commit string `json:"commit"`
	Body   string `json:"body,omitempty"`
}

// readHistory shells out to `git log` and returns one HistoryEvent per
// commit whose `aiwf-entity:` or `aiwf-prior-entity:` trailer matches
// id. Events are returned oldest-first.
//
// The git format string carries five fields per record separated by
// the ASCII unit separator (\x1f), with the ASCII record separator
// (\x1e) between commits — none of these appear in subjects or
// trailers, so a single split suffices.
func readHistory(ctx context.Context, root, id string) ([]HistoryEvent, error) {
	if !hasCommits(ctx, root) {
		return nil, nil
	}
	const sep = "\x1f"
	const recSep = "\x1e\n"
	cmd := exec.CommandContext(ctx, "git", "log",
		"--reverse",
		"-E",
		"--grep", "^aiwf-entity: "+id+"$",
		"--grep", "^aiwf-prior-entity: "+id+"$",
		"--pretty=tformat:%H"+sep+"%aI"+sep+"%s"+sep+"%(trailers:key=aiwf-verb,valueonly=true,unfold=true)"+sep+"%(trailers:key=aiwf-actor,valueonly=true,unfold=true)"+sep+"%b\x1e",
	)
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
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, sep, 6)
		if len(parts) < 6 {
			continue
		}
		events = append(events, HistoryEvent{
			Commit: shortHash(parts[0]),
			Date:   parts[1],
			Detail: strings.TrimSpace(parts[2]),
			Verb:   strings.TrimSpace(parts[3]),
			Actor:  strings.TrimSpace(parts[4]),
			Body:   stripTrailers(strings.TrimSpace(parts[5])),
		})
	}
	return events, nil
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

	report, problems := doctorReport(rootDir)
	for _, line := range report {
		fmt.Println(line)
	}
	if problems > 0 {
		return exitFindings
	}
	return exitOK
}

// doctorReport collects every doctor finding into a slice of human
// strings and returns the count of problems. Pure for testability.
func doctorReport(rootDir string) (lines []string, problems int) {
	// 1. Version check.
	cfg, err := config.Load(rootDir)
	switch {
	case errors.Is(err, config.ErrNotFound):
		lines = append(lines, "config:    aiwf.yaml not found (run `aiwf init`)")
		problems++
	case err != nil:
		lines = append(lines, "config:    "+err.Error())
		problems++
	case cfg.AiwfVersion != Version:
		lines = append(lines, fmt.Sprintf("config:    aiwf.yaml requests aiwf_version=%s, binary is %s", cfg.AiwfVersion, Version))
		problems++
	default:
		lines = append(lines, fmt.Sprintf("config:    ok (aiwf_version=%s, actor=%s)", cfg.AiwfVersion, cfg.Actor))
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
