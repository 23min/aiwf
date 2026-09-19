package skills

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactOwnership_ReadFailuresCarryFilesystemIdentity(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	target := CodexTarget()
	for _, name := range []string{"path", "manifest", "receipt", "pending content", "provenance"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			restricted := root
			switch name {
			case "manifest":
				restricted = filepath.Join(root, target.SkillsDir, ManifestFile)
				writeArtifactFixture(t, restricted, "first\n")
			case "receipt":
				restricted = filepath.Join(root, target.SkillsDir, PendingManifestFile)
				writeArtifactFixture(t, restricted, pendingHeader+"\n")
			case "pending content":
				restricted = filepath.Join(root, target.SkillsDir, "first", "SKILL.md")
				writeArtifactFixture(t, restricted, "pending")
				writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, PendingManifestFile), pendingHeader+"\n"+artifactDigest([]byte("pending"))+" first\n")
			case "provenance":
				restricted = filepath.Join(root, target.SkillsDir, ProvenanceReadme)
				writeArtifactFixture(t, restricted, provenanceOwnershipMarker+"\n")
			}
			if err := os.Chmod(restricted, 0o000); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(restricted, 0o755) })
			err := materializeArtifacts(root, target, nil, ownershipSources())
			if !errors.Is(err, fs.ErrPermission) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestArtifactOwnership_ApplyFailuresCarryFilesystemIdentity(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	for _, name := range []string{"family directory", "recovered manifest", "receipt", "skill directory", "flat artifact", "final manifest", "receipt cleanup"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			family := artifactFamily{dir: "artifacts"}
			plan := artifactPlan{family: family}
			dir := filepath.Join(root, family.dir)
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			restricted := dir
			want := fs.ErrPermission
			switch name {
			case "family directory":
				restricted = root
				plan.family.dir = "absent"
			case "recovered manifest":
				plan.recovered = true
			case "receipt":
				plan.pending = []artifactReceipt{{name: "first", digest: artifactDigest([]byte("first"))}}
			case "skill directory":
				plan.family.skills = true
				plan.family.files = []Skill{{Name: "first", Content: []byte("first")}}
			case "flat artifact":
				plan.family.files = []Skill{{Name: "first.md", Content: []byte("first")}}
			case "receipt cleanup":
				restricted = ""
				want = nil
				plan.hadPending = true
				writeArtifactFixture(t, filepath.Join(dir, PendingManifestFile, "foreign"), "foreign")
			}
			if restricted != "" {
				if err := os.Chmod(restricted, 0o555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(restricted, 0o755) })
			}
			err := applyArtifactPlan(context.Background(), root, plan)
			if name == "receipt cleanup" {
				var pathErr *os.PathError
				if !errors.As(err, &pathErr) || !strings.Contains(err.Error(), "removing recovery receipt") {
					t.Fatalf("cleanup error = %v", err)
				}
				got, readErr := os.ReadFile(filepath.Join(dir, PendingManifestFile, "foreign"))
				if readErr != nil || string(got) != "foreign" {
					t.Fatalf("cleanup consumed foreign content: %q, %v", got, readErr)
				}
			} else if !errors.Is(err, want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestArtifactOwnership_RetirementFailuresAndMissingFiles(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	for _, name := range []string{"missing flat", "missing skill", "unreadable skill", "empty directory removal"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			skillDir := filepath.Join(root, "retired")
			if name == "unreadable skill" || name == "empty directory removal" {
				writeArtifactFixture(t, filepath.Join(skillDir, "SKILL.md"), "owned")
				restricted, mode := skillDir, fs.FileMode(0o333)
				if name == "empty directory removal" {
					restricted = root
					mode = 0o555
				}
				if err := os.Chmod(restricted, mode); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(restricted, 0o755) })
			}
			err := removeOwnedArtifact(context.Background(), root, "retired", name != "missing flat")
			if strings.HasPrefix(name, "missing") {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, fs.ErrPermission) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestArtifactOwnership_CanceledContextsDoNotWrite(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	root := t.TempDir()
	checks := []func() error{
		func() error { _, err := checkedArtifactPath(ctx, root, "artifact"); return err },
		func() error { return writeOwnershipNames(ctx, root, []string{"first"}) },
		func() error {
			return applyArtifactPlan(ctx, root, artifactPlan{family: artifactFamily{dir: "artifacts"}})
		},
		func() error { return removeOwnedArtifact(ctx, root, "retired", true) },
	}
	for _, check := range checks {
		if err := check(); !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
	}
	if got := artifactTree(t, root); len(got) != 0 {
		t.Fatalf("canceled work wrote files: %v", got)
	}
}

func TestArtifactOwnership_RejectsInvalidPathsAndParentFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, relative := range []string{"", ".", "../escape", "/absolute", "a/../b", "a//b", `a\b`, "C:relative", "nul\x00byte"} {
		if _, err := checkedArtifactPath(context.Background(), root, relative); !errors.Is(err, ErrUnsafeArtifactPath) {
			t.Errorf("%q: %v", relative, err)
		}
	}
	writeArtifactFixture(t, filepath.Join(root, "file"), "user")
	if _, err := checkedArtifactPath(context.Background(), root, "file/child"); !errors.Is(err, ErrUnsafeArtifactPath) {
		t.Fatalf("parent error = %v", err)
	}
}

func TestArtifactOwnership_ValidBasenamesRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	name := "Skill_1-v2.0"
	family := artifactFamily{dir: "artifacts", skills: true, files: []Skill{{Name: name, Content: []byte("generated")}}}
	for range 2 {
		if err := materializeArtifactFamilies(context.Background(), root, []artifactFamily{family}); err != nil {
			t.Fatal(err)
		}
	}
	for relative, want := range map[string]string{name + "/SKILL.md": "generated", ManifestFile: name + "\n"} {
		got, err := os.ReadFile(filepath.Join(root, family.dir, relative))
		if err != nil || string(got) != want {
			t.Fatalf("%s = %q, %v", relative, got, err)
		}
	}
}

