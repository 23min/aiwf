package policies

import (
	"path/filepath"
	"strings"
	"testing"
)

// writeAt writes content at a slash-separated repo-relative path under
// root. The path rides in as a variable (not a filepath.Join literal),
// keeping call sites readable without tripping gocritic's filepathJoin.
func writeAt(t *testing.T, root, rel, content string) {
	t.Helper()
	mustWrite(t, filepath.Join(root, filepath.FromSlash(rel)), content)
}

// findingTableSkill returns a minimal aiwf-check skill body whose
// Findings table documents exactly the given codes (one row each).
func findingTableSkill(codes ...string) string {
	s := "# aiwf-check\n\n## Findings (errors)\n\n| Code | Meaning | Typical fix |\n|---|---|---|\n"
	for _, c := range codes {
		s += "| `" + c + "` | meaning | fix |\n"
	}
	return s
}

// TestFindingCodesDocumented_FiresOnUndocumentedBareCode is the AC-2
// firing case for a bare string-literal Code emitted with no skill entry.
func TestFindingCodesDocumented_FiresOnUndocumentedBareCode(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/check/x.go",
		"package check\n\nvar _ = Finding{Code: \"zzz-bare-undoc\"}\n")
	mustWrite(t, filepath.Join(root, skillCheckPath), findingTableSkill("some-other-code"))

	vs, err := PolicyFindingCodesDocumentedInSkill(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if !hasPolicyViolation(vs, "finding-codes-documented-in-skill") {
		t.Fatalf("expected a violation for the undocumented bare code; got %d: %+v", len(vs), vs)
	}
	if !violationMentions(vs, "zzz-bare-undoc") {
		t.Errorf("violation should name the undocumented code %q; got %+v", "zzz-bare-undoc", vs)
	}
}

// TestFindingCodesDocumented_FiresOnSamePackageDescriptorID proves the
// shared enumerator resolves `Code: CodeXxx.ID` selectors on a
// same-package codespkg.Code descriptor — the resolution the pre-M-0197
// walker lacked, which kept the branch-choreography findings invisible.
func TestFindingCodesDocumented_FiresOnSamePackageDescriptorID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/check/desc.go",
		"package check\n\nvar CodeFoo = codespkg.Code{ID: \"foo-desc-code\"}\n\nvar _ = Finding{Code: CodeFoo.ID}\n")
	mustWrite(t, filepath.Join(root, skillCheckPath), findingTableSkill("unrelated"))

	vs, err := PolicyFindingCodesDocumentedInSkill(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if !violationMentions(vs, "foo-desc-code") {
		t.Errorf("expected a violation naming the same-package descriptor code %q; got %+v", "foo-desc-code", vs)
	}
}

// TestFindingCodesDocumented_FiresOnCrossPackageDescriptorID proves the
// resolver handles `Code: check.CodeXxx.ID` — the cross-package selector
// internal/cli/check uses to emit the isolation-escape subcodes.
func TestFindingCodesDocumented_FiresOnCrossPackageDescriptorID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/check/desc.go",
		"package check\n\nvar CodeBar = codespkg.Code{ID: \"bar-desc-code\"}\n")
	writeAt(t, root, "internal/cli/check/emit.go",
		"package check\n\nvar _ = Finding{Code: check.CodeBar.ID}\n")
	mustWrite(t, filepath.Join(root, skillCheckPath), findingTableSkill("unrelated"))

	vs, err := PolicyFindingCodesDocumentedInSkill(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if !violationMentions(vs, "bar-desc-code") {
		t.Errorf("expected a violation naming the cross-package descriptor code %q; got %+v", "bar-desc-code", vs)
	}
}

// TestFindingCodesDocumented_SilentWhenDocumented proves the policy
// discriminates: a code with a skill table row produces no violation,
// while an undocumented sibling in the same fixture still fires.
func TestFindingCodesDocumented_SilentWhenDocumented(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/check/x.go",
		"package check\n\nvar _ = Finding{Code: \"alpha-code\"}\nvar _ = Finding{Code: \"omega-code\"}\n")
	mustWrite(t, filepath.Join(root, skillCheckPath), findingTableSkill("alpha-code"))

	vs, err := PolicyFindingCodesDocumentedInSkill(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if violationMentions(vs, "alpha-code") {
		t.Errorf("documented code must not fire; got %+v", vs)
	}
	if !violationMentions(vs, "omega-code") {
		t.Errorf("undocumented sibling must still fire; got %+v", vs)
	}
}

// TestFindingCodesDocumented_OptOutSuppressesSyntheticCode proves the
// rationale-annotated opt-out carves out synthetic test-fixture codes.
func TestFindingCodesDocumented_OptOutSuppressesSyntheticCode(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/check/x.go",
		"package check\n\nvar _ = Finding{Code: \"a-err\"}\nvar _ = Finding{Code: \"z-warn\"}\n")
	mustWrite(t, filepath.Join(root, skillCheckPath), findingTableSkill("unrelated"))

	vs, err := PolicyFindingCodesDocumentedInSkill(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("opt-out codes must not fire; got %+v", vs)
	}
}

// TestFindingCodesDocumented_FiresWhenSkillMissing proves the fail-loud
// contract: with no skill file present, the documented set is empty, so
// every check-layer emitted code fires (rather than silently passing).
func TestFindingCodesDocumented_FiresWhenSkillMissing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/check/x.go",
		"package check\n\nvar _ = Finding{Code: \"lonely-code\"}\n")
	// no skill file written.
	vs, err := PolicyFindingCodesDocumentedInSkill(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	if !violationMentions(vs, "lonely-code") {
		t.Errorf("missing skill must make every emitted code fire; got %+v", vs)
	}
}

// TestFindingCodesDocumented_DedupsMultipleSites proves a code emitted at
// several check-layer sites yields a single violation, not one per site.
func TestFindingCodesDocumented_DedupsMultipleSites(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAt(t, root, "internal/check/a.go",
		"package check\n\nvar _ = Finding{Code: \"twice-code\"}\n")
	writeAt(t, root, "internal/check/b.go",
		"package check\n\nvar _ = Finding{Code: \"twice-code\"}\n")
	mustWrite(t, filepath.Join(root, skillCheckPath), findingTableSkill("unrelated"))
	vs, err := PolicyFindingCodesDocumentedInSkill(root)
	if err != nil {
		t.Fatalf("policy error: %v", err)
	}
	n := 0
	for _, v := range vs {
		if strings.Contains(v.Detail, "twice-code") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("expected exactly one violation for the twice-emitted code, got %d: %+v", n, vs)
	}
}

// TestFindingCodesDocumented_ErrorsOnUnwalkableRoot covers the
// WalkGoFiles error path (a root that cannot be walked).
func TestFindingCodesDocumented_ErrorsOnUnwalkableRoot(t *testing.T) {
	t.Parallel()
	_, err := PolicyFindingCodesDocumentedInSkill(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Error("expected an error walking a nonexistent root")
	}
}

// TestPolicy_FindingCodesDocumentedInSkill is the live-tree chokepoint
// (M-0197/AC-1 evidence): every finding code the check layer emits has a
// documented entry in the aiwf-check skill.
func TestPolicy_FindingCodesDocumentedInSkill(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyFindingCodesDocumentedInSkill)
}

// violationMentions reports whether any violation's Detail contains sub.
func violationMentions(vs []Violation, sub string) bool {
	for _, v := range vs {
		if strings.Contains(v.Detail, sub) {
			return true
		}
	}
	return false
}
