package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestDocs_UnmarshalYAML_RejectsUnknownKey pins the strict-key guard. Without
// it a typo drops silently while `strict: true` still parses, so the operator
// believes they are guarding several documents at blocking severity and are
// guarding only the default one — a false sense of coverage, not an error.
func TestDocs_UnmarshalYAML_RejectsUnknownKey(t *testing.T) {
	t.Parallel()
	var c Config
	err := yaml.Unmarshal([]byte("docs:\n  pathz:\n    - README.md\n  strict: true\n"), &c)
	if err == nil {
		t.Fatal("a typo'd docs key must be rejected, not silently dropped")
	}
	if !strings.Contains(err.Error(), "pathz") {
		t.Errorf("error %q does not name the offending key", err)
	}
}

// TestDocs_UnmarshalYAML_AcceptsKnownKeys guards the other direction: the
// method must still decode a well-formed block.
func TestDocs_UnmarshalYAML_AcceptsKnownKeys(t *testing.T) {
	t.Parallel()
	var c Config
	if err := yaml.Unmarshal([]byte("docs:\n  paths:\n    - README.md\n    - docs/workflows.md\n  strict: true\n"), &c); err != nil {
		t.Fatalf("a well-formed docs block must decode: %v", err)
	}
	if got := c.DocsPaths(); len(got) != 2 || got[0] != "README.md" {
		t.Errorf("DocsPaths() = %v", got)
	}
	if !c.DocsStrict() {
		t.Error("DocsStrict() = false, want true")
	}
}
