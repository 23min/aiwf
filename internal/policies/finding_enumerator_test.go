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
