package verb

// rewidth_test.go — M-082 unit tests for the rewidth verb's plan
// computation. The dispatcher-level tests live in
// `cmd/aiwf/rewidth_cmd_test.go` (AC-1 surface). These tests pin
// AC-2 (rename), AC-3 (body rewrite), and AC-4 (idempotence) at
// the verb-body layer so a regression in either surface is caught
// independently.
//
// Per CLAUDE.md "Test the seam, not just the layer," AC-2/3/4 each
// have at least one cmd/aiwf-level test (driving `run([]string{...})`)
// and at least one internal/verb-level test (driving Rewidth or
// planRewidth directly with a synthetic root).

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// fixtureWriter is a tiny test helper: a root path + a write method
// that creates parent dirs and writes the file. Each test composes a
// custom narrow-tree under t.TempDir() this way.
type fixtureWriter struct {
	t    *testing.T
	root string
}

func newFixture(t *testing.T) *fixtureWriter {
	t.Helper()
	return &fixtureWriter{t: t, root: t.TempDir()}
}

func (f *fixtureWriter) write(rel, body string) {
	f.t.Helper()
	full := filepath.Join(f.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		f.t.Fatalf("mkdir %s: %v", filepath.Dir(rel), err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		f.t.Fatalf("write %s: %v", rel, err)
	}
}

// renamePairsToStrings flattens a rename slice into "from -> to"
// strings for assertion-friendly diffs.
func renamePairsToStrings(rs []renamePair) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.from + " -> " + r.to
	}
	return out
}

// rewriteRewritesToPaths returns just the post-move paths of every
// rewrite op (sorted) for set-membership-style assertions.
func rewriteRewritesToPaths(ws []rewidthRewrite) []string {
	out := make([]string, len(ws))
	for i, w := range ws {
		out[i] = w.path
	}
	sort.Strings(out)
	return out
}

// AC-2 — Active-tree rename in each composite case.

// TestPlanRewidth_EpicAndMilestoneNarrow — composite case: epic dir
// is narrow AND its milestones are narrow. Both rename; the milestone
// `from` paths are post-epic-rename so Apply can run moves in order.
func TestPlanRewidth_EpicAndMilestoneNarrow(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.write("work/epics/E-22-foo/epic.md",
		"---\nid: E-22\ntitle: Foo\nstatus: active\n---\nbody.\n")
	f.write("work/epics/E-22-foo/M-77-bar.md",
		"---\nid: M-77\ntitle: Bar\nstatus: in_progress\nparent: E-22\n---\nbody.\n")

	renames, err := planRewidthRenames(f.root)
	if err != nil {
		t.Fatalf("planRewidthRenames: %v", err)
	}

	got := renamePairsToStrings(renames)
	want := []string{
		"work/epics/E-22-foo -> work/epics/E-0022-foo",
		"work/epics/E-0022-foo/M-77-bar.md -> work/epics/E-0022-foo/M-0077-bar.md",
	}
	// Renames are emitted in walk order: epic first (dir-shape kinds
	// before file-shape inside them), then milestone. Pin the order
	// per the spec's determinism requirement.
	if !sliceEqual(got, want) {
		t.Errorf("rename pairs mismatch:\n  got:  %v\n  want: %v", got, want)
	}
}

// TestPlanRewidth_EpicCanonicalMilestoneNarrow — composite case:
// epic dir is already canonical, but a milestone inside is narrow.
// Only the milestone renames in place.
func TestPlanRewidth_EpicCanonicalMilestoneNarrow(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.write("work/epics/E-0022-foo/epic.md",
		"---\nid: E-0022\ntitle: Foo\nstatus: active\n---\nbody.\n")
	f.write("work/epics/E-0022-foo/M-77-bar.md",
		"---\nid: M-77\ntitle: Bar\nstatus: in_progress\nparent: E-0022\n---\nbody.\n")

	renames, err := planRewidthRenames(f.root)
	if err != nil {
		t.Fatalf("planRewidthRenames: %v", err)
	}
	got := renamePairsToStrings(renames)
	want := []string{
		"work/epics/E-0022-foo/M-77-bar.md -> work/epics/E-0022-foo/M-0077-bar.md",
	}
	if !sliceEqual(got, want) {
		t.Errorf("rename pairs mismatch:\n  got:  %v\n  want: %v", got, want)
	}
}

