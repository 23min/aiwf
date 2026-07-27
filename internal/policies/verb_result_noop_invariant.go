package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// noopExemptVerbs is the explicit, reviewed allowlist of exported
// internal/verb/*.go entry points that have no same-state input to converge
// on, so requiring a Result.NoOp test of them would be requiring a test for
// a state that cannot arise. Each Reason names why the verb is
// by-design-additive or otherwise NoOp-less — not just "has no NoOp test."
// A new entry requires a reason of the same shape, reviewed like any other
// code change.
//
// The bar is "can a caller supply input that already equals current state?"
// If yes, the verb converges to a NoOp and belongs under the policy. If the
// verb only ever appends (a new entity, a new AC), there is no same state to
// detect.
//
// Every Reason below states behavior that was verified by running the real
// binary against a disposable repo and reading the verb's source — not
// inferred. The distinction matters because one plausible-sounding inference
// was wrong across several entries: aiwf does NOT reject an empty diff. Apply's
// guard (internal/verb/apply.go) refuses only a plan with ZERO file ops, and
// `git commit-tree` has no same-tree refusal, so a verb that writes
// byte-identical content lands a real commit with an empty diffstat. Any future
// entry claiming "an identical input is an empty diff that gets rejected" is
// wrong unless the verb itself compares and refuses.
var noopExemptVerbs = []struct {
	Func   string
	Reason string
}{
	// Verified additive: a repeat allocates a new id rather than matching
	// existing state (measured — a second identical `add` produced E-0002, a
	// second identical `add ac` produced AC-2, a batch repeat produced AC-4..AC-5).
	{"Add", "purely additive: allocates a fresh id every call, so no input can already equal current state"},
	{"AddAC", "purely additive: appends a new AC; a duplicate title is a legitimate distinct AC, not a same-state input"},
	{"AddACBatch", "shares AddAC's rationale: appends N new ACs in one commit"},
	{"Reallocate", "renumbers to the next FREE id, computed rather than supplied, so the new id never equals the current one (measured: E-0001 renumbered to E-0003)"},

	// Verified to compare-and-refuse in the verb itself, so a same-state input
	// already writes nothing.
	{"EditBody", "bless mode compares the working copy against HEAD (bytes.Equal in editbody.go) and refuses when they match, so no commit lands; the refusal is informative rather than convergent because the operator asked to commit an edit they had not made"},
	{"ContractUnbind", "removes a binding: an absent binding is refused as a referential-integrity error, no commit (measured: exit 2, `no binding for <id>`)"},
	{"RecipeRemove", "shares ContractUnbind's rationale: removing an absent validator is a referential-integrity refusal (measured: exit 2, `validator not declared`)"},

	// OPEN entries. These are NOT by-design exemptions — each records a
	// deferred decision, and each is the reason this chokepoint reports green
	// with a known hole. Every one was measured, not assumed.
	{"PromoteACPhase", "OPEN, tracked in G-0458: same-phase input refuses via the TDD-phase FSM (measured: exit 1, no commit). Unlike the status case, the phase ladder is audit-bearing evidence and the verb carries a --tests payload, so convergence needs a deliberate metrics carve-out rather than a mechanical repeat — resolve by converting with that carve-out, or by rewriting this entry with a by-design reason"},
	{"AcknowledgeMistag", "OPEN, tracked in G-0459: an identical re-run appends a duplicate audit commit (measured). check.WalkAcknowledgedMistags already walks HEAD for these commits, so the dedup capability exists and is simply unused — the closest analogue to the guard acknowledge-illegal received"},
	{"PromoteAuditOnly", "OPEN, tracked in G-0459: an identical re-run appends a duplicate audit commit (measured). The verb's precondition is that the entity already sits at the target state, so a duplicate guard must key on an existing audit RECORD, not on entity state"},
	{"PromoteACPhaseAuditOnly", "OPEN, tracked in G-0459: shares PromoteAuditOnly's measured duplicate-record behavior"},
	{"CancelAuditOnly", "OPEN, tracked in G-0459: shares PromoteAuditOnly's measured duplicate-record behavior"},
	{"Authorize", "OPEN, tracked in G-0459 and G-0460: a repeat open exits 0 and appends a second commit (measured), leaving TWO simultaneously-active scopes on one entity with no check finding. Convergence may be the WRONG fix here — a second grant can be a legitimate new event — and a silent NoOp would mask the two-active-scopes defect, so G-0460 settles the invariant first"},
}

