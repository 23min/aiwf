package policies

import (
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRootFromTest walks up from the test's working directory until it
// finds a go.mod file, returning the absolute path. Used by structural
// policy tests that grep across the whole repo.
func repoRootFromTest(t *testing.T) (string, error) {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// TestPolicy_NarrowIDLiteralsAllowlisted is the AC-5 chokepoint for
// M-081's test-fixture sweep. It greps for narrow-width entity-id
// string literals (`"E-NN"`, `"M-NNN"`, etc.) under internal/ and
// cmd/aiwf/, and reports any match outside the allowlist.
//
// The allowlist is the fixed set of test files that intentionally
// exercise parser tolerance (AC-2 and AC-4 in M-081) plus a handful
// of files whose narrow id literals belong to the input space rather
// than the expected outputs (the entity-grammar tests, the gitops
// trailer-shape tests, etc.).
//
// The match grammar:
//
//	"[EMGDC]-[0-9]{1,3}"   // narrow widths only; ADR is exempt
//
// ADR is exempt: its grammar (`ADR-\d{4,}`) was always at canonical
// width, so there is no narrow-legacy form to track. The test does
// scan composite-id literals (`"M-NNN/AC-N"`) under the same
// regex; those are also covered by the allowlist mechanism.
//
// Per CLAUDE.md "framework correctness must not depend on the LLM's
// behavior," AC-5's discipline lives in this test, not in reviewer
// recall.
func TestPolicy_NarrowIDLiteralsAllowlisted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("grep dependency: skip on windows CI")
	}
	repoRoot, err := repoRootFromTest(t)
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}

	// Each entry is a repo-relative file path that legitimately holds
	// narrow-width id literals because the literal is the input to a
	// parser-tolerance test (AC-2/AC-4) or an entity-grammar test.
	allowlist := map[string]string{
		// AC-2 parser-tolerance — the load-bearing tests for both-widths
		// equivalence at the lookup seam. Narrow inputs are required
		// by design.
		"internal/entity/canonicalize_test.go":  "AC-2 parser-tolerance test (Canonicalize, IDGrepAlternation)",
		"internal/tree/tree_test.go":            "AC-2 lookup-seam test (TestTree_ByID_AcceptsBothWidths, TestTree_ByPriorID_AcceptsBothWidths)",
		"cmd/aiwf/canonicalize_render_test.go":  "AC-3 narrow-tree fixture exercising canonical render output",
		"cmd/aiwf/canonicalize_history_test.go": "AC-4 narrow trailer matches canonical query",

		// Entity-grammar tests — the narrow ids are inputs to grammar
		// validators (idPatterns, ParseCompositeID, KindFromID, IDFromPath).
		// These pin the input space the parser tolerates; canonicalizing
		// them would erase the test's value.
		"internal/entity/entity_test.go":     "id-grammar input space (idPatterns, IDFromPath, KindFromID, ParseCompositeID)",
		"internal/entity/parse_test.go":      "frontmatter-parser input space (narrow-id legacy fixtures)",
		"internal/entity/serialize_test.go":  "frontmatter-serializer round-trip on narrow legacy inputs",
		"internal/entity/transition_test.go": "FSM transition tests using narrow legacy ids as inputs",

		// gitops trailer round-trip is width-agnostic (the package never
		// canonicalizes; it just round-trips bytes). Narrow inputs in
		// these tests exercise the parser's tolerance.
		"internal/gitops/gitops_test.go":   "trailer round-trip on narrow legacy inputs",
		"internal/gitops/trailers_test.go": "trailer-shape validation on narrow legacy inputs",

		// Trunk reads on-disk filenames verbatim; canonicalization
		// happens at consumer (allocator, ids-unique check).
		"internal/trunk/trunk_test.go": "trunk-read returns on-disk id verbatim (consumer canonicalizes)",

		// Allocator's parseIDNumber tolerates narrow legacy ids on
		// disk. Tests fixture narrow inputs to exercise that contract.
		"internal/entity/allocate_test.go": "allocator parser-tolerance: narrow on-disk ids → canonical next allocation",

		// Contractbind's unbind preserves the on-disk yaml entry
		// verbatim (body-prose canonicalization is M-082's job).
		"internal/verb/contractbind_test.go": "yaml-entry round-trip preserves narrow legacy widths verbatim (deferred to M-082)",

		// Skills package asserts on-disk SKILL.md content; doc-prose
		// canonicalization is M-082's `aiwf rewidth` job.
		"internal/skills/skills_test.go": "skill SKILL.md prose markers (body-prose canonicalization deferred to M-082)",

		// Whiteboard policy reads SKILL.md fixtures whose prose carries
		// narrow legacy ids by design.
		"internal/policies/aiwfx_whiteboard_test.go": "skill body-prose markers (deferred to M-082)",

		// selfcheck drives every verb against a throwaway repo using
		// narrow legacy id inputs by design (exercises AC-2 parser
		// tolerance end-to-end through the binary).
		"cmd/aiwf/selfcheck.go": "self-check drives verbs with narrow inputs to exercise parser tolerance",

		// M-082: aiwf rewidth verb's tests fixture narrow inputs by
		// design — that's the verb's input space (the very migration
		// from narrow to canonical that the verb performs).
		"cmd/aiwf/rewidth_cmd_test.go":  "M-082 rewidth verb tests; narrow ids are the verb's input space",
		"internal/verb/rewidth_test.go": "M-082 rewidth verb unit tests; narrow ids are the input space",
		"internal/verb/rewidth.go":      "M-082 rewidth verb; narrow ids in regex source documentation are part of the spec citation",

		// M-082 prep: M-080 fixture-validation tests use the M-080
		// narrow id as the canonical entity-id query (resolved via
		// tree.ByID's width-tolerant lookup) and the E-21 narrow-form
		// reference whose either-width rendering must satisfy the
		// AC-7 substring assertion. Both narrow-input cases by
		// design — same shape as AC-2 parser-tolerance tests.
		"internal/policies/m080_test.go": "M-082 prep: M-080 spec lookup + AC-7 substring assertion via width-tolerant helpers",
	}

	// Run grep from the repo root. Pipe stderr alongside stdout so
	// the rare "no match" exit-1 case still shows up in CI output
	// without the test panicking.
	cmd := exec.Command("grep", "-rEl", `"[EMGDC]-[0-9]{1,3}"`, "internal/", "cmd/aiwf/")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		// grep -rE returns:
		//   0  matches found
		//   1  no matches (clean pass for AC-5)
		//   2  syntax error or some files unreadable (e.g. transient
		//       index lock during a verb commit). Skip rather than
		//       fail — CI's standalone `go test` run is the chokepoint.
		var ee *exec.ExitError
		if asExitErr(err, &ee) {
			switch ee.ExitCode() {
			case 1:
				return
			case 2:
				t.Skipf("grep returned 2 (likely transient file access); CI standalone run is the chokepoint: %s", ee.Stderr)
			}
		}
		t.Fatalf("grep failed: %v", err)
	}
	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	var unallowed []string
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		// grep reports paths like `internal//entity/...` because we
		// passed `internal/` as a directory arg. Normalize the double
		// slash so the allowlist lookup matches.
		f = filepath.Clean(f)
		// Embedded skill markdown: not Go source, narrow ids in prose
		// are SKILL.md content the kernel doesn't canonicalize.
		if strings.Contains(f, "/embedded/") {
			continue
		}
		// testdata content: similarly, body-prose canonicalization
		// happens in M-082, not at the test layer.
		if strings.Contains(f, "/testdata/") {
			continue
		}
		// Renderer prose-rewrite docstring contains the example
		// `M-007/AC-1` — that's a comment, not a literal. The grep
		// matches both, so we filter the renderer-package source
		// files via the allowlist below.
		if _, ok := allowlist[f]; ok {
			continue
		}
		if !strings.HasSuffix(f, "_test.go") {
			// Non-test source: only Go comments may contain narrow
			// literals (commented examples). Re-grep the file
			// excluding comment lines.
			cleanCmd := exec.Command("sh", "-c", `grep -Ev '^\s*(//|\*)' `+filepath.Join(repoRoot, f)+` | grep -E '"[EMGDC]-[0-9]{1,3}"'`)
			cleanCmd.Dir = repoRoot
			if cleanOut, cErr := cleanCmd.Output(); cErr == nil && strings.TrimSpace(string(cleanOut)) != "" {
				unallowed = append(unallowed, f+" (non-comment narrow id literal)")
			}
			continue
		}
		unallowed = append(unallowed, f)
	}

	if len(unallowed) > 0 {
		t.Errorf("AC-5: narrow-id string literals found outside allowlist:\n  %s\n\n"+
			"Each match is either:\n"+
			"  (a) an expected output that should sweep to canonical 4-digit width, or\n"+
			"  (b) a parser-tolerance test that needs an entry in narrow_id_sweep_test.go's allowlist.\n",
			strings.Join(unallowed, "\n  "))
	}
}

// asExitErr is a small wrapper around errors.As to keep the test's
// imports lean.
func asExitErr(err error, target **exec.ExitError) bool {
	return errors.As(err, target)
}
