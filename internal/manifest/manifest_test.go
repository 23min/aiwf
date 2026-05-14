package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParse_YAML covers the canonical happy path: a manifest with a
// mix of explicit and auto ids, default commit mode, and no actor
// override. Verifies that field tags work for both explicit YAML
// fields and the polymorphic Frontmatter map.
func TestParse_YAML(t *testing.T) {
	t.Parallel()
	src := []byte(`version: 1
actor: human/peter
commit:
  mode: single
  message: "import: bulk migration"
entities:
  - kind: epic
    id: E-0011
    frontmatter:
      title: "Svelte UI"
      status: active
    body: |
      ## Goal
      Build a UI.
  - kind: milestone
    id: auto
    frontmatter:
      title: "Scaffold"
      status: done
      parent: E-0011
`)
	m, err := Parse(src, "yaml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}
	if m.Actor != "human/peter" {
		t.Errorf("Actor = %q", m.Actor)
	}
	if m.EffectiveCommitMode() != CommitSingle {
		t.Errorf("EffectiveCommitMode = %q", m.EffectiveCommitMode())
	}
	if len(m.Entities) != 2 {
		t.Fatalf("len(Entities) = %d, want 2", len(m.Entities))
	}
	if m.Entities[0].Kind != "epic" || m.Entities[0].ID != "E-0011" {
		t.Errorf("entity[0] = %+v", m.Entities[0])
	}
	if !m.Entities[1].IsAuto() {
		t.Errorf("entity[1].IsAuto = false, want true")
	}
	if m.Entities[0].Frontmatter["title"] != "Svelte UI" {
		t.Errorf("entity[0].title = %v", m.Entities[0].Frontmatter["title"])
	}
	if m.Entities[0].Body == "" {
		t.Errorf("entity[0].Body empty")
	}
}

// TestParse_JSON parses the same logical manifest as JSON; field tags
// must work for both lexers.
func TestParse_JSON(t *testing.T) {
	t.Parallel()
	src := []byte(`{
  "version": 1,
  "entities": [
    {"kind": "epic", "id": "E-0011", "frontmatter": {"title": "X", "status": "active"}}
  ]
}`)
	m, err := Parse(src, "json")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(m.Entities) != 1 || m.Entities[0].ID != "E-0011" {
		t.Errorf("unexpected entities: %+v", m.Entities)
	}
}

// TestParse_DefaultsCommitToSingle: when commit.mode is omitted,
// EffectiveCommitMode falls back to single.
func TestParse_DefaultsCommitToSingle(t *testing.T) {
	t.Parallel()
	m, err := Parse([]byte("version: 1\nentities: []\n"), "yaml")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if m.EffectiveCommitMode() != CommitSingle {
		t.Errorf("default mode = %q, want %q", m.EffectiveCommitMode(), CommitSingle)
	}
}

// TestParse_AcceptsEmptyEntities: `entities: []` is a valid no-op
// manifest. Useful for testing the import pipeline without entities.
func TestParse_AcceptsEmptyEntities(t *testing.T) {
	t.Parallel()
	if _, err := Parse([]byte("version: 1\nentities: []\n"), "yaml"); err != nil {
		t.Errorf("empty entities should parse: %v", err)
	}
}

// TestValidate_Errors enumerates the structural rejections the parser
// promises. Each case is named for the field it violates.
func TestValidate_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		src     string
		wantSub string
	}{
		{
			name:    "missing version",
			src:     "entities: []\n",
			wantSub: "missing required field `version`",
		},
		{
			name:    "unsupported version",
			src:     "version: 99\nentities: []\n",
			wantSub: "version 99 is not supported",
		},
		{
			name:    "bad commit mode",
			src:     "version: 1\ncommit:\n  mode: weekly\nentities: []\n",
			wantSub: "commit.mode \"weekly\"",
		},
		{
			name:    "missing kind",
			src:     "version: 1\nentities:\n  - id: E-01\n    frontmatter: {title: X, status: active}\n",
			wantSub: "missing required field `kind`",
		},
		{
			name:    "unknown kind",
			src:     "version: 1\nentities:\n  - kind: story\n    id: S-01\n    frontmatter: {title: X}\n",
			wantSub: "unknown kind \"story\"",
		},
		{
			name:    "missing id",
			src:     "version: 1\nentities:\n  - kind: epic\n    frontmatter: {title: X, status: active}\n",
			wantSub: "missing required field `id`",
		},
		{
			name:    "id wrong format for kind",
			src:     "version: 1\nentities:\n  - kind: epic\n    id: M-001\n    frontmatter: {title: X, status: active}\n",
			wantSub: "does not match E-NN format",
		},
		{
			name:    "missing frontmatter",
			src:     "version: 1\nentities:\n  - kind: epic\n    id: E-01\n",
			wantSub: "missing required field `frontmatter`",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(tc.src), "yaml")
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error = %q\nwant substring %q", err.Error(), tc.wantSub)
			}
		})
	}
}

// TestParseFile_DetectsFormat covers extension-based dispatch and the
// rejection of unknown extensions.
func TestParseFile_DetectsFormat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "m.yaml")
	if err := writeFile(yamlPath, "version: 1\nentities: []\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(yamlPath); err != nil {
		t.Errorf("ParseFile yaml: %v", err)
	}

	jsonPath := filepath.Join(dir, "m.json")
	if err := writeFile(jsonPath, `{"version":1,"entities":[]}`); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(jsonPath); err != nil {
		t.Errorf("ParseFile json: %v", err)
	}

	badPath := filepath.Join(dir, "m.txt")
	if err := writeFile(badPath, "version: 1"); err != nil {
		t.Fatal(err)
	}
	_, err := ParseFile(badPath)
	if err == nil || !strings.Contains(err.Error(), "unsupported extension") {
		t.Errorf("expected unsupported-extension error, got %v", err)
	}
}

// writeFile is a test helper to drop a string into a temp file.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
