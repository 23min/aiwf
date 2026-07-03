package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// shippedSkillRoots are the embedded skill trees that `aiwf init` /
// `aiwf update` materialize into a consumer's `.claude/skills/<skill>/`.
var shippedSkillRoots = []string{
	"internal/skills/embedded",
	"internal/skills/embedded-rituals",
}

// mdLinkDestRe matches the destination of an inline markdown link: the
// `(...)` of `[text](dest)`. Only the inline form is scanned — reference-style
// (`[ref]: dest`) and angle-bracket (`](<dest>)`) links are not used by the
// shipped skills; extend this pattern if one is ever introduced.
var mdLinkDestRe = regexp.MustCompile(`\]\(([^)]+)\)`)

// linkHit is a repo-relative markdown link destination and the 1-based
// line it sits on.
type linkHit struct {
	line int
	dest string
}

// scanRepoRelativeLinks returns every markdown link destination in content
// that is neither an external URL nor a same-file anchor — i.e. a
// repo-relative path, dead in a consumer's materialized skill tree. Fenced
// code blocks (``` … ```) are skipped so an illustrative link inside an
// example block is not mistaken for a live one.
func scanRepoRelativeLinks(content string) []linkHit {
	var hits []linkHit
	inFence := false
	for i, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, m := range mdLinkDestRe.FindAllStringSubmatch(line, -1) {
			dest := strings.TrimSpace(m[1])
			if portableLinkDest(dest) {
				continue
			}
			hits = append(hits, linkHit{line: i + 1, dest: dest})
		}
	}
	return hits
}

// portableLinkDest reports whether a markdown link destination resolves for
// a consumer who received the materialized skill: an external URL or a
// same-file anchor. Everything else is repo-relative and dead there.
func portableLinkDest(dest string) bool {
	switch {
	case strings.HasPrefix(dest, "http://"), strings.HasPrefix(dest, "https://"):
		return true
	case strings.HasPrefix(dest, "#"):
		return true
	default:
		return false
	}
}

// TestShippedSkills_NoRepoRelativeLinks pins M-0229/AC-2: no shipped skill
// markdown link points into the repo tree. A skill materializes into a
// consumer's `.claude/skills/<skill>/` where no repo path (`docs/`,
// `internal/`, `work/`, …) exists, so any repo-relative link destination is
// dead there regardless of which tree it targets. Only external URLs and
// same-file anchors are portable.
//
// The predicate is universal, not a per-tree allowlist: the shipped skills
// carry no legitimate repo-relative link, so the rule needs no exceptions
// and can never gap on a newly-referenced tree.
func TestShippedSkills_NoRepoRelativeLinks(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	var offenders []string
	for _, rel := range shippedSkillRoots {
		err := filepath.WalkDir(filepath.Join(root, rel), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel(root, path)
			for _, h := range scanRepoRelativeLinks(string(data)) {
				offenders = append(offenders, fmt.Sprintf("%s:%d → %s", relPath, h.line, h.dest))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", rel, err)
		}
	}
	if len(offenders) > 0 {
		t.Errorf("M-0229/AC-2: shipped skills must carry no repo-relative markdown link "+
			"(a materialized skill ships without the repo tree, so the link is dead in a consumer) — %d found:\n  %s",
			len(offenders), strings.Join(offenders, "\n  "))
	}
}

// TestScanRepoRelativeLinks_ClassifiesDestinations exercises the scanner's
// branches on synthetic content so the guard is proven non-vacuous
// independent of the real tree: a prose repo-relative link is caught, an
// external URL and a same-file anchor are allowed, and a link inside a
// fenced block is skipped.
func TestScanRepoRelativeLinks_ClassifiesDestinations(t *testing.T) {
	t.Parallel()
	content := strings.Join([]string{
		"See [the ADR](../../../../docs/adr/ADR-9999-x.md) for more.",
		"An [external](https://example.com) link is fine.",
		"An [anchor](#section) is fine.",
		"A [placeholder](work/epics/E-9999/M-9999.md) is repo-relative.",
		"```",
		"[fenced](../docs/should-be-ignored.md)",
		"```",
	}, "\n")

	hits := scanRepoRelativeLinks(content)
	got := map[string]bool{}
	for _, h := range hits {
		got[h.dest] = true
	}
	if !got["../../../../docs/adr/ADR-9999-x.md"] {
		t.Error("scanner missed the prose docs/adr repo-relative link")
	}
	if !got["work/epics/E-9999/M-9999.md"] {
		t.Error("scanner missed the prose work/epics repo-relative link")
	}
	if got["../docs/should-be-ignored.md"] {
		t.Error("scanner flagged a link inside a fenced code block")
	}
	if len(hits) != 2 {
		t.Errorf("scanner returned %d hits; want 2 (the two prose repo-relative links)", len(hits))
	}
}

// TestPortableLinkDest covers the classifier's branches directly.
func TestPortableLinkDest(t *testing.T) {
	t.Parallel()
	cases := []struct {
		dest string
		want bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"#anchor", true},
		{"../../docs/adr/ADR-9999.md", false},
		{"work/epics/E-9999/M-9999.md", false},
	}
	for _, c := range cases {
		if got := portableLinkDest(c.dest); got != c.want {
			t.Errorf("portableLinkDest(%q) = %v; want %v", c.dest, got, c.want)
		}
	}
}
