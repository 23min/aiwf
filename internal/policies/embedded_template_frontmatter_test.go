package policies

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/skills"
)

// TestEmbeddedTemplateFrontmatterParses pins the invariant that every entity
// template aiwf ships decodes cleanly through the same strict frontmatter
// decoder (entity.Parse → yaml KnownFields(true)) that `aiwf check` runs
// against a consumer's tree. A template carrying a frontmatter key the Entity
// struct does not accept — as epic-spec.md once shipped `completed:` — produces
// a hard load-error the instant a consumer fills the template in, so a shipped
// scaffold must satisfy the decoder it will be validated by.
//
// entity.Parse is the production oracle, not a reimplementation: the accepted-key
// set is single-sourced from the Entity struct via the real decoder, so a
// newly-added stray key fails here with no second allowlist to maintain.
func TestEmbeddedTemplateFrontmatterParses(t *testing.T) {
	t.Parallel()
	templates, err := skills.ListRitualTemplates()
	if err != nil {
		t.Fatalf("ListRitualTemplates: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("no embedded templates found; expected the shipped entity templates")
	}
	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()
			if _, err := entity.Parse(tmpl.Name, tmpl.Content); err != nil {
				t.Errorf("embedded template %s carries frontmatter the strict entity decoder rejects: %v", tmpl.Name, err)
			}
		})
	}
}
