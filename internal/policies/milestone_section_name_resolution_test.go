package policies

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicy_MilestoneSectionNameResolution pins M-0326/AC-1 on the live tree:
// every backticked `## Section` name written in the surfaces that instruct an
// author about milestone-spec and wrap-artefact sections resolves to a heading
// some shipped template — or the wrap artefact's own scaffold — actually
// carries. A rename on either side of that relationship reddens this test,
// which is the evidence shape D-0070 leaves available over a shipped surface:
// prose is compared against the artefact it names, never asserted for its own
// wording.
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
			"## Changelog entry\n\n### Added\n\n## Summary\n```\n\nSee `## Changelog entry`. Append the report under a `## Doc findings` section.\n",
		"skills/aiwfx-start-milestone/SKILL.md": "Fill `## Goal`.\n",
		"skills/aiwfx-wrap-milestone/SKILL.md":  "Fill `## Release note`.\n",
		"agents/builder.md":                     "Maintain `## Goal`.\n",
		"agents/reviewer.md":                    "Read `## Summary`.\n",
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
			// The surface set is derived, so a file that is absent is simply not
			// walked. Nothing to report, and nothing silently unscanned either.
			name: "an absent file is not a surface",
			overrides: map[string]string{
				"agents/reviewer.md": "",
			},
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
			// The constant must be derived fact, not an exemption list: an entry
			// the ritual never creates silently widens the universe.
			name: "an unscaffolded-section entry with no creation site is reported",
			overrides: map[string]string{
				"skills/aiwfx-wrap-epic/SKILL.md": "# wrap\n\n```markdown\n# Epic wrap\n\n## Changelog entry\n\n## Summary\n```\n\nSee `## Changelog entry`.\n",
			},
			wantDetail: "an exemption that silently suppresses real findings",
		},
		{
			name: "a bad name inside a multi-name span is still caught",
			overrides: map[string]string{
				"agents/builder.md": "Fill `## Goal / ## Invented Section`.\n",
			},
			wantDetail: "## Invented Section",
		},
		{
			// An unrelated markdown example earlier in the ritual must not be
			// mistaken for the scaffold; the real one is found by its marker.
			name: "an earlier markdown example does not displace the scaffold",
			overrides: map[string]string{
				"skills/aiwfx-wrap-epic/SKILL.md": "# wrap\n\n```markdown\n## Unrelated Example\n```\n\n" +
					"```markdown\n# Epic wrap\n\n## Changelog entry\n\n## Summary\n```\n\nSee `## Changelog entry`, `## Summary` and `## Doc findings`.\n",
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
				"skills/aiwfx-wrap-epic/SKILL.md": "# wrap\n\nNo fence here. See `## Summary` and `## Doc findings`.\n",
			},
			wantDetail: "## Summary",
		},
		{
			// An unterminated fence that is not the scaffold contributes nothing,
			// rather than being read to end-of-file as if it were.
			name: "an unterminated non-scaffold fence yields no sections",
			overrides: map[string]string{
				"skills/aiwfx-wrap-epic/SKILL.md": "# wrap\n\n```markdown\n## Unrelated Example\n\nSee `## Summary` and `## Doc findings`.\n",
			},
			wantDetail: "## Summary",
		},
		{
			name: "an unterminated scaffold fence still yields its sections",
			overrides: map[string]string{
				"skills/aiwfx-wrap-epic/SKILL.md": "# wrap\n\n```markdown\n# Epic wrap\n\n## Summary\n\nSee `## Summary` and `## Doc findings`.\n",
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
			// Exercises the IsDir arm of the skip condition, which the
			// non-markdown case short-circuits past.
			name: "a subdirectory of the templates directory is skipped",
			overrides: map[string]string{
				"templates/sub/nested.md": "## Nested Only\n",
				"agents/builder.md":       "Maintain `## Nested Only`.\n",
			},
			wantDetail: "## Nested Only",
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

// TestMilestoneSectionNameResolution_UnscaffoldedWrapSectionsResolve pins that
// a wrap.md section the ritual creates outside its step-1 scaffold still
// resolves — `## Doc findings` is appended by the doc-lint sweep, so parsing
// the scaffold alone would report it.
func TestMilestoneSectionNameResolution_UnscaffoldedWrapSectionsResolve(t *testing.T) {
	t.Parallel()
	// Ranging over an empty slice would assert nothing, so the length is
	// checked before the loop rather than trusted.
	if len(wrapArtefactUnscaffoldedSections) == 0 {
		t.Fatal("no unscaffolded wrap sections declared; this test would assert nothing")
	}
	for _, name := range wrapArtefactUnscaffoldedSections {
		root := sectionFixtureRoot(t, map[string]string{
			"agents/builder.md": "Append the report under `## " + name + "`.\n",
		})
		vs, err := PolicyMilestoneSectionNameResolution(root)
		if err != nil {
			t.Fatalf("policy returned error: %v", err)
		}
		if len(vs) != 0 {
			t.Errorf("wrap section %q should resolve, got %+v", name, vs)
		}
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
