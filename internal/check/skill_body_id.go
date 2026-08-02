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
// Two shapes offend here: a real id, and a placeholder at any shape other
// than the canonical letter-N form. They share a remediation — write
// `<prefix>-NNNN` — which is why one finding code covers both.
//
// The doc-link carve-out: the scan masks non-prose link carriers
// (destinations, titles, reference definitions, autolinks), so a doc-link
// whose destination is `docs/.../ADR-NNNN-*.md` is silent automatically —
// the id rides in the destination, the visible link text is descriptive
// prose. Citing the id as the visible link TEXT is an inline citation, not
// a carve-out, and fires. Code constructs are NOT a carve-out: a command
// example ships to a consumer exactly as prose does. That is the one point
// where this scan parts company with body-prose-id, which shares the
// walker but keeps code exempt.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
// returns one finding per unique offending token, deduped within this
// content. Two shapes offend: a strict, digit-bearing id (bare or
// composite), and any placeholder that is not the canonical letter-N form.
// A canonical placeholder is the one shape that passes.
//
// Non-prose content is masked (not stripped) via proseAndCodeMask before
// scanning, so byte offsets stay stable. Code constructs ARE in scope — a
// real id in a command example ships and rots exactly as one in prose
// does — while non-prose link carriers stay exempt, preserving the
// doc-link carve-out. Finding.Line is 1-based within the given content;
// when the caller passes the whole file (skillBodyIDReference does), that
// line is already file-relative.
//
// Findings are warnings while the shipped tree still carries the debris
// this rule now detects: at error severity an incomplete sweep would block
// every push. The sweep milestone clears the tree and raises the severity
// as its last act.
//
// Path populates the finding locator only; the scanner is otherwise
// stateless, so it runs against on-disk content (skillBodyIDReference) or
// against literal test bytes.
func ScanSkillBodyID(body []byte, path string) []Finding {
	return scanMaskedForSkillIDs(proseAndCodeMask(body), path)
}

// canonicalPlaceholderPattern matches the one placeholder shape a shipped
// surface may carry: the canonical-width letter-N form. Anything id-shaped
// that is neither a real id nor this is a placeholder defect — a narrow width
// (`E-NN`), an idiosyncratic shape (`G-XYZ`), or a pseudo-arithmetic form all
// model an id shape that no allocator emits and no parser accepts.
//
// The composite arm is milestone-only, mirroring the id grammar: acceptance
// criteria hang off milestones, so a composite placeholder on any other kind
// names a shape that cannot exist.
var canonicalPlaceholderPattern = regexp.MustCompile(`^(?:E|G|D|C|ADR)-NNNN$|^M-NNNN(?:/AC-N)?$`)

// skillTokenClass names how an id-shaped token in a shipped surface reads.
type skillTokenClass int

const (
	// tokenOK is a canonical placeholder: the shape these surfaces exist to
	// teach.
	tokenOK skillTokenClass = iota
	// tokenRealID is a digit-bearing id — meaningless in a consumer repo and
	// stale-prone in this one.
	tokenRealID
	// tokenBadPlaceholder is an id-shaped token that is neither.
	tokenBadPlaceholder
)

// classifySkillToken sorts one id-shaped token into the classes above. A
// narrow NUMERIC id (`E-01`) is a real id at a legacy width, not a
// placeholder: read tolerance keeps it resolving, so it classifies as a
// citation rather than a width defect.
func classifySkillToken(tok string) skillTokenClass {
	if strictBareIDPattern.MatchString(tok) || strictCompositeIDPattern.MatchString(tok) {
		return tokenRealID
	}
	if canonicalPlaceholderPattern.MatchString(tok) {
		return tokenOK
	}
	return tokenBadPlaceholder
}

// scanMaskedForSkillIDs classifies every id-shaped token in masked — the
// same-length, exempt-content-blanked projection of a source produced by
// proseAndCodeMask (Markdown) or shellCommentMask (shell comments) — and
// returns one finding per unique offending token, deduped within masked.
// Both masks preserve newline positions, so the line counted in masked is
// the source line.
func scanMaskedForSkillIDs(masked, path string) []Finding {
	var findings []Finding
	seen := map[string]bool{}
	for _, m := range idTokenPattern.FindAllStringIndex(masked, -1) {
		tok := masked[m[0]:m[1]]
		cls := classifySkillToken(tok)
		if cls == tokenOK {
			continue
		}
		if seen[tok] {
			continue
		}
		seen[tok] = true
		line := 1 + strings.Count(masked[:m[0]], "\n")
		findings = append(findings, Finding{
			Code:     CodeSkillBodyID,
			Severity: SeverityWarning,
			Message:  skillTokenMessage(cls, tok),
			Path:     path,
			Line:     line,
			Field:    "body",
		})
	}
	return findings
}

// skillTokenMessage states what is wrong with tok. Both classes share a
// remediation — write the canonical letter-N placeholder — so the two
// messages differ in the defect they name, not in the fix they ask for.
func skillTokenMessage(cls skillTokenClass, tok string) string {
	if cls == tokenRealID {
		return fmt.Sprintf("shipped surface cites real entity id %q — shipped surfaces use a canonical placeholder (e.g. G-NNNN) or a design/ADR doc-link, not a real id", tok)
	}
	return fmt.Sprintf("shipped surface uses non-canonical placeholder %q — shipped surfaces use the canonical letter-N form (e.g. G-NNNN, M-NNNN/AC-N), which is the id shape a reader should learn", tok)
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
// first non-whitespace character OR immediately preceded by a shell word
// boundary — whitespace or a word-terminating metacharacter (`;` `|` `&`
// `(` `)` `<` `>`) — and runs to end-of-line. This matches bash's rule (a
// word beginning with '#' starts a comment) in the leak-safe direction: a
// real id in a `;#` / `)#` trailing comment fires rather than shipping
// silently. The rule exempts the common shell forms where '#' is not a
// comment: parameter expansion (`${x#foo}`, `${x##*/}` — '#' preceded by a
// letter or '#'), the positional-count `$#` ('#' preceded by '$'), and
// (harmlessly) the `#!` shebang, which carries no id.
//
// Deliberately ignored edge cases — KISS, since this scans a single file we
// author, not a general shell tokenizer: a '#' inside a quoted string that
// is preceded by a boundary char (`echo "a # b"`, `echo "a;#b"`) is treated
// as a comment start, so a real id there would fire — acceptable, as a real
// id in a shipped statusline string is itself a leak; here-doc bodies; and
// backslash line-continuation.
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
		case b == '#' && (!sawNonSpace || (i > lineStart && shellWordBoundary(src[i-1]))):
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

// shellWordBoundary reports whether b terminates a shell word, so a '#'
// immediately after it begins a comment. Bash treats whitespace and the
// metacharacters ; | & ( ) < > as word terminators.
func shellWordBoundary(b byte) bool {
	switch b {
	case ' ', '\t', ';', '|', '&', '(', ')', '<', '>':
		return true
	default:
		return false
	}
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
		findings = append(findings, scanMaskedForSkillIDs(shellCommentMask(raw), rel)...)
		return nil
	})
	return findings
}
