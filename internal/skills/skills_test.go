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
// (added with the v0.2.0 HTML render), and aiwf-edit-body (added
// in M-058 of E-15).
func TestList_AllShippedSkillsPresent(t *testing.T) {
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(skills))
	for i, s := range skills {
		got[i] = s.Name
	}
	want := []string{"aiwf-add", "aiwf-authorize", "aiwf-check", "aiwf-contract", "aiwf-edit-body", "aiwf-history", "aiwf-promote", "aiwf-reallocate", "aiwf-rename", "aiwf-render", "aiwf-status"}
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
	if len(got) != 2 {
		t.Fatalf("got %d patterns, want 2 (wildcard + manifest); got %v", len(got), got)
	}
	wantWildcard := SkillsDir + "/aiwf-*/"
	wantManifest := SkillsDir + "/" + ManifestFile

	var sawWildcard, sawManifest bool
	for _, p := range got {
		switch p {
		case wantWildcard:
			sawWildcard = true
		case wantManifest:
			sawManifest = true
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
	if !strings.HasSuffix(wantWildcard, "/") {
		t.Errorf("wildcard %q should end with / so it only matches directories", wantWildcard)
	}
}
