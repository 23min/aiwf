package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// skillFindingRow is what the aiwf-check skill documents for one
// finding code: every section a row for it appears under, and the line
// of the first such row.
//
// It is a list rather than a single section because nothing stops the
// skill from carrying two rows for one code. Collapsing them to the
// last one read would let a row in the errors table and a row in the
// warnings table coexist with only one of them ever judged, which is
// the contradiction an operator would be reading.
type skillFindingRow struct {
	Headings []string
	Classes  []findingSeverity
	// Texts is each occurrence's whole row, so a caller can read what
	// the cell claims and not only where it sits.
	Texts []string
	Line  int
}

// class reports the severity the skill declares for this code, and
// whether it declares exactly one. Repeated rows under sections of the
// same class agree with each other, so only *distinct* classes are a
// contradiction.
func (r skillFindingRow) class() (findingSeverity, bool) {
	seen := map[findingSeverity]bool{}
	for _, c := range r.Classes {
		seen[c] = true
	}
	if len(seen) != 1 {
		return "", false
	}
	return r.Classes[0], true
}

// sections renders the headings this code is documented under, for the
// message.
func (r skillFindingRow) sections() string {
	return strings.Join(r.Headings, " and ")
}

// Markers a `##` heading in the aiwf-check skill uses to declare the
// severity of the rows beneath it. Several sections may carry the same
// marker (tree findings and provenance findings each have their own
// errors table), so the marker classifies a heading rather than naming
// one. Both the classifier and the remediation message read these, so
// the text a heading must carry is written once.
const (
	skillHeadingMarkerErrors      = "(errors)"
	skillHeadingMarkerWarnings    = "(warnings)"
	skillHeadingMarkerConditional = "(conditional"
)

// skillSectionClass reads a `##` heading in the aiwf-check skill for the
// severity it declares about the rows beneath it. The claim is carried
// by the heading's own text, so a section is self-describing and no
// separate list maps headings to severities. A heading declaring
// nothing returns "".
//
// The conditional marker is tested first, so a heading that names the
// fixed severities while declaring itself conditional — the natural way
// to write one — reads as conditional rather than as whichever fixed
// severity it happens to mention.
func skillSectionClass(heading string) findingSeverity {
	switch {
	case strings.Contains(heading, skillHeadingMarkerConditional):
		return findingSeverityVaries
	case strings.Contains(heading, skillHeadingMarkerErrors):
		return findingSeverityError
	case strings.Contains(heading, skillHeadingMarkerWarnings):
		return findingSeverityWarning
	}
	return ""
}

// skillHeadingMarkerFor names the marker a heading must carry to hold
// rows of the given severity — the actionable half of the remediation
// message, since the internal class name is not what an author greps
// the skill for.
func skillHeadingMarkerFor(c findingSeverity) string {
	switch c {
	case findingSeverityError:
		return skillHeadingMarkerErrors
	case findingSeverityWarning:
		return skillHeadingMarkerWarnings
	}
	return skillHeadingMarkerConditional + " …)"
}

// loadSkillFindingRows reads the aiwf-check skill and returns every
// backticked-code table row keyed by its code, carrying every section
// the code is documented under. Rows outside a severity-declaring
// section are returned too, with an empty class, so a caller can tell
// "documented somewhere unclassified" apart from "not documented at
// all".
//
// Heading tracking follows `## ` only. A `###` subheading inside a
// findings section is part of that section, not a new one, so it must
// not reset the severity its rows are filed under.
func loadSkillFindingRows(root string) (map[string]skillFindingRow, error) {
	data, err := os.ReadFile(filepath.Join(root, skillCheckPath))
	if err != nil {
		return nil, err
	}
	rows := map[string]skillFindingRow{}
	heading := ""
	for i, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "## ") {
			heading = strings.TrimSpace(line)
		}
		m := skillDocRowPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		row, seen := rows[m[1]]
		if !seen {
			row.Line = i + 1
		}
		row.Headings = append(row.Headings, heading)
		row.Classes = append(row.Classes, skillSectionClass(heading))
		row.Texts = append(row.Texts, line)
		rows[m[1]] = row
	}
	return rows, nil
}

// rowSeverity is what one documented row's emissions add up to: the set
// of severities routed to it, plus a witness site for the message.
type rowSeverity struct {
	seen        map[findingSeverity]bool
	witnessFile string
	witnessLine int
}

// class collapses the observed severities into the one a section
// heading can declare. A row every site emits at the same determinate
// severity takes that severity; anything else — sites that disagree, or
// a Severity expression no static read can evaluate — varies.
func (r rowSeverity) class() findingSeverity {
	if len(r.seen) != 1 {
		return findingSeverityVaries
	}
	class := findingSeverityVaries
	for s := range r.seen {
		class = s
	}
	return class
}

