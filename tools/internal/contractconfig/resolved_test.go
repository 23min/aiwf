package contractconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/aiwfyaml"
	"github.com/23min/ai-workflow-v2/tools/internal/check"
)

// realRoot returns repoRoot with symlinks evaluated. Tests use this so
// macOS's /var → /private/var symlink doesn't make the helper see a
// repoRoot that's "outside" itself.
func realRoot(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	r, err := filepath.EvalSymlinks(d)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func entry(id, schema, fixtures string) aiwfyaml.Entry {
	return aiwfyaml.Entry{ID: id, Validator: "v", Schema: schema, Fixtures: fixtures}
}

func writeFile(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func hasEscape(findings []check.Finding, id, kind string) bool {
	for i := range findings {
		f := &findings[i]
		if f.Code == "contract-config" && f.Subcode == "path-escape" && f.EntityID == id {
			if strings.Contains(f.Message, kind) {
				return true
			}
		}
	}
	return false
}

func TestResolve_nilContracts(t *testing.T) {
	root := realRoot(t)
	resolved, findings := Resolve(root, nil)
	if resolved != nil {
		t.Errorf("Resolve(nil): want nil resolved, got %v", resolved)
	}
	if findings != nil {
		t.Errorf("Resolve(nil): want nil findings, got %v", findings)
	}
}

func TestResolve_bothPathsInsideAndExist(t *testing.T) {
	root := realRoot(t)
	writeFile(t, filepath.Join(root, "schema.cue"))
	mkdir(t, filepath.Join(root, "fixtures"))

	entries := []aiwfyaml.Entry{entry("C-001", "schema.cue", "fixtures")}
	resolved, findings := Resolve(root, entries)

	if got := len(findings); got != 0 {
		t.Errorf("findings: want 0, got %d (%+v)", got, findings)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved: want 1, got %d", len(resolved))
	}
	if resolved[0].Skip {
		t.Error("resolved.Skip: want false")
	}
	if !strings.HasSuffix(resolved[0].SchemaPath, "schema.cue") {
		t.Errorf("schema path = %q, want suffix schema.cue", resolved[0].SchemaPath)
	}
	if !strings.HasSuffix(resolved[0].FixturesPath, "fixtures") {
		t.Errorf("fixtures path = %q, want suffix fixtures", resolved[0].FixturesPath)
	}
}

func TestResolve_bothPathsInsideMissing(t *testing.T) {
	root := realRoot(t)
	entries := []aiwfyaml.Entry{entry("C-001", "schema.cue", "fixtures")}
	resolved, findings := Resolve(root, entries)
	if len(findings) != 0 {
		t.Errorf("missing-but-inside should not raise path-escape: %+v", findings)
	}
	if resolved[0].Skip {
		t.Error("missing-but-inside should not be marked Skip")
	}
}

func TestResolve_dotdotEscape_schema(t *testing.T) {
	root := realRoot(t)
	entries := []aiwfyaml.Entry{entry("C-001", "../../etc/passwd", "fixtures")}
	resolved, findings := Resolve(root, entries)

	if !hasEscape(findings, "C-001", "schema") {
		t.Errorf("want path-escape for schema; got findings %+v", findings)
	}
	for _, f := range findings {
		if !strings.Contains(f.Message, `"../../etc/passwd"`) {
			t.Errorf("message should quote configured path verbatim; got %q", f.Message)
		}
		if strings.Contains(f.Message, root) {
			t.Errorf("message should not leak resolved/host path; got %q", f.Message)
		}
	}
	if !resolved[0].Skip {
		t.Error("escaped entry must be marked Skip")
	}
}

func TestResolve_absoluteEscape_fixtures(t *testing.T) {
	root := realRoot(t)
	outside := realRoot(t)
	entries := []aiwfyaml.Entry{entry("C-001", "schema.cue", outside)}
	resolved, findings := Resolve(root, entries)

	if !hasEscape(findings, "C-001", "fixtures") {
		t.Errorf("want path-escape for fixtures; got %+v", findings)
	}
	if !resolved[0].Skip {
		t.Error("escaped entry must be Skip")
	}
}

func TestResolve_symlinkOutsideRepo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix symlinks")
	}
	root := realRoot(t)
	outside := realRoot(t)
	target := filepath.Join(outside, "elsewhere")
	writeFile(t, target)

	link := filepath.Join(root, "schema.cue")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	entries := []aiwfyaml.Entry{entry("C-001", "schema.cue", "fixtures")}
	resolved, findings := Resolve(root, entries)

	if !hasEscape(findings, "C-001", "schema") {
		t.Errorf("symlink-outside should produce path-escape; got %+v", findings)
	}
	if !resolved[0].Skip {
		t.Error("symlink-escape entry must be Skip")
	}
}

func TestResolve_symlinkInsideRepo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix symlinks")
	}
	root := realRoot(t)
	target := filepath.Join(root, "actual.cue")
	writeFile(t, target)
	link := filepath.Join(root, "schema.cue")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	mkdir(t, filepath.Join(root, "fixtures"))

	entries := []aiwfyaml.Entry{entry("C-001", "schema.cue", "fixtures")}
	resolved, findings := Resolve(root, entries)

	if len(findings) != 0 {
		t.Errorf("inside-symlink should not raise findings: %+v", findings)
	}
	if resolved[0].Skip {
		t.Error("inside-symlink must not be Skip")
	}
}

