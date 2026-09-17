package policies

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"

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

// The ways a template's frontmatter can fail to be checked at all.
var (
	errNoTemplateFrontmatter          = errors.New("no frontmatter block")
	errUndecodableTemplateFrontmatter = errors.New("frontmatter does not decode")
	errUnknownTemplateID              = errors.New("id matches no kind's id format")
)

// templateFieldsOutsideSchema returns the fields a template's frontmatter
// reference block lists that its own kind's schema does not declare. The block
// is where an author reads which fields a kind carries, and the strict decoder
// above accepts any field some kind declares, so a field this kind lacks still
// parses and is then ignored — a value set from the block is lost with no
// finding. The kind is found by matching the block's placeholder id against
// each schema's id format.
func templateFieldsOutsideSchema(content []byte) ([]string, error) {
	fm, _, ok := entity.Split(content)
	if !ok {
		return nil, errNoTemplateFrontmatter
	}
	var fields map[string]any
	if err := yaml.Unmarshal(fm, &fields); err != nil {
		return nil, fmt.Errorf("%w: %w", errUndecodableTemplateFrontmatter, err)
	}
	id, _ := fields["id"].(string)
	schemas := entity.AllSchemas()
	var schema *entity.Schema
	for i := range schemas {
		if schemas[i].IDFormat == id {
			schema = &schemas[i]
		}
	}
	if schema == nil {
		return nil, fmt.Errorf("%w: %q", errUnknownTemplateID, id)
	}
	declared := map[string]bool{}
	for _, field := range slices.Concat(schema.RequiredFields, schema.OptionalFields) {
		declared[field] = true
	}
	var outside []string
	for field := range fields {
		if !declared[field] {
			outside = append(outside, field)
		}
	}
	sort.Strings(outside)
	return outside, nil
}

func TestTemplateFieldsOutsideSchema(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		content string
		want    []string
		wantErr error
	}{
		{
			name:    "a field another kind declares is outside this kind's schema",
			content: "---\nid: D-NNNN\ntitle: x\nstatus: proposed\nsupersedes: []\n---\n",
			want:    []string{"supersedes"},
		},
		{
			name:    "a field the kind declares is inside it",
			content: "---\nid: ADR-NNNN\ntitle: x\nstatus: proposed\nsupersedes: []\n---\n",
		},
		{
			name:    "an id matching no kind cannot be checked",
			content: "---\nid: X-NNNN\ntitle: x\n---\n",
			wantErr: errUnknownTemplateID,
		},
		{
			name:    "a template without a frontmatter block cannot be checked",
			content: "# A body only\n",
			wantErr: errNoTemplateFrontmatter,
		},
		{
			name:    "a frontmatter block that does not decode cannot be checked",
			content: "---\nid: [unclosed\n---\n",
			wantErr: errUndecodableTemplateFrontmatter,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := templateFieldsOutsideSchema([]byte(tc.content))
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("templateFieldsOutsideSchema error = %v, want %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("templateFieldsOutsideSchema mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestEmbeddedTemplateFrontmatterNamesOnlySchemaFields holds every shipped
// template to templateFieldsOutsideSchema.
func TestEmbeddedTemplateFrontmatterNamesOnlySchemaFields(t *testing.T) {
	t.Parallel()
	templates, err := skills.ListRitualTemplates()
	if err != nil {
		t.Fatalf("ListRitualTemplates: %v", err)
	}
	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()
			outside, err := templateFieldsOutsideSchema(tmpl.Content)
			if err != nil {
				t.Fatalf("embedded template %s: %v", tmpl.Name, err)
			}
			for _, field := range outside {
				t.Errorf("embedded template %s lists %q, which its kind's schema does not declare", tmpl.Name, field)
			}
		})
	}
}
