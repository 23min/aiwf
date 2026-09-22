package projectguidance

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/testsupport"
)

const fixtureCatalogue = `{"packs":[{"id":"go/cobra","description":"Go CLI guidance","files":["packs/go/cobra/guide.md"],"detect":["go.mod","*.go"]},{"id":"explicit","description":"Explicit only","files":["packs/explicit/guide.md"],"detect":[]}]}`

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, root, name, content string) {
	t.Helper()
	dest := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func sourceRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "guidance-default")
	write(t, root, "catalogue.json", fixtureCatalogue)
	write(t, root, "packs/go/cobra/guide.md", "# Go\nUse Cobra.\n")
	// An unselected document need not be retrieved.
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "fixture")
	return root
}

func assertEmpty(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary retrieval data remains: %v", entries)
	}
}

func TestRetrieve_CurrentDefaultBranchAndCleanup(t *testing.T) {
	t.Parallel()
	root, temp := sourceRepo(t), t.TempDir()
	for _, content := range []string{"# Go\nUse Cobra.\n", "# Go\nUpdated upstream.\n"} {
		if content != "# Go\nUse Cobra.\n" {
			write(t, root, "packs/go/cobra/guide.md", content)
			git(t, root, "commit", "-qam", "update")
		}
		got, err := retrieve(t.Context(), root, []string{"go/cobra"}, temp)
		if err != nil {
			t.Fatal(err)
		}
		if got.Commit != git(t, root, "rev-parse", "HEAD") {
			t.Fatalf("wrong source revision: %q", got.Commit)
		}
		if diff := cmp.Diff(map[string][]byte{"packs/go/cobra/guide.md": []byte(content)}, got.Documents); diff != "" {
			t.Fatal(diff)
		}
		if len(got.Catalogue.Packs) != 2 || got.Catalogue.Packs[0].ID != "go/cobra" {
			t.Fatalf("lost catalogue: %+v", got.Catalogue)
		}
		assertEmpty(t, temp)
	}
}

func TestRetrieve_EmptySelection(t *testing.T) {
	t.Parallel()
	got, err := Retrieve(t.Context(), sourceRepo(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Documents) != 0 || got.Commit == "" {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
}

func TestRetrieve_RefusalsCleanTemporaryData(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, catalogue, document string
		selection                 []string
	}{
		{name: "unknown selection", selection: []string{"missing"}},
		{name: "duplicate selection", selection: []string{"go/cobra", "go/cobra"}},
		{name: "missing document", selection: []string{"explicit"}},
		{name: "invalid catalogue", catalogue: `{"packs":true}`},
		{name: "empty document", document: " \n", selection: []string{"go/cobra"}},
		{name: "non text document", document: string([]byte{0xff}), selection: []string{"go/cobra"}},
		{name: "nul document", document: "hello\x00world", selection: []string{"go/cobra"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, temp := sourceRepo(t), t.TempDir()
			if tc.catalogue != "" {
				write(t, root, "catalogue.json", tc.catalogue)
			}
			if tc.document != "" {
				write(t, root, "packs/go/cobra/guide.md", tc.document)
			}
			git(t, root, "add", ".")
			git(t, root, "commit", "--allow-empty", "-qm", "invalid source")
			got, err := retrieve(t.Context(), root, tc.selection, temp)
			if !errors.Is(err, ErrInvalidSource) {
				t.Fatalf("got %+v, %v; want invalid source", got, err)
			}
			if got != nil {
				t.Fatal("returned partially validated snapshot")
			}
			assertEmpty(t, temp)
		})
	}
}

func TestRetrieve_UnavailableAndCancelled(t *testing.T) {
	t.Parallel()
	root := sourceRepo(t)
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "unavailable", true: "cancelled"}[cancelled], func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			source := filepath.Join(root, "absent")
			if cancelled {
				cancel()
				source = root
			}
			temp := t.TempDir()
			got, err := retrieve(ctx, source, nil, temp)
			if err == nil || got != nil {
				t.Fatalf("got %+v, %v", got, err)
			}
			if cancelled && !errors.Is(err, context.Canceled) {
				t.Fatalf("lost cancellation identity: %v", err)
			}
			assertEmpty(t, temp)
		})
	}
}

