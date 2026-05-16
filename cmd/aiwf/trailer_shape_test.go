package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// M-069 AC-3 — Trailer-key shape asserted per mutating verb.
//
// `aiwf history` projects the per-entity timeline by reading
// `git log` filtered on structured commit trailers (`aiwf-verb`,
// `aiwf-entity`, `aiwf-actor`, plus the I2.5 provenance keys
// `aiwf-principal` / `aiwf-on-behalf-of` / `aiwf-authorized-by`).
// The trailer-key shape is the projection's only contract: a verb
// that forgets a key, types a key wrong (`aiwf_verb` snake-case,
// `aiwf-acto` truncated), or emits a brand-new key the canonical
// set in `internal/gitops/trailers.go` doesn't know about *silently*
// breaks `aiwf history`'s rendering of that entity's timeline, the
// provenance audit's authorized-by walk, and the policy-test catalog
// that greps for trailer values.
//
// Existing infrastructure protects only the *source side*:
//
//   - `PolicyTrailerKeysViaConstants` flags any production Go file
//     that string-literals a known trailer name instead of the
//     `gitops.Trailer*` constant.
//   - `PolicyIntegrationTestsAssertTrailers` flags integration tests
//     that drive a mutating verb without referencing the trailer-
//     assertion API.
//
// Both are static checks — they prevent source drift but say nothing
// about runtime behavior. A verb whose code wires
// `gitops.TrailerVerb` to a literal `"promot"` (typo) compiles,
// passes the source-policy, and only surfaces when a human reads
// the resulting commit.
//
// This test drives every mutating verb through the in-process
// dispatcher and reads HEAD's trailers via `gitops.HeadTrailers`,
// asserting:
//
//   - the required keys (verb / entity / actor) are present;
//   - `aiwf-verb` value matches the canonical verb name;
//   - `aiwf-actor` matches the supplied --actor;
//   - every trailer key on the commit is a member of the canonical
//     set declared in `internal/gitops/trailers.go` (the
//     `trailerOrder` slice). A new key landing without a corresponding
//     `Trailer*` constant fails this test on the next CI run.
//   - `import` (multi-entity, bundled-commit mode) emits one
//     `aiwf-entity` trailer per imported entity — the way
//     `aiwf history` discovers the entity-set on a bundled commit.

// canonicalTrailerKeys is the snapshot of the kernel's canonical
// trailer-key set at the time this test is wired. It mirrors the
// `trailerOrder` slice in `internal/gitops/trailers.go`. When a new
// trailer is added there, this slice gets a row; when one is
// removed (deprecation), this slice loses one. Drift is the
// regression we want to catch.
var canonicalTrailerKeys = map[string]bool{
	gitops.TrailerVerb:         true,
	gitops.TrailerEntity:       true,
	gitops.TrailerActor:        true,
	gitops.TrailerTo:           true,
	gitops.TrailerForce:        true,
	gitops.TrailerPriorEntity:  true,
	gitops.TrailerPriorParent:  true,
	gitops.TrailerTests:        true,
	gitops.TrailerPrincipal:    true,
	gitops.TrailerOnBehalfOf:   true,
	gitops.TrailerAuthorizedBy: true,
	gitops.TrailerScope:        true,
	gitops.TrailerScopeEnds:    true,
	gitops.TrailerReason:       true,
	gitops.TrailerAuditOnly:    true,
}

// entityIDPattern matches the `aiwf-entity` trailer values aiwf
// history relies on. Composite ids (M-NNN/AC-N) and plain entity ids
// (E-NN, M-NNN, ADR-NNNN, G-NNN, D-NNN, C-NNN) are both legal.
var entityIDPattern = regexp.MustCompile(`^(E-\d+|M-\d+|ADR-\d+|G-\d+|D-\d+|C-\d+)(/AC-\d+)?$`)

