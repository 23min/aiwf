package skills

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// names extracts the Name field from a Skill slice for set assertions.
func names(ss []Skill) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = s.Name
	}
	return out
}

// contains reports whether want is in got.
func contains(got []string, want string) bool {
	for _, g := range got {
		if g == want {
			return true
		}
	}
	return false
}

// TestListRitualAgents covers M-0150 AC-1: the four vendored ritual
// agents are discovered from the embedded-rituals tree, name-sorted,
// with content. Names carry the .md suffix (the file is the unit, not a
// directory like skills).
func TestListRitualAgents(t *testing.T) {
	t.Parallel()
	got, err := ListRitualAgents()
	if err != nil {
		t.Fatalf("ListRitualAgents: %v", err)
	}
	gotNames := names(got)
	if !sort.StringsAreSorted(gotNames) {
		t.Errorf("ListRitualAgents not name-sorted: %v", gotNames)
	}
	for _, want := range []string{"builder.md", "deployer.md", "planner.md", "reviewer.md"} {
		if !contains(gotNames, want) {
			t.Errorf("ListRitualAgents missing %q; got %v", want, gotNames)
		}
	}
	for _, s := range got {
		if !strings.HasSuffix(s.Name, ".md") {
			t.Errorf("ritual agent %q is not a .md file", s.Name)
		}
		if len(s.Content) == 0 {
			t.Errorf("ritual agent %q has empty content", s.Name)
		}
	}
}

// TestListRitualTemplates covers M-0150 AC-1: the four vendored ritual
// templates are discovered, name-sorted, with content. The wf-rituals
// templates/.gitkeep is a dotfile excluded by go:embed, so only the
// four aiwf-extensions templates qualify.
func TestListRitualTemplates(t *testing.T) {
	t.Parallel()
	got, err := ListRitualTemplates()
	if err != nil {
		t.Fatalf("ListRitualTemplates: %v", err)
	}
	gotNames := names(got)
	if !sort.StringsAreSorted(gotNames) {
		t.Errorf("ListRitualTemplates not name-sorted: %v", gotNames)
	}
	want := []string{"adr.md", "decision.md", "epic-spec.md", "milestone-spec.md"}
	for _, w := range want {
		if !contains(gotNames, w) {
			t.Errorf("ListRitualTemplates missing %q; got %v", w, gotNames)
		}
	}
	for _, s := range got {
		if !strings.HasSuffix(s.Name, ".md") {
			t.Errorf("ritual template %q is not a .md file", s.Name)
		}
		if len(s.Content) == 0 {
			t.Errorf("ritual template %q has empty content", s.Name)
		}
	}
}

// TestMaterialize_WritesRitualAgents covers AC-1: Materialize writes the
// embedded ritual agents, flat, into .claude/agents/<name>.
func TestMaterialize_WritesRitualAgents(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	agents, err := ListRitualAgents()
	if err != nil {
		t.Fatalf("ListRitualAgents: %v", err)
	}
	for _, a := range agents {
		got, err := os.ReadFile(filepath.Join(root, AgentsDir, a.Name))
		if err != nil {
			t.Errorf("ritual agent %s not materialized: %v", a.Name, err)
			continue
		}
		if string(got) != string(a.Content) {
			t.Errorf("materialized content mismatch for agent %s", a.Name)
		}
	}
}

// TestMaterialize_WritesRitualTemplates covers AC-1: Materialize writes
// the embedded ritual templates, flat, into .claude/templates/<name>
// (D-0015 — the Claude-target template location).
func TestMaterialize_WritesRitualTemplates(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	tmpls, err := ListRitualTemplates()
	if err != nil {
		t.Fatalf("ListRitualTemplates: %v", err)
	}
	for _, tm := range tmpls {
		got, err := os.ReadFile(filepath.Join(root, TemplatesDir, tm.Name))
		if err != nil {
			t.Errorf("ritual template %s not materialized: %v", tm.Name, err)
			continue
		}
		if string(got) != string(tm.Content) {
			t.Errorf("materialized content mismatch for template %s", tm.Name)
		}
	}
}

// readManifestLines is a test helper returning the trimmed non-empty
// lines of a .aiwf-owned manifest under dir.
func readManifestLines(t *testing.T, dir string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, ManifestFile))
	if err != nil {
		t.Fatalf("read manifest %s: %v", dir, err)
	}
	var out []string
	for _, l := range strings.Split(string(raw), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// TestMaterialize_ManifestOwnsAgentsAndTemplates covers AC-2: each flat
// artifact root carries its own .aiwf-owned manifest naming exactly the
// materialized files.
func TestMaterialize_ManifestOwnsAgentsAndTemplates(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	agentManifest := readManifestLines(t, filepath.Join(root, AgentsDir))
	for _, want := range []string{"builder.md", "planner.md", "reviewer.md", "deployer.md"} {
		if !contains(agentManifest, want) {
			t.Errorf("agents manifest missing %q; got %v", want, agentManifest)
		}
	}
	tmplManifest := readManifestLines(t, filepath.Join(root, TemplatesDir))
	for _, want := range []string{"adr.md", "decision.md", "epic-spec.md", "milestone-spec.md"} {
		if !contains(tmplManifest, want) {
			t.Errorf("templates manifest missing %q; got %v", want, tmplManifest)
		}
	}
}

// TestMaterialize_AgentsTemplatesIdempotent covers AC-2: re-running
// Materialize (the `aiwf update` path) leaves the agents and templates
// in place.
func TestMaterialize_AgentsTemplatesIdempotent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("first Materialize: %v", err)
	}
	if err := Materialize(root); err != nil {
		t.Fatalf("second Materialize: %v", err)
	}
	for _, p := range []string{
		filepath.Join(root, AgentsDir, "planner.md"),
		filepath.Join(root, TemplatesDir, "milestone-spec.md"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("artifact missing after re-materialize: %s: %v", p, err)
		}
	}
}

