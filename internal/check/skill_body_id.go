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
	return scanMaskedForRealIDs(proseMask(body), path)
}

// scanMaskedForRealIDs classifies every id-shaped token in masked — the
// same-length, exempt-content-blanked projection of a source produced by
// proseMask (Markdown prose) or shellCommentMask (shell comments) — and
// returns one finding per unique real-id token, deduped within masked. A
// token fires only when it matches a kind's strict, digit-bearing id
// pattern (bare or composite); canonical letter-N placeholders and
// malformed shapes are not this rule's concern. Both masks preserve
// newline positions, so the line counted in masked is the source line.
func scanMaskedForRealIDs(masked, path string) []Finding {
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
		line := 1 + strings.Count(masked[:m[0]], "\n")
		findings = append(findings, Finding{
			Code:     CodeSkillBodyID,
			Severity: SeverityError,
			Message:  fmt.Sprintf("shipped surface cites real entity id %q — shipped surfaces use a canonical placeholder (e.g. G-NNNN) or a design/ADR doc-link, not a real id", tok),
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

// statuslineScanDir is the authoring-source root for the shipped statusline
// script, relative to the tree root. Absent in a consumer repo, which is
// what makes the rule inert there.
var statuslineScanDir = filepath.Join("internal", "skills", "embedded-statusline")

// shellCommentMask returns a same-length copy of src in which every byte
// outside a shell comment is replaced with a space (newlines preserved, so
// downstream line-number resolution stays exact). The scanner then runs
// against comment text only — a real id in shell CODE (a string literal, a
// parameter expansion, a variable) is exempt by construction, the shell
// analogue of proseMask's code-span carve-out.
//
// A comment starts at the first '#' on a line that is either the line's
// first non-whitespace character OR immediately preceded by a space or tab,
// and runs to end-of-line. That rule exempts the common shell forms where
// '#' is not a comment: parameter expansion (`${x#foo}`, `${x##*/}` — '#'
// preceded by a letter or '#'), the positional-count `$#` ('#' preceded by
// '$'), and (harmlessly) the `#!` shebang, which carries no id.
//
// Deliberately ignored edge cases — KISS, since this scans a single file we
// author, not a general shell tokenizer: a '#' inside a quoted string that
// is preceded by whitespace (`echo "a # b"`) is treated as a comment start,
// so a real id there would fire — acceptable, as a real id in a shipped
// statusline string is itself a leak; here-doc bodies; and backslash
// line-continuation.
func shellCommentMask(src []byte) string {
	masked := make([]byte, len(src))
	lineStart := 0
	sawNonSpace := false
	inComment := false
	for i := 0; i < len(src); i++ {
		b := src[i]
		switch {
		case b == '\n':
			masked[i] = '\n'
			lineStart = i + 1
			sawNonSpace = false
			inComment = false
		case inComment:
			masked[i] = b
		case b == '#' && (!sawNonSpace || (i > lineStart && (src[i-1] == ' ' || src[i-1] == '\t'))):
			inComment = true
			masked[i] = b
		default:
			masked[i] = ' '
			if b != ' ' && b != '\t' {
				sawNonSpace = true
			}
		}
	}
	return string(masked)
}

// statuslineCommentIDReference walks the statusline authoring tree under the
// tree root and emits skill-body-id findings for every *.sh file whose
// COMMENTS cite a real entity id. Shell has no Markdown prose mask, so
// shellCommentMask selects comment text and exempts shell code. The rule is
// inert when the dir is absent (a consumer repo): the walk is skipped, so it
// contributes no findings rather than erroring.
func statuslineCommentIDReference(t *tree.Tree) []Finding {
	base := filepath.Join(t.Root, statuslineScanDir)
	if _, err := os.Stat(base); err != nil {
		return nil
	}
	var findings []Finding
	_ = fs.WalkDir(os.DirFS(base), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.ToLower(filepath.Ext(p)) != ".sh" {
			return nil
		}
		raw, readErr := os.ReadFile(filepath.Join(base, p))
		if readErr != nil {
			//coverage:ignore defensive: WalkDir just yielded this path; a read error here means the file vanished or became unreadable between walk and read (TOCTOU). Skip it.
			return nil
		}
		rel := filepath.Join(statuslineScanDir, p)
		findings = append(findings, scanMaskedForRealIDs(shellCommentMask(raw), rel)...)
		return nil
	})
	return findings
}
