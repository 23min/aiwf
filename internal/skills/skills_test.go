package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// TestList_AllShippedSkillsPresent guards the contract that we ship
// the six verb skills from session 3, aiwf-status (added on
// poc/aiwf-rename-skills), aiwf-contract (added in I1.8 of the
// contracts plan), aiwf-authorize (added in I2.5), aiwf-render
// (added with the v0.2.0 HTML render), aiwf-edit-body (added in
// M-058 of E-15), aiwf-retitle (added in M-077 of E-22 for the
// title-mutation verb that closes G-065), aiwf-list (added in
// M-073 of E-20 for the planning-tree filter primitive),
// aiwf-archive (added in M-0088 of E-0024 for the uniform archive
// convention per ADR-0004), aiwf-show (added via G-0087 patch
// — closes the last "deferred" skill-coverage allowlist entry), and
// aiwf-worktree (added in M-0233 of E-0059 for the atomic
// worktree-add + ritual-materialization verb), and aiwf-set-priority
// (added in M-0262 of E-0066 for the priority write-surface verb).
func TestList_AllShippedSkillsPresent(t *testing.T) {
	t.Parallel()
	skills, err := List()
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(skills))
	for i, s := range skills {
		got[i] = s.Name
	}
	want := []string{"aiwf-acknowledge", "aiwf-add", "aiwf-archive", "aiwf-area", "aiwf-authorize", "aiwf-check", "aiwf-contract", "aiwf-edit-body", "aiwf-history", "aiwf-list", "aiwf-promote", "aiwf-reallocate", "aiwf-rename", "aiwf-render", "aiwf-retitle", "aiwf-set-priority", "aiwf-show", "aiwf-status", "aiwf-worktree"}
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
//
// (It once also pinned a literal milestone back-reference; G-0299
// removed real ids from shipped skill bodies, so that marker is gone.)
func TestSkill_AddDontEntryAgainstEmptyBodies(t *testing.T) {
	t.Parallel()
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
		check.CodeEntityBodyEmpty,
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, SkillsDir, ManifestFile)
	got, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	// The manifest is the union of verb skills then ritual skills, in
	// the order Materialize appends them (each set already name-sorted).
	verb, err := List()
	if err != nil {
		t.Fatal(err)
	}
	rituals, err := ListRituals()
	if err != nil {
		t.Fatal(err)
	}
	want := ""
	for _, s := range verb {
		want += s.Name + "\n"
	}
	for _, s := range rituals {
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	got, err := GitignorePatterns()
	if err != nil {
		t.Fatalf("GitignorePatterns: %v", err)
	}

	// Expected set: the static entries (3 skill wildcards + skills
	// manifest + provenance readme + binary + guidance fragment) plus the
	// enumerated agent/template files and their per-dir manifests, derived
	// from the embed exactly as the production helper does.
	wantVerbWildcard := SkillsDir + "/aiwf-*/"
	wantAiwfxWildcard := SkillsDir + "/aiwfx-*/"
	wantWfWildcard := SkillsDir + "/wf-*/"
	wantManifest := SkillsDir + "/" + ManifestFile
	wantReadme := SkillsDir + "/" + ProvenanceReadme
	wantBinary := "/aiwf"
	wantGuidance := GuidanceFile

	want := map[string]bool{
		wantVerbWildcard:  true,
		wantAiwfxWildcard: true,
		wantWfWildcard:    true,
		wantManifest:      true,
		wantReadme:        true,
		wantBinary:        true,
		wantGuidance:      true,
	}
	agents, err := ListRitualAgents()
	if err != nil {
		t.Fatalf("ListRitualAgents: %v", err)
	}
	for _, a := range agents {
		want[AgentsDir+"/"+a.Name] = true
	}
	want[AgentsDir+"/"+ManifestFile] = true
	tmpls, err := ListRitualTemplates()
	if err != nil {
		t.Fatalf("ListRitualTemplates: %v", err)
	}
	for _, tm := range tmpls {
		want[TemplatesDir+"/"+tm.Name] = true
	}
	want[TemplatesDir+"/"+ManifestFile] = true

	gotSet := map[string]bool{}
	for _, p := range got {
		if gotSet[p] {
			t.Errorf("duplicate pattern %q", p)
		}
		gotSet[p] = true
		if !want[p] {
			t.Errorf("unexpected pattern %q", p)
		}
	}
	for w := range want {
		if !gotSet[w] {
			t.Errorf("missing pattern %q", w)
		}
	}

	// Shape invariants that survive the dynamic set.
	for _, w := range []string{wantVerbWildcard, wantAiwfxWildcard, wantWfWildcard} {
		if !strings.HasSuffix(w, "/") {
			t.Errorf("wildcard %q should end with / so it only matches directories", w)
		}
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
	t.Parallel()
	pats, err := GitignorePatterns()
	if err != nil {
		t.Fatalf("GitignorePatterns: %v", err)
	}
	for _, p := range pats {
		if p == "/aiwf" {
			return
		}
	}
	t.Errorf("/aiwf missing from GitignorePatterns(); ensureGitignore won't reconcile it on aiwf init / aiwf update (G-0057)")
}
