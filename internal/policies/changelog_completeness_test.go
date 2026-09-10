package policies

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// ownerMap builds the rollup function from an explicit table. An id with
// no entry owes its own citation, which is what a gap, an ADR or a
// decision does.
func ownerMap(m map[string]string) func(string) string {
	return func(id string) string {
		if to, ok := m[id]; ok {
			return to
		}
		return id
	}
}

// citedSet builds the citation predicate from the ids an `[Unreleased]`
// section names.
func citedSet(ids ...string) func(string) bool {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return func(id string) bool { return set[id] }
}

// uncitedIDs returns the entity each uncited finding names, so a test
// asserts on the entities reported rather than on Detail prose.
func uncitedIDs(a changelogAudit) []string {
	out := make([]string, 0, len(a.Uncited))
	for _, u := range a.Uncited {
		out = append(out, u.ID)
	}
	return out
}

// TestDetectUncitedDeltas covers the rules AC-1 states, one row per rule
// rather than one per spelling: an entity nothing cites is reported, a
// cited one is silent, a milestone rolls up to its parent epic before
// the citation is tested, and a narrower legacy id width names the same
// entity on both sides of that test.
func TestDetectUncitedDeltas(t *testing.T) {
	t.Parallel()

	rollup := ownerMap(map[string]string{"M-0330": "E-0091"})

	tests := []struct {
		name   string
		deltas []changelogDelta
		owner  func(string) string
		cited  func(string) bool
		want   []string
	}{
		{
			name:   "an entity nothing cites is reported",
			deltas: []changelogDelta{{SHA: "aaaaaaa1", Subject: "docs(guidance): x", Entity: "G-0659"}},
			owner:  ownerMap(nil),
			cited:  citedSet(),
			want:   []string{"G-0659"},
		},
		{
			name:   "a cited entity is silent",
			deltas: []changelogDelta{{SHA: "aaaaaaa2", Subject: "docs(guidance): x", Entity: "G-0659"}},
			owner:  ownerMap(nil),
			cited:  citedSet("G-0659"),
			want:   []string{},
		},
		{
			name:   "a milestone rolls up to its epic before the citation is tested",
			deltas: []changelogDelta{{SHA: "aaaaaaa3", Subject: "feat(check): x", Entity: "M-0330"}},
			owner:  rollup,
			cited:  citedSet("E-0091"),
			want:   []string{},
		},
		{
			name:   "an uncited milestone is reported under its epic, not itself",
			deltas: []changelogDelta{{SHA: "aaaaaaa4", Subject: "feat(check): x", Entity: "M-0330"}},
			owner:  rollup,
			cited:  citedSet(),
			want:   []string{"E-0091"},
		},
		{
			name:   "a composite id resolves to its milestone before the rollup",
			deltas: []changelogDelta{{SHA: "aaaaaaa5", Subject: "feat(check): x", Entity: "M-0330/AC-1"}},
			owner:  rollup,
			cited:  citedSet("E-0091"),
			want:   []string{},
		},
		{
			name:   "a narrow legacy width names the same entity",
			deltas: []changelogDelta{{SHA: "aaaaaaa6", Subject: "docs(skills): x", Entity: "E-91"}},
			owner:  ownerMap(nil),
			cited:  citedSet("E-0091"),
			want:   []string{},
		},
		{
			name: "one entity behind several commits is reported once",
			deltas: []changelogDelta{
				{SHA: "aaaaaaa7", Subject: "docs(skills): a", Entity: "G-0659"},
				{SHA: "aaaaaaa8", Subject: "docs(skills): b", Entity: "G-0659"},
			},
			owner: ownerMap(nil),
			cited: citedSet(),
			want:  []string{"G-0659"},
		},
		{
			name:   "a commit carrying no trailer names no entity to cite",
			deltas: []changelogDelta{{SHA: "aaaaaaa9", Subject: "docs(guidance): x"}},
			owner:  ownerMap(nil),
			cited:  citedSet(),
			want:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := uncitedIDs(detectUncitedDeltas(tt.deltas, tt.owner, tt.cited))
			if !equalStrings(got, tt.want) {
				t.Errorf("uncited = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetectUncitedDeltas_Untrailered covers AC-2's rule at the pure
// core: a commit naming no entity is collected rather than dropped, and
// it is collected somewhere other than Uncited — the audit cannot
// attribute it, so it cannot claim the changelog omits it.
func TestDetectUncitedDeltas_Untrailered(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		deltas           []changelogDelta
		cited            func(string) bool
		wantUncited      []string
		wantUnattributed []string
	}{
		{
			name:             "a commit with no trailer is collected as unattributable",
			deltas:           []changelogDelta{{SHA: "aaaaaaa1", Subject: "docs(guidance): x"}},
			cited:            citedSet(),
			wantUncited:      []string{},
			wantUnattributed: []string{"aaaaaaa1"},
		},
		{
			name:             "a whitespace-only trailer names nothing either",
			deltas:           []changelogDelta{{SHA: "aaaaaaa2", Subject: "docs(guidance): x", Entity: "   "}},
			cited:            citedSet(),
			wantUncited:      []string{},
			wantUnattributed: []string{"aaaaaaa2"},
		},
		{
			name:             "a trailered commit is attributable, cited or not",
			deltas:           []changelogDelta{{SHA: "aaaaaaa3", Subject: "docs(guidance): x", Entity: "G-0659"}},
			cited:            citedSet(),
			wantUncited:      []string{"G-0659"},
			wantUnattributed: []string{},
		},
		{
			name: "the two halves are independent",
			deltas: []changelogDelta{
				{SHA: "aaaaaaa4", Subject: "docs(guidance): a"},
				{SHA: "aaaaaaa5", Subject: "docs(guidance): b", Entity: "G-0659"},
			},
			cited:            citedSet("G-0659"),
			wantUncited:      []string{},
			wantUnattributed: []string{"aaaaaaa4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := detectUncitedDeltas(tt.deltas, ownerMap(nil), tt.cited)

			if ids := uncitedIDs(got); !equalStrings(ids, tt.wantUncited) {
				t.Errorf("uncited = %v, want %v", ids, tt.wantUncited)
			}
			shas := make([]string, 0, len(got.Unattributed))
			for _, d := range got.Unattributed {
				shas = append(shas, d.SHA)
			}
			if !equalStrings(shas, tt.wantUnattributed) {
				t.Errorf("unattributed = %v, want %v", shas, tt.wantUnattributed)
			}
		})
	}
}

// TestUnattributedNotes_NameTheCommitAndItsSubject pins what an operator
// gets. The repair is to work out which entity the commit belonged to,
// which needs the subject — an id-less list of SHAs is a list nobody
// acts on.
func TestUnattributedNotes_NameTheCommitAndItsSubject(t *testing.T) {
	t.Parallel()

	a := changelogAudit{Unattributed: []changelogDelta{
		{SHA: "1234567", Subject: "docs(guidance): prime against enumeration"},
	}}
	notes := unattributedNotes(a)
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1", len(notes))
	}
	for _, want := range []string{"1234567", "prime against enumeration"} {
		if !strings.Contains(notes[0], want) {
			t.Errorf("note does not carry %q:\n%s", want, notes[0])
		}
	}
}

// TestUnattributedNotes_CleanAuditSaysNothing keeps the report quiet
// when there is nothing to say. A note emitted per run regardless would
// train the reader to skip the section that carries the real ones.
func TestUnattributedNotes_CleanAuditSaysNothing(t *testing.T) {
	t.Parallel()
	if notes := unattributedNotes(changelogAudit{}); len(notes) != 0 {
		t.Errorf("got %d notes from a clean audit, want 0: %v", len(notes), notes)
	}
}

// TestChangelogAudit_UntrailteredCommitReportsWithoutFailing is AC-2's
// seam, and it asserts both halves because each fails a different wrong
// implementation. An audit that drops the commit passes the exit-code
// half; one that counts it toward the verdict passes the reporting half.
// Only together do they pin "reported, not blocking" (D-0087).
func TestChangelogAudit_UntrailteredCommitReportsWithoutFailing(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, base := changelogFixture(t)

	writeFile(clShippedRel, "fictional content\n")
	runGit("add", "-A")
	runGit("commit", "-m", "docs(fictional): a delta with no provenance")

	audit, err := changelogAuditFor(root, base)
	if err != nil {
		t.Fatalf("changelogAuditFor: %v", err)
	}
	if len(audit.Unattributed) != 1 {
		t.Errorf("got %d unattributed commits, want 1 — an untrailered shipped change must not be dropped", len(audit.Unattributed))
	}

	// Through changelogViolations rather than uncitedViolations: the
	// former is what gates a release, and an implementation that leaked
	// the unattributed half into it would satisfy the latter untouched.
	vs, err := changelogViolations(root, base)
	if err != nil {
		t.Fatalf("changelogViolations: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("got %d release-failing violations, want 0 — the audit cannot attribute this commit, so it cannot say the changelog omits it: %v", len(vs), vs)
	}
}

// TestDetectUncitedDeltas_CarriesTheCommitsBehindTheEntity pins the half
// the id assertion cannot reach. An operator handed "E-0091 is not
// cited" has to find what shipped under it, so the finding carries the
// commits rather than only the id.
func TestDetectUncitedDeltas_CarriesTheCommitsBehindTheEntity(t *testing.T) {
	t.Parallel()

	deltas := []changelogDelta{
		{SHA: "1234567", Subject: "docs(guidance): a", Entity: "G-0659"},
		{SHA: "89abcde", Subject: "docs(guidance): b", Entity: "G-0659"},
	}
	got := detectUncitedDeltas(deltas, ownerMap(nil), citedSet())
	if len(got.Uncited) != 1 {
		t.Fatalf("got %d uncited findings, want 1", len(got.Uncited))
	}
	if len(got.Uncited[0].Commits) != 2 {
		t.Errorf("finding carries %d commits, want 2 — the operator needs what shipped, not only the id",
			len(got.Uncited[0].Commits))
	}
}

// TestUncitedViolation_DetailNamesTheEntityAndACommit pins the operator-
// facing half: the rendered Detail says which entity is uncited and
// names a commit to start from. This is also the firing fixture for the
// policy id, which the meta-gate requires be covered.
func TestUncitedViolation_DetailNamesTheEntityAndACommit(t *testing.T) {
	t.Parallel()

	a := detectUncitedDeltas(
		[]changelogDelta{{SHA: "1234567", Subject: "docs(guidance): prime against enumeration", Entity: "G-0659"}},
		ownerMap(nil),
		citedSet(),
	)
	vs := uncitedViolations(a)
	if len(vs) != 1 {
		t.Fatalf("got %d violations, want 1", len(vs))
	}
	for _, want := range []string{"G-0659", "1234567"} {
		if !strings.Contains(vs[0].Detail, want) {
			t.Errorf("Detail does not name %q:\n%s", want, vs[0].Detail)
		}
	}
}

// TestUncitedViolations_LongRangeIsCappedAndCounted pins both halves of
// the commit list. An epic can carry dozens of commits across a release,
// and a Detail listing all of them is one an operator scrolls past; a
// capped list that does not say how much it hid understates the work.
func TestUncitedViolations_LongRangeIsCappedAndCounted(t *testing.T) {
	t.Parallel()

	deltas := make([]changelogDelta, 0, summarizeCommitsCap+2)
	for i := range summarizeCommitsCap + 2 {
		deltas = append(deltas, changelogDelta{
			SHA:     fmt.Sprintf("sha%04d", i),
			Subject: fmt.Sprintf("docs(skills): change %d", i),
			Entity:  "E-0091",
		})
	}
	vs := uncitedViolations(detectUncitedDeltas(deltas, ownerMap(nil), citedSet()))
	if len(vs) != 1 {
		t.Fatalf("got %d violations, want 1", len(vs))
	}
	detail := vs[0].Detail

	if got := strings.Count(detail, "docs(skills): change "); got != summarizeCommitsCap {
		t.Errorf("Detail lists %d subjects, want %d:\n%s", got, summarizeCommitsCap, detail)
	}
	if want := "and 2 more"; !strings.Contains(detail, want) {
		t.Errorf("Detail does not say how many it hid (%q):\n%s", want, detail)
	}
}

// TestUncitedViolations_ShortListClaimsNothingHidden is the other side
// of the cap. The assertion above says the tail appears when commits are
// hidden; without this one, a rendering that appends the tail
// unconditionally — "and 0 more" on every short list — passes both.
func TestUncitedViolations_ShortListClaimsNothingHidden(t *testing.T) {
	t.Parallel()

	vs := uncitedViolations(detectUncitedDeltas(
		[]changelogDelta{{SHA: "1234567", Subject: "docs(guidance): x", Entity: "G-0659"}},
		ownerMap(nil), citedSet(),
	))
	if len(vs) != 1 {
		t.Fatalf("got %d violations, want 1", len(vs))
	}
	if strings.Contains(vs[0].Detail, "more") {
		t.Errorf("a list that hid nothing must not say it did:\n%s", vs[0].Detail)
	}
}

// TestChangelogOwnerFor_RollupIsMilestoneOnlyAndCanonicalWidth pins the
// two readings the live tree cannot produce. Every parent it carries is
// already at canonical width, so dropping the canonicalization changes
// nothing there; and its gaps carry no parent, so the kind test appears
// redundant when it is not — the loader clears only Area for a kind that
// does not carry it, never Parent, so a non-milestone with a parent is
// representable and would otherwise roll up to an epic that owes nothing
// on its behalf.
func TestChangelogOwnerFor_RollupIsMilestoneOnlyAndCanonicalWidth(t *testing.T) {
	t.Parallel()

	tr := &tree.Tree{Entities: []*entity.Entity{
		// `E-02` rather than `E-2`: an epic id is `E-\d{2,}`, so a
		// single digit is not a narrow id but an invalid one, which
		// Canonicalize passes through untouched by design.
		{ID: "M-0001", Kind: entity.KindMilestone, Parent: "E-02", Path: "work/epics/E-0002-x/M-0001-y.md"},
		{ID: "G-0001", Kind: entity.KindGap, Parent: "E-0002", Path: "work/gaps/G-0001-z.md"},
	}}
	owner := changelogOwnerFor(tr)

	tests := []struct {
		name string
		id   string
		want string
	}{
		{name: "a milestone's narrow parent rolls up at canonical width", id: "M-0001", want: "E-0002"},
		{name: "a gap carrying a parent still owes its own citation", id: "G-0001", want: "G-0001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := owner(tt.id); got != tt.want {
				t.Errorf("owner(%s) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

// TestUncitedViolations_CleanAuditYieldsNone confirms the release-gating
// path reports nothing when every entity is cited. The harness reads the
// slice's length as the verdict, so an audit that returned a placeholder
// here would fail every release.
func TestUncitedViolations_CleanAuditYieldsNone(t *testing.T) {
	t.Parallel()
	if vs := uncitedViolations(changelogAudit{}); len(vs) != 0 {
		t.Errorf("got %d violations from a clean audit, want 0: %v", len(vs), vs)
	}
}

// TestChangelogOwner_TreeLoadFailure confirms the rollup refuses rather
// than silently rolling nothing up. An unreadable tree makes every
// milestone look like its own owner, which reports a run of entities no
// changelog entry has ever carried — a wrong answer is worse here than
// no answer, because the operator would go and write those entries.
func TestChangelogOwner_TreeLoadFailure(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits do not deny the walk")
	}
	root := t.TempDir()
	denied := filepath.Join(root, "work", "epics")
	if err := os.MkdirAll(denied, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(denied, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(denied, 0o755) })

	owner, err := changelogOwner(root)
	if err == nil {
		t.Fatalf("want an error when the planning tree cannot be read, got nil (owner=%v)", owner != nil)
	}
	if !strings.Contains(err.Error(), "entity tree") {
		t.Errorf("error should name what could not be loaded; got %v", err)
	}
}

// clShippedRel is a fictional shipped-surface file under the embedded
// tree, and clUnshippedRel one outside it. Together they pin what the
// audit watches: a change to shipped content owes a citation, a change
// to the kernel's own Go code does not — that is what the release notes
// describe in prose rather than by entity.
const (
	clShippedRel   = "internal/skills/embedded/aiwf-fictional/SKILL.md"
	clUnshippedRel = "internal/check/fictional.go"
)

// changelogFixture writes a fixture repo carrying an entity the audit
// can resolve and a CHANGELOG whose `[Unreleased]` section names the ids
// given, then returns the repo root and the base ref to audit forward
// from.
func changelogFixture(t *testing.T, cites ...string) (root string, runGit func(...string) string, writeFile func(string, string), base string) {
	t.Helper()
	root, runGit, writeFile, _ = skillFixtureBase(t)
	writeFile(provFixtureEntityAt, provFixtureEntity)

	body := "# Changelog\n\n## [Unreleased]\n\n"
	for _, id := range cites {
		body += "### Changed — " + id + ": a fictional delta\n\nProse.\n\n"
	}
	body += "## [0.1.0] — 2026-01-01\n\n### Added — the first release\n"
	writeFile("CHANGELOG.md", body)

	runGit("add", "-A")
	runGit("commit", "-m", "seed the fixture tree")
	return root, runGit, writeFile, trimLine(runGit("rev-parse", "HEAD"))
}

// TestChangelogViolations_Seam drives the whole audit against a real
// repository, which is where the parts that cannot be unit-tested meet:
// the range scan, the section parse and the rollup.
func TestChangelogViolations_Seam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cites   []string
		path    string
		trailer string
		wantIDs []string
	}{
		{
			name:    "a shipped delta nothing cites is reported",
			path:    clShippedRel,
			trailer: provFixtureEntityID,
			wantIDs: []string{provFixtureEntityID},
		},
		{
			name:    "the same delta, cited, is silent",
			cites:   []string{provFixtureEntityID},
			path:    clShippedRel,
			trailer: provFixtureEntityID,
			wantIDs: nil,
		},
		{
			name:    "a change outside the shipped tree owes no citation",
			path:    clUnshippedRel,
			trailer: provFixtureEntityID,
			wantIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, runGit, writeFile, base := changelogFixture(t, tt.cites...)

			writeFile(tt.path, "fictional content\n")
			runGit("add", "-A")
			runGit("commit", "-m", "docs(fictional): a delta", "--trailer", "aiwf-entity: "+tt.trailer)

			vs, err := changelogViolations(root, base)
			if err != nil {
				t.Fatalf("changelogViolations: %v", err)
			}
			var got []string
			for _, v := range vs {
				got = append(got, strings.Fields(v.Detail)[0])
			}
			if !equalStrings(got, tt.wantIDs) {
				t.Errorf("reported %v, want %v (details: %v)", got, tt.wantIDs, vs)
			}
		})
	}
}

// TestChangelogViolations_GoUnderTheShippedTreeShipsNothing pins the
// exclusion the pathspec cannot express. `internal/skills` holds two
// unlike things: the embedded trees, whose bytes materialize into a
// consumer repo, and the Go that materializes them. Only the first is a
// shipped surface. Every `go:embed` in the package names a path under an
// `embedded*` directory and none of those directories holds a `.go`
// file, so excluding Go loses no shipped content.
//
// Both rows matter and the narrower one alone is not enough: excluding
// only `_test.go` leaves a change to the materializer demanding a
// changelog entry, which is how a release gets blocked over a rename in
// the kernel's own source. Measured over v0.20.0..HEAD, twelve commits
// touch only Go under this tree.
func TestChangelogViolations_GoUnderTheShippedTreeShipsNothing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{name: "a test file", path: changelogShippedDir + "/fictional_test.go"},
		{name: "the materializer itself", path: changelogShippedDir + "/fictional.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, runGit, writeFile, base := changelogFixture(t)

			writeFile(tt.path, "package skills\n")
			runGit("add", "-A")
			runGit("commit", "-m", "refactor(fictional): a Go-only change", "--trailer", "aiwf-entity: "+provFixtureEntityID)

			vs, err := changelogViolations(root, base)
			if err != nil {
				t.Fatalf("changelogViolations: %v", err)
			}
			if len(vs) != 0 {
				t.Errorf("a Go-only change under the shipped tree reported %d violations, want 0: %v", len(vs), vs)
			}
		})
	}
}

// TestChangelogViolations_BaseUnresolvable confirms the audit no-ops
// without a comparison point rather than auditing all of history. A
// brand-new branch's all-zero base and an unset environment both reach
// this path, and treating either as "audit everything" would report
// every entity the project has ever shipped.
func TestChangelogViolations_BaseUnresolvable(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := changelogFixture(t)
	writeFile(clShippedRel, "fictional content\n")
	runGit("add", "-A")
	runGit("commit", "-m", "docs(fictional): a delta", "--trailer", "aiwf-entity: "+provFixtureEntityID)

	for _, base := range []string{"", zeroSHA, "   "} {
		vs, err := changelogViolations(root, base)
		if err != nil {
			t.Fatalf("base %q: unexpected error: %v", base, err)
		}
		if len(vs) != 0 {
			t.Errorf("base %q: got %d violations, want 0", base, len(vs))
		}
	}
}

// TestChangelogViolations_MergeCommitContributesNothing pins the
// exclusion AC-1 names. A merge carries the merged branch's whole file
// list, already attributed to the commits that made it, so counting one
// attributes the same delta twice — and would attribute it to whatever
// entity the merge commit happened to name.
func TestChangelogViolations_MergeCommitContributesNothing(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, base := changelogFixture(t)

	runGit("checkout", "-b", "side")
	writeFile(clShippedRel, "fictional content\n")
	runGit("add", "-A")
	runGit("commit", "-m", "docs(fictional): a delta", "--trailer", "aiwf-entity: "+provFixtureEntityID)
	sideHead := trimLine(runGit("rev-parse", "HEAD"))

	runGit("checkout", "-")
	runGit("merge", "--no-ff", "--no-commit", "side")
	runGit("commit", "-m", "Merge side", "--trailer", "aiwf-entity: M-0001")

	vs, err := changelogViolations(root, sideHead)
	if err != nil {
		t.Fatalf("changelogViolations: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("a range holding only a merge reported %d violations, want 0: %v", len(vs), vs)
	}
	_ = base
}

// TestPolicy_ChangelogCompleteness is the release-gate entry point. It
// runs the audit against the live tree using the base supplied via the
// environment; without one it skips, because the authoritative
// invocation is the release target rather than the every-push suite.
//
// It renders both halves of the verdict from one audit rather than
// calling runPolicy, because the harness's shape carries only the
// blocking half. reportViolations fails the test once per uncited
// entity, which is how a finding reaches a non-zero exit; the
// unattributed commits are logged, so they reach the operator without
// stopping a release the audit cannot prove is incomplete (D-0087).
func TestPolicy_ChangelogCompleteness(t *testing.T) {
	t.Parallel()
	base := os.Getenv(changelogBaseEnv)
	if base == "" {
		t.Skip(changelogBaseEnv + " unset; run via `make changelog-audit` or the release-tag workflow step")
	}

	audit, err := changelogAuditFor(repoRoot(t), base)
	if err != nil {
		t.Fatalf("changelog audit returned error: %v", err)
	}
	for _, note := range unattributedNotes(audit) {
		t.Log("[changelog-completeness] " + note)
	}
	reportViolations(t, uncitedViolations(audit))
}

// TestPolicyChangelogCompleteness_Env drives the env-fed entry point so
// its body is exercised whichever way the suite is invoked. Serial —
// t.Setenv panics under t.Parallel — and recorded in setup_test.go's
// skip-list.
func TestPolicyChangelogCompleteness_Env(t *testing.T) {
	root := repoRoot(t)

	for _, base := range []string{"", zeroSHA} {
		t.Setenv(changelogBaseEnv, base)
		vs, err := PolicyChangelogCompleteness(root)
		if err != nil {
			t.Fatalf("base %q: unexpected error: %v", base, err)
		}
		if len(vs) != 0 {
			t.Errorf("base %q: got %d violations, want 0 without a comparison point", base, len(vs))
		}
	}
}

// TestChangelogViolations_Errors covers the paths where the audit
// cannot answer its own question. Each returns an error rather than an
// empty result, because an empty result here reads as "everything is
// cited" — the audit would report a clean release precisely when it had
// failed to look.
func TestChangelogViolations_Errors(t *testing.T) {
	t.Parallel()

	t.Run("an unresolvable base ref", func(t *testing.T) {
		t.Parallel()
		root, _, _, _ := changelogFixture(t)

		_, err := changelogViolations(root, "refs/tags/v-does-not-exist")
		if err == nil {
			t.Fatal("want an error for a base ref that resolves to nothing, got nil")
		}
		if !strings.Contains(err.Error(), "git log") {
			t.Errorf("error should name the command that failed; got %v", err)
		}
	})

	t.Run("no changelog to read", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, _ := skillFixtureBase(t)
		writeFile(provFixtureEntityAt, provFixtureEntity)
		runGit("add", "-A")
		runGit("commit", "-m", "seed the fixture tree")
		base := trimLine(runGit("rev-parse", "HEAD"))

		writeFile(clShippedRel, "fictional content\n")
		runGit("add", "-A")
		runGit("commit", "-m", "docs(fictional): a delta", "--trailer", "aiwf-entity: "+provFixtureEntityID)

		_, err := changelogViolations(root, base)
		if err == nil {
			t.Fatal("want an error when there is no changelog to compare against, got nil")
		}
		if !strings.Contains(err.Error(), changelogFile) {
			t.Errorf("error should name the file it could not read; got %v", err)
		}
	})

	t.Run("an unreadable planning tree", func(t *testing.T) {
		t.Parallel()
		if os.Geteuid() == 0 {
			t.Skip("running as root; permission bits do not deny the walk")
		}
		root, runGit, writeFile, base := changelogFixture(t)
		writeFile(clShippedRel, "fictional content\n")
		runGit("add", "-A")
		runGit("commit", "-m", "docs(fictional): a delta", "--trailer", "aiwf-entity: "+provFixtureEntityID)

		denied := filepath.Join(root, "work", "epics")
		if err := os.Chmod(denied, 0o000); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(denied, 0o755) })

		_, err := changelogViolations(root, base)
		if err == nil {
			t.Fatal("want an error when the rollup cannot be built, got nil")
		}
		if !strings.Contains(err.Error(), "entity tree") {
			t.Errorf("error should name what could not be loaded; got %v", err)
		}
	})
}

