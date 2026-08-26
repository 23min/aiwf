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
//   - narrow-width — the token resolves, but only once padded to
//     canonical width, and no entity is stored under the narrow
//     spelling. Catches a citation of a real entity written the way an
//     older tree spelled ids (E-19 for E-0019).
//
//   - unresolved-milestone / unresolved-ac — composite ids (M-NNN/AC-N)
//     whose parent milestone is missing, or whose parent is present but
//     has no AC at the named position. Mirror the subcodes refsResolve
//     emits for the structured-frontmatter composite case.
//
// Code exemption (G-0240): the body is masked through a CommonMark
// parse (goldmark) before the scan, so only prose-visible text is
// scanned. Tokens inside any code construct — inline code spans of
// any backtick count, fenced blocks (``` and ~~~), indented code
// blocks — and inside non-prose link carriers (destinations,
// reference-link definitions, autolinks) are exempt by construction,
// so prose discussing id syntax (`M-NNN` in CLAUDE.md prose,
// `^M-\d{3,}$` regex quotes, command examples) does not self-trip.
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

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/trunk"
)

// The CodeBodyProseID constant is declared in check.go alongside the
// other finding codes per the closed-set convention (G-0129).

// idTokenPattern picks up any token shaped like an aiwf id: a known
// prefix followed by a letter/digit/underscore suffix, with an optional
// composite `/AC-<suffix>` tail. Loose by design — the classifier
// below decides malformed-shape vs strict-form-unresolved vs silent.
//
// The suffix is matched in two alternatives, not by one Unicode class,
// because neither shape of a single class works. Keep the trailing `\b`
// and the class rejects what the widening exists to admit: Go's `\b` is
// ASCII-only, so it cannot match after a non-ASCII letter and `M-α`
// yields no token at all. Drop the `\b` and the class has no right
// boundary, so it runs on through whatever follows a real id where a
// script sets no space between words — the citation in `M-0001の仕様`
// becomes one malformed token.
//
// The split keeps both properties. The first alternative governs ASCII
// input; its class stops at the first non-ASCII rune, which is the
// boundary a lone Unicode class lacks. The second admits a suffix that
// starts outside ASCII, which is how `G-α` becomes a candidate at all.
//
// The second alternative has no right boundary of its own: in
// `G-αはM-0001` it runs on, and the real id inside the run is not
// scanned as its own token. The run-on itself always fires — it can
// match neither strict pattern — so nothing passes silently. Widening
// that class to also reach a punctuation suffix costs more than it
// repays: `M-…` is the shape-notation for an unallocated id, and
// skillBodyID scans code spans, so any class admitting `…` fires on the
// shipped guidance fragment that writes it deliberately.
var idTokenPattern = regexp.MustCompile(`\b(?:E|M|G|D|C|ADR)-(?:[A-Za-z0-9_]+(?:/AC-[A-Za-z0-9_]+)?\b|[\p{L}\p{N}_]+)`)

// strictBareIDPattern matches strict-form bare ids per kind. Anchored
// for whole-token matching after idTokenPattern picks the candidate.
// Widths mirror entity.idPatterns: E ≥ 2 digits, M/G/D/C ≥ 3, ADR ≥ 4.
var strictBareIDPattern = regexp.MustCompile(`^(?:E-\d{2,}|M-\d{3,}|G-\d{3,}|D-\d{3,}|C-\d{3,}|ADR-\d{4,})$`)

// strictCompositeIDPattern matches strict-form composite ids.
// Mirrors entity.compositeIDPattern.
var strictCompositeIDPattern = regexp.MustCompile(`^M-\d{3,}/AC-\d+$`)

