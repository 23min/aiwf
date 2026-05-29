package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestListRituals confirms the vendored ritual skills (aiwfx-*, wf-*) are
// discovered from the embedded-rituals tree, name-sorted, with content.
func TestListRituals(t *testing.T) {
	t.Parallel()
	got, err := ListRituals()
	if err != nil {
		t.Fatalf("ListRituals: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("ListRituals returned no skills")
	}
	names := make([]string, len(got))
	for i, s := range got {
		names[i] = s.Name
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("ListRituals not name-sorted: %v", names)
	}
	for _, want := range []string{"aiwfx-plan-epic", "aiwfx-start-milestone", "aiwfx-wrap-epic", "wf-tdd-cycle", "wf-review-code"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ListRituals missing %q; got %v", want, names)
		}
	}
	for _, s := range got {
		if !strings.HasPrefix(s.Name, "aiwfx-") && !strings.HasPrefix(s.Name, "wf-") {
			t.Errorf("ritual skill %q has unexpected prefix (want aiwfx-/wf-)", s.Name)
		}
		if len(s.Content) == 0 {
			t.Errorf("ritual skill %q has empty content", s.Name)
		}
	}
}

// TestMaterialize_WritesRitualSkills covers AC-1: Materialize writes the
// embedded ritual skills, flattened, into .claude/skills/<name>/SKILL.md.
func TestMaterialize_WritesRitualSkills(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	rituals, err := ListRituals()
	if err != nil {
		t.Fatalf("ListRituals: %v", err)
	}
	for _, s := range rituals {
		got, err := os.ReadFile(filepath.Join(root, SkillsDir, s.Name, "SKILL.md"))
		if err != nil {
			t.Errorf("ritual skill %s not materialized: %v", s.Name, err)
			continue
		}
		if !bytes.Equal(got, s.Content) {
			t.Errorf("materialized content mismatch for %s", s.Name)
		}
	}
}

// TestMaterialize_WritesVerbSkillsToo covers AC-1's no-regression edge:
// verb skills still materialize alongside the rituals.
func TestMaterialize_WritesVerbSkillsToo(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, SkillsDir, "aiwf-check", "SKILL.md")); err != nil {
		t.Errorf("verb skill aiwf-check not materialized: %v", err)
	}
}

// TestMaterialize_ManifestOwnsRitualSkills covers AC-2: the ownership
// manifest names the ritual skills (and still the verb skills).
func TestMaterialize_ManifestOwnsRitualSkills(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(root, SkillsDir, ManifestFile))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	for _, want := range []string{"aiwfx-plan-epic", "wf-tdd-cycle", "aiwf-check"} {
		if !strings.Contains(string(manifest), want+"\n") {
			t.Errorf("manifest missing %q:\n%s", want, manifest)
		}
	}
}

// TestMaterialize_RitualsIdempotent covers AC-2: re-running Materialize
// (the `aiwf update` path) leaves the ritual skills in place.
func TestMaterialize_RitualsIdempotent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("first Materialize: %v", err)
	}
	if err := Materialize(root); err != nil {
		t.Fatalf("second Materialize: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, SkillsDir, "wf-tdd-cycle", "SKILL.md")); err != nil {
		t.Errorf("ritual skill missing after re-materialize: %v", err)
	}
}

// TestMaterialize_DoesNotClobberUserSkills covers AC-2: a user-authored
// skill dir is never touched by materialization.
func TestMaterialize_DoesNotClobberUserSkills(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	userDir := filepath.Join(root, SkillsDir, "my-custom-skill")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("---\nname: my-custom-skill\ndescription: mine\n---\nhi\n")
	if err := os.WriteFile(filepath.Join(userDir, "SKILL.md"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(userDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("user skill removed by Materialize: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("user skill content changed by Materialize")
	}
}

// TestGitignorePatterns_CoverRituals covers AC-2: the gitignore patterns
// mask the ritual skill dirs alongside the verb skill dirs.
func TestGitignorePatterns_CoverRituals(t *testing.T) {
	t.Parallel()
	pats, err := GitignorePatterns()
	if err != nil {
		t.Fatalf("GitignorePatterns: %v", err)
	}
	for _, want := range []string{SkillsDir + "/aiwf-*/", SkillsDir + "/aiwfx-*/", SkillsDir + "/wf-*/"} {
		found := false
		for _, p := range pats {
			if p == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GitignorePatterns missing %q; got %v", want, pats)
		}
	}
}
