package cliutil

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/23min/aiwf/internal/logger"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/skills"
	"github.com/23min/aiwf/internal/version"
)

// StatuslineOpts carries the flags for the statusline scaffold +
// consent-gated settings wiring flow.
type StatuslineOpts struct {
	RootDir      string
	Scope        string
	WireSettings bool
	FormatJSON   bool

	// AllowUntagged bypasses the G-0367 version-confirmation gate: when
	// the running binary's version is untagged (a dev/worktree build),
	// the explicit scaffold write requires either this flag or an
	// interactive [y/N] confirmation, since it lands unconditionally in
	// the shared, cross-project scope with no version marker
	// distinguishing it as non-release.
	AllowUntagged bool
}

// RunStatuslineScaffold invokes the shared scaffold-if-absent helper
// in skills/ and, when consent is given, wires the statusLine key
// into the scope-appropriate settings file (M-0156). Resolves the
// running binary's version via version.Current(); RunStatuslineScaffoldForVersion
// is the testable core with the version injected.
//
// Returns one of the Exit* codes.
func RunStatuslineScaffold(opts StatuslineOpts) int {
	return RunStatuslineScaffoldForVersion(opts, version.Current())
}

// RunStatuslineScaffoldForVersion is RunStatuslineScaffold with the
// running binary's version injected — the testable core, so tests can
// drive both the G-0367 version gate and the ADR-0015 settings-consent
// flow without depending on `go test`'s own (always untagged) binary
// version.
//
// Consent model:
//   - G-0367 version gate (script write): binary untagged (per
//     skills.StatuslineWriteNeedsConfirmation) and --allow-untagged-statusline
//     not given → TTY present and not --format=json prompts [y/N];
//     otherwise refuses (ExitOK, no write, explains the override).
//   - ADR-0015 (settings wiring, unchanged): --wire-settings flag →
//     write unconditionally (non-TTY consent); TTY present and not
//     --format=json → interactive [y/N] prompt; otherwise skip write,
//     emit snippet.
//
// Returns one of the Exit* codes.
func RunStatuslineScaffoldForVersion(opts StatuslineOpts, binary version.Info) (code int) {
	defer emitVerbCompletedIfOK(&code, "statusline-scaffold", opts.RootDir)

	if skills.StatuslineWriteNeedsConfirmation(binary) && !opts.AllowUntagged {
		confirmed := !opts.FormatJSON && render.IsTTY(os.Stdin) &&
			promptYN(fmt.Sprintf("Binary version %q is untagged (dev/worktree build) — write its statusline script into the shared scope anyway?", binary.Version))
		if !confirmed {
			Printf("aiwf --statusline: binary version %q is untagged — refusing to write without confirmation (re-run with --allow-untagged-statusline, or confirm interactively, to proceed anyway)\n", binary.Version)
			return ExitOK
		}
	}

	sc := skills.StatuslineScope(opts.Scope)
	res, err := skills.ScaffoldStatusline(opts.RootDir, sc)
	if err != nil {
		Errorf("aiwf --statusline: %v\n", err)
		return ExitUsage
	}
	if res.Wrote {
		Printf("\naiwf --statusline: wrote %s\n", res.Path)
	} else {
		Printf("\naiwf --statusline: %s already current, left untouched\n", res.Path)
	}
	if res.GitignoreAppended {
		Println("aiwf --statusline: appended `.claude/statusline.sh` to .gitignore")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		Errorf("aiwf --statusline: resolving home: %v\n", err)
		return ExitInternal
	}

	settingsPath, err := skills.SettingsPathForScope(opts.RootDir, home, sc)
	if err != nil {
		Errorf("aiwf --statusline: %v\n", err)
		return ExitUsage
	}

	cmdPath := res.Command

	consent := opts.WireSettings
	if !consent && !opts.FormatJSON && render.IsTTY(os.Stdin) {
		consent = promptYN(fmt.Sprintf("Wire statusLine into %s?", settingsPath))
	}

	if !consent {
		Println("\nTo activate, add this to your Claude Code settings file:")
		Println()
		Println(res.Snippet)
		return ExitOK
	}

	wr, err := skills.WireStatuslineSettings(settingsPath, cmdPath)
	if err != nil {
		Errorf("aiwf --statusline: %v\n", err)
		return ExitInternal
	}
	if wr.Idempotent {
		Printf("\naiwf --statusline: %s already contains the matching statusLine key; nothing to do.\n", settingsPath)
		return ExitOK
	}
	if !wr.Wrote {
		Printf("\naiwf --statusline: %s already contains a statusLine key:\n  %s\n", settingsPath, wr.ExistingValue)
		Println("To use the aiwf statusline instead, replace the existing statusLine value with:")
		Println()
		Println(res.Snippet)
		return ExitFindings
	}
	if wr.BackupPath != "" {
		Printf("aiwf --statusline: backed up %s to %s\n", settingsPath, wr.BackupPath)
	}
	Printf("aiwf --statusline: wired statusLine into %s\n", settingsPath)
	return ExitOK
}

// emitVerbCompletedIfOK fires a "verb.completed" diagnostic event
// through the WithVerb-bound logger when *code is ExitOK, deferred so
// a many-return-point function emits exactly once regardless of which
// branch returned. actor is always empty: none of statusline's flows
// have an --actor flag.
func emitVerbCompletedIfOK(code *int, verbName, entity string) {
	if *code != ExitOK {
		return
	}
	diagLog, closeDiagLog := ResolveLogger(os.Getenv)
	defer func() { _ = closeDiagLog() }()
	logger.WithVerb(diagLog, verbName, entity, "").Info("verb.completed")
}

