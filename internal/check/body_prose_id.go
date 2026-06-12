package check

// G-0184: body-prose-id rule.
//
// Walks every active entity's body prose for tokens that look like
// aiwf ids and classifies each into one of three failure modes:
//
//   - malformed-shape — the token has a known prefix (E-/M-/G-/D-/C-/
//     ADR-) but the suffix is not a valid id shape. Catches the literal
//     LLM Phase-A/B labeling anti-pattern (M-a, M-alpha), uppercase
//     placeholder leaks (M-NNNN, E-NN), compound English-word suffixes
//     (ADR-shaped, C-option), and narrow-numeric forms (M-1, E-1) that
//     don't match the kind's strict pattern. The narrow-numeric case
//     covers conversational labels that leak from chat into a committed
//     body without being upgraded to the allocator-assigned canonical id.
//
//   - unresolved — the token matches the kind's strict pattern but
//     resolves to no entity in the tree. Catches fabricated canonical-
//     width tokens (M-9999) and stale references to deleted entities.
//
//   - unresolved-milestone / unresolved-ac — composite ids (M-NNN/AC-N)
//     whose parent milestone is missing, or whose parent is present but
//     has no AC at the named position. Mirror the subcodes refsResolve
//     emits for the structured-frontmatter composite case.
//
// Backtick exemption: inline code spans (`...`) and fenced code blocks
// (```...``` and ~~~...~~~) are stripped before the scan, so prose
// discussing id syntax (`M-NNN` in CLAUDE.md prose, `^M-\d{3,}$` regex
// quotes, command examples) does not self-trip.
//
// Archive scoping: archive entities are skipped, mirroring refsResolve
// per ADR-0004 §"Check shape rules".
//
// Frontmatter is split off via entity.Split before the scan; structured
// frontmatter references are already covered by refsResolve.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// The CodeBodyProseID constant is declared in check.go alongside the
// other finding codes per the closed-set convention (G-0129).

// idTokenPattern picks up any token shaped like an aiwf id: a known
// prefix followed by an alphanumeric suffix, with an optional
// composite `/AC-<suffix>` tail. Loose by design — the classifier
// below decides malformed-shape vs strict-form-unresolved vs silent.
//
// The trailing `\b` boundary matches at the transition between a word
// character (digit/letter) and a non-word character, so tokens
// embedded in URL paths or sentence punctuation are picked up cleanly.
var idTokenPattern = regexp.MustCompile(`\b(?:E|M|G|D|C|ADR)-[A-Za-z0-9_]+(?:/AC-[A-Za-z0-9_]+)?\b`)

// strictBareIDPattern matches strict-form bare ids per kind. Anchored
// for whole-token matching after idTokenPattern picks the candidate.
// Widths mirror entity.idPatterns: E ≥ 2 digits, M/G/D/C ≥ 3, ADR ≥ 4.
var strictBareIDPattern = regexp.MustCompile(`^(?:E-\d{2,}|M-\d{3,}|G-\d{3,}|D-\d{3,}|C-\d{3,}|ADR-\d{4,})$`)

// strictCompositeIDPattern matches strict-form composite ids.
// Mirrors entity.compositeIDPattern.
var strictCompositeIDPattern = regexp.MustCompile(`^M-\d{3,}/AC-\d+$`)

// backtickFencePattern matches ```...``` fenced blocks.
// tildeFencePattern matches ~~~...~~~ fenced blocks.
// Split into two patterns because RE2 has no backreferences; a fence
// opened with ``` only closes on ```, never on ~~~.
var (
	backtickFencePattern = regexp.MustCompile("(?s)```.*?```")
	tildeFencePattern    = regexp.MustCompile("(?s)~~~.*?~~~")
)

// codeSpanPattern matches a single-backtick inline code span on one
// line. CommonMark allows multi-backtick spans, but the single-tick
// form covers every id-syntax discussion in the live tree.
var codeSpanPattern = regexp.MustCompile("`[^`\n]*`")

