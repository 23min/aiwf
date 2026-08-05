package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// severitySkill returns a minimal aiwf-check skill body whose three
// severity-declaring sections hold the given codes, one row each. The
// heading text is the real one, since the classifier reads the marker
// out of it.
func severitySkill(errors, warnings, conditional []string) string {
	section := func(heading string, codes []string) string {
		s := "\n## " + heading + "\n\n| Code | Meaning |\n|---|---|\n"
		for _, c := range codes {
			s += "| `" + c + "` | meaning |\n"
		}
		return s
	}
	return "# aiwf-check\n" +
		section("Findings (errors)", errors) +
		section("Findings (warnings)", warnings) +
		section("Findings (conditional severity)", conditional)
}

// emitSrc is a check-layer source file emitting one code at one severity.
func emitSrc(code, severityExpr string) string {
	return "package check\n\nvar _ = Finding{Code: \"" + code + "\", Severity: " + severityExpr + "}\n"
}

// TestSkillTableSeverityPlacement_Placement is the firing/silent matrix:
// each row states what the rule emits, where the skill files it, and
// whether that pairing is a violation.
func TestSkillTableSeverityPlacement_Placement(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		// src is the check-layer file emitting the code.
		src string
		// errors / warnings / conditional are the codes each section holds.
		errors, warnings, conditional []string
		wantFire                      bool
		wantDetail                    []string
	}{
		{
			name:     "error rule in the errors table is silent",
			src:      emitSrc("probe", "SeverityError"),
			errors:   []string{"probe"},
			wantFire: false,
		},
		{
			name:     "warning rule in the warnings table is silent",
			src:      emitSrc("probe", "SeverityWarning"),
			warnings: []string{"probe"},
			wantFire: false,
		},
		{
			name:        "conditional rule in the conditional table is silent",
			src:         emitSrc("probe", "severity"),
			conditional: []string{"probe"},
			wantFire:    false,
		},
		{
			name:       "error rule filed under warnings fires",
			src:        emitSrc("probe", "SeverityError"),
			warnings:   []string{"probe"},
			wantFire:   true,
			wantDetail: []string{`"probe"`, "error at every site", "Findings (warnings)", `"(errors)"`},
		},
		{
			name:       "warning rule filed under errors fires",
			src:        emitSrc("probe", "SeverityWarning"),
			errors:     []string{"probe"},
			wantFire:   true,
			wantDetail: []string{"warning at every site", "Findings (errors)", `"(warnings)"`},
		},
		{
			name:       "conditional rule filed under warnings fires",
			src:        emitSrc("probe", "severity"),
			warnings:   []string{"probe"},
			wantFire:   true,
			wantDetail: []string{"not a literal", "decided at run time", `"(conditional …)"`},
		},
		{
			name: "sites that disagree fire even though each is a literal",
			src: "package check\n\n" +
				"var _ = Finding{Code: \"probe\", Severity: SeverityError}\n" +
				"var _ = Finding{Code: \"probe\", Severity: SeverityWarning}\n",
			errors:     []string{"probe"},
			wantFire:   true,
			wantDetail: []string{"error, and warning", `"(conditional …)"`},
		},
		{
			name:       "an emission qualified across packages is judged like any other",
			src:        emitSrc("probe", "check.SeverityError"),
			warnings:   []string{"probe"},
			wantFire:   true,
			wantDetail: []string{"error at every site"},
		},
		{
			// The conditional table is the natural place to park a row
			// an author is unsure about, so a fixed-severity rule
			// sitting there has to fire — otherwise it is an escape
			// hatch from the whole policy.
			name:        "error rule parked in the conditional table fires",
			src:         emitSrc("probe", "SeverityError"),
			conditional: []string{"probe"},
			wantFire:    true,
			wantDetail:  []string{"error at every site", "conditional severity", `"(errors)"`},
		},
		{
			name:        "warning rule parked in the conditional table fires",
			src:         emitSrc("probe", "SeverityWarning"),
			conditional: []string{"probe"},
			wantFire:    true,
			wantDetail:  []string{"warning at every site", `"(warnings)"`},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeAt(t, root, "internal/check/x.go", tc.src)
			mustWrite(t, filepath.Join(root, skillCheckPath),
				severitySkill(tc.errors, tc.warnings, tc.conditional))

			vs, err := PolicySkillTableSeverityPlacement(root)
			if err != nil {
				t.Fatalf("policy error: %v", err)
			}
			got := hasPolicyViolation(vs, "skill-table-severity-placement")
			if got != tc.wantFire {
				t.Fatalf("fired = %v, want %v; violations: %+v", got, tc.wantFire, vs)
			}
			for _, want := range tc.wantDetail {
				if !violationMentions(vs, want) {
					t.Errorf("violation detail should contain %q; got %+v", want, vs)
				}
			}
			if !tc.wantFire {
				return
			}
			// A violation an author cannot navigate to is not actionable:
			// it must point at the skill row to move, and name a witness
			// emission so the claimed severity can be checked.
			if vs[0].File != skillCheckPath {
				t.Errorf("Violation.File = %q, want the skill path %q", vs[0].File, skillCheckPath)
			}
			if vs[0].Line <= 0 {
				t.Errorf("Violation.Line = %d, want the skill row's line", vs[0].Line)
			}
			if !violationMentions(vs, "internal/check/x.go:3") {
				t.Errorf("violation should cite the emitting site as a witness; got %+v", vs)
			}
		})
	}
}

