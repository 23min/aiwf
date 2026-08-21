package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Synthetic skill paths for the pure-detector table, using an
// obviously-fictional skill name.
const (
	provSkillA = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-fictional-charlie/SKILL.md"
	provSkillB = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-fictional-delta/SKILL.md"
)

// resolvesOnly returns a resolver that recognizes exactly the supplied
// ids — the injected stand-in for a loaded entity tree, so the detector's
// three arms are decidable without one.
func resolvesOnly(ids ...string) func(string) bool {
	known := make(map[string]bool, len(ids))
	for _, id := range ids {
		known[id] = true
	}
	return func(id string) bool { return known[id] }
}

// TestDetectUnownedSkillEdits drives the pure core across all three
// provenance arms: an edit with no owning entity fires (AC-1), an edit
// whose aiwf-entity resolves is silent (AC-2), and an edit naming an id
// that resolves to nothing fires (AC-3).
func TestDetectUnownedSkillEdits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		edits     []skillEdit
		known     []string
		wantFiles []string
	}{
		{
			name:      "edit with no aiwf-entity trailer fires (AC-1)",
			edits:     []skillEdit{{SHA: "aaaaaaa", Path: provSkillA, Entity: ""}},
			known:     []string{"M-0312"},
			wantFiles: []string{provSkillA},
		},
		{
			name:      "edit whose aiwf-entity resolves is silent (AC-2)",
			edits:     []skillEdit{{SHA: "bbbbbbb", Path: provSkillA, Entity: "M-0312"}},
			known:     []string{"M-0312"},
			wantFiles: nil,
		},
		{
			name:      "edit naming an unresolvable entity fires (AC-3)",
			edits:     []skillEdit{{SHA: "ccccccc", Path: provSkillA, Entity: "M-9999"}},
			known:     []string{"M-0312"},
			wantFiles: []string{provSkillA},
		},
		{
			name: "mixed input fires only for the unowned paths",
			edits: []skillEdit{
				{SHA: "ddddddd", Path: provSkillA, Entity: "M-0312"},
				{SHA: "eeeeeee", Path: provSkillB, Entity: ""},
			},
			known:     []string{"M-0312"},
			wantFiles: []string{provSkillB},
		},
		{
			name:      "a composite id resolves through its parent milestone",
			edits:     []skillEdit{{SHA: "fffffff", Path: provSkillA, Entity: "M-0312/AC-1"}},
			known:     []string{"M-0312"},
			wantFiles: nil,
		},
		{
			name:      "a narrow legacy id names the same entity",
			edits:     []skillEdit{{SHA: "9999999", Path: provSkillA, Entity: "M-312"}},
			known:     []string{"M-0312"},
			wantFiles: nil,
		},
		{
			name:      "no changed skills is silent",
			edits:     nil,
			known:     []string{"M-0312"},
			wantFiles: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := detectUnownedSkillEdits(tt.edits, resolvesOnly(tt.known...))
			if !equalStrings(violationFiles(got), tt.wantFiles) {
				t.Errorf("violation files = %v, want %v", violationFiles(got), tt.wantFiles)
			}
			for _, v := range got {
				if v.Policy != "skill-edit-provenance-backstop" {
					t.Errorf("violation Policy = %q, want skill-edit-provenance-backstop", v.Policy)
				}
			}
		})
	}
}

// TestDetectUnownedSkillEdits_DetailNamesTheCause pins that each arm's
// Detail states which fault it found, so the operator learns whether the
// commit carried no owning entity at all or named one that resolves to
// nothing — two different repairs.
func TestDetectUnownedSkillEdits_DetailNamesTheCause(t *testing.T) {
	t.Parallel()

	missing := detectUnownedSkillEdits(
		[]skillEdit{{SHA: "aaaaaaa", Path: provSkillA}},
		resolvesOnly("M-0312"),
	)
	if len(missing) != 1 {
		t.Fatalf("want 1 violation for the trailerless edit, got %d", len(missing))
	}
	if !strings.Contains(missing[0].Detail, "no aiwf-entity: trailer") {
		t.Errorf("missing-trailer Detail must say the trailer is absent; got %q", missing[0].Detail)
	}

	unresolvable := detectUnownedSkillEdits(
		[]skillEdit{{SHA: "ccccccc", Path: provSkillA, Entity: "M-9999"}},
		resolvesOnly("M-0312"),
	)
	if len(unresolvable) != 1 {
		t.Fatalf("want 1 violation for the unresolvable edit, got %d", len(unresolvable))
	}
	if !strings.Contains(unresolvable[0].Detail, "M-9999") ||
		!strings.Contains(unresolvable[0].Detail, "resolves to no entity") {
		t.Errorf("unresolvable Detail must name the id and say it resolves to nothing; got %q", unresolvable[0].Detail)
	}
	if missing[0].Detail == unresolvable[0].Detail {
		t.Error("the two arms must not share one Detail; they name different repairs")
	}
}

