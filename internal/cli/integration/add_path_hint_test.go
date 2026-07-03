package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// setupPathAreaRepo initializes a repo whose areas declare `paths:` globs (the
// object form) — the oracle `aiwf add --path-hint` derivation (M-0182) reads.
func setupPathAreaRepo(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) +
		"areas:\n" +
		"  members:\n" +
		"    - {name: platform, paths: [projects/platform/**]}\n" +
		"    - {name: billing, paths: [projects/billing/**]}\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	return root
}

// TestRunAdd_PathHintDerivesArea pins M-0182/AC-4: with --area omitted, a
// --path-hint falling under exactly one declared area's paths derives that
// area into the created entity's frontmatter; a hint matching no area leaves
// the entity untagged. Driven through the real dispatcher so the
// deriveAreaFromHint seam (not just areamatch.Derive) is exercised.
func TestRunAdd_PathHintDerivesArea(t *testing.T) {
	t.Run("single unambiguous hint derives area", func(t *testing.T) {
		root := setupPathAreaRepo(t)
		mustRun(t, "add", "epic", "--title", "Platform work",
			"--path-hint", "projects/platform/auth/login.go",
			"--actor", "human/test", "--root", root)
		fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md"))
		if !strings.Contains(fm, "area: platform") {
			t.Errorf("epic frontmatter missing derived `area: platform`:\n%s", fm)
		}
	})

	t.Run("hint matching no area leaves entity untagged", func(t *testing.T) {
		root := setupPathAreaRepo(t)
		mustRun(t, "add", "epic", "--title", "Orphan work",
			"--path-hint", "services/unmapped/x.go",
			"--actor", "human/test", "--root", root)
		fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md"))
		if strings.Contains(fm, "area:") {
			t.Errorf("expected untagged epic for a non-matching hint, got area:\n%s", fm)
		}
	})
}

// setupOverlapPathAreaRepo declares two areas whose paths claim the SAME glob,
// so a hint under it is genuinely ambiguous (the M-0182/AC-6 multi-match case).
func setupOverlapPathAreaRepo(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) +
		"areas:\n" +
		"  members:\n" +
		"    - {name: platform, paths: [projects/shared/**]}\n" +
		"    - {name: billing, paths: [projects/shared/**]}\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	return root
}

// TestRunAdd_PathHintAmbiguousSuggests pins M-0182/AC-6: a --path-hint that
// matches zero or several declared areas does not set area — the entity is
// created untagged and a suggestion is printed to stderr (the candidate areas,
// or "matches no declared area"). Driven through the dispatcher so the stderr
// output is captured, not just the silent untagged result.
func TestRunAdd_PathHintAmbiguousSuggests(t *testing.T) {
	t.Run("zero match suggests and leaves untagged", func(t *testing.T) {
		root := setupPathAreaRepo(t)
		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{
				"add", "epic", "--title", "Orphan",
				"--path-hint", "services/unmapped/x.go", "--actor", "human/test", "--root", root,
			})
		})
		if rc != cliutil.ExitOK {
			t.Errorf("rc = %d, want ExitOK (untagged create, not refusal)", rc)
		}
		if !strings.Contains(stderr, "services/unmapped/x.go") || !strings.Contains(stderr, "no declared area") {
			t.Errorf("stderr should name the hint and explain no area claims it:\n%s", stderr)
		}
		if fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md")); strings.Contains(fm, "area:") {
			t.Errorf("expected untagged epic, got area:\n%s", fm)
		}
	})

	t.Run("multi match lists candidates and leaves untagged", func(t *testing.T) {
		root := setupOverlapPathAreaRepo(t)
		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{
				"add", "epic", "--title", "Shared",
				"--path-hint", "projects/shared/x.go", "--actor", "human/test", "--root", root,
			})
		})
		if rc != cliutil.ExitOK {
			t.Errorf("rc = %d, want ExitOK (untagged create, not refusal)", rc)
		}
		for _, want := range []string{"ambiguous", "platform", "billing"} {
			if !strings.Contains(stderr, want) {
				t.Errorf("stderr %q missing %q (should list the candidate areas)", stderr, want)
			}
		}
		if fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md")); strings.Contains(fm, "area:") {
			t.Errorf("expected untagged epic for an ambiguous hint, got area:\n%s", fm)
		}
	})
}

