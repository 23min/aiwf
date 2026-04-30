package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestList_AllShippedSkillsPresent guards the contract that we ship
// the six verb skills from session 3, aiwf-status (added on
// poc/aiwf-rename-skills), and aiwf-contract (added in I1.8 of the
// contracts plan).
func TestList_AllShippedSkillsPresent(t *testing.T) {
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(skills))
	for i, s := range skills {
		got[i] = s.Name
	}
	want := []string{"aiwf-add", "aiwf-check", "aiwf-contract", "aiwf-history", "aiwf-promote", "aiwf-reallocate", "aiwf-rename", "aiwf-status"}
	if len(got) != len(want) {
		t.Fatalf("got %d skills, want %d (%v vs %v)", len(got), len(want), got, want)
	}
	sort.Strings(got)
	for i, name := range want {
		if got[i] != name {
			t.Errorf("[%d] got %q, want %q", i, got[i], name)
		}
	}
}

// TestList_ContentNonEmptyAndYAMLFrontmatter sanity-checks that every
// embedded SKILL.md starts with a YAML front-matter block; a missing
// front-matter would silently break Claude Code's skill loader.
func TestList_ContentNonEmptyAndYAMLFrontmatter(t *testing.T) {
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range skills {
		if len(s.Content) == 0 {
			t.Errorf("%s: empty content", s.Name)
			continue
		}
		if !strings.HasPrefix(string(s.Content), "---\n") {
			t.Errorf("%s: missing YAML front-matter (no leading ---)", s.Name)
		}
		if !strings.Contains(string(s.Content), "\nname: "+s.Name+"\n") {
			t.Errorf("%s: front-matter `name:` does not match dir", s.Name)
		}
	}
}

// TestMaterialize_FreshDir writes every embedded skill into a clean
// directory and verifies the on-disk content matches the embed
// byte-for-byte.
func TestMaterialize_FreshDir(t *testing.T) {
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range skills {
		on := filepath.Join(root, SkillsDir, s.Name, "SKILL.md")
		got, err := os.ReadFile(on)
		if err != nil {
			t.Fatalf("read %s: %v", on, err)
		}
		if !bytes.Equal(got, s.Content) {
			t.Errorf("%s: on-disk content differs from embed", s.Name)
		}
	}
}

// TestMaterialize_WipesStale puts a stale aiwf-something/ dir in place
// and verifies Materialize removes it (the cache contract).
func TestMaterialize_WipesStale(t *testing.T) {
	root := t.TempDir()
	stale := filepath.Join(root, SkillsDir, "aiwf-removed")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, "SKILL.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale aiwf-removed/ should have been wiped, stat err=%v", err)
	}
}

// TestMaterialize_PreservesNonAiwfDirs guards the namespace boundary —
// user-authored `.claude/skills/<not-aiwf>/` directories must not be
// touched by Materialize.
func TestMaterialize_PreservesNonAiwfDirs(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, SkillsDir, "my-custom-skill")
	if err := os.MkdirAll(user, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(user, "SKILL.md"), []byte("user content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(user, "SKILL.md"))
	if err != nil {
		t.Fatalf("user skill removed: %v", err)
	}
	if string(got) != "user content" {
		t.Errorf("user skill content changed: %q", got)
	}
}

func TestMaterializedPaths(t *testing.T) {
	got, err := MaterializedPaths()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("expected non-empty list")
	}
	for _, p := range got {
		if !strings.HasPrefix(p, SkillsDir+"/aiwf-") {
			t.Errorf("path %q lacks expected prefix", p)
		}
		if !strings.HasSuffix(p, "/") {
			t.Errorf("path %q should end with / (gitignore convention)", p)
		}
	}
}
