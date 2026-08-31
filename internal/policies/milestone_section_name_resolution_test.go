package policies

import (
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestPolicy_MilestoneSectionNameResolution pins M-0326/AC-1 on the live tree:
// every backticked `## Section` name written in the surfaces that instruct an
// author about milestone-spec and wrap-artefact sections resolves to a heading
// some shipped template — or the wrap artefact's own scaffold — actually
// carries. A name matching no artefact at all reddens this test, which is the
// evidence shape D-0070 leaves available over a shipped surface: prose is
// compared against the artefact it names, never asserted for its own wording.
//
// It does not redden on a rename of a heading another template also carries —
// the universe is their union, and the limit is stated in the policy's doc
// comment.
func TestPolicy_MilestoneSectionNameResolution(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyMilestoneSectionNameResolution)
}

// sectionFixtureRoot writes a minimal shipped tree that the policy reports
// clean, then applies overrides. An override with empty content deletes the
// file, which is how the unreadable-surface cases are built.
func sectionFixtureRoot(t *testing.T, overrides map[string]string) string {
	t.Helper()
	root := t.TempDir()

	base := map[string]string{
		"templates/milestone-spec.md": "# T\n\n## Goal\n\n## Release note\n",
		"templates/epic-spec.md":      "# T\n\n## Goal\n",
		"skills/aiwfx-wrap-epic/SKILL.md": "# wrap\n\n```markdown\n# Epic wrap\n\n" +
			"## Changelog entry\n\n### Added\n\n## Summary\n```\n\nSee `## Changelog entry`.\n",
		"skills/aiwfx-start-milestone/SKILL.md": "Fill `## Goal`.\n",
		"skills/aiwfx-wrap-milestone/SKILL.md":  "Fill `## Release note`.\n",
		"agents/builder.md":                     "Maintain `## Goal`.\n",
		// DFS order and full-path lexical order diverge only across a directory
		// boundary; without this pair the sort is untestable.
		"agents/x-y.md":      "See `## Goal`.\n",
		"agents/x/z.md":      "See `## Goal`.\n",
		"agents/reviewer.md": "Read `## Summary`.\n",
	}
	maps.Copy(base, overrides)

	for rel, content := range base {
		full := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), filepath.FromSlash(rel))
		if content == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}