// TestSkillTableSeverityPlacement_JudgesTheCLICheckLayer proves the
// second half of isCheckLayerFile is live: `aiwf check` surfaces
// findings emitted from internal/cli/check/ too, and narrowing the
// scan to internal/check/ would drop them silently.
func TestSkillTableSeverityPlacement_JudgesTheCLICheckLayer(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/cli/check/x.go", emitSrc("probe", "check.SeverityError"))
	mustWrite(t, filepath.Join(root, skillCheckPath), severitySkill(nil, []string{"probe"}, nil))

	vs, err := PolicySkillTableSeverityPlacement(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if !hasPolicyViolation(vs, "skill-table-severity-placement") {
		t.Fatalf("a misfiled CLI-check-layer code must fire; got %+v", vs)
	}
	if !violationMentions(vs, "internal/cli/check/x.go") {
		t.Errorf("violation should cite the CLI-layer witness; got %+v", vs)
	}
}

// TestSkillTableSeverityPlacement_ContradictoryDuplicateFires closes the
// hole a per-code map would leave: with two rows for one code, keeping
// only the last one read lets the other sit in a table that says the
// opposite, unjudged. An operator reading the errors table would be
// told the code blocks their push while the policy reports clean.
func TestSkillTableSeverityPlacement_ContradictoryDuplicateFires(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                          string
		errors, warnings, conditional []string
		wantFire                      bool
	}{
		{"same code in the errors and warnings tables", []string{"probe"}, []string{"probe"}, nil, true},
		{"same code in the errors and conditional tables", []string{"probe"}, nil, []string{"probe"}, true},
		// Repeated under one class the two rows agree, so there is no
		// contradiction about severity for this policy to report.
		{"repeated within one table", []string{"probe", "probe"}, nil, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeAt(t, root, "internal/check/x.go", emitSrc("probe", "SeverityError"))
			mustWrite(t, filepath.Join(root, skillCheckPath),
				severitySkill(tc.errors, tc.warnings, tc.conditional))

			vs, err := PolicySkillTableSeverityPlacement(root)
			if err != nil {
				t.Fatalf("policy error: %v", err)
			}
			if got := hasPolicyViolation(vs, "skill-table-severity-placement"); got != tc.wantFire {
				t.Fatalf("fired = %v, want %v; violations: %+v", got, tc.wantFire, vs)
			}
			if tc.wantFire && !violationMentions(vs, "contradicts itself") {
				t.Errorf("violation should name the contradiction; got %+v", vs)
			}
		})
	}
}

