package check

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/trunk"
)

// TestBodyProseID_Matrix walks the rule's classification space:
// malformed-shape, unresolved bare, unresolved composite parent,
// unresolved composite AC, and the silent positive controls.
// Per G-0184 — pins the id-shape chokepoint at the committed body
// prose layer.
func TestBodyProseID_Matrix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		body        string
		wantSubcode string
		wantToken   string
		silent      bool
	}{
		// --- malformed-shape ---
		{
			name:        "single-lowercase-letter (triggering instance)",
			body:        "We depend on the milestone M-a.",
			wantSubcode: "malformed-shape",
			wantToken:   "M-a",
		},
		{
			name:        "lowercase-word suffix",
			body:        "The plan starts with the milestone M-alpha.",
			wantSubcode: "malformed-shape",
			wantToken:   "M-alpha",
		},
		{
			name:        "uppercase placeholder M-NNNN",
			body:        "Once allocated, promote to addressed --by M-NNNN.",
			wantSubcode: "malformed-shape",
			wantToken:   "M-NNNN",
		},
		{
			name:        "narrow-numeric milestone M-1 (conversational leak)",
			body:        "The first milestone is M-1.",
			wantSubcode: "malformed-shape",
			wantToken:   "M-1",
		},
		{
			name:        "narrow-numeric epic E-1",
			body:        "Scope leak through E-1's depends_on chain.",
			wantSubcode: "malformed-shape",
			wantToken:   "E-1",
		},
		{
			name:        "compound English word ADR-shaped",
			body:        "This is an ADR-shaped concern.",
			wantSubcode: "malformed-shape",
			wantToken:   "ADR-shaped",
		},

		// --- unresolved bare ---
		{
			name:        "unresolved well-formed milestone",
			body:        "See M-9999 for the proposed rule.",
			wantSubcode: "unresolved",
			wantToken:   "M-9999",
		},
		{
			name:        "unresolved well-formed ADR (4-digit canonical)",
			body:        "Per ADR-9999, the decision stands.",
			wantSubcode: "unresolved",
			wantToken:   "ADR-9999",
		},

		// --- unresolved composite ---
		{
			name:        "unresolved composite milestone",
			body:        "Cross-reference to M-9999/AC-1.",
			wantSubcode: "unresolved-milestone",
			wantToken:   "M-9999/AC-1",
		},
		{
			name:        "composite parent present, AC missing",
			body:        "Per M-0001/AC-9, the gap is closed.",
			wantSubcode: "unresolved-ac",
			wantToken:   "M-0001/AC-9",
		},

		// --- silent positive controls ---
		{
			name:   "well-formed resolved",
			body:   "Per M-0001, the rule applies.",
			silent: true,
		},
		{
			name:   "composite resolved",
			body:   "Per M-0001/AC-1, the AC holds.",
			silent: true,
		},
		{
			name:   "malformed inside inline code span",
			body:   "Discussion of `M-a` and `M-NNNN` shapes is fine in code spans.",
			silent: true,
		},
		{
			name:   "malformed inside fenced code block",
			body:   "Example:\n```\nM-a\nM-NNNN\n```\nDone.",
			silent: true,
		},
		{
			name:   "malformed inside tilde fenced code block",
			body:   "Example:\n~~~\nM-a\n~~~\nDone.",
			silent: true,
		},
		{
			name:   "unresolved well-formed inside backticks",
			body:   "Hypothetical id `M-9999` is OK in backticks.",
			silent: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			ents := writeBodyProseFixture(t, root, tc.body)
			tr := &tree.Tree{Root: root, Entities: ents}

			got := bodyProseID(tr)
			if tc.silent {
				if len(got) != 0 {
					t.Fatalf("expected silent, got %d findings: %+v", len(got), got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("findings = %d, want 1: %+v", len(got), got)
			}
			f := got[0]
			if f.Code != CodeBodyProseID {
				t.Errorf("Code = %q, want %q", f.Code, CodeBodyProseID)
			}
			if f.Severity != SeverityError {
				t.Errorf("Severity = %v, want error", f.Severity)
			}
			if f.Subcode != tc.wantSubcode {
				t.Errorf("Subcode = %q, want %q", f.Subcode, tc.wantSubcode)
			}
			if !strings.Contains(f.Message, tc.wantToken) {
				t.Errorf("Message %q should contain token %q", f.Message, tc.wantToken)
			}
			if f.Path == "" {
				t.Errorf("Path empty; finding must name the file path")
			}
		})
	}
}