// TestPlanRewidth_EpicNarrowMilestoneCanonical — composite case:
// epic dir is narrow but the milestone inside is already canonical.
// Only the epic renames; the milestone filename inside stays.
func TestPlanRewidth_EpicNarrowMilestoneCanonical(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.write("work/epics/E-22-foo/epic.md",
		"---\nid: E-22\ntitle: Foo\nstatus: active\n---\nbody.\n")
	f.write("work/epics/E-22-foo/M-0077-bar.md",
		"---\nid: M-0077\ntitle: Bar\nstatus: in_progress\nparent: E-22\n---\nbody.\n")

	renames, err := planRewidthRenames(f.root)
	if err != nil {
		t.Fatalf("planRewidthRenames: %v", err)
	}
	got := renamePairsToStrings(renames)
	want := []string{
		"work/epics/E-22-foo -> work/epics/E-0022-foo",
	}
	if !sliceEqual(got, want) {
		t.Errorf("rename pairs mismatch:\n  got:  %v\n  want: %v", got, want)
	}
}

// TestPlanRewidth_AllKindsNarrow — verifies every kind's directory
// gets walked when populated with a narrow file. Confirms walk order
// (epic, milestone, gap, decision, contract, adr) per the spec.
func TestPlanRewidth_AllKindsNarrow(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.write("work/epics/E-22-e/epic.md", "---\nid: E-22\ntitle: E\nstatus: active\n---\n")
	f.write("work/epics/E-22-e/M-77-m.md", "---\nid: M-77\ntitle: M\nstatus: in_progress\nparent: E-22\n---\n")
	f.write("work/gaps/G-9-g.md", "---\nid: G-9\ntitle: G\nstatus: open\n---\n")
	f.write("work/decisions/D-3-d.md", "---\nid: D-3\ntitle: D\nstatus: proposed\n---\n## Question\n## Decision\n## Reasoning\n")
	f.write("work/contracts/C-5-c/contract.md", "---\nid: C-5\ntitle: C\nstatus: draft\n---\n")
	// ADR was always at canonical width by ADR-0007; planRewidth
	// should emit no rename for it even when present.
	f.write("docs/adr/ADR-0042-already-canonical.md", "---\nid: ADR-0042\ntitle: A\nstatus: proposed\n---\n## Context\n## Decision\n## Consequences\n")

	renames, err := planRewidthRenames(f.root)
	if err != nil {
		t.Fatalf("planRewidthRenames: %v", err)
	}
	got := renamePairsToStrings(renames)
	want := []string{
		// epic first (composite-parent kind)
		"work/epics/E-22-e -> work/epics/E-0022-e",
		// milestone next (post-epic-rename path)
		"work/epics/E-0022-e/M-77-m.md -> work/epics/E-0022-e/M-0077-m.md",
		// gap, decision, contract, adr in fixed sequence
		"work/gaps/G-9-g.md -> work/gaps/G-0009-g.md",
		"work/decisions/D-3-d.md -> work/decisions/D-0003-d.md",
		"work/contracts/C-5-c -> work/contracts/C-0005-c",
		// ADR omitted: already canonical
	}
	if !sliceEqual(got, want) {
		t.Errorf("rename pairs mismatch:\n  got:  %v\n  want: %v", got, want)
	}
}