func TestMilestoneSectionNameResolution_Fires(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		overrides map[string]string
		// wantDetail is a phrase the first violation's Detail must contain.
		// Empty means the fixture must report no violation at all.
		wantDetail string
	}{
		{
			name:      "clean tree reports nothing",
			overrides: nil,
		},
		{
			name: "a surface naming a section no artefact carries is reported",
			overrides: map[string]string{
				"agents/builder.md": "Maintain `## Goal` and `## Invented Section`.\n",
			},
			wantDetail: "## Invented Section",
		},
		{
			// One backtick span may list several sections. Read whole it yields
			// one name matching nothing and checks neither real one.
			name: "each name in a multi-name span is resolved separately",
			overrides: map[string]string{
				"agents/builder.md": "Fill `## Goal / ## Release note`.\n",
			},
		},
		{
			// Only `/ ##` separates two names; a slash inside one does not.
			name: "a section name containing a slash is not split",
			overrides: map[string]string{
				"templates/milestone-spec.md": "# T\n\n## Goal\n\n## Release note\n\n## Client/server split\n",
				"agents/builder.md":           "Fill `## Client/server split`.\n",
			},
		},
		{
			// The mention side trims too: a backtick span may carry trailing
			// space, and the heading it names cannot.
			name: "a mention's trailing whitespace is trimmed",
			overrides: map[string]string{
				"agents/builder.md": "Fill `## Goal `.\n",
			},
		},
		{
			name: "a bad name inside a multi-name span is still caught",
			overrides: map[string]string{
				"agents/builder.md": "Fill `## Goal / ## Invented Section`.\n",
			},
			wantDetail: "## Invented Section",
		},
		{
			// A blank fence has no opening line to match. Asserting only that
			// some name is unresolved would hold whichever fence were picked;
			// what must hold is that the search continued to the real scaffold.
			name: "a blank fence before the scaffold does not displace it",
			overrides: map[string]string{
				"skills/aiwfx-wrap-epic/SKILL.md": "# wrap\n\n```markdown\n\n```\n\n" +
					"```markdown\n# Epic wrap\n\n## Changelog entry\n\n## Summary\n```\n\n" +
					"See `## Changelog entry` and `## Summary`.\n",
			},
		},
		{
			// A fence that merely discusses the artefact mentions the marker in
			// prose. Selecting it would resolve every real section nowhere.
			name: "an earlier fence mentioning the marker does not displace the scaffold",
			overrides: map[string]string{
				"skills/aiwfx-wrap-epic/SKILL.md": "# wrap\n\n```markdown\nTalking about the `# Epic wrap` artefact.\n```\n\n" +
					"```markdown\n# Epic wrap\n\n## Changelog entry\n\n## Summary\n```\n\nSee `## Changelog entry` and `## Summary`.\n",
			},
		},
		{
			// An unrelated markdown example earlier in the ritual must not be
			// mistaken for the scaffold; the real one is found by its marker.
			name: "an earlier markdown example does not displace the scaffold",
			overrides: map[string]string{
				"skills/aiwfx-wrap-epic/SKILL.md": "# wrap\n\n```markdown\n## Unrelated Example\n```\n\n" +
					"```markdown\n# Epic wrap\n\n## Changelog entry\n\n## Summary\n```\n\nSee `## Changelog entry` and `## Summary`.\n",
			},
		},
		{
			name: "an absent templates directory is reported once",
			overrides: map[string]string{
				"templates/milestone-spec.md": "",
				"templates/epic-spec.md":      "",
			},
			wantDetail: "section-name universe is unreadable",
		},
		{
			name: "a missing wrap ritual is reported as an unreadable universe",
			overrides: map[string]string{
				"skills/aiwfx-wrap-epic/SKILL.md": "",
			},
			wantDetail: "section-name universe is unreadable",
		},
		{
			name: "a wrap ritual with no scaffold fence contributes no artefact sections",
			overrides: map[string]string{
				"skills/aiwfx-wrap-epic/SKILL.md": "# wrap\n\nNo fence here. See `## Summary`.\n",
			},
			wantDetail: "## Summary",
		},
		{
			// An unterminated fence that is not the scaffold contributes nothing.
			// The name asserted is the one that fence would have supplied, so
			// the case fails if it is read to end-of-file as the scaffold —
			// asserting an unrelated unresolved name would hold either way.
			name: "an unterminated non-scaffold fence yields no sections",
			overrides: map[string]string{
				"skills/aiwfx-wrap-epic/SKILL.md": "# wrap\n\n```markdown\n## Unrelated Example\n",
				"agents/builder.md":               "See `## Unrelated Example`.\n",
			},
			wantDetail: "## Unrelated Example",
		},
		{
			name: "an unterminated scaffold fence still yields its sections",
			overrides: map[string]string{
				"skills/aiwfx-wrap-epic/SKILL.md": "# wrap\n\n```markdown\n# Epic wrap\n\n## Summary\n\nSee `## Summary`.\n",
			},
		},
		{
			name: "a deeper heading in a template is not a top-level section name",
			overrides: map[string]string{
				"templates/milestone-spec.md": "# T\n\n## Goal\n\n### AC-1 — x\n",
				"agents/builder.md":           "Maintain `## AC-1 — x`.\n",
			},
			wantDetail: "## AC-1 — x",
		},
		{
			name: "non-markdown entries in the templates directory are skipped",
			overrides: map[string]string{
				"templates/notes.txt": "## Not A Section\n",
				"agents/builder.md":   "Maintain `## Not A Section`.\n",
			},
			wantDetail: "## Not A Section",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := sectionFixtureRoot(t, tc.overrides)
			vs, err := PolicyMilestoneSectionNameResolution(root)
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			if tc.wantDetail == "" {
				if len(vs) != 0 {
					t.Fatalf("want no violations, got %d: %+v", len(vs), vs)
				}
				return
			}
			if len(vs) == 0 {
				t.Fatalf("want a violation containing %q, got none", tc.wantDetail)
			}
			var found bool
			for _, v := range vs {
				if v.Policy != "milestone-section-name-resolution" {
					t.Errorf("violation carries policy %q, want milestone-section-name-resolution", v.Policy)
				}
				if strings.Contains(v.Detail, tc.wantDetail) {
					found = true
				}
			}
			if !found {
				t.Fatalf("no violation Detail contains %q; got %+v", tc.wantDetail, vs)
			}
		})
	}
}

