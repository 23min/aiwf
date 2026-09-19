package skills

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestMaterializeTo_CodexWritesCompleteNativeInventory(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := MaterializeTo(root, CodexTarget()); err != nil {
		t.Fatal(err)
	}
	verbs, err := listVerbSources()
	if err != nil {
		t.Fatal(err)
	}
	rituals, err := listRitualSources()
	if err != nil {
		t.Fatal(err)
	}
	templates, err := listRitualFiles("templates")
	if err != nil {
		t.Fatal(err)
	}
	bindings := ClaudeRenderBindings()
	bindings.Target = CodexTarget()
	definitions := append(append(verbs, rituals...), templates...)
	rendered, err := RenderSkills(definitions, bindings)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".agents/skills/.aiwf-owned", ".agents/skills/README.md", ".agents/aiwf/templates/.aiwf-owned"}
	var skillNames, templateNames []string
	for _, skill := range rendered[:len(verbs)+len(rituals)] {
		skillNames = append(skillNames, skill.Name)
		path := ".agents/skills/" + skill.Name + "/SKILL.md"
		want = append(want, path)
		content, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || !bytes.Equal(content, skill.Content) {
			t.Errorf("%s differs from its rendered canonical definition: %v", path, err)
		}
	}
	for _, template := range rendered[len(verbs)+len(rituals):] {
		templateNames = append(templateNames, template.Name)
		path := ".agents/aiwf/templates/" + template.Name
		want = append(want, path)
		content, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || !bytes.Equal(content, template.Content) {
			t.Errorf("%s differs from its rendered canonical definition: %v", path, err)
		}
	}
	var got []string
	for path, entry := range artifactTree(t, root) {
		if entry.Mode.IsRegular() {
			got = append(got, filepath.ToSlash(path))
			if entry.Mode.Perm() != 0o644 {
				t.Errorf("%s mode = %v, want 0644", path, entry.Mode.Perm())
			}
		}
	}
	sort.Strings(want)
	sort.Strings(got)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("artifact inventory (-want +got):\n%s", diff)
	}
	for manifest, names := range map[string][]string{
		".agents/skills/.aiwf-owned":         skillNames,
		".agents/aiwf/templates/.aiwf-owned": templateNames,
	} {
		content, err := os.ReadFile(filepath.Join(root, manifest))
		if err != nil {
			t.Fatal(err)
		}
		owned := strings.Fields(string(content))
		sort.Strings(owned)
		sort.Strings(names)
		if diff := cmp.Diff(names, owned); diff != "" {
			t.Errorf("%s ownership (-want +got):\n%s", manifest, diff)
		}
	}
	for _, path := range []string{".claude", ".codex", ".agents/agents", ".agents/templates", ".agents/hooks"} {
		if _, err := os.Lstat(filepath.Join(root, path)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("unsupported artifact %s exists or cannot be inspected: %v", path, err)
		}
	}
}

func TestMaterializeTo_CodexMetadataAndTemplateReferencesResolve(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := MaterializeTo(root, CodexTarget()); err != nil {
		t.Fatal(err)
	}
	citations := regexp.MustCompile("(?:\\.agents/aiwf|\\.claude)/templates/([^\\s`)\\]]*)")
	references := 0
	for path, entry := range artifactTree(t, root) {
		if !entry.Mode.IsRegular() {
			continue
		}
		if strings.Contains(entry.Content, renderPrefix) {
			t.Errorf("%s contains an unresolved rendering token", path)
		}
		if filepath.Base(path) != "SKILL.md" {
			continue
		}
		parts := bytes.SplitN([]byte(entry.Content), []byte("---\n"), 3)
		if len(parts) != 3 || len(parts[0]) != 0 {
			t.Fatalf("%s has no YAML frontmatter", path)
		}
		var metadata struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		}
		if err := yaml.Unmarshal(parts[1], &metadata); err != nil {
			t.Fatalf("%s frontmatter: %v", path, err)
		}
		if metadata.Name != filepath.Base(filepath.Dir(path)) || metadata.Description == "" {
			t.Errorf("%s metadata = %+v", path, metadata)
		}
		for _, match := range citations.FindAllStringSubmatch(entry.Content, -1) {
			if match[1] == "" {
				continue
			}
			references++
			if _, err := os.Stat(filepath.Join(root, match[0])); err != nil {
				t.Errorf("%s references missing support file %s: %v", path, match[0], err)
			}
		}
	}
	if references == 0 {
		t.Fatal("no generated template references were checked")
	}
}