// TestResolveChangelogBase_ReachableNotNewest is AC-3's rule. The audit
// runs on a branch in practice — the release gate fires before the merge
// — and on trunk the newest tag and the newest reachable tag are usually
// the same commit. A resolution that sorts the tag list instead of
// walking history therefore passes every trunk-only test and compares a
// branch against a release that never contained it.
//
// The fixture puts a *higher-versioned* tag on a branch the commit under
// test cannot reach, so version order and reachability disagree and only
// the reachable answer is correct.
func TestResolveChangelogBase_ReachableNotNewest(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := skillFixtureBase(t)
	commitAt := datedCommitter(t, root)

	runGit("tag", "v0.1.0")

	// The unreachable tag is made unambiguously the newest, so that
	// reachability is the only thing separating the right answer from
	// the wrong one. Left to the clock, every fixture commit lands in
	// the same second and a date-ordered resolver returns the right tag
	// on a tie — measured, that is exactly what happens, and it makes
	// the wrong implementation look correct. The far-future date is what
	// removes the tie without depending on when the test runs.
	writeFile("main.txt", "main\n")
	runGit("add", "-A")
	commitAt("main work", "2026-01-01T00:00:00+00:00")
	mainHead := trimLine(runGit("rev-parse", "HEAD"))

	runGit("checkout", "-b", "side", "v0.1.0")
	writeFile("side.txt", "side\n")
	runGit("add", "-A")
	commitAt("side work", "2099-01-01T00:00:00+00:00")
	runGit("tag", "v0.9.0")
	runGit("checkout", mainHead)

	got, err := resolveChangelogBase(root)
	if err != nil {
		t.Fatalf("resolveChangelogBase: %v", err)
	}
	if got != "v0.1.0" {
		t.Errorf("resolveChangelogBase = %q, want %q — v0.9.0 is both newer and higher-versioned, and unreachable from this commit", got, "v0.1.0")
	}

	// State the property directly too, so a future resolver returning
	// some other reachable ref is judged on reachability rather than on
	// matching this fixture's tag name.
	anc := exec.Command("git", "merge-base", "--is-ancestor", got, "HEAD")
	anc.Dir = root
	if err := anc.Run(); err != nil {
		t.Errorf("resolved base %q is not an ancestor of HEAD: %v", got, err)
	}
}

