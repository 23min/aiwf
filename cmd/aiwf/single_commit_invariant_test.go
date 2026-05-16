package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// M-069 AC-2 — Single-commit-per-verb invariant asserted per
// mutating verb.
//
// CLAUDE.md design decision §7: "Every mutating verb produces
// exactly one git commit. That gives per-mutation atomicity for
// free." This is one of the load-bearing properties any change must
// preserve. The audit closed via G-051 ("Planning sessions emit one
// commit per entity, not per logical mutation") was the user-visible
// symptom of an earlier era when this invariant was not enforced.
//
// `TestBinary_MutatingVerbs_Subprocess` already runs every migrated
// mutating verb as a subprocess sequence and asserts each invocation
// exits cleanly. It does *not* assert the commit-count delta per
// verb. A regression where `aiwf promote` started emitting two
// commits (one for the entity, one for a side-effect projection) — or
// where `aiwf cancel` emitted zero commits and stamped its mutation
// as part of the *next* verb's commit — would still pass that test.
// The kernel's atomicity guarantee, the property `aiwf history`
// projects against, and the per-mutation rollback story all depend
// on this delta being exactly 1.
//
// This test drives every user-facing mutating verb through the
// in-process dispatcher (`run([]string{...})`), records
// `git rev-list --count HEAD` before and after, and asserts strict
// equality `delta == 1`. Strict equality catches both ends of the
// regression class: a verb that silently produces a *second* commit
// (an audit-trail commit, a projection-rebuild commit) and a verb
// that emits *zero* commits and defers its mutation to the next
// verb's commit.
//
// Coverage: `add` (each kind), `promote` (entity status, AC status,
// AC tdd_phase), `rename`, `edit-body`, `move`, `cancel`, `authorize`
// (open / pause / resume), `import` (default bundled-commit mode —
// multi-entity manifest must still be one commit), `reallocate`, and
// the `contract` family (`recipe install`, `bind`, `unbind`, `recipe
// remove`). Adding a new mutating verb without a row here is the
// regression this test surfaces.