// observed renders what the row's sites were seen emitting, sorted so
// the message is stable.
func (r rowSeverity) observed() string {
	parts := make([]string, 0, len(r.seen))
	for s := range r.seen {
		if s == findingSeverityVaries {
			parts = append(parts, "a Severity expression that is not a literal")
			continue
		}
		parts = append(parts, string(s))
	}
	sort.Strings(parts)
	if len(parts) == 1 && !r.seen[findingSeverityVaries] {
		return parts[0] + " at every site"
	}
	return strings.Join(parts, ", and ") + " — so its severity is decided at run time"
}

// PolicySkillTableSeverityPlacement asserts that every finding code the
// check layer emits is documented under a section of the aiwf-check
// skill whose heading declares the severity the rule actually emits.
//
// Which table a code sits in is how an operator answers "will this block
// my push?" — the placement is the signal, and prose in the cell cannot
// substitute for it. So the two sides are a contract, and both halves of
// it are derived rather than listed: the expected severity comes from
// the rule's own `Severity:` field, and the severity a section claims
// comes from its own heading text. Neither side is a hand-maintained
// map, so neither can drift from the thing it describes.
//
// A static read yields exactly three answers and the section set mirrors
// them one-to-one: always error, always warning, and varies — the last
// covering both a code whose sites disagree and one whose severity is an
// expression rather than a literal, because it is decided at run time by
// config (`tree.strict`) or by entity state (a milestone's own `tdd:`).
// Which arm a given consumer hits is not something the source carries,
// so the honest documented answer is that it depends, and the row's own
// cell says which way.
//
// Scope is the emitting side: a code with no row at all is
// PolicyFindingCodesDocumentedInSkill's finding, not this one. Two
// things do fire here beyond a plain mismatch — a row sitting outside
// any severity-declaring section, which keeps a renamed heading from
// silently unclassifying everything beneath it, and a code documented
// under two sections that declare different severities, which is the
// skill contradicting itself about whether the code blocks a push.
//
// What it cannot judge is a row whose emission the enumerator cannot
// resolve statically: a subcode assigned to a local before the literal
// is built, a code passed into a shared helper as a parameter, or a
// finding emitted outside the check layer. Those rows keep whatever
// placement their author chose. The share of the table actually judged
// is asserted separately, so narrowing this policy fails a test rather
// than quietly reading as a clean tree.
//
// The property is an aiwf-repo development invariant — it enumerates Go
// Finding{} literals by AST — and is meaningless in a consumer tree
// where internal/check is absent and the skill is materialized rather
// than authored, so it lives here as a CI-tier policy rather than an
// aiwf check finding.
func PolicySkillTableSeverityPlacement(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	rows, err := loadSkillFindingRows(root)
	if err != nil {
		return nil, err
	}

	routed := routeCheckLayerSeverities(files, rows)

	keys := make([]string, 0, len(routed))
	for k := range routed {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out []Violation
	for _, key := range keys {
		r := routed[key]
		row := rows[key]
		want := r.class()
		declared, unambiguous := row.class()
		switch {
		case !unambiguous:
			out = append(out, Violation{
				Policy: "skill-table-severity-placement",
				File:   skillCheckPath,
				Line:   row.Line,
				Detail: "finding code " + strconv.Quote(key) + " is documented under sections declaring " +
					"different severities (" + row.sections() + "), so the skill contradicts itself about " +
					"whether it blocks a push; keep one row, in a section whose heading is marked " +
					strconv.Quote(skillHeadingMarkerFor(want)),
			})
		case declared != want:
			out = append(out, Violation{
				Policy: "skill-table-severity-placement",
				File:   skillCheckPath,
				Line:   row.Line,
				Detail: "finding code " + strconv.Quote(key) + " emits " + r.observed() +
					" (e.g. " + r.witnessFile + ":" + strconv.Itoa(r.witnessLine) + ") but its row is under " +
					strconv.Quote(row.sections()) + ", which declares " + declaredBy(declared) +
					"; move the row to a section whose heading is marked " + strconv.Quote(skillHeadingMarkerFor(want)),
			})
		}
	}
	return out, nil
}

// routeCheckLayerSeverities maps each documented row to the severities
// the check layer's emissions for it were seen carrying. Split out from
// the policy so a test can assert the live tree still routes a
// substantial share of the table — a policy that routes nothing returns
// zero violations, which is indistinguishable from a clean tree.
func routeCheckLayerSeverities(files []FileEntry, rows map[string]skillFindingRow) map[string]*rowSeverity {
	routed := map[string]*rowSeverity{}
	for _, sc := range emittedFindingCodeSites(files) {
		// The aiwf-check skill documents what `aiwf check` surfaces;
		// a verb-layer finding is surfaced by its own verb instead.
		if !isCheckLayerFile(sc.File) {
			continue
		}
		// A literal with no Severity field asserts nothing about
		// severity, so it must not drag its code into "varies".
		if sc.Severity == "" {
			continue
		}
		key, ok := skillRowKeyFor(sc, rows)
		if !ok {
			continue
		}
		r := routed[key]
		if r == nil {
			r = &rowSeverity{seen: map[findingSeverity]bool{}, witnessFile: sc.File, witnessLine: sc.Line}
			routed[key] = r
		}
		r.seen[sc.Severity] = true
	}
	// A post-pass rewrites severity after construction, so a rule whose
	// literal says warning can still reach an operator as an error. That
	// is the same conditional shape as a rule that picks its severity
	// inline, and filing the two differently would sort the table by
	// where aiwf happens to implement the escalation rather than by
	// anything a consumer can observe.
	for code, sev := range postPassSeverities(files) {
		r := routed[code]
		if r == nil {
			continue
		}
		r.seen[sev] = true
	}
	return routed
}

// postPassSeverities returns every finding code whose severity a
// post-construction pass rewrites, mapped to the severity it rewrites
// them to. These are the `Apply…` functions `aiwf check` runs over the
// assembled finding list to honor an aiwf.yaml strictness knob.
//
// They are recognized by shape: a statement assigning a severity
// literal to a `.Severity` field. Rule sites never take that form —
// they set `Severity:` inside the Finding composite literal — so the
// two cannot be confused. A post-pass written some other way goes
// unseen, the same tolerance the rest of this policy takes toward what
// a static read cannot resolve.
func postPassSeverities(files []FileEntry) map[string]findingSeverity {
	codeConsts := loadCheckCodeConstants(files)
	fset := token.NewFileSet()
	out := map[string]findingSeverity{}
	for _, f := range files {
		if !isCheckLayerFile(f.Path) || strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		astFile, err := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if err != nil {
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			sev := assignedSeverity(fn.Body)
			if sev == "" {
				continue
			}
			// Every Code* the escalating function names is one it
			// escalates: these functions exist only to switch on
			// Code and rewrite Severity. The Code prefix matters —
			// codeConsts indexes every string constant in the check
			// package, so an unfiltered sweep also picks up the
			// SeverityError ident in the assignment itself and
			// records a phantom code named "error".
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				ident, ok := n.(*ast.Ident)
				if !ok || !strings.HasPrefix(ident.Name, "Code") {
					return true
				}
				if code, known := codeConsts[ident.Name]; known {
					out[code] = sev
				}
				return true
			})
		}
	}
	return out
}

