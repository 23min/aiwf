package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// Guard treatments. The set is closed: a new verb takes one of these or
// the set grows deliberately, which is a design change rather than a
// list edit.
const (
	// guardTreatmentGuarded means the verb's plan carries file ops and
	// reaches verb.Apply, where the uncommitted-change guard compares
	// every path the plan touches. This is the default and needs no
	// per-verb machinery — Apply is the single commit-construction seam,
	// so a verb is guarded by routing through it at all.
	guardTreatmentGuarded = "guarded"

	// guardTreatmentAdopts means the verb declares AdoptsWorkingCopy on
	// a write, so the guard permits that path to diverge from HEAD after
	// verifying the divergence is body-only.
	guardTreatmentAdopts = "adopts"

	// guardTreatmentRecordOnly means the verb's plan carries no file ops
	// at all (Plan.AllowEmpty, no Ops), so the guard returns before
	// examining anything. These verbs append an audit or scope record;
	// they write no entity bytes, and there is nothing for an operator's
	// working copy to be laundered into.
	guardTreatmentRecordOnly = "record-only"
)

// verbGuardTreatments records how the commit-side write guard (ADR-0038,
// M-0283) treats each exported internal/verb entry point returning
// (*Result, error). That set is derived from the source by the AST scan
// below rather than from this list, so adding a verb of that shape
// without recording its treatment fails.
//
// A verb returning a bespoke result type escapes the scan — verb.Import
// is the one that does today, and routesOutsideTheEntryPointScan records
// it. The scan keys on the shared signature, not on "is a verb", and a
// reader deciding whether a new route is covered has to check which of
// those two it is.
//
// What this list is for: the guard lives at one seam, and "everything
// routes through Apply" is true today and easy to stop being true. A
// verb that grew its own commit path, or a record-only verb that gained
// a file op, changes its treatment silently. Writing the treatment down
// per verb makes that a visible edit.
//
// What it is NOT: evidence that a treatment is correct. It records the
// answer, not its justification-by-measurement — that is read at review.
// Overstating it would repeat the mistake M-0282 recorded, a chokepoint
// that reads as enforcing more than it does.
type guardTreatment struct {
	Func      string
	Treatment string
	Reason    string
}

var verbGuardTreatments = []guardTreatment{
	// Record-only: measured to build Plan{AllowEmpty: true} with no Ops
	// field, so checkUncommittedConflict returns on its len(ops) == 0
	// arm before reading the working tree.
	{"AcknowledgeIllegal", guardTreatmentRecordOnly, "appends an exemption record; the plan carries no file ops"},
	{"AcknowledgeMistag", guardTreatmentRecordOnly, "appends an acknowledgement record; the plan carries no file ops"},
	{"Authorize", guardTreatmentRecordOnly, "opens or closes a scope via an empty-diff commit; the plan carries no file ops"},
	{"CancelAuditOnly", guardTreatmentRecordOnly, "backfills an audit trail for a state the entity already holds; the plan carries no file ops"},
	{"PromoteAuditOnly", guardTreatmentRecordOnly, "shares CancelAuditOnly's shape: an audit record, no file ops"},
	{"PromoteACPhaseAuditOnly", guardTreatmentRecordOnly, "shares CancelAuditOnly's shape: an audit record, no file ops"},

	// Adopts: the one verb whose contract is to commit an uncommitted
	// body edit. Both modes declare it; the guard verifies the working
	// copy's frontmatter still matches HEAD's, so the exemption carries
	// a changed body and never a hand-edited field.
	{"EditBody", guardTreatmentAdopts, "exists to commit a divergent working-copy body; refusing divergence would block the route every other refusal recommends"},

	// Guarded: writes entity bytes through Apply. The serializing verbs
	// re-serialize a whole file around the frontmatter they computed,
	// and the move-shaped ones carry every file under a moved directory
	// — the two mechanisms E-0075 names.
	{"Add", guardTreatmentGuarded, "writes a new entity file, and aiwf.yaml on the --bind-validator path; an untracked destination it names is created rather than contradicted, but a dirty aiwf.yaml refuses"},
	{"AddAC", guardTreatmentGuarded, "rewrites the milestone body and frontmatter acs[]"},
	{"AddACBatch", guardTreatmentGuarded, "shares AddAC's write shape across N criteria in one commit"},
	{"Archive", guardTreatmentGuarded, "sweeps terminal entities via OpMove, carrying every file under each moved path, plus link-rewrite OpWrites on entities referencing them"},
	{"Cancel", guardTreatmentGuarded, "re-serializes the entity around a status it computed"},
	{"ContractBind", guardTreatmentGuarded, "writes the contract entity and aiwf.yaml"},
	{"ContractUnbind", guardTreatmentGuarded, "shares ContractBind's write set"},
	{"MilestoneDependsOn", guardTreatmentGuarded, "re-serializes the milestone around a depends_on list it computed"},
	{"MilestoneTDD", guardTreatmentGuarded, "re-serializes the milestone around a tdd policy it computed"},
	{"Move", guardTreatmentGuarded, "spans two surfaces: the parent field and the file's location under the epic directory"},
	{"Promote", guardTreatmentGuarded, "re-serializes the entity around a status it computed"},
	{"PromoteACPhase", guardTreatmentGuarded, "re-serializes the milestone around an AC tdd_phase it computed"},
	{"Reallocate", guardTreatmentGuarded, "moves the entity and rewrites every cross-reference to its old id"},
	{"RecipeInstall", guardTreatmentGuarded, "writes aiwf.yaml and the recipe's own files"},
	{"RecipeRemove", guardTreatmentGuarded, "shares RecipeInstall's write set"},
	{"Rename", guardTreatmentGuarded, "OpMove on a file or directory; a directory move carries every nested entity"},
	{"RenameArea", guardTreatmentGuarded, "writes aiwf.yaml plus every tagged entity in one commit"},
	{"Retitle", guardTreatmentGuarded, "sits in both mechanisms: builds an OpMove and an OpWrite, and repairs the body H1"},
	{"Rewidth", guardTreatmentGuarded, "rewrites ids tree-wide, moving and re-serializing many entities in one commit"},
	{"SetArea", guardTreatmentGuarded, "re-serializes the entity around an area it computed"},
	{"SetPriority", guardTreatmentGuarded, "re-serializes the entity around a priority it computed"},
}