// TestSingleCommitPerMutatingVerb_Invariant (M-069 AC-2) walks a
// representative consumer-repo lifecycle through the in-process
// dispatcher, asserting each verb invocation grows HEAD by exactly
// one commit. A pre-step "init" prepares scaffolding (uncommitted by
// design — `aiwf init` writes files and defers the first commit to
// the user) so the first commit-producing verb is `add epic`.
func TestSingleCommitPerMutatingVerb_Invariant(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)

	// init writes scaffolding without producing a commit. Track that
	// explicitly so the first mutating verb's delta is measured from
	// "0 commits", not from "init's commit".
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if n := commitCountSafe(t, root); n != 0 {
		t.Fatalf("aiwf init must not produce a commit (CLAUDE.md: scaffolding writes files; user commits aiwf.yaml when ready); got %d commits at HEAD", n)
	}

	// A body-file for the edit-body step. Written once up front so the
	// step itself just runs the verb.
	bodyFile := filepath.Join(root, "fixtures-edit-body.md")
	if err := os.WriteFile(bodyFile, []byte("## Goal\n\nReplaced via single-commit invariant test.\n"), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	// A multi-entity manifest for the import step. Default mode must
	// produce ONE commit even with N entities — that's the audit's
	// namesake gap (G-051).
	manifest := writeManifest(t, root, `version: 1
actor: human/test
entities:
  - kind: gap
    id: auto
    frontmatter:
      title: Imported sample gap A
      status: open
  - kind: gap
    id: auto
    frontmatter:
      title: Imported sample gap B
      status: open
  - kind: decision
    id: auto
    frontmatter:
      title: Imported sample decision
      status: proposed
`)

	type step struct {
		name string
		args []string
	}
	steps := []step{
		// add — every kind emits one commit per invocation.
		{"add epic E-01 (decoy)", []string{"add", "epic", "--title", "Decoy", "--actor", "human/test", "--root", root}},
		{"add epic E-02 (engine)", []string{"add", "epic", "--title", "Engine", "--actor", "human/test", "--root", root}},
		{"add milestone M-001 (tdd none)", []string{"add", "milestone", "--tdd", "none", "--epic", "E-0002", "--title", "Cache", "--actor", "human/test", "--root", root}},
		{"add ac M-001/AC-1", []string{"add", "ac", "M-0001", "--title", "AC: warm-up works", "--actor", "human/test", "--root", root}},
		{"add gap G-001", []string{"add", "gap", "--title", "Sample gap", "--actor", "human/test", "--root", root}},
		{"add adr ADR-0001", []string{"add", "adr", "--title", "Sample ADR", "--actor", "human/test", "--root", root}},
		{"add decision D-001", []string{"add", "decision", "--title", "Sample decision", "--actor", "human/test", "--root", root}},
		{"add milestone M-002 (tdd required)", []string{"add", "milestone", "--tdd", "required", "--epic", "E-0002", "--title", "Strict", "--actor", "human/test", "--root", root}},
		{"add ac M-002/AC-1 (under tdd-required)", []string{"add", "ac", "M-0002", "--title", "AC: under tdd required", "--actor", "human/test", "--root", root}},

		// promote — entity status, AC status, AC tdd_phase.
		{"promote E-02 → active", []string{"promote", "E-0002", "active", "--actor", "human/test", "--root", root}},
		{"promote M-001 → in_progress", []string{"promote", "M-0001", "in_progress", "--actor", "human/test", "--root", root}},
		{"promote M-001/AC-1 → met", []string{"promote", "M-0001/AC-1", "met", "--actor", "human/test", "--root", root}},
		{"promote M-002/AC-1 phase → green", []string{"promote", "M-0002/AC-1", "--phase", "green", "--tests", "pass=1 fail=0 skip=0", "--actor", "human/test", "--root", root}},

		// rename — slug-only mutation; id preserved.
		{"rename E-02 → engine-renamed", []string{"rename", "E-0002", "engine-renamed", "--actor", "human/test", "--root", root}},

		// edit-body --body-file (explicit mode; bless mode is exercised
		// elsewhere — its delta is also 1, but the invariant under test
		// is the same).
		{"edit-body M-001 --body-file", []string{"edit-body", "M-0001", "--body-file", bodyFile, "--reason", "single-commit test", "--actor", "human/test", "--root", root}},

		// move — reparent milestone to a different epic.
		{"move M-002 --epic E-01", []string{"move", "M-0002", "--epic", "E-0001", "--actor", "human/test", "--root", root}},

		// authorize lifecycle on the active epic. open / pause / resume
		// are each their own verb invocation and each must be one commit.
		{"authorize E-02 --to ai/claude", []string{"authorize", "E-0002", "--to", "ai/claude", "--actor", "human/test", "--root", root}},
		{"authorize E-02 --pause", []string{"authorize", "E-0002", "--pause", "blocked on review", "--actor", "human/test", "--root", root}},
		{"authorize E-02 --resume", []string{"authorize", "E-0002", "--resume", "review unblocked", "--actor", "human/test", "--root", root}},

		// import — default mode is "bundled commit": N entities = 1 commit.
		// This is the load-bearing case G-051 was about.
		{"import (default bundled mode, 3 entities)", []string{"import", manifest, "--actor", "human/test", "--root", root}},

		// cancel — terminates the in-progress milestone.
		{"cancel M-001", []string{"cancel", "M-0001", "--reason", "test cleanup", "--actor", "human/test", "--root", root}},

		// reallocate — renumber an entity (id collision recovery surface,
		// invoked here on a non-colliding entity to exercise the verb's
		// commit shape).
		{"reallocate E-02", []string{"reallocate", "E-0002", "--actor", "human/test", "--root", root}},

		// contract family — recipe install / bind / unbind / recipe remove.
		{"add contract C-001", []string{"add", "contract", "--title", "Sample API contract", "--actor", "human/test", "--root", root}},
		{"contract recipe install jsonschema", []string{"contract", "recipe", "install", "jsonschema", "--actor", "human/test", "--root", root}},
		// `contract bind` needs concrete schema + fixtures paths to
		// validate against. Plant minimal placeholders before the verb.
		{"plant schema + fixtures (test setup, not a verb)", nil},
		{"contract bind C-001", []string{"contract", "bind", "C-0001", "--validator", "jsonschema", "--schema", "fixtures-contract-schema.json", "--fixtures", "fixtures-contract-data", "--actor", "human/test", "--root", root}},
		{"contract unbind C-001", []string{"contract", "unbind", "C-0001", "--actor", "human/test", "--root", root}},
		{"contract recipe remove jsonschema", []string{"contract", "recipe", "remove", "jsonschema", "--actor", "human/test", "--root", root}},
	}

	for _, s := range steps {
		t.Run(s.name, func(t *testing.T) {
			// Special non-verb step: plant schema + fixtures so contract
			// bind has something concrete to record. Counted as zero-commit
			// (it's working-tree setup, not a kernel mutation).
			if s.args == nil && s.name == "plant schema + fixtures (test setup, not a verb)" {
				schemaPath := filepath.Join(root, "fixtures-contract-schema.json")
				if err := os.WriteFile(schemaPath, []byte(`{"type":"object"}`), 0o644); err != nil {
					t.Fatalf("write schema: %v", err)
				}
				fixturesDir := filepath.Join(root, "fixtures-contract-data")
				if err := os.MkdirAll(fixturesDir, 0o755); err != nil {
					t.Fatalf("mkdir fixtures: %v", err)
				}
				if err := os.WriteFile(filepath.Join(fixturesDir, "sample.json"), []byte(`{}`), 0o644); err != nil {
					t.Fatalf("write fixture: %v", err)
				}
				return
			}

			before := commitCountSafe(t, root)
			if rc := run(s.args); rc != cliutil.ExitOK {
				t.Fatalf("verb %v rc = %d (want cliutil.ExitOK)", s.args, rc)
			}
			after := commitCountSafe(t, root)
			delta := after - before
			if delta != 1 {
				t.Errorf("verb %q produced %d commit(s), want exactly 1 (CLAUDE.md §7: every mutating verb produces exactly one git commit)\n  args: %v\n  before HEAD count: %d\n  after  HEAD count: %d", s.name, delta, s.args, before, after)
			}
		})
	}
}

// commitCountSafe returns the number of commits reachable from HEAD,
// or 0 when HEAD doesn't yet exist (a fresh repo with no commits
// makes `git rev-list --count HEAD` exit 128). Differs from
// commitCount in import_cmd_test.go which propagates that error and
// is unsuitable for the AC-2 invariant test, where the very first
// step in the sequence runs against a 0-commit repo.
func commitCountSafe(t *testing.T, root string) int {
	t.Helper()
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Treat "unknown revision HEAD" as 0 commits. Any other git
		// failure is a real test environment problem — surface it.
		if strings.Contains(string(out), "unknown revision") || strings.Contains(string(out), "ambiguous argument 'HEAD'") {
			return 0
		}
		t.Fatalf("git rev-list --count HEAD: %v\n%s", err, out)
	}
	var n int
	for _, c := range strings.TrimSpace(string(out)) {
		if c < '0' || c > '9' {
			t.Fatalf("git rev-list --count HEAD: non-numeric output %q", out)
		}
		n = n*10 + int(c-'0')
	}
	return n
}