// TestBodyProseID_EdgeCases pins the rule's contract at its edges: a
// suffix starting outside ASCII fires like an ASCII one, while a real id
// abutting non-ASCII text stays a resolvable citation; HTML tags are
// scanned as prose; an empty body is silent; prefix-suffix concatenated
// tokens (M-0001prefix) deliberately fire malformed-shape, as does the
// narrow numeric leak (M-1).
// The cases here document intent so future "simplification" attempts
// surface as test failures rather than silent behavior shifts.
func TestBodyProseID_EdgeCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		body        string
		wantSubcode string
		wantToken   string // quoted in the message; a truncated token is a wrong locator
		silent      bool
	}{
		{
			name:   "empty body — no tokens to scan",
			body:   "",
			silent: true,
		},
		{
			name:        "Unicode suffix — Greek M-α fires malformed-shape",
			body:        "References M-α which is not an aiwf id shape.",
			wantSubcode: "malformed-shape",
			wantToken:   "M-α",
		},
		{
			name:        "Unicode suffix — Cyrillic M-АБВ fires malformed-shape",
			body:        "References M-АБВ.",
			wantSubcode: "malformed-shape",
			wantToken:   "M-АБВ",
		},
		{
			// Languages that set no space between a citation and the
			// following word: without a boundary the token would absorb
			// the sentence and a resolvable id would read as malformed.
			name:   "real id abutting non-ASCII text stays a resolvable citation",
			body:   "詳細は M-0001の仕様を参照してください。",
			silent: true,
		},
		{
			name:   "composite id abutting non-ASCII text stays resolvable",
			body:   "M-0001/AC-1の記述を参照。",
			silent: true,
		},
		{
			name:        "HTML tag wrapping a malformed token still fires",
			body:        "<p>The token M-foo is malformed.</p>",
			wantSubcode: "malformed-shape",
		},
		{
			name:        "prefix-suffix concatenation M-0001prefix fires malformed-shape",
			body:        "Reference M-0001prefix is a suspicious concatenation.",
			wantSubcode: "malformed-shape",
		},
		{
			name:        "narrow numeric leak M-1 (conversational label) fires",
			body:        "Casual conversational label M-1 from chat leaked here.",
			wantSubcode: "malformed-shape",
		},
		{
			name:   "token at start of body still picked up",
			body:   "M-a is the first thing in this body.",
			silent: false, // expected to fire malformed-shape
		},
		{
			name:   "token at end of body (no trailing newline) still picked up",
			body:   "The body ends with the malformed token M-a",
			silent: false,
		},
		{
			name:   "id-shape inside backticks at line start is silent",
			body:   "`M-a` at line start should not fire.",
			silent: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			ents := writeBodyProseFixture(t, root, tc.body)
			tr := &tree.Tree{Root: root, Entities: ents}
			got := bodyProseID(tr)

			if tc.silent {
				if len(got) != 0 {
					t.Fatalf("expected silent, got %d findings: %+v", len(got), got)
				}
				return
			}
			if len(got) == 0 {
				t.Fatalf("expected at least one finding, got none")
			}
			if tc.wantSubcode != "" && got[0].Subcode != tc.wantSubcode {
				t.Errorf("Subcode = %q, want %q", got[0].Subcode, tc.wantSubcode)
			}
			if tc.wantToken != "" {
				if len(got) != 1 {
					t.Errorf("got %d findings, want 1: %+v", len(got), got)
				}
				if !strings.Contains(got[0].Message, strconv.Quote(tc.wantToken)) {
					t.Errorf("Message = %q, want it to quote token %q", got[0].Message, tc.wantToken)
				}
			}
		})
	}
}

// TestBodyProseID_ResolvesLineFromTokenOffset pins the Line-resolution
// contract: each finding's Line is the 1-based line within the body
// where the matched token starts (not the body's start-of-file, not
// hardcoded 1). ScanBodyProseID returns body-relative Line; bodyProseID
// adjusts to file-relative by adding the body's start-of-file line
// offset (counts newlines in the pre-body bytes).
func TestBodyProseID_ResolvesLineFromTokenOffset(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	body := "Filler line one.\nFiller line two.\nM-a fires here on line three.\n"
	ents := writeBodyProseFixture(t, root, body)
	tr := &tree.Tree{Root: root, Entities: ents}

	got := bodyProseID(tr)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(got), got)
	}
	// The token M-a is at body-relative line 3 (two filler lines, then
	// the token line); bodyProseID adjusts by adding the file-relative
	// offset to the body start, so the final Line is well above 1 and
	// at least 3. Asserting > 2 pins both the body-offset arithmetic
	// in ScanBodyProseID and bodyProseID's frontmatter adjustment.
	if got[0].Line <= 2 {
		t.Errorf("Line = %d, want > 2 (token-offset resolution + body-start adjustment)", got[0].Line)
	}
}

// TestBodyProseID_DedupePerEntityToken pins the dedupe contract:
// repeated mentions of the same bad token in one entity body produce
// one finding, not one per occurrence.
func TestBodyProseID_DedupePerEntityToken(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	body := "M-a appears here. And M-a appears again. And once more: M-a."
	ents := writeBodyProseFixture(t, root, body)
	tr := &tree.Tree{Root: root, Entities: ents}

	got := bodyProseID(tr)
	if len(got) != 1 {
		t.Fatalf("dedupe broken: got %d findings, want 1: %+v", len(got), got)
	}
}

// TestBodyProseID_ArchivedEntitySkipped pins the archive-scoping
// contract per ADR-0004 §"Check shape rules". An archived entity's
// body is not scanned even if it contains malformed tokens.
func TestBodyProseID_ArchivedEntitySkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	body := "Stale reference to M-a in an archived gap."
	path := "work/gaps/archive/G-0001-archived.md"
	abs := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	fm := "---\nid: G-0001\ntitle: Old\nstatus: addressed\n---\n\n## What's missing\n\n" + body + "\n## Why it matters\n\nDoes not matter.\n"
	if err := os.WriteFile(abs, []byte(fm), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{{
		ID: "G-0001", Kind: entity.KindGap, Title: "Old", Status: "addressed", Path: path,
	}}}

	got := bodyProseID(tr)
	if len(got) != 0 {
		t.Fatalf("archived entity should be skipped, got %d findings: %+v", len(got), got)
	}
}

