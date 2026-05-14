package contractcheck

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

func TestRun_NilContractsReturnsNil(t *testing.T) {
	t.Parallel()
	got := Run(&tree.Tree{}, nil, "/tmp")
	if got != nil {
		t.Errorf("expected nil; got %+v", got)
	}
}

func TestRun_MissingEntity(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "schema.cue"), "")
	if err := os.MkdirAll(filepath.Join(repo, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}

	tr := &tree.Tree{Root: repo, Entities: []*entity.Entity{}}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue", Args: []string{}}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-0001", Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/missing-entity") {
		t.Errorf("missing-entity finding not produced; got %v", codes)
	}
}

func TestRun_MissingSchemaPath(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-0001", Validator: "cue", Schema: "missing.cue", Fixtures: "fixtures",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/missing-schema") {
		t.Errorf("missing-schema not produced; got %v", codes)
	}
}

func TestRun_MissingFixturesPath(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "schema.cue"), "")
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-0001", Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/missing-fixtures") {
		t.Errorf("missing-fixtures not produced; got %v", codes)
	}
}

func TestRun_FixturesIsAFile_NotDirectory(t *testing.T) {
	t.Parallel()
	// fixtures: pointing at a regular file rather than a directory
	// is the same shape of error as missing.
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "schema.cue"), "")
	mustWriteFile(t, filepath.Join(repo, "fixtures"), "this is a file, not a dir")
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-0001", Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/missing-fixtures") {
		t.Errorf("missing-fixtures not produced for file-instead-of-dir; got %v", codes)
	}
}

func TestRun_NoBindingForActiveEntity(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{},
		Entries:    nil,
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/no-binding") {
		t.Errorf("no-binding warning not produced; got %v", codes)
	}
	// And it should be a warning, not an error.
	for _, f := range got {
		if f.Code == "contract-config" && f.Subcode == "no-binding" && f.Severity != check.SeverityWarning {
			t.Errorf("no-binding severity = %v, want warning", f.Severity)
		}
	}
}

func TestRun_TerminalEntityNoBinding_NoFinding(t *testing.T) {
	t.Parallel()
	// retired/rejected contract with no binding: silent (no finding).
	repo := t.TempDir()
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "Old", Status: "retired", Path: "work/contracts/C-001-old/contract.md"},
		},
	}
	got := Run(tr, &aiwfyaml.Contracts{}, repo)
	for _, f := range got {
		if f.Subcode == "no-binding" {
			t.Errorf("retired contract should not produce no-binding warning; got %+v", f)
		}
	}
}

func TestRun_CleanRepoNoFindings(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "schema.cue"), "")
	if err := os.MkdirAll(filepath.Join(repo, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-0001", Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
		}},
	}
	got := Run(tr, contracts, repo)
	if len(got) != 0 {
		t.Errorf("expected no findings on clean repo; got %+v", got)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func codesAndSubcodes(fs []check.Finding) []string {
	out := make([]string, len(fs))
	for i := range fs {
		k := fs[i].Code
		if fs[i].Subcode != "" {
			k = k + "/" + fs[i].Subcode
		}
		out[i] = k
	}
	return out
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// --- Edge case coverage (added during the I1 hardening pass) ---

// TestRun_AllThreeProblemsAtOnce: a single binding can be wrong in
// every dimension simultaneously — missing entity, missing schema
// path, missing fixtures path. All three findings must surface.
func TestRun_AllThreeProblemsAtOnce(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	tr := &tree.Tree{Root: repo, Entities: nil}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-0001", Validator: "cue", Schema: "ghost.cue", Fixtures: "ghost-dir",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	for _, want := range []string{
		"contract-config/missing-entity",
		"contract-config/missing-schema",
		"contract-config/missing-fixtures",
	} {
		if !contains(codes, want) {
			t.Errorf("missing finding %q; got %v", want, codes)
		}
	}
}

// TestRun_TerminalEntityBoundFixtureProblems_StillReportsConfig:
// per the no-binding skip, terminal-state contracts that have no
// binding don't get the "no-binding" warning. But when they DO have
// a binding, missing-schema/fixtures findings should still fire,
// because contract-config validates the binding's structural
// correctness regardless of entity status.
func TestRun_TerminalEntityWithBindingStillReportsConfig(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "Old", Status: "retired", Path: "work/contracts/C-001-old/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-0001", Validator: "cue", Schema: "ghost.cue", Fixtures: "ghost",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/missing-schema") {
		t.Errorf("retired contract with broken binding should still report missing-schema; got %v", codes)
	}
}

// TestRun_DotDotEscape_Schema: a `..` in the schema path that
// resolves outside the repo root must produce one path-escape
// finding and suppress the missing-schema finding (we don't
// double-report on entries we won't trust anyway).
func TestRun_DotDotEscape_Schema(t *testing.T) {
	t.Parallel()
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-0001", Validator: "cue", Schema: "../../etc/passwd", Fixtures: "fixtures",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/path-escape") {
		t.Errorf("want path-escape; got %v", codes)
	}
	if contains(codes, "contract-config/missing-schema") {
		t.Errorf("must not double-report missing-schema for an escaped path; got %v", codes)
	}
	for _, f := range got {
		if f.Subcode == "path-escape" && !strings.Contains(f.Message, `"../../etc/passwd"`) {
			t.Errorf("message must quote configured path verbatim; got %q", f.Message)
		}
	}
}

// TestRun_AbsoluteEscape_Fixtures: an absolute path in the fixtures
// field that points outside the repo must escape (filepath.Join
// silently rebases absolute arguments, so this is the
// regression-prone path).
func TestRun_AbsoluteEscape_Fixtures(t *testing.T) {
	t.Parallel()
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(repo, "schema.cue"), "")
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-0001", Validator: "cue", Schema: "schema.cue", Fixtures: outside,
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/path-escape") {
		t.Errorf("want path-escape; got %v", codes)
	}
	if contains(codes, "contract-config/missing-fixtures") {
		t.Errorf("must not double-report missing-fixtures for escaped path; got %v", codes)
	}
}

// TestRun_SymlinkOutside_Fixtures: a symlink inside the fixtures
// path that resolves outside the repo must produce path-escape and
// must suppress missing-fixtures (the escaped path is untrustworthy).
func TestRun_SymlinkOutside_Fixtures(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("posix symlinks")
	}
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(repo, "fixtures")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(repo, "schema.cue"), "")
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-0001", Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/path-escape") {
		t.Errorf("want path-escape for out-of-repo symlink; got %v", codes)
	}
}

// TestRun_MultipleBindings_FindingsCarryEntityID: when several
// bindings have different problems, each finding must name the
// correct entity id so the user knows which row to fix.
func TestRun_MultipleBindings_FindingsCarryEntityID(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "good.cue"), "")
	if err := os.MkdirAll(filepath.Join(repo, "good-fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "Good", Status: "accepted", Path: "work/contracts/C-001-good/contract.md"},
			{ID: "C-0002", Kind: entity.KindContract, Title: "Bad", Status: "accepted", Path: "work/contracts/C-002-bad/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{
			{ID: "C-0001", Validator: "cue", Schema: "good.cue", Fixtures: "good-fixtures"},
			{ID: "C-0002", Validator: "cue", Schema: "ghost.cue", Fixtures: "good-fixtures"},
		},
	}
	got := Run(tr, contracts, repo)
	if len(got) != 1 {
		t.Fatalf("expected one finding (C-002 missing-schema); got %d: %+v", len(got), got)
	}
	if got[0].EntityID != "C-0002" {
		t.Errorf("entity id = %q, want C-002", got[0].EntityID)
	}
}
