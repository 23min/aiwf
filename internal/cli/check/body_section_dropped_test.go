package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// body_section_dropped_test.go — M-0331/AC-1. The seam test for the push gate:
// a body that reached a commit without passing a body-supplying verb, carrying
// one required section fewer than it had, is refused.
//
// The history shape is the wrap-milestone ritual's. The spec is edited on a
// milestone branch by a plain `git commit` carrying the ritual's three trailers,
// that branch is merged into the epic branch with `--no-ff`, and the epic branch
// is what gets pushed against its upstream. The milestone branch's own first
// push has no upstream and judges nothing, so the push that carries the merge is
// the one that must refuse. The trailers are why the rule cannot filter on them:
// the untrailered audit beside it passes the commit, and it still went through
// no verb.

const (
	specPath = "work/epics/E-0001-seed/M-0001-seed.md"

	specComplete = `---
id: M-0001
title: Seed
status: in_progress
parent: E-0001
---
## Goal

Ship the seed.

## Acceptance criteria

### AC-1 — It ships

It ships.
`

	// specDropped is specComplete with `## Acceptance criteria` removed — the
	// regression the gate refuses.
	specDropped = `---
id: M-0001
title: Seed
status: in_progress
parent: E-0001
---
## Goal

Ship the seed.
`

	// specAlreadyIncomplete never carried `## Acceptance criteria`, and an edit
	// that keeps it that way is not a regression.
	specAlreadyIncomplete = `---
id: M-0001
title: Seed
status: in_progress
parent: E-0001
---
## Goal

Ship the seed.
`

	specAlreadyIncompleteEdited = `---
id: M-0001
title: Seed
status: in_progress
parent: E-0001
---
## Goal

Ship the seed, with feeling.
`
)

// wrapRitualTrailers is the trailer set aiwfx-wrap-milestone stamps on its plain
// `git commit` of the milestone spec.
func wrapRitualTrailers() []gitops.Trailer {
	return []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "wrap-milestone"},
		{Key: gitops.TrailerEntity, Value: "M-0001"},
		{Key: gitops.TrailerActor, Value: "human/tester"},
	}
}

// writeSpec writes the milestone spec and commits it with the supplied message
// and trailers, returning the resulting SHA.
func writeSpec(t *testing.T, ctx context.Context, root, content, msg string, trailers []gitops.Trailer) string {
	t.Helper()
	abs := filepath.Join(root, specPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, specPath); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, msg, "", trailers); err != nil {
		t.Fatal(err)
	}
	return headSHA(t, root)
}