// TestBodyProseID_MultipleEntitiesEachReportSeparately pins per-entity
// scoping: two entities each containing the same malformed token
// produce two findings (one per entity), not a single deduped finding.
func TestBodyProseID_MultipleEntitiesEachReportSeparately(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ents := writeTwoGapsBodyProseFixture(t, root, "M-a appears.")
	tr := &tree.Tree{Root: root, Entities: ents}

	got := bodyProseID(tr)
	if len(got) != 2 {
		t.Fatalf("per-entity finding broken: got %d, want 2: %+v", len(got), got)
	}
}

// TestBodyProseID_CommonMarkShapes_G0240 pins the CommonMark-aware
// masking contract (G-0240): the scanner sees only what CommonMark
// renders as prose. Each case names one of the shapes the regex-based
// masker got wrong — multi-backtick spans, indented code blocks, link
// URLs, unclosed-span chew-through — plus the deliberate behavior
// pins that came with the parser swap: unclosed fences run to EOF
// (CommonMark semantics), link LABELS stay scanned (they're prose),
// and bare URLs in prose stay scanned (no Linkify extension).
func TestBodyProseID_CommonMarkShapes_G0240(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		body        string
		wantSubcode string
		silent      bool
	}{
		{
			name:   "double-backtick code span silent",
			body:   "The shape ``M-a`` is under discussion.",
			silent: true,
		},
		{
			name:   "double-backtick span embedding a backticked id silent",
			body:   "The shape `` `M-a` `` embeds backticks in a span.",
			silent: true,
		},
		{
			name:   "indented code block silent",
			body:   "Example follows:\n\n    M-a inside indented code\n\nDone.",
			silent: true,
		},
		{
			name:   "link destination silent",
			body:   "[the old gap](work/gaps/G-9999-old.md) was deleted.",
			silent: true,
		},
		{
			name:        "link label is prose and still scanned",
			body:        "[see M-a for details](https://example.com/page) anchors prose.",
			wantSubcode: "malformed-shape",
		},
		{
			name:   "link title silent",
			body:   "[label](https://example.com \"about G-9999\") has a title.",
			silent: true,
		},
		{
			name:   "reference-link definition silent",
			body:   "See [the gap][ref].\n\n[ref]: work/gaps/G-9999-old.md",
			silent: true,
		},
		{
			name:   "autolink silent",
			body:   "<https://example.com/G-9999.md> is an autolink URL.",
			silent: true,
		},
		{
			name:        "unclosed backtick does not chew through following prose",
			body:        "An unclosed ` tick ends this line.\nNext line M-a must still fire.",
			wantSubcode: "malformed-shape",
		},
		{
			name:   "unclosed fence runs to EOF per CommonMark — content silent",
			body:   "Prose before.\n\n```\nM-a inside a fence nobody closed\n",
			silent: true,
		},
		{
			name:        "bare URL in prose still scanned (no Linkify, deliberate)",
			body:        "See https://example.com/G-9999.md pasted bare into prose.",
			wantSubcode: "unresolved",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			ents := writeBodyProseFixture(t, root, tc.body)
			tr := &tree.Tree{Root: root, Entities: ents}

			got := bodyProseID(tr)
			if tc.silent {
				if len(got) != 0 {
					t.Fatalf("expected silent, got %d findings: %+v", len(got), got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("findings = %d, want 1: %+v", len(got), got)
			}
			if got[0].Subcode != tc.wantSubcode {
				t.Errorf("Subcode = %q, want %q", got[0].Subcode, tc.wantSubcode)
			}
		})
	}
}

