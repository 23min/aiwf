package policies

// `adr.md` tells its author what an ADR is for. Both spec templates carry the
// same kind of instruction in a preamble — the region between the frontmatter
// and the first section heading — routing the reasoning behind a choice to an
// ADR or a decision record rather than into the spec, where a builder reads it
// as requirement (G-0592).
//
// WHAT IS PINNED HERE, AND WHAT IS NOT.
//
// Pinned: the preamble region exists, and it is an HTML comment. Both are
// structural. Deleting the genre instruction removes the region; moving it below
// a section heading removes it from the region; writing it as visible prose
// instead of a comment changes what the region is — and that last one matters,
// because the comment is what does not survive into the entity an author writes
// from this template.
//
// Not pinned: that the preamble still says the right thing. That is content
// correctness, and D-0050 records the measurement behind holding it at review
// instead — a phrase-content assertion pins a reading rather than a rule, and
// the negator that makes a rule binding can sit outside the asserted span, so
// an inverted instruction passes a `Contains` check unchanged. The same applies
// to the `## Context` prompts this change rewords and to the matching bullets in
// `aiwfx-plan-epic` / `aiwfx-plan-milestones`: nothing here asserts they stay
// free of a request for justification. Held at review, per the disposition D5
// prescribes for what cannot be pinned.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// specTemplates are the shipped entity templates whose genre is "states what
// will be built". `adr.md` and `decision.md` are excluded deliberately —
// carrying the reasoning is what those two are for.
var specTemplates = []string{
	"epic-spec.md",
	"milestone-spec.md",
}

func TestSpecTemplates_OpenWithACommentedPreamble(t *testing.T) {
	t.Parallel()
	for _, name := range specTemplates {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			preamble := templatePreamble(readSpecTemplateBody(t, name))
			if preamble == "" {
				t.Fatal("nothing between the frontmatter and the first `## ` heading: the genre instruction has nowhere to live, and an author meets the section prompts with no statement of what the document is")
			}
			singleComment := strings.HasPrefix(preamble, "<!--") && strings.HasSuffix(preamble, "-->") &&
				strings.Count(preamble, "<!--") == 1 && strings.Count(preamble, "-->") == 1
			if !singleComment {
				t.Errorf("preamble is not a single HTML comment, so part of it survives into every entity written from this template:\n%s", preamble)
			}
		})
	}
}

// readSpecTemplateBody returns the post-frontmatter body of a shipped entity
// template. The frontmatter is dropped because the region asserted over is body
// prose; a trailing `#` comment on a frontmatter field would otherwise read as
// preamble.
func readSpecTemplateBody(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(repoRoot(t), "internal", "skills", "embedded-rituals",
		"plugins", "aiwf-extensions", "templates", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading shipped template %s: %v", name, err)
	}
	if _, b, ok := entity.Split(raw); ok {
		return string(b)
	}
	return string(raw)
}

// templatePreamble returns the region between the frontmatter and the first
// level-2 heading, empty when the body opens straight onto a section or carries
// no level-2 heading at all. Fenced blocks are skipped so a `## ` inside an
// example neither terminates the region early nor stands in for the real
// heading, matching extractMarkdownSection's walk.
func templatePreamble(body string) string {
	var preamble []string
	inFence := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
		}
		if !inFence && strings.HasPrefix(line, "## ") {
			return strings.TrimSpace(strings.Join(preamble, "\n"))
		}
		preamble = append(preamble, line)
	}
	// No level-2 heading: the file is not a spec template shape, and the whole
	// body is not a preamble.
	return ""
}
