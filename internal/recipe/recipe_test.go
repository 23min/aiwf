package recipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestList_EmbeddedHasTwoRecipes(t *testing.T) {
	got, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	names := make([]string, 0, len(got))
	for _, r := range got {
		names = append(names, r.Name)
	}
	if diff := cmp.Diff([]string{"cue", "jsonschema"}, names); diff != "" {
		t.Errorf("recipe names mismatch (-want +got):\n%s", diff)
	}
	for _, r := range got {
		if r.Validator.Command == "" {
			t.Errorf("recipe %q: command is empty", r.Name)
		}
		if !strings.HasPrefix(string(r.Markdown), "---") {
			t.Errorf("recipe %q markdown should start with frontmatter", r.Name)
		}
	}
}

func TestGet_HitsAndMisses(t *testing.T) {
	got, err := Get("cue")
	if err != nil {
		t.Fatalf("Get(cue): %v", err)
	}
	if got.Validator.Command != "cue" {
		t.Errorf("command = %q, want %q", got.Validator.Command, "cue")
	}
	wantArgs := []string{"vet", "{{schema}}", "{{fixture}}"}
	if diff := cmp.Diff(wantArgs, got.Validator.Args); diff != "" {
		t.Errorf("args mismatch:\n%s", diff)
	}

	if _, err := Get("ghost"); err == nil {
		t.Error("expected ErrNotFound for unknown name")
	}
}

func TestParseFile_CustomValidator(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "pydantic.yaml")
	body := []byte(`name: pydantic
command: python
args:
  - -m
  - my_validator
  - "{{schema}}"
  - "{{fixture}}"
`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if r.Name != "pydantic" {
		t.Errorf("name = %q", r.Name)
	}
	if r.Validator.Command != "python" {
		t.Errorf("command = %q", r.Validator.Command)
	}
}

func TestParseFile_RejectsUnknownField(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.yaml")
	body := []byte(`name: bad
command: x
args: []
mystery: nope
`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(path); err == nil {
		t.Error("expected error for unknown field")
	}
}

func TestParseFile_RejectsMissingFields(t *testing.T) {
	tmp := t.TempDir()
	tests := []struct {
		name string
		body string
	}{
		{"missing-name", "command: x\nargs: []\n"},
		{"missing-command", "name: foo\nargs: []\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmp, tt.name+".yaml")
			if err := os.WriteFile(path, []byte(tt.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := ParseFile(path); err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestSplitFrontmatter_RejectsMissingDelimiter(t *testing.T) {
	if _, _, err := splitFrontmatter([]byte("# just markdown\n")); err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestSplitFrontmatter_RejectsUnterminated(t *testing.T) {
	if _, _, err := splitFrontmatter([]byte("---\nname: foo\n")); err == nil {
		t.Error("expected error for unterminated frontmatter")
	}
}