// TestBodyProseID_TrunkTier_G0241 pins the second-tier trunk
// resolution (G-0241): a strict-form token that misses the working-
// tree index but appears in Tree.TrunkIDs carries no resolution defect
// — the id IS allocated, just not visible on this branch. The negative
// cases pin that the trunk tier does not widen anything else:
// truly-unknown ids still fire with a populated trunk set, malformed
// shapes are never laundered by trunk membership, and a locally-visible
// parent stays authoritative for AC validation even when its id also
// appears on trunk. Resolving on the trunk tier does not settle the
// token's spelling either, so a narrow token reaching a canonical trunk
// id still fires narrow-width. All pre-existing tests in this file run
// with TrunkIDs nil and pin the degraded (primary-tier-only) behavior.
func TestBodyProseID_TrunkTier_G0241(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		body        string
		trunkIDs    []string
		wantSubcode string
		silent      bool
	}{
		{
			name:     "bare trunk-only id silent",
			body:     "Depends on G-0500 which was filed on trunk.",
			trunkIDs: []string{"G-0500"},
			silent:   true,
		},
		{
			name:     "composite with trunk-only parent silent (AC position unverifiable without the file)",
			body:     "Per M-0500/AC-1, the contract holds.",
			trunkIDs: []string{"M-0500"},
			silent:   true,
		},
		{
			name:     "narrow-legacy trunk id resolves canonical-width token",
			body:     "Depends on G-0500 from a pre-rewidth trunk.",
			trunkIDs: []string{"G-500"},
			silent:   true,
		},
		{
			name:        "narrow token against canonical-width trunk id fires narrow-width",
			body:        "Depends on G-500 (narrow legacy form).",
			trunkIDs:    []string{"G-0500"},
			wantSubcode: "narrow-width",
		},
		{
			// Trunk ids enter the as-written set too, so a trunk-only
			// entity that IS stored narrow makes a narrow citation of it
			// correct — the same tolerance a working-tree entity gets.
			name:     "narrow token against a trunk id stored narrow",
			body:     "Depends on G-500 (narrow on trunk too).",
			trunkIDs: []string{"G-500"},
			silent:   true,
		},
		{
			name:        "truly-unknown id still fires with populated trunk set",
			body:        "See M-9999 for the proposed rule.",
			trunkIDs:    []string{"G-0500"},
			wantSubcode: "unresolved",
		},
		{
			name:        "truly-unknown composite parent still fires with populated trunk set",
			body:        "See M-9999/AC-1.",
			trunkIDs:    []string{"G-0500"},
			wantSubcode: "unresolved-milestone",
		},
		{
			name:        "malformed shape never laundered by trunk membership",
			body:        "We depend on the milestone M-a.",
			trunkIDs:    []string{"G-0500"},
			wantSubcode: "malformed-shape",
		},
		{
			name:        "local parent stays authoritative for AC validation despite trunk membership",
			body:        "Per M-0001/AC-9, the gap is closed.",
			trunkIDs:    []string{"M-0001"},
			wantSubcode: "unresolved-ac",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			ents := writeBodyProseFixture(t, root, tc.body)
			tr := &tree.Tree{Root: root, Entities: ents}
			for _, id := range tc.trunkIDs {
				tr.TrunkIDs = append(tr.TrunkIDs, trunk.ID{ID: id})
			}

			got := bodyProseID(tr)
			if tc.silent {
				if len(got) != 0 {
					t.Fatalf("expected silent, got %d findings: %+v", len(got), got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("findings = %d, want 1: %+v", len(got), got)
			}
			if got[0].Subcode != tc.wantSubcode {
				t.Errorf("Subcode = %q, want %q", got[0].Subcode, tc.wantSubcode)
			}
		})
	}
}

// TestBodyProseID_CrossBranchPendingTier_M0259AC2 mirrors the
// TrunkTier (G-0241) test's shape but for the cross-branch view
// (ADR-0030, M-0259/AC-2): unlike the silent Trunk tier, a
// cross-branch hit is a VISIBLE finding — trunk is authoritative, a
// sibling branch is provisional. Whether it blocks is decided by the
// ref carrying it (ADR-0041), which is why `ref` is per-case: the same
// prose and the same id classify differently from a pushed branch than
// from an unpushed one.
func TestBodyProseID_CrossBranchPendingTier_M0259AC2(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name            string
		body            string
		crossBranchIDs  []string
		ref             string
		wantSubcode     string
		wantSeverity    Severity
		wantFindingsLen int
	}{
		{
			name:            "published cross-branch-only id fires visibly, non-blocking",
			body:            "Depends on G-0500 which was filed on a sibling branch.",
			crossBranchIDs:  []string{"G-0500"},
			ref:             "refs/remotes/origin/sibling",
			wantSubcode:     "cross-branch-pending",
			wantSeverity:    SeverityWarning,
			wantFindingsLen: 1,
		},
		{
			name:            "same id on an unpushed branch blocks instead",
			body:            "Depends on G-0500 which was filed on a sibling branch.",
			crossBranchIDs:  []string{"G-0500"},
			ref:             "refs/heads/sibling",
			wantSubcode:     "cross-branch-local-only",
			wantSeverity:    SeverityError,
			wantFindingsLen: 1,
		},
		{
			name:            "truly-unknown id still hard-fails despite a populated cross-branch set",
			body:            "See M-9999 for the proposed rule.",
			crossBranchIDs:  []string{"G-0500"},
			ref:             "refs/remotes/origin/sibling",
			wantSubcode:     "unresolved",
			wantSeverity:    SeverityError,
			wantFindingsLen: 1,
		},
		{
			name:            "narrow-legacy cross-branch id resolves canonical-width token",
			body:            "Depends on G-0500 from a pre-rewidth sibling branch.",
			crossBranchIDs:  []string{"G-500"},
			ref:             "refs/remotes/origin/sibling",
			wantSubcode:     "cross-branch-pending",
			wantSeverity:    SeverityWarning,
			wantFindingsLen: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			ents := writeBodyProseFixture(t, root, tc.body)
			// A repository with a remote: publication is expressible, so
			// an id on local refs alone classifies local-only.
			tr := &tree.Tree{Root: root, Entities: ents, HasRemoteTrackingRefs: true}
			for _, id := range tc.crossBranchIDs {
				tr.CrossBranchHits = append(tr.CrossBranchHits, trunk.RefHit{
					Kind: entity.KindGap, ID: id, Path: "work/gaps/" + id + "-x.md", Ref: tc.ref,
				})
			}

			got := bodyProseID(tr)
			if len(got) != tc.wantFindingsLen {
				t.Fatalf("findings = %d, want %d: %+v", len(got), tc.wantFindingsLen, got)
			}
			if got[0].Subcode != tc.wantSubcode {
				t.Errorf("Subcode = %q, want %q", got[0].Subcode, tc.wantSubcode)
			}
			if got[0].Severity != tc.wantSeverity {
				t.Errorf("Severity = %q, want %q", got[0].Severity, tc.wantSeverity)
			}
		})
	}
}

