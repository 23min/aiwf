package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestList_AllShippedSkillsPresent guards the contract that we ship
// the six verb skills from session 3, aiwf-status (added on
// poc/aiwf-rename-skills), aiwf-contract (added in I1.8 of the
// contracts plan), aiwf-authorize (added in I2.5), aiwf-render
// (added with the v0.2.0 HTML render), aiwf-edit-body (added in
// M-058 of E-15), aiwf-retitle (added in M-077 of E-22 for the
// title-mutation verb that closes G-065), and aiwf-list (added in
// M-073 of E-20 for the planning-tree filter primitive).
func TestList_AllShippedSkillsPresent(t *testing.T) {
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(skills))
	for i, s := range skills {
		got[i] = s.Name
	}
	want := []string{"aiwf-add", "aiwf-authorize", "aiwf-check", "aiwf-contract", "aiwf-edit-body", "aiwf-history", "aiwf-list", "aiwf-promote", "aiwf-reallocate", "aiwf-rename", "aiwf-render", "aiwf-retitle", "aiwf-status"}
	if len(got) != len(want) {
		t.Fatalf("got %d skills, want %d (%v vs %v)", len(got), len(want), got, want)
	}
	sort.Strings(got)
	for i, name := range want {
		if got[i] != name {
			t.Errorf("[%d] got %q, want %q", i, got[i], name)
		}
	}
}

// TestList_ContentNonEmptyAndYAMLFrontmatter sanity-checks that every
// embedded SKILL.md starts with a YAML front-matter block; a missing
// front-matter would silently break Claude Code's skill loader.
func TestList_ContentNonEmptyAndYAMLFrontmatter(t *testing.T) {
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range skills {
		if len(s.Content) == 0 {
			t.Errorf("%s: empty content", s.Name)
			continue
		}
		if !strings.HasPrefix(string(s.Content), "---\n") {
			t.Errorf("%s: missing YAML front-matter (no leading ---)", s.Name)
		}
		if !strings.Contains(string(s.Content), "\nname: "+s.Name+"\n") {
			t.Errorf("%s: front-matter `name:` does not match dir", s.Name)
		}
	}
}

