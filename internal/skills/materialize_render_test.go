package skills

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestMaterializeArtifacts_RejectsInvalidSourcesBeforeChangingFiles(t *testing.T) {
	t.Parallel()
	for _, family := range []string{"skills", "agents", "templates"} {
		for _, existing := range []bool{false, true} {
			t.Run(family+map[bool]string{false: "/new", true: "/existing"}[existing], func(t *testing.T) {
				t.Parallel()
				root := t.TempDir()
				if existing {
					if err := Materialize(root); err != nil {
						t.Fatal(err)
					}
				}
				before := artifactTree(t, root)
				sources := artifactSources{
					skills:    []Skill{{Name: "fixture", Content: []byte("{{aiwf:templates_dir}}/fixture.md")}},
					agents:    []Skill{{Name: "fixture.md", Content: []byte("{{aiwf:host}}")}},
					templates: []Skill{{Name: "fixture.md", Content: []byte("{{aiwf:skills_dir}}")}},
				}
				bad := []byte("{{aiwf:unknown}}")
				switch family {
				case "skills":
					sources.skills[0].Content = bad
				case "agents":
					sources.agents[0].Content = bad
				case "templates":
					sources.templates[0].Content = bad
				}
				err := materializeArtifacts(root, ClaudeTarget, nil, sources)
				if !errors.Is(err, ErrUnknownRenderBinding) {
					t.Fatalf("got %v, want unknown binding", err)
				}
				if diff := cmp.Diff(before, artifactTree(t, root)); diff != "" {
					t.Fatalf("invalid source changed filesystem (-before +after):\n%s", diff)
				}
			})
		}
	}
}

func TestMaterializeArtifacts_RendersTargetPathsAndSkipsAbsentAgentFamily(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	target := Target{Name: "fixture", SkillsDir: "adapter/skills", TemplatesDir: "adapter/templates"}
	sources := artifactSources{
		skills:    []Skill{{Name: "fixture", Content: []byte("{{aiwf:templates_dir}}/fixture.md")}},
		agents:    []Skill{{Name: "unused.md", Content: []byte("{{aiwf:unknown}}")}},
		templates: []Skill{{Name: "fixture.md", Content: []byte("{{aiwf:host}} {{aiwf:skills_dir}}")}},
	}
	if err := materializeArtifacts(root, target, nil, sources); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{
		"adapter/skills/fixture/SKILL.md": "adapter/templates/fixture.md",
		"adapter/templates/fixture.md":    "fixture adapter/skills",
	} {
		got, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", path, got, err, want)
		}
	}
}

func TestMaterializeArtifacts_ReportsFilesystemFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, blocker, readOnly, context string
		directory                        bool
		want                             error
	}{
		{name: "skills directory", blocker: SkillsDir, context: SkillsDir, want: ErrUnsafeArtifactPath},
		{name: "manifest read", blocker: SkillsDir + "/" + ManifestFile, directory: true, context: ManifestFile, want: ErrInvalidOwnership},
		{name: "obsolete removal", readOnly: SkillsDir + "/obsolete", context: "retiring artifact", want: fs.ErrPermission},
		{name: "skill directory", blocker: SkillsDir + "/fixture", context: "fixture", want: ErrOwnershipConflict},
		{name: "skill write", blocker: SkillsDir + "/fixture/SKILL.md", directory: true, context: "fixture/SKILL.md", want: ErrOwnershipConflict},
		{name: "provenance permission", readOnly: SkillsDir, context: "writing provenance", want: fs.ErrPermission},
		{name: "provenance write", blocker: SkillsDir + "/" + ProvenanceReadme, directory: true, context: ProvenanceReadme, want: ErrOwnershipConflict},
		{name: "agents", blocker: AgentsDir, context: AgentsDir, want: ErrUnsafeArtifactPath},
		{name: "templates", blocker: TemplatesDir, context: TemplatesDir, want: ErrUnsafeArtifactPath},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if tc.blocker != "" {
				path := filepath.Join(root, tc.blocker)
				if tc.directory {
					if err := os.MkdirAll(path, 0o755); err != nil {
						t.Fatal(err)
					}
				} else {
					writeArtifactFixture(t, path, "blocker")
				}
			}
			if tc.readOnly != "" {
				writeArtifactFixture(t, filepath.Join(root, SkillsDir, "fixture", "SKILL.md"), "old")
				writeArtifactFixture(t, filepath.Join(root, SkillsDir, ManifestFile), "fixture\n")
				if tc.name == "obsolete removal" {
					writeArtifactFixture(t, filepath.Join(root, SkillsDir, ManifestFile), "fixture\nobsolete\n")
					writeArtifactFixture(t, filepath.Join(root, tc.readOnly, "SKILL.md"), "old")
				}
				path := filepath.Join(root, tc.readOnly)
				if err := os.Chmod(path, 0o555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.Chmod(path, 0o755); err != nil {
						t.Error(err)
					}
				})
			}
			sources := artifactSources{skills: []Skill{{Name: "fixture", Content: []byte("new")}}}
			err := materializeArtifacts(root, ClaudeTarget, nil, sources)
			if !errors.Is(err, tc.want) || !strings.Contains(err.Error(), tc.context) {
				t.Fatalf("error = %v; want classified failure with context %q", err, tc.context)
			}
		})
	}
}

func writeArtifactFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMaterializeGuidance_RendersBeforeChangingFiles(t *testing.T) {
	t.Parallel()
	for _, source := range []string{"{{aiwf:unknown}}", "{{aiwf:templates_dir}} __AIWF_VERSION__"} {
		t.Run(source, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			err := materializeGuidance(root, []byte(source), "v-fixture")
			if strings.Contains(source, "unknown") {
				if !errors.Is(err, ErrUnknownRenderBinding) {
					t.Fatalf("got %v, want unknown binding", err)
				}
				if got := artifactTree(t, root); len(got) != 0 {
					t.Fatalf("invalid guidance changed filesystem: %v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(root, GuidanceFile))
			if err != nil || string(got) != TemplatesDir+" v-fixture" {
				t.Fatalf("guidance = %q, %v", got, err)
			}
		})
	}
}

func TestRenderGuidance_MatchesMaterializedDefinition(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := materializeGuidance(root, guidanceEmbed, "v-fixture"); err != nil {
		t.Fatal(err)
	}
	want, err := RenderGuidance("v-fixture")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, GuidanceFile))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("guidance reader and writer differ (-want +got):\n%s", diff)
	}
}

func TestMaterialize_RenderedSkillsHaveValidFrontmatterAndNoReservedTokens(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatal(err)
	}
	if err := MaterializeGuidance(root); err != nil {
		t.Fatal(err)
	}
	for path, entry := range artifactTree(t, root) {
		if strings.Contains(entry.Content, renderPrefix) {
			t.Errorf("%s contains an unresolved rendering token", path)
		}
		if filepath.Base(path) != "SKILL.md" {
			continue
		}
		parts := bytes.SplitN([]byte(entry.Content), []byte("---\n"), 3)
		if len(parts) != 3 || len(parts[0]) != 0 {
			t.Errorf("%s: missing frontmatter fences", path)
			continue
		}
		var metadata struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		}
		if err := yaml.Unmarshal(parts[1], &metadata); err != nil {
			t.Errorf("%s: %v", path, err)
		}
		if metadata.Name != filepath.Base(filepath.Dir(path)) || metadata.Description == "" {
			t.Errorf("%s: invalid skill metadata: %+v", path, metadata)
		}
	}
}

type artifactEntry struct {
	Mode    fs.FileMode
	Content string
}

// Includes directories so even creating an empty destination before validation
// is observable. Content and modes also catch overwritten or removed artifacts.
func artifactTree(t *testing.T, root string) map[string]artifactEntry {
	t.Helper()
	out := make(map[string]artifactEntry)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		var content []byte
		if info.Mode()&os.ModeSymlink != 0 {
			var target string
			target, err = os.Readlink(path)
			if err != nil {
				return err
			}
			content = []byte(target)
		} else if !entry.IsDir() {
			content, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[rel] = artifactEntry{Mode: info.Mode(), Content: string(content)}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
