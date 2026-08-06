package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// M-0293/AC-1. Every surface that describes force enforcement states
// what the kernel actually does.
//
// Each claim is asserted inside the region that must carry it, never by
// a document-wide grep: per CLAUDE.md §"Substring assertions are not
// structural assertions", the phrases here are short and appear
// legitimately elsewhere in the same files. `verb.Apply` in particular
// is named in half a dozen places in `internal/verb`, so a file-scoped
// assertion on a doc comment would pass while the doc comment said
// nothing.
//
// Most regions carry a negative assertion as well. A surface that gains
// the true sentence while keeping the false one contradicts itself and
// is no more usable than it was; the positive half alone cannot see
// that, so the specific superseded phrase is named and banned.

// forceSurfaceRegion is one located region of one surface, carrying
// enough identity for a failure to name where the claim belongs. Text
// is whitespace-normalized, so every needle below is written as one
// line regardless of how the source wraps it.
//
// Normalizing matters most for the negative assertions. A needle that
// spans a line break in the source but not in the test matches nothing,
// and a `mustNotSay` that matches nothing passes — silently, and from
// the first run, so it never goes red and never proves the superseded
// phrase left.
type forceSurfaceRegion struct {
	Where string
	Text  string
}

// newRegion normalizes every run of whitespace to a single space.
func newRegion(where, text string) forceSurfaceRegion {
	return forceSurfaceRegion{Where: where, Text: strings.Join(strings.Fields(text), " ")}
}

func (r forceSurfaceRegion) mustSay(t *testing.T, needle, why string) {
	t.Helper()
	if !strings.Contains(r.Text, needle) {
		t.Errorf("%s does not say %q — %s", r.Where, needle, why)
	}
}

func (r forceSurfaceRegion) mustNotSay(t *testing.T, needle, why string) {
	t.Helper()
	if strings.Contains(r.Text, needle) {
		t.Errorf("%s still says %q — %s", r.Where, needle, why)
	}
}

// readRepoFile reads a repo-root-relative file. Fails rather than
// returning "": every assertion below is a Contains, so an unreadable
// file would turn the whole test green.
func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return string(raw)
}

// mdSection locates a markdown section and fails when the heading is
// gone, which is the locator going stale rather than the claim holding.
func mdSection(t *testing.T, rel string, level int, heading string) forceSurfaceRegion {
	t.Helper()
	body := extractMarkdownSection(readRepoFile(t, rel), level, heading)
	if body == "" {
		t.Fatalf("%s has no level-%d section %q; the locator is stale and every assertion "+
			"against it would be reading an empty string", rel, level, heading)
	}
	return newRegion(rel+" §"+heading, body)
}

// mdTableRow locates the one markdown table row whose first cell is key.
// A key matching zero rows means the catalogue entry was renamed or
// removed; more than one means the key is ambiguous and the assertions
// would be reading whichever came first.
func mdTableRow(t *testing.T, rel, key string) forceSurfaceRegion {
	t.Helper()
	prefix := "| " + key + " |"
	var hits []string
	for _, line := range strings.Split(readRepoFile(t, rel), "\n") {
		if strings.HasPrefix(line, prefix) {
			hits = append(hits, line)
		}
	}
	switch len(hits) {
	case 1:
		return newRegion(rel+" row "+key, hits[0])
	case 0:
		t.Fatalf("%s has no table row keyed %q; the locator is stale", rel, key)
	default:
		t.Fatalf("%s has %d table rows keyed %q; the key must identify one row", rel, len(hits), key)
	}
	return forceSurfaceRegion{}
}

// parseRepoGoFile parses one repo-root-relative Go file with comments,
// returning the AST, the file set, and the raw source.
func parseRepoGoFile(t *testing.T, rel string) (*ast.File, *token.FileSet, string) {
	t.Helper()
	src := readRepoFile(t, rel)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join(repoRoot(t), rel), src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing %s: %v", rel, err)
	}
	return f, fset, src
}

