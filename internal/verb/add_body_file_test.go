package verb_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// TestAdd_BodyFile_Epic: BodyOverride content lands as the epic's
// body in the created file (M-056/AC-1, AC-2).
func TestAdd_BodyFile_Epic(t *testing.T) {
	r := newRunner(t)
	bodyText := "## Goal\n\nUser-supplied prose explaining the epic.\n"
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Body-file epic", testActor, verb.AddOptions{
		BodyOverride: []byte(bodyText),
	}))

	got, err := os.ReadFile(filepath.Join(r.root, "work", "epics", "E-01-body-file-epic", "epic.md"))
	if err != nil {
		t.Fatalf("read epic file: %v", err)
	}
	_, body, ok := entity.Split(got)
	if !ok {
		t.Fatalf("epic file has no frontmatter delimiter:\n%s", got)
	}
	if string(body) != bodyText {
		t.Errorf("body = %q, want %q", body, bodyText)
	}
}

// TestAdd_BodyFile_AllKinds is the load-bearing AC-1 check: --body-file
// works for *every* kind, not just one. Table-driven so the verb
// staying kind-agnostic on this seam is pinned by a single regression.
func TestAdd_BodyFile_AllKinds(t *testing.T) {
	cases := []struct {
		name string
		kind entity.Kind
		opts verb.AddOptions
		// glob is the directory glob the entity lands under, used to
		// locate the created file without hand-coding the slug+id.
		glob string
	}{
		{"epic", entity.KindEpic, verb.AddOptions{}, "work/epics/E-*/epic.md"},
		{"milestone", entity.KindMilestone, verb.AddOptions{EpicID: "E-01", TDD: "none"}, "work/epics/E-*/M-*.md"},
		{"adr", entity.KindADR, verb.AddOptions{}, "docs/adr/ADR-*.md"},
		{"gap", entity.KindGap, verb.AddOptions{}, "work/gaps/G-*.md"},
		{"decision", entity.KindDecision, verb.AddOptions{}, "work/decisions/D-*.md"},
		{"contract", entity.KindContract, verb.AddOptions{}, "work/contracts/C-*/contract.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRunner(t)
			// Milestone needs a parent epic on disk; add a default
			// epic for every case so the harness is uniform (it's a
			// no-op for kinds whose Path doesn't depend on it).
			r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Parent epic", testActor, verb.AddOptions{}))

			body := []byte("## " + tc.name + " body\n\nProse content for " + tc.name + ".\n")
			opts := tc.opts
			opts.BodyOverride = body
			r.must(verb.Add(r.ctx, r.tree(), tc.kind, tc.name+" entity", testActor, opts))

			matches, err := filepath.Glob(filepath.Join(r.root, tc.glob))
			if err != nil {
				t.Fatalf("glob %s: %v", tc.glob, err)
			}
			// Pick the most recently created file (the one we just added).
			var path string
			for _, m := range matches {
				if tc.kind == entity.KindEpic && strings.Contains(m, "parent-epic") {
					continue
				}
				path = m
				break
			}
			if path == "" {
				t.Fatalf("no match for %s under %s; got %v", tc.kind, tc.glob, matches)
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			_, gotBody, ok := entity.Split(content)
			if !ok {
				t.Fatalf("%s: no frontmatter:\n%s", tc.kind, content)
			}
			if !bytes.Equal(gotBody, body) {
				t.Errorf("%s body = %q, want %q", tc.kind, gotBody, body)
			}
		})
	}
}

// TestAdd_BodyFile_AbsencePreservesTemplate: opts.BodyOverride == nil
// keeps current behavior — the per-kind template lands as the body
// (M-056/AC-3). Pins the regression that adding the field would
// silently change the default for callers that don't set it.
func TestAdd_BodyFile_AbsencePreservesTemplate(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Default body", testActor, verb.AddOptions{}))

	got, err := os.ReadFile(filepath.Join(r.root, "work", "epics", "E-01-default-body", "epic.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	_, body, ok := entity.Split(got)
	if !ok {
		t.Fatalf("no frontmatter:\n%s", got)
	}
	if !bytes.Equal(body, entity.BodyTemplate(entity.KindEpic)) {
		t.Errorf("default-body epic body = %q, want template %q", body, entity.BodyTemplate(entity.KindEpic))
	}
}

// TestAdd_BodyFile_SingleOpWrite is the AC-4 contract: even with
// BodyOverride set, the resulting Plan still produces exactly one
// OpWrite for the entity file — frontmatter and body land together
// in the same atomic commit, no separate body-edit step.
func TestAdd_BodyFile_SingleOpWrite(t *testing.T) {
	r := newRunner(t)
	res, err := verb.Add(r.ctx, r.tree(), entity.KindEpic, "Atomic body", testActor, verb.AddOptions{
		BodyOverride: []byte("body content\n"),
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("no plan")
	}
	if len(res.Plan.Ops) != 1 {
		t.Fatalf("plan has %d ops, want 1; ops=%+v", len(res.Plan.Ops), res.Plan.Ops)
	}
	if res.Plan.Ops[0].Type != verb.OpWrite {
		t.Errorf("op type = %v, want OpWrite", res.Plan.Ops[0].Type)
	}
	if !strings.Contains(string(res.Plan.Ops[0].Content), "body content") {
		t.Errorf("op content does not include body bytes:\n%s", res.Plan.Ops[0].Content)
	}
}

// TestAdd_BodyFile_RejectsFrontmatter: a file that starts with a
// frontmatter delimiter is refused. Concatenating the verb's
// frontmatter with user-supplied frontmatter would produce a
// double-block file the loader can't parse — better to refuse early
// with a clear message than to silently strip and surprise the user.
func TestAdd_BodyFile_RejectsFrontmatter(t *testing.T) {
	r := newRunner(t)
	bad := []byte("---\nid: PRETEND-1\n---\n\n## Pretend body\n")
	_, err := verb.Add(r.ctx, r.tree(), entity.KindEpic, "Has frontmatter", testActor, verb.AddOptions{
		BodyOverride: bad,
	})
	if err == nil || !strings.Contains(err.Error(), "frontmatter delimiter") {
		t.Errorf("expected frontmatter-delimiter error, got %v", err)
	}
}

// TestAdd_BodyFile_RejectsLeadingWhitespaceFrontmatter: leading
// whitespace before the `---` doesn't smuggle frontmatter past the
// check. The trim is intentional — `---` after a couple of newlines
// would still produce a malformed serialized file.
func TestAdd_BodyFile_RejectsLeadingWhitespaceFrontmatter(t *testing.T) {
	r := newRunner(t)
	bad := []byte("\n\n---\nid: PRETEND-1\n---\n")
	_, err := verb.Add(r.ctx, r.tree(), entity.KindEpic, "Whitespace shield", testActor, verb.AddOptions{
		BodyOverride: bad,
	})
	if err == nil || !strings.Contains(err.Error(), "frontmatter delimiter") {
		t.Errorf("expected frontmatter-delimiter error even with leading whitespace, got %v", err)
	}
}

// TestAdd_BodyFile_PostAddTreeIsClean: an entity created with
// --body-file content runs through `aiwf check` clean. Catches the
// regression where the override accidentally produced a malformed
// file (e.g., missing trailing newline between frontmatter and
// body).
func TestAdd_BodyFile_PostAddTreeIsClean(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Clean body", testActor, verb.AddOptions{
		BodyOverride: []byte("## Goal\n\nClean body content.\n"),
	}))
	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("post-add tree has errors with --body-file content: %+v", findings)
	}
}
