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
// The commit shape is the wrap-milestone ritual's — a plain `git commit` of the
// milestone spec carrying the ritual's three trailers. That is the path the gate
// exists for, and it is why the rule cannot filter on trailer presence: the
// commit carries `aiwf-verb`, `aiwf-entity` and `aiwf-actor`, so the untrailered
// audit beside this rule passes it, and it still went through no verb.

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

func TestRunProvenanceCheck_BodySectionDropped_WrapRitualCommitIsRefused(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx := context.Background()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	base := writeSpec(t, ctx, root, specComplete, "aiwf add milestone M-0001",
		[]gitops.Trailer{{Key: gitops.TrailerVerb, Value: "add"}})
	writeSpec(t, ctx, root, specDropped,
		"chore(milestone): wrap M-0001", wrapRitualTrailers())

	findings, err := RunProvenanceCheck(ctx, root, &tree.Tree{}, base,
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
		t.Fatalf("%s did not fire on a wrap-ritual commit that dropped a required section; got %d findings",
			check.CodeEntityBodySectionDropped.ID, len(findings))
	}
	if found.Severity != check.SeverityError {
		t.Errorf("Severity = %q, want %q", found.Severity, check.SeverityError)
	}
	if !strings.Contains(found.Message, "Acceptance criteria") {
		t.Errorf("message must name the dropped section; got %q", found.Message)
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