func TestArtifactOwnership_RecoverySurvivesChangedSourcesAndRepeatedFailures(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	target := CodexTarget()
	sources := ownershipSources()
	second := filepath.Join(root, target.SkillsDir, "second")
	writeArtifactFixture(t, filepath.Join(second, "SKILL.md"), "old")
	writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, ManifestFile), "second\n")
	if err := os.Chmod(second, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(second, 0o755) })
	for _, content := range []string{"first release", "second release"} {
		sources.skills[0].Content = []byte(content)
		if err := materializeArtifacts(root, target, nil, sources); !errors.Is(err, fs.ErrPermission) {
			t.Fatalf("error = %v", err)
		}
	}
	sources.skills[0].Content = []byte("third release")
	if err := os.Chmod(second, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := materializeArtifacts(root, target, nil, sources); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, target.SkillsDir, "first", "SKILL.md"))
	if err != nil || string(got) != "third release" {
		t.Fatalf("recovered = %q, %v", got, err)
	}
}

func TestArtifactOwnership_RecoversEmptyPendingDirectoriesAndFinishedReceipts(t *testing.T) {
	t.Parallel()
	for _, state := range []string{"absent file", "empty directory", "already owned", "retired pending", "empty receipt", "uppercase digest", "missing receipt at cleanup"} {
		t.Run(state, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			target := CodexTarget()
			sources := ownershipSources()
			receipt := pendingHeader + "\n" + artifactDigest([]byte("old")) + " first\n"
			switch state {
			case "empty receipt":
				receipt = pendingHeader + "\n"
			case "uppercase digest":
				receipt = pendingHeader + "\n" + strings.ToUpper(artifactDigest([]byte("old"))) + " first\n"
				writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, "first", "SKILL.md"), "old")
			case "empty directory":
				if err := os.MkdirAll(filepath.Join(root, target.SkillsDir, "first"), 0o755); err != nil {
					t.Fatal(err)
				}
			case "already owned":
				writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, ManifestFile), "first\n")
				writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, "first", "SKILL.md"), "owned edits")
			case "retired pending":
				receipt = pendingHeader + "\n" + artifactDigest([]byte("old")) + " retired\n"
				writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, "retired", "SKILL.md"), "old")
			case "missing receipt at cleanup":
				if err := applyArtifactPlan(context.Background(), root, artifactPlan{family: artifactFamily{dir: "artifacts"}, hadPending: true}); err != nil {
					t.Fatal(err)
				}
				return
			}
			writeArtifactFixture(t, filepath.Join(root, target.SkillsDir, PendingManifestFile), receipt)
			if err := materializeArtifacts(root, target, nil, sources); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(root, target.SkillsDir, "first", "SKILL.md"))
			if err != nil || !bytes.Equal(got, sources.skills[0].Content) {
				t.Fatalf("recovered = %q, %v", got, err)
			}
			if _, statErr := os.Stat(filepath.Join(root, target.SkillsDir, PendingManifestFile)); !errors.Is(statErr, fs.ErrNotExist) {
				t.Fatalf("receipt remains: %v", statErr)
			}
		})
	}
}