// bodyProseID emits one finding per (entity, token, subcode) for any
// id-shaped token in entity body prose that does not resolve to an
// allocated entity. Dedupe is per-token-per-entity so repeated
// mentions of the same bad token in one body produce one finding,
// not one per occurrence.
func bodyProseID(t *tree.Tree) []Finding {
	idx := BodyProseIDIndex(t)
	var findings []Finding
	for _, e := range t.Entities {
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(t.Root, e.Path))
		if err != nil {
			continue
		}
		_, body, ok := entity.Split(raw)
		if !ok {
			continue
		}
		scanned := ScanBodyProseID(body, e.ID, e.Path, idx)
		// Adjust body-relative Line to file-relative. The body starts
		// after the frontmatter delimiter; count newlines in the
		// pre-body bytes and add to each finding.
		if offset := bytes.Index(raw, body); offset > 0 {
			preBody := bytes.Count(raw[:offset], []byte{'\n'})
			for i := range scanned {
				scanned[i].Line += preBody
			}
		}
		findings = append(findings, scanned...)
	}
	return findings
}

// BodyProseIndex is the two-tier id-resolution view ScanBodyProseID
// consults.
//
// ByID is the primary tier: canonicalized id → entity for every
// working-tree entity (active + archive) and stub. Stubs are included
// so a body referencing an entity whose file failed to parse resolves
// silently (the parse failure is already reported as a load-error
// finding; re-reporting via body-prose-id would be noise).
//
// Trunk is the second tier (G-0241): the canonicalized id set observed
// on the configured trunk ref. A strict-form token that misses ByID
// but hits Trunk is silent — the id IS allocated, just not visible in
// this branch's working tree (typical case: an entity filed on trunk
// in another session while this branch is in flight). Resolution is
// thereby symmetric with allocation, where AllocateID already treats
// trunk ids as authoritative. Nil Trunk (in-memory test trees,
// no-remote repos, dispatchers that load without a trunk read)
// degrades to primary-tier-only behavior — the pre-G-0241 default.
type BodyProseIndex struct {
	ByID  map[string]*entity.Entity
	Trunk map[string]bool
}

// BodyProseIDIndex builds the id-resolution index that ScanBodyProseID
// consumes from the tree's entities, stubs, and trunk-id set.
//
// Exposed so verbs that scan planned-write body content at verb time
// (G-0184 verb-time scan) share the index with the tree-walking
// bodyProseID rule. Verbs should build the index once before the loop
// over planned files, then pass it to ScanBodyProseID per file.
func BodyProseIDIndex(t *tree.Tree) BodyProseIndex {
	idx := BodyProseIndex{
		ByID: make(map[string]*entity.Entity, len(t.Entities)+len(t.Stubs)),
	}
	for _, e := range t.Entities {
		key := entity.Canonicalize(e.ID)
		if _, exists := idx.ByID[key]; exists {
			continue
		}
		idx.ByID[key] = e
	}
	for _, e := range t.Stubs {
		key := entity.Canonicalize(e.ID)
		if _, exists := idx.ByID[key]; exists {
			continue
		}
		idx.ByID[key] = e
	}
	if len(t.TrunkIDs) > 0 {
		idx.Trunk = make(map[string]bool, len(t.TrunkIDs))
		for _, tid := range t.TrunkIDs {
			idx.Trunk[entity.Canonicalize(tid.ID)] = true
		}
	}
	return idx
}

