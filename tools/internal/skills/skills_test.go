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

// TestMaterialize_WipesPreviouslyOwnedStale: when a previous aiwf
// version listed `aiwf-removed` in its ownership manifest and the
// current version no longer embeds it, Materialize wipes the stale
// dir. This is the "skill removed from a release" cleanup path.
func TestMaterialize_WipesPreviouslyOwnedStale(t *testing.T) {
	root := t.TempDir()
	skillsRoot := filepath.Join(root, SkillsDir)
	if err := os.MkdirAll(skillsRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(skillsRoot, "aiwf-removed")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, "SKILL.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Pretend a previous aiwf wrote a manifest claiming to own `aiwf-removed`.
	if err := os.WriteFile(filepath.Join(skillsRoot, ManifestFile), []byte("aiwf-removed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale aiwf-removed/ should have been wiped (was in old manifest); stat err=%v", err)
	}
}

// TestMaterialize_LeavesForeignAiwfPrefixedDirAlone is the load-bearing
// test for G7: a directory named like `aiwf-rituals-something` that
// aiwf never owned (not in any prior manifest) must NOT be wiped, even
// though it shares the `aiwf-` prefix. Third-party plugins under the
// prefix are safe.
func TestMaterialize_LeavesForeignAiwfPrefixedDirAlone(t *testing.T) {
	root := t.TempDir()
	foreign := filepath.Join(root, SkillsDir, "aiwf-rituals-tdd")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(foreign, "MARKER")
	if err := os.WriteFile(marker, []byte("third-party"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("foreign aiwf-prefixed dir was wiped (it should not be); read err=%v", err)
	}
	if string(got) != "third-party" {
		t.Errorf("foreign content modified: %q", got)
	}
}

// TestMaterialize_WritesManifest: after Materialize succeeds, the
// ownership manifest lists exactly the names of currently-embedded
// skills, one per line.
func TestMaterialize_WritesManifest(t *testing.T) {
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, SkillsDir, ManifestFile)
	got, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	want := ""
	for _, s := range skills {
		want += s.Name + "\n"
	}
	if string(got) != want {
		t.Errorf("manifest content mismatch:\nwant:\n%s\ngot:\n%s", want, string(got))
	}
}

// TestMaterialize_RoundTripPreservesForeignAcrossUpdates: a foreign
// dir survives multiple Materialize calls (simulating successive
// `aiwf update` invocations).
func TestMaterialize_RoundTripPreservesForeignAcrossUpdates(t *testing.T) {
	root := t.TempDir()
	foreign := filepath.Join(root, SkillsDir, "aiwf-userplugin")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, "SKILL.md"), []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := Materialize(root); err != nil {
			t.Fatalf("Materialize iteration %d: %v", i, err)
		}
	}
	got, err := os.ReadFile(filepath.Join(foreign, "SKILL.md"))
	if err != nil {
		t.Fatalf("foreign skill removed across updates: %v", err)
	}
	if string(got) != "user" {
		t.Errorf("foreign content changed: %q", got)
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
	manifestSeen := false
	for _, p := range got {
		if p == SkillsDir+"/"+ManifestFile {
			manifestSeen = true
			continue
		}
		if !strings.HasPrefix(p, SkillsDir+"/aiwf-") {
			t.Errorf("path %q lacks expected prefix", p)
		}
		if !strings.HasSuffix(p, "/") {
			t.Errorf("skill dir path %q should end with / (gitignore convention)", p)
		}
	}
	if !manifestSeen {
		t.Errorf("manifest path %s/%s missing from MaterializedPaths so it'd land in git commits", SkillsDir, ManifestFile)
	}
}
