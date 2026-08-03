package check

// M-0289 AC-1: the width rule over repo-facing docs.
//
// The polarity is the inverse of skill-body-id's: a real id is LEGITIMATE
// here, because these docs are read in the repo that owns the ids. Only the
// width is the defect. That makes the two rules siblings rather than modes of
// one another, and it is why this file asserts the canonical arm as hard as
// the narrow one — a rule that fired on every id would make the corpus
// unwritable.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// TestScanDocIDWidth_NonNumericSuffixSilent pins this rule's scope limit. A
// suffix that is neither digits nor N's is a malformed id shape, and in a
// document nothing reports it — body-prose-id walks entities, which documents
// are not. The silence is deliberate and this test is where it is recorded, so
// that widening the rule later is a decision rather than a surprise.
func TestScanDocIDWidth_NonNumericSuffixSilent(t *testing.T) {
	t.Parallel()
	for _, tok := range []string{"M-abc", "G-XYZ", "E-a1"} {
		t.Run(tok, func(t *testing.T) {
			t.Parallel()
			body := fmt.Sprintf("# Doc\n\nSee %s.\n", tok)
			if got := ScanDocIDWidth([]byte(body), "README.md"); len(got) != 0 {
				t.Fatalf("malformed shape %q belongs to body-prose-id, got %d: %+v", tok, len(got), got)
			}
		})
	}
}

// TestDocIDWidthReference_ReadsConfiguredPaths covers the walker: a declared
// doc is scanned and its findings carry the repo-relative path an operator
// can act on.
func TestDocIDWidthReference_ReadsConfiguredPaths(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# R\n\nSee E-01.\n"), 0o644); err != nil {
		t.Fatalf("seeding doc: %v", err)
	}
	got := DocIDWidthReference(&tree.Tree{Root: root}, []string{"README.md"})
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].Path != "README.md" {
		t.Errorf("path = %q, want the repo-relative %q", got[0].Path, "README.md")
	}
}

// TestDocIDWidthReference_MissingPathIsSilent guards the default. README.md is
// scanned in every repo whether or not one exists, so a missing file must be a
// skip — reporting it would turn the default into a nag for repos the rule has
// nothing to say about.
func TestDocIDWidthReference_MissingPathIsSilent(t *testing.T) {
	t.Parallel()
	got := DocIDWidthReference(&tree.Tree{Root: t.TempDir()}, []string{"README.md", "docs/nope.md"})
	if len(got) != 0 {
		t.Fatalf("missing docs must be silent, got %d: %+v", len(got), got)
	}
}

// TestDocIDWidthReference_RejectsPathEscapingRoot is the containment guard.
// The path list is operator-supplied config, so without this a checked-in
// aiwf.yaml could aim the scan at any file readable by whoever runs the check
// — and the finding message quotes what it found.
func TestDocIDWidthReference_RejectsPathEscapingRoot(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("making root: %v", err)
	}
	outside := filepath.Join(filepath.Dir(root), "secret.md")
	if err := os.WriteFile(outside, []byte("# S\n\nSee E-01.\n"), 0o644); err != nil {
		t.Fatalf("seeding outside file: %v", err)
	}
	got := DocIDWidthReference(&tree.Tree{Root: root}, []string{"../secret.md"})
	if len(got) != 0 {
		t.Fatalf("a path escaping the root must be skipped, got %d: %+v", len(got), got)
	}
}

// TestDocIDWidthReference_RejectsSymlinkEscapingRoot is the other half of the
// containment guard. The threat the guard names is a checked-in aiwf.yaml
// aiming the scan at arbitrary files — and an actor who can commit that can
// commit a symlink, so a lexical check alone does not constrain them. The
// finding quotes what it reads, which is what makes the escape worth blocking.
func TestDocIDWidthReference_RejectsSymlinkEscapingRoot(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("making root: %v", err)
	}
	outside := filepath.Join(tmp, "secret.md")
	if err := os.WriteFile(outside, []byte("# S\n\nSee E-01.\n"), 0o644); err != nil {
		t.Fatalf("seeding outside file: %v", err)
	}
	link := filepath.Join(root, "inside-link.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if got := DocIDWidthReference(&tree.Tree{Root: root}, []string{"inside-link.md"}); len(got) != 0 {
		t.Fatalf("a symlink leaving the root must be skipped, got %d: %+v", len(got), got)
	}
}

