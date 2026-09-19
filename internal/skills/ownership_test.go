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
)

func ownershipSources() artifactSources {
	return artifactSources{
		skills:    []Skill{{Name: "first", Content: []byte("first native skill")}, {Name: "second", Content: []byte("second native skill")}},
		agents:    []Skill{{Name: "agent.md", Content: []byte("native agent")}},
		templates: []Skill{{Name: "template.md", Content: []byte("native template")}},
	}
}

func TestArtifactOwnership_RefusesForeignNamesBeforeAnyWrites(t *testing.T) {
	t.Parallel()
	for _, target := range []Target{ClaudeTarget, CodexTarget()} {
		paths := []string{target.SkillsDir + "/first", target.SkillsDir + "/first/SKILL.md", target.TemplatesDir + "/template.md", target.SkillsDir + "/" + ProvenanceReadme}
		if target.AgentsDir != "" {
			paths = append(paths, target.AgentsDir+"/agent.md")
		}
		for _, relative := range paths {
			for _, directory := range []bool{false, true} {
				t.Run(target.Name+"/"+relative+map[bool]string{false: "/file", true: "/directory"}[directory], func(t *testing.T) {
					t.Parallel()
					root := t.TempDir()
					path := filepath.Join(root, relative)
					if directory {
						if err := os.MkdirAll(path, 0o750); err != nil {
							t.Fatal(err)
						}
					} else {
						writeArtifactFixture(t, path, "foreign data")
					}
					before := artifactTree(t, root)
					err := materializeArtifacts(root, target, nil, ownershipSources())
					if !errors.Is(err, ErrOwnershipConflict) {
						t.Fatalf("error = %v, want ownership collision", err)
					}
					if !strings.Contains(err.Error(), relative) && !strings.Contains(relative, target.SkillsDir+"/first/") {
						t.Fatalf("missing collision path: %v", err)
					}
					if diff := cmp.Diff(before, artifactTree(t, root)); diff != "" {
						t.Fatalf("collision changed files: %s", diff)
					}
				})
			}
		}
	}
}

func TestArtifactOwnership_InvalidManifestsCannotDeleteOrOverwrite(t *testing.T) {
	t.Parallel()
	for _, family := range []string{CodexTarget().SkillsDir, CodexTarget().TemplatesDir} {
		for _, record := range []string{"../victim", "/tmp/victim", ".", "..", `..\victim`, "nested/victim", ".aiwf-owned", ".aiwf-pending", "name\x00bad", " first", "first\nfirst", "first\r", "C:relative"} {
			t.Run(family+"/"+record, func(t *testing.T) {
				t.Parallel()
				root := t.TempDir()
				writeArtifactFixture(t, filepath.Join(root, family, ManifestFile), record+"\n")
				writeArtifactFixture(t, filepath.Join(root, "victim", "SKILL.md"), "user data")
				before := artifactTree(t, root)
				err := materializeArtifacts(root, CodexTarget(), nil, ownershipSources())
				if !errors.Is(err, ErrInvalidOwnership) {
					t.Fatalf("error = %v", err)
				}
				if diff := cmp.Diff(before, artifactTree(t, root)); diff != "" {
					t.Fatalf("invalid manifest changed files: %s", diff)
				}
			})
		}
	}
}

func TestArtifactOwnership_RejectsLinkedPathsAndPreservesTargets(t *testing.T) {
	t.Parallel()
	target := CodexTarget()
	for _, relative := range []string{".agents", target.SkillsDir, target.TemplatesDir, target.SkillsDir + "/first", target.SkillsDir + "/first/SKILL.md", target.SkillsDir + "/" + ManifestFile, target.SkillsDir + "/" + PendingManifestFile, target.SkillsDir + "/" + ProvenanceReadme, target.TemplatesDir + "/template.md", target.TemplatesDir + "/obsolete.md"} {
		t.Run(relative, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			external := t.TempDir()
			writeArtifactFixture(t, filepath.Join(external, "sentinel"), "outside data")
			beforeExternal := artifactTree(t, external)
			path := filepath.Join(root, relative)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(external, path); err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(relative, target.SkillsDir+"/first/") {
				writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, ManifestFile), "first\n")
			}
			if strings.HasSuffix(relative, "obsolete.md") {
				writeArtifactFixture(t, filepath.Join(root, target.TemplatesDir, ManifestFile), "obsolete.md\n")
			}
			before := artifactTree(t, root)
			err := materializeArtifacts(root, target, nil, ownershipSources())
			if !errors.Is(err, ErrUnsafeArtifactPath) {
				t.Fatalf("error = %v", err)
			}
			if diff := cmp.Diff(before, artifactTree(t, root)); diff != "" {
				t.Fatalf("link changed: %s", diff)
			}
			if diff := cmp.Diff(beforeExternal, artifactTree(t, external)); diff != "" {
				t.Fatalf("target changed: %s", diff)
			}
		})
	}
}