// TestPlanRewidth_ArchiveSkipped — files under <kind>/archive/ MUST
// NOT be touched. Per ADR-0004's forget-by-default principle.
func TestPlanRewidth_ArchiveSkipped(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	// Active narrow file (should rename).
	f.write("work/gaps/G-9-active.md", "---\nid: G-9\ntitle: A\nstatus: open\n---\n")
	// Archive narrow file (should NOT rename).
	f.write("work/gaps/archive/G-2-archived.md", "---\nid: G-2\ntitle: B\nstatus: wontfix\n---\n")

	renames, err := planRewidthRenames(f.root)
	if err != nil {
		t.Fatalf("planRewidthRenames: %v", err)
	}
	got := renamePairsToStrings(renames)
	for _, r := range got {
		if strings.Contains(r, "archive") {
			t.Errorf("archive entry %q appears in rename plan — must be skipped per ADR-0004", r)
		}
	}
	want := []string{"work/gaps/G-9-active.md -> work/gaps/G-0009-active.md"}
	if !sliceEqual(got, want) {
		t.Errorf("rename pairs mismatch:\n  got:  %v\n  want: %v", got, want)
	}
}

// TestPlanRewidth_EpicArchiveSkipped — `work/epics/archive/...` is
// skipped per ADR-0004. Covers the dir-shape kind's archive filter.
// Distinct from TestPlanRewidth_ArchiveSkipped which exercises a
// flat-dir kind (gap).
func TestPlanRewidth_EpicArchiveSkipped(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	// Active narrow epic.
	f.write("work/epics/E-22-active/epic.md", "---\nid: E-22\ntitle: A\nstatus: active\n---\n")
	// An archive subdir alongside epic dirs (the dir name is
	// literally `archive`). Whatever's inside must NOT participate
	// in the rename plan.
	f.write("work/epics/archive/E-2-old/epic.md", "---\nid: E-2\ntitle: O\nstatus: cancelled\n---\n")

	renames, err := planRewidthRenames(f.root)
	if err != nil {
		t.Fatalf("planRewidthRenames: %v", err)
	}
	for _, r := range renamePairsToStrings(renames) {
		if strings.Contains(r, "archive") {
			t.Errorf("epic-dir archive entry %q appears in rename plan — must be skipped", r)
		}
	}
}

// TestPlanRewidth_DeterministicOrderWithinKind — alphabetical-by-
// filename order within a kind, per the spec. Two narrow gaps with
// out-of-order filenames return in alphabetical order.
func TestPlanRewidth_DeterministicOrderWithinKind(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.write("work/gaps/G-9-zeta.md", "---\nid: G-9\ntitle: Z\nstatus: open\n---\n")
	f.write("work/gaps/G-3-alpha.md", "---\nid: G-3\ntitle: A\nstatus: open\n---\n")
	f.write("work/gaps/G-7-mu.md", "---\nid: G-7\ntitle: M\nstatus: open\n---\n")

	renames, err := planRewidthRenames(f.root)
	if err != nil {
		t.Fatalf("planRewidthRenames: %v", err)
	}
	got := renamePairsToStrings(renames)
	// Sorted alphabetically by filename, NOT by id-numeric value.
	want := []string{
		"work/gaps/G-3-alpha.md -> work/gaps/G-0003-alpha.md",
		"work/gaps/G-7-mu.md -> work/gaps/G-0007-mu.md",
		"work/gaps/G-9-zeta.md -> work/gaps/G-0009-zeta.md",
	}
	if !sliceEqual(got, want) {
		t.Errorf("alphabetical-by-filename order broken:\n  got:  %v\n  want: %v", got, want)
	}
}

