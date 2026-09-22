package projectguidance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDetect_ExplainsEveryMatchingPackWithoutSelecting(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	git(t, root, "init", "-q")
	write(t, root, "nested/go.mod", "module example\n")
	write(t, root, "nested/main.go", "package main\n")
	write(t, root, "new/thing.xyz", "source")
	catalogue := Catalogue{Packs: []Pack{
		{ID: "go/a", Description: "First Go policy", Detect: []string{"go.mod", "*.go"}},
		{ID: "go/b", Description: "Another Go policy", Detect: []string{"*.go"}},
		{ID: "new-language", Description: "External addition", Detect: []string{"thing.?yz"}},
		{ID: "explicit", Description: "Manual only"},
		{ID: "wrong-case", Detect: []string{"*.GO"}},
	}}
	got, err := Detect(t.Context(), root, catalogue)
	if err != nil {
		t.Fatal(err)
	}
	want := []Match{
		{PackID: "go/a", Description: "First Go policy", File: "nested/go.mod", Pattern: "go.mod"},
		{PackID: "go/b", Description: "Another Go policy", File: "nested/main.go", Pattern: "*.go"},
		{PackID: "new-language", Description: "External addition", File: "new/thing.xyz", Pattern: "thing.?yz"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatal(diff)
	}
}

func TestDetect_ExcludesIgnoredDependenciesAndOtherRepositories(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	git(t, root, "init", "-q")
	write(t, root, ".gitignore", "ignored/\ntracked.xyz\nbuild/\n")
	write(t, root, "ignored/a.xyz", "ignored")
	write(t, root, "tracked.xyz", "ignored even when tracked")
	git(t, root, "add", "-f", "tracked.xyz")
	for _, dir := range []string{"node_modules", "vendor", ".venv", "venv", "__pycache__", ".svelte-kit", ".next", ".nuxt", ".gradle", ".tox", ".mypy_cache", ".pytest_cache", "build"} {
		write(t, root, "nested/"+dir+"/a.xyz", "excluded")
	}
	write(t, root, "other/a.xyz", "nested repository")
	git(t, root, "add", "other/a.xyz")
	git(t, filepath.Join(root, "other"), "init", "-q")
	write(t, root, "linked/a.xyz", "nested worktree")
	git(t, root, "add", "linked/a.xyz")
	write(t, root, "linked/.git", "gitdir: elsewhere\n")
	write(t, root, "deleted.xyz", "gone")
	git(t, root, "add", "deleted.xyz")
	if err := os.Remove(filepath.Join(root, "deleted.xyz")); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	write(t, outside, "a.xyz", "outside")
	if err := os.Symlink(filepath.Join(outside, "a.xyz"), filepath.Join(root, "link.xyz")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "symlink-dir")); err != nil {
		t.Fatal(err)
	}
	got, err := Detect(t.Context(), root, Catalogue{Packs: []Pack{{ID: "external", Detect: []string{"*.xyz"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("excluded files suggested guidance: %+v", got)
	}
}

func TestDetect_AmbiguousDirectoryNamesRemainEligible(t *testing.T) {
	t.Parallel()
	for _, dir := range []string{"bin", "build", "out", "target", "dist", "obj"} {
		t.Run(dir, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			git(t, root, "init", "-q")
			write(t, root, dir+"/handwritten.xyz", "source")
			got, err := Detect(t.Context(), root, Catalogue{Packs: []Pack{{ID: "external", Detect: []string{"*.xyz"}}}})
			if err != nil {
				t.Fatal(err)
			}
			want := []Match{{PackID: "external", File: dir + "/handwritten.xyz", Pattern: "*.xyz"}}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestDetect_EmptyAndFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	git(t, root, "init", "-q")
	got, err := Detect(t.Context(), root, Catalogue{Packs: []Pack{{ID: "general", Detect: []string{"*"}}}})
	if err != nil || len(got) != 0 {
		t.Fatalf("empty repository: %+v, %v", got, err)
	}
	if _, err = Detect(t.Context(), t.TempDir(), Catalogue{}); err == nil {
		t.Fatal("non-repository accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err = Detect(ctx, root, Catalogue{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestDetectionFile_RejectsMissingAndNonDirectoryParents(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "regular", "not a directory")
	write(t, root, ".git/config", "metadata")
	outside := t.TempDir()
	write(t, outside, "a.xyz", "outside")
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"missing/a.xyz", "regular/a.xyz", "linked/a.xyz", ".git/config"} {
		ok, err := detectionFile(root, name)
		if err != nil || ok {
			t.Fatalf("%s: eligible=%v, error=%v", name, ok, err)
		}
	}
}

func TestDetectionFile_ReportsInspectionFailures(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("permission denial requires non-root")
	}
	root := t.TempDir()
	write(t, root, "blocked/child/a.xyz", "source")
	if err := os.Chmod(filepath.Join(root, "blocked"), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "blocked"), 0o755) })
	for _, name := range []string{"blocked/child/a.xyz", "blocked/a.xyz"} {
		if _, err := detectionFile(root, name); !errors.Is(err, os.ErrPermission) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// A root that cannot be searched fails when inspecting a root-level file.
	if _, err := detectionFile(filepath.Join(root, "blocked"), "a.xyz"); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("root inspection: %v", err)
	}
}

func TestDetect_UsesNestedIgnoreRulesAndLiteralFilenames(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	git(t, root, "init", "-q")
	write(t, root, "nested/.gitignore", "*.xyz\n!keep*.xyz\n")
	write(t, root, "nested/ignored.xyz", "ignored")
	name := "nested/keep space\nand newline.xyz"
	write(t, root, name, "source")
	git(t, root, "add", name)
	got, err := Detect(t.Context(), root, Catalogue{Packs: []Pack{{ID: "external", Detect: []string{"*.xyz"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]Match{{PackID: "external", File: name, Pattern: "*.xyz"}}, got); diff != "" {
		t.Fatal(diff)
	}
}

func TestDetect_ReportsUnreadableCandidate(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("permission denial requires non-root")
	}
	root := t.TempDir()
	git(t, root, "init", "-q")
	write(t, root, "locked/a.xyz", "source")
	git(t, root, "add", "locked/a.xyz")
	if err := os.Chmod(filepath.Join(root, "locked"), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "locked"), 0o755) })
	if _, err := Detect(t.Context(), root, Catalogue{}); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("inspection error: %v", err)
	}
	if _, err := detectionFile(filepath.Join(root, "locked"), "child/a.xyz"); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("parent inspection: %v", err)
	}
}