// TestSkillTableSeverityPlacement_PostPassEscalationCountsAsConditional
// is the reason the two escalation styles are not filed differently. A
// rule can pick its severity at the construction site, or emit a
// warning that a later `Apply…` pass rewrites to an error when a knob
// is set. A consumer cannot tell those apart — both reach them as
// "warning unless configured otherwise" — so both belong in the
// conditional table, and a fixed warning row would be wrong for either.
func TestSkillTableSeverityPlacement_PostPassEscalation(t *testing.T) {
	t.Parallel()
	// The pass also escalates a code with no row and no construction
	// site. A post-pass can name one — a code documented nowhere, or
	// emitted from outside the scanned layer — and it must be dropped
	// rather than invent a row to judge.
	const postPass = "package check\n\n" +
		"func ApplyProbeStrict(findings []Finding, strict bool) {\n" +
		"\tif !strict {\n\t\treturn\n\t}\n" +
		"\tfor i := range findings {\n" +
		"\t\tswitch findings[i].Code {\n" +
		"\t\tcase CodeProbe, CodeUnrouted:\n" +
		"\t\t\tfindings[i].Severity = SeverityError\n" +
		"\t\t}\n\t}\n}\n"
	const decl = "package check\n\nconst CodeProbe = \"probe\"\n\nconst CodeUnrouted = \"unrouted\"\n"
	emit := "package check\n\nvar _ = Finding{Code: CodeProbe, Severity: SeverityWarning}\n"

	cases := []struct {
		name                  string
		files                 map[string]string
		warnings, conditional []string
		wantFire              bool
		wantDetail            []string
	}{
		{
			name:        "escalated code in the conditional table is silent",
			files:       map[string]string{"internal/check/d.go": decl, "internal/check/x.go": emit, "internal/check/p.go": postPass},
			conditional: []string{"probe"},
			wantFire:    false,
		},
		{
			name:       "escalated code left in the warnings table fires",
			files:      map[string]string{"internal/check/d.go": decl, "internal/check/x.go": emit, "internal/check/p.go": postPass},
			warnings:   []string{"probe"},
			wantFire:   true,
			wantDetail: []string{"error, and warning", `"(conditional …)"`},
		},
		{
			// Without the post-pass the same emission is a plain
			// warning, so the warnings table is right — which is what
			// makes the case above attributable to the post-pass alone.
			name:     "the same code with no post-pass belongs in warnings",
			files:    map[string]string{"internal/check/d.go": decl, "internal/check/x.go": emit},
			warnings: []string{"probe"},
			wantFire: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for path, src := range tc.files {
				writeAt(t, root, path, src)
			}
			mustWrite(t, filepath.Join(root, skillCheckPath),
				severitySkill(nil, tc.warnings, tc.conditional))

			vs, err := PolicySkillTableSeverityPlacement(root)
			if err != nil {
				t.Fatalf("policy error: %v", err)
			}
			if got := hasPolicyViolation(vs, "skill-table-severity-placement"); got != tc.wantFire {
				t.Fatalf("fired = %v, want %v; violations: %+v", got, tc.wantFire, vs)
			}
			for _, want := range tc.wantDetail {
				if !violationMentions(vs, want) {
					t.Errorf("violation detail should contain %q; got %+v", want, vs)
				}
			}
		})
	}
}

// TestPostPassSeverities_SkipsUnparseableSource proves a check-layer
// file that does not parse costs only itself: the escalators in every
// other file are still found. A policy that gave up on the first
// syntax error would silently stop escalating mid-edit.
func TestPostPassSeverities_SkipsUnparseableSource(t *testing.T) {
	t.Parallel()
	files := []FileEntry{
		fe("internal/check/broken.go", "package check\n\nfunc ( this is not Go {{{\n"),
		fe("internal/check/decl.go", "package check\n\nconst CodeProbe = \"probe\"\n"),
		fe("internal/check/apply.go", "package check\n\n"+
			"func ApplyProbeStrict(findings []Finding) {\n"+
			"\tfor i := range findings {\n"+
			"\t\tif findings[i].Code == CodeProbe {\n"+
			"\t\t\tfindings[i].Severity = SeverityError\n"+
			"\t\t}\n\t}\n}\n"),
	}
	got := postPassSeverities(files)
	if got["probe"] != findingSeverityError {
		t.Errorf("postPassSeverities()[%q] = %q, want %q — an unparseable sibling must not suppress it",
			"probe", got["probe"], findingSeverityError)
	}
}

