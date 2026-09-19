package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestHostExamples_ReferenceIsInertAndUncommentedSelectionRoundTrips(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(GenerateExample()), 0o600); err != nil {
		t.Fatal(err)
	}
	inert, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if inert.Hosts != nil || !inert.WireClaudeMd() || !inert.WireAgentsMd() || inert.WorktreeDir() != ".claude/worktrees" {
		t.Fatalf("commented example changes defaults: %+v", inert)
	}
	var parsed Config
	if decodeErr := yaml.Unmarshal([]byte(uncommentYAML(GenerateExample())), &parsed); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	// Only uncomment the host setup blocks; unrelated placeholder entries remain inert.
	selected := &Config{Hosts: parsed.Hosts, Guidance: parsed.Guidance, Worktree: parsed.Worktree}
	selectedRoot := t.TempDir()
	if writeErr := Write(selectedRoot, selected); writeErr != nil {
		t.Fatal(writeErr)
	}
	loaded, err := Load(selectedRoot)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(&[]string{}, loaded.Hosts); diff != "" {
		t.Fatalf("explicit empty example lost its meaning:\n%s", diff)
	}
	if !loaded.WireClaudeMd() || !loaded.WireAgentsMd() || loaded.WorktreeDir() != ".claude/worktrees" {
		t.Fatalf("uncommented setup defaults changed: %+v", loaded)
	}
}
