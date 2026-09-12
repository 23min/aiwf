package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// banAlways and banEscapable are synthetic rules, so the engine's tests pin
// the walk rather than either shipped ban's literals. banAlways is the
// no-escape shape both real tables open with; banEscapable is the
// conditional shape both close with.
var (
	banAlways    = lineBan{pattern: regexp.MustCompile(`(?i)ALPHA`), detail: "alpha"}
	banEscapable = lineBan{pattern: regexp.MustCompile(`(?i)beta`), escape: "allowed", detail: "beta"}
)

// ruleTag names the rule in bans whose detail v carries, as `r<index>`, and
// fails when no rule in the table carries it.
//
// A ban's rules are told apart only by the detail they report, so a route test
// rendering the location alone passes just as happily when the two rules'
// details are swapped — and an operator then reads the wrong reason for the
// line they are looking at. Tagging by index makes each route say which rule
// caught it, derived from the production table rather than restated.
func ruleTag(t *testing.T, bans []lineBan, v Violation) string {
	t.Helper()
	for i, b := range bans {
		if b.detail == v.Detail {
			return "r" + strconv.Itoa(i)
		}
	}
	t.Errorf("%s:%d carries a detail no rule in the table reports: %q", v.File, v.Line, v.Detail)
	return "r?"
}

// scanReports runs the ban over a synthetic tree and renders each violation as
// `file:line:detail`, so a test asserts where it fired and which rule caught
// it — not merely that something did. The policy id is a placeholder; the
// tests that care about the id live beside the two real bans.
func scanReports(t *testing.T, bans []lineBan, files map[string]string) []string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		mustWrite(t, filepath.Join(root, rel), content)
	}
	vs, err := runShippedBan(root, bans, func() Violation {
		return Violation{Policy: "synthetic-ban"}
	})
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}
	got := make([]string, 0, len(vs))
	for _, v := range vs {
		got = append(got, v.File+":"+strconv.Itoa(v.Line)+":"+v.Detail)
	}
	return got
}

// TestRunShippedBan_ScopeIsEveryEmbeddedTree pins what the walk reads:
// every tree under internal/skills/ whose name begins "embedded", and nothing
// else there. A sibling package is aiwf's own source rather than a surface a
// consumer receives, so a mention in it is not a shipped instruction — and the
// scope is every embedded tree rather than one, which is the property the two
// bans now share instead of disagreeing about.
func TestRunShippedBan_ScopeIsEveryEmbeddedTree(t *testing.T) {
	t.Parallel()
	// One row per file, carrying whether the walk should read it, so the
	// expectation is derived from the same table the fixture is built from —
	// adding a file cannot leave a stale want list behind. Rows sit in the
	// walk's own order: WalkDir reads each directory lexically, and
	// "embedded" sorts ahead of its hyphenated siblings.
	rows := []struct {
		rel  string
		read bool
	}{
		// Every shipped tree is read, whatever the file's extension.
		{"internal/skills/embedded/aiwf-show/SKILL.md", true},
		{"internal/skills/embedded-guidance/aiwf-guidance.md", true},
		{"internal/skills/embedded-hooks/pre-push.sh", true},
		{"internal/skills/embedded-rituals/plugins/x/SKILL.md", true},
		{"internal/skills/embedded-statusline/status.sh", true},
		// aiwf's own source and testdata beside those trees are not surfaces a
		// consumer receives, a file sitting directly in internal/skills/ is
		// not in a tree at all, and an "embedded" directory under another
		// package is not one of aiwf's.
		{"internal/skills/embed.go", false},
		{"internal/skills/materialize.go", false},
		{"internal/skills/materialize_test.go", false},
		{"internal/skills/testdata/golden/spec.md", false},
		{"internal/policies/embedded/x.md", false},
	}
	files := make(map[string]string, len(rows))
	var want []string
	for _, r := range rows {
		files[r.rel] = "alpha\n"
		if r.read {
			want = append(want, r.rel+":1:"+banAlways.detail)
		}
	}
	if got := scanReports(t, []lineBan{banAlways}, files); !slices.Equal(got, want) {
		t.Errorf("scope mismatch:\n got %v\nwant %v", got, want)
	}
}