// TestAssignedSeverity pins what counts as a post-pass rewrite. Only a
// statement assigning a severity literal to a `.Severity` field does —
// a rule setting `Severity:` inside its Finding literal must not read
// as one, or every rule in the tree would look like an escalator.
func TestAssignedSeverity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want findingSeverity
	}{
		{"escalating assignment", "func f() {\n\tfindings[i].Severity = SeverityError\n}", findingSeverityError},
		{"de-escalating assignment", "func f() {\n\tfindings[i].Severity = SeverityWarning\n}", findingSeverityWarning},
		{"qualified literal", "func f() {\n\tf.Severity = check.SeverityError\n}", findingSeverityError},
		{"composite literal is not an assignment", "func f() {\n\t_ = Finding{Severity: SeverityError}\n}", ""},
		{"assignment of a computed value", "func f() {\n\tfindings[i].Severity = pick(x)\n}", ""},
		{"assignment to some other field", "func f() {\n\tfindings[i].Message = SeverityError\n}", ""},
		{"no assignment at all", "func f() {\n\treturn\n}", ""},
		{
			// A pass with one arm per severity settles on neither, the
			// same answer rowSeverity gives construction sites that
			// disagree. Reporting whichever arm the walk reached last
			// would make the result depend on source order.
			"arms that assign different severities",
			"func f() {\n\tif x {\n\t\tfindings[i].Severity = SeverityError\n\t} else {\n\t\tfindings[i].Severity = SeverityWarning\n\t}\n}",
			findingSeverityVaries,
		},
		{
			"the same severity on two arms still settles",
			"func f() {\n\tif x {\n\t\tfindings[i].Severity = SeverityError\n\t} else {\n\t\tfindings[i].Severity = check.SeverityError\n\t}\n}",
			findingSeverityError,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fset := token.NewFileSet()
			af, err := parser.ParseFile(fset, "x.go", "package check\n\n"+tc.body+"\n", parser.AllErrors)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			fn := af.Decls[0].(*ast.FuncDecl)
			if got := assignedSeverity(fn.Body); got != tc.want {
				t.Errorf("assignedSeverity() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestPostPassSeverities_FindsTheLiveEscalators is the live-tree
// counterpart: the `Apply…` passes really are discovered by shape, so a
// new one written the same way is picked up with no list to update.
func TestPostPassSeverities_FindsTheLiveEscalators(t *testing.T) {
	t.Parallel()
	root, err := repoRootFromTest(t)
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	files, err := WalkGoFiles(root, true)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	got := postPassSeverities(files)
	// One representative per live Apply… pass, so this fails if any of
	// them stops being recognized rather than only if all of them do.
	for _, code := range []string{
		"area-unknown",             // ApplyAreaRequiredStrict
		"doc-id-width",             // ApplyDocsStrict
		"milestone-tdd-undeclared", // ApplyTDDStrict
		"archive-sweep-pending",    // ApplyArchiveSweepThreshold
	} {
		if got[code] != findingSeverityError {
			t.Errorf("postPassSeverities()[%q] = %q, want %q", code, got[code], findingSeverityError)
		}
	}
	// A rule that only ever constructs its severity must not be read as
	// escalated, or the conditional table would swallow the whole tree.
	if _, escalated := got["ids-unique"]; escalated {
		t.Errorf("ids-unique has no post-pass but reads as escalated: %+v", got["ids-unique"])
	}
	// The ident sweep resolves through codeConsts, which indexes every
	// string constant in the check package — including SeverityError,
	// whose value is "error". Unfiltered, the sweep reads the severity
	// literal in the assignment as one more code the pass escalates.
	if sev, phantom := got["error"]; phantom {
		t.Errorf(`postPassSeverities invented a code named "error" -> %q; the Code-prefix filter is not holding`, sev)
	}
	for code := range got {
		if strings.Contains(code, " ") || code == "" {
			t.Errorf("postPassSeverities returned an implausible code %q", code)
		}
	}
}

// TestSkillTableSeverityPlacement_ViolationsAreOrdered pins the stable
// ordering. Without it the punch list is map-iteration order and
// reshuffles between runs, which makes a CI diff unreadable.
func TestSkillTableSeverityPlacement_ViolationsAreOrdered(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/check/x.go", "package check\n\n"+
		"var _ = Finding{Code: \"zeta\", Severity: SeverityError}\n"+
		"var _ = Finding{Code: \"alpha\", Severity: SeverityError}\n"+
		"var _ = Finding{Code: \"mid\", Severity: SeverityError}\n")
	mustWrite(t, filepath.Join(root, skillCheckPath),
		severitySkill(nil, []string{"zeta", "alpha", "mid"}, nil))

	vs, err := PolicySkillTableSeverityPlacement(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if len(vs) != 3 {
		t.Fatalf("want 3 violations, got %d: %+v", len(vs), vs)
	}
	for i, want := range []string{`"alpha"`, `"mid"`, `"zeta"`} {
		if !strings.Contains(vs[i].Detail, want) {
			t.Errorf("violation %d should name %s; got %q", i, want, vs[i].Detail)
		}
	}
}

// TestSkillTableSeverityPlacement_SubcodeRouting proves an emission is
// judged against the row that actually documents it: its own
// `code/subcode` row when one exists, otherwise the bare `code` row it
// falls back to.
func TestSkillTableSeverityPlacement_SubcodeRouting(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                          string
		errors, warnings, conditional []string
		wantFire                      bool
		wantRow                       string
	}{
		{
			name:     "the subcode's own row is what is judged",
			errors:   []string{"probe/variant"},
			warnings: []string{"probe"},
			wantFire: false,
		},
		{
			name:     "with no subcode row, the bare row is judged",
			errors:   []string{"probe"},
			wantFire: false,
		},
		{
			name:     "a misfiled bare row is named as the row, not the subcode",
			warnings: []string{"probe"},
			wantFire: true,
			wantRow:  `"probe"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeAt(t, root, "internal/check/x.go",
				"package check\n\nvar _ = Finding{Code: \"probe\", Subcode: \"variant\", Severity: SeverityError}\n")
			mustWrite(t, filepath.Join(root, skillCheckPath),
				severitySkill(tc.errors, tc.warnings, tc.conditional))

			vs, err := PolicySkillTableSeverityPlacement(root)
			if err != nil {
				t.Fatalf("policy error: %v", err)
			}
			if got := hasPolicyViolation(vs, "skill-table-severity-placement"); got != tc.wantFire {
				t.Fatalf("fired = %v, want %v; violations: %+v", got, tc.wantFire, vs)
			}
			if tc.wantRow != "" && !violationMentions(vs, tc.wantRow) {
				t.Errorf("violation should name row %s; got %+v", tc.wantRow, vs)
			}
		})
	}
}

// TestSkillTableSeverityPlacement_UndocumentedCodeIsSilent proves the
// policy stays inside its lane: a code with no row at all is
// PolicyFindingCodesDocumentedInSkill's finding, and reporting it here
// too would double-report one defect.
func TestSkillTableSeverityPlacement_UndocumentedCodeIsSilent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/check/x.go", emitSrc("undocumented", "SeverityError"))
	mustWrite(t, filepath.Join(root, skillCheckPath), severitySkill([]string{"unrelated"}, nil, nil))

	vs, err := PolicySkillTableSeverityPlacement(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("an undocumented code must not fire here; got %+v", vs)
	}
}

// TestSkillTableSeverityPlacement_UnclassifiedHeadingFires closes the
// fail-open a heading rename would otherwise leave: rows under a
// heading that declares no severity are constrained by nothing, so the
// rename would silently unclassify every row beneath it.
func TestSkillTableSeverityPlacement_UnclassifiedHeadingFires(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/check/x.go", emitSrc("probe", "SeverityError"))
	mustWrite(t, filepath.Join(root, skillCheckPath),
		"# aiwf-check\n\n## Findings\n\n| Code | Meaning |\n|---|---|\n| `probe` | meaning |\n")

	vs, err := PolicySkillTableSeverityPlacement(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if !hasPolicyViolation(vs, "skill-table-severity-placement") {
		t.Fatalf("a row under an unclassified heading must fire; got %+v", vs)
	}
	if !violationMentions(vs, "which declares no severity") {
		t.Errorf("violation should say the heading declares no severity; got %+v", vs)
	}
}

// TestSkillTableSeverityPlacement_IgnoresNonCheckLayerAndSeveritylessSites
// covers the two skips before routing: a Finding{} outside the check
// layer is surfaced by its own verb rather than by `aiwf check`, and a
// literal with no Severity field asserts nothing about severity — so
// neither may pull a code into the conditional table.
func TestSkillTableSeverityPlacement_IgnoresNonCheckLayerAndSeveritylessSites(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/check/x.go", emitSrc("probe", "SeverityError"))
	writeAt(t, root, "internal/check/silent.go",
		"package check\n\nvar _ = Finding{Code: \"probe\"}\n")
	writeAt(t, root, "internal/verb/y.go", emitSrc("probe", "SeverityWarning"))
	mustWrite(t, filepath.Join(root, skillCheckPath), severitySkill([]string{"probe"}, nil, nil))

	vs, err := PolicySkillTableSeverityPlacement(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("neither a verb-layer site nor a severity-less literal may fire; got %+v", vs)
	}
}

// TestSkillTableSeverityPlacement_Errors covers the two failure exits:
// an unwalkable root, and a root with no aiwf-check skill to read. The
// second is the fail-loud half — with no skill, nothing is constrained,
// so returning silence would be a pass that proves nothing.
func TestSkillTableSeverityPlacement_Errors(t *testing.T) {
	t.Parallel()
	t.Run("unwalkable root", func(t *testing.T) {
		t.Parallel()
		if _, err := PolicySkillTableSeverityPlacement(filepath.Join(t.TempDir(), "nope")); err == nil {
			t.Error("expected an error walking a nonexistent root")
		}
	})
	t.Run("missing skill", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeAt(t, root, "internal/check/x.go", emitSrc("probe", "SeverityError"))
		if _, err := PolicySkillTableSeverityPlacement(root); err == nil {
			t.Error("expected an error when the aiwf-check skill is absent")
		}
	})
}

// TestSkillSectionClass pins the heading-marker classifier, including
// the conditional marker winning over the word "errors" appearing
// elsewhere in the same heading.
func TestSkillSectionClass(t *testing.T) {
	t.Parallel()
	cases := []struct {
		heading string
		want    findingSeverity
	}{
		{"## Findings (errors)", findingSeverityError},
		{"## Provenance findings (errors)", findingSeverityError},
		{"## Findings (warnings)", findingSeverityWarning},
		{"## Findings (conditional severity)", findingSeverityVaries},
		{"## Findings (conditional) — may be (errors)", findingSeverityVaries},
		{"## Which id rule applies where", ""},
		{"## What to run", ""},
	}
	for _, tc := range cases {
		t.Run(tc.heading, func(t *testing.T) {
			t.Parallel()
			if got := skillSectionClass(tc.heading); got != tc.want {
				t.Errorf("skillSectionClass(%q) = %q, want %q", tc.heading, got, tc.want)
			}
		})
	}
}

// TestSkillHeadingMarkerFor pins the remediation half: the message names
// the marker an author greps the skill for, not the internal class name.
func TestSkillHeadingMarkerFor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		class findingSeverity
		want  string
	}{
		{findingSeverityError, "(errors)"},
		{findingSeverityWarning, "(warnings)"},
		{findingSeverityVaries, "(conditional …)"},
	}
	for _, tc := range cases {
		t.Run(string(tc.class), func(t *testing.T) {
			t.Parallel()
			if got := skillHeadingMarkerFor(tc.class); got != tc.want {
				t.Errorf("skillHeadingMarkerFor(%q) = %q, want %q", tc.class, got, tc.want)
			}
		})
	}
}

// TestSkillHeadingMarkerRoundTrip proves the classifier and the
// remediation message agree: the marker the message tells an author to
// use is one the classifier reads back as the severity that was asked
// for. Were they to drift, the policy would name an unreachable fix.
func TestSkillHeadingMarkerRoundTrip(t *testing.T) {
	t.Parallel()
	for _, class := range []findingSeverity{findingSeverityError, findingSeverityWarning, findingSeverityVaries} {
		t.Run(string(class), func(t *testing.T) {
			t.Parallel()
			heading := "## Findings " + skillHeadingMarkerFor(class)
			if got := skillSectionClass(heading); got != class {
				t.Errorf("heading %q built for %q classifies as %q", heading, class, got)
			}
		})
	}
}

// TestRowSeverityClass pins the collapse from observed severities to the
// one a heading can declare.
func TestRowSeverityClass(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		seen []findingSeverity
		want findingSeverity
	}{
		{"error everywhere", []findingSeverity{findingSeverityError}, findingSeverityError},
		{"warning everywhere", []findingSeverity{findingSeverityWarning}, findingSeverityWarning},
		{"non-literal only", []findingSeverity{findingSeverityVaries}, findingSeverityVaries},
		{"literals that disagree", []findingSeverity{findingSeverityError, findingSeverityWarning}, findingSeverityVaries},
		{"a literal and a non-literal", []findingSeverity{findingSeverityWarning, findingSeverityVaries}, findingSeverityVaries},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := rowSeverity{seen: map[findingSeverity]bool{}}
			for _, s := range tc.seen {
				r.seen[s] = true
			}
			if got := r.class(); got != tc.want {
				t.Errorf("class() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRowSeverityObserved pins the message's description of what was
// seen, including the stable ordering a multi-severity row renders in.
func TestRowSeverityObserved(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		seen []findingSeverity
		want string
	}{
		{"error everywhere", []findingSeverity{findingSeverityError}, "error at every site"},
		{"warning everywhere", []findingSeverity{findingSeverityWarning}, "warning at every site"},
		{
			"non-literal only",
			[]findingSeverity{findingSeverityVaries},
			"a Severity expression that is not a literal — so its severity is decided at run time",
		},
		{
			"literals that disagree",
			[]findingSeverity{findingSeverityWarning, findingSeverityError},
			"error, and warning — so its severity is decided at run time",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := rowSeverity{seen: map[findingSeverity]bool{}}
			for _, s := range tc.seen {
				r.seen[s] = true
			}
			if got := r.observed(); got != tc.want {
				t.Errorf("observed() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestSkillRowKeyFor pins the row an emission is routed to, including
// the miss that leaves a code unconstrained here.
func TestSkillRowKeyFor(t *testing.T) {
	t.Parallel()
	rows := map[string]skillFindingRow{
		"probe":         {},
		"probe/variant": {},
		"bare-only":     {},
	}
	cases := []struct {
		name    string
		site    findingCodeSite
		wantKey string
		wantOK  bool
	}{
		{"subcode with its own row", findingCodeSite{Code: "probe", Subcode: "variant"}, "probe/variant", true},
		{"subcode falling back to the bare row", findingCodeSite{Code: "bare-only", Subcode: "x"}, "bare-only", true},
		{"bare code with a row", findingCodeSite{Code: "probe"}, "probe", true},
		{"nothing documented", findingCodeSite{Code: "absent", Subcode: "x"}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			key, ok := skillRowKeyFor(tc.site, rows)
			if key != tc.wantKey || ok != tc.wantOK {
				t.Errorf("skillRowKeyFor() = (%q, %v), want (%q, %v)", key, ok, tc.wantKey, tc.wantOK)
			}
		})
	}
}

// TestLoadSkillFindingRows_RecordsHeadingAndLine proves rows outside a
// severity-declaring section are still returned — the distinction that
// lets the policy tell "documented somewhere unclassified" apart from
// "not documented at all" — and that a `###` subheading inside a
// findings section does not reset the severity its rows are filed under.
func TestLoadSkillFindingRows_RecordsHeadingAndLine(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, skillCheckPath),
		"# aiwf-check\n\n## What to run\n\n| Hook | Command |\n|---|---|\n| `pre-commit` | run it |\n\n"+
			"## Findings (errors)\n\n| Code | Meaning |\n|---|---|\n| `probe` | meaning |\n\n"+
			"### A subheading inside the section\n\n| Code | Meaning |\n|---|---|\n| `after-sub` | meaning |\n")

	rows, err := loadSkillFindingRows(root)
	if err != nil {
		t.Fatalf("loadSkillFindingRows: %v", err)
	}
	probe, ok := rows["probe"]
	if !ok {
		t.Fatalf("probe row missing; got %+v", rows)
	}
	if got, _ := probe.class(); got != findingSeverityError {
		t.Errorf("probe class = %q, want %q", got, findingSeverityError)
	}
	if probe.Line != 13 {
		t.Errorf("probe Line = %d, want 13", probe.Line)
	}
	sub, ok := rows["after-sub"]
	if !ok {
		t.Fatal("a row after a `###` subheading must still be recorded")
	}
	if got, _ := sub.class(); got != findingSeverityError {
		t.Errorf("after-sub class = %q, want %q — a `###` subheading is part of its `##` section, not a new one", got, findingSeverityError)
	}
	hook, ok := rows["pre-commit"]
	if !ok {
		t.Fatal("a backticked first cell outside a Findings section must still be recorded")
	}
	if got, _ := hook.class(); got != "" {
		t.Errorf("pre-commit class = %q, want the empty string", got)
	}
}

// TestSkillFindingRowClass pins the contradiction detector: rows
// repeated under one class agree, rows split across classes do not.
func TestSkillFindingRowClass(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		classes        []findingSeverity
		want           findingSeverity
		wantUnambigous bool
	}{
		{"one section", []findingSeverity{findingSeverityError}, findingSeverityError, true},
		{"repeated under one class", []findingSeverity{findingSeverityWarning, findingSeverityWarning}, findingSeverityWarning, true},
		{"split across classes", []findingSeverity{findingSeverityError, findingSeverityWarning}, "", false},
		{"classified and unclassified", []findingSeverity{findingSeverityError, ""}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := skillFindingRow{Classes: tc.classes}.class()
			if got != tc.want || ok != tc.wantUnambigous {
				t.Errorf("class() = (%q, %v), want (%q, %v)", got, ok, tc.want, tc.wantUnambigous)
			}
		})
	}
}

// TestSkillCheck_TablesAreWellFormed guards the tables this policy
// reads from a defect the policy itself cannot see: a row carrying more
// cells than its table's header declares.
//
// Markdown silently drops the surplus, so such a row renders with its
// last cell missing — the guidance is in the file, passes every grep,
// and reaches no reader. That failure is invisible to a severity check,
// which only looks at the first cell, and invisible to review, which
// reads the source rather than the render.
func TestSkillCheck_TablesAreWellFormed(t *testing.T) {
	t.Parallel()
	root, err := repoRootFromTest(t)
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, skillCheckPath))
	if err != nil {
		t.Fatalf("read aiwf-check skill: %v", err)
	}
	heading, want := "", 0
	for i, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(line, "#"):
			heading, want = strings.TrimSpace(line), 0
			continue
		case !strings.HasPrefix(line, "|"):
			continue
		}
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), " | ")
		if isTableDivider(line) {
			continue
		}
		if want == 0 {
			want = len(cells)
			continue
		}
		if len(cells) != want {
			t.Errorf("%s:%d under %q has %d cells but its header declares %d; markdown drops the surplus, so cell %d never renders",
				skillCheckPath, i+1, heading, len(cells), want, want+1)
		}
	}
}

