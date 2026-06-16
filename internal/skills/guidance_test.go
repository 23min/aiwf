package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/version"
)

// guidanceRules are the full-set guidance rules ADR-0018's inclusion
// principle admits to the consumer fragment (M-0163/AC-1). Each phrase
// is distinctive enough that substring presence in the rendered
// fragment is the right "contains every rule" assertion; drift in the
// embedded content fails the build.
var guidanceRules = []string{
	"Each mutating action is its own approval gate",
	"Never suggest the human pause",
	"run `aiwf reallocate`, not `git mv`",
	"Promote an AC to met only with mechanical evidence",
	"Decide one thing at a time",
	"Fix closely-related issues in place",
	"Never write a fake id-shaped token in committed prose",
}

func TestRenderGuidance_ContainsAllRules(t *testing.T) {
	t.Parallel()
	got := string(RenderGuidance("v0.0.0-test"))
	for _, rule := range guidanceRules {
		if !strings.Contains(got, rule) {
			t.Errorf("AC-1: rendered guidance is missing rule %q", rule)
		}
	}
}

// TestRenderGuidance_SubstitutesVersion pins the version marker
// behavior: the given version appears and the sentinel is fully
// replaced (M-0163/AC-2).
func TestRenderGuidance_SubstitutesVersion(t *testing.T) {
	t.Parallel()
	out := string(RenderGuidance("v9.9.9"))
	if !strings.Contains(out, "aiwf-version: v9.9.9") {
		t.Errorf("AC-2: rendered guidance missing version marker 'aiwf-version: v9.9.9'")
	}
	if strings.Contains(out, guidanceVersionSentinel) {
		t.Errorf("AC-2: version sentinel %q left unsubstituted", guidanceVersionSentinel)
	}
}

// TestMaterializeGuidance_DeclaresBinaryVersion pins the seam: the
// materialized file declares the running binary's version, not a
// hardcoded or sentinel value (M-0163/AC-2).
func TestMaterializeGuidance_DeclaresBinaryVersion(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := MaterializeGuidance(root); err != nil {
		t.Fatalf("MaterializeGuidance: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".claude", "aiwf-guidance.md"))
	if err != nil {
		t.Fatalf("reading materialized guidance: %v", err)
	}
	want := "aiwf-version: " + version.Current().Version
	if !strings.Contains(string(data), want) {
		t.Errorf("AC-2: materialized file missing version marker %q", want)
	}
	if strings.Contains(string(data), guidanceVersionSentinel) {
		t.Errorf("AC-2: materialized file left the version sentinel unsubstituted")
	}
}

// TestGuidance_WithinLineBudget is the per-turn line-budget guard: the
// fragment must stay terse enough to re-anchor every turn (M-0163/AC-4).
func TestGuidance_WithinLineBudget(t *testing.T) {
	t.Parallel()
	const budget = 50
	lines := bytes.Count(GuidanceBytes(), []byte("\n"))
	if lines > budget {
		t.Errorf("AC-4: guidance fragment is %d lines, over the %d-line per-turn budget", lines, budget)
	}
}

// TestMaterializeGuidance_WritesFile covers the success path directly
// (the initrepo seam test exercises it cross-package; this pins it in
// the skills layer too) (M-0163/AC-3).
func TestMaterializeGuidance_WritesFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := MaterializeGuidance(root); err != nil {
		t.Fatalf("MaterializeGuidance: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".claude", "aiwf-guidance.md"))
	if err != nil {
		t.Fatalf("guidance file not written: %v", err)
	}
	if len(data) == 0 {
		t.Error("materialized guidance file is empty")
	}
}

// TestMaterializeGuidance_MkdirError covers the MkdirAll error branch:
// when `.claude` already exists as a regular file, the directory create
// must fail and the error propagate (M-0163/AC-3 branch coverage).
func TestMaterializeGuidance_MkdirError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".claude"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seeding .claude as a file: %v", err)
	}
	if err := MaterializeGuidance(root); err == nil {
		t.Error("expected MaterializeGuidance to fail when .claude is a file, got nil")
	}
}

// TestMaterializeGuidance_WriteError covers the AtomicWriteFile error
// branch: writing into a read-only `.claude` directory must fail and the
// error propagate (M-0163/AC-3 branch coverage). Relies on non-root
// perm enforcement; the repo's test environment runs as a non-root user.
func TestMaterializeGuidance_WriteError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	claude := filepath.Join(root, ".claude")
	if err := os.Mkdir(claude, 0o555); err != nil {
		t.Fatalf("creating read-only .claude: %v", err)
	}
	if err := MaterializeGuidance(root); err == nil {
		t.Error("expected MaterializeGuidance to fail writing into a read-only .claude, got nil")
	}
}
