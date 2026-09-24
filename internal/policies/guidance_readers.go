package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// PolicyGuidanceReaders holds D-0091 for this repository's development
// guidance — a CLAUDE.md or AGENTS.md at any depth, the project router, and
// the documents the router links to. No acceptance criterion is evidenced by
// a sentence pinned there, so every test that reads one of those documents
// from the repository root is listed in guidanceReaderList, and each entry
// says what the test is: a pin E-0092 retires when its passage moves, a
// relationship check, or an absence check. The list is held equal to the
// tree's readers in both directions: a new reader fails until someone
// decides what it is, and a retired pin's entry must go with it.
//
// The rule asks whether a test reads the guidance, not whether it asserts a
// phrase: the first is decidable from syntax, the second is not. A test
// reads the guidance when it, or a function of its package it calls or
// passes along, names a guidance document — by a literal or a
// package-level constant — and resolves the repository root, through a
// repoRoot helper or a path climbing to it. A document of the same name in a
// fixture repository resolves no root and is not a read. The rule is
// internal to this repository (ADR-0053).
func PolicyGuidanceReaders(root string) ([]Violation, error) {
	readers, err := guidanceReaderTests(root)
	if err != nil {
		return nil, err
	}
	return guidanceReaderViolations(readers, guidanceReaderList), nil
}

// readerKind is what a listed guidance reader is.
type readerKind int

const (
	// readerPin pins a passage; E-0092 retires it when the passage moves.
	readerPin readerKind = iota + 1
	// readerRelationship compares the guidance with another artefact.
	readerRelationship
	// readerAbsence asserts that something is not in the guidance.
	readerAbsence
)

// guidanceReaderEntry is one listed reader and why it reads the guidance.
type guidanceReaderEntry struct {
	kind readerKind
	why  string
}

// guidanceReaderList names every test that reads development guidance, keyed
// by package directory and test name. The pins are E-0092's to retire as
// their passages move; the milestone that retires one deletes its entry.
var guidanceReaderList = map[string]guidanceReaderEntry{
	"internal/policies.TestAiwfArchive_AC6_ClaudeMdNamesArchiveConvention":    {readerPin, "pins the archive-convention commitment of CLAUDE.md"},
	"internal/policies.TestM0154_AC2_CLAUDEMDOperatorSetupAmended":            {readerPin, "pins the operator-setup passage of CLAUDE.md"},
	"internal/policies.TestM0195_AC5_SkillBodyDisciplineInClaudeMd":           {readerPin, "pins the shipped-surface discipline passage of CLAUDE.md"},
	"internal/policies.TestM0209_AC1_GeneralizedGateInClaudeMd":               {readerPin, "pins the declared-sequence gate passage of CLAUDE.md"},
	"internal/policies.TestM0211_AC1_ClaudeMdIdCollisionSplitInPlace":         {readerPin, "pins the id-collision resolution passage of CLAUDE.md"},
	"internal/policies.TestM0211_AC3_AuthoringRuleNamesDividingPrinciple":     {readerPin, "pins the audience-dividing passage of CLAUDE.md"},
	"internal/policies.TestM0234_AC4_ClaudeMdWorktreeSectionsCiteWorktreeAdd": {readerPin, "pins the worktree passages of CLAUDE.md"},
	"internal/policies.TestM0293_KernelGuidanceStatesForceIsHumanOnly":        {readerPin, "pins the force-is-human-only statements of CLAUDE.md"},
	"internal/policies.TestM083_AC2_CLAUDEMdCommitment2":                      {readerPin, "pins the stable-ids commitment of CLAUDE.md"},
	"internal/policies.TestPolicy_CLAUDEMDCLIConventionsLogging":              {readerPin, "pins the CLI-conventions logging passage of CLAUDE.md"},
	"internal/policies.TestPolicy_ClaudeMdTestDisciplineSection":              {readerPin, "pins the test-discipline section of CLAUDE.md"},
	"internal/policies.TestPolicy_M0128DocumentationHierarchy":                {readerPin, "pins the documentation-hierarchy section of CLAUDE.md"},
	"internal/policies.TestPolicy_M0132ClaudeMdDevcontainerSection":           {readerPin, "pins the devcontainer subsection of CLAUDE.md"},
	"internal/policies.TestPolicy_M0134ClaudeMdTestRunningSections":           {readerPin, "pins the test-running subsections of CLAUDE.md"},
	"internal/policies.TestPolicy_M0228SkillsPolicyBroadenedPrinciple":        {readerPin, "pins the skills-policy principle of CLAUDE.md"},
	"internal/policies.TestM0127_AC3_NoDanglingDocsPocv3References":           {readerAbsence, "no document references the retired pocv3 tree"},
	"internal/policies.TestM0290_AC4_NoNormativeDocOffersTheRetiredVerb":      {readerAbsence, "no normative document offers the retired verb"},
	"internal/policies.TestSkillEditProvenance_DocumentedInClaudeMd":          {readerAbsence, "the retired skill-edit mandate stays out"},
	"internal/policies.TestPolicy_ConfigFieldsAreDiscoverable":                {readerRelationship, "reads the router to leave routed documents out of the channels"},
	"internal/policies.TestPolicy_FindingCodesAreDiscoverable":                {readerRelationship, "reads the router to leave routed documents out of the channels"},
	"internal/policies.TestPolicy_GuidanceCeiling":                            {readerRelationship, "measures each host's primed load against its ceiling"},
	"internal/policies.TestPolicy_GuidanceFence":                              {readerRelationship, "judges commits that change the guidance"},
	"internal/policies.TestPolicy_GuidanceReaders":                            {readerRelationship, "reads the router to know which documents are guidance"},
}