// datedCommitter returns a commit closure that fixes both git dates, so
// tag ordering in a fixture is decided by the test rather than by how
// fast it happens to run.
func datedCommitter(t *testing.T, root string) func(msg, date string) {
	t.Helper()
	return func(msg, date string) {
		t.Helper()
		cmd := exec.Command("git", "commit", "-m", msg)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_DATE="+date,
			"GIT_COMMITTER_DATE="+date,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git commit %q at %s: %v\n%s", msg, date, err, out)
		}
	}
}

// TestResolveChangelogBase_NoTagYetFallsBackToTheRootCommit covers the
// first release, which has no predecessor to compare against. Failing
// there would make the audit unrunnable until the first tag exists,
// which is exactly the release that most needs its notes checked.
func TestResolveChangelogBase_NoTagYetFallsBackToTheRootCommit(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := skillFixtureBase(t)
	writeFile("more.txt", "more\n")
	runGit("add", "-A")
	runGit("commit", "-m", "second")

	got, err := resolveChangelogBase(root)
	if err != nil {
		t.Fatalf("resolveChangelogBase: %v", err)
	}
	wantRoot := trimLine(runGit("rev-list", "--max-parents=0", "HEAD"))
	if got != wantRoot {
		t.Errorf("resolveChangelogBase = %q, want the root commit %q", got, wantRoot)
	}
}

