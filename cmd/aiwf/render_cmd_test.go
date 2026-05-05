package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/gitops"
)

// TestRun_RenderRoadmap_Stdout: a freshly-init'd repo with one epic
// and one milestone produces a markdown table on stdout. No commit
// lands without --write.
func TestRun_RenderRoadmap_Stdout(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-01", "--title", "Schema", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}

	subjectBefore, err := gitops.HeadSubject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"render", "roadmap", "--root", root}); rc != exitOK {
			t.Fatalf("render roadmap: %d", rc)
		}
	})

	out := string(captured)
	for _, want := range []string{
		"# Roadmap",
		"## E-01 — Foundations (proposed)",
		"| M-001 | Schema | draft |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout missing %q:\n%s", want, out)
		}
	}

	// No commit landed.
	if _, statErr := os.Stat(filepath.Join(root, "ROADMAP.md")); !os.IsNotExist(statErr) {
		t.Errorf("ROADMAP.md should not exist after stdout render: err=%v", statErr)
	}
	subjectAfter, err := gitops.HeadSubject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if subjectAfter != subjectBefore {
		t.Errorf("HEAD advanced without --write: %q -> %q", subjectBefore, subjectAfter)
	}
}

// TestRun_RenderRoadmap_WriteCommits: --write writes ROADMAP.md and
// produces a commit with structured trailers. A second --write is a
// no-op (HEAD doesn't advance) because content is unchanged.
func TestRun_RenderRoadmap_WriteCommits(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add: %d", rc)
	}

	if rc := run([]string{"render", "roadmap", "--write", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("render --write: %d", rc)
	}

	body, err := os.ReadFile(filepath.Join(root, "ROADMAP.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Foundations") {
		t.Errorf("ROADMAP.md missing epic title:\n%s", body)
	}

	ctx := context.Background()
	subj, err := gitops.HeadSubject(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if subj != "aiwf render roadmap" {
		t.Errorf("HEAD subject = %q, want %q", subj, "aiwf render roadmap")
	}
	trailers, err := gitops.HeadTrailers(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	wantPairs := map[string]string{
		"aiwf-verb":  "render-roadmap",
		"aiwf-actor": "human/test",
	}
	for _, tr := range trailers {
		if want, ok := wantPairs[tr.Key]; ok {
			if tr.Value != want {
				t.Errorf("trailer %s = %q, want %q", tr.Key, tr.Value, want)
			}
			delete(wantPairs, tr.Key)
		}
	}
	for k := range wantPairs {
		t.Errorf("missing trailer %q", k)
	}

	// Second --write with no tree changes should be a no-op.
	subjectBefore, _ := gitops.HeadSubject(ctx, root)
	if rc := run([]string{"render", "roadmap", "--write", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("re-render --write: %d", rc)
	}
	subjectAfter, _ := gitops.HeadSubject(ctx, root)
	if subjectAfter != subjectBefore {
		t.Errorf("idempotent --write should not advance HEAD: %q -> %q", subjectBefore, subjectAfter)
	}
}

// TestRun_RenderRoadmap_UnknownSubcommand reports a usage error.
func TestRun_RenderRoadmap_UnknownSubcommand(t *testing.T) {
	root := setupCLITestRepo(t)
	if got := run([]string{"render", "treemap", "--root", root}); got != exitUsage {
		t.Errorf("got %d, want %d", got, exitUsage)
	}
	if got := run([]string{"render", "--root", root}); got != exitUsage {
		t.Errorf("got %d, want %d (no subcommand)", got, exitUsage)
	}
}

// TestRun_RenderRoadmap_EmptyRepo prints the empty-tree placeholder
// without errors and without writing a commit.
func TestRun_RenderRoadmap_EmptyRepo(t *testing.T) {
	root := setupCLITestRepo(t)
	captured := captureStdout(t, func() {
		if rc := run([]string{"render", "roadmap", "--root", root}); rc != exitOK {
			t.Fatalf("render: %d", rc)
		}
	})
	if !strings.Contains(string(captured), "_No epics yet._") {
		t.Errorf("empty-tree marker missing:\n%s", captured)
	}
}