// ScanBodyProseID classifies every id-shaped token in body (the bytes
// after the YAML frontmatter delimiter) and returns one finding per
// unique (token, subcode) pair, deduped within this body. Path and
// entityID are used only to populate the Finding's locator fields —
// the scanner is otherwise stateless, so it can run against on-disk
// content (the tree-walking bodyProseID rule) or against planned-
// write bytes that don't yet exist on disk (verb-time pre-flight).
//
// Code spans (`...`) and fenced blocks (``` and ~~~) are masked (not
// stripped) before scanning, so byte offsets in the input remain
// stable across the masking step. Finding.Line is set to the 1-based
// line number within body where the matched token starts; callers
// that want file-relative Line (the bodyProseID tree-walk rule) add
// the body's start-of-file line offset themselves.
//
// The idx parameter is the resolution index from BodyProseIDIndex;
// callers that scan multiple bodies should build it once and reuse.
func ScanBodyProseID(body []byte, entityID, path string, idx BodyProseIndex) []Finding {
	// Mask code spans and fences with same-length runs of spaces so the
	// regex doesn't match tokens inside them but byte offsets stay
	// aligned with the original body for line-number resolution.
	masked := maskSameLength(backtickFencePattern, string(body))
	masked = maskSameLength(tildeFencePattern, masked)
	masked = maskSameLength(codeSpanPattern, masked)

	var findings []Finding
	seen := map[string]bool{}
	for _, m := range idTokenPattern.FindAllStringIndex(masked, -1) {
		tok := masked[m[0]:m[1]]
		subcode, msg := classifyBodyToken(tok, idx)
		if subcode == "" {
			continue
		}
		key := tok + ":" + subcode
		if seen[key] {
			continue
		}
		seen[key] = true
		line := 1 + bytes.Count(body[:m[0]], []byte{'\n'})
		findings = append(findings, Finding{
			Code:     CodeBodyProseID,
			Severity: SeverityError,
			Subcode:  subcode,
			Message:  fmt.Sprintf("%s body prose contains %s", entityID, msg),
			Path:     path,
			Line:     line,
			EntityID: entityID,
			Field:    "body",
		})
	}
	return findings
}

// maskSameLength replaces every match of re in s with a run of spaces
// of the same length as the match. Used by ScanBodyProseID so byte
// offsets in the input stay aligned after code-span and code-fence
// suppression — line-number resolution depends on the alignment.
func maskSameLength(re *regexp.Regexp, s string) string {
	return re.ReplaceAllStringFunc(s, func(m string) string {
		return strings.Repeat(" ", len(m))
	})
}

// classifyBodyToken returns the finding subcode and detail message for
// a candidate token, or ("", "") if the token resolves cleanly.
//
// Resolution order per tier: the working-tree index (idx.ByID) is
// authoritative when it has the id — a locally-visible milestone with
// a missing AC fires unresolved-ac even if the id also appears on
// trunk. The trunk tier (idx.Trunk) is consulted only on a ByID miss
// (G-0241): a strict-form token known on trunk is silent. For a
// composite token whose parent is trunk-only, the AC position cannot
// be validated without the parent's file, so the whole token is
// silent — refusing would re-create the verb-time refusal G-0241
// fixes, and the position is validated by the tree-walking rule once
// the file is visible (post rebase/merge). Malformed-shape tokens
// never reach the trunk tier: trunk ids are strict-form by
// construction, so trunk membership cannot launder a malformed token.
func classifyBodyToken(tok string, idx BodyProseIndex) (subcode, msg string) {
	if strictCompositeIDPattern.MatchString(tok) {
		parent, sub, _ := entity.ParseCompositeID(tok)
		canonParent := entity.Canonicalize(parent)
		parentEntity, ok := idx.ByID[canonParent]
		if !ok {
			if idx.Trunk[canonParent] {
				return "", ""
			}
			return "unresolved-milestone", fmt.Sprintf("composite id %q whose parent %q is not allocated", tok, parent)
		}
		for _, ac := range parentEntity.ACs {
			if ac.ID == sub {
				return "", ""
			}
		}
		return "unresolved-ac", fmt.Sprintf("composite id %q but %s has no %s in acs[]", tok, parent, sub)
	}
	if strictBareIDPattern.MatchString(tok) {
		canon := entity.Canonicalize(tok)
		if _, ok := idx.ByID[canon]; ok {
			return "", ""
		}
		if idx.Trunk[canon] {
			return "", ""
		}
		return "unresolved", fmt.Sprintf("unknown id %q (no entity allocated at this id)", tok)
	}
	return "malformed-shape", fmt.Sprintf("id-shaped token %q that does not match the kind's strict id pattern (wrap in backticks if discussing id syntax)", tok)
}