// TestResolveChangelogBase_GraftedHistoryTakesTheEarliestRoot pins the
// choice the single-root fallback cannot show. A repo built by grafting
// two histories has more than one root reachable from HEAD, and taking
// the wrong one silently narrows the audited range to the grafted-in
// side — every commit before the graft would go unexamined.
func TestResolveChangelogBase_GraftedHistoryTakesTheEarliestRoot(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := skillFixtureBase(t)
	commitAt := datedCommitter(t, root)

	writeFile("old.txt", "old\n")
	runGit("add", "-A")
	commitAt("older history", "2020-01-01T00:00:00+00:00")
	mainHead := trimLine(runGit("rev-parse", "HEAD"))

	runGit("checkout", "--orphan", "grafted")
	runGit("rm", "-rf", "--cached", ".")
	writeFile("new.txt", "new\n")
	runGit("add", "-A")
	commitAt("newer unrelated history", "2030-01-01T00:00:00+00:00")

	runGit("checkout", mainHead)
	runGit("merge", "--allow-unrelated-histories", "--no-edit", "grafted")

	roots := strings.Fields(runGit("rev-list", "--max-parents=0", "HEAD"))
	if len(roots) != 2 {
		t.Fatalf("fixture has %d roots, want 2 — the test proves nothing with one", len(roots))
	}
	want := roots[len(roots)-1]

	got, err := resolveChangelogBase(root)
	if err != nil {
		t.Fatalf("resolveChangelogBase: %v", err)
	}
	if got != want {
		t.Errorf("resolveChangelogBase = %q, want the earliest root %q (the other is %q)", got, want, roots[0])
	}
}