// TestMaterialize_DoesNotClobberUserAgents covers AC-2: a user-authored
// agent or template (a name aiwf never claimed in its manifest) is left
// untouched by materialization.
func TestMaterialize_DoesNotClobberUserAgents(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	userAgent := filepath.Join(root, AgentsDir, "my-agent.md")
	userTmpl := filepath.Join(root, TemplatesDir, "my-template.md")
	for _, p := range []string{userAgent, userTmpl} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("mine\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	for _, p := range []string{userAgent, userTmpl} {
		got, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("user file removed by Materialize: %s: %v", p, err)
			continue
		}
		if string(got) != "mine\n" {
			t.Errorf("user file content changed by Materialize: %s", p)
		}
	}
}

// TestMaterialize_FlatFilesWipeRemoved covers AC-2's wipe branch in
// materializeFlatFiles: a file the prior manifest claimed but the
// current embed no longer ships is removed on the next Materialize,
// while foreign files survive. The first Materialize is real (writes
// the agent manifest); we then plant a stale entry by hand and confirm
// the next Materialize sweeps the matching file but not the foreign one.
func TestMaterialize_FlatFilesWipeRemoved(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("first Materialize: %v", err)
	}
	agentsDir := filepath.Join(root, AgentsDir)
	stale := filepath.Join(agentsDir, "retired-agent.md")
	foreign := filepath.Join(agentsDir, "user-agent.md")
	for _, p := range []string{stale, foreign} {
		if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Append only the stale name to the manifest, so it is "previously
	// owned but no longer embedded"; the foreign file is never named.
	manifestPath := filepath.Join(agentsDir, ManifestFile)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(raw, []byte("retired-agent.md\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Materialize(root); err != nil {
		t.Fatalf("second Materialize: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale owned file not wiped: stat err = %v", err)
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Errorf("foreign file wrongly wiped: %v", err)
	}
}

// TestGitignorePatterns_CoverAgentsAndTemplates covers AC-2: the
// gitignore patterns enumerate every materialized agent and template
// file plus each flat root's manifest. Enumeration (not a wildcard) is
// required because these basenames have no namespacing prefix.
func TestGitignorePatterns_CoverAgentsAndTemplates(t *testing.T) {
	t.Parallel()
	pats, err := GitignorePatterns()
	if err != nil {
		t.Fatalf("GitignorePatterns: %v", err)
	}
	agents, err := ListRitualAgents()
	if err != nil {
		t.Fatalf("ListRitualAgents: %v", err)
	}
	tmpls, err := ListRitualTemplates()
	if err != nil {
		t.Fatalf("ListRitualTemplates: %v", err)
	}
	var want []string
	for _, a := range agents {
		want = append(want, AgentsDir+"/"+a.Name)
	}
	want = append(want, AgentsDir+"/"+ManifestFile)
	for _, tm := range tmpls {
		want = append(want, TemplatesDir+"/"+tm.Name)
	}
	want = append(want, TemplatesDir+"/"+ManifestFile)
	for _, w := range want {
		if !contains(pats, w) {
			t.Errorf("GitignorePatterns missing %q; got %v", w, pats)
		}
	}
}

// looksLikeHook reports whether a path segment or filename indicates a
// Claude/agent hook artifact. ADR-0014 §3: rituals ship skills + agents
// + templates only — never hooks. (aiwf's own git hooks live under
// .git/hooks and are installed by init, orthogonal to the rituals.)
func looksLikeHook(name string) bool {
	switch name {
	case "hooks", "hooks.json", "hooks.toml", "hooks.yaml":
		return true
	}
	return strings.HasSuffix(name, ".hook")
}

// TestRituals_NoHookSurface covers AC-3: the vendored ritual snapshot
// carries no hook artifacts, so materialization introduces no new hook
// surface beyond aiwf's existing git hooks. The guard fires if a future
// upstream sync ever pulls in a hooks/ dir or hooks.json.
func TestRituals_NoHookSurface(t *testing.T) {
	t.Parallel()

	// (a) The embed itself contains no hook artifact.
	err := fs.WalkDir(ritualsFS, ritualsRoot, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		for _, seg := range strings.Split(p, "/") {
			if looksLikeHook(seg) {
				t.Errorf("embedded rituals ship a hook artifact at %q (ADR-0014 §3: hooks are not rituals)", p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking embedded rituals: %v", err)
	}

	// (b) Materialize writes no hook artifact anywhere under .claude/.
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	claudeDir := filepath.Join(root, ".claude")
	err = filepath.WalkDir(claudeDir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if looksLikeHook(d.Name()) {
			t.Errorf("materialization created a hook artifact at %q (ADR-0014 §3)", p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking materialized .claude: %v", err)
	}
}
