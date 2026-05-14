package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestPolicy_DocTreeNarrowIDsCanonicalized is the M-083 AC-3
// chokepoint: post-sweep, narrow-id mentions in non-archive doc-tree
// markdown files appear only inside code fences, inline backtick
// spans, or the explicit allowlist of historical / foreign / worked-
// example mentions.
//
// Per CLAUDE.md "substring assertions are not structural assertions",
// the grep is per-line and tracks markdown code-fence and inline-span
// context so a stray narrow id inside a fenced bash example or
// inside an inline-code span does not false-positive.
//
// Allowlist file paths each carry a one-line rationale comment.
// Allowlist categories:
//   - **Foreign-project surveys** under docs/explorations/surveys/ —
//     refer to other projects' (FlowTime, Liminara) entity ids;
//     not aiwf entities and not subject to ADR-0008.
//   - **Hypothetical worked examples** in design proposals —
//     entities like E-12, M-042 in 07-tdd-architecture-proposal.md
//     do not exist and are illustrative.
//   - **Illustrative-only docs** — mining/policy-design exploratory
//     docs whose narrow-id mentions are illustrative shorthand
//     (`M-9`, `M-12`) rather than references to real entities.
//   - **Historical archive** — docs/pocv3/archive/ (excluded by
//     directory walk; not in the allowlist surface).
//   - **CHANGELOG.md** — release notes describing the historical
//     state at the time of release; entities born narrow stay
//     narrow in the changelog (per the spec's "v0.1.0 introduced
//     E-NN" example).
//
// Per CLAUDE.md "framework correctness must not depend on the LLM's
// behavior," AC-3's discipline lives in this test, not in reviewer
// recall.
func TestPolicy_DocTreeNarrowIDsCanonicalized(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	// Each entry is a repo-relative path that legitimately holds
	// narrow-id mentions outside code fences and backtick spans.
	// Each rationale states why canonicalization would erase signal.
	allowlist := map[string]string{
		// Foreign-project surveys — narrow ids belong to FlowTime /
		// Liminara, not aiwf, and are not subject to ADR-0008.
		"docs/explorations/surveys/flowtime/00-overview.md":                              "foreign-project (FlowTime) entity ids",
		"docs/explorations/surveys/flowtime/01-policies-general.md":                      "foreign-project (FlowTime) policy ids G-1..G-37",
		"docs/explorations/surveys/flowtime/02-policies-project-specific.md":             "foreign-project (FlowTime) policy ids",
		"docs/explorations/surveys/flowtime/03-policies-workflow.md":                     "foreign-project (FlowTime) policy ids",
		"docs/explorations/surveys/flowtime/04-policies-rest.md":                         "foreign-project (FlowTime) policy ids + decisions D-053 etc.",
		"docs/explorations/surveys/flowtime/05-skills-needed.md":                         "foreign-project (FlowTime) skill mentions referencing E-25",
		"docs/explorations/surveys/flowtime/06-cross-cuts.md":                            "foreign-project (FlowTime) cross-cut policy mentions",
		"docs/explorations/surveys/liminara/00-survey.md":                                "foreign-project (Liminara) entity ids E-01..E-27",
		"docs/explorations/surveys/liminara/10-ai-scaffolding-policies.md":               "foreign-project (Liminara) policy ids referencing E-21..E-22",
		"docs/explorations/surveys/liminara/20-project-docs-policies.md":                 "foreign-project (Liminara) entity + policy ids (D-012/D-013, E-21..E-27)",
		"docs/explorations/surveys/liminara/30-conversation-and-claude-code-policies.md": "foreign-project (Liminara) policy ids",
		"docs/explorations/surveys/liminara/40-categorized.md":                           "foreign-project (Liminara) decision ids D-012/D-013",

		// Mining / policy-corpus exploratory docs reference foreign
		// or illustrative ids by design.
		"docs/explorations/03-policy-corpus-mining-and-the-agent-side.md": "foreign-project mentions (FlowTime's E-25 / M-066, post-E-24)",
		"docs/explorations/04-policy-system-ux-mining-and-compression.md": "illustrative policy id `M-9` (truth.precedence) — not an aiwf entity",
		"docs/explorations/05-policy-model-design.md":                     "illustrative example `E-3 / M-7` in policy-applicability table — not real ids",
		"docs/explorations/01-policies-design-space.md":                   "illustrative counterexample `M-12` — not a real aiwf entity",

		// Hypothetical worked-example design proposal — entities
		// E-08, E-12, E-13, M-041..M-052 are illustrative, do not
		// exist in the tree, and underpin the proposal's narrative.
		"docs/explorations/07-tdd-architecture-proposal.md": "hypothetical worked-example ids in deferred design proposal (E-08, E-12, E-13, M-041..M-052)",

		// CHANGELOG release notes describing what shipped at the
		// time. Entities born narrow stay narrow in the changelog
		// per the spec's release-note rule.
		"CHANGELOG.md": "release notes describe the historical state; entities born at narrow widths stay narrow per spec",
	}

	// Roots to scan. The walk excludes any directory named "archive"
	// (per ADR-0004 forget-by-default and the AC-3 spec).
	scanRoots := []string{
		"docs/explorations",
		"docs/pocv3/design",
		"docs/pocv3/plans",
	}
	// Top-level files.
	scanFiles := []string{
		"README.md",
		"CHANGELOG.md",
	}

	pat := regexp.MustCompile(`\b[EMGDC]-\d{1,3}\b`)

	type hit struct {
		Path string
		Line int
		Text string
	}
	var hits []hit

	scanFile := func(repoRel string) {
		full := filepath.Join(root, repoRel)
		data, err := os.ReadFile(full)
		if err != nil {
			t.Errorf("reading %s: %v", repoRel, err)
			return
		}
		lines := strings.Split(string(data), "\n")
		inFence := false
		for i, line := range lines {
			trimmed := strings.TrimLeft(line, " \t")
			// Toggle fenced-code state on lines starting with ``` or ~~~.
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inFence = !inFence
				continue
			}
			if inFence {
				continue
			}
			// Strip inline backtick spans (literal id text). Handles
			// single-backtick spans; the doc tree doesn't use double-
			// or triple-backtick inline spans in regular prose.
			stripped := stripInlineCode(line)
			if pat.MatchString(stripped) {
				hits = append(hits, hit{Path: repoRel, Line: i + 1, Text: line})
			}
		}
	}

	// Walk the configured roots (excluding archive subtrees).
	for _, base := range scanRoots {
		baseAbs := filepath.Join(root, base)
		err := filepath.Walk(baseAbs, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() == "archive" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".md") {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			rel = filepath.ToSlash(rel)
			if _, ok := allowlist[rel]; ok {
				return nil
			}
			scanFile(rel)
			return nil
		})
		if err != nil {
			t.Errorf("walking %s: %v", base, err)
		}
	}

	// Top-level files (non-recursive).
	for _, f := range scanFiles {
		if _, ok := allowlist[f]; ok {
			continue
		}
		scanFile(f)
	}

	if len(hits) > 0 {
		// Sort hits for stable output across runs.
		sort.Slice(hits, func(i, j int) bool {
			if hits[i].Path != hits[j].Path {
				return hits[i].Path < hits[j].Path
			}
			return hits[i].Line < hits[j].Line
		})
		// Group by path so the failure message reads as a list of files
		// to fix.
		var sb strings.Builder
		sb.WriteString("AC-3: narrow-id mentions found in non-archive doc-tree files:\n")
		var lastPath string
		for _, h := range hits {
			if h.Path != lastPath {
				sb.WriteString("  " + h.Path + ":\n")
				lastPath = h.Path
			}
			sb.WriteString("    line ")
			sb.WriteString(itoaBase(h.Line))
			sb.WriteString(": ")
			sb.WriteString(strings.TrimSpace(h.Text))
			sb.WriteString("\n")
		}
		sb.WriteString("\nEach hit is either:\n")
		sb.WriteString("  (a) a narrow-id mention that should be canonicalized (4-digit) per ADR-0008, or\n")
		sb.WriteString("  (b) a foreign-project / illustrative / historical mention that needs an allowlist entry in m083_doc_sweep_test.go.\n")
		t.Error(sb.String())
	}
}

// stripInlineCode replaces every `...` inline code span with an
// equivalent length of space characters so column-anchored regex
// matches outside spans still work, while contents inside spans are
// excluded from the match.
//
// Markdown inline code is delimited by paired single backticks; the
// doc tree doesn't use multi-backtick inline spans in prose, so the
// simple paired-single shape is sufficient.
func stripInlineCode(line string) string {
	out := make([]byte, len(line))
	inSpan := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '`' {
			inSpan = !inSpan
			out[i] = ' '
			continue
		}
		if inSpan {
			out[i] = ' '
			continue
		}
		out[i] = c
	}
	return string(out)
}

// itoaBase is a tiny int-to-string helper used by the failure
// message; pulling in strconv just for one call site is heavier than
// needed.
func itoaBase(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