// routesOutsideTheEntryPointScan names guard-relevant routes the AST
// scan below cannot see, so they are recorded rather than silently
// absent. Nothing mechanical derives this list — that is precisely why
// it is written down.
//
// Both entries are guarded, and for the same structural reason: every
// plan reaches verb.Apply regardless of which layer produced it.
var routesOutsideTheEntryPointScan = []struct {
	Route  string
	Reason string
}{
	{
		"aiwf import",
		"verb.Import returns (*ImportResult, error), so a scan keyed on (*Result, error) misses it — as it would miss any exported verb returning a bespoke result type. Its plans reach verb.Apply through cliutil.FinishVerbOutcome and are guarded there",
	},
	{
		"unexported composite-id branches",
		"a composite route (<id>/AC-N) dispatched inside an exported verb is invisible to a scan of exported signatures; the branch produces a Plan that reaches verb.Apply like any other",
	},
}

// adoptsFlagOwner is the one file permitted to set AdoptsWorkingCopy.
// The flag is the guard's single bypass lever, and the ledger's `adopts`
// row is the only row that grants one — so which verb holds it is a fact
// worth pinning rather than a claim worth writing down.
const adoptsFlagOwner = "internal/verb/editbody.go"

// checkAdoptsFlagOwnership reports every file outside adoptsFlagOwner
// that sets AdoptsWorkingCopy to true in a composite literal. A verb that
// starts setting it moves itself into the exempt class, where the guard
// stops comparing the paths it writes against HEAD — a change the ledger
// cannot see, because the ledger records names and this is a behaviour.
func checkAdoptsFlagOwnership(files []FileEntry) []Violation {
	fset := token.NewFileSet()
	var out []Violation
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/verb/") ||
			strings.HasSuffix(f.Path, "_test.go") ||
			f.Path == adoptsFlagOwner {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil { //coverage:ignore defensive: the tree under scan compiles, so a parse failure needs a file edited mid-run
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			kv, ok := n.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "AdoptsWorkingCopy" {
				return true
			}
			if lit, ok := kv.Value.(*ast.Ident); ok && lit.Name == "true" {
				out = append(out, Violation{
					Policy: "verb-write-guard-coverage",
					File:   f.Path,
					Line:   fset.Position(kv.Pos()).Line,
					Detail: "sets AdoptsWorkingCopy: true outside " + adoptsFlagOwner +
						" — that flag exempts a write from the uncommitted-change guard, so a verb taking it on is a design change (ADR-0038): move the write, or widen adoptsFlagOwner deliberately",
				})
			}
			return true
		})
	}
	return out
}