// TestBodyProseID_CrossBranchCollision_M0259AC3 pins the escalation
// pair: a bare id known on more than one ref with divergent content
// fires cross-branch-collision (a distinct, visible, non-blocking
// subcode — see the D-0036 note on the refsResolve twin test), not
// cross-branch-pending.
func TestBodyProseID_CrossBranchCollision_M0259AC3(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ents := writeBodyProseFixture(t, root, "Depends on G-0500 which diverges across two sibling branches.")
	tr := &tree.Tree{Root: root, Entities: ents}
	// Both refs are remote-tracking, so the id is published and the
	// classification reaches the collision arm rather than blocking as
	// local-only first (ADR-0041).
	tr.CrossBranchHits = []trunk.RefHit{
		{Kind: entity.KindGap, ID: "G-0500", Path: "work/gaps/G-0500-x.md", Ref: "refs/remotes/origin/sibling"},
		{Kind: entity.KindGap, ID: "G-0500", Path: "work/gaps/G-0500-x.md", Ref: "refs/remotes/origin/other"},
	}
	tr.CrossBranchCollisions = map[string]bool{"G-0500": true}

	got := bodyProseID(tr)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(got), got)
	}
	if got[0].Subcode != "cross-branch-collision" {
		t.Errorf("Subcode = %q, want cross-branch-collision", got[0].Subcode)
	}
	if got[0].Severity != SeverityWarning {
		t.Errorf("Severity = %q, want warning (non-blocking — see D-0036)", got[0].Severity)
	}
}

// TestBodyProseID_CrossBranchLocalOnlyOutranksCollision is the prose
// mirror of the refsResolve twin: content diverging across two unpushed
// branches is still a reference no other machine can follow, so the
// blocking classification is reached before the collision one. Without
// the ordering this reports a warning and the tree pushes.
func TestBodyProseID_CrossBranchLocalOnlyOutranksCollision(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ents := writeBodyProseFixture(t, root, "Depends on G-0500 which diverges across two sibling branches.")
	tr := &tree.Tree{Root: root, Entities: ents}
	tr.CrossBranchHits = []trunk.RefHit{
		{Kind: entity.KindGap, ID: "G-0500", Path: "work/gaps/G-0500-x.md", Ref: "refs/heads/sibling"},
		{Kind: entity.KindGap, ID: "G-0500", Path: "work/gaps/G-0500-x.md", Ref: "refs/heads/other"},
	}
	tr.CrossBranchCollisions = map[string]bool{"G-0500": true}
	tr.HasRemoteTrackingRefs = true

	got := bodyProseID(tr)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(got), got)
	}
	if got[0].Subcode != "cross-branch-local-only" {
		t.Errorf("Subcode = %q, want cross-branch-local-only — neither ref is published", got[0].Subcode)
	}
	if got[0].Severity != SeverityError {
		t.Errorf("Severity = %q, want error", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "refs/heads/sibling") {
		t.Errorf("Message = %q, want it to name the unpublished refs", got[0].Message)
	}
}

// TestBodyProseID_CrossBranchLocalOnlyStaysPendingWithoutARemote is the
// prose mirror: with nowhere to push, the blocking classification names
// no remedy, so the reference stays the ordinary pending warning.
func TestBodyProseID_CrossBranchLocalOnlyStaysPendingWithoutARemote(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ents := writeBodyProseFixture(t, root, "Depends on G-0500, filed on a branch in a repo with no remote.")
	tr := &tree.Tree{Root: root, Entities: ents, HasRemoteTrackingRefs: false}
	tr.CrossBranchHits = []trunk.RefHit{
		{Kind: entity.KindGap, ID: "G-0500", Path: "work/gaps/G-0500-x.md", Ref: "refs/heads/sibling"},
	}

	got := bodyProseID(tr)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(got), got)
	}
	if got[0].Subcode != "cross-branch-pending" {
		t.Errorf("Subcode = %q, want cross-branch-pending — nothing here can be published", got[0].Subcode)
	}
	if got[0].Severity != SeverityWarning {
		t.Errorf("Severity = %q, want warning", got[0].Severity)
	}
}

// TestBodyProseID_CrossBranchMixedRefsClassifyAsPublished pins that one
// remote-tracking ref is enough: the classification reads the MOST
// visible ref carrying the id, not every ref.
func TestBodyProseID_CrossBranchMixedRefsClassifyAsPublished(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ents := writeBodyProseFixture(t, root, "Depends on G-0500 filed on a branch that has since been pushed.")
	tr := &tree.Tree{Root: root, Entities: ents}
	tr.CrossBranchHits = []trunk.RefHit{
		{Kind: entity.KindGap, ID: "G-0500", Path: "work/gaps/G-0500-x.md", Ref: "refs/heads/sibling"},
		{Kind: entity.KindGap, ID: "G-0500", Path: "work/gaps/G-0500-x.md", Ref: "refs/remotes/origin/sibling"},
	}

	got := bodyProseID(tr)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(got), got)
	}
	if got[0].Subcode != "cross-branch-pending" {
		t.Errorf("Subcode = %q, want cross-branch-pending — the remote-tracking hit publishes the id", got[0].Subcode)
	}
	if got[0].Severity != SeverityWarning {
		t.Errorf("Severity = %q, want warning", got[0].Severity)
	}
}