// TestList_I2_5ContentMarkers guards against content drift on the
// I2.5 documentation surface. Each skill that received a step-10
// update must still mention the load-bearing concept the update
// added; if a future edit drops a marker, this test surfaces it
// before a release.
//
// Markers are deliberately small substrings — section anchors and
// flag/code names — chosen so an editor can rephrase prose freely.
// Add a marker only when its absence would represent a regression
// in AI-discoverability.
func TestList_I2_5ContentMarkers(t *testing.T) {
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	contentByName := make(map[string]string, len(skills))
	for _, s := range skills {
		contentByName[s.Name] = string(s.Content)
	}

	cases := []struct {
		skill   string
		markers []string
	}{
		{
			skill: "aiwf-authorize",
			markers: []string{
				"--to <agent>",
				"--pause",
				"--resume",
				"Tool vs. agent",
				"`provenance-no-active-scope`",
				"`provenance-authorization-out-of-scope`",
				"`provenance-authorization-ended`",
				"`provenance-authorization-missing`",
				"`provenance-trailer-incoherent`",
			},
		},
		{
			skill: "aiwf-add",
			markers: []string{
				"--principal human/<id>",
				"`provenance-trailer-incoherent`",
			},
		},
		{
			skill: "aiwf-promote",
			markers: []string{
				"--audit-only",
				"--principal human/<id>",
				"`provenance-no-active-scope`",
				"aiwf-scope-ends",
			},
		},
		{
			skill: "aiwf-history",
			markers: []string{
				"--show-authorization",
				"principal via agent",
				"[scope: opened]",
				"[audit-only:",
				"provenance-untrailered-entity-commit",
			},
		},
		{
			skill: "aiwf-check",
			markers: []string{
				"`provenance-trailer-incoherent`",
				"`provenance-force-non-human`",
				"`provenance-actor-malformed`",
				"`provenance-principal-non-human`",
				"`provenance-on-behalf-of-non-human`",
				"`provenance-authorized-by-malformed`",
				"`provenance-authorization-missing`",
				"`provenance-authorization-out-of-scope`",
				"`provenance-authorization-ended`",
				"`provenance-no-active-scope`",
				"`provenance-audit-only-non-human`",
				"`provenance-untrailered-entity-commit`",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.skill, func(t *testing.T) {
			content, ok := contentByName[c.skill]
			if !ok {
				t.Fatalf("skill %s not found in embedded set", c.skill)
			}
			for _, m := range c.markers {
				if !strings.Contains(content, m) {
					t.Errorf("%s: missing marker %q (I2.5 content drift)", c.skill, m)
				}
			}
		})
	}
}

// extractH2Section returns the body of the named `## <heading>`
// section in markdown content, honoring fenced code blocks so a
// `## ` line inside a fenced example does not terminate the scan
// early. Used by M-068's AC tests to scope assertions to the
// body-prose subsection.
func extractH2Section(content, heading string) (string, bool) {
	idx := strings.Index(content, heading)
	if idx < 0 {
		return "", false
	}
	body := content[idx:]
	lines := strings.Split(body, "\n")
	if len(lines) == 0 {
		return body, true
	}
	out := []string{lines[0]}
	inFence := false
	for _, line := range lines[1:] {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			out = append(out, line)
			continue
		}
		if !inFence && strings.HasPrefix(line, "## ") {
			break
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), true
}

// TestExtractH2Section covers the helper's branches the AC tests
// don't exercise directly. The AC tests all pass a real heading
// and use the populated `ok=true` arm; this test pins the
// `heading missing` arm and the fence-aware behavior the helper
// was added for.
func TestExtractH2Section(t *testing.T) {
	t.Run("heading missing returns ok=false", func(t *testing.T) {
		body, ok := extractH2Section("# only h1 here\n\nsome text\n", "## Missing")
		if ok {
			t.Errorf("ok = true, want false; body = %q", body)
		}
		if body != "" {
			t.Errorf("body = %q, want empty", body)
		}
	})

	t.Run("fenced ## inside example does not terminate scope", func(t *testing.T) {
		input := "## Target\n\nfirst paragraph\n\n```markdown\n## What's missing\nfake heading inside fence\n```\n\nsecond paragraph\n\n## Next section\n\nafter\n"
		body, ok := extractH2Section(input, "## Target")
		if !ok {
			t.Fatal("ok = false, want true")
		}
		// The body should contain both paragraphs and the fenced
		// example, but stop at `## Next section`.
		if !strings.Contains(body, "first paragraph") {
			t.Error("body missing first paragraph")
		}
		if !strings.Contains(body, "second paragraph") {
			t.Error("body missing second paragraph (fence-aware cap broken)")
		}
		if !strings.Contains(body, "fake heading inside fence") {
			t.Error("body missing fenced example body")
		}
		if strings.Contains(body, "after") {
			t.Error("body included content past `## Next section` — cap broken")
		}
	})
}

// TestSkill_AddNamesFillInBodyAsRequiredNextStep pins M-068/AC-1:
// the embedded `aiwf-add` SKILL.md must name "fill in the body" as a
// required follow-up step — not optional, not just for ACs — across
// every entity kind. Today the skill describes each `aiwf add <kind>`
// invocation and stops at the verb's atomic commit; an LLM (or
// human) following the skill ends up with bare body sections by
// default. M-068 makes the skill teach the design intent explicitly
// so the typical entity-creation flow produces non-empty bodies.
//
// The AC has two surfaces inside the skill:
//
//   - A body-prose subsection (heading + body) stating step 1 is
//     scaffolding and step 2 is filling the body, that step 2 is
//     **required** rather than optional, and that the requirement
//     applies across all six entity kinds plus ACs.
//   - A new step in the existing "What aiwf does" numbered list
//     calling out that scaffolded body sections are empty by design
//     and must be filled in before the entity counts as complete.
//
// Both surfaces target the same failure mode from different angles
// so an LLM scanning the skill can't miss the requirement no matter
// which section it reads first.
func TestSkill_AddNamesFillInBodyAsRequiredNextStep(t *testing.T) {
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	var content string
	for _, s := range skills {
		if s.Name == "aiwf-add" {
			content = string(s.Content)
			break
		}
	}
	if content == "" {
		t.Fatal("aiwf-add skill not found in embedded set")
	}

	// AC-1 surface 1 — body-prose subsection. We assert markers that
	// any reasonable phrasing of the spec would hit: a heading that
	// names "fill in the body" (or equivalent), explicit "required"
	// language so the operator can't read it as optional, and the
	// per-kind list so the requirement applies to more than ACs.
	mustContain := []string{
		// A heading marker — the subsection lands as a `## ...`
		// section, not a stray sentence buried in another section.
		"## After `aiwf add",
		// Required-not-optional language. The exact wording can be
		// "required, not optional" or "is required" — both flavors
		// pass; what matters is the operator sees "required."
		"required",
		// Per-kind reach. The subsection (or step 6 below) names the
		// load-bearing body sections per kind, not just AC bodies.
		// We sample three kinds that operators commonly create.
		"epic",
		"milestone",
		"gap",
		// AC body shape — `### AC-N — <title>` is the AC's body
		// heading; the skill should reference it explicitly.
		"### AC-N",
	}
	for _, m := range mustContain {
		if !strings.Contains(content, m) {
			t.Errorf("AC-1 surface (body-prose subsection): missing marker %q", m)
		}
	}

	// AC-1 surface 2 — step 6 in "What aiwf does." The numbered list
	// today ends at step 5 (creates one commit). M-068 adds step 6
	// pointing at the body. We assert the literal "6." plus the
	// "fill" verb co-occurring inside that section's body.
	idx := strings.Index(content, "## What aiwf does")
	if idx < 0 {
		t.Fatal("aiwf-add skill missing the `## What aiwf does` section heading")
	}
	// Cap the search at the next top-level section so we don't
	// accidentally match a "6." in a later unrelated section.
	tail := content[idx:]
	if next := strings.Index(tail[2:], "\n## "); next > 0 {
		tail = tail[:next+2]
	}
	step6Markers := []string{
		"6.",
		"fill",
	}
	for _, m := range step6Markers {
		if !strings.Contains(tail, m) {
			t.Errorf("AC-1 surface (`## What aiwf does` step 6): missing marker %q", m)
		}
	}
}

// TestSkill_AddCitesDesignIntent pins M-068/AC-2: the body-prose
// subsection introduced by AC-1 must cite the two canonical design
// sources for the "prose is not parsed" / body-as-spec stance.
// Without explicit citations, an LLM following the skill has no
// breadcrumb back to the kernel-level rationale and is likely to
// treat the body-prose requirement as a per-skill quirk rather than
// a design invariant — exactly the failure mode the design-doc-anchors
// principle exists to prevent.
//
// Two paths must appear in the SKILL.md content **and** must land
// inside the body-prose subsection (not buried elsewhere — the
// citation must be co-located with the prescription so the operator
// sees them together):
//
//   - docs/pocv3/plans/acs-and-tdd-plan.md:22 — the "prose is not
//     parsed" line and the AC body-shape recommendation.
//   - docs/pocv3/design/design-decisions.md:139 — the broader
//     "tree carries semantic detail in prose, not in structure"
//     stance.
//
// Both literal paths must be present; both must be inside the
// `## After aiwf add <kind>: fill in the body` section. Substring
// presence anywhere is not enough — that would let a future change
// move the citation into a footnote and the test wouldn't notice.
func TestSkill_AddCitesDesignIntent(t *testing.T) {
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	var content string
	for _, s := range skills {
		if s.Name == "aiwf-add" {
			content = string(s.Content)
			break
		}
	}
	if content == "" {
		t.Fatal("aiwf-add skill not found in embedded set")
	}

	// Locate the body-prose subsection introduced by AC-1, then
	// scope the citation assertions to its body so a future
	// drift can't slip them past us by relocating the text.
	tail, ok := extractH2Section(content, "## After `aiwf add")
	if !ok {
		t.Fatal("AC-2 prerequisite: body-prose subsection missing — AC-1 must land first")
	}

	citations := []string{
		"docs/pocv3/plans/acs-and-tdd-plan.md:22",
		"docs/pocv3/design/design-decisions.md:139",
	}
	for _, c := range citations {
		if !strings.Contains(tail, c) {
			t.Errorf("AC-2: citation %q missing from the body-prose subsection", c)
		}
	}
}

// TestSkill_AddRecommendsBodyShape pins M-068/AC-3: the body-prose
// subsection prescribes a body-shape recommendation per kind plus
// at least one short concrete example block per shape, so an
// operator (or LLM) following the skill has a copyable starting
// point rather than a "fill it in somehow" hand-wave.
//
// Two structural surfaces inside the body-prose subsection:
//
//   - A "What to write per kind" sub-heading (`### `) carrying
//     the per-kind shape guidance paragraphs.
//   - At least one fenced code block (` ``` `) inside that sub-
//     section so the operator can see and copy a concrete shape.
//
// Plus content markers covering the spec's three required pieces
// for the AC-body shape (pass criterion, edge cases, code
// references) and at least one top-level kind's shape phrase
// (gap "What's missing" — "concrete defect", per the spec's
// own example).
func TestSkill_AddRecommendsBodyShape(t *testing.T) {
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	var content string
	for _, s := range skills {
		if s.Name == "aiwf-add" {
			content = string(s.Content)
			break
		}
	}
	if content == "" {
		t.Fatal("aiwf-add skill not found in embedded set")
	}

	// Scope to the body-prose subsection so the per-kind shape
	// guidance lands in the same place the operator reads the
	// rule itself, not in a stray section elsewhere.
	tail, ok := extractH2Section(content, "## After `aiwf add")
	if !ok {
		t.Fatal("AC-3 prerequisite: body-prose subsection missing — AC-1 must land first")
	}

	// AC-3 surface 1 — sub-heading anchoring the per-kind shape
	// guidance. Without an anchor heading, the prescriptions
	// could drift across the rest of the section over time.
	if !strings.Contains(tail, "### What to write per kind") {
		t.Errorf("AC-3 surface (anchor heading): missing %q", "### What to write per kind")
	}

	// AC-3 surface 2 — at least one fenced code block (example).
	// We require the closing fence too so a stray ``` doesn't
	// pass — the example must actually be a complete fenced
	// block.
	openCount := strings.Count(tail, "\n```")
	if openCount < 2 {
		// Each fenced block has an opening and a closing fence,
		// so at least one block means at least 2 occurrences.
		t.Errorf("AC-3 surface (example block): need at least one fenced code block in the body-prose subsection (got %d fence markers)",
			openCount)
	}

	// AC-3 content markers — the AC-body shape spec says the
	// paragraph covers pass criterion, edge cases, and code
	// references. All three must appear inside the sub-section
	// so the operator sees the full shape for AC bodies.
	acBodyMarkers := []string{
		"pass criterion",
		"edge cases",
		"code references",
	}
	for _, m := range acBodyMarkers {
		if !strings.Contains(tail, m) {
			t.Errorf("AC-3 content (AC body shape): missing marker %q", m)
		}
	}

	// AC-3 content markers — at least one top-level kind's
	// shape phrase. The spec's own example for `## What's
	// missing` is "the concrete defect"; we pin that literal
	// (cheap, exact) rather than try to assert across all six
	// top-level kinds, which would couple the test too tightly
	// to wording.
	if !strings.Contains(tail, "concrete defect") {
		t.Errorf("AC-3 content (top-level shape): missing marker %q (gap What's-missing example phrase)",
			"concrete defect")
	}
}

// TestSkill_AddNamesBodyFileAsAlternative pins M-068/AC-4: the
// body-prose subsection names `--body-file` as the in-verb
// alternative to the default two-step (`aiwf add` then
// `aiwf edit-body`) flow, with an explicit cross-reference to
// M-067 so the operator can trace the verb history. The cross-
// reference is two-way: M-067/AC-8's tests pin the analogous
// reference in the other direction.
//
// "When to use" guidance: the skill should make clear that the
// in-verb form is the right choice when the operator has the
// body content already drafted (mining from a design doc, prior
// conversation, etc.) — landing it in the create commit avoids
// the follow-up untrailered hand-edit and the
// `provenance-untrailered-entity-commit` warning that would
// otherwise fire.
//
// AC-4's spec asserted the flag was AC-only (with G-066 capturing
// the non-AC follow-up). That's stale: M-056 already extended
// `--body-file` to all six top-level kinds before M-067 added
// the AC variant. This test asserts the skill's actual content,
// which is the universal availability — it does NOT pin the
// stale "AC-only" framing the spec text used.
func TestSkill_AddNamesBodyFileAsAlternative(t *testing.T) {
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	var content string
	for _, s := range skills {
		if s.Name == "aiwf-add" {
			content = string(s.Content)
			break
		}
	}
	if content == "" {
		t.Fatal("aiwf-add skill not found in embedded set")
	}

	tail, ok := extractH2Section(content, "## After `aiwf add")
	if !ok {
		t.Fatal("AC-4 prerequisite: body-prose subsection missing — AC-1 must land first")
	}

	// AC-4 surface — the body-prose subsection must call the
	// flag by name and pair it with the two-step alternative
	// so the operator knows both paths.
	mustContain := []string{
		// The flag itself, named in the subsection (not just
		// referenced "above" via cross-link).
		"--body-file",
		// The two-step alternative co-located so the operator
		// reads them in one place rather than separately.
		"aiwf edit-body",
		// Explicit M-067 cross-reference. The spec calls for
		// a two-way pointer; without the literal id the trail
		// from this skill back to the verb history is lost.
		// Narrow width matches the SKILL.md prose verbatim;
		// body-prose canonicalization is M-082's `aiwf rewidth`.
		"M-067",
		// "When to use" — the skill should signal that the
		// in-verb form is for content already drafted, not a
		// universal default. The literal phrase the AC's spec
		// uses is "already drafted"; we pin that wording.
		"already drafted",
	}
	for _, m := range mustContain {
		if !strings.Contains(tail, m) {
			t.Errorf("AC-4 (body-file cross-reference): missing marker %q", m)
		}
	}
}

// TestSkill_AddDontEntryAgainstEmptyBodies pins M-068/AC-5: the
// skill's `## Don't` section gains a concise entry against shipping
// load-bearing body sections empty. The body-prose subsection
// (AC-1, AC-2, AC-3, AC-4) is the long-form prescription; the
// Don't entry is the short reminder. Both surfaces target the same
// failure mode at different reading depths so an LLM scanning the
// skill catches the requirement whichever section it lands in
// first.
//
// The entry must:
//
//   - Live inside the `## Don't` section, not floating elsewhere.
//   - Name the failure mode in operator-facing language ("empty
//     body sections" or equivalent).
//   - Reference `entity-body-empty` so the operator knows the
//     finding code that surfaces the omission.
//   - Reference M-066 so the cross-link to the rule is explicit.
func TestSkill_AddDontEntryAgainstEmptyBodies(t *testing.T) {
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	var content string
	for _, s := range skills {
		if s.Name == "aiwf-add" {
			content = string(s.Content)
			break
		}
	}
	if content == "" {
		t.Fatal("aiwf-add skill not found in embedded set")
	}

	tail, ok := extractH2Section(content, "## Don't")
	if !ok {
		t.Fatal("AC-5 prerequisite: `## Don't` section missing from aiwf-add SKILL.md")
	}

	mustContain := []string{
		// Operator-facing phrasing — the entry must use the
		// load-bearing-body language, not abstract jargon.
		"empty",
		"body",
		// Finding code so the operator knows what `aiwf check`
		// will surface.
		"entity-body-empty",
		// Cross-reference to the rule's milestone so the trail
		// from the Don't entry back to the rule is one click.
		// Narrow width matches the SKILL.md prose verbatim;
		// body-prose canonicalization is M-082's `aiwf rewidth`.
		"M-066",
	}
	for _, m := range mustContain {
		if !strings.Contains(tail, m) {
			t.Errorf("AC-5 (Don't entry): missing marker %q from `## Don't` section", m)
		}
	}
}

// TestMaterialize_FreshDir writes every embedded skill into a clean
// directory and verifies the on-disk content matches the embed
// byte-for-byte.
func TestMaterialize_FreshDir(t *testing.T) {
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range skills {
		on := filepath.Join(root, SkillsDir, s.Name, "SKILL.md")
		got, err := os.ReadFile(on)
		if err != nil {
			t.Fatalf("read %s: %v", on, err)
		}
		if !bytes.Equal(got, s.Content) {
			t.Errorf("%s: on-disk content differs from embed", s.Name)
		}
	}
}

// TestMaterialize_WipesPreviouslyOwnedStale: when a previous aiwf
// version listed `aiwf-removed` in its ownership manifest and the
// current version no longer embeds it, Materialize wipes the stale
// dir. This is the "skill removed from a release" cleanup path.
func TestMaterialize_WipesPreviouslyOwnedStale(t *testing.T) {
	root := t.TempDir()
	skillsRoot := filepath.Join(root, SkillsDir)
	if err := os.MkdirAll(skillsRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(skillsRoot, "aiwf-removed")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, "SKILL.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Pretend a previous aiwf wrote a manifest claiming to own `aiwf-removed`.
	if err := os.WriteFile(filepath.Join(skillsRoot, ManifestFile), []byte("aiwf-removed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale aiwf-removed/ should have been wiped (was in old manifest); stat err=%v", err)
	}
}

// TestMaterialize_LeavesForeignAiwfPrefixedDirAlone is the load-bearing
// test for G7: a directory named like `aiwf-rituals-something` that
// aiwf never owned (not in any prior manifest) must NOT be wiped, even
// though it shares the `aiwf-` prefix. Third-party plugins under the
// prefix are safe.
func TestMaterialize_LeavesForeignAiwfPrefixedDirAlone(t *testing.T) {
	root := t.TempDir()
	foreign := filepath.Join(root, SkillsDir, "aiwf-rituals-tdd")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(foreign, "MARKER")
	if err := os.WriteFile(marker, []byte("third-party"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("foreign aiwf-prefixed dir was wiped (it should not be); read err=%v", err)
	}
	if string(got) != "third-party" {
		t.Errorf("foreign content modified: %q", got)
	}
}

// TestMaterialize_WritesManifest: after Materialize succeeds, the
// ownership manifest lists exactly the names of currently-embedded
// skills, one per line.
func TestMaterialize_WritesManifest(t *testing.T) {
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, SkillsDir, ManifestFile)
	got, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	want := ""
	for _, s := range skills {
		want += s.Name + "\n"
	}
	if string(got) != want {
		t.Errorf("manifest content mismatch:\nwant:\n%s\ngot:\n%s", want, string(got))
	}
}

// TestMaterialize_RoundTripPreservesForeignAcrossUpdates: a foreign
// dir survives multiple Materialize calls (simulating successive
// `aiwf update` invocations).
func TestMaterialize_RoundTripPreservesForeignAcrossUpdates(t *testing.T) {
	root := t.TempDir()
	foreign := filepath.Join(root, SkillsDir, "aiwf-userplugin")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, "SKILL.md"), []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := Materialize(root); err != nil {
			t.Fatalf("Materialize iteration %d: %v", i, err)
		}
	}
	got, err := os.ReadFile(filepath.Join(foreign, "SKILL.md"))
	if err != nil {
		t.Fatalf("foreign skill removed across updates: %v", err)
	}
	if string(got) != "user" {
		t.Errorf("foreign content changed: %q", got)
	}
}