// TestResolveChangelogBase_NoHistoryToResolveFrom covers the path where
// neither a tag nor a root commit exists. Returning some plausible-
// looking ref here would audit a range nobody chose; the error says the
// base could not be worked out, which is the only honest answer.
func TestResolveChangelogBase_NoHistoryToResolveFrom(t *testing.T) {
	t.Parallel()

	_, err := resolveChangelogBase(t.TempDir())
	if err == nil {
		t.Fatal("want an error resolving a base outside a repository, got nil")
	}
	if !strings.Contains(err.Error(), "no reachable tag") {
		t.Errorf("error should say what it could not find; got %v", err)
	}
}

// TestChangelogAuditFor_AutoPropagatesAResolverFailure confirms the
// sentinel path surfaces a resolution failure rather than falling
// through to an audit of some other range. An audit that quietly picked
// a different base would report findings against commits the operator
// never asked about.
func TestChangelogAuditFor_AutoPropagatesAResolverFailure(t *testing.T) {
	t.Parallel()

	_, err := changelogAuditFor(t.TempDir(), changelogBaseAuto)
	if err == nil {
		t.Fatal("want an error when the base cannot be resolved, got nil")
	}
	if !strings.Contains(err.Error(), "no reachable tag") {
		t.Errorf("error should carry the resolver's cause; got %v", err)
	}
}