// TestPlanRewidth_NonEntityFilesIgnored — files that don't parse as
// entity filenames (a stray README, a .gitkeep, a .DS_Store) are
// silently ignored. Covers the canonicalizeFilename "ok=false" branch
// and the file-shape suffix-mismatch branch.
func TestPlanRewidth_NonEntityFilesIgnored(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	// One real narrow gap.
	f.write("work/gaps/G-9-real.md", "---\nid: G-9\ntitle: R\nstatus: open\n---\n")
	// Stray non-id file in the gap dir (no prefix match — covers
	// the !HasPrefix(layout.prefix) branch in the gap-walk).
	f.write("work/gaps/README.md", "# notes\n")
	// File with the right prefix but no `.md` suffix (covers
	// !HasSuffix(.md) for the gap-walk).
	f.write("work/gaps/G-12-no-suffix", "binary?\n")
	// Gap-prefixed but unparseable (no digits after `G-`); covers
	// the gap-walk's canonicalizeFilename ok=false branch.
	f.write("work/gaps/G-no-digits.md", "malformed.\n")
	// Hidden file (skipped by readActiveDirSorted's . filter).
	f.write("work/gaps/.gitkeep", "")
	// Extra non-dir entry alongside epic dirs (covers !isDir in
	// the dir-shape walk).
	f.write("work/epics/README.md", "# epics readme\n")
	// Extra non-id directory alongside epic dirs (covers
	// canonicalizeFilename returning ok=false for the dir-shape
	// kind — name doesn't have an [E-] prefix).
	f.write("work/epics/scratch-area/notes.md", "# scratch\n")
	// Inside an epic dir: an `epic.md` (filtered out by the
	// !HasPrefix(M-) check), a non-milestone-prefix file (also
	// !HasPrefix), and a malformed M-prefixed entry that's not .md
	// (covers the !HasSuffix branch inside milestone-walk).
	f.write("work/epics/E-22-foo/epic.md", "---\nid: E-22\ntitle: F\nstatus: active\n---\n")
	f.write("work/epics/E-22-foo/notes.txt", "scratch.\n")
	f.write("work/epics/E-22-foo/M-77-real.md", "---\nid: M-77\ntitle: R\nstatus: in_progress\nparent: E-22\n---\n")
	f.write("work/epics/E-22-foo/M-prefix-no-md", "M- prefix but not .md\n")
	f.write("work/epics/E-22-foo/M-no-digits-bad.md", "malformed.\n")

	renames, err := planRewidthRenames(f.root)
	if err != nil {
		t.Fatalf("planRewidthRenames: %v", err)
	}
	got := renamePairsToStrings(renames)
	want := []string{
		"work/epics/E-22-foo -> work/epics/E-0022-foo",
		"work/epics/E-0022-foo/M-77-real.md -> work/epics/E-0022-foo/M-0077-real.md",
		"work/gaps/G-9-real.md -> work/gaps/G-0009-real.md",
	}
	if !sliceEqual(got, want) {
		t.Errorf("non-entity-files-ignored mismatch:\n  got:  %v\n  want: %v", got, want)
	}
}

// TestPlanRewidth_MissingKindDirSkipped — a kind directory that
// doesn't exist on disk shouldn't error. The verb runs against any
// consumer tree; absence of work/contracts/ is normal.
func TestPlanRewidth_MissingKindDirSkipped(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.write("work/gaps/G-9-foo.md", "---\nid: G-9\ntitle: F\nstatus: open\n---\n")
	// No work/epics, work/decisions, work/contracts, docs/adr.

	renames, err := planRewidthRenames(f.root)
	if err != nil {
		t.Fatalf("planRewidthRenames: %v", err)
	}
	got := renamePairsToStrings(renames)
	want := []string{"work/gaps/G-9-foo.md -> work/gaps/G-0009-foo.md"}
	if !sliceEqual(got, want) {
		t.Errorf("missing-kind-dir handling broken:\n  got:  %v\n  want: %v", got, want)
	}
}

// AC-3 — body-content rewrite engine.

