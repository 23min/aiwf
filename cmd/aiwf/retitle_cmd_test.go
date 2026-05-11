package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-077/AC-1: `aiwf retitle <id> "<new-title>" [--reason ...]` updates
// the entity's frontmatter `title:` field for any of the six top-level
// kinds. Per G-0108, the on-disk slug is also re-derived from the new
// title in the same commit, so frontmatter and filesystem stay in sync.
// Closes the top-level half of G-065 + G-0108.

// retitleSetup gives every test in this file a freshly-init'd repo
// with one entity per top-level kind so the verb has live targets to
// retitle.
func retitleSetup(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "First Milestone", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "adr", "--title", "First ADR", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add adr: %d", rc)
	}
	if rc := run([]string{"add", "gap", "--title", "First Gap", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add gap: %d", rc)
	}
	if rc := run([]string{"add", "decision", "--title", "First Decision", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add decision: %d", rc)
	}
	if rc := run([]string{"add", "contract", "--title", "First Contract", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add contract: %d", rc)
	}
	return root
}

// TestRetitle_AllKinds pins AC-1: retitle works for every top-level
// kind, updating the frontmatter `title:` field AND the on-disk slug
// in one commit. The `wantPath` reflects the post-rename location
// (G-0108) — the slug is re-derived from the new title.
func TestRetitle_AllKinds(t *testing.T) {
	cases := []struct {
		name        string
		id          string
		newTitle    string
		wantPath    string
		oldPathGone string
	}{
		{"epic", "E-0001", "Refocused Foundations", "work/epics/E-0001-refocused-foundations/epic.md", "work/epics/E-0001-foundations/epic.md"},
		{"milestone", "M-0001", "Refocused First Milestone", "work/epics/E-0001-foundations/M-0001-refocused-first-milestone.md", "work/epics/E-0001-foundations/M-0001-first-milestone.md"},
		{"adr", "ADR-0001", "Refocused First ADR", "docs/adr/ADR-0001-refocused-first-adr.md", "docs/adr/ADR-0001-first-adr.md"},
		{"gap", "G-0001", "Refocused First Gap", "work/gaps/G-0001-refocused-first-gap.md", "work/gaps/G-0001-first-gap.md"},
		{"decision", "D-0001", "Refocused First Decision", "work/decisions/D-0001-refocused-first-decision.md", "work/decisions/D-0001-first-decision.md"},
		{"contract", "C-0001", "Refocused First Contract", "work/contracts/C-0001-refocused-first-contract/contract.md", "work/contracts/C-0001-first-contract/contract.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := retitleSetup(t)

			rc := run([]string{
				"retitle", tc.id, tc.newTitle,
				"--actor", "human/test",
				"--root", root,
			})
			if rc != exitOK {
				t.Fatalf("retitle %s: %d", tc.id, rc)
			}

			body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(tc.wantPath)))
			if err != nil {
				t.Fatalf("read %s: %v", tc.wantPath, err)
			}
			want := "title: " + tc.newTitle
			if !strings.Contains(string(body), want) {
				t.Errorf("frontmatter missing %q after retitle:\n%s", want, body)
			}
			// G-0108: the OLD path must no longer exist; retitle moved
			// the file as part of the same commit, not left a copy.
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(tc.oldPathGone))); !os.IsNotExist(err) {
				t.Errorf("old path %q still exists after retitle (slug should have moved); stat err = %v", tc.oldPathGone, err)
			}
		})
	}
}

// TestRetitle_Reason pins AC-1's --reason flag: the prose lands in
// the commit body (visible to `aiwf history`).
func TestRetitle_Reason(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "E-0001", "Reasoned Title",
		"--reason", "scope absorbed M-076 work",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("retitle with --reason: %d", rc)
	}

	// History should land — the trailer chain reached git.
	if rc := run([]string{"history", "E-0001", "--root", root}); rc != exitOK {
		t.Errorf("history E-01: %d", rc)
	}
}

