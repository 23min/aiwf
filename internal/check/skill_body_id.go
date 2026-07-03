package check

// G-0299: skill-body-id rule.
//
// The mirror image of body-prose-id (G-0184). body-prose-id walks ENTITY
// bodies, where a real id is required and a placeholder is the defect.
// This rule walks shipped Markdown surfaces whole-file (SKILL.md bodies
// AND descriptions, entity templates, role-agent cards, and the guidance
// fragment), where the polarity is inverted: a real (digit-bearing) entity
// id is the defect and a canonical letter-N placeholder is correct.
//
// Why: these surfaces ship to consumer repos (materialized into
// `.claude/` by `aiwf init` / `aiwf update`). aiwf's own ids are
// meaningless in a consumer tree and rot as entities change status /
// archive / rewidth, so a real-id reference in a shipped surface is both
// stale-prone and contextually wrong. Illustrative content uses
// canonical-shape placeholders (`G-NNNN`) or shape-descriptions; a
// markdown link to a design/ADR doc is the one carve-out.
//
// Dogfooding scope: the authoring source for these surfaces lives under
// this repo's `internal/skills/embedded{,-rituals,-guidance}/`. A consumer
// repo has no such tree, so the rule is inert there by construction (the
// dirs are absent). This is why the rule lives in internal/check (pre-push,
// the earliest in-context tier for aiwf's own development) rather than a
// CI-only policy test — and why it costs consumers nothing.
//
// Carve-out for free: the scan reuses body-prose-id's proseMask, which
// exempts code constructs AND non-prose link carriers (destinations,
// titles, reference definitions, autolinks). So a doc-link whose
// destination is `docs/.../ADR-NNNN-*.md` is silent automatically — the id
// rides in the destination, the visible link text is descriptive prose.
// Citing the id as the visible link TEXT is an inline citation, not a
// carve-out, and fires.

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/tree"
)

// The CodeSkillBodyID constant is declared in check.go alongside the
// other finding codes per the closed-set convention (G-0129).

// skillScanDirs are the authoring-source roots scanned for real-id
// references in shipped Markdown surfaces, relative to the tree root.
// Every *.md under these roots is scanned whole-file (frontmatter
// included) — SKILL.md bodies AND descriptions, entity templates,
// role-agent cards, and the always-on guidance fragment. Absent in a
// consumer repo, which is what makes the rule inert there.
var skillScanDirs = []string{
	filepath.Join("internal", "skills", "embedded"),
	filepath.Join("internal", "skills", "embedded-rituals"),
	filepath.Join("internal", "skills", "embedded-guidance"),
}

// ScanSkillBodyID classifies every id-shaped token in the given content
// (a whole shipped *.md file, frontmatter included, or a bare body) and
// returns one finding per unique real-id token, deduped within this
// content. A token fires only when it matches a kind's strict,
// digit-bearing id pattern (bare or composite); canonical letter-N
// placeholders and malformed shapes are not this rule's concern
// (placeholder normalization is policed separately).
//
// Non-prose content is masked (not stripped) via proseMask before
// scanning, so byte offsets stay stable and tokens inside code constructs
// or non-prose link carriers are exempt by construction. Finding.Line is
// 1-based within the given content; when the caller passes the whole file
// (skillBodyIDReference does), that line is already file-relative.
//
// Path populates the finding locator only; the scanner is otherwise
// stateless, so it runs against on-disk content (skillBodyIDReference) or
// against literal test bytes.
func ScanSkillBodyID(body []byte, path string) []Finding {
	masked := proseMask(body)

	var findings []Finding
	seen := map[string]bool{}
	for _, m := range idTokenPattern.FindAllStringIndex(masked, -1) {
		tok := masked[m[0]:m[1]]
		if !strictBareIDPattern.MatchString(tok) && !strictCompositeIDPattern.MatchString(tok) {
			continue
		}
		if seen[tok] {
			continue
		}
		seen[tok] = true
		line := 1 + bytes.Count(body[:m[0]], []byte{'\n'})
		findings = append(findings, Finding{
			Code:     CodeSkillBodyID,
			Severity: SeverityError,
			Message:  fmt.Sprintf("skill body cites real entity id %q — shipped skills use a canonical placeholder (e.g. G-NNNN) or a design/ADR doc-link, not a real id", tok),
			Path:     path,
			Line:     line,
			Field:    "body",
		})
	}
	return findings
}

// skillBodyIDReference walks the authoring-source skill trees under the
// tree root and emits skill-body-id findings for every *.md file whose
// content cites a real entity id. Each Markdown surface is scanned
// whole-file (frontmatter included), so a real id in a description: field
// or a template's frontmatter comment fires alongside one in the body.
// The rule is inert when the scan dirs are absent (a consumer repo): each
// missing dir is skipped, so the rule contributes no findings rather than
// erroring.
func skillBodyIDReference(t *tree.Tree) []Finding {
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
			full := filepath.Join(base, p)
			raw, readErr := os.ReadFile(full)
			if readErr != nil {
				//coverage:ignore defensive: WalkDir just yielded this path; a read error here means the file vanished or became unreadable between walk and read (TOCTOU). Skip it like body-prose-id does.
				return nil
			}
			// The finding path is repo-relative: dir is already
			// repo-relative and p is relative to base (= Root/dir), so
			// dir/p is the repo-relative path without a filepath.Rel call.
			// The whole file is scanned, so ScanSkillBodyID's line is
			// already file-relative — no body-offset adjustment.
			rel := filepath.Join(dir, p)
			findings = append(findings, ScanSkillBodyID(raw, rel)...)
			return nil
		})
	}
	return findings
}
