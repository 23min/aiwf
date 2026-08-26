package verb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/tree"
)

// TestIDMatcherAgreement pins the relationship between the two walks
// that answer "does this body mention this id".
//
// findProseMentions matches raw body bytes against proseRewritePattern,
// built from entity.IDGrepAlternation. check.CitersOf matches
// idTokenPattern against a masked copy and compares through
// entity.Canonicalize. Both resolve a width to an entity the same way,
// because the alternation is derived from Canonicalize; what remains is
// reach. findProseMentions reads raw body bytes, so it sees every
// non-prose carrier the mask blanks — among them link destinations and
// titles, reference-link definitions, autolinks, and a fenced block's
// info string. The destination row below is
// the load-bearing member of that class, because a path left alone is a
// path broken; the others diverge the same way for the same reason.
//
// The walks differ in one further respect this table does not cover:
// CitersOf drops the target's own body, archived files and records at a
// terminal status, none of which findProseMentions drops. Those
// exclusions are listed on CitersOf and pinned by its own tests; this
// table is about the matching.
//
// Driving both real call sites rather than reproducing either matcher
// is the point: a reproduction would pin the copy, and the copy is
// what drifts.
func TestIDMatcherAgreement(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		target      string
		prose       string
		wantRewrite bool // findProseMentions: reallocate would repair this body
		wantCiter   bool // check.CitersOf: a closure notice would name it
		why         string
	}{
		{
			name:        "canonical mention",
			target:      "E-0022",
			prose:       "The work tracked as E-0022 is the premise here.",
			wantRewrite: true, wantCiter: true,
			why: "the ordinary case both exist to catch; the reference point the rows below are read against",
		},
		{
			name:        "legal narrower width",
			target:      "E-0022",
			prose:       "The work tracked as E-22 is the premise here.",
			wantRewrite: true, wantCiter: true,
			why: "ADR-0008's permanent tolerance: an epic's floor is two digits, so this spells the same entity and both matchers must resolve it",
		},
		{
			name:        "composite names its parent",
			target:      "M-0007",
			prose:       "This rests on M-0007/AC-2 landing first.",
			wantRewrite: true, wantCiter: true,
			why: "reached by different means — the parent prefix matches on its own, and the trailing word boundary does not veto it because '/' is not a word character, while CitersOf widens explicitly in names()",
		},
		{
			name:        "width below the kind's floor",
			target:      "M-0007",
			prose:       "The work tracked as M-7 is the premise here.",
			wantRewrite: false, wantCiter: false,
			why: "M-7 is below the milestone floor, so it names no entity at all; rewriting it would turn a malformed token into a live reference",
		},
		{
			name:        "width above the canonical pad",
			target:      "M-0007",
			prose:       "This rests on M-00007 landing first.",
			wantRewrite: false, wantCiter: false,
			why: "M-00007 is a legal milestone of its own — the grammar takes three digits or more and CanonicalPad is a minimum — so matching it would rewrite a citation of a different entity",
		},
		{
			name:        "mention only in a link destination",
			target:      "E-0022",
			prose:       "See [the platform epic](../epics/E-0022-platform/epic.md) for context.",
			wantRewrite: true, wantCiter: false,
			why: "the deliberate divergence, and the member with a live consequence of a wider class — reallocate reads raw bytes, so every non-prose carrier the mask blanks parts the two the same way. It must repair a path it would otherwise break; a URL is not a claim a closure can falsify",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := buildMatcherTree(t, tc.prose)

			mentions, err := findProseMentions(tr, tc.target)
			if err != nil {
				t.Fatalf("findProseMentions(%s): %v", tc.target, err)
			}
			gotRewrite := false
			for _, e := range mentions {
				if e.ID == "G-0001" {
					gotRewrite = true
				}
			}

			gotCiter := false
			for _, c := range check.CitersOf(tr, tc.target) {
				if c.ID == "G-0001" {
					gotCiter = true
				}
			}

			if gotRewrite != tc.wantRewrite {
				t.Errorf("findProseMentions(%s) saw G-0001 = %v, want %v\nprose: %s\nwhy: %s",
					tc.target, gotRewrite, tc.wantRewrite, tc.prose, tc.why)
			}
			if gotCiter != tc.wantCiter {
				t.Errorf("check.CitersOf(%s) named G-0001 = %v, want %v\nprose: %s\nwhy: %s",
					tc.target, gotCiter, tc.wantCiter, tc.prose, tc.why)
			}
		})
	}
}

// buildMatcherTree writes a tree whose only citing record is G-0001,
// carrying prose as its whole body. The epic and milestone exist so the
// targets resolve; their bodies are empty so neither becomes a second
// match.
func buildMatcherTree(t *testing.T, prose string) *tree.Tree {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"work/epics/E-0022-platform/epic.md": "---\nid: E-0022\ntitle: Platform\nstatus: active\n---\n",
		"work/epics/E-0022-platform/M-0007-cache.md": "---\nid: M-0007\ntitle: Cache warmup\n" +
			"status: in_progress\nparent: E-0022\ntdd: none\n---\n",
		"work/gaps/G-0001-citing-record.md": "---\nid: G-0001\ntitle: A record that cites\n" +
			"status: open\npriority: medium\n---\n## What's missing\n\n" + prose + "\n",
	}
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	if len(tr.Entities) != 3 {
		t.Fatalf("tree has %d entities; want 3", len(tr.Entities))
	}
	return tr
}