// TestRunShippedBan_FirstMatchingRuleDescribesTheLine pins that order
// decides how a line is described. A line matching both rules is reported once,
// as the earlier one — which is what lets a ban name the outright shape
// separately from the weaker phrase that also catches it.
func TestRunShippedBan_FirstMatchingRuleDescribesTheLine(t *testing.T) {
	t.Parallel()
	rel := "internal/skills/embedded/aiwf-show/SKILL.md"
	got := scanReports(t, []lineBan{banAlways, banEscapable}, map[string]string{
		rel: "alpha and beta on one line\n",
	})
	want := []string{rel + ":1:alpha"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRunShippedBan_EscapeClearsOnlyItsOwnRule pins the escape's reach.
// It clears the rule carrying it and no other, so an escaped line still
// reports through a rule that has none — the shape that stops a ban being
// switched off by a phrase that happens to appear on the line.
func TestRunShippedBan_EscapeClearsOnlyItsOwnRule(t *testing.T) {
	t.Parallel()
	rel := "internal/skills/embedded-rituals/plugins/x/SKILL.md"
	for _, tc := range []struct {
		name string
		bans []lineBan
		line string
		want []string
	}{
		{
			name: "escape clears the rule carrying it",
			bans: []lineBan{banAlways, banEscapable},
			line: "beta is allowed here",
			want: nil,
		},
		{
			name: "a line cased differently from the escape still escapes",
			bans: []lineBan{banAlways, banEscapable},
			line: "beta is ALLOWED here",
			want: nil,
		},
		{
			// An escape written in mixed case must behave as one written in
			// lower case. The line is lowered before the comparison, so an
			// escape that is not lowered too can never match, and the rule
			// carrying it silently loses its exemption.
			name: "an escape written in mixed case still escapes",
			bans: []lineBan{{pattern: regexp.MustCompile(`(?i)beta`), escape: "AlLoWeD", detail: "beta"}},
			line: "beta is allowed here",
			want: nil,
		},
		{
			// The escaped rule comes FIRST here, so the search has to carry
			// on past it to reach banAlways. With banAlways first this row
			// would pass without the fall-through ever running.
			name: "an escaped rule does not stop the rules after it",
			bans: []lineBan{banEscapable, banAlways},
			line: "beta allowed alpha",
			want: []string{rel + ":1:alpha"},
		},
		{
			name: "without the escape the rule reports",
			bans: []lineBan{banAlways, banEscapable},
			line: "beta stands alone",
			want: []string{rel + ":1:beta"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := scanReports(t, tc.bans, map[string]string{rel: tc.line + "\n"})
			if !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestRunShippedBan_ReportsEveryMatchingLineByNumber pins that the scan
// reports per line rather than per file, numbering from one. A ban that
// reported a file once would name the first reintroduction and hide the rest.
func TestRunShippedBan_ReportsEveryMatchingLineByNumber(t *testing.T) {
	t.Parallel()
	rel := "internal/skills/embedded-guidance/aiwf-guidance.md"
	got := scanReports(t, []lineBan{banAlways}, map[string]string{
		rel: "clean\nalpha\nclean\nalpha\n",
	})
	want := []string{rel + ":2:alpha", rel + ":4:alpha"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRunShippedBan_StampsTheIDAndTheRuleThatFired pins the whole violation a
// caller gets back: the policy id it built, and the file, line and detail of
// the rule that caught the line. Every field is asserted together because a
// stamp that carried the location but dropped the detail would hand an
// operator a violation naming no reason, and a location-only assertion cannot
// tell the two apart.
func TestRunShippedBan_StampsTheIDAndTheRuleThatFired(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	rel := "internal/skills/embedded/aiwf-show/SKILL.md"
	mustWrite(t, filepath.Join(root, rel), "clean\nalpha here\n")
	got, err := runShippedBan(root, []lineBan{banAlways}, func() Violation {
		return Violation{Policy: "synthetic-ban"}
	})
	if err != nil {
		t.Fatalf("runShippedBan: %v", err)
	}
	want := []Violation{{Policy: "synthetic-ban", File: rel, Line: 2, Detail: banAlways.detail}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("stamped violation mismatch (-want +got):\n%s", diff)
	}
}

// TestRunShippedBan_CleanTreeStampsNothing pins that a tree no rule catches
// yields no violation and no error — the everyday case.
func TestRunShippedBan_CleanTreeStampsNothing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "internal", "skills", "embedded", "aiwf-show", "SKILL.md"), "nothing banned here\n")
	got, err := runShippedBan(root, []lineBan{banAlways}, func() Violation {
		return Violation{Policy: "synthetic-ban"}
	})
	if err != nil {
		t.Fatalf("runShippedBan: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("a clean tree stamped %v", got)
	}
}

// TestRunShippedBan_BuildsTheViolationOncePerFiring pins that the constructor
// runs per hit and not at all on a clean tree.
//
// This is what keeps PolicyFiringFixturePresence honest for every policy built
// on runShippedBan: that gate calls a policy unproven by finding its
// `Policy: "<id>"` line uncovered, so the line has to run on a firing and only
// on a firing. Hoisting the call above the loop changes no returned violation
// and no count — every other assertion here stays green — while lighting the
// line on a clean tree and making the gate pass vacuously for both policies.
// Counting the calls is what makes that visible.
func TestRunShippedBan_BuildsTheViolationOncePerFiring(t *testing.T) {
	t.Parallel()
	// Called through an explicitly typed variable, so the constructor
	// parameter is part of what this test pins. The counting below catches the
	// call being hoisted above the loop; it cannot catch the parameter being
	// flattened to a plain Violation, because whoever flattens it adapts these
	// call sites in the same edit and the counting goes with them. The
	// annotation makes that edit a compile error, so between them the two
	// leave no quiet route back to a per-call stamp.
	//nolint:staticcheck // QF1011: the explicit type is the assertion — inferring it from runShippedBan would pin nothing.
	var run func(string, []lineBan, func() Violation) ([]Violation, error) = runShippedBan

	for _, tc := range []struct {
		name string
		body string
		want int
	}{
		{name: "a clean tree never builds one", body: "nothing banned here\n", want: 0},
		{name: "one per caught line", body: "alpha\nclean\nalpha\nalpha\n", want: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			mustWrite(t, filepath.Join(root, "internal", "skills", "embedded", "aiwf-show", "SKILL.md"), tc.body)
			calls := 0
			got, err := run(root, []lineBan{banAlways}, func() Violation {
				calls++
				return Violation{Policy: "synthetic-ban"}
			})
			if err != nil {
				t.Fatalf("runShippedBan: %v", err)
			}
			if calls != tc.want {
				t.Errorf("constructor ran %d times, want %d", calls, tc.want)
			}
			if len(got) != tc.want {
				t.Errorf("stamped %d violations, want %d", len(got), tc.want)
			}
		})
	}
}

// TestRunShippedBan_MissingSkillsTreeIsAnError pins that a root with no
// internal/skills/ reports the walk failure rather than a clean verdict. A ban
// that passes when it read nothing is the failure mode these policies exist to
// prevent, one level up.
func TestRunShippedBan_MissingSkillsTreeIsAnError(t *testing.T) {
	t.Parallel()
	if _, err := runShippedBan(t.TempDir(), []lineBan{banAlways}, func() Violation {
		return Violation{Policy: "synthetic-ban"}
	}); err == nil {
		t.Error("a root carrying no internal/skills/ tree returned no error")
	}
}

// TestRunShippedBan_UnreadableFileIsAnError pins that a file the walk cannot
// read fails the ban rather than passing it. A ban that reports clean on the
// bytes it never saw is worse than no ban, because the clean verdict is what
// stops the next reader looking.
func TestRunShippedBan_UnreadableFileIsAnError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	rel := filepath.Join("internal", "skills", "embedded", "aiwf-show", "SKILL.md")
	mustWrite(t, filepath.Join(root, rel), "# S\n")
	// A dangling symlink reads as a non-directory entry the walk yields and
	// every uid fails to open, so the arm is reachable without file modes.
	if err := os.Remove(filepath.Join(root, rel)); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "no-such-target"), filepath.Join(root, rel)); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := runShippedBan(root, []lineBan{banAlways}, func() Violation {
		return Violation{Policy: "synthetic-ban"}
	}); err == nil {
		t.Error("an unreadable shipped surface returned no error")
	}
}