// goDeclDoc locates the doc comment of a named top-level declaration —
// a func, or a type/var/const whose GenDecl carries the comment.
//
// Scoping to the doc comment rather than the file is what makes the
// assertions structural here: both files this is used on name the seam
// elsewhere in their bodies, so a file-wide search would pass on a doc
// comment that still described the old one.
func goDeclDoc(t *testing.T, rel, name string) forceSurfaceRegion {
	t.Helper()
	f, _, _ := parseRepoGoFile(t, rel)
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == name && d.Doc != nil {
				return newRegion(rel+" doc on "+name, d.Doc.Text())
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				var got string
				switch s := spec.(type) {
				case *ast.TypeSpec:
					got = s.Name.Name
				case *ast.ValueSpec:
					if len(s.Names) > 0 {
						got = s.Names[0].Name
					}
				}
				if got == name && d.Doc != nil {
					return newRegion(rel+" doc on "+name, d.Doc.Text())
				}
			}
		}
	}
	t.Fatalf("%s has no documented top-level declaration named %q; the locator is stale", rel, name)
	return forceSurfaceRegion{}
}

// goFuncBody locates the source text of a named function's body,
// comments included. Used where the claim lives in an inline comment or
// a message string rather than in a doc comment.
func goFuncBody(t *testing.T, rel, name string) forceSurfaceRegion {
	t.Helper()
	f, fset, src := parseRepoGoFile(t, rel)
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != name || fn.Body == nil {
			continue
		}
		start := fset.Position(fn.Body.Lbrace).Offset
		end := fset.Position(fn.Body.Rbrace).Offset
		return newRegion(rel+" body of "+name, src[start:end])
	}
	t.Fatalf("%s has no function named %q with a body; the locator is stale", rel, name)
	return forceSurfaceRegion{}
}

// ---------------------------------------------------------------------
// Rows 1-4 — the kernel's own guidance and its design record.
// ---------------------------------------------------------------------

// TestM0293_KernelGuidanceStatesForceIsHumanOnly pins the four surfaces
// that state the guarantee without naming a chokepoint. All four were
// false until the guard reached the apply seam; none needs to name the
// seam, because a principle and a commitment describe what holds rather
// than where it is enforced. What they need is to keep saying it, which
// is what these assertions hold.
func TestM0293_KernelGuidanceStatesForceIsHumanOnly(t *testing.T) {
	t.Parallel()

	// Row 1 — the engineering principle.
	principles := mdSection(t, "CLAUDE.md", 2, "Engineering principles")
	principles.mustSay(t, "`--force` is sovereign — humans only",
		"the provenance principle is where a reader learns force is not delegable")

	// Row 2 — the kernel commitment.
	commits := mdSection(t, "CLAUDE.md", 2, "What aiwf commits to")
	commits.mustSay(t, "`--force` requires a human actor",
		"commitment 9 is the load-bearing list; a guarantee absent from it is not committed to")

	// Row 3 — the cross-cutting property.
	crossCutting := mdSection(t, "docs/design/design-decisions.md", 2, "Cross-cutting properties")
	crossCutting.mustSay(t, "`--force` requires a human actor",
		"the provenance property states the guarantee for a reader who reads no further")

	// Row 4 — the provenance model's own bullet.
	provenance := mdSection(t, "docs/design/design-decisions.md", 3, "Provenance model")
	provenance.mustSay(t, "refuses an `aiwf-force:` trailer from any actor whose role is not `human/",
		"this is the design record of the refusal itself")
	// The rule is keyed on the trailer, not the flag, and the difference
	// is observable: `aiwf add --force` by a non-human actor exits 0 on a
	// kind with no body gate, because the flag is inert there and no
	// sovereign act is recorded. A record claiming the flag is refused
	// would be false in exactly the case this milestone exempted `add`
	// for.
	provenance.mustNotSay(t, "refuses `--force` from any actor",
		"the kernel refuses the trailer rather than the flag; stating it as the flag contradicts "+
			"the measured behaviour of `aiwf add --force` under a non-human actor")
}

// ---------------------------------------------------------------------
// Rows 5-7 — the rule catalogue, whose citations name the chokepoint.
// ---------------------------------------------------------------------

const auditCatalogue = "docs/design/legal-workflows-audit.md"