// docKindPrefixes is every kind prefix the id grammar admits. Driving the
// table from this list rather than a sample is what makes "for each kind
// prefix" true as the grammar changes rather than true when it was written.
var docKindPrefixes = []string{"E", "M", "G", "D", "C", "ADR"}

// TestScanDocIDWidth_NarrowNumericFires is the core claim: a real id written
// at a legacy width is the defect, for every kind. The finding names the line
// so an operator can go straight to it.
func TestScanDocIDWidth_NarrowNumericFires(t *testing.T) {
	t.Parallel()
	for _, k := range docKindPrefixes {
		for _, digits := range []string{"1", "01", "001"} {
			tok := k + "-" + digits
			t.Run("narrow/"+tok, func(t *testing.T) {
				t.Parallel()
				body := fmt.Sprintf("# Doc\n\nRun `aiwf show %s` to inspect it.\n", tok)
				got := ScanDocIDWidth([]byte(body), "README.md")
				if len(got) != 1 {
					t.Fatalf("want 1 finding for %q, got %d: %+v", tok, len(got), got)
				}
				if got[0].Code != CodeDocIDWidth {
					t.Errorf("code = %q, want %q", got[0].Code, CodeDocIDWidth)
				}
				if got[0].Severity != SeverityWarning {
					t.Errorf("severity = %q, want %q — the rule ships advisory and is escalated by config, never the reverse", got[0].Severity, SeverityWarning)
				}
				if got[0].Line != 3 {
					t.Errorf("line = %d, want 3", got[0].Line)
				}
				if !strings.Contains(got[0].Message, tok) {
					t.Errorf("message %q does not name the offending token", got[0].Message)
				}
			})
		}
	}
}

// TestScanDocIDWidth_CanonicalSilent guards the arm that makes the corpus
// writable at all. Both shapes a doc may legitimately carry stay silent: a
// real id at canonical width, and the canonical letter-N placeholder.
func TestScanDocIDWidth_CanonicalSilent(t *testing.T) {
	t.Parallel()
	for _, k := range docKindPrefixes {
		for _, suffix := range []string{"0001", "NNNN"} {
			tok := k + "-" + suffix
			t.Run("canonical/"+tok, func(t *testing.T) {
				t.Parallel()
				body := fmt.Sprintf("# Doc\n\nRun `aiwf show %s` to inspect it.\n", tok)
				if got := ScanDocIDWidth([]byte(body), "README.md"); len(got) != 0 {
					t.Fatalf("canonical %q must be silent, got %d: %+v", tok, len(got), got)
				}
			})
		}
	}
}

// TestScanDocIDWidth_WiderThanCanonicalSilent pins the boundary on the far
// side. CanonicalPad is a MINIMUM, not a fixed width: a tree that allocates
// past 9999 emits five digits legitimately, so a rule keyed on "not exactly
// four" would fire on correct ids in exactly the trees that have the most.
func TestScanDocIDWidth_WiderThanCanonicalSilent(t *testing.T) {
	t.Parallel()
	for _, tok := range []string{"M-00001", "E-123456", "ADR-99999"} {
		t.Run(tok, func(t *testing.T) {
			t.Parallel()
			body := fmt.Sprintf("# Doc\n\nSee %s.\n", tok)
			if got := ScanDocIDWidth([]byte(body), "README.md"); len(got) != 0 {
				t.Fatalf("wider-than-canonical %q must be silent, got %d: %+v", tok, len(got), got)
			}
		})
	}
}

// TestScanDocIDWidth_NarrowPlaceholderFires covers the other half of the
// population: a letter-N placeholder teaches a width directly, so a narrow one
// models a shape no allocator emits.
func TestScanDocIDWidth_NarrowPlaceholderFires(t *testing.T) {
	t.Parallel()
	for _, tok := range []string{"E-NN", "M-NNN", "G-NNN", "D-NNN", "C-NNN", "ADR-NNN"} {
		t.Run(tok, func(t *testing.T) {
			t.Parallel()
			body := fmt.Sprintf("# Doc\n\nAllocate the next %s id.\n", tok)
			got := ScanDocIDWidth([]byte(body), "README.md")
			if len(got) != 1 {
				t.Fatalf("want 1 finding for narrow placeholder %q, got %d: %+v", tok, len(got), got)
			}
			if got[0].Severity != SeverityWarning {
				t.Errorf("severity = %q, want %q", got[0].Severity, SeverityWarning)
			}
		})
	}
}

