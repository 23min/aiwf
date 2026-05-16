package main

import (
	"strings"
	"testing"
)

// TestBinary_ArchiveHelp pins M-0088/AC-7: `aiwf archive --help`
// is complete — usage line, every flag listed with a description,
// and worked examples for the three operator-facing call sites
// (dry-run, --apply, --kind).
//
// Binary-level integration per CLAUDE.md §"Test the seam, not just
// the layer": Cobra's help output composes the `Use:` line, `Short:`,
// `Long:`, the flag table, and `Example:` — testing the verb's
// definition struct in isolation would miss any future refactor that
// flips the verb to a non-runnable parent or drops the Long/Example.
// Running the built binary as a subprocess exercises the same path
// an operator's tab-completion-driven `--help` lookup follows.
func TestBinary_ArchiveHelp(t *testing.T) {
	t.Parallel()
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp /* no ldflags */)

	out, err := runBinary(bin, "archive", "--help")
	if err != nil {
		t.Fatalf("aiwf archive --help failed: %v\noutput:\n%s", err, out)
	}

	// Usage line: Cobra renders "Usage:\n  aiwf archive ...". The
	// canonical use-string from newArchiveCmd is "archive [--apply]
	// [--kind <kind>]"; assert the verb name and shape appear.
	if !strings.Contains(out, "Usage:") {
		t.Errorf("--help output missing Usage: header:\n%s", out)
	}
	if !strings.Contains(out, "aiwf archive") {
		t.Errorf("--help Usage line must show `aiwf archive`:\n%s", out)
	}

	// Every operator-facing flag the verb declares must appear in
	// the flag table with a description. The four flags are --apply,
	// --kind, --actor, --principal (--root is included for parity
	// with the other verbs). We assert presence by flag name; Cobra's
	// rendering always emits "--<name>" + the description on the
	// same line.
	requiredFlags := []string{"--apply", "--kind", "--actor", "--principal", "--root"}
	for _, flag := range requiredFlags {
		if !strings.Contains(out, flag) {
			t.Errorf("--help output missing flag %q:\n%s", flag, out)
		}
	}

	// Per-flag description sanity: --apply's description must name
	// the verb's destructive shape (i.e. that without the flag,
	// the verb is dry-run). The point of asserting the description
	// rather than just the flag name is that a flag declaration
	// without text would still show "--apply" but leave the operator
	// guessing what it does.
	if !strings.Contains(out, "dry-run") {
		t.Errorf("--help output must mention dry-run semantics (the --apply flag's purpose):\n%s", out)
	}

	// --kind's description must enumerate the closed set of kinds
	// it accepts. The set is documented in newArchiveCmd's flag
	// declaration as "epic, contract, gap, decision, adr".
	for _, kind := range []string{"epic", "contract", "gap", "decision", "adr"} {
		if !strings.Contains(out, kind) {
			t.Errorf("--help output must name kind %q (the --kind flag's accepted set):\n%s", kind, out)
		}
	}

	// Examples block: Cobra emits "Examples:" when the cobra.Command
	// carries an Example: field. The three required examples are
	// dry-run, --apply, and --kind. Assert presence of each example
	// invocation by its distinctive token.
	if !strings.Contains(out, "Examples:") {
		t.Errorf("--help output missing Examples: block — required by AC-7:\n%s", out)
	}
	exampleSnippets := []struct {
		description string
		needle      string
	}{
		{"dry-run example", "aiwf archive\n"}, // bare invocation
		{"--apply example", "aiwf archive --apply"},
		{"--kind example", "aiwf archive --apply --kind gap"},
	}
	for _, ex := range exampleSnippets {
		if !strings.Contains(out, ex.needle) {
			t.Errorf("--help Examples block missing %s (looking for %q):\n%s", ex.description, ex.needle, out)
		}
	}
}
