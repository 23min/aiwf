package contractcheck

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/aiwfyaml"
	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

func TestRun_NilContractsReturnsNil(t *testing.T) {
	got := Run(&tree.Tree{}, nil, "/tmp")
	if got != nil {
		t.Errorf("expected nil; got %+v", got)
	}
}

func TestRun_MissingEntity(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "schema.cue"), "")
	if err := os.MkdirAll(filepath.Join(repo, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}

	tr := &tree.Tree{Root: repo, Entities: []*entity.Entity{}}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue", Args: []string{}}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/missing-entity") {
		t.Errorf("missing-entity finding not produced; got %v", codes)
	}
}

func TestRun_MissingSchemaPath(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "cue", Schema: "missing.cue", Fixtures: "fixtures",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/missing-schema") {
		t.Errorf("missing-schema not produced; got %v", codes)
	}
}

func TestRun_MissingFixturesPath(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "schema.cue"), "")
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/missing-fixtures") {
		t.Errorf("missing-fixtures not produced; got %v", codes)
	}
}

func TestRun_FixturesIsAFile_NotDirectory(t *testing.T) {
	// fixtures: pointing at a regular file rather than a directory
	// is the same shape of error as missing.
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "schema.cue"), "")
	mustWriteFile(t, filepath.Join(repo, "fixtures"), "this is a file, not a dir")
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/missing-fixtures") {
		t.Errorf("missing-fixtures not produced for file-instead-of-dir; got %v", codes)
	}
}

func TestRun_NoBindingForActiveEntity(t *testing.T) {
	repo := t.TempDir()
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
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
	// retired/rejected contract with no binding: silent (no finding).
	repo := t.TempDir()
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-001", Kind: entity.KindContract, Title: "Old", Status: "retired", Path: "work/contracts/C-001-old/contract.md"},
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
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "schema.cue"), "")
	if err := os.MkdirAll(filepath.Join(repo, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-001", Kind: entity.KindContract, Title: "Foo", Status: "accepted", Path: "work/contracts/C-001-foo/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
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
	repo := t.TempDir()
	tr := &tree.Tree{Root: repo, Entities: nil}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "cue", Schema: "ghost.cue", Fixtures: "ghost-dir",
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
	repo := t.TempDir()
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-001", Kind: entity.KindContract, Title: "Old", Status: "retired", Path: "work/contracts/C-001-old/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{{
			ID: "C-001", Validator: "cue", Schema: "ghost.cue", Fixtures: "ghost",
		}},
	}
	got := Run(tr, contracts, repo)
	codes := codesAndSubcodes(got)
	if !contains(codes, "contract-config/missing-schema") {
		t.Errorf("retired contract with broken binding should still report missing-schema; got %v", codes)
	}
}

// TestRun_MultipleBindings_FindingsCarryEntityID: when several
// bindings have different problems, each finding must name the
// correct entity id so the user knows which row to fix.
func TestRun_MultipleBindings_FindingsCarryEntityID(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, filepath.Join(repo, "good.cue"), "")
	if err := os.MkdirAll(filepath.Join(repo, "good-fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	tr := &tree.Tree{
		Root: repo,
		Entities: []*entity.Entity{
			{ID: "C-001", Kind: entity.KindContract, Title: "Good", Status: "accepted", Path: "work/contracts/C-001-good/contract.md"},
			{ID: "C-002", Kind: entity.KindContract, Title: "Bad", Status: "accepted", Path: "work/contracts/C-002-bad/contract.md"},
		},
	}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{
			{ID: "C-001", Validator: "cue", Schema: "good.cue", Fixtures: "good-fixtures"},
			{ID: "C-002", Validator: "cue", Schema: "ghost.cue", Fixtures: "good-fixtures"},
		},
	}
	got := Run(tr, contracts, repo)
	if len(got) != 1 {
		t.Fatalf("expected one finding (C-002 missing-schema); got %d: %+v", len(got), got)
	}
	if got[0].EntityID != "C-002" {
		t.Errorf("entity id = %q, want C-002", got[0].EntityID)
	}
}