func TestArtifactOwnership_RejectsUnsafeNamesAndOverlappingRoots(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"../outside", "nested/file", ".aiwf-owned", "", `C:\outside`, ProvenanceReadme, "second"} {
		t.Run("source/"+name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			sources := ownershipSources()
			sources.skills[0].Name = name
			if err := materializeArtifacts(root, CodexTarget(), nil, sources); !errors.Is(err, ErrUnsafeArtifactPath) {
				t.Fatalf("error = %v", err)
			}
			if got := artifactTree(t, root); len(got) != 0 {
				t.Fatalf("unsafe source wrote files: %v", got)
			}
		})
	}
	for _, directory := range []string{"../escape", ".", ".agents", ".agents/skills", ".agents/skills/nested", `.agents\outside`} {
		t.Run("root/"+directory, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			target := CodexTarget()
			target.TemplatesDir = directory
			if err := materializeArtifacts(root, target, nil, ownershipSources()); !errors.Is(err, ErrUnsafeArtifactPath) {
				t.Fatalf("error = %v", err)
			}
			if got := artifactTree(t, root); len(got) != 0 {
				t.Fatalf("unsafe target wrote files: %v", got)
			}
		})
	}
}

func TestArtifactOwnership_RefusesUnsafePendingArtifacts(t *testing.T) {
	t.Parallel()
	for _, state := range []string{"skill replaced by file", "linked content", "flat replaced by directory"} {
		t.Run(state, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			target := CodexTarget()
			family, name := target.SkillsDir, "first"
			want := ErrOwnershipConflict
			switch state {
			case "skill replaced by file":
				writeArtifactFixture(t, filepath.Join(root, family, name), "foreign")
			case "linked content":
				want = ErrUnsafeArtifactPath
				if err := os.MkdirAll(filepath.Join(root, family, name), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("missing", filepath.Join(root, family, name, "SKILL.md")); err != nil {
					t.Fatal(err)
				}
			case "flat replaced by directory":
				family, name = target.TemplatesDir, "template.md"
				writeArtifactFixture(t, filepath.Join(root, family, name, "foreign"), "foreign")
			}
			writeArtifactFixture(t, filepath.Join(root, family, PendingManifestFile), pendingHeader+"\n"+artifactDigest([]byte("expected"))+" "+name+"\n")
			before := artifactTree(t, root)
			if err := materializeArtifacts(root, target, nil, ownershipSources()); !errors.Is(err, want) {
				t.Fatalf("error = %v", err)
			}
			if diff := cmp.Diff(before, artifactTree(t, root)); diff != "" {
				t.Fatalf("unsafe recovery changed files: %s", diff)
			}
		})
	}
}

func TestArtifactOwnership_RefreshesOwnedFilesAndRetiresOnlyOwnedContent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	target := CodexTarget()
	sources := ownershipSources()
	if err := materializeArtifacts(root, target, nil, sources); err != nil {
		t.Fatal(err)
	}
	writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, "first", "SKILL.md"), "owned edit")
	writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, "second", "notes.md"), "user notes")
	writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, "foreign", "SKILL.md"), "foreign skill")
	writeArtifactFixture(t, filepath.Join(root, target.TemplatesDir, "foreign.md"), "foreign template")
	sources.skills = sources.skills[:1]
	sources.templates = nil
	if err := materializeArtifacts(root, target, nil, sources); err != nil {
		t.Fatal(err)
	}
	for relative, want := range map[string]string{
		target.SkillsDir + "/first/SKILL.md": "first native skill", target.SkillsDir + "/second/notes.md": "user notes",
		target.SkillsDir + "/foreign/SKILL.md": "foreign skill", target.TemplatesDir + "/foreign.md": "foreign template",
	} {
		got, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil || string(got) != want {
			t.Fatalf("%s = %q, %v", relative, got, err)
		}
	}
	for _, relative := range []string{target.SkillsDir + "/second/SKILL.md", target.TemplatesDir + "/template.md", target.SkillsDir + "/" + PendingManifestFile, target.TemplatesDir + "/" + PendingManifestFile} {
		if _, err := os.Lstat(filepath.Join(root, relative)); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("retained %s: %v", relative, err)
		}
	}
	before := artifactTree(t, root)
	if err := materializeArtifacts(root, target, nil, sources); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(before, artifactTree(t, root)); diff != "" {
		t.Fatalf("repeat changed output: %s", diff)
	}
}