// TestRewriteRewidthBody_BareIDsInProse covers the core rewrite case:
// `E-22` → `E-0022`. Word-boundary guards for trailing digits.
func TestRewriteRewidthBody_BareIDsInProse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		// Per ADR-0008 §"Reference-rewrite scope": bare-id mentions
		// `\b[EMGDCF]-[0-9]{1,3}\b` rewrite to canonical 4-digit.
		{"epic narrow", "See E-22 for context.", "See E-0022 for context."},
		{"milestone narrow", "M-77 depends on M-50.", "M-0077 depends on M-0050."},
		{"gap narrow", "G-9 was filed yesterday.", "G-0009 was filed yesterday."},
		{"decision narrow", "Per D-3 the policy is set.", "Per D-0003 the policy is set."},
		{"contract narrow", "C-5 binds the validator.", "C-0005 binds the validator."},
		// `F-NNN` is the planned 7th kind from §07 TDD architecture
		// proposal; including F is forward-compatible per the spec
		// (today's trees have no F entities so rewrite is a no-op
		// shape-wise but the regex must accept it).
		{"finding narrow forward-compat", "F-2 was discovered.", "F-0002 was discovered."},

		// Trailing-digit guard: `E-220` (3 digits but it's already a
		// matchable narrow value since 220 fits in the [0-9]{1,3} range);
		// this WILL canonicalize to E-0220. So the trailing guard's
		// real job is preventing `E-22` from greedily matching `E-220`.
		// Testing the boundary: a 4-digit number is not in [0-9]{1,3}
		// but `\b` still wouldn't fire mid-number — confirm `E-2200` is
		// untouched (the 4-digit form is already canonical-or-wider).
		{"already-canonical untouched", "E-0022 is canonical.", "E-0022 is canonical."},
		{"five-digit untouched", "E-22000 stays.", "E-22000 stays."},
		{"four-digit untouched", "E-2200 stays.", "E-2200 stays."},

		// Negative cases: malformed or wrong-prefix tokens stay verbatim.
		{"no dash", "E22 isn't an id.", "E22 isn't an id."},
		{"lowercase prefix", "e-22 not an id either.", "e-22 not an id either."},
		{"wrong prefix", "EM-22 has the wrong shape.", "EM-22 has the wrong shape."},
		// ADR has no narrow form (always 4-digit since ADR-0007), so
		// `ADR-22` is not in the bare-id pattern's grammar.
		{"adr untouched", "ADR-22 not in narrow grammar.", "ADR-22 not in narrow grammar."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := string(rewriteRewidthBody([]byte(c.in)))
			if got != c.want {
				t.Errorf("rewrite mismatch:\n  in:   %q\n  got:  %q\n  want: %q", c.in, got, c.want)
			}
		})
	}
}

// TestRewriteRewidthBody_CompositeIDs covers `M-NN/AC-N` form.
// The AC suffix is preserved verbatim; only the milestone portion
// rewrites.
func TestRewriteRewidthBody_CompositeIDs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"See M-22/AC-1 for context.", "See M-0022/AC-1 for context."},
		{"M-7/AC-12 met yesterday.", "M-0007/AC-12 met yesterday."},
		// Already-canonical composite stays.
		{"M-0022/AC-1 already canonical.", "M-0022/AC-1 already canonical."},
		// Multi-mention on one line.
		{"both M-22/AC-1 and M-77/AC-2 fire.", "both M-0022/AC-1 and M-0077/AC-2 fire."},
	}
	for _, c := range cases {
		got := string(rewriteRewidthBody([]byte(c.in)))
		if got != c.want {
			t.Errorf("composite mismatch:\n  in:   %q\n  got:  %q\n  want: %q", c.in, got, c.want)
		}
	}
}

// TestRewriteRewidthBody_MarkdownLinks covers active-tree link
// rewriting. Archive-prefixed link paths are NOT rewritten.
func TestRewriteRewidthBody_MarkdownLinks(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"epic dir link",
			"See [foo](work/epics/E-22-foo).",
			"See [foo](work/epics/E-0022-foo).",
		},
		{
			"file link with .md",
			"See [bar](work/gaps/G-9-bar.md).",
			"See [bar](work/gaps/G-0009-bar.md).",
		},
		{
			"archive link untouched",
			"See [archived](work/gaps/archive/G-001-foo.md).",
			"See [archived](work/gaps/archive/G-001-foo.md).",
		},
		{
			"non-active path untouched",
			"See [external](https://example.com/E-22-issue).",
			"See [external](https://example.com/E-22-issue).",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := string(rewriteRewidthBody([]byte(c.in)))
			if got != c.want {
				t.Errorf("link rewrite mismatch:\n  in:   %q\n  got:  %q\n  want: %q", c.in, got, c.want)
			}
		})
	}
}

