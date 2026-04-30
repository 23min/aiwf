package check_test

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// TestFixture_Clean asserts that the synthetic clean tree produces no
// findings of any severity. If this test breaks, either a check has a
// false positive or the fixture has drifted.
func TestFixture_Clean(t *testing.T) {
	tr, loadErrs, err := tree.Load(context.Background(), "testdata/clean")
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if len(loadErrs) != 0 {
		t.Fatalf("loadErrs = %+v", loadErrs)
	}
	got := check.Run(tr, loadErrs)
	if len(got) != 0 {
		t.Errorf("expected zero findings, got:\n%+v", got)
	}
}

// TestFixture_Messy asserts that every finding code documented in
// poc-plan.md Session 1 is exercised by the messy tree. The test does
// not assert exact counts — multiple checks fire across many entities,
// and counts shift when the fixture is extended — only that each
// expected code appears at least once.
func TestFixture_Messy(t *testing.T) {
	tr, loadErrs, err := tree.Load(context.Background(), "testdata/messy")
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	got := check.Run(tr, loadErrs)
	if len(got) == 0 {
		t.Fatal("expected findings, got none")
	}

	seen := make(map[string]bool)
	for _, f := range got {
		key := f.Code
		if f.Subcode != "" {
			key = f.Code + "/" + f.Subcode
		}
		seen[key] = true
	}

	expected := []string{
		"ids-unique",
		"frontmatter-shape",
		"status-valid",
		"refs-resolve/unresolved",
		"refs-resolve/wrong-kind",
		"no-cycles/depends_on",
		"no-cycles/supersedes",
		"titles-nonempty",
		"adr-supersession-mutual",
		"gap-resolved-has-resolver",
	}
	var missing []string
	for _, code := range expected {
		if !seen[code] {
			missing = append(missing, code)
		}
	}
	if len(missing) > 0 {
		seenList := make([]string, 0, len(seen))
		for k := range seen {
			seenList = append(seenList, k)
		}
		sort.Strings(seenList)
		t.Errorf("missing expected finding codes: %v\nfindings seen: %v", missing, seenList)
	}

	// All errors should sort before all warnings (Run.sortFindings).
	for i := 1; i < len(got); i++ {
		if got[i-1].Severity == check.SeverityWarning && got[i].Severity == check.SeverityError {
			t.Errorf("findings not sorted: warning at %d precedes error at %d", i-1, i)
		}
	}

	// Also confirm HasErrors agrees with our expectation.
	if !check.HasErrors(got) {
		t.Error("HasErrors = false on the messy fixture")
	}

	_ = cmp.Diff // keep import for future granular asserts
}