// guidanceReaderListFile is where guidanceReaderList lives, for a finding
// about the list itself.
const guidanceReaderListFile = "internal/policies/guidance_readers.go"

// guidanceReaderViolations compares the readers found, keyed as the list is
// and mapped to the file declaring each, with the list.
func guidanceReaderViolations(readers map[string]string, list map[string]guidanceReaderEntry) []Violation {
	var out []Violation
	for _, key := range sortedMapKeys(readers) {
		if _, ok := list[key]; !ok {
			out = append(out, Violation{
				Policy: "guidance-readers",
				File:   readers[key],
				Detail: fmt.Sprintf("%s reads development guidance from the repository root and is not in guidanceReaderList. D-0091 bars evidencing a criterion with a sentence pinned there: if the test compares the guidance with another artefact or asserts an absence, list it with that reason; if it pins a phrase, state the claim as a relationship check or record it as an observation.", key),
			})
		}
	}
	for _, key := range sortedMapKeys(list) {
		if _, ok := readers[key]; !ok {
			out = append(out, Violation{
				Policy: "guidance-readers",
				File:   guidanceReaderListFile,
				Detail: fmt.Sprintf("guidanceReaderList names %s, which reads no development guidance; delete the entry.", key),
			})
		}
	}
	return out
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// guidanceReaderTests returns every test in the tree that reads development
// guidance, keyed by package directory and test name, mapped to its file.
func guidanceReaderTests(root string) (map[string]string, error) {
	dirs, err := testPackageDirs(root)
	if err != nil {
		return nil, err
	}
	namesGuidance := repoNamesGuidance(root)
	out := map[string]string{}
	for _, dir := range dirs {
		files, paths, err := parsePackageDir(root, dir)
		if err != nil {
			return nil, err
		}
		for name, file := range readersInPackage(files, paths, namesGuidance) {
			out[dir+"."+name] = file
		}
	}
	return out, nil
}

// parsePackageDir parses every Go file in one directory, test files and
// the rest together, since a test reaches the guidance through either.
func parsePackageDir(root, dir string) ([]*ast.File, map[*ast.File]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil { //coverage:ignore testPackageDirs just walked this directory to list it
		return nil, nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	paths := map[*ast.File]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		rel := dir + "/" + e.Name()
		f, err := parser.ParseFile(fset, filepath.Join(root, filepath.FromSlash(rel)), nil, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing %s: %w", rel, err)
		}
		files = append(files, f)
		paths[f] = rel
	}
	return files, paths, nil
}

// readersInPackage returns the tests in one parsed package that read
// development guidance, mapped to their files. What a function names and
// resolves is followed through every function or method of the package it
// calls or passes as a value, to a fixpoint.
func readersInPackage(files []*ast.File, paths map[*ast.File]string, namesGuidance func(string) bool) map[string]string {
	funcs := map[string]*ast.FuncDecl{}
	consts := map[string]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			switch decl := d.(type) {
			case *ast.FuncDecl:
				// Methods are keyed by name as functions are: a call
				// `x.m(…)` reaches m's body the same way `m(…)` does.
				if _, seen := funcs[decl.Name.Name]; decl.Body != nil && !seen {
					funcs[decl.Name.Name] = decl
				}
			case *ast.GenDecl:
				for _, sp := range decl.Specs {
					vs, ok := sp.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, nm := range vs.Names {
						if i < len(vs.Values) && literalNamesGuidance(vs.Values[i], namesGuidance) {
							consts[nm.Name] = true
						}
					}
				}
			}
		}
	}

	type reach struct{ names, roots bool }
	own := map[string]reach{}
	edges := map[string][]string{}
	for name, fd := range funcs {
		var r reach
		phrases := phraseListElements(fd.Body)
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BasicLit:
				if x.Kind == token.STRING && !phrases[x] && namesGuidance(litValue(x)) {
					r.names = true
					// A relative path climbing to the guidance resolves the
					// root on its own.
					if strings.HasPrefix(litValue(x), "../") {
						r.roots = true
					}
				}
			case *ast.Ident:
				if consts[x.Name] {
					r.names = true
				}
				if _, ok := funcs[x.Name]; ok && x.Name != name {
					edges[name] = append(edges[name], x.Name)
				}
			case *ast.CallExpr:
				if strings.HasPrefix(calleeFuncName(x.Fun), "repoRoot") {
					r.roots = true
				}
			}
			return true
		})
		own[name] = r
	}
	// Propagate along the edges until nothing changes; a cycle settles.
	total := map[string]reach{}
	for name, r := range own {
		total[name] = r
	}
	for changed := true; changed; {
		changed = false
		for name, callees := range edges {
			r := total[name]
			for _, c := range callees {
				r.names = r.names || total[c].names
				r.roots = r.roots || total[c].roots
			}
			if r != total[name] {
				total[name], changed = r, true
			}
		}
	}

	out := map[string]string{}
	for _, f := range files {
		if !strings.HasSuffix(paths[f], "_test.go") {
			continue
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv != nil || !strings.HasPrefix(fd.Name.Name, "Test") || fd.Name.Name == "TestMain" {
				continue
			}
			if r := total[fd.Name.Name]; r.names && r.roots {
				out[fd.Name.Name] = paths[f]
			}
		}
	}
	return out
}

