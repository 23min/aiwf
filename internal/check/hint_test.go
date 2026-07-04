package check

import (
	"regexp"
	"strings"
	"testing"
)

// commandInSpan matches an `aiwf ...`/`git ...` invocation inside a
// single inline code span (G-0199). The whitespace after the verb name
// is deliberate: it accepts `aiwf check` / `git rm` but rejects a bare
// config reference like `aiwf.yaml` (a `.`, not a space, follows) and a
// finding-code lookalike like `git-config-core-worktree-misset`.
var commandInSpan = regexp.MustCompile(`\b(?:aiwf|git)\s`)

// hintNamesCommand reports whether a hint string names at least one
// remediation command inside a backtick-delimited code span. Splitting
// on the backtick and inspecting only the odd-indexed segments (the true
// span interiors) means an `aiwf`/`git` mention loose in the prose
// *between* two spans does not count — only a command genuinely inside a
// span does.
func hintNamesCommand(hint string) bool {
	parts := strings.Split(hint, "`")
	for i := 1; i < len(parts); i += 2 {
		if commandInSpan.MatchString(parts[i]) {
			return true
		}
	}
	return false
}

// TestHintTable_EveryHintNamesACommand is the finding-hints-name-command
// chokepoint (G-0199): every hint must name the exact remediation command
// so an LLM or operator reading a finding has the fix in hand and never
// falls through to a source grep. A new finding that ships a command-free
// hint reddens this test. It ranges the live hintTable rather than
// re-parsing source, so it pins the data the renderer actually serves.
func TestHintTable_EveryHintNamesACommand(t *testing.T) {
	t.Parallel()
	for code, hint := range hintTable {
		if !hintNamesCommand(hint) {
			t.Errorf("hint for %q names no backtick-delimited `aiwf ...`/`git ...` command: %q", code, hint)
		}
	}
}

// TestHintNamesCommand_Discriminates is the vacuity guard for the
// chokepoint above: it pins that the detector actually distinguishes a
// named command from prose that merely mentions a field or config file,
// so TestHintTable_EveryHintNamesACommand cannot pass tautologically.
func TestHintNamesCommand_Discriminates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		hint string
		want bool
	}{
		{"aiwf command", "then re-run `aiwf check` to confirm", true},
		{"git command", "remove it with `git rm <path>`", true},
		{"aiwf.yaml config reference is not a command", "fix the `schema:` path in `aiwf.yaml`", false},
		{"bare flag span is not a command", "override with `--force --reason`", false},
		{"unbacked prose names nothing", "set a non-empty title in the frontmatter", false},
		{"finding-code lookalike is not a command", "see `git-config-core-worktree-misset`", false},
		{"aiwf loose in prose between two spans is not a command", "set `x` then aiwf runs `y`", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := hintNamesCommand(tc.hint); got != tc.want {
				t.Errorf("hintNamesCommand(%q) = %v, want %v", tc.hint, got, tc.want)
			}
		})
	}
}

// TestHint_AreaRequired pins M-0178/AC-6 (hint half): the area-required
// finding carries a remediation hint pointing operators at `aiwf set-area`
// (the M-0183 tag verb). Removing the hint entry reddens this test (and
// the PolicyFindingCodesHaveHints chokepoint).
func TestHint_AreaRequired(t *testing.T) {
	t.Parallel()
	h := HintFor(CodeAreaRequired, "")
	if h == "" {
		t.Fatal("expected a hint for area-required, got empty")
	}
	if !strings.Contains(h, "set-area") {
		t.Errorf("hint %q should point at `aiwf set-area`", h)
	}
}