// PolicyVerbWriteGuardCoverage asserts that every exported
// internal/verb entry point — a function returning (*Result, error) —
// has a recorded decision about how the commit-side write guard treats
// it, and that no recorded decision names a function that no longer
// exists.
//
// Pins M-0283/AC-4's mechanical half. Its reach is bounded twice over:
// it covers the exported (*Result, error) entry points and nothing else
// (routesOutsideTheEntryPointScan names what that leaves out), and for
// those it proves only that a treatment is *recorded* — never that the
// treatment is correct, nor that anybody measured it. Those judgements
// are the wrap review's.
func PolicyVerbWriteGuardCoverage(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil { //coverage:ignore defensive: WalkGoFiles errors only when the scan root is unreadable, which every other policy in this package would fail on first
		return nil, err
	}

	type entryPoint struct {
		name string
		file string
		line int
	}

	fset := token.NewFileSet()
	var entries []entryPoint
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/verb/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil { //coverage:ignore defensive: the tree under scan compiles, so a parse failure here needs a file edited mid-run
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil {
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

	if len(entries) == 0 {
		// Fail closed, the shape PolicyVerbResultNoOpInvariant uses for
		// the same condition: an empty set would otherwise produce an
		// empty violation list and report green while scanning nothing.
		return []Violation{{
			Policy: "verb-write-guard-coverage",
			File:   "internal/verb/",
			Detail: "no exported (*Result, error) entry points found under internal/verb/ — the tree moved or the verb signature changed, so this policy is scanning nothing and cannot vouch for the guard's route coverage; repoint it at the verbs' new location",
		}}, nil
	}

	recorded, out := validateGuardTreatments(verbGuardTreatments)
	out = append(out, checkAdoptsFlagOwnership(files)...)

	present := make(map[string]bool, len(entries))
	for _, e := range entries {
		present[e.name] = true
		if _, ok := recorded[e.name]; ok {
			continue
		}
		out = append(out, Violation{
			Policy: "verb-write-guard-coverage",
			File:   e.file,
			Line:   e.line,
			Detail: e.name + " has no recorded decision about how the commit-side write guard treats it — add an entry to verbGuardTreatments in internal/policies/verb_write_guard_coverage.go naming its treatment and why (ADR-0038, M-0283/AC-4)",
		})
	}

	// A stale entry is the other direction of the same drift: it makes
	// the ledger read as covering a route that no longer exists, and
	// hides the fact that nothing checks that name any more.
	stale := make([]string, 0, len(recorded))
	for name := range recorded {
		if !present[name] {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)
	for _, name := range stale {
		out = append(out, Violation{
			Policy: "verb-write-guard-coverage",
			File:   "internal/policies/verb_write_guard_coverage.go",
			Detail: name + " is recorded in verbGuardTreatments but is no longer an exported (*Result, error) entry point under internal/verb/ — remove the entry, or repoint it if the verb was renamed",
		})
	}

	return out, nil
}

// validateGuardTreatments checks the ledger's own shape and returns the
// recorded treatments by verb name. Split out from the policy so the
// malformed-ledger arms are reachable from a test: the real ledger is
// well-formed, so driving them through the policy would mean mutating a
// package-level var that parallel tests share.
func validateGuardTreatments(ledger []guardTreatment) (map[string]string, []Violation) {
	recorded := make(map[string]string, len(ledger))
	var out []Violation
	for _, t := range ledger {
		switch t.Treatment {
		case guardTreatmentGuarded, guardTreatmentAdopts, guardTreatmentRecordOnly:
		default:
			out = append(out, Violation{
				Policy: "verb-write-guard-coverage",
				File:   "internal/policies/verb_write_guard_coverage.go",
				Detail: fmt.Sprintf("%s records treatment %q, which is not one of %q, %q, %q — widening the set is a design change, not a list edit",
					t.Func, t.Treatment, guardTreatmentGuarded, guardTreatmentAdopts, guardTreatmentRecordOnly),
			})
			continue
		}
		if strings.TrimSpace(t.Reason) == "" {
			out = append(out, Violation{
				Policy: "verb-write-guard-coverage",
				File:   "internal/policies/verb_write_guard_coverage.go",
				Detail: t.Func + " records a treatment with no reason; the reason is what a reader checks the treatment against",
			})
		}
		recorded[t.Func] = t.Treatment
	}
	return recorded, out
}
