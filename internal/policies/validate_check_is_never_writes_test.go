package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSrcFixture writes src to <root>/<rel> (creating parent dirs) so
// a policy can scan it. rel is a forward-slash repo-relative path. (The
// arity-2 writeFixture in walk_test.go writes a fixed stub; this one
// carries caller-supplied source.)
func writeSrcFixture(t *testing.T, root, rel, src string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

// firedFor reports whether any violation names fn in its Detail (the
// policy embeds the function name in every Detail string).
func firedFor(violations []Violation, fn string) bool {
	for _, v := range violations {
		if strings.Contains(v.Detail, fn+" is a query-family function") {
			return true
		}
	}
	return false
}

// TestPolicyValidateCheckIsNeverWrites_FiresAcrossPrimitivesAndFamilies
// is the core fixture: one synthetic file holds a function per
// (family × primitive) combination that must fire, alongside the
// precision and word-boundary cases that must not. Asserting the exact
// fire/no-fire partition pins both the catch and the false-positive
// guards in one place.
func TestPolicyValidateCheckIsNeverWrites_FiresAcrossPrimitivesAndFamilies(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// Not type-checked — parser.ParseFile is syntax-only, so unresolved
	// package references (gitops, pathutil) parse fine without imports.
	src := `package fixture

// --- must FIRE: a query-family name reaching a write primitive ---

func ValidateAndWrite(p string) error { return os.WriteFile(p, nil, 0o644) }

func IsReadyAtomic(p string) bool { _ = pathutil.AtomicWriteFile(p, nil, 0o644); return true }

func CheckAndCommit(root string) error { return gitops.Commit(root, "msg", nil) }

func HasStaleOpenWrite(p string) bool {
	_, _ = os.OpenFile(p, os.O_CREATE|os.O_WRONLY, 0o644)
	return false
}

func ValidateRemoves(p string) error { return os.RemoveAll(p) }

// name == family, exact (no suffix).
func Check(dir string) error { return os.Mkdir(dir, 0o755) }

// --- must NOT fire ---

// AST-selector precision: AddCommitSHA is a read despite the Add prefix.
func IsBirthKnown(root, p string) bool { _, _ = gitops.AddCommitSHA(root, p); return true }

// read-mode OpenFile and a plain Open are not writes.
func HasOpenForRead(p string) bool {
	_, _ = os.OpenFile(p, os.O_RDONLY, 0)
	_, _ = os.Open(p)
	return false
}

// non-selector call, selector-of-selector (os.Stdout is a value), and a
// non-write package selector all fall through writePrimitive cleanly.
func IsStdoutOK() bool {
	helper()
	_, _ = os.Stdout.Write(nil)
	fmt.Println("ok")
	return true
}

// word boundary: Is + lowercase 's' (Issue) and Has + lowercase 'h'
// (Hash) are nouns, not query families — never inspected.
func Issue(p string) error { return os.WriteFile(p, nil, 0o644) }
func Hash(p string) error  { _, err := os.Create(p); return err }

// not a query-family name at all.
func writeCache(p string) error { return os.WriteFile(p, nil, 0o644) }

// body-less declaration (assembly-style) exercises the fn.Body == nil skip.
func IsExternal() bool
`
	writeSrcFixture(t, root, "internal/fixture/fixture.go", src)

	violations, err := PolicyValidateCheckIsNeverWrites(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}

	mustFire := []string{
		"ValidateAndWrite", "IsReadyAtomic", "CheckAndCommit",
		"HasStaleOpenWrite", "ValidateRemoves", "Check",
	}
	for _, fn := range mustFire {
		if !firedFor(violations, fn) {
			t.Errorf("expected a violation for %s; got %+v", fn, violations)
		}
	}
	mustNotFire := []string{
		"IsBirthKnown", "HasOpenForRead", "IsStdoutOK",
		"Issue", "Hash", "writeCache", "IsExternal",
	}
	for _, fn := range mustNotFire {
		if firedFor(violations, fn) {
			t.Errorf("did not expect a violation for %s; got %+v", fn, violations)
		}
	}
}

// TestPolicyValidateCheckIsNeverWrites_NamesPrimitiveAndLine proves the
// finding points at the offending call line (not the func decl) and
// names the primitive, so a contributor can act on it directly.
func TestPolicyValidateCheckIsNeverWrites_NamesPrimitiveAndLine(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := `package fixture

func IsCacheWarm(p string) bool {
	_ = os.WriteFile(p, nil, 0o644)
	return true
}
`
	writeSrcFixture(t, root, "internal/fixture/cache.go", src)

	violations, err := PolicyValidateCheckIsNeverWrites(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly one violation; got %+v", violations)
	}
	v := violations[0]
	if v.Policy != "validate-check-is-never-writes" {
		t.Errorf("policy id = %q", v.Policy)
	}
	if v.File != "internal/fixture/cache.go" {
		t.Errorf("file = %q", v.File)
	}
	if v.Line != 4 { // the os.WriteFile call line, not the func decl (line 3)
		t.Errorf("line = %d, want 4 (the call site)", v.Line)
	}
	if !strings.Contains(v.Detail, "os.WriteFile") {
		t.Errorf("detail does not name the primitive: %q", v.Detail)
	}
}

// TestPolicyValidateCheckIsNeverWrites_OnlyScansInternal proves a
// query-family writer outside internal/ is not flagged: the policy's
// scope is the kernel's internal packages.
func TestPolicyValidateCheckIsNeverWrites_OnlyScansInternal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := `package main

func IsFlagSet(p string) bool { _ = os.WriteFile(p, nil, 0o644); return true }
`
	writeSrcFixture(t, root, "cmd/aiwf/flags.go", src)

	violations, err := PolicyValidateCheckIsNeverWrites(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if firedFor(violations, "IsFlagSet") {
		t.Errorf("policy fired outside internal/: %+v", violations)
	}
}

// TestPolicyValidateCheckIsNeverWrites_SkipsUnparseableFile proves the
// scan tolerates a .go file that does not parse rather than erroring,
// even one that textually contains a firing pattern.
func TestPolicyValidateCheckIsNeverWrites_SkipsUnparseableFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := `package broken

func IsBroken( {  // deliberate syntax error
	os.WriteFile("x", nil, 0o644)
`
	writeSrcFixture(t, root, "internal/broken/broken.go", src)

	violations, err := PolicyValidateCheckIsNeverWrites(root)
	if err != nil {
		t.Fatalf("policy errored on an unparseable file: %v", err)
	}
	if firedFor(violations, "IsBroken") {
		t.Errorf("policy fired on an unparseable file it should have skipped: %+v", violations)
	}
}