// isTableDivider reports whether a table line is the `|---|---|`
// separator, which carries no cells to count.
func isTableDivider(line string) bool {
	return strings.Trim(line, "|-: \t") == ""
}

// noEscalationClaim matches a cell telling a reader that nothing
// raises this finding's severity. Placement cannot carry that claim —
// a warning row and a never-escalates warning row look identical — so
// the cells state it in prose, and prose is what drifts.
var noEscalationClaim = regexp.MustCompile(`no strictness knob|never escalates|does not escalate`)

// TestSkillCheck_NoEscalationClaimsAreTrue pins the prose half of the
// severity contract, which placement leaves unguarded.
//
// A row saying "no strictness knob" is making a promise about run-time
// behavior, and a reader acts on it by not configuring around the
// finding. When a later change adds that code to a strictness pass, the
// cell becomes a lie that no placement check can see: the code now
// varies, so the conditional table is where it belongs, and the false
// sentence rides along into the very table whose rows are supposed to
// be the authority.
func TestSkillCheck_NoEscalationClaimsAreTrue(t *testing.T) {
	t.Parallel()
	root, err := repoRootFromTest(t)
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	files, err := WalkGoFiles(root, true)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	rows, err := loadSkillFindingRows(root)
	if err != nil {
		t.Fatalf("read aiwf-check skill: %v", err)
	}
	escalated := postPassSeverities(files)
	checked := 0
	for code, row := range rows {
		for _, text := range row.Texts {
			if !noEscalationClaim.MatchString(text) {
				continue
			}
			checked++
			if sev, ok := escalated[code]; ok {
				t.Errorf("%q says nothing escalates it, but a strictness pass rewrites it to %q; "+
					"either drop the claim and name the knob, or stop escalating the code",
					code, sev)
			}
		}
	}
	// Without a claim to check the loop above proves nothing, and the
	// phrase list silently going stale is the way that happens.
	if checked == 0 {
		t.Fatalf("no row carries a no-escalation claim, so this test asserts nothing; "+
			"the patterns in %q no longer match how the skill words it", noEscalationClaim)
	}
}

