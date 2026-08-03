package policies

// M-0289: README's sample `aiwf check` output is transcribed, not paraphrased.
//
// The block is fenced as `text` and introduced as what the tool prints, so a
// reader greps their real output for it and a contributor "fixes" the code to
// match. Text in it the binary cannot produce is therefore a defect the block
// itself gives no signal of.
//
// This pins the part with an exact source: every `— hint:` must be the hint the
// kernel carries for that code. Message text, line numbers, the severity word,
// the warning-summary line (which has no `path:line:` prefix and so matches
// nothing here) and the footer counts rest on transcription alone.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// sampleFindingLine captures the code and hint from one rendered finding line.
var sampleFindingLine = regexp.MustCompile(`(?m)^\S+:\d+: (?:error|warning) ([a-z-]+(?:/[a-z-]+)?): .* — hint: (.+)$`)

func TestM0289_ReadmeSampleHintsMatchTheKernel(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("reading README: %v", err)
	}
	section := extractMarkdownSection(string(raw), 3, "Sample of `aiwf check` output")
	if strings.TrimSpace(section) == "" {
		t.Fatal("README has no sample-output section — this pin would assert nothing")
	}

	matches := sampleFindingLine.FindAllStringSubmatch(section, -1)
	if len(matches) == 0 {
		t.Fatal("the sample block shows no per-line findings — either the block " +
			"changed shape or the pattern stopped matching it; either way this " +
			"test is no longer checking what it claims to")
	}
	for _, m := range matches {
		code, shown := m[1], strings.TrimSpace(m[2])
		want := check.HintFor(code, "")
		if want == "" {
			t.Errorf("sample cites finding code %q, which carries no hint in the kernel", code)
			continue
		}
		if shown != want {
			t.Errorf("sample hint for %q is not what the kernel emits.\n shown: %s\n  real: %s",
				code, shown, want)
		}
	}
}