func TestResolve_symlinkLoop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix symlinks")
	}
	root := realRoot(t)
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatal(err)
	}

	entries := []aiwfyaml.Entry{entry("C-001", "a", "fixtures")}
	resolved, findings := Resolve(root, entries)

	if !hasEscape(findings, "C-001", "schema") {
		t.Errorf("symlink-loop should fail closed as path-escape; got %+v", findings)
	}
	if !resolved[0].Skip {
		t.Error("loop entry must be Skip")
	}
}

func TestResolve_bothPathsEscape(t *testing.T) {
	root := realRoot(t)
	entries := []aiwfyaml.Entry{entry("C-001", "../escape1", "../escape2")}
	resolved, findings := Resolve(root, entries)

	if !hasEscape(findings, "C-001", "schema") {
		t.Error("want schema path-escape")
	}
	if !hasEscape(findings, "C-001", "fixtures") {
		t.Error("want fixtures path-escape")
	}
	if !resolved[0].Skip {
		t.Error("entry must be Skip")
	}
}

func TestResolve_emptyConfiguredPath(t *testing.T) {
	root := realRoot(t)
	entries := []aiwfyaml.Entry{entry("C-001", "", "fixtures")}
	resolved, findings := Resolve(root, entries)
	if !hasEscape(findings, "C-001", "schema") {
		t.Errorf("empty schema must raise path-escape; got %+v", findings)
	}
	if !resolved[0].Skip {
		t.Error("empty-path entry must be Skip")
	}
}

func TestResolve_relativeRepoRootRejected(t *testing.T) {
	// Relative repoRoot is a usage error: callers (the engine) always
	// pass absolute. We fail closed — every entry is marked Skip with
	// path-escape findings.
	entries := []aiwfyaml.Entry{entry("C-001", "schema.cue", "fixtures")}
	resolved, findings := Resolve("relative/root", entries)
	if !hasEscape(findings, "C-001", "schema") {
		t.Errorf("relative repoRoot must escape schema; got %+v", findings)
	}
	if !resolved[0].Skip {
		t.Error("relative root should mark entry Skip")
	}
}

func TestResolve_threeEntriesMixed(t *testing.T) {
	root := realRoot(t)
	writeFile(t, filepath.Join(root, "schema.cue"))
	mkdir(t, filepath.Join(root, "fixtures"))

	entries := []aiwfyaml.Entry{
		entry("C-001", "../escape", "fixtures"),
		entry("C-002", "schema.cue", "fixtures"),
		entry("C-003", "schema.cue", "../oops"),
	}
	resolved, findings := Resolve(root, entries)

	if !hasEscape(findings, "C-001", "schema") {
		t.Error("C-001 schema escape missing")
	}
	if !hasEscape(findings, "C-003", "fixtures") {
		t.Error("C-003 fixtures escape missing")
	}
	for _, f := range findings {
		if f.EntityID == "C-002" && f.Subcode == "path-escape" {
			t.Errorf("C-002 should not have path-escape: %+v", f)
		}
	}
	if !resolved[0].Skip {
		t.Error("C-001 must be Skip")
	}
	if resolved[1].Skip {
		t.Error("C-002 must NOT be Skip")
	}
	if !resolved[2].Skip {
		t.Error("C-003 must be Skip")
	}
}

func TestResolve_unresolvableRepoRoot(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does", "not", "exist")
	entries := []aiwfyaml.Entry{entry("C-001", "schema.cue", "fixtures")}
	resolved, findings := Resolve(missing, entries)
	// A non-existent root resolves lexically; entries below it are still
	// "inside" relative to that lexical form, so no path-escape is
	// expected. The downstream existence check is what reports missing.
	for _, f := range findings {
		if f.Subcode == "path-escape" {
			t.Errorf("missing root should not produce path-escape: %+v", f)
		}
	}
	if resolved[0].Skip {
		t.Error("missing root with in-tree entry must not be Skip")
	}
}

func TestResolve_repoRootSymlinkLoop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix symlinks")
	}
	tmp := realRoot(t)
	a := filepath.Join(tmp, "a")
	b := filepath.Join(tmp, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatal(err)
	}
	// repoRoot is a symlink loop; resolveRepoRoot falls back to the
	// lexical absolute form. Entries with relative paths still join
	// cleanly against it.
	entries := []aiwfyaml.Entry{entry("C-001", "schema.cue", "fixtures")}
	resolved, _ := Resolve(a, entries)
	if resolved == nil {
		t.Fatal("Resolve returned nil")
	}
}

func TestResolve_findingShape(t *testing.T) {
	root := realRoot(t)
	entries := []aiwfyaml.Entry{entry("C-042", "../bad", "fixtures")}
	_, findings := Resolve(root, entries)
	if len(findings) == 0 {
		t.Fatal("want at least one finding")
	}
	f := findings[0]
	if f.Code != "contract-config" {
		t.Errorf("Code = %q, want contract-config", f.Code)
	}
	if f.Severity != check.SeverityError {
		t.Errorf("Severity = %v, want error", f.Severity)
	}
	if f.Subcode != "path-escape" {
		t.Errorf("Subcode = %q, want path-escape", f.Subcode)
	}
	if f.EntityID != "C-042" {
		t.Errorf("EntityID = %q, want C-042", f.EntityID)
	}
	if f.Path != "aiwf.yaml" {
		t.Errorf("Path = %q, want aiwf.yaml", f.Path)
	}
}