// escalationClaim matches a cell telling a reader that something raises
// this finding's severity at run time.
var escalationClaim = regexp.MustCompile(`(?i)escalated to error|escalates to \*\*error\*\*|promoted to \*\*error\*\*|severity escalates`)

// TestSkillCheck_EscalationClaimsSitInTheConditionalTable is the mirror
// of the no-escalation pin, and closes the other half of the class that
// produced two separate false cells in review.
//
// A row in a fixed-severity table that describes an escalation is
// contradicting its own placement: the table says one severity always,
// the cell says two. Either the row is in the wrong table or the
// sentence is stale, and both readings are a defect. The placement
// policy cannot see this — it reads the rule, not the prose — so a cell
// can go on describing a knob long after the knob stops reaching it.
func TestSkillCheck_EscalationClaimsSitInTheConditionalTable(t *testing.T) {
	t.Parallel()
	root, err := repoRootFromTest(t)
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	rows, err := loadSkillFindingRows(root)
	if err != nil {
		t.Fatalf("read aiwf-check skill: %v", err)
	}
	checked := 0
	for code, row := range rows {
		for i, text := range row.Texts {
			// A cell may name another code's escalation, or deny its
			// own; neither is a claim about this row's severity.
			if !escalationClaim.MatchString(text) || noEscalationClaim.MatchString(text) {
				continue
			}
			checked++
			if row.Classes[i] != findingSeverityVaries {
				t.Errorf("%q describes an escalation but is filed under %q, which promises one severity; "+
					"move the row to the conditional table, or drop the sentence if nothing escalates it any more",
					code, row.Headings[i])
			}
		}
	}
	if checked == 0 {
		t.Fatalf("no row carries an escalation claim, so this test asserts nothing; "+
			"the patterns in %q no longer match how the skill words it", escalationClaim)
	}
}