// TestM0293_AuditCatalogueCitesTheSeamThatRefuses covers the three
// catalogue rows. These are the surfaces where naming the seam is the
// whole content: a rule row's citation column is what a reader follows
// to find the code, and all three pointed at a guard that was never
// built.
func TestM0293_AuditCatalogueCitesTheSeamThatRefuses(t *testing.T) {
	t.Parallel()

	// Row 5 — the universal sovereign rule.
	rule076 := mdTableRow(t, auditCatalogue, "R-RULE-076")
	rule076.mustSay(t, "verb.Apply",
		"the chokepoint column must name the seam that refuses; a reader follows it to the code")
	rule076.mustNotSay(t, "cliutil sovereign guard",
		"no such guard exists in internal/cli/cliutil — the citation sends a reader to nothing")
	rule076.mustNotSay(t, "policies/sovereign.go",
		"internal/policies/sovereign.go does not exist; the rule is kept at the apply seam, and "+
			"a citation pointing anywhere else sends a reader to nothing")

	// Row 6 — the universal mutating-verb rule. Its scope column covers
	// --force and --audit-only together, and they are enforced at
	// different seams: ADR-0040 keeps audit-only-non-human off the apply
	// seam deliberately, since it fails the satisfiability test the
	// force rules pass. One citation for both would be a fresh false
	// claim, so the row names each.
	audit0105 := mdTableRow(t, auditCatalogue, "R-AUDIT-0105")
	audit0105.mustSay(t, "verb.Apply",
		"the source column must name the seam that refuses a forced non-human act")
	audit0105.mustSay(t, "auditonly.go",
		"audit-only is refused on the audit-only verb's own path, not at the apply seam; "+
			"one citation for both flags misstates where each is caught")
	audit0105.mustNotSay(t, "cliutil sovereign guard",
		"no such guard exists in internal/cli/cliutil")

	// The audit-only sovereign rule, cited to the same absent guard as
	// rows 5 and 6. Not in the milestone's table, which enumerated the
	// surfaces found by searching for force claims; this one states the
	// rule for the sibling flag and carries the identical citation.
	rule077 := mdTableRow(t, auditCatalogue, "R-RULE-077")
	rule077.mustSay(t, "auditonly.go",
		"the chokepoint column must name the path that refuses a non-human audit-only act")
	rule077.mustNotSay(t, "cliutil sovereign guard",
		"no such guard exists in internal/cli/cliutil")

	// Row 7 — epic activation, the canonical sovereign act.
	audit0113 := mdTableRow(t, auditCatalogue, "R-AUDIT-0113")
	audit0113.mustNotSay(t, "requires `--force --reason \"...\"` AND a `human/...` actor",
		"both halves are wrong: a human actor needs no --force to activate an epic, and "+
			"--force cannot carry a non-human one past the seam")
	audit0113.mustSay(t, "does not substitute",
		"the row must say what --force fails to do here, or a reader takes the flag for the "+
			"non-human path — which is the reading this epic exists to close")

	// The rule-catalogue half of the same claim. The two catalogues each
	// carry an epic-activate entry and both said --force was required;
	// correcting one and leaving the other is the drift this milestone
	// ends rather than repeats.
	rule078 := mdTableRow(t, auditCatalogue, "R-RULE-078")
	rule078.mustNotSay(t, "requires `--force --reason \"...\"` AND `human/...` actor",
		"a human actor needs no --force to activate an epic, and --force cannot carry a "+
			"non-human one past the seam")
	rule078.mustSay(t, "does not substitute",
		"the row must say what --force fails to do here")
	rule078.mustNotSay(t, "policies/sovereign.go",
		"internal/policies/sovereign.go does not exist; the rule is kept at the apply seam, and "+
			"a citation pointing anywhere else sends a reader to nothing")
}

// ---------------------------------------------------------------------
// Row 8 and its three siblings — the sovereign --force flag help.
// ---------------------------------------------------------------------

// sovereignForceFlag is one dispatcher's `--force` registration, found
// alongside a `--reason` registration in the same function.
type sovereignForceFlag struct {
	File string
	Func string
	Help string
}

