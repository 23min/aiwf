package projectguidance

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func installationFixture(t *testing.T) (*Snapshot, InstallOptions) {
	t.Helper()
	source := sourceRepo(t)
	snapshot, err := Retrieve(t.Context(), source, []string{"go/cobra"})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot, InstallOptions{Source: source, Selected: []string{"go/cobra"}, HostFiles: []string{"AGENTS.md", "CLAUDE.md"}}
}

func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(name string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestInstall_PreservesOverridesAndConverges(t *testing.T) {
	t.Parallel()
	snapshot, opts := installationFixture(t)
	root := t.TempDir()
	write(t, root, ".guidance/project.md", "# Our exceptions\n")
	write(t, root, "AGENTS.md", "# Personal project instructions\n")
	changed, err := Install(t.Context(), root, snapshot, opts)
	if err != nil || !changed {
		t.Fatalf("install: changed=%v, %v", changed, err)
	}
	files := readTree(t, root)
	if files[".guidance/packs/go/cobra/guide.md"] != "# Go\nUse Cobra.\n" || files[".guidance/project.md"] != "# Our exceptions\n" {
		t.Fatalf("wrong documents: %v", files)
	}
	if !strings.Contains(files[".guidance/index.md"], snapshot.Commit) || !strings.Contains(files[".guidance/index.md"], "<!-- aiwf:engineering-guidance -->") {
		t.Fatal("missing installed source/ownership")
	}
	for _, host := range opts.HostFiles {
		if !strings.Contains(files[host], ".guidance/project.md") || !strings.Contains(files[host], ".guidance/index.md") {
			t.Fatalf("missing local routing in %s", host)
		}
	}
	if !strings.HasPrefix(files["AGENTS.md"], "# Personal project instructions\n") {
		t.Fatal("handwritten instructions changed")
	}
	changed, err = Install(t.Context(), root, snapshot, opts)
	if err != nil || changed {
		t.Fatalf("repeat: changed=%v, %v", changed, err)
	}
	if diff := cmp.Diff(files, readTree(t, root)); diff != "" {
		t.Fatal(diff)
	}
}

func TestInstall_ConflictsDoNotChangeGuidance(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"foreign document", "foreign index", "edited document", "edited index", "edited route", "malformed route", "linked directory"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			snapshot, opts := installationFixture(t)
			root := t.TempDir()
			if strings.HasPrefix(kind, "edited") {
				if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "foreign document", "edited document":
				write(t, root, ".guidance/packs/go/cobra/guide.md", "mine\n")
			case "foreign index", "edited index":
				write(t, root, ".guidance/index.md", "mine\n")
			case "edited route":
				content, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
				if err != nil {
					t.Fatal(err)
				}
				write(t, root, "AGENTS.md", strings.Replace(string(content), ".guidance/index.md", "elsewhere.md", 1))
			case "malformed route":
				write(t, root, "AGENTS.md", "<!-- aiwf:engineering-guidance:END -->\n")
			case "linked directory":
				if err := os.Symlink(t.TempDir(), filepath.Join(root, ".guidance")); err != nil {
					t.Fatal(err)
				}
			}
			// Snapshot only regular files; the symlink fixture has no owned content yet.
			var before map[string]string
			if kind != "linked directory" {
				before = readTree(t, root)
			}
			if _, err := Install(t.Context(), root, snapshot, opts); err == nil {
				t.Fatal("accepted unsafe installation")
			}
			if before != nil {
				if diff := cmp.Diff(before, readTree(t, root)); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}

func TestInstall_EmptySelectionStillOwnsPolicy(t *testing.T) {
	t.Parallel()
	snapshot, opts := installationFixture(t)
	opts.Selected = nil
	snapshot.Documents = nil
	root := t.TempDir()
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".guidance", "index.md"))
	if err != nil || !strings.Contains(string(data), "<!-- aiwf:engineering-guidance -->") {
		t.Fatalf("empty policy: %s %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".guidance", "packs")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty selection wrote packs: %v", err)
	}
}

func TestInstall_RecoversAfterRepeatedInterruption(t *testing.T) {
	t.Parallel()
	snapshot, opts := installationFixture(t)
	root := t.TempDir()
	desired, err := renderInstallation(snapshot, opts)
	if err != nil {
		t.Fatal(err)
	}
	writes, _, err := planInstallation(t.Context(), root, desired, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	pending := make(map[string]receipt)
	for _, operation := range writes {
		pending[operation.path] = operation.receipt
	}
	// A first interrupted installation has published one document, with no final
	// ownership record yet. Every path still has its original recovery receipt.
	document := ".guidance/packs/go/cobra/guide.md"
	write(t, root, document, string(desired[document]))
	if err = writeInstallRecord(root, pendingFile, pending); err != nil {
		t.Fatal(err)
	}
	writes, owned, err := planInstallation(t.Context(), root, desired, nil, pending)
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := stageInstallation(t.Context(), root, writes, pending)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err = publishInstallation(ctx, root, writes, owned, replacement); !errors.Is(err, context.Canceled) {
		t.Fatalf("interruption: %v", err)
	}
	// Recovery installs the current upstream revision, not a cached interrupted one.
	write(t, opts.Source, "packs/go/cobra/guide.md", "# Updated during interruption\n")
	git(t, opts.Source, "commit", "-qam", "new upstream revision")
	snapshot, err = Retrieve(t.Context(), opts.Source, opts.Selected)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatalf("second recovery: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, pendingFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pending receipt remains: %v", err)
	}
	clean := t.TempDir()
	if _, err := Install(t.Context(), clean, snapshot, opts); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(readTree(t, clean), readTree(t, root)); diff != "" {
		t.Fatal(diff)
	}
}

func TestInstall_DoesNotRecreateAnUnselectedHostFile(t *testing.T) {
	t.Parallel()
	snapshot, opts := installationFixture(t)
	root := t.TempDir()
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	opts.HostFiles = []string{"CLAUDE.md"}
	before := readTree(t, root)
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(before, readTree(t, root)); diff != "" {
		t.Fatal(diff)
	}
}

func TestInstall_RecoversRecognizedIntermediateBytes(t *testing.T) {
	t.Parallel()
	snapshot, opts := installationFixture(t)
	root := t.TempDir()
	document := ".guidance/packs/go/cobra/guide.md"
	intermediate := "# Previously published intermediate revision\n"
	write(t, root, document, intermediate)
	pending := map[string]receipt{document: {Before: digest([]byte(intermediate)), After: digest([]byte("next interrupted revision"))}}
	if err := writeInstallRecord(root, pendingFile, pending); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatalf("recognized intermediate content refused: %v", err)
	}
	installed, err := os.ReadFile(filepath.Join(root, document))
	if err != nil || !bytes.Equal(installed, snapshot.Documents["packs/go/cobra/guide.md"]) {
		t.Fatalf("installed %q, %v", installed, err)
	}
}