// TestPolicy_SkillTableSeverityPlacement is the live-tree chokepoint:
// every finding code `aiwf check` emits is documented under a section
// whose heading declares the severity the rule actually emits.
func TestPolicy_SkillTableSeverityPlacement(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicySkillTableSeverityPlacement)
}

// TestPolicy_SkillTableSeverityPlacementJudgesMostOfTheTable is the
// positive control the chokepoint above needs. That test passes when
// the policy finds nothing wrong — which is also what it returns when
// it has stopped looking, so on its own it cannot tell a clean table
// from a narrowed scan. This one asserts the policy still reaches a
// substantial share of the documented rows.
//
// The floor is deliberately well below the current figure: it is a
// tripwire for a scan that collapses, not a ratchet on coverage, and a
// row this policy cannot reach is named in its doc comment rather than
// treated as a regression.
func TestPolicy_SkillTableSeverityPlacementJudgesMostOfTheTable(t *testing.T) {
	t.Parallel()
	root, err := repoRootFromTest(t)
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	files, err := WalkGoFiles(root, true)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	rows, err := loadSkillFindingRows(root)
	if err != nil {
		t.Fatalf("read aiwf-check skill: %v", err)
	}
	const floor = 70
	judged := len(routeCheckLayerSeverities(files, rows))
	if judged < floor {
		t.Fatalf("policy judges only %d of %d documented rows, want at least %d; "+
			"the scan has narrowed and a clean run no longer proves the table is right",
			judged, len(rows), floor)
	}
}

// TestSkillCheck_HasAConditionalSeveritySection guards the section the
// live tree needs to exist: without it the conditional-severity rules
// have nowhere legal to sit, and every one of them would fire.
func TestSkillCheck_HasAConditionalSeveritySection(t *testing.T) {
	t.Parallel()
	root, err := repoRootFromTest(t)
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	rows, err := loadSkillFindingRows(root)
	if err != nil {
		t.Fatalf("read aiwf-check skill: %v", err)
	}
	for _, row := range rows {
		if c, ok := row.class(); ok && c == findingSeverityVaries {
			return
		}
	}
	var headings []string
	for code, row := range rows {
		headings = append(headings, code+" → "+row.sections())
	}
	t.Fatalf("no aiwf-check section declares conditional severity; rows: %s", strings.Join(headings, ", "))
}
