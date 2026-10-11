package policies

import (
	"testing"
)

// TestExtractMarkdownSubsection_TerminatesOnNextParentHeading,
// TestExtractMarkdownSubsection_TerminatesOnTopLevelHeading, and
// TestExtractMarkdownSubsection_RunsToEOF exercise how a child section
// ends besides a sibling "### " heading: a new "## " section starts, a
// "# " top-level heading starts, or the document ends while still inside
// the child section.

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
// heading nested under an unrelated "## " parent must not match.
func TestExtractMarkdownSubsection_IgnoresChildUnderWrongParent(t *testing.T) {
	t.Parallel()
	doc := "## Some other topic\n\n### CLI conventions\n\nwrong-parent body\n\n## Go conventions\n\nno CLI conventions heading here\n"
	_, found := extractMarkdownSubsection(doc, "Go conventions", "CLI conventions")
	if found {
		t.Error("expected found=false for a CLI conventions heading nested under the wrong parent")
	}
}

// TestExtractMarkdownSubsection_TopLevelHeadingClosesTheParent proves a
// `# ` heading ends the parent section as well as the child: a `### child`
// after it no longer sits inside `## parent`, so it is not found.
func TestExtractMarkdownSubsection_TopLevelHeadingClosesTheParent(t *testing.T) {
	t.Parallel()
	doc := "## Go conventions\n\n# Another document\n\n### CLI conventions\n\norphaned body\n"
	if got, ok := extractMarkdownSubsection(doc, "Go conventions", "CLI conventions"); ok {
		t.Errorf("found %q under a parent a top-level heading closed; want not found", got)
	}
}