// TestMaterialize_PreservesNonAiwfDirs guards the namespace boundary —
// user-authored `.claude/skills/<not-aiwf>/` directories must not be
// touched by Materialize.
func TestMaterialize_PreservesNonAiwfDirs(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, SkillsDir, "my-custom-skill")
	if err := os.MkdirAll(user, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(user, "SKILL.md"), []byte("user content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(user, "SKILL.md"))
	if err != nil {
		t.Fatalf("user skill removed: %v", err)
	}
	if string(got) != "user content" {
		t.Errorf("user skill content changed: %q", got)
	}
}

func TestGitignorePatterns(t *testing.T) {
	got := GitignorePatterns()
	if len(got) != 3 {
		t.Fatalf("got %d patterns, want 3 (wildcard + manifest + binary); got %v", len(got), got)
	}
	wantWildcard := SkillsDir + "/aiwf-*/"
	wantManifest := SkillsDir + "/" + ManifestFile
	wantBinary := "/aiwf"

	var sawWildcard, sawManifest, sawBinary bool
	for _, p := range got {
		switch p {
		case wantWildcard:
			sawWildcard = true
		case wantManifest:
			sawManifest = true
		case wantBinary:
			sawBinary = true
		default:
			t.Errorf("unexpected pattern %q", p)
		}
	}
	if !sawWildcard {
		t.Errorf("missing directory wildcard %q (G19: makes .gitignore future-proof against new aiwf-* skills)", wantWildcard)
	}
	if !sawManifest {
		t.Errorf("missing manifest entry %q (otherwise .aiwf-owned would land in git commits)", wantManifest)
	}
	if !sawBinary {
		t.Errorf("missing binary entry %q (G-0057: bare `go build ./cmd/aiwf` drops a binary at repo root that must not land in commits)", wantBinary)
	}
	if !strings.HasSuffix(wantWildcard, "/") {
		t.Errorf("wildcard %q should end with / so it only matches directories", wantWildcard)
	}
	if !strings.HasPrefix(wantBinary, "/") {
		t.Errorf("binary entry %q should start with / so it only anchors to repo root (cmd/aiwf/ stays trackable)", wantBinary)
	}
}

// TestGitignorePatterns_BinaryWrittenByInit pins G-0057's load-bearing
// claim: a fresh `aiwf init` writes `/aiwf` into the consumer's
// .gitignore. The unit test on GitignorePatterns above asserts the
// helper returns the pattern; this test asserts the seam to
// ensureGitignore actually writes it. Without the seam test, a future
// refactor could drop the pattern from the iteration without breaking
// the helper-level test.
//
// Lives next to TestGitignorePatterns rather than in initrepo_test.go
// because the assertion is about what skills.GitignorePatterns()
// promises to its caller, not about ensureGitignore's other branches.
func TestGitignorePatterns_BinaryEntryListed(t *testing.T) {
	for _, p := range GitignorePatterns() {
		if p == "/aiwf" {
			return
		}
	}
	t.Errorf("/aiwf missing from GitignorePatterns(); ensureGitignore won't reconcile it on aiwf init / aiwf update (G-0057)")
}