// TestRunAdd_PathHintInertWithoutPaths pins M-0182/AC-7: when no declared area
// carries a `paths:` glob (here the legacy string form), --path-hint has no
// oracle — it derives nothing and emits a note so the operator knows the hint
// was ignored, rather than silently doing nothing.
func TestRunAdd_PathHintInertWithoutPaths(t *testing.T) {
	root := setupAreaRepo(t) // string-form members: platform, billing — no paths
	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{
			"add", "epic", "--title", "No oracle",
			"--path-hint", "projects/platform/x.go", "--actor", "human/test", "--root", root,
		})
	})
	if rc != cliutil.ExitOK {
		t.Errorf("rc = %d, want ExitOK", rc)
	}
	if !strings.Contains(stderr, "no declared area has a paths") {
		t.Errorf("stderr should note the absent paths oracle:\n%s", stderr)
	}
	if fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md")); strings.Contains(fm, "area:") {
		t.Errorf("expected untagged epic when no paths declared, got area:\n%s", fm)
	}
}

// TestRunAdd_PathHintAmbiguousFallsBackToDiscoveredIn pins the precedence and
// the honest-message fix (M-0182): when a gap's --path-hint is ambiguous,
// derivation sets nothing and the message says "no area derived" — NOT "left
// untagged" — because the --discovered-in fallback then tags the gap from its
// source. The suggestion must describe the hint outcome, not the entity's
// final state.
func TestRunAdd_PathHintAmbiguousFallsBackToDiscoveredIn(t *testing.T) {
	root := setupOverlapPathAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Source", "--area", "platform", "--actor", "human/test", "--root", root)
	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{
			"add", "gap", "--body", "## What's missing\n\nFixture prose for test setup; not the subject under test.\n\n## Why it matters\n\nFixture prose for test setup; not the subject under test.\n", "--title", "Leak",
			"--discovered-in", "E-0001", "--path-hint", "projects/shared/x.go",
			"--actor", "human/test", "--root", root,
		})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("rc = %d, want ExitOK", rc)
	}
	if strings.Contains(stderr, "left untagged") {
		t.Errorf("message must not claim 'left untagged' — the gap is tagged via --discovered-in:\n%s", stderr)
	}
	if !strings.Contains(stderr, "no area derived") || !strings.Contains(stderr, "ambiguous") {
		t.Errorf("message should describe the hint outcome ('no area derived', ambiguous):\n%s", stderr)
	}
	if fm := frontmatterOf(readOne(t, root, "work/gaps/G-*.md")); !strings.Contains(fm, "area: platform") {
		t.Errorf("gap should be tagged platform via the --discovered-in fallback:\n%s", fm)
	}
}

// TestRunAdd_PathHintIgnoredOnMilestone pins the milestone consistency fix
// (M-0182): --path-hint on a milestone is ignored with a note (a milestone's
// area derives from its parent epic), rather than silently doing nothing — the
// same don't-silently-ignore principle AC-7 applies to the no-paths case.
func TestRunAdd_PathHintIgnoredOnMilestone(t *testing.T) {
	root := setupPathAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Parent", "--area", "platform", "--actor", "human/test", "--root", root)
	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{
			"add", "milestone", "--epic", "E-0001", "--tdd", "none",
			"--title", "Child", "--path-hint", "projects/platform/y.go",
			"--actor", "human/test", "--root", root,
		})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("rc = %d, want ExitOK (milestone created, hint ignored)", rc)
	}
	if !strings.Contains(stderr, "--path-hint ignored") || !strings.Contains(stderr, "milestone") {
		t.Errorf("stderr should note the hint was ignored for a milestone:\n%s", stderr)
	}
	if matches, _ := filepath.Glob(filepath.Join(root, "work", "epics", "E-*", "M-*.md")); len(matches) != 1 {
		t.Errorf("milestone should be created; found %v", matches)
	}
}