// proseMaskEngine is the package-level goldmark instance whose parser
// decides what counts as prose for the body-prose-id scan. Plain
// CommonMark, no extensions — deliberately narrower than htmlrender's
// render engine: its Linkify extension would turn bare URLs into
// autolinks and silently exempt id-shaped tokens inside them, which
// the rule scans as prose today. Immutable after init, same idiom as
// a compiled regexp (and as htmlrender.markdownEngine).
var proseMaskEngine = goldmark.New()

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
//
// CrossBranch is the third tier (M-0259/AC-2): canonicalized id →
// the cross-branch hits carrying it (see crossBranchIndex). Consulted
// only after both ByID and Trunk miss. Unlike Trunk, a hit here is
// VISIBLE — because a sibling branch is provisional (it can be
// rebased, renamed, or abandoned before merging), unlike trunk which is
// authoritative. Whether it blocks depends on the refs carrying it: a
// published id is the non-blocking cross-branch-pending, one on local
// branch refs alone is cross-branch-local-only at error severity
// (ADR-0041).
type BodyProseIndex struct {
	ByID        map[string]*entity.Entity
	Trunk       map[string]bool
	CrossBranch map[string][]trunk.RefHit
	// Collisions is t.CrossBranchCollisions verbatim (M-0259/AC-3): the
	// canonicalized-id set whose cross-branch hits diverge in content.
	// classifyBodyToken escalates a CrossBranch hit here to the
	// distinct cross-branch-collision subcode instead of the ordinary
	// cross-branch-pending one — both are non-blocking warnings
	// (D-0036). Read only once the id is known to be published: an
	// unpublished one classifies cross-branch-local-only and blocks
	// before divergence is consulted (ADR-0041).
	Collisions map[string]bool
	// HasRemoteTrackingRefs is t.HasRemoteTrackingRefs verbatim: whether
	// this repository can express publication at all. classifyBodyToken
	// escalates a local-only hit to error severity only when it can
	// (ADR-0041).
	HasRemoteTrackingRefs bool
	// AsWritten is the id set spelled exactly as the tree stores it —
	// no canonicalization on either side — across the same populations
	// ByID and Trunk cover. It is what lets narrowCitation ask whether a
	// narrow token names an entity that is itself narrow (G-0518). The
	// entity loader keeps `id:` verbatim, which is what makes the
	// question answerable at all: every other index in this struct has
	// already canonicalized the width away.
	AsWritten map[string]bool
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
		ByID:      make(map[string]*entity.Entity, len(t.Entities)+len(t.Stubs)),
		AsWritten: make(map[string]bool, len(t.Entities)+len(t.Stubs)),
	}
	for _, e := range t.Entities {
		idx.AsWritten[e.ID] = true
		key := entity.Canonicalize(e.ID)
		if _, exists := idx.ByID[key]; exists {
			continue
		}
		idx.ByID[key] = e
	}
	for _, e := range t.Stubs {
		idx.AsWritten[e.ID] = true
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
			idx.AsWritten[tid.ID] = true
		}
	}
	idx.CrossBranch = crossBranchIndex(t)
	idx.Collisions = t.CrossBranchCollisions
	idx.HasRemoteTrackingRefs = t.HasRemoteTrackingRefs
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
// Non-prose content is masked (not stripped) via proseMask before
// scanning, so byte offsets in the input remain stable across the
// masking step. Finding.Line is set to the 1-based line number within
// body where the matched token starts; callers that want
// file-relative Line (the bodyProseID tree-walk rule) add the body's
// start-of-file line offset themselves.
//
// The idx parameter is the resolution index from BodyProseIDIndex;
// callers that scan multiple bodies should build it once and reuse.
func ScanBodyProseID(body []byte, entityID, path string, idx BodyProseIndex) []Finding {
	masked := proseMask(body)

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
			Severity: bodyProseIDSeverity(subcode),
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

// proseMask returns a same-length copy of body in which every byte
// CommonMark does not render as prose text is replaced with a space
// (newlines preserved, so line-number resolution downstream stays
// exact). The scanner then runs against prose only: tokens inside ANY
// code construct — inline code spans of any backtick count, fenced
// blocks (``` and ~~~), indented code blocks — and inside non-prose
// link carriers (destinations, titles, reference-link definitions,
// autolinks) are exempt by construction rather than by per-shape
// regex (G-0240). Link/image LABEL text is prose and stays scanned.
//
// Two deliberate inclusions keep the scan surface equal to the
// pre-G-0240 behavior outside the fixed shapes: HTML blocks and
// inline raw HTML are copied through as prose, because CommonMark
// passes raw HTML to the renderer verbatim — its text content is
// user-visible (`<p>M-foo</p>` displays M-foo). The pre-existing
// edge-case test pins this.
//
// CommonMark quirk worth naming: an unclosed fence runs to the end of
// the document, so everything after a typo'd ``` opener is code and
// exempt. That matches what a rendered page shows — the masking
// failure mode is "the body LOOKS wrong when rendered", not a silent
// divergence between the masker and CommonMark semantics.
func proseMask(body []byte) string { return maskFor(body, false) }

// proseAndCodeMask is proseMask widened to also copy code constructs —
// inline code spans and fenced/indented code blocks. Non-prose link
// carriers stay blanked, so the doc-link carve-out is unaffected.
//
// The shipped-surface rule uses this wider mask because a real entity id
// in a command example is a citation like any other: it ships to consumer
// repos and rots there exactly as one in prose does. body-prose-id keeps
// the narrower mask, where a backticked id-shape is how an entity body
// legitimately discusses id syntax. The two masks are opposite answers to
// "is code content in scope", which is why they are distinct entry points
// onto one walker rather than one mask serving both.
func proseAndCodeMask(body []byte) string { return maskFor(body, true) }

// maskFor implements both masks. includeCode selects whether code
// constructs are copied through as scannable content or blanked.
func maskFor(body []byte, includeCode bool) string {
	masked := make([]byte, len(body))
	for i, b := range body {
		if b == '\n' {
			masked[i] = '\n'
		} else {
			masked[i] = ' '
		}
	}
	copySeg := func(seg text.Segment) {
		if seg.Start < 0 || seg.Stop > len(body) || seg.Start >= seg.Stop {
			return
		}
		copy(masked[seg.Start:seg.Stop], body[seg.Start:seg.Stop])
	}
	doc := proseMaskEngine.Parser().Parse(text.NewReader(body))
	// The walker never returns an error: the callback below has no
	// error path.
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.CodeSpan:
			// Children are Text nodes carrying the span's content. When
			// code is out of scope, skipping them leaves the span masked;
			// when it is in scope, the walk continues and the *ast.Text
			// case below copies them like any prose text.
			if !includeCode {
				return ast.WalkSkipChildren, nil
			}
		case *ast.FencedCodeBlock:
			// Block content lives in Lines(), never in Text children, so
			// it stays masked by default and needs an explicit copy to
			// come into scope. Lines() spans the content only — the fence
			// markers and info string are outside it.
			if includeCode {
				for i := 0; i < v.Lines().Len(); i++ {
					copySeg(v.Lines().At(i))
				}
			}
		case *ast.CodeBlock:
			// Indented code block — a distinct goldmark type from
			// FencedCodeBlock, so it needs its own arm.
			if includeCode {
				for i := 0; i < v.Lines().Len(); i++ {
					copySeg(v.Lines().At(i))
				}
			}
		case *ast.Text:
			copySeg(v.Segment)
		case *ast.HTMLBlock:
			for i := 0; i < v.Lines().Len(); i++ {
				copySeg(v.Lines().At(i))
			}
			if v.HasClosure() {
				copySeg(v.ClosureLine)
			}
		case *ast.RawHTML:
			for i := 0; i < v.Segments.Len(); i++ {
				copySeg(v.Segments.At(i))
			}
		}
		return ast.WalkContinue, nil
	})
	return string(masked)
}