// TestTrailerShapePerMutatingVerb (M-069 AC-3) drives each mutating
// verb through the in-process dispatcher and asserts the resulting
// commit's trailer set conforms to the kernel's canonical shape.
//
// The lifecycle mirrors AC-2's commit-count test (same verb sequence,
// same setup) so a regression in any one verb's trailer wiring
// surfaces with the verb name in the failing-subtest line.
func TestTrailerShapePerMutatingVerb(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)

	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}

	bodyFile := filepath.Join(root, "fixtures-edit-body.md")
	if err := os.WriteFile(bodyFile, []byte("## Goal\n\nReplaced via trailer-shape test.\n"), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}

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
`)

	type step struct {
		name         string
		args         []string
		wantVerb     string // expected aiwf-verb value
		wantEntities int    // expected count of aiwf-entity trailers (1 unless multi-entity import)
	}
	steps := []step{
		{"add epic E-01", []string{"add", "epic", "--title", "Decoy", "--actor", "human/test", "--root", root}, "add", 1},
		{"add epic E-02", []string{"add", "epic", "--title", "Engine", "--actor", "human/test", "--root", root}, "add", 1},
		{"add milestone M-001", []string{"add", "milestone", "--tdd", "none", "--epic", "E-0002", "--title", "Cache", "--actor", "human/test", "--root", root}, "add", 1},
		{"add ac M-001/AC-1", []string{"add", "ac", "M-0001", "--title", "AC: warm-up works", "--actor", "human/test", "--root", root}, "add", 1},
		{"add gap G-001", []string{"add", "gap", "--title", "Sample gap", "--actor", "human/test", "--root", root}, "add", 1},
		{"add adr ADR-0001", []string{"add", "adr", "--title", "Sample ADR", "--actor", "human/test", "--root", root}, "add", 1},
		{"add decision D-001", []string{"add", "decision", "--title", "Sample decision", "--actor", "human/test", "--root", root}, "add", 1},

		{"promote E-02 → active", []string{"promote", "E-0002", "active", "--actor", "human/test", "--root", root}, "promote", 1},
		{"promote M-001 → in_progress", []string{"promote", "M-0001", "in_progress", "--actor", "human/test", "--root", root}, "promote", 1},
		{"promote M-001/AC-1 → met", []string{"promote", "M-0001/AC-1", "met", "--actor", "human/test", "--root", root}, "promote", 1},

		{"rename E-02", []string{"rename", "E-0002", "engine-renamed", "--actor", "human/test", "--root", root}, "rename", 1},
		{"edit-body M-001", []string{"edit-body", "M-0001", "--body-file", bodyFile, "--reason", "trailer-shape test", "--actor", "human/test", "--root", root}, "edit-body", 1},
		{"move M-001 → E-01", []string{"move", "M-0001", "--epic", "E-0001", "--actor", "human/test", "--root", root}, "move", 1},

		{"authorize E-02 --to ai/claude", []string{"authorize", "E-0002", "--to", "ai/claude", "--actor", "human/test", "--root", root}, "authorize", 1},
		{"authorize E-02 --pause", []string{"authorize", "E-0002", "--pause", "blocked on review", "--actor", "human/test", "--root", root}, "authorize", 1},
		{"authorize E-02 --resume", []string{"authorize", "E-0002", "--resume", "review unblocked", "--actor", "human/test", "--root", root}, "authorize", 1},

		// Multi-entity import — one commit MUST carry one aiwf-entity
		// trailer per imported entity (2 in the manifest above).
		{"import (bundled, 2 entities)", []string{"import", manifest, "--actor", "human/test", "--root", root}, "import", 2},

		{"cancel M-001", []string{"cancel", "M-0001", "--reason", "test cleanup", "--actor", "human/test", "--root", root}, "cancel", 1},
		{"reallocate E-02", []string{"reallocate", "E-0002", "--actor", "human/test", "--root", root}, "reallocate", 1},

		{"add contract C-001", []string{"add", "contract", "--title", "Sample API contract", "--actor", "human/test", "--root", root}, "add", 1},
		// recipe-install and recipe-remove operate on aiwf.yaml's
		// validators block, not on a planning entity. The verb adds
		// aiwf-entity trailers only for bindings already referencing
		// the validator (so installing-then-binding-later, the
		// install commit carries no entity trailer; binding later is
		// its own commit). At test time there are no bindings yet, so
		// the install/remove commits carry zero aiwf-entity trailers
		// — which is the documented shape (no entity to attribute).
		{"contract recipe install jsonschema", []string{"contract", "recipe", "install", "jsonschema", "--actor", "human/test", "--root", root}, "recipe-install", 0},
	}

	for _, s := range steps {
		t.Run(s.name, func(t *testing.T) {
			if rc := run(s.args); rc != exitOK {
				t.Fatalf("verb %v rc = %d (want exitOK)", s.args, rc)
			}
			assertTrailerShape(t, root, s.wantVerb, s.wantEntities)
		})
	}

	// contract bind / unbind / recipe remove require a planted
	// schema + fixtures path. Plant before running them.
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

	bindSteps := []step{
		{"contract bind C-001", []string{"contract", "bind", "C-0001", "--validator", "jsonschema", "--schema", "fixtures-contract-schema.json", "--fixtures", "fixtures-contract-data", "--actor", "human/test", "--root", root}, "bind", 1},
		{"contract unbind C-001", []string{"contract", "unbind", "C-0001", "--actor", "human/test", "--root", root}, "unbind", 1},
		{"contract recipe remove jsonschema", []string{"contract", "recipe", "remove", "jsonschema", "--actor", "human/test", "--root", root}, "recipe-remove", 0},
	}
	for _, s := range bindSteps {
		t.Run(s.name, func(t *testing.T) {
			if rc := run(s.args); rc != exitOK {
				t.Fatalf("verb %v rc = %d (want exitOK)", s.args, rc)
			}
			assertTrailerShape(t, root, s.wantVerb, s.wantEntities)
		})
	}
}

// assertTrailerShape reads HEAD's trailers and asserts the shape:
//
//   - aiwf-verb present once with value wantVerb;
//   - aiwf-actor present once with value "human/test" (the test's
//     supplied --actor);
//   - aiwf-entity present at least wantEntities times, each value a
//     valid entity-id pattern;
//   - every trailer key on the commit is a member of
//     canonicalTrailerKeys (which mirrors gitops.trailerOrder).
func assertTrailerShape(t *testing.T, root, wantVerb string, wantEntities int) {
	t.Helper()
	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}

	// Group by key for the membership / cardinality checks.
	byKey := map[string][]string{}
	for _, e := range tr {
		byKey[e.Key] = append(byKey[e.Key], e.Value)
	}

	// Required keys: aiwf-verb and aiwf-actor are present on every
	// mutating commit. aiwf-entity is per-case (most verbs require
	// it; config-global verbs like recipe-install/remove with no
	// referenced bindings legitimately omit it). Cardinality is
	// pinned by the wantEntities count check below.
	for _, k := range []string{gitops.TrailerVerb, gitops.TrailerActor} {
		if _, ok := byKey[k]; !ok {
			t.Errorf("HEAD trailers missing required key %q\n  trailers: %+v", k, tr)
		}
	}

	// Verb value pinned per case.
	if got := byKey[gitops.TrailerVerb]; len(got) != 1 || got[0] != wantVerb {
		t.Errorf("aiwf-verb = %v, want exactly [%q]\n  trailers: %+v", got, wantVerb, tr)
	}

	// Actor value pinned. The test supplies --actor human/test for
	// every step, so the resolved trailer must echo it.
	if got := byKey[gitops.TrailerActor]; len(got) != 1 || got[0] != "human/test" {
		t.Errorf("aiwf-actor = %v, want exactly [%q]\n  trailers: %+v", got, "human/test", tr)
	}

	// Entity trailer cardinality (1 for most verbs, N for bundled
	// import) plus value-shape pin.
	ents := byKey[gitops.TrailerEntity]
	if len(ents) != wantEntities {
		t.Errorf("aiwf-entity count = %d, want %d\n  values: %v\n  trailers: %+v", len(ents), wantEntities, ents, tr)
	}
	for _, v := range ents {
		if !entityIDPattern.MatchString(v) {
			t.Errorf("aiwf-entity value %q does not match entity-id pattern %s\n  trailers: %+v", v, entityIDPattern.String(), tr)
		}
	}

	// Closed-set membership: every key must be a known canonical
	// trailer. An unknown key here is the regression — either a
	// verb wired a typo, or a new trailer was introduced without
	// adding a Trailer* constant in internal/gitops/trailers.go.
	for _, e := range tr {
		if !canonicalTrailerKeys[e.Key] {
			t.Errorf("HEAD trailer key %q is not in the canonical set (internal/gitops/trailers.go trailerOrder)\n  full trailer set: %+v", e.Key, tr)
		}
	}
}