func TestRetrieve_RejectsSymlinks(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"catalogue.json", "packs/go/cobra/guide.md", "packs/go"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root, temp := sourceRepo(t), t.TempDir()
			dest := filepath.Join(root, filepath.FromSlash(name))
			if err := os.RemoveAll(dest); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(t.TempDir(), dest); err != nil {
				t.Fatal(err)
			}
			git(t, root, "add", "-A")
			git(t, root, "commit", "-qm", "symlink")
			got, err := retrieve(t.Context(), root, []string{"go/cobra"}, temp)
			if !errors.Is(err, ErrInvalidSource) || got != nil {
				t.Fatalf("got %+v, %v", got, err)
			}
			assertEmpty(t, temp)
		})
	}
}

func TestRetrieve_TemporaryDirectoryFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "file", "occupied")
	_, err := retrieve(t.Context(), "unused", nil, filepath.Join(root, "file"))
	if !errors.Is(err, syscall.ENOTDIR) {
		t.Fatalf("got %v", err)
	}
}

func TestRetrieve_EmptyRepository(t *testing.T) {
	t.Parallel()
	root, temp := t.TempDir(), t.TempDir()
	git(t, root, "init", "-q", "-b", "empty-default")
	got, err := retrieve(t.Context(), root, nil, temp)
	if err == nil || got != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
	assertEmpty(t, temp)
}

func TestRetrieve_GitConfigurationWithoutCheckout(t *testing.T) {
	root := sourceRepo(t)
	write(t, root, ".gitattributes", "*.md filter=guidance-test\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "attributes")
	marker := filepath.Join(t.TempDir(), "executed")
	t.Setenv("AIWF_TEST_MARKER", marker)
	t.Setenv("GIT_CONFIG_COUNT", "2")
	t.Setenv("GIT_CONFIG_KEY_0", "url."+root+".insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", "fixture:corpus")
	t.Setenv("GIT_CONFIG_KEY_1", "filter.guidance-test.smudge")
	t.Setenv("GIT_CONFIG_VALUE_1", `touch "$AIWF_TEST_MARKER"; cat`)
	got, err := Retrieve(t.Context(), "fixture:corpus", []string{"go/cobra"})
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Documents["packs/go/cobra/guide.md"]) != "# Go\nUse Cobra.\n" {
		t.Fatal("document changed")
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("checkout filter ran: %v", err)
	}
}

func TestRetrieve_CleanupFailureIsReported(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission denial requires non-root")
	}
	temp, bin := t.TempDir(), t.TempDir()
	if err := testsupport.WriteExecutable(filepath.Join(bin, "git"), []byte(`#!/bin/sh
for dest do :; done
mkdir -p "$dest/locked"
printf data > "$dest/locked/file"
chmod 500 "$dest/locked"
exit 1
`)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Cleanup(func() {
		matches, _ := filepath.Glob(filepath.Join(temp, "*", "source", "locked"))
		for _, name := range matches {
			_ = os.Chmod(name, 0o700)
		}
	})
	got, err := retrieve(t.Context(), "fixture", nil, temp)
	if got != nil || !errors.Is(err, os.ErrPermission) {
		t.Fatalf("got %+v, %v", got, err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("cleanup hid original clone error: %v", err)
	}
}

func TestRetrieve_CancelDuringClone(t *testing.T) {
	bin, temp := t.TempDir(), t.TempDir()
	fifo := filepath.Join(t.TempDir(), "ready")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	ready, err := os.OpenFile(fifo, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ready.Close() })
	if err = testsupport.WriteExecutable(filepath.Join(bin, "ssh"), []byte(`#!/bin/sh
printf x > "$AIWF_TEST_READY"
sleep 60
printf survived > "$AIWF_TEST_SURVIVED"
`)); err != nil {
		t.Fatal(err)
	}
	// The SSH transport is the local cancellation fixture, never a network client.
	t.Setenv("GIT_ALLOW_PROTOCOL", "ssh")
	t.Setenv("GIT_SSH_COMMAND", filepath.Join(bin, "ssh"))
	t.Setenv("GIT_SSH_VARIANT", "ssh")
	t.Setenv("AIWF_TEST_READY", fifo)
	survived := filepath.Join(t.TempDir(), "survived")
	t.Setenv("AIWF_TEST_SURVIVED", survived)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	signalled := make(chan error, 1)
	go func() {
		_, readErr := ready.Read(make([]byte, 1))
		cancel()
		signalled <- readErr
	}()
	got, err := retrieve(ctx, "ssh://example.invalid/corpus", nil, temp)
	if !errors.Is(err, context.Canceled) || got != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
	if readErr := <-signalled; readErr != nil {
		t.Fatal(readErr)
	}
	if _, statErr := os.Stat(survived); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("transport ran to completion after cancellation: %v", statErr)
	}
	assertEmpty(t, temp)
}