// assignedSeverity returns the severity a function body assigns to a
// `.Severity` field as a statement, or "" if it assigns none. Only a
// determinate literal counts — an assignment of some computed value
// says nothing a caller can file a row under.
//
// A body whose arms assign different severities reports varies rather
// than whichever the walk reached last, mirroring how rowSeverity
// collapses construction sites that disagree.
func assignedSeverity(body *ast.BlockStmt) findingSeverity {
	var found findingSeverity
	ast.Inspect(body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}
		sel, ok := assign.Lhs[0].(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Severity" {
			return true
		}
		s := resolveSeverityExpr(assign.Rhs[0])
		switch {
		case s == findingSeverityVaries:
		case found == "":
			found = s
		case found != s:
			found = findingSeverityVaries
		}
		return true
	})
	return found
}

// declaredBy renders a section's claimed severity for the message,
// naming the unclassified case rather than printing an empty string.
func declaredBy(c findingSeverity) string {
	if c == "" {
		return "no severity"
	}
	return string(c)
}

// skillRowKeyFor returns the documented row that covers an emission
// site: the exact `code/subcode` row when one exists, else the bare
// `code` row, mirroring the subcode fallback the hint table and
// PolicyFindingCodesDocumentedInSkill both use. Reports false when
// neither row is documented.
func skillRowKeyFor(sc findingCodeSite, rows map[string]skillFindingRow) (string, bool) {
	if sc.Subcode != "" {
		key := sc.Code + "/" + sc.Subcode
		if _, ok := rows[key]; ok {
			return key, true
		}
	}
	if _, ok := rows[sc.Code]; ok {
		return sc.Code, true
	}
	return "", false
}
