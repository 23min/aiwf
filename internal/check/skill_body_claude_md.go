package check

// G-0599: skill-body-claude-md-section rule.
//
// A shipped surface may name the consumer's own CLAUDE.md; it may not cite
// a section of one. The repo whose CLAUDE.md carries the cited sections is
// this one, and CLAUDE.md never ships: `aiwf init` / `aiwf update`
// materialize rituals into a consumer's `.claude/` and maintain a single
// marker-wrapped import line in the consumer's root CLAUDE.md, writing
// nothing else into it. A section citation therefore resolves here and
// dangles everywhere the surface actually runs — or resolves against a
// section the consumer happens to have named the same way, substituting
// their prose for the rule the surface meant.
//
// The shipped-surface rule this enforces predates it: a shipped surface
// carries only imperative, consumer-scoped instruction and cites no
// filesystem path. Its id half is mechanized by skill-body-id; this is the
// repo-context half, which drifted across eleven skills while unenforced.
//
// Scope and inertness match the sibling: the scan dirs are the authoring
// tree under internal/skills/embedded{,-rituals,-guidance}/, absent in a
// consumer repo, so the rule contributes nothing there.
//
// Unlike skill-body-id this scan applies no mask. That rule exempts
// non-prose link carriers so an id may ride in a doc-link destination; here
// a link is exactly as unresolvable as prose, so a citation dressed as one
// fires too.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/23min/aiwf/internal/tree"
)

// The CodeSkillClaudeMDSection constant is declared in check.go alongside
// the other finding codes per the closed-set convention (G-0129).

// claudeMDSectionPattern matches a citation into a named section of this
// repo's CLAUDE.md, in the two shapes such a citation takes.
//
// The first alternative is the "Per CLAUDE.md" idiom, which introduces a
// borrowed rule whether or not a section follows it. The second is the file
// name followed by a section marker: the section sign, a possessive opening
// a named section, an italicized section name, or a quoted one — with an
// optional closing backtick between, since the name is usually code-spanned.
//
// A bare mention is deliberately outside both: naming the consumer's own
// CLAUDE.md as a place their rules live is correct and must keep passing.
var claudeMDSectionPattern = regexp.MustCompile(
	`(?i)per\s+CLAUDE\.md` + "`" + `?\s*(?:§|'s|\*|")?` +
		`|CLAUDE\.md` + "`" + `?\s*(?:§|'s|\*|")`)

// ScanSkillClaudeMDSection returns one finding per distinct CLAUDE.md
// section citation in body, deduped within the file so a rule cited twice
// reports once. Findings are errors: a dangling citation in a shipped
// surface blocks the push.
//
// Path populates the finding locator only; the scanner is otherwise
// stateless, so it runs against on-disk content or literal test bytes.
func ScanSkillClaudeMDSection(body []byte, path string) []Finding {
	src := string(body)
	var findings []Finding
	seen := map[string]bool{}
	for _, m := range claudeMDSectionPattern.FindAllStringIndex(src, -1) {
		cite := src[m[0]:m[1]]
		if seen[cite] {
			continue
		}
		seen[cite] = true
		findings = append(findings, Finding{
			Code:     CodeSkillClaudeMDSection,
			Severity: SeverityError,
			Message:  fmt.Sprintf("shipped surface cites %q — CLAUDE.md is repo-development guidance and never ships, so the citation dangles in every consumer repo; state the rule inline instead", cite),
			Path:     path,
			Line:     1 + strings.Count(src[:m[0]], "\n"),
			Field:    "body",
		})
	}
	return findings
}

// skillClaudeMDSectionReference walks the authoring-source skill trees under
// the tree root and emits a finding for every *.md file citing a CLAUDE.md
// section. Each surface is scanned whole-file (frontmatter included), so a
// citation in a description: field fires alongside one in the body. The rule
// is inert when the scan dirs are absent (a consumer repo): each missing dir
// is skipped, so it contributes no findings rather than erroring.
func skillClaudeMDSectionReference(t *tree.Tree) []Finding {
	var findings []Finding
	for _, dir := range skillScanDirs {
		base := filepath.Join(t.Root, dir)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		_ = fs.WalkDir(os.DirFS(base), ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || strings.ToLower(filepath.Ext(p)) != ".md" {
				return nil
			}
			raw, readErr := os.ReadFile(filepath.Join(base, p))
			if readErr != nil {
				//coverage:ignore defensive: WalkDir just yielded this path; a read error here means the file vanished or became unreadable between walk and read (TOCTOU). Skip it like the sibling rule does.
				return nil
			}
			findings = append(findings, ScanSkillClaudeMDSection(raw, filepath.Join(dir, p))...)
			return nil
		})
	}
	return findings
}