// StatuslineRemoveOpts carries the flags for `aiwf update --remove`.
type StatuslineRemoveOpts struct {
	RootDir string
	Scope   string
	Force   bool
}

// RunStatuslineRemove deletes a scope's aiwf-managed statusline script
// and strips its statusLine settings key (G-0354). The two artifacts
// are inspected read-only first, and the refuse-vs-proceed decision is
// made from that inspection alone, BEFORE either mutation runs: if
// either the script or the settings key exists and does not look
// aiwf-authored (the script carries the version marker; the settings
// key's command matches what aiwf itself would have written for this
// scope) and --force was not given, the call refuses and mutates
// NEITHER artifact — a mixed case (one aiwf-authored, one foreign) must
// not silently tear down the aiwf-owned half while reporting a refusal
// for the other. Nothing to remove at the target scope is a no-op, not
// an error.
//
// Returns one of the Exit* codes.
func RunStatuslineRemove(opts StatuslineRemoveOpts) (code int) {
	defer emitVerbCompletedIfOK(&code, "statusline-remove", opts.RootDir)

	sc := skills.StatuslineScope(opts.Scope)

	home, err := os.UserHomeDir()
	if err != nil {
		Errorf("aiwf update --remove: resolving home: %v\n", err) //coverage:ignore os.UserHomeDir fails only when $HOME is unset; not reproducible in the test env (mirrors RunStatuslineScaffold's sibling)
		return ExitInternal
	}

	dest, cmdPath, err := skills.StatuslineDestForScope(opts.RootDir, home, sc)
	if err != nil {
		Errorf("aiwf update --remove: %v\n", err)
		return ExitUsage
	}
	settingsPath, err := skills.SettingsPathForScope(opts.RootDir, home, sc)
	if err != nil { //coverage:ignore unreachable: StatuslineDestForScope above validates the same closed set of scopes and returns first, so an invalid scope never reaches this call
		Errorf("aiwf update --remove: %v\n", err)
		return ExitUsage
	}

	// Phase 1: read-only inspection of BOTH artifacts. Neither mutates
	// anything, so the refuse-vs-proceed decision below sees the full
	// picture before any deletion happens.
	scriptExisted, scriptAiwfAuthored, err := skills.StatuslineScriptStatus(dest)
	if err != nil {
		Errorf("aiwf update --remove: %v\n", err)
		return ExitInternal
	}
	settingsExisted, settingsMatches, settingsExistingValue, err := skills.StatuslineSettingsKeyStatus(settingsPath, cmdPath)
	if err != nil {
		Errorf("aiwf update --remove: %v\n", err)
		return ExitInternal
	}

	scriptForeign := scriptExisted && !scriptAiwfAuthored && !opts.Force
	settingsForeign := settingsExisted && !settingsMatches && !opts.Force

	// Phase 2: refuse-vs-proceed, decided entirely from phase 1's
	// read-only findings. On refusal, return here — before either act
	// call below runs — so a foreign settingsLine key never lets a
	// same-invocation aiwf-owned script get deleted anyway, or vice
	// versa (G-0354 review finding: partial, silently-mutating refusal).
	if scriptForeign || settingsForeign {
		if scriptForeign {
			Errorf("aiwf update --remove: %s does not look aiwf-authored (no `# aiwf-statusline version:` marker) — refusing to delete; re-run with --force to remove it anyway\n", dest)
		}
		if settingsForeign {
			Errorf("aiwf update --remove: %s statusLine key does not match aiwf's own wiring for this scope:\n  %s\nrefusing to strip it — re-run with --force to remove it anyway\n", settingsPath, settingsExistingValue)
		}
		return ExitFindings
	}

	// Phase 3: both artifacts are authorized (aiwf-authored, or --force)
	// — now actually mutate.
	scriptRemoved := false
	if scriptExisted {
		scriptRemoved, err = skills.RemoveStatuslineScriptFile(dest)
		if err != nil { //coverage:ignore RemoveStatuslineScriptFile errors only on a non-ENOENT os.Remove fault (TOCTOU race or filesystem fault) after StatuslineScriptStatus just confirmed the file is readable; not reproducible in a tempdir test
			Errorf("aiwf update --remove: %v\n", err)
			return ExitInternal
		}
	}
	settingsRemoved := false
	if settingsExisted {
		settingsRemoved, err = skills.RemoveStatuslineSettingsKey(settingsPath)
		if err != nil { //coverage:ignore RemoveStatuslineSettingsKey errors only via the same TOCTOU/filesystem-fault class as its own coverage:ignore'd internals, moments after StatuslineSettingsKeyStatus just read the same file successfully; not reproducible in a tempdir test
			Errorf("aiwf update --remove: %v\n", err)
			return ExitInternal
		}
	}

	if !scriptExisted && !settingsExisted {
		Printf("aiwf update --remove: nothing to remove at %s scope (no script, no statusLine key)\n", sc)
		return ExitOK
	}

	if scriptRemoved {
		Printf("aiwf update --remove: deleted %s\n", dest)
	}
	if settingsRemoved {
		Printf("aiwf update --remove: stripped statusLine key from %s\n", settingsPath)
	}
	return ExitOK
}

// promptYN prints prompt + " [y/N] " to stderr and reads one line
// from stdin. Returns true only on "y" or "yes" (case-insensitive).
func promptYN(prompt string) bool {
	Errorf("\n%s [y/N] ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return ans == "y" || ans == "yes"
}