// TestRetitle_EmptyTitleRejected pins the empty-title guard.
func TestRetitle_EmptyTitleRejected(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "E-0001", "   ",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("retitle E-01 with whitespace title = %d, want %d", rc, exitUsage)
	}
}

// TestRetitle_SameTitleRejected pins the no-op guard: passing the
// current title produces a clear error so the operator notices the
// typo (no commit lands).
func TestRetitle_SameTitleRejected(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "E-0001", "Foundations",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("retitle E-01 with same title = %d, want %d", rc, exitUsage)
	}
}

// TestRetitle_UnknownIdRejected pins the missing-target guard.
func TestRetitle_UnknownIdRejected(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "E-0099", "Whatever",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("retitle E-99 = %d, want %d (E-99 doesn't exist)", rc, exitUsage)
	}
}

// TestRetitle_TopLevel_BodyH1Sync pins G-0083: when a top-level
// entity's body carries a canonical `# <ID> — <title>` H1, retitle
// rewrites it in the same atomic commit as the frontmatter `title:`
// update — mirroring the AC behavior (`### AC-N — <title>` rewrite).
// Most entities have no H1 (the BodyTemplate scaffold doesn't include
// one), but those that do — historical entities, hand-added headings —
// must not drift from the frontmatter after a retitle.
func TestRetitle_TopLevel_BodyH1Sync(t *testing.T) {
	root := retitleSetup(t)

	// Inject a canonical H1 into the epic body so retitle has something
	// to sync. Use aiwf edit-body --body-file so the change lands as a
	// proper trailered commit (the retitle verb otherwise sees a dirty
	// working tree and the projection diff drifts).
	bodyFile := filepath.Join(t.TempDir(), "body-with-h1.md")
	bodyContent := "# E-0001 — Foundations\n\n## Goal\n\n## Scope\n\n## Out of scope\n"
	if err := os.WriteFile(bodyFile, []byte(bodyContent), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}
	if rc := run([]string{"edit-body", "E-0001", "--body-file", bodyFile, "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("edit-body to inject H1: %d", rc)
	}

	if rc := run([]string{"retitle", "E-0001", "Refocused Foundations", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("retitle: %d", rc)
	}

	body, err := os.ReadFile(filepath.Join(root, "work", "epics", "E-0001-refocused-foundations", "epic.md"))
	if err != nil {
		t.Fatalf("read epic at new slug: %v", err)
	}
	s := string(body)
	if !strings.Contains(s, "# E-0001 — Refocused Foundations\n") {
		t.Errorf("H1 not synced to new title; body:\n%s", s)
	}
	if strings.Contains(s, "# E-0001 — Foundations\n") {
		t.Errorf("stale H1 (old title) still present; body:\n%s", s)
	}
	if !strings.Contains(s, "title: Refocused Foundations") {
		t.Errorf("frontmatter title not updated; body:\n%s", s)
	}
}

// TestRetitle_TopLevel_NoH1_BodyUnchanged pins the silent-no-op shape
// for G-0083: the kernel's BodyTemplate scaffold doesn't produce an H1
// (frontmatter `title:` is the single source of truth), so most
// freshly-added entities have no H1. Retitle must not introduce one —
// frontmatter updates, body stays exactly as it was minus the
// frontmatter block.
func TestRetitle_TopLevel_NoH1_BodyUnchanged(t *testing.T) {
	root := retitleSetup(t)

	if rc := run([]string{"retitle", "E-0001", "Refocused", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("retitle: %d", rc)
	}

	body, err := os.ReadFile(filepath.Join(root, "work", "epics", "E-0001-refocused", "epic.md"))
	if err != nil {
		t.Fatalf("read epic at new slug: %v", err)
	}
	s := string(body)
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "# ") {
			t.Errorf("unexpected H1 in body after retitle on a no-H1 entity: %q", line)
		}
	}
}

// TestRetitle_TopLevel_NonCanonicalH1_LeftAlone pins the conservative
// rewrite shape for G-0083: only the canonical `# <ID> — <title>`
// pattern is touched. Non-canonical variants (colon, hyphen, missing
// id, etc.) are operator-owned hand edits — retitle leaves them as-is
// so an intentional divergence isn't silently clobbered.
func TestRetitle_TopLevel_NonCanonicalH1_LeftAlone(t *testing.T) {
	root := retitleSetup(t)

	// Hand-shaped H1 that doesn't match the `# E-0001 — ` canonical
	// prefix — the rewrite should skip it entirely.
	bodyFile := filepath.Join(t.TempDir(), "body-noncanon.md")
	bodyContent := "# Custom heading the operator wrote\n\n## Goal\n\n## Scope\n\n## Out of scope\n"
	if err := os.WriteFile(bodyFile, []byte(bodyContent), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}
	if rc := run([]string{"edit-body", "E-0001", "--body-file", bodyFile, "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("edit-body to inject non-canonical H1: %d", rc)
	}

	if rc := run([]string{"retitle", "E-0001", "Refocused Foundations", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("retitle: %d", rc)
	}

	body, err := os.ReadFile(filepath.Join(root, "work", "epics", "E-0001-refocused-foundations", "epic.md"))
	if err != nil {
		t.Fatalf("read epic at new slug: %v", err)
	}
	s := string(body)
	if !strings.Contains(s, "# Custom heading the operator wrote\n") {
		t.Errorf("non-canonical H1 was rewritten; body:\n%s", s)
	}
	if !strings.Contains(s, "title: Refocused Foundations") {
		t.Errorf("frontmatter title not updated; body:\n%s", s)
	}
}

// TestRetitle_AC_FrontmatterAndBody pins AC-2: composite-id retitle
// updates BOTH the parent milestone's acs[i].title AND the matching
// `### AC-N — <title>` body heading, atomically in one commit.
func TestRetitle_AC_FrontmatterAndBody(t *testing.T) {
	root := retitleSetup(t)
	if rc := run([]string{"add", "ac", "M-0001", "--title", "original ac title", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}

	rc := run([]string{
		"retitle", "M-0001/AC-1", "refocused ac title",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("retitle M-001/AC-1: %d", rc)
	}

	mPath := filepath.Join(root, "work", "epics", "E-0001-foundations", "M-0001-first-milestone.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	content := string(body)

	if !strings.Contains(content, "title: refocused ac title") {
		t.Errorf("acs[].title not updated:\n%s", content)
	}
	if !strings.Contains(content, "### AC-1 — refocused ac title") {
		t.Errorf("body heading not regenerated:\n%s", content)
	}
	if strings.Contains(content, "### AC-1 — original ac title") {
		t.Errorf("body still carries old AC-1 heading:\n%s", content)
	}
}

// TestRetitle_AC_UnknownRejected pins the missing-AC guard.
func TestRetitle_AC_UnknownRejected(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "M-0001/AC-9", "Whatever",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("retitle M-001/AC-9 = %d, want %d (no AC-9)", rc, exitUsage)
	}
}

// TestRetitle_DispatcherSeam_TopLevel is the seam test for AC-6 on
// the top-level path, per CLAUDE.md "Test the seam, not just the
// layer". Drives the dispatcher end-to-end and asserts both the on-
// disk title change AND the trailered commit landed (history finds it).
func TestRetitle_DispatcherSeam_TopLevel(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "E-0001", "Seam-tested Foundations",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("retitle (seam top-level): %d", rc)
	}

	body, err := os.ReadFile(filepath.Join(root, "work", "epics", "E-0001-seam-tested-foundations", "epic.md"))
	if err != nil {
		t.Fatalf("read epic at new slug: %v", err)
	}
	if !strings.Contains(string(body), "title: Seam-tested Foundations") {
		t.Errorf("epic frontmatter missing new title (seam):\n%s", body)
	}
	if rc := run([]string{"history", "E-0001", "--root", root}); rc != exitOK {
		t.Errorf("aiwf history E-01 (seam): %d", rc)
	}
}

// TestRetitle_SlugSyncedToTitle is the focused G-0108 pin: after
// retitle the on-disk slug matches the slugified new title; both the
// frontmatter title change and the file rename land in the same commit
// (one `aiwf-verb: retitle` trailer, one rename + modify diff). Without
// this behavior the operator has to follow every retitle with a manual
// `aiwf rename` and the two-step workflow leaks back in.
func TestRetitle_SlugSyncedToTitle(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "G-0001", "Sync the slug too",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("retitle G-0001: %d", rc)
	}

	// New path exists.
	newPath := filepath.Join(root, "work", "gaps", "G-0001-sync-the-slug-too.md")
	body, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("expected gap at new slug %s: %v", newPath, err)
	}
	if !strings.Contains(string(body), "title: Sync the slug too") {
		t.Errorf("frontmatter title not synced at new path:\n%s", body)
	}

	// Old path is gone.
	oldPath := filepath.Join(root, "work", "gaps", "G-0001-first-gap.md")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old slug path still exists after retitle: %v", err)
	}

	// History should show one retitle commit covering this id; the
	// rename + frontmatter change land together.
	if rc := run([]string{"history", "G-0001", "--root", root}); rc != exitOK {
		t.Errorf("aiwf history G-0001: %d", rc)
	}
}

// TestRetitle_TitleChangeButSameSlug pins the edge case where the new
// title slugifies to the same slug as the current path — e.g., a
// punctuation-only or capitalization-only tweak. The frontmatter
// updates, but no rename happens (source == dest, OpMove skipped).
func TestRetitle_TitleChangeButSameSlug(t *testing.T) {
	root := retitleSetup(t)

	// "First Gap" slugifies to "first-gap"; "first gap!" slugifies to
	// the same thing. Title change only, no rename.
	rc := run([]string{
		"retitle", "G-0001", "first gap!",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("retitle G-0001 same-slug: %d", rc)
	}

	body, err := os.ReadFile(filepath.Join(root, "work", "gaps", "G-0001-first-gap.md"))
	if err != nil {
		t.Fatalf("read gap: %v", err)
	}
	if !strings.Contains(string(body), "title: first gap!") {
		t.Errorf("frontmatter title not updated when slug stays:\n%s", body)
	}
}

// TestRetitle_EmptySlugRejected pins the punctuation-only-title guard
// (G-0108). A title that slugifies to the empty string would orphan
// the file at a path with no slug body; retitle errors with a clear
// pointer at `aiwf rename` for the operator who genuinely needs that
// shape.
func TestRetitle_EmptySlugRejected(t *testing.T) {
	root := retitleSetup(t)

	rc := run([]string{
		"retitle", "G-0001", "!!!",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("retitle with punctuation-only title = %d, want %d (slug would be empty)", rc, exitUsage)
	}
	// Old path is still there — retitle aborted cleanly.
	if _, err := os.Stat(filepath.Join(root, "work", "gaps", "G-0001-first-gap.md")); err != nil {
		t.Errorf("original gap path lost after rejected retitle: %v", err)
	}
}

// TestRetitle_DispatcherSeam_Composite is the seam test for AC-6 on
// the composite-id path.
func TestRetitle_DispatcherSeam_Composite(t *testing.T) {
	root := retitleSetup(t)
	if rc := run([]string{"add", "ac", "M-0001", "--title", "original", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}

	rc := run([]string{
		"retitle", "M-0001/AC-1", "seam-tested",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("retitle (seam composite): %d", rc)
	}
	if rc := run([]string{"history", "M-0001/AC-1", "--root", root}); rc != exitOK {
		t.Errorf("aiwf history M-001/AC-1 (seam): %d", rc)
	}
}