// TestBodyProseID_UnresolvedWhenAbsentFromEveryTier_M0259AC5 — M-0259/
// AC-5: the guard against the new cross-branch tier ever softening a
// genuinely fabricated or deleted id. All three resolution tiers
// (ByID, Trunk, CrossBranch) are explicitly populated with OTHER
// ids — proving the fabricated token really isn't found anywhere, not
// that a tier was never consulted — and it still hard-fails
// unresolved.
func TestBodyProseID_UnresolvedWhenAbsentFromEveryTier_M0259AC5(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ents := writeBodyProseFixture(t, root, "See M-9999 for the proposed rule.")
	tr := &tree.Tree{Root: root, Entities: ents}
	tr.TrunkIDs = []trunk.ID{{Kind: entity.KindGap, ID: "G-0500", Path: "work/gaps/G-0500-x.md"}}
	tr.CrossBranchHits = []trunk.RefHit{
		{Kind: entity.KindGap, ID: "G-0600", Path: "work/gaps/G-0600-x.md", Ref: "refs/heads/sibling"},
	}

	got := bodyProseID(tr)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(got), got)
	}
	if got[0].Subcode != "unresolved" {
		t.Errorf("Subcode = %q, want unresolved — M-9999 exists nowhere, not locally, not on trunk, not cross-branch", got[0].Subcode)
	}
	if got[0].Severity != SeverityError {
		t.Errorf("Severity = %q, want error (blocking)", got[0].Severity)
	}
}

// TestBodyProseID_LocalTreeStaysAuthoritativeOverCrossBranch pins that
// a locally-resolvable id is never softened by a same-id cross-branch
// hit — the working-tree index is checked first, same ordering as the
// existing Trunk tier.
func TestBodyProseID_LocalTreeStaysAuthoritativeOverCrossBranch(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ents := writeBodyProseFixture(t, root, "Per M-0001/AC-9, the gap is closed.")
	tr := &tree.Tree{Root: root, Entities: ents}
	tr.CrossBranchHits = []trunk.RefHit{
		{Kind: entity.KindMilestone, ID: "M-0001", Path: "elsewhere.md", Ref: "refs/heads/sibling"},
	}

	got := bodyProseID(tr)
	if len(got) != 1 || got[0].Subcode != "unresolved-ac" {
		t.Errorf("got %+v, want the local unresolved-ac finding unaffected by the cross-branch hit", got)
	}
}

// TestBodyProseID_LocallyPresentIDWithCollisionResolvesLocally pins the
// behavior-preservation basis for the lazy cross-branch scan (E-0067/
// M-0265/AC-3): even with a collision recorded for an id present in the
// local tree, bodyProseID resolves the prose token against the local
// index first and never reads the collision — so no cross-branch
// finding surfaces. This is why the lazy filter declining to compute a
// locally-present id's collision changes no finding.
func TestBodyProseID_LocallyPresentIDWithCollisionResolvesLocally(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// The prose references G-0002, the fixture's own local gap (present
	// in the ByID index).
	ents := writeBodyProseFixture(t, root, "See G-0002 for the fixture gap.")
	tr := &tree.Tree{Root: root, Entities: ents}
	tr.CrossBranchHits = []trunk.RefHit{
		{Kind: entity.KindGap, ID: "G-0002", Path: "work/gaps/G-0002-fixture.md", Ref: "refs/heads/sibling"},
		{Kind: entity.KindGap, ID: "G-0002", Path: "work/gaps/G-0002-fixture.md", Ref: "refs/heads/other"},
	}
	tr.CrossBranchCollisions = map[string]bool{"G-0002": true}

	for _, f := range bodyProseID(tr) {
		// Any subcode from the cross-branch tier, so a classification
		// added to that tier is caught rather than passing unnoticed.
		if strings.HasPrefix(f.Subcode, "cross-branch-") {
			t.Errorf("got %+v, want no cross-branch finding — G-0002 resolves locally, its collision must never surface", f)
		}
	}
}

// writeBodyProseFixture lays down a gap G-0001 with the supplied body
// prose under `## What's missing`, plus a milestone M-0001 with AC-1
// to back the composite-resolution positive controls. Both are loaded
// into the returned slice so the bodyProseID rule's id index sees them.
func writeBodyProseFixture(t *testing.T, root, prose string) []*entity.Entity {
	t.Helper()
	gapPath := "work/gaps/G-0002-fixture.md"
	gapBody := "---\nid: G-0002\ntitle: Fixture\nstatus: open\n---\n\n## What's missing\n\n" +
		prose + "\n\n## Why it matters\n\nIt matters.\n"
	mustWriteFile(t, root, gapPath, gapBody)

	mPath := "work/epics/E-0001-foo/M-0001-bar.md"
	mBody := `---
id: M-0001
title: Bar
status: in_progress
parent: E-0001
tdd: none
acs:
    - id: AC-1
      title: First AC
      status: open
---

## Goal

Goal prose.

## Acceptance criteria

Each AC pins one observable behavior.

### AC-1 — First AC

Body prose for AC-1.
`
	mustWriteFile(t, root, mPath, mBody)

	return []*entity.Entity{
		{ID: "G-0002", Kind: entity.KindGap, Title: "Fixture", Status: "open", Path: gapPath},
		{
			ID: "M-0001", Kind: entity.KindMilestone, Title: "Bar",
			Status: "in_progress", Parent: "E-0001", TDD: "none", Path: mPath,
			ACs: []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First AC", Status: "open"}},
		},
	}
}

