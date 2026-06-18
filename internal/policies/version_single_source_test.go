package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPolicyVersionSingleSource_FiresOnParallelGlobal proves the
// policy catches the regression class it exists to prevent: a
// production package outside internal/version declaring a
// package-level string var that acts as a parallel binary-version
// source (the exact shape of the pre-fix internal/cli.Version global).
func TestPolicyVersionSingleSource_FiresOnParallelGlobal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "cli", "drift")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Untyped declaration with a string-literal initializer.
	src := `package drift

var Version = "dev"

func use() string { return Version }
`
	if err := os.WriteFile(filepath.Join(dir, "drift.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyVersionSingleSource(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.File == "internal/cli/drift/drift.go" && v.Line == 3 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected violation on drift.go:3 (var Version); got: %+v", violations)
	}
}

// TestPolicyVersionSingleSource_FiresOnTypedStampGlobal covers the
// explicit-`string`-type branch of valueSpecIsString and the
// case-insensitive name match against a differently-named stamp
// global.
func TestPolicyVersionSingleSource_FiresOnTypedStampGlobal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "build")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package build

var BuildStamp string
`
	if err := os.WriteFile(filepath.Join(dir, "build.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyVersionSingleSource(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.File == "internal/build/build.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected violation on build.go (var BuildStamp string); got: %+v", violations)
	}
}

// TestPolicyVersionSingleSource_SkipsUnparseableFile proves the
// policy gracefully skips a .go file that does not parse (rather than
// erroring or panicking) — even one that textually contains a version
// global. Production trees never carry unparseable files, but the
// guard keeps the policy robust.
func TestPolicyVersionSingleSource_SkipsUnparseableFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package broken

var Version = "dev"

func incomplete( {  // deliberate syntax error
`
	if err := os.WriteFile(filepath.Join(dir, "broken.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyVersionSingleSource(root)
	if err != nil {
		t.Fatalf("policy errored on an unparseable file: %v", err)
	}
	for _, v := range violations {
		if v.File == "internal/broken/broken.go" {
			t.Errorf("policy fired on an unparseable file it should have skipped: %+v", v)
		}
	}
}

// TestPolicyVersionSingleSource_AcceptsVersionPackageHome proves the
// one legitimate home for the Stamp global — internal/version — is
// allowlisted by path.
func TestPolicyVersionSingleSource_AcceptsVersionPackageHome(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "version")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package version

var Stamp string
`
	if err := os.WriteFile(filepath.Join(dir, "version.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyVersionSingleSource(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	for _, v := range violations {
		if v.File == "internal/version/version.go" {
			t.Errorf("policy fired on the legitimate internal/version home: %+v", v)
		}
	}
}

// TestPolicyVersionSingleSource_IgnoresNonStringAndLocal covers the
// false branches of valueSpecIsString (a non-string-typed global with
// a version-ish name) and confirms the policy is file-scope-only (a
// function-local var named version is never a declaration the policy
// inspects).
func TestPolicyVersionSingleSource_IgnoresNonStringAndLocal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "counter")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package counter

// version here is an int counter, not a version string.
var version int

// Version names a global but its initializer is a non-literal
// expression, so valueSpecIsString cannot prove it is a string and
// the policy must not fire.
var Version = version

func bump() string {
	version := "local-not-a-global"
	return version
}
`
	if err := os.WriteFile(filepath.Join(dir, "counter.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyVersionSingleSource(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	for _, v := range violations {
		if v.File == "internal/counter/counter.go" {
			t.Errorf("policy fired on a non-string global / local var: %+v", v)
		}
	}
}