// TestChangelogAuditFor_AutoResolvesTheBase pins the seam: the sentinel
// reaches the resolver rather than being handed to git as a ref named
// "auto", which would fail rather than resolve.
func TestChangelogAuditFor_AutoResolvesTheBase(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := changelogFixture(t)
	runGit("tag", "v0.1.0")

	writeFile(clShippedRel, "fictional content\n")
	runGit("add", "-A")
	runGit("commit", "-m", "docs(fictional): a delta", "--trailer", "aiwf-entity: "+provFixtureEntityID)

	audit, err := changelogAuditFor(root, changelogBaseAuto)
	if err != nil {
		t.Fatalf("changelogAuditFor(auto): %v", err)
	}
	if ids := uncitedIDs(audit); !equalStrings(ids, []string{provFixtureEntityID}) {
		t.Errorf("auto-resolved audit reported %v, want %v", ids, []string{provFixtureEntityID})
	}
}

// TestUnreleasedSection covers the rules that decide what counts as the
// section: it starts at the Unreleased heading and stops at the next
// release heading, so a citation in an already-shipped section does not
// satisfy a delta that has not shipped yet.
func TestUnreleasedSection(t *testing.T) {
	t.Parallel()

	const doc = "# Changelog\n\n" +
		"## [Unreleased]\n\n### Changed — E-0001: a\n\n" +
		"## [0.2.0] — 2026-02-02\n\n### Changed — E-0002: b\n"

	tests := []struct {
		name     string
		doc      string
		cited    string
		wantHeld bool
	}{
		{name: "an id under Unreleased is held", doc: doc, cited: "E-0001", wantHeld: true},
		{name: "an id under a shipped release is not", doc: doc, cited: "E-0002", wantHeld: false},
		{name: "a file with no Unreleased section holds nothing", doc: "# Changelog\n\n## [0.1.0]\n\n### Added — E-0003: c\n", cited: "E-0003", wantHeld: false},
		{name: "a legacy width in the entry names the same entity", doc: "# Changelog\n\n## [Unreleased]\n\n### Changed — E-01: a\n", cited: "E-0001", wantHeld: true},
		// A milestone id is `M-\d{3,}`, so `M-7` is not a narrow id but
		// an unrecognized one. It matches its own spelling and unifies
		// with no width, which is what keeps a malformed trailer from
		// being satisfied by an entry naming a real entity.
		{name: "an id below the kind's floor matches its own spelling", doc: "# Changelog\n\n## [Unreleased]\n\n### Changed — M-7: a\n", cited: "M-7", wantHeld: true},
		{name: "an id below the floor unifies with no other width", doc: "# Changelog\n\n## [Unreleased]\n\n### Changed — M-7: a\n", cited: "M-0007", wantHeld: false},
		{name: "an empty id is cited by nothing", doc: doc, cited: "", wantHeld: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cited := changelogCitedIn(unreleasedSection(tt.doc))
			if got := cited(tt.cited); got != tt.wantHeld {
				t.Errorf("cited(%s) = %v, want %v", tt.cited, got, tt.wantHeld)
			}
		})
	}
}