// writeTwoGapsBodyProseFixture is the per-entity-scoping fixture:
// two gaps with identical body prose so a per-(entity, token, subcode)
// dedupe surfaces both, while a global dedupe would mask one.
func writeTwoGapsBodyProseFixture(t *testing.T, root, prose string) []*entity.Entity {
	t.Helper()
	g1Path := "work/gaps/G-0002-fixture-a.md"
	g2Path := "work/gaps/G-0003-fixture-b.md"
	body := func(id string) string {
		return "---\nid: " + id + "\ntitle: Fixture\nstatus: open\n---\n\n## What's missing\n\n" +
			prose + "\n\n## Why it matters\n\nIt matters.\n"
	}
	mustWriteFile(t, root, g1Path, body("G-0002"))
	mustWriteFile(t, root, g2Path, body("G-0003"))
	return []*entity.Entity{
		{ID: "G-0002", Kind: entity.KindGap, Title: "Fixture", Status: "open", Path: g1Path},
		{ID: "G-0003", Kind: entity.KindGap, Title: "Fixture", Status: "open", Path: g2Path},
	}
}

// TestBodyProseID_NarrowWidth_G0518 pins the narrow-width subcode: a
// citation that resolves ONLY after canonicalization fires, and one
// that resolves as written does not.
//
// The rule is reference-shaped rather than width-shaped, so both
// legitimate spellings of a genuinely-narrow entity stay silent — the
// narrow one because read tolerance is permanent and no verb widens an
// id in place, the canonical one because that is what every aiwf
// surface prints. A narrow token resolving nowhere is a resolution
// defect rather than a width one and keeps reporting `unresolved`.
func TestBodyProseID_NarrowWidth_G0518(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		// extraID, when non-empty, is written as one more entity so the
		// prose has something beyond the base fixture to resolve
		// against. archived places it under the kind's archive subdir,
		// which is where a narrow id lives once canonical width is
		// adopted — no verb widens one in place.
		extraID  string
		archived bool
		// stubID, when non-empty, enters the tree as a parse-failure
		// stub rather than a loaded entity.
		stubID      string
		wantSubcode string
		wantMessage string
		silent      bool
	}{
		{
			name:        "narrow bare id naming a canonical entity",
			body:        "The render is invisible to anyone auditing M-123.",
			extraID:     "M-0123",
			wantSubcode: "narrow-width",
			wantMessage: `id "M-123" below canonical width — write M-0123`,
		},
		{
			name:        "narrow epic id, two digits below canonical",
			body:        "Scope leaks through E-19's depends_on chain.",
			extraID:     "E-0019",
			wantSubcode: "narrow-width",
			wantMessage: `write E-0019`,
		},
		{
			name:        "narrow composite parent",
			body:        "Per M-001/AC-1, the AC holds.",
			wantSubcode: "narrow-width",
			wantMessage: `id "M-001/AC-1" below canonical width — write M-0001/AC-1`,
		},
		{
			name:   "canonical bare id",
			body:   "Per M-0001, the rule applies.",
			silent: true,
		},
		{
			name:     "narrow token naming an entity stored narrow",
			body:     "Superseded by G-018, archived before canonical width.",
			extraID:  "G-018",
			archived: true,
			silent:   true,
		},
		{
			name:     "canonical token naming an entity stored narrow",
			body:     "Superseded by G-0018, archived before canonical width.",
			extraID:  "G-018",
			archived: true,
			silent:   true,
		},
		{
			name:        "narrow token resolving nowhere stays unresolved",
			body:        "See G-777 for the proposed rule.",
			wantSubcode: "unresolved",
		},
		{
			// The parent segment is what carries the width, so the
			// as-written test has to be applied to it and not to the
			// whole composite token — no entity is ever stored under a
			// composite spelling, so testing the token would make the
			// silence path unreachable here.
			name:    "narrow composite parent naming a milestone stored narrow",
			body:    "Per M-002/AC-1, the AC holds.",
			extraID: "M-002",
			silent:  true,
		},
		{
			// Width is settled the moment the parent resolves, so it is
			// reported ahead of the AC position rather than after it.
			name:        "narrow composite parent with an absent AC reports the width",
			body:        "Per M-001/AC-9, the gap is closed.",
			wantSubcode: "narrow-width",
			wantMessage: `write M-0001/AC-9`,
		},
		{
			// A stub is an entity whose file failed to parse. That
			// failure is already its own finding, so resolution against
			// a stub is silent — including its stored spelling.
			name:   "narrow token naming a narrow stub",
			body:   "Superseded by G-018, whose file does not parse.",
			stubID: "G-018",
			silent: true,
		},
		{
			name:   "narrow token inside backticks",
			body:   "Historical trailers carry narrow widths, e.g. `G-018` for `G-0018`.",
			silent: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			ents := writeBodyProseFixture(t, root, tc.body)
			if tc.extraID != "" {
				ents = append(ents, writeExtraEntity(t, root, tc.extraID, tc.archived))
			}
			tr := &tree.Tree{Root: root, Entities: ents}
			if tc.stubID != "" {
				tr.Stubs = []*entity.Entity{{ID: tc.stubID, Path: "work/gaps/" + tc.stubID + "-unparseable.md"}}
			}

			got := bodyProseID(tr)
			if tc.silent {
				if len(got) != 0 {
					t.Fatalf("expected silent, got %d findings: %+v", len(got), got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("findings = %d, want 1: %+v", len(got), got)
			}
			f := got[0]
			if f.Subcode != tc.wantSubcode {
				t.Fatalf("Subcode = %q, want %q (message %q)", f.Subcode, tc.wantSubcode, f.Message)
			}
			// narrow-width is non-blocking: it would otherwise refuse
			// edit-body / import / reallocate over a pre-existing
			// citation the verb is not touching. Every other subcode
			// here blocks.
			wantSeverity := SeverityError
			if tc.wantSubcode == "narrow-width" {
				wantSeverity = SeverityWarning
			}
			if f.Severity != wantSeverity {
				t.Errorf("Severity = %v, want %v", f.Severity, wantSeverity)
			}
			if tc.wantMessage != "" && !strings.Contains(f.Message, tc.wantMessage) {
				t.Errorf("Message = %q, want it to contain %q", f.Message, tc.wantMessage)
			}
		})
	}
}