// provSkillRel is the fictional embedded-rituals SKILL.md path the seam
// fixtures add.
const provSkillRel = skillRitualsDir + "/plugins/aiwf-extensions/skills/aiwfx-fictional-echo/SKILL.md"

// provFixtureEntity seeds a fictional epic into the fixture repo's
// planning tree so an aiwf-entity trailer naming it resolves.
const (
	provFixtureEntityID = "E-0001"
	provFixtureEntity   = "---\nid: E-0001\ntitle: Fictional fixture epic\nstatus: proposed\n---\n## Goal\n\nFixture.\n"
	provFixtureEntityAt = "work/epics/E-0001-fictional-fixture-epic/epic.md"
)

// TestSkillEditProvenance_Seam drives the IO shell end to end against a
// synthetic repo. The three arms are the three ACs, and each fixture
// repo's internal/policies/ carries a test file that references no skill
// path at all — so AC-2's silence is established under the
// no-referencing-test condition specifically, by construction, rather
// than by happening to run in a tree where something names the path.
func TestSkillEditProvenance_Seam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		trailer   string // aiwf-entity value; empty means commit with no trailer
		wantFiles []string
	}{
		{
			name:      "no aiwf-entity trailer fires (AC-1)",
			trailer:   "",
			wantFiles: []string{provSkillRel},
		},
		{
			name:      "resolving aiwf-entity is silent with no test naming the path (AC-2)",
			trailer:   provFixtureEntityID,
			wantFiles: nil,
		},
		{
			name:      "unresolvable aiwf-entity fires (AC-3)",
			trailer:   "M-9999",
			wantFiles: []string{provSkillRel},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, runGit, writeFile, _ := skillFixtureBase(t)
			writeFile(provFixtureEntityAt, provFixtureEntity)
			writeFile("internal/policies/seam_test.go", "package policies\n\n// references no skill path\n")
			runGit("add", "-A")
			runGit("commit", "-m", "seed the fixture tree")
			baseSHA := trimLine(runGit("rev-parse", "HEAD"))

			writeFile(provSkillRel, "# fictional echo skill\n\nprescriptive content\n")
			runGit("add", "-A")
			if tt.trailer == "" {
				runGit("commit", "-m", "feat(skill): edit a shipped surface")
			} else {
				runGit("commit", "-m", "feat(skill): edit a shipped surface",
					"--trailer", "aiwf-entity: "+tt.trailer)
			}

			vs, err := skillEditProvenanceViolations(root, baseSHA)
			if err != nil {
				t.Fatalf("skillEditProvenanceViolations: %v", err)
			}
			if got := violationFiles(vs); !equalStrings(got, tt.wantFiles) {
				t.Errorf("violation files = %v, want %v", got, tt.wantFiles)
			}
		})
	}
}

// violationFiles returns the File field of each violation, preserving
// order (the detector emits in the caller-supplied order, which
// skillEditsInRange sorts).
func violationFiles(vs []Violation) []string {
	if len(vs) == 0 {
		return nil
	}
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.File)
	}
	return out
}

func equalStrings(a, b []string) bool {
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

// skillFixtureBase inits a throwaway git repo with a base commit and
// returns the root, a git runner, a file writer, and the base SHA. The
// HEAD mutation is the caller's to stage and commit.
func skillFixtureBase(t *testing.T) (root string, runGit func(...string) string, writeFile func(string, string), baseSHA string) {
	t.Helper()
	root = t.TempDir()
	runGit = repoGitRunner(t, root)
	writeFile = repoFileWriter(t, root)
	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "aiwf-test")
	writeFile("go.mod", "module example.com/seam\n\ngo 1.24\n")
	writeFile("README.md", "base\n")
	runGit("add", "-A")
	runGit("commit", "-m", "base")
	baseSHA = trimLine(runGit("rev-parse", "HEAD"))
	return root, runGit, writeFile, baseSHA
}