// TestChangelogOwner_ArchivedMilestoneRollsUpToItsEpic pins the rollup
// against the live tree, which is the only place the path-scan trap
// shows. A glob over the active epic directories misses an archived
// milestone, and its id then survives the rollup and reports as an
// entity owing a citation in its own right — a false report naming an id
// no changelog entry was ever expected to carry.
//
// The expectation is read from the loader rather than written as a pair,
// so a reallocate or an archive sweep moves both sides together.
func TestChangelogOwner_ArchivedMilestoneRollsUpToItsEpic(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)

	var mid, want string
	for _, e := range tr.Entities {
		if e.Kind == entity.KindMilestone && entity.IsArchivedPath(e.Path) && e.Parent != "" {
			mid, want = e.ID, entity.Canonicalize(e.Parent)
			break
		}
	}
	if mid == "" {
		t.Fatal("live tree carries no archived milestone with a parent; this test proves nothing without one")
	}

	owner, err := changelogOwner(root)
	if err != nil {
		t.Fatalf("changelogOwner(%s): %v", root, err)
	}
	if got := owner(mid); got != want {
		t.Errorf("owner(%s) = %q, want %q — an archived milestone must roll up to its parent epic", mid, got, want)
	}
}

// TestChangelogOwnerFor_NoRollupToNameLeavesTheIDAlone covers the two
// readings where a rollup has nowhere to go. A trailer naming an id the
// tree does not carry is stale provenance, and a milestone with no
// parent is a hand-edited file `aiwf check` reports on its own; neither
// is this audit's to diagnose, and reporting an empty owner would name
// no entity at all.
func TestChangelogOwnerFor_NoRollupToNameLeavesTheIDAlone(t *testing.T) {
	t.Parallel()

	tr := &tree.Tree{Entities: []*entity.Entity{
		{ID: "M-0001", Kind: entity.KindMilestone, Path: "work/epics/E-0001-x/M-0001-y.md"},
	}}
	owner := changelogOwnerFor(tr)

	tests := []struct {
		name string
		id   string
		want string
	}{
		{name: "an id the tree does not carry", id: "M-0002", want: "M-0002"},
		{name: "a milestone with no parent", id: "M-0001", want: "M-0001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := owner(tt.id); got != tt.want {
				t.Errorf("owner(%s) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

// TestChangelogOwner_ANonMilestoneOwesItsOwnCitation confirms the rollup
// applies to milestones alone. A gap, decision or ADR is cited by its
// own id, so an owner that rolled everything up would name the wrong
// entity for the majority of the entries this file carries.
func TestChangelogOwner_ANonMilestoneOwesItsOwnCitation(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)

	var gid string
	for _, e := range tr.Entities {
		if e.Kind == entity.KindGap {
			gid = e.ID
			break
		}
	}
	if gid == "" {
		t.Fatal("live tree carries no gap; this test proves nothing without one")
	}

	owner, err := changelogOwner(root)
	if err != nil {
		t.Fatalf("changelogOwner(%s): %v", root, err)
	}
	if got := owner(gid); got != gid {
		t.Errorf("owner(%s) = %q, want %q — only a milestone rolls up", gid, got, gid)
	}
}
