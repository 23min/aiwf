package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCLAUDEMDFixture writes a synthetic CLAUDE.md whose "### CLI
// conventions" section (under "## Go conventions") is exactly cliConventionsBody.
func writeCLAUDEMDFixture(t *testing.T, root, cliConventionsBody string) {
	t.Helper()
	src := "# CLAUDE.md\n\n## Go conventions\n\n### CLI conventions\n\n" +
		cliConventionsBody +
		"\n\n### Commit conventions\n\n- irrelevant, and mentions neither opt-in nor ADR-0017\n"
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestPolicyCLAUDEMDCLIConventionsLogging_FiresOnStaleClaim proves the
// policy catches the exact M-0239/AC-5 regression: the section still
// carries the literal pre-ADR-0017 phrase ADR-0017's own Context
// section quotes as the thing it replaces. The fixture is otherwise
// fully compliant (opt-in wording + ADR-0017 link present) so this
// test isolates the stale-claim check specifically — a fixture that
// also failed the other two checks would prove nothing about this
// branch in particular.
func TestPolicyCLAUDEMDCLIConventionsLogging_FiresOnStaleClaim(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeCLAUDEMDFixture(t, root, "- **Output:** opt-in and default-off (see ADR-0017); `log/slog` → stderr; tool output → stdout.")
	violations, err := PolicyCLAUDEMDCLIConventionsLogging(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly one violation (the stale-claim check); got %d: %+v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Detail, "still describes the pre-ADR-0017 default") {
		t.Errorf("violation.Detail = %q, want it to name the stale-claim regression", violations[0].Detail)
	}
}

// TestPolicyCLAUDEMDCLIConventionsLogging_FiresOnMissingOptInDescription
// isolates the opt-in/default-off check: no stale phrase, ADR-0017 is
// linked, but the section never says logging is opt-in/default-off.
func TestPolicyCLAUDEMDCLIConventionsLogging_FiresOnMissingOptInDescription(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeCLAUDEMDFixture(t, root, "- **Output:** tool output → stdout. Diagnostic logging is described in ADR-0017.")
	violations, err := PolicyCLAUDEMDCLIConventionsLogging(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly one violation (the opt-in/default-off check); got %d: %+v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Detail, "opt-in/default-off") {
		t.Errorf("violation.Detail = %q, want it to name the missing opt-in/default-off description", violations[0].Detail)
	}
}

// TestPolicyCLAUDEMDCLIConventionsLogging_FiresOnMissingADRLink
// isolates the cross-link check: no stale phrase, opt-in/default-off
// is described, but ADR-0017 is never named.
func TestPolicyCLAUDEMDCLIConventionsLogging_FiresOnMissingADRLink(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeCLAUDEMDFixture(t, root, "- **Output:** tool output → stdout. Diagnostic logging is opt-in and default-off.")
	violations, err := PolicyCLAUDEMDCLIConventionsLogging(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly one violation (the ADR-0017 cross-link check); got %d: %+v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Detail, "does not cross-link ADR-0017") {
		t.Errorf("violation.Detail = %q, want it to name the missing ADR-0017 cross-link", violations[0].Detail)
	}
}

// TestPolicyCLAUDEMDCLIConventionsLogging_FiresOnMissingSection proves
// the policy also fires when the section itself can't be found —
// distinct from finding it but seeing stale content.
func TestPolicyCLAUDEMDCLIConventionsLogging_FiresOnMissingSection(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := `# CLAUDE.md

## Go conventions

### Something else entirely

- no CLI conventions heading here
`
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyCLAUDEMDCLIConventionsLogging(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected a violation when the CLI conventions section is missing; got none")
	}
}

// TestPolicyCLAUDEMDCLIConventionsLogging_UnreadableFile proves a
// generic read failure surfaces as a violation (never a panic or a
// silent pass) — mirrors envelope_structural_assertion's identical
// coverage of its own os.ReadFile error path.
func TestPolicyCLAUDEMDCLIConventionsLogging_UnreadableFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "CLAUDE.md")
	writeCLAUDEMDFixture(t, root, "- irrelevant")
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(path, 0o644) }() // restore so t.TempDir() cleanup can remove it
	violations, err := PolicyCLAUDEMDCLIConventionsLogging(root)
	if err != nil {
		t.Fatalf("policy returned a Go error instead of a violation: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected a violation on an unreadable CLAUDE.md; got none")
	}
}