// TestMilestoneSectionNameResolution_ReportsTheLine pins that a violation
// carries the line the offending mention sits on, so the report names where to
// look rather than only what is wrong.
func TestMilestoneSectionNameResolution_ReportsTheLine(t *testing.T) {
	t.Parallel()
	root := sectionFixtureRoot(t, map[string]string{
		"agents/builder.md": "first line\n\nMaintain `## Invented Section`.\n",
	})
	vs, err := PolicyMilestoneSectionNameResolution(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("want exactly 1 violation, got %d: %+v", len(vs), vs)
	}
	if vs[0].Line != 3 {
		t.Errorf("violation Line = %d, want 3", vs[0].Line)
	}
	if want := "internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/builder.md"; vs[0].File != want {
		t.Errorf("violation File = %q, want %q", vs[0].File, want)
	}
}

// TestMilestoneSectionNameResolution_UnreadableTemplateIsReported pins the
// branch where ReadDir lists a template that ReadFile then cannot open — a
// dangling symlink here, a permission change or a concurrent delete in the
// wild. The universe is incomplete in that case, so every name that would have
// resolved against the missing template reads as unresolved; reporting the read
// failure instead is what keeps the output about the real fault.
func TestMilestoneSectionNameResolution_UnreadableTemplateIsReported(t *testing.T) {
	t.Parallel()
	root := sectionFixtureRoot(t, nil)
	dangling := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), "templates", "dangling.md")
	if err := os.Symlink(filepath.Join(root, "no-such-target.md"), dangling); err != nil {
		t.Skipf("symlinks unavailable on this filesystem: %v", err)
	}
	vs, err := PolicyMilestoneSectionNameResolution(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("want exactly 1 violation, got %d: %+v", len(vs), vs)
	}
	if !strings.Contains(vs[0].Detail, "section-name universe is unreadable") {
		t.Errorf("Detail = %q, want it to name the unreadable universe", vs[0].Detail)
	}
	if !strings.Contains(vs[0].Detail, "dangling.md") {
		t.Errorf("Detail = %q, want it to name the template that could not be read", vs[0].Detail)
	}
}

// TestPolicy_ReleaseNoteHeadingResolves pins on the live tree that the heading
// the kernel's release-note rule reads is one the shipped milestone template
// carries. The failure it guards against is silent: a coherent rename moves both
// sides of the sibling name-resolution check together and passes it, while the
// kernel rule is left reading a heading no spec will carry again.
func TestPolicy_ReleaseNoteHeadingResolves(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyReleaseNoteHeadingResolves)
}

func TestReleaseNoteHeadingResolves_Fires(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "a coherent rename away from the constant is reported",
			template: "# T\n\n## Goal\n\n## Release summary\n",
			want:     "ships no \"## Release note\" heading",
		},
		{
			name:     "an unreadable template is reported",
			template: "",
			want:     "milestone template is unreadable",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := sectionFixtureRoot(t, map[string]string{"templates/milestone-spec.md": tc.template})
			vs, err := PolicyReleaseNoteHeadingResolves(root)
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			if len(vs) != 1 {
				t.Fatalf("want exactly 1 violation, got %d: %+v", len(vs), vs)
			}
			if !strings.Contains(vs[0].Detail, tc.want) {
				t.Errorf("Detail = %q, want it to contain %q", vs[0].Detail, tc.want)
			}
		})
	}
}

// TestReleaseNoteHeadingResolves_CleanTemplatePasses is the regression
// companion: the heading present means no violation.
func TestReleaseNoteHeadingResolves_CleanTemplatePasses(t *testing.T) {
	t.Parallel()
	root := sectionFixtureRoot(t, nil)
	vs, err := PolicyReleaseNoteHeadingResolves(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 0 {
		t.Fatalf("want no violations, got %+v", vs)
	}
}

// TestMilestoneSectionNameResolution_UnwalkableTreeIsReported pins the walk's
// own error path: with no authoring tree at all there is no surface to check,
// and the policy says so rather than reporting a clean scan.
func TestMilestoneSectionNameResolution_UnwalkableTreeIsReported(t *testing.T) {
	t.Parallel()
	vs, err := PolicyMilestoneSectionNameResolution(t.TempDir())
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("want exactly 1 violation, got %d: %+v", len(vs), vs)
	}
	if !strings.Contains(vs[0].Detail, "authoring tree is unwalkable") {
		t.Errorf("Detail = %q, want it to name the unwalkable tree", vs[0].Detail)
	}
}