// classifyBodyToken returns the finding subcode and detail message for
// a candidate token, or ("", "") if the token resolves cleanly.
//
// Resolution order per tier: the working-tree index (idx.ByID) is
// authoritative when it has the id — a locally-visible milestone with
// a missing AC fires unresolved-ac even if the id also appears on
// trunk. The trunk tier (idx.Trunk) is consulted only on a ByID miss
// (G-0241): a strict-form token known on trunk carries no resolution
// defect. For a composite token whose parent is trunk-only, the AC
// position cannot be validated without the parent's file, so the
// position goes unjudged — refusing would re-create the verb-time
// refusal G-0241 fixes, and the tree-walking rule judges it once the
// file is visible (post rebase/merge). Malformed-shape tokens never
// reach the trunk tier: trunk ids are strict-form by construction, so
// trunk membership cannot launder a malformed token.
//
// Resolving on the working-tree or trunk tier settles WHICH entity the
// token names but not whether it is spelled the way that entity is
// stored, so both hand off to narrowCitation (G-0518) rather than
// returning silent. The cross-branch tier does not, and the asymmetry
// is the one ADR-0030 and ADR-0041 already draw: trunk is
// authoritative, a sibling branch is provisional. A width verdict
// against a spelling that may never land would fire an error over an
// entity the operator cannot reach, and the citation is visible on
// that tier anyway through its own cross-branch- subcode. Once the
// branch merges, the entity is in the working tree and the width is
// judged like any other — the same deferral the composite AC position
// takes on a trunk-only parent.
func classifyBodyToken(tok string, idx BodyProseIndex) (subcode, msg string) {
	if strictCompositeIDPattern.MatchString(tok) {
		parent, sub, _ := entity.ParseCompositeID(tok)
		canonParent := entity.Canonicalize(parent)
		parentEntity, ok := idx.ByID[canonParent]
		if !ok && !idx.Trunk[canonParent] {
			return "unresolved-milestone", fmt.Sprintf("composite id %q whose parent %q is not allocated", tok, parent)
		}
		// Only the parent segment carries a width claim — the AC segment
		// is a single digit by grammar. Judged ahead of the AC position
		// because it is settled the moment the parent resolves, on either
		// tier; a token carrying both defects reports the second one on
		// the run after this is fixed.
		if code, msg := narrowCitation(tok, parent, idx); code != "" {
			return code, msg
		}
		if !ok {
			// Parent visible on trunk only: its acs[] is not in hand, so
			// the AC position cannot be judged here.
			return "", ""
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
		if _, inTree := idx.ByID[canon]; inTree || idx.Trunk[canon] {
			return narrowCitation(tok, tok, idx)
		}
		// M-0259/AC-2: a miss against both ByID and the (silent) Trunk
		// tier consults the cross-branch view before hard-failing
		// (ADR-0030). Unlike Trunk, a hit here is visible: it fires a
		// cross-branch subcode rather than resolving silently, since a
		// sibling branch — unlike trunk — is provisional.
		if hits, known := idx.CrossBranch[canon]; known {
			// ADR-0041: hits confined to local branch refs name an
			// entity reachable from this working copy alone, which
			// blocks. Ordered ahead of the collision branch for the
			// reason spelled out on refsResolve's mirror in check.go.
			if idx.HasRemoteTrackingRefs && !trunk.RemoteVisible(hits) {
				return "cross-branch-local-only", fmt.Sprintf("id %q known only on unpublished local refs (%s) — the reference resolves on this machine and nowhere else", tok, joinRefNames(hits))
			}
			// Non-blocking (D-0036): see the identical rationale on
			// refsResolve's collision branch in check.go.
			if idx.Collisions[canon] {
				return "cross-branch-collision", fmt.Sprintf("id %q has diverging content across %s (may be an in-flight edit on one of the branches, or a genuine duplicate-mint collision — compare manually)", tok, joinRefNames(hits))
			}
			return "cross-branch-pending", fmt.Sprintf("id %q known only on %s (not yet merged into this branch)", tok, joinRefNames(hits))
		}
		return "unresolved", fmt.Sprintf("unknown id %q (no entity allocated at this id)", tok)
	}
	return "malformed-shape", fmt.Sprintf("id-shaped token %q that does not match the kind's strict id pattern (wrap in backticks if discussing id syntax)", tok)
}

// narrowCitation reports the narrow-width subcode for a token that has
// already resolved, or ("", "") when its spelling is legitimate. bareID
// is the segment carrying the width — the token itself for a bare id,
// the parent for a composite.
//
// The rule is reference-shaped rather than width-shaped (G-0518): what
// fires is a token that resolves ONLY after canonicalization. Narrow
// read tolerance is permanent, so a repo that archived entities before
// canonical width was adopted holds genuinely-narrow ids under
// `<kind>/archive/` forever, and a body citing one is correct as
// written — idx.AsWritten is what keeps it silent. The converse case is
// held by the width test: that same archived entity cited at canonical
// width is correct too, because canonical is what every aiwf surface
// emits, and only a token narrower than what it resolves to fires.
//
// The two facts together mean a tree whose ids are uniformly narrow
// never fires this at all — every citation matches an entity spelled
// the same way. Such a tree is already reported by
// entity-id-narrow-width, which judges the entities themselves.
func narrowCitation(tok, bareID string, idx BodyProseIndex) (subcode, msg string) {
	canon := entity.Canonicalize(tok)
	if canon == tok || idx.AsWritten[bareID] {
		return "", ""
	}
	return "narrow-width", fmt.Sprintf("id %q below canonical width — write %s, the entity it resolves to", tok, canon)
}

// bodyProseIDSeverity maps a classifyBodyToken subcode to its finding
// severity. Every subcode is a hard, blocking error except three, which
// are visible but non-blocking warnings.
//
// cross-branch-pending and cross-branch-collision (M-0259/AC-2/AC-3,
// ADR-0030, D-0036): the former because the id is published and merely
// unmerged; the latter because divergent content is ambiguous between a
// genuine duplicate-mint collision and an ordinary same-entity edit on
// an unmerged sibling branch — the actual duplicate-mint case is still
// caught, just later, by the pre-existing blocking
// ids-unique/trunk-collision check.
//
// Those two are named rather than matched on the cross-branch- prefix
// they share, so membership in that family does not by itself confer
// non-blocking severity: cross-branch-local-only is an error, since a
// reference resolvable from no published ref makes the tree valid on
// one machine only (ADR-0041).
//
// narrow-width (G-0518) is the third, and takes docIDWidth's posture
// for the reason that rule states: a citation below canonical width is
// a real defect, but blocking one falls on prose the operator did not
// write and is not editing. What makes blocking cost more here than at
// the push is the verb layer. entityIDNarrowWidth reaches verbs only
// through projectionFindings, which reports what a change INTRODUCES,
// so a pre-existing narrow entity never blocks a write. ScanBodyProseID
// has no such diff — it scans the whole body — so at error severity a
// pre-existing narrow citation would refuse edit-body, import and
// reallocate on the entity carrying it. reallocate is the sharpest,
// since it rewrites every active body and refuses on any error: an
// id-collision fix, which is remedial work under time pressure, could
// be declined over a citation it never touched. A warning keeps the
// finding visible on every check while leaving those verbs free.
//
// Unlike docIDWidth this has no strictness knob, because none is asked
// for yet — add one when a consumer wants to block on it, not before.
func bodyProseIDSeverity(subcode string) Severity {
	switch subcode {
	case "cross-branch-pending", "cross-branch-collision", "narrow-width":
		return SeverityWarning
	default:
		return SeverityError
	}
}