// TestRewriteRewidthBody_CodeFences pins the code-fence exclusion:
// content inside triple-backtick blocks is preserved verbatim. This
// is load-bearing — kernel docs frequently quote literal narrow ids
// in shell snippets ("aiwf show E-22").
func TestRewriteRewidthBody_CodeFences(t *testing.T) {
	t.Parallel()
	in := "Outside: E-22 should rewrite.\n" +
		"```\n" +
		"Inside fence: E-22 stays narrow.\n" +
		"M-77/AC-1 also stays.\n" +
		"```\n" +
		"After fence: E-22 rewrites again.\n"
	want := "Outside: E-0022 should rewrite.\n" +
		"```\n" +
		"Inside fence: E-22 stays narrow.\n" +
		"M-77/AC-1 also stays.\n" +
		"```\n" +
		"After fence: E-0022 rewrites again.\n"
	got := string(rewriteRewidthBody([]byte(in)))
	if got != want {
		t.Errorf("code-fence preservation broken:\n  got:  %q\n  want: %q", got, want)
	}
}

// TestRewriteRewidthBody_InlineBackticks pins the inline-code-span
// exclusion. Kernel docs use “ `E-22` “ to denote literal id text;
// rewriting that would erase the literal-quote semantics.
func TestRewriteRewidthBody_InlineBackticks(t *testing.T) {
	t.Parallel()
	in := "Mention `E-22` literally vs. mention E-22 in prose."
	want := "Mention `E-22` literally vs. mention E-0022 in prose."
	got := string(rewriteRewidthBody([]byte(in)))
	if got != want {
		t.Errorf("inline-backtick preservation broken:\n  got:  %q\n  want: %q", got, want)
	}
}

// TestRewriteRewidthBody_BareURLPreserved — `https://...` URLs in
// prose (not inside a markdown link) are preserved verbatim. The
// AC-3 spec's URL-fragment exclusion applies regardless of whether
// the URL is inside `(...)`.
func TestRewriteRewidthBody_BareURLPreserved(t *testing.T) {
	t.Parallel()
	in := "Visit https://example.com/E-22-issue today; also see E-22 in prose."
	want := "Visit https://example.com/E-22-issue today; also see E-0022 in prose."
	got := string(rewriteRewidthBody([]byte(in)))
	if got != want {
		t.Errorf("bare-url preservation broken:\n  got:  %q\n  want: %q", got, want)
	}
}

// TestRewriteRewidthBody_UnbalancedLinkParens — a `]` followed by
// `(` with no matching `)` later in the line. The
// splitLinkPathRegions function falls into its `closeRel < 0`
// defensive branch and treats the rest of the chunk as
// outside-link prose. Malformed markdown — but we still want a
// graceful, predictable behaviour: out-of-link rewriting fires on
// every narrow id past the unmatched `(`. Bare-id matches inside
// path-form text (e.g., `G-9` next to `/`) DO rewrite per the
// outside-link bare-id pass, even though that's not what a
// well-formed link would look like. The test pins the behaviour so
// a future change to splitLinkPathRegions doesn't silently shift
// the contract.
func TestRewriteRewidthBody_UnbalancedLinkParens(t *testing.T) {
	t.Parallel()
	in := "Look at [text](work/gaps/G-9-foo.md and continue with E-22"
	// Both `G-9` and `E-22` are rewritten because the absence of a
	// closing `)` puts everything into the outside-link region.
	want := "Look at [text](work/gaps/G-0009-foo.md and continue with E-0022"
	got := string(rewriteRewidthBody([]byte(in)))
	if got != want {
		t.Errorf("unbalanced parens:\n  got:  %q\n  want: %q", got, want)
	}
}

// TestRewriteRewidthBody_EmptyInput exercises tokenizeBySpace's
// empty-input branch (returns nil) and the rewriter's pass-through
// behavior for empty content.
func TestRewriteRewidthBody_EmptyInput(t *testing.T) {
	t.Parallel()
	if got := string(rewriteRewidthBody([]byte(""))); got != "" {
		t.Errorf("empty input: got %q, want empty string", got)
	}
}