// TestScanDocIDWidth_CodeIsInScope is the divergence from body-prose-id, and
// the reason this rule cannot be a subcode of it. Measured over the real
// corpus, the debris lives overwhelmingly inside command examples and fenced
// blocks; a mask that exempts code would see almost none of it and would miss
// README entirely. A reader copies a command example at least as readily as a
// sentence, so backticks cannot be an opt-out.
func TestScanDocIDWidth_CodeIsInScope(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
	}{
		{"inline-span", "# Doc\n\nThe frontmatter carries `parent: E-01` here.\n"},
		{"fenced-block", "# Doc\n\n```bash\naiwf promote M-001 in_progress\n```\n"},
		{"indented-comment", "# Doc\n\n```bash\naiwf add epic --title \"x\"   # → E-01\n```\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ScanDocIDWidth([]byte(tc.body), "README.md")
			if len(got) != 1 {
				t.Fatalf("want 1 finding (code is in scope), got %d: %+v", len(got), got)
			}
		})
	}
}

// TestScanDocIDWidth_CompositeParentWidth covers the composite form, whose
// parent segment carries the width. The AC segment is a single digit by
// grammar and is not a width claim.
func TestScanDocIDWidth_CompositeParentWidth(t *testing.T) {
	t.Parallel()
	if got := ScanDocIDWidth([]byte("# Doc\n\nAddress `M-001/AC-1` next.\n"), "README.md"); len(got) != 1 {
		t.Fatalf("narrow composite parent must fire, got %d: %+v", len(got), got)
	}
	if got := ScanDocIDWidth([]byte("# Doc\n\nAddress `M-0001/AC-1` next.\n"), "README.md"); len(got) != 0 {
		t.Fatalf("canonical composite must be silent, got %d: %+v", len(got), got)
	}
}

// TestScanDocIDWidth_DedupesPerToken mirrors the sibling rules: a token
// repeated in one file produces one finding, not one per occurrence, so a
// tutorial that says E-01 nine times yields a worklist rather than a wall.
func TestScanDocIDWidth_DedupesPerToken(t *testing.T) {
	t.Parallel()
	body := "# Doc\n\nE-01 then E-01 again.\n\nAnd E-01 once more, plus M-001.\n"
	got := ScanDocIDWidth([]byte(body), "README.md")
	if len(got) != 2 {
		t.Fatalf("want 2 findings (one per distinct token), got %d: %+v", len(got), got)
	}
}

// TestApplyDocsStrict is the severity contract. The rule ships advisory
// so that upgrading aiwf cannot block a push over prose the operator never
// edited — a repo whose entities were migrated still carries narrow ids
// through its docs, and there is neither a fixer nor a suppression mechanism
// for them. A repo raises the bar once its own sweep is done.
func TestApplyDocsStrict(t *testing.T) {
	t.Parallel()
	t.Run("strict escalates", func(t *testing.T) {
		t.Parallel()
		f := []Finding{{Code: CodeDocIDWidth, Severity: SeverityWarning}}
		ApplyDocsStrict(f, true)
		if f[0].Severity != SeverityError {
			t.Errorf("severity = %q, want %q under strict", f[0].Severity, SeverityError)
		}
	})
	t.Run("default leaves warning", func(t *testing.T) {
		t.Parallel()
		f := []Finding{{Code: CodeDocIDWidth, Severity: SeverityWarning}}
		ApplyDocsStrict(f, false)
		if f[0].Severity != SeverityWarning {
			t.Errorf("severity = %q, want %q with strict off", f[0].Severity, SeverityWarning)
		}
	})
	t.Run("scoped to the doc codes", func(t *testing.T) {
		t.Parallel()
		f := []Finding{{Code: CodeBodyProseID, Severity: SeverityWarning}}
		ApplyDocsStrict(f, true)
		if f[0].Severity != SeverityWarning {
			t.Errorf("escalated an unrelated code %q — the bumper must be scoped", f[0].Code)
		}
	})
	// Both doc rules share the corpus and therefore the knob; escalating one
	// without the other would leave a repo that opted in half-guarded.
	t.Run("covers doc-id-slug too", func(t *testing.T) {
		t.Parallel()
		f := []Finding{{Code: CodeDocIDSlug, Severity: SeverityWarning}}
		ApplyDocsStrict(f, true)
		if f[0].Severity != SeverityError {
			t.Errorf("severity = %q, want %q — docs.strict governs every doc rule", f[0].Severity, SeverityError)
		}
	})
}
