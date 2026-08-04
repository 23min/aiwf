package check

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestEntityIDNarrowWidth tests the drift-check rule: a narrow-width
// id anywhere in the active tree (entities outside `<kind>/archive/`)
// is a defect, and fires one error-severity finding per narrow file.
// Whether canonical ids sit alongside it makes no difference.
//
// Both id axes are in scope — the filename and the frontmatter `id:` —
// and an entity narrow on both is still one finding, since the count is
// per entity rather than per axis.
//
// Archive entries never fire: per ADR-0008's "Drift control" subsection
// they are excluded outright, and per ADR-0004's forget-by-default
// principle a narrow archived file stays narrow permanently. The
// archive fixtures below carry `G-001` rather than a one-digit id
// because entity.IDFromPath rejects anything below the gap grammar's
// three-digit floor — a one-digit fixture is dropped before the
// archive test runs, so it would assert nothing about the exclusion.
//
// ADR never fires either, structurally rather than by exemption: the
// ADR grammar floor is canonical width, so IDFromPath admits no narrow
// ADR for the width test to see.
func TestEntityIDNarrowWidth(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		tr          *tree.Tree
		wantCount   int
		wantNarrows []string // entity ids the rule should fire on, in any order
	}{
		{
			name:        "empty active tree is silent",
			tr:          makeTree(),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "uniform narrow active tree fires on every narrow entry",
			tr: makeTree(
				&entity.Entity{ID: "E-22", Kind: entity.KindEpic, Path: "work/epics/E-22-foo/epic.md"},
				&entity.Entity{ID: "M-100", Kind: entity.KindMilestone, Path: "work/epics/E-22-foo/M-100-bar.md"},
			),
			wantCount:   2,
			wantNarrows: []string{"E-22", "M-100"},
		},
		{
			name: "uniform canonical active tree is silent",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-foo/epic.md"},
				&entity.Entity{ID: "M-0083", Kind: entity.KindMilestone, Path: "work/epics/E-0023-foo/M-0083-bar.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "a lone narrow entity fires",
			tr: makeTree(
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-foo.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-100"},
		},
		{
			name: "single canonical entity is silent",
			tr: makeTree(
				&entity.Entity{ID: "G-0100", Kind: entity.KindGap, Path: "work/gaps/G-0100-foo.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "canonical alongside narrow fires on the narrow entry only",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-old.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-100"},
		},
		{
			name: "multiple narrow entries fire once each",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-old.md"},
				&entity.Entity{ID: "D-001", Kind: entity.KindDecision, Path: "work/decisions/D-001-old.md"},
			),
			wantCount:   2,
			wantNarrows: []string{"G-100", "D-001"},
		},
		{
			name: "narrow archive entries never fire alongside a canonical active tree",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "G-001", Kind: entity.KindGap, Path: "work/gaps/archive/G-001-old.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "narrow archive entries never fire; the narrow active entry alongside them still does",
			tr: makeTree(
				&entity.Entity{ID: "E-22", Kind: entity.KindEpic, Path: "work/epics/E-22-foo/epic.md"},
				&entity.Entity{ID: "G-001", Kind: entity.KindGap, Path: "work/gaps/archive/G-001-old.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"E-22"},
		},
		{
			name: "narrow archive alongside a mixed active tree: fires only on the active narrow",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-old.md"},
				&entity.Entity{ID: "G-001", Kind: entity.KindGap, Path: "work/gaps/archive/G-001-archived.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-100"},
		},
		{
			// The file-move case: `git mv` narrows the filename and
			// leaves frontmatter alone. The width tested is the
			// filename's, so this must fire — and must not quote the
			// canonical frontmatter id while calling it narrow.
			name: "a narrow filename fires even when frontmatter is canonical",
			tr: makeTree(
				&entity.Entity{ID: "G-0100", Kind: entity.KindGap, Path: "work/gaps/G-100-diverged.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-0100"},
		},
		{
			// The reverse divergence — a hand-edited frontmatter id
			// under a canonical filename. No other rule sees it:
			// idPathConsistent canonicalizes both sides, so a
			// width-only difference reads as a match to it.
			name: "a narrow frontmatter id fires even when the filename is canonical",
			tr: makeTree(
				&entity.Entity{ID: "G-200", Kind: entity.KindGap, Path: "work/gaps/G-0200-reverse.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-200"},
		},
		{
			// Both axes narrow at the same spelling is the uniform
			// legacy shape, and it stays one finding — one entity is
			// one defect with one fix.
			name: "an entity narrow on both axes fires once, not once per axis",
			tr: makeTree(
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-both.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-100"},
		},
		{
			// Both narrow and disagreeing. idPathConsistent fires too,
			// on the disagreement; this rule still owes exactly one
			// width finding.
			name: "an entity narrow on both axes at different spellings still fires once",
			tr: makeTree(
				&entity.Entity{ID: "G-200", Kind: entity.KindGap, Path: "work/gaps/G-100-diverged-narrow.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-200"},
		},
		{
			// Below the kind's grammar floor the frontmatter id is
			// malformed rather than narrow, and frontmatter-shape names
			// the expected format. Mirrors the filename axis, where
			// IDFromPath rejects a sub-floor path before the width test.
			name: "a frontmatter id below the kind's grammar floor is left to frontmatter-shape",
			tr: makeTree(
				&entity.Entity{ID: "G-1", Kind: entity.KindGap, Path: "work/gaps/G-0200-subfloor-frontmatter.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			// Below the kind's grammar floor, entity.IDFromPath rejects
			// the path outright. The loader still admits the file, so
			// this reaches the rule and falls through — frontmatter-shape
			// is what reports it.
			name: "an id below the kind's grammar floor is left to frontmatter-shape",
			tr: makeTree(
				&entity.Entity{ID: "G-1", Kind: entity.KindGap, Path: "work/gaps/G-1-tiny.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			// The axes are judged independently. A filename IDFromPath
			// rejects must not take a readable frontmatter id down with
			// it: frontmatter-shape sees nothing wrong with `G-200`, so
			// silence here would leave the narrow id unreported by every
			// rule in the tree.
			name: "a narrow frontmatter id fires under a filename below the grammar floor",
			tr: makeTree(
				&entity.Entity{ID: "G-200", Kind: entity.KindGap, Path: "work/gaps/G-1-tiny.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-200"},
		},
		{
			// Both axes unreadable for width: the filename is below the
			// floor and the frontmatter id is too. Nothing to report
			// here; frontmatter-shape names the malformed id.
			name: "a sub-floor filename over a sub-floor frontmatter id fires nothing here",
			tr: makeTree(
				&entity.Entity{ID: "G-2", Kind: entity.KindGap, Path: "work/gaps/G-1-tiny.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			// The archive exclusion covers the frontmatter axis too, not
			// just the filename the other archive fixtures vary.
			name: "a narrow frontmatter id under an archived canonical filename never fires",
			tr: makeTree(
				&entity.Entity{ID: "G-500", Kind: entity.KindGap, Path: "work/gaps/archive/G-0500-narrow-frontmatter.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			// The exclusion is per path *segment*. A slug that merely
			// contains the word is an active entity and still fires.
			name: "a slug containing the word archive is not an archive path",
			tr: makeTree(
				&entity.Entity{ID: "G-0600", Kind: entity.KindGap, Path: "work/gaps/G-600-archive-format-notes.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-0600"},
		},
		{
			name: "a canonical ADR never fires, and the narrow entry beside it still does",
			tr: makeTree(
				&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-foo.md"},
				&entity.Entity{ID: "E-22", Kind: entity.KindEpic, Path: "work/epics/E-22-foo/epic.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"E-22"},
		},
		{
			name: "ADR alone is silent",
			tr: makeTree(
				&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-foo.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "ADR alongside a canonical and a narrow entry fires on the narrow only",
			tr: makeTree(
				&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-foo.md"},
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "M-100", Kind: entity.KindMilestone, Path: "work/epics/E-0023-new/M-100-bar.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"M-100"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := entityIDNarrowWidth(tc.tr)
			if len(got) != tc.wantCount {
				t.Fatalf("entityIDNarrowWidth findings = %d, want %d: %+v",
					len(got), tc.wantCount, got)
			}
			if tc.wantCount == 0 {
				return
			}
			seen := make(map[string]bool, len(got))
			for _, f := range got {
				seen[f.EntityID] = true
				if f.Code != CodeEntityIDNarrowWidth {
					t.Errorf("Code = %q, want entity-id-narrow-width", f.Code)
				}
				if f.Severity != SeverityError {
					t.Errorf("Severity = %q, want error", f.Severity)
				}
				if f.Path == "" {
					t.Errorf("Path must be set on finding for %s", f.EntityID)
				}
			}
			for _, want := range tc.wantNarrows {
				if !seen[want] {
					t.Errorf("expected finding for entity %q, got %+v", want, got)
				}
			}
		})
	}
}

// TestEntityIDNarrowWidth_RemediationNamesNoVerb pins what the finding
// tells an operator to do. No verb widens an id in place, so the
// message must point at undoing the hand-edit or file move that
// produced the narrow id. Naming a verb here would send the reader
// somewhere that cannot help: `aiwf reallocate` assigns a *different*
// number rather than the same one at canonical width.
func TestEntityIDNarrowWidth_RemediationNamesNoVerb(t *testing.T) {
	t.Parallel()
	got := entityIDNarrowWidth(makeTree(
		&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-old.md"},
	))
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(got), got)
	}
	msg := got[0].Message
	for _, forbidden := range []string{"rewidth", "reallocate"} {
		if strings.Contains(msg, forbidden) {
			t.Errorf("remediation names %q, which cannot widen an id in place: %s", forbidden, msg)
		}
	}
	for _, want := range []string{"hand-edit", "move"} {
		if !strings.Contains(msg, want) {
			t.Errorf("remediation does not mention %q — an operator cannot act on it: %s", want, msg)
		}
	}
}

// TestEntityIDNarrowWidth_MessageQuotesOnlyTheNarrowID pins which of an
// entity's two ids the operator is shown when the axes diverge. Quoting
// the canonical side and calling it narrow would be self-contradicting
// text at the one seam an operator reads to find the offending file —
// in either direction, so both are pinned here.
//
// EntityID stays the frontmatter id throughout: it is the
// machine-readable handle every other finding uses to name the entity,
// independent of which axis the message quotes.
//
// wantPhrase is what pins the helper to its call site. narrowIDPhrase's
// own test proves it labels each axis correctly in isolation; only an
// assertion here catches the rule handing it the two ids the wrong way
// round, which would name both axes backwards while quoting each id
// exactly where a quotes-only assertion expects to find it.
func TestEntityIDNarrowWidth_MessageQuotesOnlyTheNarrowID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		e            *entity.Entity
		wantPhrase   string
		wantAbsent   string
		wantEntityID string
	}{
		{
			name:         "narrow filename under canonical frontmatter",
			e:            &entity.Entity{ID: "G-0100", Kind: entity.KindGap, Path: "work/gaps/G-100-diverged.md"},
			wantPhrase:   `filename id "G-100"`,
			wantAbsent:   `"G-0100"`,
			wantEntityID: "G-0100",
		},
		{
			name:         "narrow frontmatter under canonical filename",
			e:            &entity.Entity{ID: "G-200", Kind: entity.KindGap, Path: "work/gaps/G-0200-reverse.md"},
			wantPhrase:   `frontmatter id "G-200"`,
			wantAbsent:   `"G-0200"`,
			wantEntityID: "G-200",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := entityIDNarrowWidth(makeTree(tc.e))
			if len(got) != 1 {
				t.Fatalf("findings = %d, want 1: %+v", len(got), got)
			}
			msg := got[0].Message
			if !strings.Contains(msg, tc.wantPhrase) {
				t.Errorf("message does not name the narrow axis and its id (%s): %s", tc.wantPhrase, msg)
			}
			if strings.Contains(msg, tc.wantAbsent) {
				t.Errorf("message quotes the canonical id %s and calls it narrow: %s", tc.wantAbsent, msg)
			}
			if !strings.Contains(msg, "4 digits") {
				t.Errorf("message does not state the canonical width an operator must restore: %s", msg)
			}
			if got[0].EntityID != tc.wantEntityID {
				t.Errorf("EntityID = %q, want the frontmatter id %q", got[0].EntityID, tc.wantEntityID)
			}
			if got[0].Path != tc.e.Path {
				t.Errorf("Path = %q, want %q — the finding must locate the file its message describes",
					got[0].Path, tc.e.Path)
			}
		})
	}
}

// TestNarrowIDPhrase covers every shape the message can take, one per
// reachable arm. The uniform-narrow case names no axis on purpose:
// both sides are narrow, so naming one would imply the other is clean.
func TestNarrowIDPhrase(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		pathID        string
		frontmatterID string
		want          string
	}{
		{"neither narrow", "G-0100", "G-0100", ""},
		{"filename narrow only", "G-100", "G-0100", `filename id "G-100"`},
		{"frontmatter narrow only", "G-0200", "G-200", `frontmatter id "G-200"`},
		{"both narrow, same spelling", "G-100", "G-100", `id "G-100"`},
		{"both narrow, disagreeing", "G-100", "G-200", `filename id "G-100" and frontmatter id "G-200"`},
		{"frontmatter unreadable for width", "G-100", "", `filename id "G-100"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := narrowIDPhrase(tc.pathID, tc.frontmatterID); got != tc.want {
				t.Errorf("narrowIDPhrase(%q, %q) = %q, want %q",
					tc.pathID, tc.frontmatterID, got, tc.want)
			}
		})
	}
}

// TestFrontmatterWidthID pins the grammar-floor gate on the frontmatter
// axis: width is only a meaningful question about an id the kind admits.
// A sub-floor id is malformed, and frontmatter-shape is the rule that
// says so with the expected format — this returns "" so the width rule
// does not stack a second, less informative finding on top.
func TestFrontmatterWidthID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		e    *entity.Entity
		want string
	}{
		{"canonical gap id", &entity.Entity{ID: "G-0200", Kind: entity.KindGap}, "G-0200"},
		{"narrow but well-formed gap id", &entity.Entity{ID: "G-200", Kind: entity.KindGap}, "G-200"},
		{"gap id below the three-digit floor", &entity.Entity{ID: "G-1", Kind: entity.KindGap}, ""},
		{"narrow but well-formed epic id", &entity.Entity{ID: "E-22", Kind: entity.KindEpic}, "E-22"},
		{"epic id below the two-digit floor", &entity.Entity{ID: "E-2", Kind: entity.KindEpic}, ""},
		{"ADR id below canonical width", &entity.Entity{ID: "ADR-001", Kind: entity.KindADR}, ""},
		{"missing id", &entity.Entity{ID: "", Kind: entity.KindGap}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := frontmatterWidthID(tc.e); got != tc.want {
				t.Errorf("frontmatterWidthID(id %q, kind %s) = %q, want %q",
					tc.e.ID, tc.e.Kind, got, tc.want)
			}
		})
	}
}

// TestIsNarrowID_DefensiveBranches exercises the defensive
// fall-through branches in isNarrowID directly. The parent rule
// (entityIDNarrowWidth) only feeds path-validated ids in production,
// so these inputs are unreachable through the rule's table-driven
// fixtures. Kept as a separate unit test so a future change to the
// helper's contract still catches the malformed-input cases.
func TestIsNarrowID_DefensiveBranches(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		id   string
		want bool
	}{
		{"prefix only, no digits", "E-", false},
		{"non-digit in tail", "E-12a", false},
		{"unknown prefix", "X-12", false},
		{"empty string", "", false},
		{"narrow E", "E-1", true},
		{"narrow M", "M-99", true},
		{"canonical E", "E-0001", false},
		{"natural-width above pad", "M-12345", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isNarrowID(tc.id); got != tc.want {
				t.Errorf("isNarrowID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}

// TestEntityIDNarrowWidth_FiresThroughCheckRun is the seam test. Every
// other assertion here calls entityIDNarrowWidth directly, which leaves
// the rule's *registration* in check.Run unpinned: delete that line and
// the unit tests all still pass while the shipped `aiwf check` — the
// binary the pre-push hook and CI run — stops reporting narrow ids
// entirely.
//
// That gap matters most for exactly this rule, whose whole point is
// being an unconditional error-severity gate rather than an advisory
// warning. So this drives the real path: files on disk, through
// tree.Load, through check.Run.
func TestEntityIDNarrowWidth_FiresThroughCheckRun(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	mustWrite(t, root, "work/gaps/G-093-legacy.md", `---
id: G-093
title: Legacy-width gap
status: open
---

## What's missing

Prose.
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	findings := Run(tr, loadErrs)
	var got *Finding
	for i := range findings {
		if findings[i].Code == CodeEntityIDNarrowWidth {
			got = &findings[i]
			break
		}
	}
	if got == nil {
		t.Fatal("check.Run reported no entity-id-narrow-width finding for an active narrow id — " +
			"the rule is not registered in Run, so `aiwf check` does not gate narrow ids at all")
	}
	if got.Severity != SeverityError {
		t.Errorf("Severity = %q through check.Run, want error — the gate is advisory, not blocking", got.Severity)
	}
	if got.EntityID != "G-093" {
		t.Errorf("EntityID = %q, want %q", got.EntityID, "G-093")
	}
}

// TestEntityIDNarrowWidth_FrontmatterAxisFiresThroughCheckRun is the
// seam test for the frontmatter axis, and it drives real files because
// that axis is the one nothing else covers: a canonical filename over a
// narrow `id:` clears frontmatter-shape (the id is within the kind's
// grammar) and clears idPathConsistent (which canonicalizes both sides
// before comparing), so if this rule does not fire, the whole tree
// validates clean with a narrow id in it.
func TestEntityIDNarrowWidth_FrontmatterAxisFiresThroughCheckRun(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	mustWrite(t, root, "work/gaps/G-0200-reverse.md", `---
id: G-200
title: Narrow frontmatter under a canonical filename
status: open
---

## What's missing

Prose.
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	findings := Run(tr, loadErrs)
	// Exactly one finding, not merely one of this code: the rationale
	// above is that no other rule sees this shape, and asserting the
	// total is what keeps that claim honest if one later does.
	if len(findings) != 1 {
		t.Fatalf("check.Run findings = %d, want exactly 1: %+v", len(findings), findings)
	}
	got := findings[0]
	if got.Code != CodeEntityIDNarrowWidth {
		t.Fatalf("Code = %q, want entity-id-narrow-width — a narrow frontmatter id is unreported: %+v",
			got.Code, got)
	}
	if got.Severity != SeverityError {
		t.Errorf("Severity = %q through check.Run, want error — the gate is advisory, not blocking", got.Severity)
	}
	if !strings.Contains(got.Message, `frontmatter id "G-200"`) {
		t.Errorf("message does not name the narrow frontmatter id: %s", got.Message)
	}
}
