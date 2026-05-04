package trunk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/ai-workflow-v2/tools/internal/config"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
)

func TestRead_NoRemotes_Skips(t *testing.T) {
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-001-foo.md", "# foo\n")

	res, err := Read(ctx, dir, nil)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !res.Skipped {
		t.Error("Skipped = false, want true (no remotes configured)")
	}
	if len(res.IDs) != 0 {
		t.Errorf("IDs = %v, want empty when skipped", res.IDs)
	}
}

func TestRead_RemoteAndDefaultTrunk_ReturnsIDs(t *testing.T) {
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-001-foo.md", "# foo\n")
	commitFile(t, ctx, dir, "docs/adr/ADR-0001-baz.md", "# baz\n")
	commitFile(t, ctx, dir, "README.md", "readme\n")
	mustRun(t, ctx, dir, "remote", "add", "origin", "https://example.invalid/x.git")
	// Mirror HEAD as the default trunk ref so Read finds it.
	mustRun(t, ctx, dir, "update-ref", config.DefaultAllocateTrunk, "HEAD")

	res, err := Read(ctx, dir, nil)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if res.Skipped {
		t.Error("Skipped = true, want false")
	}
	want := []ID{
		{Kind: entity.KindADR, ID: "ADR-0001", Path: "docs/adr/ADR-0001-baz.md"},
		{Kind: entity.KindGap, ID: "G-001", Path: "work/gaps/G-001-foo.md"},
	}
	if diff := cmp.Diff(want, res.IDs); diff != "" {
		t.Errorf("IDs mismatch (-want +got):\n%s", diff)
	}
}

func TestRead_RemoteButTrunkMissing_HardError(t *testing.T) {
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "README.md", "readme\n")
	mustRun(t, ctx, dir, "remote", "add", "origin", "https://example.invalid/x.git")

	_, err := Read(ctx, dir, nil)
	if err == nil {
		t.Fatal("Read: expected error for missing default trunk with remote configured, got nil")
	}
	if !strings.Contains(err.Error(), config.DefaultAllocateTrunk) {
		t.Errorf("error %q should mention the missing ref %q", err, config.DefaultAllocateTrunk)
	}
	if !strings.Contains(err.Error(), "allocate.trunk") {
		t.Errorf("error %q should hint at allocate.trunk config", err)
	}
}

func TestRead_ExplicitTrunk_UsedInsteadOfDefault(t *testing.T) {
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-007-explicit.md", "# explicit\n")
	mustRun(t, ctx, dir, "remote", "add", "origin", "https://example.invalid/x.git")
	mustRun(t, ctx, dir, "update-ref", "refs/remotes/origin/develop", "HEAD")

	cfg := &config.Config{Allocate: config.Allocate{Trunk: "refs/remotes/origin/develop"}}
	res, err := Read(ctx, dir, cfg)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if res.Skipped {
		t.Error("Skipped = true, want false")
	}
	want := []ID{{Kind: entity.KindGap, ID: "G-007", Path: "work/gaps/G-007-explicit.md"}}
	if diff := cmp.Diff(want, res.IDs); diff != "" {
		t.Errorf("IDs mismatch (-want +got):\n%s", diff)
	}
}

func TestRead_ExplicitTrunkMissing_HardError(t *testing.T) {
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "README.md", "readme\n")
	mustRun(t, ctx, dir, "remote", "add", "origin", "https://example.invalid/x.git")

	cfg := &config.Config{Allocate: config.Allocate{Trunk: "refs/remotes/origin/typo"}}
	_, err := Read(ctx, dir, cfg)
	if err == nil {
		t.Fatal("Read: expected error for missing explicit trunk, got nil")
	}
	if !strings.Contains(err.Error(), "refs/remotes/origin/typo") {
		t.Errorf("error %q should mention the missing ref", err)
	}
}

func TestResult_IDStrings(t *testing.T) {
	r := Result{IDs: []ID{
		{Kind: entity.KindGap, ID: "G-001", Path: "work/gaps/G-001-foo.md"},
		{Kind: entity.KindADR, ID: "ADR-0001", Path: "docs/adr/ADR-0001-baz.md"},
	}}
	got := r.IDStrings()
	want := []string{"G-001", "ADR-0001"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("IDStrings mismatch (-want +got):\n%s", diff)
	}

	if (Result{}).IDStrings() != nil {
		t.Error("empty Result.IDStrings should be nil")
	}
}

// initRepo / commitFile / mustRun mirror the helpers in
// gitops/refs_test.go; duplicated here so this package's tests don't
// depend on internal-test-helper exports from gitops.
func initRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.invalid")
	dir := t.TempDir()
	if err := gitops.Init(context.Background(), dir); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return dir
}

func commitFile(t *testing.T, ctx context.Context, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	mustRun(t, ctx, dir, "add", "--", path)
	mustRun(t, ctx, dir, "commit", "-q", "-m", "add "+path)
}

func mustRun(t *testing.T, ctx context.Context, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