// TestRewriteRewidthBody_MultipleSpansPerLine confirms the inline-
// span tracker handles multiple spans on one line correctly.
func TestRewriteRewidthBody_MultipleSpansPerLine(t *testing.T) {
	t.Parallel()
	in := "First `E-22` then E-22 then `E-77` then E-77."
	want := "First `E-22` then E-0022 then `E-77` then E-0077."
	got := string(rewriteRewidthBody([]byte(in)))
	if got != want {
		t.Errorf("multi-span tracking broken:\n  got:  %q\n  want: %q", got, want)
	}
}

// TestRewriteRewidthBody_UnterminatedSpan exercises the defensive
// "unterminated span on this line — treat as in-span" path. An odd
// number of backticks on a line is markdown-illegal but real
// documents have them; we conservatively don't rewrite so we don't
// silently mangle prose.
func TestRewriteRewidthBody_UnterminatedSpan(t *testing.T) {
	t.Parallel()
	in := "Mid-line ` opens but never closes E-22."
	// Buffer captures everything after `, treats as in-span verbatim.
	want := "Mid-line ` opens but never closes E-22."
	got := string(rewriteRewidthBody([]byte(in)))
	if got != want {
		t.Errorf("unterminated span: %q vs want %q", got, want)
	}
}

// TestRewriteRewidthBody_Idempotent — running rewrite twice produces
// the same output as running it once. AC-4's textual-rewrite half.
func TestRewriteRewidthBody_Idempotent(t *testing.T) {
	t.Parallel()
	in := "E-22 and M-77/AC-1 and [link](work/gaps/G-9-foo.md)."
	once := rewriteRewidthBody([]byte(in))
	twice := rewriteRewidthBody(once)
	if !bytes.Equal(once, twice) {
		t.Errorf("rewrite not idempotent:\n  once:  %q\n  twice: %q", once, twice)
	}
}

// AC-4 — Idempotent and deterministic on canonical / empty trees.

// TestRewidth_AlreadyCanonical_NoOp — calling Rewidth on a tree where
// every id is already canonical produces a NoOp result. No commit.
func TestRewidth_AlreadyCanonical_NoOp(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.write("work/epics/E-0022-foo/epic.md",
		"---\nid: E-0022\ntitle: Foo\nstatus: active\n---\nbody.\n")
	f.write("work/gaps/G-0009-bar.md",
		"---\nid: G-0009\ntitle: Bar\nstatus: open\n---\n## What's missing\n\nNo refs.\n")

	res, err := Rewidth(context.Background(), f.root, "human/test")
	if err != nil {
		t.Fatalf("Rewidth: %v", err)
	}
	if !res.NoOp {
		t.Errorf("Rewidth on already-canonical tree must be NoOp; got Plan = %+v", res.Plan)
	}
	if res.Plan != nil {
		t.Errorf("NoOp result must have nil Plan; got %+v", res.Plan)
	}
	if !strings.Contains(res.NoOpMessage, "no changes") {
		t.Errorf("NoOpMessage = %q, want a 'no changes' message", res.NoOpMessage)
	}
}

// TestRewidth_EmptyTree_NoOp — calling Rewidth on a tree with no
// entity files at all (just the consumer-repo scaffolding) produces
// NoOp. Caller exits 0; no commit.
func TestRewidth_EmptyTree_NoOp(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	// No files written. The kind dirs may not exist either; verify
	// that's fine.
	res, err := Rewidth(context.Background(), f.root, "human/test")
	if err != nil {
		t.Fatalf("Rewidth: %v", err)
	}
	if !res.NoOp {
		t.Errorf("Rewidth on empty tree must be NoOp; got %+v", res)
	}
}