// sovereignForceFlags returns every `--force` flag registered by a
// dispatcher that also registers `--reason`.
//
// The pairing is the kernel's own discriminator for the sovereign
// meaning of the word, quoted from internal/policies/sovereign.go: a
// `--force` without `--reason` is force-replace, a different word spelled
// the same. Taking the discriminator rather than an allowlist is what
// keeps `contract bind`, `contract recipe` and `update --remove` out by
// construction instead of by a list someone has to maintain — and the
// negative control below asserts they really are out.
func sovereignForceFlags(t *testing.T) []sovereignForceFlag {
	t.Helper()
	root := repoRoot(t)
	cliDir := filepath.Join(root, "internal", "cli")
	fset := token.NewFileSet()
	var out []sovereignForceFlag

	err := filepath.WalkDir(cliDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		src := string(raw)
		astFile, parseErr := parser.ParseFile(fset, path, src, 0)
		if parseErr != nil {
			return parseErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			body := src[fset.Position(fn.Body.Lbrace).Offset:fset.Position(fn.Body.Rbrace).Offset]
			if !strings.Contains(body, `"reason"`) {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, isCall := n.(*ast.CallExpr)
				if !isCall || len(call.Args) != 4 {
					return true
				}
				sel, isSel := call.Fun.(*ast.SelectorExpr)
				if !isSel || sel.Sel.Name != "BoolVar" {
					return true
				}
				if name, okLit := goStringLit(call.Args[1]); !okLit || name != "force" {
					return true
				}
				help, okLit := goStringLit(call.Args[3])
				if !okLit {
					return true
				}
				out = append(out, sovereignForceFlag{File: filepath.ToSlash(rel), Func: fn.Name.Name, Help: help})
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking internal/cli: %v", err)
	}
	return out
}

// goStringLit unquotes a string-literal argument, reporting false for
// anything else (an identifier, a concatenation) so the caller skips it
// rather than asserting against a rendering of the AST.
func goStringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

// TestM0293_SovereignForceFlagHelpStatesTheActorConstraint pins row 8
// and the three dispatchers of identical shape.
//
// The table names only `promote`, but `cancel`, `add` and `authorize`
// register the same sovereign flag, reach the same seam and were
// equally silent about it. Correcting one and leaving three is the
// drift this milestone exists to end, so the assertion is over the
// discriminator rather than over the one file the table happened to
// name.
func TestM0293_SovereignForceFlagHelpStatesTheActorConstraint(t *testing.T) {
	t.Parallel()
	flags := sovereignForceFlags(t)

	// Vacuity guard. A walk that finds nothing reports no failures,
	// which is indistinguishable from a tree where every help is right.
	// The named surface must be among the findings, and the population
	// must be the several this scans rather than one lucky match.
	var files []string
	foundPromote := false
	for _, f := range flags {
		files = append(files, f.File)
		if f.File == "internal/cli/promote/promote.go" {
			foundPromote = true
		}
	}
	if !foundPromote {
		t.Fatalf("the sovereign --force scan did not find internal/cli/promote/promote.go — "+
			"the walk or the discriminator is broken, and every assertion below is vacuous "+
			"(found: %v)", files)
	}
	if len(flags) < 4 {
		t.Fatalf("the sovereign --force scan found %d flags (%v); four dispatchers register "+
			"--force alongside --reason, so a smaller population means the scan is missing some",
			len(flags), files)
	}

	// Negative control: force-replace must stay out. Sweeping those in
	// would claim a human-only constraint the kernel does not enforce on
	// them, and would break legitimate automation if anyone acted on it.
	for _, f := range flags {
		if strings.HasPrefix(f.File, "internal/cli/contract/") || f.File == "internal/cli/update/update.go" {
			t.Errorf("%s (%s) was classified sovereign; its --force means force-replace and is "+
				"open to non-human actors", f.File, f.Func)
		}
	}

	for _, f := range flags {
		if !strings.Contains(f.Help, "human/") {
			t.Errorf("%s (%s): the --force help does not name the human/ actor constraint. "+
				"An operator reading it learns what the flag relaxes but not that the kernel "+
				"refuses it from a non-human actor before anything is written. Help was: %q",
				f.File, f.Func, f.Help)
		}
	}
}

// ---------------------------------------------------------------------
// Rows 9-10 — the two verb-layer comments that contradicted each other.
// ---------------------------------------------------------------------

// TestM0293_VerbCommentsNameTheSeamTheyHandOffTo covers the pair the
// epic's Context singled out: one comment reasoned its way to stepping
// aside for a guard its own verb never called, and the other asserted
// the opposite stance one package away.
func TestM0293_VerbCommentsNameTheSeamTheyHandOffTo(t *testing.T) {
	t.Parallel()

	// Row 9 — the sovereign-act gate's handoff. It steps aside for
	// --force, so its doc comment is where a reader learns what catches
	// the case it declined. Scoped to the doc comment: the function's
	// body already names verb.Apply in the comment above its error
	// message, so a file-wide assertion would pass on a stale doc.
	sovereignGate := goDeclDoc(t, "internal/verb/promote_sovereign_act.go", "requireHumanActorForSovereignAct")
	sovereignGate.mustSay(t, "verb.Apply",
		"the gate hands off to the coherence guard for --force; naming no seam is what let the "+
			"handoff point at a guard that was never called")

	// Row 10 — add's force branch, which asserted the opposite: that a
	// check-time audit was the only backstop and no verb-time gate
	// existed. It is the same package as row 9.
	addForce := goFuncBody(t, "internal/verb/add.go", "Add")
	addForce.mustNotSay(t, "rather than a verb-time human-actor gate here",
		"a verb-time gate now runs at the apply seam, so this reads as a live design stance "+
			"against the guard the epic installed")
	addForce.mustSay(t, "verb.Apply",
		"the force branch must name the seam that refuses a non-human actor, or the reader is "+
			"left with the check-time audit as the only backstop")
}

// ---------------------------------------------------------------------
// Row 11 — the ADR asserting that verb.Apply validates nothing.
// ---------------------------------------------------------------------

// TestM0293_ADR0029ScopesItsNoValidationClaim pins the mirror case: a
// surface asserting the *absence* of a guarantee that now exists.
//
// The claim was already imprecise before this epic, since Apply refuses
// several conditions ahead of any write. What makes it worth correcting
// now is its stated reasoning — that a check inside Apply would
// duplicate the real gate and should not be added — which a reader
// meeting beside ADR-0040 takes as opposite guidance about the same
// function.
func TestM0293_ADR0029ScopesItsNoValidationClaim(t *testing.T) {
	t.Parallel()
	body := adrBody(t, "ADR-0029")

	// The two claims sit in different sections, so each is asserted
	// against the one that carries it. Pointing both at one section
	// would leave whichever landed in the other section unasserted —
	// and a negative assertion whose phrase is not in the region it
	// searches passes from the first run without ever going red.
	decision := extractMarkdownSection(body, 2, "Decision")
	if decision == "" {
		t.Fatal("ADR-0029 has no ## Decision section; the locator is stale")
	}
	dec := newRegion("ADR-0029 §Decision", decision)

	dec.mustNotSay(t, "pure mechanical write-then-commit",
		"Apply refuses a forced non-human act, a git operation in progress, a staged conflict, "+
			"a carried symlink and a HEAD-divergent path before it writes anything")
	dec.mustSay(t, "CheckForceTrailerCoherence",
		"the decision must name what Apply does refuse, or its no-validation claim reads as a "+
			"claim about everything Apply does rather than about entity content")

	consequences := extractMarkdownSection(body, 2, "Consequences")
	if consequences == "" {
		t.Fatal("ADR-0029 has no ## Consequences section; the locator is stale")
	}
	cons := newRegion("ADR-0029 §Consequences", consequences)

	cons.mustNotSay(t, "staying validation-free is intentional and should not be",
		"a guard was added inside Apply and this reads as standing guidance against it")
	cons.mustSay(t, "ADR-0040",
		"without the pointer a reader has two ADRs giving opposite guidance on whether Apply "+
			"may refuse anything, and no way to tell which is current")
}

// ---------------------------------------------------------------------
// Rows 12-13 — the two surfaces offering --force to a non-human actor.
// ---------------------------------------------------------------------

// TestM0293_SovereignActShapeDocRetractsTheNonHumanForcePath covers row
// 12. The runtime message that gave the same advice was corrected in
// M-0291, because it is reachable only for the population the advice is
// wrong for; this is prose, and waited for the guard to land.
func TestM0293_SovereignActShapeDocRetractsTheNonHumanForcePath(t *testing.T) {
	t.Parallel()
	doc := goDeclDoc(t, "internal/entity/sovereign.go", "SovereignActShape")

	doc.mustNotSay(t, "or `--force --reason \"...\"` from a non-human actor",
		"the seam refuses exactly that combination; the doc offers it as the sovereign gesture")
	doc.mustSay(t, "human/",
		"the doc defines what the sovereign gesture is, so it must state the actor constraint")
}

// TestM0293_SovereignAuditPolicyMessageRetractsTheNonHumanForcePath
// covers row 13 — the same shape as row 12, in a failure message.
//
// The message is what a contributor reads when the policy fires on a
// script or workflow, so its remedy is acted on directly. Offering
// `--force --reason` there sends a CI bot at a seam that refuses it.
func TestM0293_SovereignAuditPolicyMessageRetractsTheNonHumanForcePath(t *testing.T) {
	t.Parallel()
	body := goFuncBody(t, "internal/policies/aiwf_promote_epic_active_audit_test.go",
		"TestPolicy_NoNonForcedSovereignActPromoteInCIScripts")

	body.mustNotSay(t, "append `--force --reason \\\"...\\\"` to the same line",
		"a scripted invocation runs as a non-human actor, and the seam refuses a force trailer "+
			"from one; the remedy names the only path that works")
	body.mustSay(t, "have a human run the verb",
		"the remedy must name the human-run path, which is what actually clears the finding")
}