func TestRetrieve_MultiplePacksAndDocuments(t *testing.T) {
	t.Parallel()
	root := sourceRepo(t)
	catalogue := strings.Replace(fixtureCatalogue, `["packs/go/cobra/guide.md"]`, `["packs/go/cobra/guide.md","packs/go/cobra/reference.md"]`, 1)
	write(t, root, "catalogue.json", catalogue)
	write(t, root, "packs/go/cobra/reference.md", "# Reference\n")
	write(t, root, "packs/explicit/guide.md", "# Explicit\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "multiple selected documents")
	got, err := Retrieve(t.Context(), root, []string{"explicit", "go/cobra"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]byte{
		"packs/go/cobra/guide.md":     []byte("# Go\nUse Cobra.\n"),
		"packs/go/cobra/reference.md": []byte("# Reference\n"),
		"packs/explicit/guide.md":     []byte("# Explicit\n"),
	}
	if diff := cmp.Diff(want, got.Documents); diff != "" {
		t.Fatal(diff)
	}
	wantPacks := []Pack{
		{ID: "go/cobra", Description: "Go CLI guidance", Files: []string{"packs/go/cobra/guide.md", "packs/go/cobra/reference.md"}, Detect: []string{"go.mod", "*.go"}},
		{ID: "explicit", Description: "Explicit only", Files: []string{"packs/explicit/guide.md"}, Detect: []string{}},
	}
	if diff := cmp.Diff(wantPacks, got.Catalogue.Packs); diff != "" {
		t.Fatal(diff)
	}
}

func TestRetrieveWithSelection_UsesOneRevisionAndCleansUp(t *testing.T) {
	t.Parallel()
	source := sourceRepo(t)
	before := git(t, source, "rev-parse", "HEAD")
	got, err := RetrieveWithSelection(t.Context(), source, func(c Catalogue) ([]string, error) {
		if c.Packs[0].ID != "go/cobra" {
			t.Fatalf("catalogue: %+v", c)
		}
		write(t, source, "packs/go/cobra/guide.md", "# Changed upstream\n")
		git(t, source, "commit", "-qam", "new revision")
		return []string{"go/cobra"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Commit != before || string(got.Documents["packs/go/cobra/guide.md"]) != "# Go\nUse Cobra.\n" {
		t.Fatalf("selection refetched a different revision: %+v", got)
	}
}

func TestRetrieveWithSelection_InterruptedChoiceCleansTemporarySource(t *testing.T) {
	t.Parallel()
	source, temp := sourceRepo(t), t.TempDir()
	sentinel := errors.New("choice interrupted")
	got, err := retrieveWithSelection(t.Context(), source, func(Catalogue) ([]string, error) { return nil, sentinel }, temp)
	if got != nil || !errors.Is(err, sentinel) {
		t.Fatalf("interruption: %+v, %v", got, err)
	}
	assertEmpty(t, temp)
}
