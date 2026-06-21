package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ritualByName returns the embedded ritual skill with the given name, or
// fails the test. Centralizes the ListRituals lookup the cases below share.
func ritualByName(t *testing.T, name string) Skill {
	t.Helper()
	got, err := ListRituals()
	if err != nil {
		t.Fatalf("ListRituals: %v", err)
	}
	for _, s := range got {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("ritual skill %q not found among embedded rituals", name)
	return Skill{}
}

// TestWfCodebaseHealth_ShipsFullRubricStructurally is the structural
// chokepoint for ADR-0019 / G-0265: the wf-codebase-health skill ships the
// complete A1–G3 code-health rubric, each principle nested under its category
// heading. This is a structural assertion over the markdown heading hierarchy,
// not a flat substring scan — a principle dropped or mis-filed under the wrong
// category fails the test, where a substring match for "A1" would not.
func TestWfCodebaseHealth_ShipsFullRubricStructurally(t *testing.T) {
	t.Parallel()
	s := ritualByName(t, "wf-codebase-health")
	if len(s.Content) == 0 {
		t.Fatal("wf-codebase-health has empty content")
	}

	// The rubric's contract: every category and the principles that belong to
	// it. Pinning the full set is deliberate — "ships the complete rubric" is
	// the claim ADR-0019 makes, so dropping a principle from the embedded
	// authoring source must break this test.
	categories := []struct {
		letter     string
		heading    string
		principles []string
	}{
		{"A", "## A. Module boundaries", []string{"A1", "A2", "A3"}},
		{"B", "## B. Contracts", []string{"B1", "B2", "B3"}},
		{"C", "## C. Data discipline", []string{"C1", "C2", "C3", "C4"}},
		{"D", "## D. Tests that pin behavior, not implementation", []string{"D1", "D2", "D3", "D4"}},
		{"E", "## E. Errors, logs, audit trail", []string{"E1", "E2", "E3", "E4"}},
		{"F", "## F. Reasoning aids", []string{"F1", "F2", "F3"}},
		{"G", "## G. Operational properties", []string{"G1", "G2", "G3"}},
	}

	// Walk the heading hierarchy: track the current category H2, and record
	// the category each principle H3 (### A1. ...) is nested under.
	catHeadingRE := regexp.MustCompile(`^## ([A-G])\. `)
	principleRE := regexp.MustCompile(`^### ([A-G]\d+)\. `)
	codeToCat := map[string]string{}
	seenHeading := map[string]bool{}
	currentCat := ""
	for line := range strings.SplitSeq(string(s.Content), "\n") {
		switch {
		case catHeadingRE.MatchString(line):
			currentCat = catHeadingRE.FindStringSubmatch(line)[1]
			seenHeading[strings.TrimRight(line, " \t")] = true
		case strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "# "):
			// Left the principle categories (e.g. the meta sections).
			currentCat = ""
		case principleRE.MatchString(line):
			code := principleRE.FindStringSubmatch(line)[1]
			codeToCat[code] = currentCat
		}
	}

	for _, cat := range categories {
		if !seenHeading[cat.heading] {
			t.Errorf("missing category heading %q", cat.heading)
		}
		for _, code := range cat.principles {
			gotCat, ok := codeToCat[code]
			if !ok {
				t.Errorf("principle %s has no `### %s.` heading in the rubric", code, code)
				continue
			}
			if gotCat != cat.letter {
				t.Errorf("principle %s nested under category %q, want %q", code, gotCat, cat.letter)
			}
		}
	}
}

// TestWfCodebaseHealth_DeclaresItselfAdvisory pins the load-bearing claim of
// ADR-0019: the skill is advisory (forces, not rules), not a mechanical gate.
// Scoped to the advisory section, not a flat substring scan, so the claim must
// live where a reader meets it.
func TestWfCodebaseHealth_DeclaresItselfAdvisory(t *testing.T) {
	t.Parallel()
	s := ritualByName(t, "wf-codebase-health")
	section := markdownSection(string(s.Content), "## This is advisory")
	if section == "" {
		t.Fatal("wf-codebase-health is missing its `## This is advisory` section")
	}
	// Normalize whitespace so the claim is matched against prose, not the
	// skill's line-wrapping.
	norm := strings.Join(strings.Fields(section), " ")
	for _, want := range []string{"forces, not", "does not enforce anything", "stays advisory by design"} {
		if !strings.Contains(norm, want) {
			t.Errorf("advisory section missing %q; section:\n%s", want, section)
		}
	}
}

// TestWfCodebaseHealth_MaterializesToClaudeSkills covers the seam: the embedded
// rubric is written into .claude/skills/wf-codebase-health/SKILL.md by
// Materialize (the aiwf init / update path), byte-for-byte.
func TestWfCodebaseHealth_MaterializesToClaudeSkills(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, SkillsDir, "wf-codebase-health", "SKILL.md"))
	if err != nil {
		t.Fatalf("wf-codebase-health not materialized: %v", err)
	}
	if !bytes.Equal(got, ritualByName(t, "wf-codebase-health").Content) {
		t.Error("materialized wf-codebase-health content differs from embedded source")
	}
}

// markdownSection returns the text from the line whose trimmed form starts with
// heading up to (excluding) the next heading at the same-or-higher level, or ""
// if the heading is absent.
func markdownSection(content, heading string) string {
	lines := strings.Split(content, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimRight(line, " \t"), heading) {
			start = i
			break
		}
	}
	if start == -1 {
		return ""
	}
	var b strings.Builder
	b.WriteString(lines[start] + "\n")
	for _, line := range lines[start+1:] {
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "# ") {
			break
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}