// coverageGateRunLine returns the single line in content that invokes the
// profile-driven gates via `go test -run '^TestPolicy_(...)$'`. Scoping
// an assertion to that exact line (rather than a flat file-wide
// substring) keeps it structural: it pins the run-set, not an incidental
// mention of a test name elsewhere in the file.
func coverageGateRunLine(t *testing.T, fname, content string) string {
	t.Helper()
	var found []string
	for _, ln := range strings.Split(content, "\n") {
		if strings.Contains(ln, "-run '^TestPolicy_(") {
			found = append(found, strings.TrimSpace(ln))
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s: want exactly one coverage-gate run-pattern line, found %d: %v", fname, len(found), found)
	}
	return found[0]
}

// TestSkillEditProvenanceViolations_BaseUnresolvable confirms the gate
// no-ops on an empty or all-zero base ref — the broad `go test ./...`
// job and a brand-new branch's all-zero github.event.before both hit
// this path.
func TestSkillEditProvenanceViolations_BaseUnresolvable(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := skillFixtureBase(t)
	writeFile(provSkillRel, "# x\n")
	runGit("add", "-A")
	runGit("commit", "-m", "head")

	for _, base := range []string{"", zeroSHA, "   "} {
		vs, err := skillEditProvenanceViolations(root, base)
		if err != nil {
			t.Fatalf("base %q: unexpected error: %v", base, err)
		}
		if len(vs) != 0 {
			t.Errorf("base %q: got %d violations, want 0", base, len(vs))
		}
	}
}

// TestSkillEditProvenanceViolations_Errors exercises the IO core's
// error and early-return branches: an unresolvable base ref (git log
// fails) and a range touching no watched surface (the len==0
// short-circuit, which returns before the tree is loaded).
func TestSkillEditProvenanceViolations_Errors(t *testing.T) {
	t.Parallel()

	t.Run("git log error on bad base ref", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, _ := skillFixtureBase(t)
		writeFile(provSkillRel, "# x\n")
		runGit("add", "-A")
		runGit("commit", "-m", "head")
		_, err := skillEditProvenanceViolations(root, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
		if err == nil {
			t.Fatal("want error for nonexistent base ref, got nil")
		}
	})

	t.Run("no watched edit in range is silent", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, baseSHA := skillFixtureBase(t)
		writeFile("README.md", "base\nmore\n")
		runGit("add", "-A")
		runGit("commit", "-m", "head")
		vs, err := skillEditProvenanceViolations(root, baseSHA)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vs) != 0 {
			t.Errorf("want 0 violations, got %+v", vs)
		}
	})
}