// PolicyVerbResultNoOpInvariant asserts that every exported
// internal/verb/*.go entry point — a function returning (*Result, error) —
// has at least one test under internal/verb/ that both drives that verb and
// asserts on Result.NoOp, unless the verb appears on the reviewed allowlist
// above.
//
// The property it protects is the same-state convergence convention
// (ADR-0036): a mutating verb handed input that already equals current state
// returns a NoOp at exit 0 rather than an error. That convention was
// half-rolled-out once already — four verbs had it, six did not — and
// nothing mechanical stopped the next verb from landing without it. This
// policy is that chokepoint: a new verb either carries a NoOp test or earns
// an allowlist entry stating why it cannot.
//
// Granularity is deliberately structural, not semantic: it verifies a test
// function exists that mentions both the verb call and Result.NoOp. It
// cannot prove the test drives genuinely same-state input — that is what
// review and the AC's own tests are for. What it does catch is the failure
// mode that actually recurs: a verb with no same-state NoOp coverage at all.
func PolicyVerbResultNoOpInvariant(root string) ([]Violation, error) {
	// excludeTests=false: this policy needs the test bodies as much as the
	// production ones — the property spans both halves.
	files, err := WalkGoFiles(root, false)
	if err != nil {
		return nil, err
	}

	type entryPoint struct {
		name string
		file string
		line int
	}

	fset := token.NewFileSet()
	var entries []entryPoint
	var noopTestBodies []string

	// One AST pass collects both halves: the entry points (from production
	// files) and the bodies of every test function that asserts on
	// Result.NoOp. The entry-point set is derived here rather than
	// hardcoded, so a newly-added verb is picked up with no list to update.
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/verb/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		isTest := strings.HasSuffix(f.Path, "_test.go")
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil {
				continue
			}
			start := fset.Position(fn.Body.Lbrace).Offset
			end := fset.Position(fn.Body.Rbrace).Offset
			if start < 0 || end <= start || end > len(f.Contents) {
				continue
			}
			body := string(f.Contents[start:end])

			if isTest {
				// A test function counts toward coverage only if it asserts
				// on Result.NoOp. Which verbs it credits is resolved in the
				// second loop, once the entry-point set is complete — both
				// signals must come from the SAME function, since a
				// file-level co-occurrence would credit any verb merely used
				// as fixture setup alongside an unrelated NoOp assertion.
				if strings.Contains(body, ".NoOp") {
					noopTestBodies = append(noopTestBodies, body)
				}
				continue
			}
			if isCapitalized(fn.Name.Name) && returnsResultAndError(fn.Type) {
				entries = append(entries, entryPoint{
					name: fn.Name.Name,
					file: f.Path,
					line: fset.Position(fn.Pos()).Line,
				})
			}
		}
	}

	covered := map[string]bool{}
	for _, body := range noopTestBodies {
		for _, e := range entries {
			if callsVerb(body, e.name) {
				covered[e.name] = true
			}
		}
	}

	exempt := map[string]bool{}
	for _, e := range noopExemptVerbs {
		exempt[e.Func] = true
	}

	var out []Violation
	for _, e := range entries {
		if exempt[e.name] || covered[e.name] {
			continue
		}
		out = append(out, Violation{
			Policy: "verb-result-noop-invariant",
			File:   e.file,
			Line:   e.line,
			Detail: e.name + " has no test under internal/verb/ that drives it and asserts Result.NoOp, and is not on the reviewed allowlist in verb_result_noop_invariant.go — add a same-state test asserting the verb converges to a NoOp, or add an allowlist entry stating why it has no same-state input",
		})
	}
	return out, nil
}

// callsVerb reports whether a test body calls the verb entry point named
// name. Recognizes the two spellings tests use: the external-test form
// `verb.Promote(` (package verb_test) and the in-package form `Promote(`.
// The bare form requires a non-identifier byte before the name so `Promote(`
// does not match inside `PromoteAuditOnly(` — the substring trap that would
// silently credit a longer verb's test to a shorter verb.
func callsVerb(body, name string) bool {
	return strings.Contains(body, "verb."+name+"(") || containsBareCall(body, name)
}

// containsBareCall reports whether body calls name( with no qualifier and no
// surrounding identifier characters — the in-package call spelling.
func containsBareCall(body, name string) bool {
	needle := name + "("
	for i := 0; ; {
		idx := strings.Index(body[i:], needle)
		if idx < 0 {
			return false
		}
		at := i + idx
		if at == 0 || !isIdentByte(body[at-1]) {
			return true
		}
		i = at + len(needle)
	}
}

// isIdentByte reports whether b can appear inside a Go identifier.
func isIdentByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9', b == '_', b == '.':
		return true
	}
	return false
}