func TestRunProvenanceCheck_BodySectionDropped_WrapRitualMergeIsRefused(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	remote := t.TempDir()
	ctx := context.Background()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	gitRun(t, remote, "init", "-q", "--bare")
	gitRun(t, root, "checkout", "-q", "-b", "epic/E-0001-seed")
	writeSpec(t, ctx, root, specComplete, "aiwf add milestone M-0001",
		[]gitops.Trailer{{Key: gitops.TrailerVerb, Value: "add"}})
	gitRun(t, root, "remote", "add", "origin", remote)
	gitRun(t, root, "push", "-q", "-u", "origin", "epic/E-0001-seed")

	gitRun(t, root, "checkout", "-q", "-b", "milestone/M-0001-seed")
	drop := writeSpec(t, ctx, root, specDropped, "chore(milestone): wrap M-0001", wrapRitualTrailers())
	gitRun(t, root, "checkout", "-q", "epic/E-0001-seed")
	gitRun(t, root, "merge", "-q", "--no-ff", "--no-commit", "milestone/M-0001-seed")
	if err := gitops.Commit(ctx, root, "chore(milestone): wrap M-0001 — Seed", "", wrapRitualTrailers()); err != nil {
		t.Fatalf("merge commit: %v", err)
	}

	findings, err := RunProvenanceCheck(ctx, root, &tree.Tree{}, "",
		map[string]struct{}{"add": {}}, nil, nil, nil, mustHead(t, ctx, root))
	if err != nil {
		t.Fatalf("RunProvenanceCheck: %v", err)
	}
	var found *check.Finding
	for i := range findings {
		if findings[i].Code == check.CodeEntityBodySectionDropped.ID {
			found = &findings[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("%s did not fire on a wrap-ritual merge that dropped a required section; got %+v",
			check.CodeEntityBodySectionDropped.ID, findings)
	}
	if found.Severity != check.SeverityError {
		t.Errorf("Severity = %q, want %q", found.Severity, check.SeverityError)
	}
	if !strings.Contains(found.Message, "Acceptance criteria") {
		t.Errorf("message must name the dropped section; got %q", found.Message)
	}
	if !strings.Contains(found.Message, drop[:8]) {
		t.Errorf("message must name the milestone-branch commit that dropped it (%s), not the merge; got %q", drop[:8], found.Message)
	}
	if found.EntityID != "M-0001" {
		t.Errorf("EntityID = %q, want %q", found.EntityID, "M-0001")
	}
	if found.Path != specPath {
		t.Errorf("Path = %q, want %q", found.Path, specPath)
	}
}

// TestRunProvenanceCheck_BodySectionDropped_KeepingAnExistingOmissionIsNotRefused
// is the other arm of the same rule. An entity whose body never carried a
// required section is edited, and the edit keeps it absent — the author did not
// introduce the omission, so the gate stays silent. Without this arm the rule
// would be completeness, which ADR-0048 rejected at the edit seam and which
// would refuse the commit `aiwf edit-body` had just made.
func TestRunProvenanceCheck_BodySectionDropped_KeepingAnExistingOmissionIsNotRefused(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx := context.Background()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	base := writeSpec(t, ctx, root, specAlreadyIncomplete, "aiwf add milestone M-0001",
		[]gitops.Trailer{{Key: gitops.TrailerVerb, Value: "add"}})
	writeSpec(t, ctx, root, specAlreadyIncompleteEdited,
		"chore(milestone): wrap M-0001", wrapRitualTrailers())

	findings, err := RunProvenanceCheck(ctx, root, &tree.Tree{}, base,
		map[string]struct{}{"add": {}}, nil, nil, nil, mustHead(t, ctx, root))
	if err != nil {
		t.Fatalf("RunProvenanceCheck: %v", err)
	}
	for i := range findings {
		if findings[i].Code == check.CodeEntityBodySectionDropped.ID {
			t.Fatalf("%s fired on an edit that kept an omission it did not introduce: %s",
				check.CodeEntityBodySectionDropped.ID, findings[i].Message)
		}
	}
}

// TestRunProvenanceCheck_BodySectionDropped_RunsWithoutALoadedTree pins that the
// gate does not depend on a loaded tree. RunProvenanceCheck accepts a nil tree,
// and then there is no trunk view to exempt against; the drop is still refused.
func TestRunProvenanceCheck_BodySectionDropped_RunsWithoutALoadedTree(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx := context.Background()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	base := writeSpec(t, ctx, root, specComplete, "aiwf add milestone M-0001",
		[]gitops.Trailer{{Key: gitops.TrailerVerb, Value: "add"}})
	writeSpec(t, ctx, root, specDropped, "chore(milestone): wrap M-0001", wrapRitualTrailers())

	findings, err := RunProvenanceCheck(ctx, root, nil, base,
		map[string]struct{}{"add": {}}, nil, nil, nil, mustHead(t, ctx, root))
	if err != nil {
		t.Fatalf("RunProvenanceCheck: %v", err)
	}
	for i := range findings {
		if findings[i].Code == check.CodeEntityBodySectionDropped.ID {
			return
		}
	}
	t.Fatalf("%s did not fire without a loaded tree; got %+v", check.CodeEntityBodySectionDropped.ID, findings)
}