// TestPolicyCLAUDEMDCLIConventionsLogging_AcceptsUpdatedShape proves
// the policy accepts a section carrying the shipped-behavior
// description and an ADR-0017 cross-link, and does not fire on
// prose in an unrelated section merely mentioning ADR-0017 or
// "opt-in" out of scope.
func TestPolicyCLAUDEMDCLIConventionsLogging_AcceptsUpdatedShape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeCLAUDEMDFixture(t, root, "- **Output:** human-readable by default; tool output stdout. Diagnostic\n  logging is opt-in and default-off; see ADR-0017.")
	violations, err := PolicyCLAUDEMDCLIConventionsLogging(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("policy fired on the updated shape: %+v", violations)
	}
}

// TestExtractMarkdownSubsection_TerminatesOnNextParentHeading,
// TestExtractMarkdownSubsection_TerminatesOnTopLevelHeading, and
// TestExtractMarkdownSubsection_RunsToEOF exercise the three
// termination paths every fixture above never reaches (they always
// follow "### CLI conventions" with a sibling "### " heading): the
// child section ending because a new "## " section starts, because a
// "# " top-level heading starts, or because the document simply ends
// while still inside the child section.

func TestExtractMarkdownSubsection_TerminatesOnNextParentHeading(t *testing.T) {
	t.Parallel()
	doc := "## Go conventions\n\n### CLI conventions\n\nbody line\n\n## Another top section\n\nunrelated\n"
	got, found := extractMarkdownSubsection(doc, "Go conventions", "CLI conventions")
	if !found {
		t.Fatal("expected found=true")
	}
	if got != "body line" {
		t.Errorf("got %q, want %q", got, "body line")
	}
}

func TestExtractMarkdownSubsection_TerminatesOnTopLevelHeading(t *testing.T) {
	t.Parallel()
	doc := "## Go conventions\n\n### CLI conventions\n\nbody line\n\n# Top-level heading\n\nunrelated\n"
	got, found := extractMarkdownSubsection(doc, "Go conventions", "CLI conventions")
	if !found {
		t.Fatal("expected found=true")
	}
	if got != "body line" {
		t.Errorf("got %q, want %q", got, "body line")
	}
}

func TestExtractMarkdownSubsection_RunsToEOF(t *testing.T) {
	t.Parallel()
	doc := "## Go conventions\n\n### CLI conventions\n\nbody line\n"
	got, found := extractMarkdownSubsection(doc, "Go conventions", "CLI conventions")
	if !found {
		t.Fatal("expected found=true")
	}
	if got != "body line" {
		t.Errorf("got %q, want %q", got, "body line")
	}
}

// TestExtractMarkdownSubsection_IgnoresChildUnderWrongParent proves
// the parent-scoping actually matters: a "### CLI conventions"
// heading nested under an unrelated "## " parent must not match —
// caught this exact gap during wf-vacuity: a mutation that dropped
// the parent check entirely survived every other test in this file,
// since none of them put the child heading under the wrong parent.
func TestExtractMarkdownSubsection_IgnoresChildUnderWrongParent(t *testing.T) {
	t.Parallel()
	doc := "## Some other topic\n\n### CLI conventions\n\nwrong-parent body\n\n## Go conventions\n\nno CLI conventions heading here\n"
	_, found := extractMarkdownSubsection(doc, "Go conventions", "CLI conventions")
	if found {
		t.Error("expected found=false for a CLI conventions heading nested under the wrong parent")
	}
}