// TestRunAdd_PathHintNormalized pins the second-review normalization fixes
// (M-0182): a hint with ./ or .. segments is path.Clean'd before matching — so
// it can't lexically derive a confidently-wrong area — and an absolute path
// under the repo root is relativized first (the LLM primary user carries
// absolute paths). All three derive the area the resolved path actually lives in.
func TestRunAdd_PathHintNormalized(t *testing.T) {
	t.Run("dot-dot segment resolves before matching (no wrong-area derive)", func(t *testing.T) {
		root := setupPathAreaRepo(t)
		// Lexically under platform's glob, but resolves into billing's tree.
		mustRun(t, "add", "epic", "--title", "Dotdot",
			"--path-hint", "projects/platform/../billing/x.go",
			"--actor", "human/test", "--root", root)
		fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md"))
		if !strings.Contains(fm, "area: billing") {
			t.Errorf("`..` should resolve to billing before matching, got:\n%s", fm)
		}
	})

	t.Run("leading ./ is collapsed and still derives", func(t *testing.T) {
		root := setupPathAreaRepo(t)
		mustRun(t, "add", "epic", "--title", "Dotslash",
			"--path-hint", "./projects/platform/x.go",
			"--actor", "human/test", "--root", root)
		fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md"))
		if !strings.Contains(fm, "area: platform") {
			t.Errorf("leading ./ should still derive platform, got:\n%s", fm)
		}
	})

	t.Run("absolute path under the repo root is relativized and derives", func(t *testing.T) {
		root := setupPathAreaRepo(t)
		abs := filepath.Join(root, "projects", "platform", "deep", "x.go")
		mustRun(t, "add", "epic", "--title", "Absolute",
			"--path-hint", abs,
			"--actor", "human/test", "--root", root)
		fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md"))
		if !strings.Contains(fm, "area: platform") {
			t.Errorf("absolute path under root should relativize and derive platform, got:\n%s", fm)
		}
	})
}

// TestRunAdd_PathHintConflictWithExplicitArea pins M-0182/AC-5: an explicit
// --area always wins, but a --path-hint that unambiguously points to a
// DIFFERENT area is reported (a cheap at-add mistag-prevention signal) without
// overriding the explicit choice; an agreeing hint is silent.
func TestRunAdd_PathHintConflictWithExplicitArea(t *testing.T) {
	t.Run("conflicting hint is reported, --area still wins", func(t *testing.T) {
		root := setupPathAreaRepo(t)
		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{
				"add", "epic", "--title", "Explicit",
				"--area", "billing", "--path-hint", "projects/platform/x.go",
				"--actor", "human/test", "--root", root,
			})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc = %d, want ExitOK", rc)
		}
		for _, want := range []string{"overrides", "billing", "platform"} {
			if !strings.Contains(stderr, want) {
				t.Errorf("conflict note missing %q:\n%s", want, stderr)
			}
		}
		if fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md")); !strings.Contains(fm, "area: billing") {
			t.Errorf("--area must win over the hint; want area: billing:\n%s", fm)
		}
	})

	t.Run("agreeing hint is silent", func(t *testing.T) {
		root := setupPathAreaRepo(t)
		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{
				"add", "epic", "--title", "Agree",
				"--area", "platform", "--path-hint", "projects/platform/x.go",
				"--actor", "human/test", "--root", root,
			})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc = %d, want ExitOK", rc)
		}
		if strings.Contains(stderr, "overrides") {
			t.Errorf("an agreeing hint should be silent, got:\n%s", stderr)
		}
		if fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md")); !strings.Contains(fm, "area: platform") {
			t.Errorf("want area: platform:\n%s", fm)
		}
	})

	t.Run("ambiguous hint with explicit --area is silent", func(t *testing.T) {
		root := setupOverlapPathAreaRepo(t) // platform + billing both claim projects/shared/**
		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{
				"add", "epic", "--title", "AmbWithArea",
				"--area", "platform", "--path-hint", "projects/shared/x.go",
				"--actor", "human/test", "--root", root,
			})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc = %d, want ExitOK", rc)
		}
		if strings.Contains(stderr, "overrides") {
			t.Errorf("an ambiguous hint must not produce a single-area conflict note:\n%s", stderr)
		}
		if fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md")); !strings.Contains(fm, "area: platform") {
			t.Errorf("want area: platform:\n%s", fm)
		}
	})
}