func TestMaterializeTo_CodexRefreshPreservesClaudeAndForeignSiblings(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatal(err)
	}
	claude := artifactTree(t, filepath.Join(root, ".claude"))
	foreign := map[string]string{
		".agents/skills/personal/SKILL.md":   "personal skill\n",
		".agents/aiwf/templates/personal.md": "personal template\n",
		"AGENTS.md":                          "personal instructions\n",
	}
	for path, content := range foreign {
		writeArtifactFixture(t, filepath.Join(root, path), content)
	}
	if err := MaterializeTo(root, CodexTarget()); err != nil {
		t.Fatal(err)
	}
	before := artifactTree(t, root)
	if err := MaterializeTo(root, CodexTarget()); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(before, artifactTree(t, root)); diff != "" {
		t.Errorf("refresh changed artifacts (-before +after):\n%s", diff)
	}
	if diff := cmp.Diff(claude, artifactTree(t, filepath.Join(root, ".claude"))); diff != "" {
		t.Errorf("Codex changed Claude artifacts (-before +after):\n%s", diff)
	}
	for path, content := range foreign {
		got, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || string(got) != content {
			t.Errorf("foreign %s = %q, %v; want %q", path, got, err, content)
		}
	}
}

func TestMaterializeTo_ProvenanceReferencesSelectedSupportFiles(t *testing.T) {
	t.Parallel()
	for _, target := range []Target{ClaudeTarget, CodexTarget(), {
		Name: "fixture", SkillsDir: "adapter/skills", AgentsDir: "support/roles", TemplatesDir: "support/templates",
	}} {
		t.Run(target.Name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := MaterializeTo(root, target); err != nil {
				t.Fatal(err)
			}
			readmeDir := filepath.Join(root, target.SkillsDir)
			content, err := os.ReadFile(filepath.Join(readmeDir, ProvenanceReadme))
			if err != nil {
				t.Fatal(err)
			}
			var referenced, materialized []string
			citations := regexp.MustCompile("`([^`]+/\\*\\.md)`")
			for _, match := range citations.FindAllStringSubmatch(string(content), -1) {
				paths, err := filepath.Glob(filepath.Join(readmeDir, match[1]))
				if err != nil || len(paths) == 0 {
					t.Fatalf("provenance references no support files at %s: %v", match[1], err)
				}
				referenced = append(referenced, paths...)
			}
			for _, dir := range []string{target.AgentsDir, target.TemplatesDir} {
				if dir == "" {
					continue
				}
				paths, err := filepath.Glob(filepath.Join(root, dir, "*.md"))
				if err != nil {
					t.Fatal(err)
				}
				materialized = append(materialized, paths...)
			}
			sort.Strings(referenced)
			sort.Strings(materialized)
			if diff := cmp.Diff(materialized, referenced); diff != "" {
				t.Errorf("provenance support references (-materialized +referenced):\n%s", diff)
			}
		})
	}
}

func TestMaterializeArtifacts_InvalidSupportLayoutLeavesFilesUntouched(t *testing.T) {
	t.Parallel()
	for _, family := range []string{"templates", "agents"} {
		t.Run(family, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeArtifactFixture(t, filepath.Join(root, "personal.md"), "user content")
			before := artifactTree(t, root)
			target := ClaudeTarget
			if family == "templates" {
				target.TemplatesDir = filepath.Join(root, "templates")
			} else {
				target.AgentsDir = filepath.Join(root, "agents")
			}
			if err := materializeArtifacts(root, target, nil, artifactSources{}); err == nil {
				t.Fatal("mixed absolute and relative support layout accepted")
			}
			if diff := cmp.Diff(before, artifactTree(t, root)); diff != "" {
				t.Errorf("invalid support layout changed files (-before +after):\n%s", diff)
			}
		})
	}
}