// TestSkillEditProvenance_MergeCommitContributesNothing pins that an
// ordinary merge is not judged: git emits no diff for one without an
// explicit --diff-merges, and the commit that actually carries the edit
// is in the range on its own with its own trailer.
func TestSkillEditProvenance_MergeCommitContributesNothing(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := skillFixtureBase(t)
	writeFile(provFixtureEntityAt, provFixtureEntity)
	runGit("add", "-A")
	runGit("commit", "-m", "seed the fixture tree")
	baseSHA := trimLine(runGit("rev-parse", "HEAD"))
	trunk := trimLine(runGit("rev-parse", "--abbrev-ref", "HEAD"))

	runGit("checkout", "-b", "side")
	writeFile(provSkillRel, "# fictional echo skill\n")
	runGit("add", "-A")
	runGit("commit", "-m", "feat(skill): owned edit", "--trailer", "aiwf-entity: "+provFixtureEntityID)
	runGit("checkout", trunk)
	runGit("merge", "--no-ff", "-m", "Merge side", "side")

	vs, err := skillEditProvenanceViolations(root, baseSHA)
	if err != nil {
		t.Fatalf("skillEditProvenanceViolations: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("a merge of an owned edit must be silent; got %+v", vs)
	}
}

// TestSkillEditProvenance_DocumentedInClaudeMd (AC-4) is an absence
// assertion: no section of CLAUDE.md may state that an embedded-rituals
// SKILL.md edit needs a referencing structural test under
// internal/policies/. It is a ban, so it pins no wording — it fires only
// if the retired mandate is re-added.
//
// The two named sections are asserted to exist first. Without that, the
// absence would pass trivially against a CLAUDE.md that had lost the
// section entirely.
func TestSkillEditProvenance_DocumentedInClaudeMd(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	content := string(data)

	// Signatures of the retired content mandate. Each names the old
	// predicate rather than paraphrasing the prose around it.
	retired := []string{
		"structural test",
		"structural-test",
		"skill-edit-structural-test-backstop",
		"skill_edit_structural_test_backstop",
	}

	for _, heading := range []string{"## Ritual content authoring", "### What's enforced and where"} {
		section := markdownSection(content, heading)
		if section == "" {
			t.Fatalf("CLAUDE.md has no %q section — the absence assertion below would be vacuous", heading)
		}
		for _, sig := range retired {
			if strings.Contains(section, sig) {
				t.Errorf("%q must not state the retired content mandate, but contains %q (D-0071)", heading, sig)
			}
		}
	}
}

// TestSkillEditProvenanceBackstop_WiredIntoCoverageGate pins that the
// gate actually runs at the integration boundary: the policy test is
// named in the coverage-gate run-pattern of both the CI workflow and the
// Makefile target. Without this a future edit could drop the gate from
// the run set and it would silently never fire.
func TestSkillEditProvenanceBackstop_WiredIntoCoverageGate(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	const testName = "SkillEditProvenanceBackstop"

	for _, f := range []string{".github/workflows/go.yml", "Makefile"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		line := coverageGateRunLine(t, f, string(data))
		if !strings.Contains(line, testName) {
			t.Errorf("%s: coverage-gate run-pattern does not include %s:\n  %s", f, testName, line)
		}
	}
}

// TestPolicy_SkillEditProvenanceBackstop is the CI gate entry point. It
// runs the diff-scoped backstop against the live tree using the base ref
// supplied via AIWF_COVERAGE_BASE. Without a base (the default in the
// broad `go test ./...` job) it skips — the authoritative invocations
// are the dedicated CI coverage-gate step and `make coverage-gate`.
func TestPolicy_SkillEditProvenanceBackstop(t *testing.T) {
	t.Parallel()
	if os.Getenv("AIWF_COVERAGE_BASE") == "" {
		t.Skip("AIWF_COVERAGE_BASE unset; run via `make coverage-gate` or the CI coverage-gate step")
	}
	runPolicy(t, PolicySkillEditProvenanceBackstop)
}

// TestParseSkillEditLog_OrdersByPathThenSHA drives the log parser
// directly. It covers the multi-record path and the comparator's two
// arms: different paths sort by path, and two commits touching the same
// path sort by SHA so the output is stable.
func TestParseSkillEditLog_OrdersByPathThenSHA(t *testing.T) {
	t.Parallel()

	rec := func(sha, ent string, paths ...string) string {
		return skillEditRecSep + sha + skillEditFldSep + ent + "\n\n" + strings.Join(paths, "\n") + "\n"
	}
	// Deliberately out of order on both keys.
	log := rec("bbb2222", "M-0002", provSkillB) +
		rec("aaa1111", "M-0001", provSkillA) +
		rec("ccc3333", "M-0003", provSkillA)

	got := parseSkillEditLog(log)
	want := []skillEdit{
		{SHA: "aaa1111", Path: provSkillA, Entity: "M-0001"},
		{SHA: "ccc3333", Path: provSkillA, Entity: "M-0003"},
		{SHA: "bbb2222", Path: provSkillB, Entity: "M-0002"},
	}
	if len(got) != len(want) {
		t.Fatalf("parsed %d edits, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("edit %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestParseSkillEditLog_SkipsNonSkillAndEmptyRecords covers the two
// continue arms: a touched path that is not a SKILL.md, and an empty
// record (the leading split fragment before the first separator).
func TestParseSkillEditLog_SkipsNonSkillAndEmptyRecords(t *testing.T) {
	t.Parallel()

	log := "\n" + // empty leading fragment
		skillEditRecSep + "aaa1111" + skillEditFldSep + "M-0001" + "\n\n" +
		"internal/skills/embedded-rituals/plugins/p/skills/s/README.md\n" +
		provSkillA + "\n"

	got := parseSkillEditLog(log)
	if len(got) != 1 || got[0].Path != provSkillA {
		t.Fatalf("want only the SKILL.md path, got %+v", got)
	}
}

// TestSkillEditProvenanceViolations_TreeLoadFailure covers the resolver
// error path: an unreadable planning tree cannot answer the resolution
// question, and the gate surfaces that rather than reporting green.
func TestSkillEditProvenanceViolations_TreeLoadFailure(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits do not deny the walk")
	}
	root, runGit, writeFile, _ := skillFixtureBase(t)
	writeFile(provFixtureEntityAt, provFixtureEntity)
	runGit("add", "-A")
	runGit("commit", "-m", "seed the fixture tree")
	baseSHA := trimLine(runGit("rev-parse", "HEAD"))

	writeFile(provSkillRel, "# fictional echo skill\n")
	runGit("add", "-A")
	runGit("commit", "-m", "feat(skill): edit", "--trailer", "aiwf-entity: "+provFixtureEntityID)

	// Deny the walk of the epics directory the loader must read.
	denied := filepath.Join(root, "work", "epics")
	if err := os.Chmod(denied, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(denied, 0o755) })

	_, err := skillEditProvenanceViolations(root, baseSHA)
	if err == nil {
		t.Fatal("want an error when the planning tree cannot be read, got nil")
	}
	if !strings.Contains(err.Error(), "entity tree") {
		t.Errorf("error should name what could not be loaded; got %v", err)
	}
}

// TestPolicySkillEditProvenanceBackstop_Env drives the env-fed entry
// point so the wrapper body is exercised. Serial (t.Setenv panics under
// t.Parallel) and documented in setup_test.go's skip-list.
func TestPolicySkillEditProvenanceBackstop_Env(t *testing.T) {
	// Unset base → no-op.
	t.Setenv("AIWF_COVERAGE_BASE", "")
	vs, err := PolicySkillEditProvenanceBackstop(t.TempDir())
	if err != nil {
		t.Fatalf("unset base: unexpected error: %v", err)
	}
	if vs != nil {
		t.Fatalf("unset base: want nil violations, got %+v", vs)
	}

	// Set base → delegates and surfaces the unowned skill edit.
	root, runGit, writeFile, baseSHA := skillFixtureBase(t)
	writeFile(provSkillRel, "# fictional echo skill\n")
	runGit("add", "-A")
	runGit("commit", "-m", "feat(skill): edit with no owner")

	t.Setenv("AIWF_COVERAGE_BASE", baseSHA)
	vs, err = PolicySkillEditProvenanceBackstop(root)
	if err != nil {
		t.Fatalf("set base: unexpected error: %v", err)
	}
	if len(vs) != 1 || vs[0].File != provSkillRel {
		t.Fatalf("set base: want one violation for %s, got %+v", provSkillRel, vs)
	}
}

// TestSkillEditProvenance_RenameIsWatched pins that a rename does not
// escape the gate. Git reports a sufficiently similar rename as one R
// entry rather than a delete plus an add, so a filter admitting only A
// and M would let a commit that moves a skill and rewrites part of it
// through with no owner named. The fixture is deliberately over the
// similarity threshold — a low-similarity move degrades to delete+add
// and would pass even under the narrower filter, proving nothing.
func TestSkillEditProvenance_RenameIsWatched(t *testing.T) {
	t.Parallel()
	const from = skillRitualsDir + "/plugins/aiwf-extensions/skills/aiwfx-fictional-foxtrot/SKILL.md"
	const to = skillRitualsDir + "/plugins/aiwf-extensions/skills/aiwfx-fictional-golf/SKILL.md"

	body := strings.Repeat("prescriptive line\n", 60)
	root, runGit, writeFile, _ := skillFixtureBase(t)
	writeFile(provFixtureEntityAt, provFixtureEntity)
	writeFile(from, body)
	runGit("add", "-A")
	runGit("commit", "-m", "seed the skill")
	baseSHA := trimLine(runGit("rev-parse", "HEAD"))

	// Written at the new path and removed from the old. Git detects the
	// rename at diff time, not at commit time, so this is the same R
	// entry `git mv` would produce.
	writeFile(to, body+"BRAND NEW PRESCRIPTIVE CONTENT\n")
	runGit("rm", "-q", from)
	runGit("add", "-A")
	runGit("commit", "-m", "rename and rewrite, no owner named")

	// Guard the fixture's own premise: if git did not detect a rename,
	// this test would pass under the narrower filter too and prove
	// nothing about R.
	status := runGit("log", "--format=", "--name-status", "-1")
	if !strings.Contains(status, "R") {
		t.Fatalf("fixture premise broken: git did not report a rename:\n%s", status)
	}

	vs, err := skillEditProvenanceViolations(root, baseSHA)
	if err != nil {
		t.Fatalf("skillEditProvenanceViolations: %v", err)
	}
	want := []string{to}
	if got := violationFiles(vs); !equalStrings(got, want) {
		t.Errorf("violation files = %v, want %v (a rename must not escape the gate)", got, want)
	}
}

// TestSkillEditProvenance_ResolvesThroughPriorIDs pins the arm that
// keeps an older commit trailer resolving after `aiwf reallocate`
// renumbers its owning entity. The verb rewrites references in the tree;
// it cannot rewrite a commit message, so without this the renumber would
// turn every landed skill-edit commit naming the old id red.
func TestSkillEditProvenance_ResolvesThroughPriorIDs(t *testing.T) {
	t.Parallel()
	const renumbered = "---\nid: E-0002\ntitle: Renumbered fixture epic\nstatus: proposed\nprior_ids:\n    - E-0001\n---\n## Goal\n\nFixture.\n"

	root, runGit, writeFile, _ := skillFixtureBase(t)
	writeFile("work/epics/E-0002-renumbered-fixture-epic/epic.md", renumbered)
	runGit("add", "-A")
	runGit("commit", "-m", "seed the renumbered entity")
	baseSHA := trimLine(runGit("rev-parse", "HEAD"))

	// The commit names the id the entity used to carry.
	writeFile(provSkillRel, "# fictional echo skill\n")
	runGit("add", "-A")
	runGit("commit", "-m", "feat(skill): edit", "--trailer", "aiwf-entity: E-0001")

	vs, err := skillEditProvenanceViolations(root, baseSHA)
	if err != nil {
		t.Fatalf("skillEditProvenanceViolations: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("an id carried in prior_ids must resolve; got %+v", vs)
	}
}

// TestSkillEditProvenance_UnparseableOwnerStillResolves pins that a
// malformed owning entity does not read as a missing one. The loader
// records an unparseable file as a path-derived stub, which still proves
// the entity exists; its broken YAML is a finding for `aiwf check`. If
// this arm regressed, an edit to an unrelated file could turn a landed
// commit red and advise naming an entity that already exists.
func TestSkillEditProvenance_UnparseableOwnerStillResolves(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := skillFixtureBase(t)
	writeFile(provFixtureEntityAt, provFixtureEntity)
	runGit("add", "-A")
	runGit("commit", "-m", "seed the fixture tree")
	baseSHA := trimLine(runGit("rev-parse", "HEAD"))

	writeFile(provSkillRel, "# fictional echo skill\n")
	runGit("add", "-A")
	runGit("commit", "-m", "feat(skill): edit", "--trailer", "aiwf-entity: "+provFixtureEntityID)

	// Break the owning entity's frontmatter and nothing else.
	writeFile(provFixtureEntityAt, "---\nid: [unclosed\ntitle: broken\n---\n## Goal\n\nx\n")

	vs, err := skillEditProvenanceViolations(root, baseSHA)
	if err != nil {
		t.Fatalf("skillEditProvenanceViolations: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("an unparseable owning entity still exists; got %+v", vs)
	}
}

// TestSkillEditProvenance_NonASCIIPathIsWatched pins that a skill whose
// path carries a non-ASCII character is still judged. Git C-quotes such
// a path by default, which would defeat the /SKILL.md suffix test and
// let the edit through unowned.
func TestSkillEditProvenance_NonASCIIPathIsWatched(t *testing.T) {
	t.Parallel()
	const weird = skillRitualsDir + "/plugins/aiwf-extensions/skills/aiwfx-fictional-wéird/SKILL.md"

	root, runGit, writeFile, baseSHA := skillFixtureBase(t)
	writeFile(weird, "# fictional skill with a non-ascii path\n")
	runGit("add", "-A")
	runGit("commit", "-m", "feat(skill): edit with no owner named")

	vs, err := skillEditProvenanceViolations(root, baseSHA)
	if err != nil {
		t.Fatalf("skillEditProvenanceViolations: %v", err)
	}
	want := []string{weird}
	if got := violationFiles(vs); !equalStrings(got, want) {
		t.Errorf("violation files = %v, want %v (a non-ASCII path must not escape)", got, want)
	}
}
