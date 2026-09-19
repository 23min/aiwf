package initrepo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostRefreshErrors_BlockedFilesPropagate(t *testing.T) {
	t.Parallel()
	for _, relative := range []string{".gitignore", ".git/hooks/pre-push", ".git/hooks/commit-msg", ".git/hooks/post-commit"} {
		t.Run(relative, func(t *testing.T) {
			t.Parallel()
			root := freshGitRepo(t)
			if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: []\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(root, relative), 0o700); err != nil {
				t.Fatal(err)
			}
			_, err := RefreshArtifacts(t.Context(), root, RefreshOptions{StatusMdAutoUpdate: true})
			if err == nil || !strings.Contains(err.Error(), filepath.Base(relative)) {
				t.Fatalf("blocked %s: error=%v", relative, err)
			}
		})
	}
}

func TestHostRefreshErrors_LegacyWritesFailWithoutChangingConfig(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	for _, legacy := range []string{"actor: human/test\n", "aiwf_version: v0.1.0\n"} {
		t.Run(strings.TrimSpace(legacy), func(t *testing.T) {
			t.Parallel()
			root := freshGitRepo(t)
			content := "hosts: []\n" + legacy
			path := filepath.Join(root, "aiwf.yaml")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(root, 0o500); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(root, 0o700) })
			_, err := RefreshArtifacts(t.Context(), root, RefreshOptions{SkipHooks: true})
			if !errors.Is(err, os.ErrPermission) {
				t.Fatalf("expected permission error: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != content {
				t.Fatalf("config changed: %q, %v", got, err)
			}
		})
	}
}

func TestHostRefreshErrors_InitReportsInstructionWriteFailure(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := freshGitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(t.Context(), root, Options{SkipHook: true}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o700) })
	_, err := Init(t.Context(), root, Options{SkipHook: true})
	if !errors.Is(err, os.ErrPermission) || !strings.Contains(err.Error(), "CLAUDE.md") {
		t.Fatalf("instruction write failure = %v", err)
	}
}