// TestRewidth_MixedState_OnlyNarrowMigrates — when some files are
// canonical and others narrow, only the narrow ones change. The
// canonical files are byte-identical pre/post. AC-4 mixed-state.
func TestRewidth_MixedState_OnlyNarrowMigrates(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	canonicalBody := "---\nid: G-0099\ntitle: C\nstatus: open\n---\n## What's missing\n\nNo refs.\n"
	f.write("work/gaps/G-0099-canonical.md", canonicalBody)
	f.write("work/gaps/G-9-narrow.md",
		"---\nid: G-9\ntitle: N\nstatus: open\n---\n## What's missing\n\nNo refs.\n")

	plan, err := planRewidth(f.root)
	if err != nil {
		t.Fatalf("planRewidth: %v", err)
	}
	if plan == nil {
		t.Fatal("expected a Plan for mixed-state tree, got nil")
	}

	// Only the narrow file should appear in renames or rewrites.
	for _, op := range plan.Ops {
		switch op.Type {
		case OpMove:
			if !strings.Contains(op.Path, "G-9-narrow") {
				t.Errorf("unexpected move on canonical file: %s -> %s", op.Path, op.NewPath)
			}
		case OpWrite:
			if strings.Contains(op.Path, "G-0099-canonical") {
				t.Errorf("OpWrite on canonical file %q — should be byte-identical", op.Path)
			}
		}
	}

	// And the canonical file on disk is unchanged (planRewidth doesn't
	// write yet — it's a pure plan computation).
	got, err := os.ReadFile(filepath.Join(f.root, "work", "gaps", "G-0099-canonical.md"))
	if err != nil {
		t.Fatalf("read canonical: %v", err)
	}
	if string(got) != canonicalBody {
		t.Errorf("canonical file diverged from on-disk byte content (would not be byte-identical)")
	}
}

// TestPlanRewidth_NoOpOnCanonical — direct test that planRewidth
// returns nil (no plan) when the tree has no work to do. Distinguishes
// from the public Rewidth result (NoOp) so the caller-vs-verb seam is
// asserted at both layers.
func TestPlanRewidth_NoOpOnCanonical(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.write("work/gaps/G-0099-c.md", "---\nid: G-0099\ntitle: C\nstatus: open\n---\nbody.\n")

	plan, err := planRewidth(f.root)
	if err != nil {
		t.Fatalf("planRewidth: %v", err)
	}
	if plan != nil {
		t.Errorf("planRewidth on canonical tree returned non-nil Plan with %d ops", len(plan.Ops))
	}
}

// TestPlanRewidthRewrites_PostMovePathing — when a milestone file
// lives inside an epic dir that's being renamed, the rewrite path
// must be the post-move location so OpWrite lands at the new path.
// Asserts the seam between rename and rewrite.
func TestPlanRewidthRewrites_PostMovePathing(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	// Milestone body has no rewritable content at first, so we
	// give it some prose that needs rewriting.
	f.write("work/epics/E-22-x/epic.md",
		"---\nid: E-22\ntitle: X\nstatus: active\n---\n## Goal\n\nMentions M-77 in body.\n")
	f.write("work/epics/E-22-x/M-77-y.md",
		"---\nid: M-77\ntitle: Y\nstatus: in_progress\nparent: E-22\n---\n## Goal\n\nMentions E-22 here.\n")

	plan, err := planRewidth(f.root)
	if err != nil {
		t.Fatalf("planRewidth: %v", err)
	}
	if plan == nil {
		t.Fatal("expected a Plan, got nil")
	}

	// Collect every OpWrite path. Both files have body content that
	// references narrow ids, so both should be rewritten — and at
	// their post-move locations.
	var writePaths []string
	for _, op := range plan.Ops {
		if op.Type == OpWrite {
			writePaths = append(writePaths, op.Path)
		}
	}
	sort.Strings(writePaths)
	want := []string{
		"work/epics/E-0022-x/M-0077-y.md", // post-rename file path
		"work/epics/E-0022-x/epic.md",     // post-rename dir
	}
	sort.Strings(want)
	if !sliceEqual(writePaths, want) {
		t.Errorf("post-move write paths mismatch:\n  got:  %v\n  want: %v", writePaths, want)
	}
}

// sliceEqual is a tiny helper. We don't pull in go-cmp here because
// the strings are flat and equality is the only comparison.
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// rewriteRewritesToPaths is exported for cross-test reuse if needed.
var _ = rewriteRewritesToPaths
