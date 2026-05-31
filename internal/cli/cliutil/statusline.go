package cliutil

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/skills"
)

// StatuslineOpts carries the flags for the statusline scaffold +
// consent-gated settings wiring flow.
type StatuslineOpts struct {
	RootDir      string
	Scope        string
	WireSettings bool
	FormatJSON   bool
}

// RunStatuslineScaffold invokes the shared scaffold-if-absent helper
// in skills/ and, when consent is given, wires the statusLine key
// into the scope-appropriate settings file (M-0156).
//
// Consent model (per ADR-0015):
//   - --wire-settings flag → write unconditionally (non-TTY consent)
//   - TTY present and not --format=json → interactive [y/N] prompt
//   - Otherwise (no TTY, or --format=json) → skip write, emit snippet
//
// Returns one of the Exit* codes.
func RunStatuslineScaffold(opts StatuslineOpts) int {
	sc := skills.StatuslineScope(opts.Scope)
	res, err := skills.ScaffoldStatusline(opts.RootDir, sc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf --statusline: %v\n", err)
		return ExitUsage
	}
	if res.Wrote {
		fmt.Printf("\naiwf --statusline: wrote %s\n", res.Path)
	} else {
		fmt.Printf("\naiwf --statusline: %s already exists, left untouched\n", res.Path)
	}
	if res.GitignoreAppended {
		fmt.Println("aiwf --statusline: appended `.claude/statusline.sh` to .gitignore")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf --statusline: resolving home: %v\n", err)
		return ExitInternal
	}

	settingsPath, err := skills.SettingsPathForScope(opts.RootDir, home, sc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf --statusline: %v\n", err)
		return ExitUsage
	}

	cmdPath := statuslineCmdPath(res)

	consent := opts.WireSettings
	if !consent && !opts.FormatJSON && render.IsTTY(os.Stdin) {
		consent = promptYN(fmt.Sprintf("Wire statusLine into %s?", settingsPath))
	}

	if !consent {
		fmt.Println("\nTo activate, add this to your Claude Code settings file:")
		fmt.Println()
		fmt.Println(res.Snippet)
		return ExitOK
	}

	wr, err := skills.WireStatuslineSettings(settingsPath, cmdPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf --statusline: %v\n", err)
		return ExitInternal
	}
	if wr.Idempotent {
		fmt.Printf("\naiwf --statusline: %s already contains the matching statusLine key; nothing to do.\n", settingsPath)
		return ExitOK
	}
	if !wr.Wrote {
		fmt.Printf("\naiwf --statusline: %s already contains a statusLine key:\n  %s\n", settingsPath, wr.ExistingValue)
		fmt.Println("To use the aiwf statusline instead, replace the existing statusLine value with:")
		fmt.Println()
		fmt.Println(res.Snippet)
		return ExitFindings
	}
	if wr.BackupPath != "" {
		fmt.Printf("aiwf --statusline: backed up %s to %s\n", settingsPath, wr.BackupPath)
	}
	fmt.Printf("aiwf --statusline: wired statusLine into %s\n", settingsPath)
	return ExitOK
}

// statuslineCmdPath extracts the command path from the scaffold
// result's snippet. The snippet has the shape:
//
//	"statusLine": {
//	  "type": "command",
//	  "command": "<path>"
//	}
//
// We parse the command value out. For project scope it's relative
// (`.claude/statusline.sh`); for user scope it's absolute.
func statuslineCmdPath(res skills.StatuslineScaffoldResult) string {
	for _, line := range strings.Split(res.Snippet, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `"command"`) {
			// "command": "<path>"
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				v := strings.TrimSpace(parts[1])
				v = strings.TrimSuffix(v, ",")
				v = strings.Trim(v, `"`)
				return v
			}
		}
	}
	return ".claude/statusline.sh"
}

// promptYN prints prompt + " [y/N] " to stderr and reads one line
// from stdin. Returns true only on "y" or "yes" (case-insensitive).
func promptYN(prompt string) bool {
	fmt.Fprintf(os.Stderr, "\n%s [y/N] ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return ans == "y" || ans == "yes"
}
