package policies

import (
	"strings"
	"testing"
)

// fe builds a FileEntry from a repo-relative path and Go source. The
// enumerator parses from Contents (AbsPath is only used for positions),
// so no file needs to exist on disk.
func fe(path, src string) FileEntry {
	return FileEntry{Path: path, AbsPath: path, Contents: []byte(src)}
}

// TestEmittedFindingCodeSites_ResolutionEdges exercises the resolver's
// edge branches: a non-string const, a descriptor whose ID is not a
// string literal, and a `.ID` selector on a non-identifier expression —
// each of which must resolve to "" (the code is skipped) without
// disturbing the codes that do resolve.
func TestEmittedFindingCodeSites_ResolutionEdges(t *testing.T) {
	t.Parallel()
	files := []FileEntry{
		fe("internal/check/decl.go", "package check\n\n"+
			// non-string const → loadCheckCodeConstants skips it
			"const IntConst = 5\n"+
			// descriptor whose ID is not a string literal → compositeLitStringField skips it
			"var Weird = codespkg.Code{ID: someIdent}\n"+
			// a resolvable descriptor, to prove resolution still works alongside the skips
			"var CodeReal = codespkg.Code{ID: \"real-code\"}\n"),
		fe("internal/check/emit.go", "package check\n\n"+
			// `.ID` selector on a call expr → resolveStringExpr falls through to \"\"
			"var _ = Finding{Code: makeCode().ID}\n"+
			// bare use of the resolvable descriptor
			"var _ = Finding{Code: CodeReal.ID}\n"),
	}
	sites := emittedFindingCodeSites(files)
	var codes []string
	for _, s := range sites {
		codes = append(codes, s.Code)
	}
	joined := strings.Join(codes, ",")
	if !strings.Contains(joined, "real-code") {
		t.Errorf("expected the resolvable descriptor code %q in emitted sites; got %v", "real-code", codes)
	}
	// The unresolvable `makeCode().ID` site resolves to "" and is skipped,
	// so it must not appear as an empty-code site.
	for _, s := range sites {
		if s.Code == "" {
			t.Errorf("emitted site has empty code (should have been skipped): %+v", s)
		}
	}
}

// TestEmittedFindingCodeSites_ResolvesCrossPackageStringConstant pins the
// selector arm that carries no `.ID`: `check.CodeFoo` names a
// string-constant code declared in the check package, which is how the
// CLI layer emits codes the check package owns. Resolving only
// `pkg.CodeFoo.ID` would leave those sites invisible to every policy
// built on this enumerator.
func TestEmittedFindingCodeSites_ResolvesCrossPackageStringConstant(t *testing.T) {
	t.Parallel()
	files := []FileEntry{
		fe("internal/check/decl.go", "package check\n\nconst CodeScopeUndefined = \"scope-undefined\"\n"),
		fe("internal/cli/check/emit.go", "package check\n\n"+
			"var _ = Finding{Code: check.CodeScopeUndefined, Severity: check.SeverityWarning}\n"+
			// A selector naming nothing the check package declares stays
			// unresolved, so the qualifier-dropping lookup cannot invent codes.
			"var _ = Finding{Code: entity.KindGap}\n"),
	}
	var got []findingCodeSite
	for _, s := range emittedFindingCodeSites(files) {
		if s.File == "internal/cli/check/emit.go" {
			got = append(got, s)
		}
	}
	if len(got) != 1 {
		t.Fatalf("want exactly 1 resolved site in the CLI file, got %d: %+v", len(got), got)
	}
	if got[0].Code != "scope-undefined" {
		t.Errorf("Code = %q, want %q", got[0].Code, "scope-undefined")
	}
	if got[0].Severity != findingSeverityWarning {
		t.Errorf("Severity = %q, want %q", got[0].Severity, findingSeverityWarning)
	}
}

// TestEmittedFindingCodeSites_Severity pins the severity a site is read
// as carrying, across every shape the check layer writes: a bare
// literal, a package-qualified one, a local assigned on a branch, a
// call, and no Severity field at all.
func TestEmittedFindingCodeSites_Severity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		expr string
		want findingSeverity
	}{
		{"bare error literal", "SeverityError", findingSeverityError},
		{"bare warning literal", "SeverityWarning", findingSeverityWarning},
		{"qualified error literal", "check.SeverityError", findingSeverityError},
		{"qualified warning literal", "check.SeverityWarning", findingSeverityWarning},
		{"local assigned on a branch", "severity", findingSeverityVaries},
		{"call", "severityFor(subcode)", findingSeverityVaries},
		{"qualified non-severity ident", "check.Something", findingSeverityVaries},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			files := []FileEntry{fe("internal/check/x.go",
				"package check\n\nvar _ = Finding{Code: \"sev-probe\", Severity: "+tc.expr+"}\n")}
			sites := emittedFindingCodeSites(files)
			if len(sites) != 1 {
				t.Fatalf("want 1 site, got %d: %+v", len(sites), sites)
			}
			if sites[0].Severity != tc.want {
				t.Errorf("Severity = %q, want %q", sites[0].Severity, tc.want)
			}
		})
	}
}

// TestEmittedFindingCodeSites_SeverityAbsent proves a literal with no
// Severity field reads as "" rather than as varying — the two mean
// different things to the placement policy, which must not let a
// silent literal drag its code into the conditional table.
func TestEmittedFindingCodeSites_SeverityAbsent(t *testing.T) {
	t.Parallel()
	files := []FileEntry{fe("internal/check/x.go",
		"package check\n\nvar _ = Finding{Code: \"no-sev\"}\n")}
	sites := emittedFindingCodeSites(files)
	if len(sites) != 1 {
		t.Fatalf("want 1 site, got %d: %+v", len(sites), sites)
	}
	if sites[0].Severity != "" {
		t.Errorf("Severity = %q, want the empty string", sites[0].Severity)
	}
}

// TestFindingCodesHaveHints_FiresWithSubcode covers the subcode branch of
// the hint policy's violation-detail: a hint-missing code carrying a
// subcode renders the ", Subcode: …" clause.
func TestFindingCodesHaveHints_FiresWithSubcode(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/check/x.go",
		"package check\n\nvar _ = Finding{Code: \"nohint-code\", Subcode: \"variant\"}\n")
	// no internal/check/hint.go → empty hint table → the code fires.
	vs, err := PolicyFindingCodesHaveHints(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	found := false
	for _, v := range vs {
		if strings.Contains(v.Detail, "nohint-code") && strings.Contains(v.Detail, "Subcode: \"variant\"") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a hint violation naming the code and its subcode; got %+v", vs)
	}
}