// TestMilestoneSectionNameResolution_UnreadableSurfaceIsReported pins the
// per-surface read failure: the walk lists a file the read then cannot open — a
// dangling symlink here, a permission change or a concurrent delete in the wild.
func TestMilestoneSectionNameResolution_UnreadableSurfaceIsReported(t *testing.T) {
	t.Parallel()
	root := sectionFixtureRoot(t, nil)
	dangling := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), "agents", "gone.md")
	if err := os.Symlink(filepath.Join(root, "no-such-target.md"), dangling); err != nil {
		t.Skipf("symlinks unavailable on this filesystem: %v", err)
	}
	vs, err := PolicyMilestoneSectionNameResolution(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("want exactly 1 violation, got %d: %+v", len(vs), vs)
	}
	if !strings.Contains(vs[0].Detail, "surface is unreadable") {
		t.Errorf("Detail = %q, want it to name the unreadable surface", vs[0].Detail)
	}
}

// TestMilestoneSectionNameResolution_DirectoryNamedMarkdownIsSkipped reaches the
// IsDir arm of the templates skip condition. A subdirectory with an ordinary
// name is rejected by the `.md` suffix test first, so only a directory *named*
// `*.md` decides on IsDir — without this the arm reads as covered while nothing
// exercises it.
func TestMilestoneSectionNameResolution_DirectoryNamedMarkdownIsSkipped(t *testing.T) {
	t.Parallel()
	root := sectionFixtureRoot(t, nil)
	trap := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), "templates", "trap.md")
	if err := os.MkdirAll(trap, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	vs, err := PolicyMilestoneSectionNameResolution(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 0 {
		t.Fatalf("a directory named *.md must be skipped, not read; got %+v", vs)
	}
}

// TestMilestoneSectionNameResolution_NonMarkdownSurfaceIsNotScanned pins the
// suffix arm of the surface walk: a non-markdown file naming a section that
// resolves nowhere must not be reported, because it is not a surface.
func TestMilestoneSectionNameResolution_NonMarkdownSurfaceIsNotScanned(t *testing.T) {
	t.Parallel()
	root := sectionFixtureRoot(t, nil)
	notes := filepath.Join(root, filepath.FromSlash(sectionRitualsDir), "agents", "notes.txt")
	if err := os.WriteFile(notes, []byte("Fill `## Invented Section`.\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	vs, err := PolicyMilestoneSectionNameResolution(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 0 {
		t.Fatalf("a non-markdown file is not a surface; got %+v", vs)
	}
}

// TestMilestoneSectionNameResolution_HeadingWhitespaceIsTrimmed pins the trim in
// topLevelHeadings. A template heading with trailing whitespace would otherwise
// enter the universe carrying it, and every mention of that section — which
// cannot carry the whitespace, being written inside backticks — would stop
// resolving.
func TestMilestoneSectionNameResolution_HeadingWhitespaceIsTrimmed(t *testing.T) {
	t.Parallel()
	root := sectionFixtureRoot(t, map[string]string{
		"templates/milestone-spec.md": "# T\n\n## Goal\n\n## Release note   \n",
	})
	vs, err := PolicyMilestoneSectionNameResolution(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 0 {
		t.Fatalf("a heading's trailing whitespace must not break resolution; got %+v", vs)
	}
}

// TestSectionSurfaces_IsOrdered pins the sort. Violation order follows surface
// order, so an unsorted walk makes the report order depend on directory
// iteration and turns any future golden comparison flaky.
func TestSectionSurfaces_IsOrdered(t *testing.T) {
	t.Parallel()
	got, err := sectionSurfaces(sectionFixtureRoot(t, nil))
	if err != nil {
		t.Fatalf("sectionSurfaces: %v", err)
	}
	if !sort.StringsAreSorted(got) {
		t.Errorf("surfaces are not sorted: %v", got)
	}
}

// TestReleaseNoteHeadingResolves_MatchesTheRulesOwnComparison pins that this
// policy compares the way the rule does. The rule resolves the section by slug,
// so a template heading differing only in case is not a drift — reporting it
// would send the operator to rename a constant that already matches.
func TestReleaseNoteHeadingResolves_MatchesTheRulesOwnComparison(t *testing.T) {
	t.Parallel()
	root := sectionFixtureRoot(t, map[string]string{
		"templates/milestone-spec.md": "# T\n\n## Goal\n\n## Release Note\n",
	})
	vs, err := PolicyReleaseNoteHeadingResolves(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 0 {
		t.Fatalf("a case difference resolves under the rule's own slug match; got %+v", vs)
	}
}