// TestBodyProseID_NarrowWidth_ThroughLoader_G0518 crosses the seam the
// table test above stops short of. Every case there hands bodyProseID a
// hand-built *entity.Entity, which asserts the classifier's contract but
// takes on faith the premise underneath it: that tree.Load stores an
// entity's `id:` verbatim. narrowCitation is only able to tell a narrow
// citation from a narrow entity because it does — several sibling
// helpers in tree.go canonicalize on read, and an entity loaded that way
// would invert this rule for exactly the trees it protects.
//
// So this drives the real loader against a real archived narrow entity
// and asserts the silence end to end.
func TestBodyProseID_NarrowWidth_ThroughLoader_G0518(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	mustWriteFile(t, root, "work/gaps/archive/G-018-legacy.md",
		"---\nid: G-018\ntitle: Legacy\nstatus: wontfix\n---\n\n## What's missing\n\nNothing now.\n\n## Why it matters\n\nIt did once.\n")
	mustWriteFile(t, root, "work/gaps/G-0002-citing.md",
		"---\nid: G-0002\ntitle: Citing\nstatus: open\n---\n\n## What's missing\n\nSupersedes G-018, archived before canonical width.\n\n## Why it matters\n\nIt matters.\n")

	tr, _, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	legacy := tr.ByID("G-018")
	if legacy == nil {
		t.Fatal("tree.Load did not load the archived narrow gap")
	}
	if legacy.ID != "G-018" {
		t.Fatalf("loaded ID = %q, want %q verbatim — narrowCitation cannot tell a narrow citation "+
			"from a narrow entity once the loader canonicalizes", legacy.ID, "G-018")
	}
	if got := bodyProseID(tr); len(got) != 0 {
		t.Fatalf("citing an archived narrow entity at its stored width fired %d findings, want 0: %+v", len(got), got)
	}
}

// writeExtraEntity writes one more entity into the fixture tree so body
// prose has something beyond the base fixture to resolve against, and
// returns its loaded form. id is stored verbatim, since a narrow
// spelling is what these cases turn on; archived places the file under
// the kind's archive subdir. The body carries no id-shaped token, so
// scanning it adds no finding of its own.
func writeExtraEntity(t *testing.T, root, id string, archived bool) *entity.Entity {
	t.Helper()
	var dir, name string
	var kind entity.Kind
	switch {
	case strings.HasPrefix(id, "G-"):
		dir, name, kind = "work/gaps", id+"-extra.md", entity.KindGap
	case strings.HasPrefix(id, "E-"):
		dir, name, kind = "work/epics", id+"-extra/epic.md", entity.KindEpic
	case strings.HasPrefix(id, "M-"):
		// Under the epic writeBodyProseFixture already laid down.
		dir, name, kind = "work/epics/E-0001-foo", id+"-extra.md", entity.KindMilestone
	default:
		t.Fatalf("fixture has no layout for id %q", id)
	}
	if archived {
		dir += "/archive"
	}
	rel := dir + "/" + name
	mustWriteFile(t, root, rel,
		"---\nid: "+id+"\ntitle: Extra\nstatus: open\n---\n\n## Goal\n\nSomething.\n")
	e := &entity.Entity{ID: id, Kind: kind, Title: "Extra", Status: "open", Path: rel}
	if kind == entity.KindMilestone {
		// So a composite citation of this milestone has a position to
		// resolve against, and the AC arm is reached rather than
		// short-circuited by a missing AC.
		e.ACs = []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First AC", Status: "open"}}
	}
	return e
}

func mustWriteFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", abs, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
}