// phraseListElements returns the string literals that are elements of a
// `[]string{…}` list. Such a list holds phrases to look for, and a phrase
// that happens to read "CLAUDE.md" names no document; a path is built or
// passed, or sits in a table row, a call or a constant.
func phraseListElements(body *ast.BlockStmt) map[*ast.BasicLit]bool {
	out := map[*ast.BasicLit]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		arr, ok := cl.Type.(*ast.ArrayType)
		if !ok {
			return true
		}
		if elt, ok := arr.Elt.(*ast.Ident); !ok || elt.Name != "string" {
			return true
		}
		for _, e := range cl.Elts {
			if lit, ok := e.(*ast.BasicLit); ok {
				out[lit] = true
			}
		}
		return true
	})
	return out
}

// literalNamesGuidance reports whether e contains a string literal naming a
// guidance document.
func literalNamesGuidance(e ast.Expr, namesGuidance func(string) bool) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && namesGuidance(litValue(lit)) {
			found = true
		}
		return true
	})
	return found
}

// namesGuidancePath reports whether a string literal names a development
// guidance document: a CLAUDE.md or AGENTS.md at any depth, the project
// router, or one of the documents the router links to.
func namesGuidancePath(s string, routed map[string]bool) bool {
	s = path.Clean(filepath.ToSlash(s))
	for strings.HasPrefix(s, "../") {
		s = strings.TrimPrefix(s, "../")
	}
	switch path.Base(s) {
	case fenceClaudeMD, fenceAgentsMD:
		return true
	}
	return s == fenceRouter || strings.HasSuffix(s, "/"+fenceRouter) || routed[s]
}

// repoNamesGuidance builds the guidance-path predicate for the tree at root,
// with the router's routes read from disk.
func repoNamesGuidance(root string) func(string) bool {
	routed := map[string]bool{}
	if router, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fenceRouter))); err == nil {
		for _, p := range routedDocuments(string(router)) {
			routed[p] = true
		}
	}
	return func(s string) bool { return namesGuidancePath(s, routed) }
}