func TestArtifactOwnership_RecoversPartialWritesWithoutClaimingForeignData(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	for _, tamper := range []bool{false, true} {
		t.Run(map[bool]string{false: "recover", true: "foreign replacement"}[tamper], func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			target := CodexTarget()
			sources := ownershipSources()
			second := filepath.Join(root, target.SkillsDir, "second")
			writeArtifactFixture(t, filepath.Join(second, "SKILL.md"), "old owned content")
			writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, ManifestFile), "second\n")
			writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, "foreign", "SKILL.md"), "foreign")
			if err := os.Chmod(second, 0o555); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(second, 0o755) })
			if err := materializeArtifacts(root, target, nil, sources); !errors.Is(err, fs.ErrPermission) {
				t.Fatalf("write failure = %v", err)
			}
			first := filepath.Join(root, target.SkillsDir, "first", "SKILL.md")
			got, err := os.ReadFile(first)
			if err != nil || !bytes.Equal(got, sources.skills[0].Content) {
				t.Fatalf("no partial write: %q, %v", got, err)
			}
			if _, statErr := os.Stat(filepath.Join(root, target.SkillsDir, PendingManifestFile)); statErr != nil {
				t.Fatal(statErr)
			}
			if chmodErr := os.Chmod(second, 0o755); chmodErr != nil {
				t.Fatal(chmodErr)
			}
			if tamper {
				writeArtifactFixture(t, first, "foreign replacement")
				before := artifactTree(t, root)
				if retryErr := materializeArtifacts(root, target, nil, sources); !errors.Is(retryErr, ErrOwnershipConflict) {
					t.Fatalf("retry = %v", retryErr)
				}
				if diff := cmp.Diff(before, artifactTree(t, root)); diff != "" {
					t.Fatalf("retry consumed foreign content: %s", diff)
				}
				return
			}
			if retryErr := materializeArtifacts(root, target, nil, sources); retryErr != nil {
				t.Fatal(retryErr)
			}
			expectedRoot := t.TempDir()
			writeArtifactFixture(t, filepath.Join(expectedRoot, target.SkillsDir, "foreign", "SKILL.md"), "foreign")
			if cleanErr := materializeArtifacts(expectedRoot, target, nil, sources); cleanErr != nil {
				t.Fatal(cleanErr)
			}
			if diff := cmp.Diff(artifactTree(t, expectedRoot), artifactTree(t, root)); diff != "" {
				t.Fatalf("recovery differs from clean install: %s", diff)
			}
		})
	}
}

func TestArtifactOwnership_RejectsMalformedRecoveryRecords(t *testing.T) {
	t.Parallel()
	digest := strings.Repeat("0", 64)
	for _, record := range []string{"", "foreign text\n", "aiwf-pending-v2\n", pendingHeader + "\nshort first\n", pendingHeader + "\n" + digest + " ../outside\n", pendingHeader + "\n" + digest + " first\n" + digest + " first\n", pendingHeader + "\n" + strings.Repeat("z", 64) + " first\n", pendingHeader + "\n" + digest + "\n"} {
		t.Run(record, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			target := CodexTarget()
			writeArtifactFixture(t, filepath.Join(root, target.TemplatesDir, PendingManifestFile), record)
			before := artifactTree(t, root)
			if err := materializeArtifacts(root, target, nil, ownershipSources()); !errors.Is(err, ErrInvalidOwnership) {
				t.Fatalf("error = %v", err)
			}
			if diff := cmp.Diff(before, artifactTree(t, root)); diff != "" {
				t.Fatalf("invalid receipt changed files: %s", diff)
			}
		})
	}
}

func TestArtifactOwnership_ProvenanceRequiresStandaloneOwnershipMarker(t *testing.T) {
	t.Parallel()
	for _, body := range []string{"<!-- Generated by aiwf; just an example -->\n", "Quoted `" + provenanceOwnershipMarker + "` example.\n"} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			target := CodexTarget()
			writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, ProvenanceReadme), body)
			before := artifactTree(t, root)
			if err := materializeArtifacts(root, target, nil, ownershipSources()); !errors.Is(err, ErrOwnershipConflict) {
				t.Fatalf("error = %v", err)
			}
			if diff := cmp.Diff(before, artifactTree(t, root)); diff != "" {
				t.Fatalf("foreign README changed: %s", diff)
			}
		})
	}
}
