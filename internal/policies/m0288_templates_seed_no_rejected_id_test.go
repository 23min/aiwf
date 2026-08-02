package policies

// M-0288 AC-2: a shipped entity template is the body a consumer's entity starts
// life with, so any id-shaped token it seeds in prose is one that survives into
// that entity — where `body-prose-id` scans it and, for a letter-N placeholder,
// rejects it. The two rules disagree by design about which shape is correct
// (a shipped surface wants the placeholder, an entity body wants a resolved id),
// and a template is the one artifact both scan. It satisfies both only by
// carrying no id-shaped token its prose exposes.
//
// The resolution index is empty on purpose. A consumer's tree resolves none of
// aiwf's own ids, so a template that passes only against aiwf's tree is not
// clean where it actually ships.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// shippedTemplates is every template `aiwf init` / `aiwf update` materializes
// into a consumer's .claude/templates/. Held as a class rather than as the
// subset carrying debris today, so a clean one cannot regress unnoticed.
var shippedTemplates = []string{
	"adr.md",
	"decision.md",
	"epic-spec.md",
	"milestone-spec.md",
}

func TestM0288_AC2_ShippedTemplatesSeedNoRejectedIDShape(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(repoRoot(t), "internal", "skills", "embedded-rituals",
		"plugins", "aiwf-extensions", "templates")
	emptyConsumerTree := check.BodyProseIDIndex(&tree.Tree{})

	for _, name := range shippedTemplates {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			raw, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("reading shipped template: %v", err)
			}
			// body-prose-id scans the post-frontmatter body, so the
			// frontmatter's own placeholders — which `aiwf add` overwrites
			// with the allocated id — are correctly out of scope here.
			body := raw
			if _, b, ok := entity.Split(raw); ok {
				body = b
			}
			for _, f := range check.ScanBodyProseID(body, "<template>", name, emptyConsumerTree) {
				t.Errorf("line %d: %s [%s]", f.Line, f.Message, f.Subcode)
			}
		})
	}
}
