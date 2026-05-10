package verb

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// Rewidth sweeps a consumer's active planning tree from narrow legacy
// id widths (E-NN, M-NNN, G-NNN, D-NNN, C-NNN) to canonical 4-digit
// width (E-NNNN, M-NNNN, G-NNNN, D-NNNN, C-NNNN). Per ADR-0008:
//
//   - Default is dry-run: the verb computes a Plan and the caller
//     prints planned ops without applying. `--apply` (caller flag)
//     causes the dispatcher to run verb.Apply on the Plan.
//   - Single commit per --apply per kernel principle #7. Trailer is
//     `aiwf-verb: rewidth`; no `aiwf-entity:` trailer (multi-entity
//     sweep, same shape as `aiwf archive`).
//   - Active-tree only. Files under `<kind>/archive/` are skipped
//     entirely per ADR-0004's forget-by-default principle.
//   - Idempotent. An already-canonical or empty tree returns a NoOp
//     Result; the caller prints "no changes needed" and exits 0.
//
// The verb walks each kind's active directory in a fixed sequence
// (epic, milestone, gap, decision, contract, adr) and within a kind
// iterates in alphabetical order by current filename. Determinism is
// load-bearing: a second invocation on the same tree visits files in
// the same order and produces zero ops.
//
// Three reference patterns are rewritten in active-tree markdown
// bodies:
//
//   - Bare id mentions in prose (`E-22` → `E-0022`). Word-boundary
//     guarded so `E-220` doesn't match.
//   - Composite ids (`M-22/AC-1` → `M-0022/AC-1`).
//   - Markdown links to active-tree paths (`(work/epics/E-22-foo)`).
//     Links targeting `<kind>/archive/...` are excluded by design.
//
// Code fences (triple-backtick) and inline-code spans (single-backtick)
// are excluded from rewriting — content inside them stays as-is.
//
// The F-prefix is included in the regex by spec (planned 7th kind from
// the §07 TDD architecture proposal). No F entities exist today, so
// it's a forward-compatible no-op for current consumers.
//
// Rewidth does not call check.Run on a projected tree the way other
// verbs do: the operation is purely structural (rename + body rewrite)
// and `aiwf check` is the chokepoint for post-migration validation.
// The pre-push hook will run check after the user pushes the rewidth
// commit; spurious mid-verb check noise from a tree mid-rename is not
// what we want.
func Rewidth(ctx context.Context, root, actor string) (*Result, error) {
	_ = ctx
	plan, err := planRewidth(root)
	if err != nil { //coverage:ignore filesystem read errors at the verb's top level — the underlying os.ReadDir/os.ReadFile paths bubble up here, but tempdir-based tests don't reproduce real ENOENT/EACCES races without invasive setup
		return nil, err
	}
	if plan == nil {
		return &Result{NoOp: true, NoOpMessage: "aiwf rewidth: no changes needed (active tree is already at canonical width)"}, nil
	}
	plan.Trailers = []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "rewidth"},
		{Key: gitops.TrailerActor, Value: actor},
	}
	return &Result{Plan: plan}, nil
}

// planRewidth computes the rename + body-rewrite ops over the active
// tree. Returns nil when nothing needs to change (idempotent path);
// otherwise returns a Plan ready for verb.Apply.
func planRewidth(root string) (*Plan, error) {
	renames, err := planRewidthRenames(root)
	if err != nil { //coverage:ignore filesystem read errors propagate from planRewidthRenames — covered by the planRewidthRenames-side defensive ignores
		return nil, err
	}

	// Per-file body rewrites are computed AFTER renames so the rewrite
	// reads each file once at its post-move path. We sort the active
	// tree, then for every active markdown file compute (orig content,
	// new content) and emit an OpWrite when they differ. Path is the
	// post-move path so writes land where the rename ops put them.
	rewrites, err := planRewidthRewrites(root, renames)
	if err != nil { //coverage:ignore filesystem read errors propagate from planRewidthRewrites — covered by the planRewidthRewrites-side defensive ignores
		return nil, err
	}

	if len(renames) == 0 && len(rewrites) == 0 {
		return nil, nil
	}

	ops := make([]FileOp, 0, len(renames)+len(rewrites))
	for _, r := range renames {
		ops = append(ops, FileOp{Type: OpMove, Path: r.from, NewPath: r.to})
	}
	for _, w := range rewrites {
		ops = append(ops, FileOp{Type: OpWrite, Path: w.path, Content: w.content})
	}

	subject := fmt.Sprintf("aiwf rewidth: %d rename(s), %d body rewrite(s)", len(renames), len(rewrites))
	return &Plan{
		Subject: subject,
		Body:    rewidthCommitBody(renames, rewrites),
		Ops:     ops,
	}, nil
}

// renamePair is the (from, to) of a single git mv that the verb plans.
type renamePair struct {
	from string
	to   string
}

// rewidthRewrite is one OpWrite — body content has been rewritten and
// must land at path (which is the post-move path when path was
// affected by a rename).
type rewidthRewrite struct {
	path    string
	content []byte
}

// kindActiveLayout maps each kind to (a) its containing directory
// under `work/` or `docs/`, (b) whether the entity is dir-shaped
// (epic, contract) or file-shaped, and (c) the filename prefix. The
// walk-order in the surrounding caller (planRewidthRenames) iterates
// kinds in this fixed sequence.
type kindActiveLayout struct {
	kind     entity.Kind
	rootDir  string // repo-relative dir, forward-slash
	dirShape bool   // true when entity lives in its own directory (epic, contract)
	prefix   string // e.g. "E-", "M-", "G-"
}

// activeKindLayouts returns the fixed-order sequence of kinds plus
// their layout metadata. This is the canonical walk order — both for
// determinism and for the trailer-test assertion (`epic, milestone,
// gap, decision, contract, adr`). ADR is last per the spec.
func activeKindLayouts() []kindActiveLayout {
	return []kindActiveLayout{
		{kind: entity.KindEpic, rootDir: "work/epics", dirShape: true, prefix: "E-"},
		{kind: entity.KindMilestone, rootDir: "work/epics", dirShape: false, prefix: "M-"},
		{kind: entity.KindGap, rootDir: "work/gaps", dirShape: false, prefix: "G-"},
		{kind: entity.KindDecision, rootDir: "work/decisions", dirShape: false, prefix: "D-"},
		{kind: entity.KindContract, rootDir: "work/contracts", dirShape: true, prefix: "C-"},
		{kind: entity.KindADR, rootDir: "docs/adr", dirShape: false, prefix: "ADR-"},
	}
}

// planRewidthRenames computes the list of `git mv` ops that rename
// narrow-width entity files/dirs to canonical width. Skips archive
// entries (`<kind>/archive/...`), missing kind directories, and any
// filename whose id portion is already canonical.
//
// Composite handling: epic dirs are processed first, so when an epic
// renames the milestone files inside come along under the new dir.
// The subsequent milestone-pass iterates the post-rename dir and
// renames any milestone whose own filename id is still narrow.
func planRewidthRenames(root string) ([]renamePair, error) {
	layouts := activeKindLayouts()
	// Use a path-mapping so the milestone pass can resolve a parent
	// epic's post-rename dir; the milestone's `from` path uses the
	// renamed parent dir, not the original.
	dirRenames := map[string]string{} // forward-slash, repo-relative
	var out []renamePair

	for _, layout := range layouts {
		rootDir := filepath.Join(root, filepath.FromSlash(layout.rootDir))
		entries, err := readActiveDirSorted(rootDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err //coverage:ignore non-ENOENT readdir failure (EACCES, EIO) — defensive; tempdir-based tests can't reproduce without invasive setup
		}
		switch layout.kind {
		case entity.KindEpic, entity.KindContract:
			for _, name := range entries {
				if name == "archive" {
					continue
				}
				// Skip non-directory entries (a stray README.md in
				// work/epics/ is not an epic; ignore it).
				if !isDir(filepath.Join(rootDir, name)) {
					continue
				}
				newName, ok := canonicalizeFilename(name, layout.prefix, true)
				if !ok {
					continue
				}
				if newName == name {
					continue
				}
				fromRel := filepath.ToSlash(filepath.Join(layout.rootDir, name))
				toRel := filepath.ToSlash(filepath.Join(layout.rootDir, newName))
				out = append(out, renamePair{from: fromRel, to: toRel})
				dirRenames[fromRel] = toRel
			}
		case entity.KindMilestone:
			// Milestones live inside epic dirs. Walk every (post-rename)
			// epic dir, list its milestone files, rename narrow ones in
			// place. Use applyDirRename to translate the original
			// relative path to its post-move equivalent for the
			// emitted FileOp.
			for _, epicName := range entries {
				if epicName == "archive" {
					continue
				}
				epicDirRel := filepath.ToSlash(filepath.Join(layout.rootDir, epicName))
				epicAbs := filepath.Join(root, filepath.FromSlash(epicDirRel))
				// Stray non-directory entries in work/epics/ are not
				// epic dirs; nothing to walk inside.
				if !isDir(epicAbs) {
					continue
				}
				milestones, err := readActiveDirSorted(epicAbs)
				if err != nil { //coverage:ignore filesystem-error path; the readActiveDirSorted call only errors on race conditions tempdir-based tests can't reliably reproduce
					if os.IsNotExist(err) {
						continue
					}
					return nil, err
				}
				for _, mName := range milestones {
					// Filter to milestone files only. `epic.md` is
					// excluded by the prefix check (no `M-` prefix);
					// non-milestone hygiene files are too. The
					// suffix check screens `M-`-prefixed entries
					// without `.md` extension (rare but worth a
					// guard so we don't try to git mv a non-md
					// fixture).
					if !strings.HasPrefix(mName, layout.prefix) {
						continue
					}
					if !strings.HasSuffix(mName, ".md") {
						continue
					}
					newName, ok := canonicalizeFilename(mName, layout.prefix, false)
					if !ok {
						continue
					}
					if newName == mName {
						continue
					}
					// Resolve the path the milestone file will live at
					// AFTER the parent epic's rename has been applied.
					// `from` in the OpMove is the post-epic-rename path
					// (because Apply runs all moves in order; the epic
					// move runs first, so by the time the milestone move
					// runs, the file is at the new dir).
					postEpicDir := applyDirRename(epicDirRel, dirRenames)
					fromRel := filepath.ToSlash(filepath.Join(postEpicDir, mName))
					toRel := filepath.ToSlash(filepath.Join(postEpicDir, newName))
					out = append(out, renamePair{from: fromRel, to: toRel})
				}
			}
		default:
			// Gap / decision / ADR — flat directory of .md files.
			for _, name := range entries {
				if name == "archive" {
					continue
				}
				if !strings.HasSuffix(name, ".md") {
					continue
				}
				if !strings.HasPrefix(name, layout.prefix) {
					continue
				}
				newName, ok := canonicalizeFilename(name, layout.prefix, false)
				if !ok {
					continue
				}
				if newName == name {
					continue
				}
				fromRel := filepath.ToSlash(filepath.Join(layout.rootDir, name))
				toRel := filepath.ToSlash(filepath.Join(layout.rootDir, newName))
				out = append(out, renamePair{from: fromRel, to: toRel})
			}
		}
	}
	return out, nil
}

// applyDirRename returns dst when src exactly matches a key in
// renames; otherwise returns src unchanged. Used to chase
// epic-dir-rename when computing the milestone OpMove's source path.
func applyDirRename(src string, renames map[string]string) string {
	if dst, ok := renames[src]; ok {
		return dst
	}
	return src
}

// isDir reports whether path is a directory. Errors collapse to
// false (the caller treats "can't tell" as "skip"). Used by the
// dir-shape kinds (epic, contract) to filter out stray non-dir
// entries (a README.md, a .DS_Store) that share the parent dir with
// genuine entity directories.
func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil { //coverage:ignore stat failures (race: file deleted between readdir and stat) — collapse to non-dir; defensive
		return false
	}
	return info.IsDir()
}

// readActiveDirSorted reads dir's immediate child entries (not
// recursive) and returns the names in alphabetical order. Skips
// `.DS_Store` and other hidden files but keeps `archive` (the caller
// filters that explicitly so the skip-decision is visible at the call
// site). Returns os.ErrNotExist when dir doesn't exist; the caller
// translates that to "kind dir absent — skip this kind."
func readActiveDirSorted(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ents))
	for _, e := range ents {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

// canonicalizeFilename takes a filename of the form
// `<prefix><digits>-<slug>` (or just `<prefix><digits>` for dir-shape
// composites) and returns its canonical-width counterpart. Returns
// (newName, true) when the name parses as an entity filename of the
// expected prefix; (newName, false) when the name doesn't parse.
//
// `dirShape=true` skips the `.md` requirement (epic dirs / contract
// dirs have no extension); otherwise the suffix `.md` is required.
//
// When the id portion is already at canonical width or wider, returns
// the input unchanged (and ok=true).
func canonicalizeFilename(name, prefix string, dirShape bool) (string, bool) {
	stripped := name
	if !dirShape {
		if !strings.HasSuffix(name, ".md") { //coverage:ignore current callers (gap/decision/adr/milestone walks) all pre-filter on .md before calling; defensive to keep canonicalizeFilename usable from a future caller that doesn't
			return "", false
		}
		stripped = strings.TrimSuffix(name, ".md")
	}
	if !strings.HasPrefix(stripped, prefix) { //coverage:ignore current callers (file-shape walks) pre-filter on prefix; defensive
		return "", false
	}
	rest := stripped[len(prefix):]
	// Find the slug separator: the first `-` after at least one digit.
	// `<digits>` may be followed by `-<slug>` or be alone (dir-shape
	// can be just `E-22` with no slug — though in practice every epic
	// has a slug).
	digitsEnd := 0
	for digitsEnd < len(rest) && rest[digitsEnd] >= '0' && rest[digitsEnd] <= '9' {
		digitsEnd++
	}
	if digitsEnd == 0 {
		return "", false
	}
	digits := rest[:digitsEnd]
	tail := rest[digitsEnd:]
	// Pad to canonical width. Use padToCanonical (not
	// entity.Canonicalize) because narrow legacy ids below the per-
	// kind grammar minimum (e.g., `M-22`, `G-9`) would fail the
	// grammar check and pass through unchanged — and the verb's
	// whole job is to widen exactly those.
	prefixSans := strings.TrimSuffix(prefix, "-")
	canonical := padToCanonical(prefixSans, digits) + tail
	if !dirShape {
		canonical += ".md"
	}
	return canonical, true
}

// planRewidthRewrites computes the body-content rewrites for every
// active markdown file. Reads each file (at its post-move path),
// rewrites the three id-mention patterns plus markdown links to
// active-tree paths, and emits an OpWrite when content differs.
//
// Skips files in `<kind>/archive/...`. Reads from disk via the
// post-move path so a file affected by a rename has its rewrite land
// at its new location (Apply runs moves first, then writes).
func planRewidthRewrites(root string, renames []renamePair) ([]rewidthRewrite, error) {
	// Build a map of pre-rename → post-rename paths so files moved by
	// renames are read from their original disk location but written
	// to the new path.
	preToPost := map[string]string{}
	for _, r := range renames {
		preToPost[r.from] = r.to
	}

	files, err := walkActiveMarkdown(root)
	if err != nil { //coverage:ignore filesystem walk errors propagate; tempdir tests can't reproduce ENOENT/EACCES races
		return nil, err
	}

	var out []rewidthRewrite
	for _, fileRel := range files {
		// `fileRel` is the on-disk pre-rename path (the walk happens
		// before any rename is applied). Resolve its post-rename path.
		postRel := mapAfterRenames(fileRel, preToPost, renames)
		full := filepath.Join(root, filepath.FromSlash(fileRel))
		content, err := os.ReadFile(full)
		if err != nil { //coverage:ignore race: file disappeared between walk and read; defensive
			return nil, fmt.Errorf("reading %s: %w", fileRel, err)
		}
		newContent := rewriteRewidthBody(content)
		if !equalBytes(newContent, content) {
			out = append(out, rewidthRewrite{path: postRel, content: newContent})
		}
	}
	return out, nil
}

// mapAfterRenames returns the post-rename path for a file's
// pre-rename location. Three cases stack:
//
//  1. The file itself was renamed in place (preToPost hit on the
//     pre-rename path directly).
//  2. The file lives inside a directory that was renamed — the
//     parent dir prefix is substituted from the longest matching
//     dir-rename `from`.
//  3. The file lives inside a renamed directory AND was itself
//     renamed (the renamePair's `from` is the post-dir-rename path,
//     because Apply runs moves in walk order). The chain is: apply
//     dir-rename first to get the path the milestone-rename `from`
//     references, then apply the file-rename.
//
// fileRel is the pre-rename on-disk path; preToPost keys are the
// renamePair `from` values (post-dir-rename for milestones). The
// function tries case 1 first against fileRel; failing that, applies
// case 2 to get the post-dir-rename path, then re-checks preToPost
// against that intermediate path; failing that, returns the
// dir-rename-only path.
func mapAfterRenames(fileRel string, preToPost map[string]string, renames []renamePair) string {
	// Case 1: direct file rename keyed on the pre-rename path.
	if dst, ok := preToPost[fileRel]; ok {
		return dst
	}
	// Case 2: parent-dir rename. Find the longest dir-rename whose
	// `from` is a prefix of fileRel.
	best := ""
	bestTo := ""
	for _, r := range renames {
		if !strings.HasPrefix(fileRel, r.from+"/") {
			continue
		}
		if len(r.from) > len(best) {
			best = r.from
			bestTo = r.to
		}
	}
	if best == "" {
		return fileRel
	}
	postDir := bestTo + fileRel[len(best):]
	// Case 3: after the dir-rename, is there a per-file rename keyed
	// on the post-dir path? Milestones inside renamed epic dirs hit
	// this branch.
	if dst, ok := preToPost[postDir]; ok {
		return dst
	}
	return postDir
}

// walkActiveMarkdown returns every active-tree markdown file under
// the kinds rewidth touches, in deterministic order (sorted at each
// directory level). Excludes archive subtrees. Returns repo-relative
// forward-slash paths.
func walkActiveMarkdown(root string) ([]string, error) {
	roots := []string{
		"work/epics",
		"work/gaps",
		"work/decisions",
		"work/contracts",
		"docs/adr",
	}
	var out []string
	for _, base := range roots {
		baseAbs := filepath.Join(root, filepath.FromSlash(base))
		err := filepath.Walk(baseAbs, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return filepath.SkipDir //coverage:ignore race: subdir disappeared during walk; defensive
				}
				return err //coverage:ignore non-ENOENT walk error (EACCES); defensive
			}
			if info.IsDir() {
				if info.Name() == "archive" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(info.Name(), ".md") {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil { //coverage:ignore filepath.Rel only fails on malformed inputs; both args come from valid paths above
				return relErr
			}
			out = append(out, filepath.ToSlash(rel))
			return nil
		})
		if err != nil { //coverage:ignore filepath.Walk only returns non-nil err when the inner callback returned err that wasn't SkipDir; tempdir tests can't reproduce
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

// equalBytes is a tiny equality helper. Pulls in the bytes stdlib so
// the comparison shape stays canonical; rewriteRewidthBody almost
// always changes length when it changes content (E-22 → E-0022 adds
// two bytes), so the equal-length-different-content branch is rare
// in practice but worth handling correctly.
func equalBytes(a, b []byte) bool {
	return bytes.Equal(a, b)
}

// Body-rewrite engine. Three patterns:
//
//   1. Bare id mentions:    \b[EMGDCF]-[0-9]{1,3}\b
//   2. Composite ids:       \bM-[0-9]{1,3}/AC-[0-9]+\b
//   3. Markdown links:      \(work/<kind>/[EMGDCF]-[0-9]{1,3}-<slug>(?:\.md)?\)
//      excluding `work/<kind>/archive/...`
//
// Code fences and inline-code spans are excluded from rewriting. The
// F prefix is included for forward compatibility with the planned 7th
// kind (ADR-0003 amendment / §07 TDD architecture proposal); current
// trees have no F entities so it's a no-op.
//
// Trailing-digit guards are critical: \b in Go's regexp matches
// transitions between word and non-word chars, so `E-22` matches but
// `E-220` does not (the `2` after `22` is still a word char, no
// boundary). Same logic prevents `E-2200` from matching.

// bareIDPattern matches narrow-width id-form mentions in prose. The
// `(?:[EMGDCF])` non-capturing group covers the six narrow-form
// kinds (ADR is exempt — its grammar was always 4-digit). Width 1-3
// per spec; 4+ digit forms are already canonical and pass through.
var bareIDPattern = regexp.MustCompile(`\b([EMGDCF])-(\d{1,3})\b`)

// compositeIDPattern matches narrow-width composite ids like
// `M-22/AC-1`. The AC suffix is preserved verbatim; only the
// milestone portion is rewritten. M is the only kind that supports
// composite ids in current grammar.
var compositeIDPattern = regexp.MustCompile(`\bM-(\d{1,3})/(AC-\d+)\b`)

// linkPathPattern matches markdown links to active-tree paths (i.e.,
// `(work/<kind>/<id>-<slug>...)`) with a narrow-width id. The
// non-capturing group `(?:archive/)?` is intentionally absent — we do
// NOT want to match archive paths, so we inspect the captured group
// post-match and reject any match whose path includes `/archive/`.
//
// The grammar is loose by design: `<slug>` may include any non-`)`
// characters (kebab text, dots from `.md`, etc.). Anchored by the
// preceding `(` so only proper markdown links are touched, not bare
// inline text mentioning a path.
var linkPathPattern = regexp.MustCompile(`\(work/([a-z]+)/([EMGDCF])-(\d{1,3})(-[^)]*)?\)`)

// rewriteRewidthBody applies the three rewrite patterns to body,
// excluding code fences and inline-code spans. Returns the rewritten
// body. Pure: no I/O. Idempotent: running twice on the same input
// produces the same output as running once.
func rewriteRewidthBody(body []byte) []byte {
	src := string(body)
	out := strings.Builder{}
	out.Grow(len(src))

	lines := strings.Split(src, "\n")
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			out.WriteString(line)
			if i < len(lines)-1 {
				out.WriteByte('\n')
			}
			inFence = !inFence
			continue
		}
		if inFence {
			out.WriteString(line)
			if i < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}
		out.WriteString(rewriteLineOutsideFence(line))
		if i < len(lines)-1 {
			out.WriteByte('\n')
		}
	}
	return []byte(out.String())
}

// rewriteLineOutsideFence applies rewrites to a single line of prose,
// honoring two exclusion regions:
//
//  1. Inline-code spans (` `text` `) — content between backticks is
//     preserved verbatim. Documentation often quotes literal id
//     strings inside backticks for emphasis; rewriting those would
//     erase the quote semantics.
//
//  2. URL-shaped tokens — any contiguous non-whitespace token that
//     contains `://` is a URL; an id in its path is part of the
//     URL's identity, not an entity reference. Per the AC-3 spec,
//     `E-22` inside `https://example.com/E-22-issue` must stay
//     narrow.
//
// The traversal walks the line scanning for backticks (toggle
// in-span) and whitespace boundaries (commit each non-whitespace
// token after deciding whether it's a URL). Code-span content always
// passes through verbatim regardless of URL shape.
func rewriteLineOutsideFence(line string) string {
	var out strings.Builder
	out.Grow(len(line))
	inSpan := false
	var buf strings.Builder
	flushOutside := func() {
		if buf.Len() == 0 {
			return
		}
		out.WriteString(rewriteOutsideChunk(buf.String()))
		buf.Reset()
	}
	for _, r := range line {
		if r == '`' {
			if inSpan {
				// Closing backtick — flush the in-span buffer verbatim.
				out.WriteString(buf.String())
				buf.Reset()
				out.WriteRune(r)
				inSpan = false
				continue
			}
			// Opening backtick — flush the out-of-span buffer with
			// rewriting applied.
			flushOutside()
			out.WriteRune(r)
			inSpan = true
			continue
		}
		buf.WriteRune(r)
	}
	if inSpan {
		// Unterminated span on this line — treat the buffer as in-span
		// (verbatim) per markdown convention (an unmatched backtick is
		// not a code-span open). Conservative: don't rewrite.
		out.WriteString(buf.String())
	} else {
		flushOutside()
	}
	return out.String()
}

// rewriteOutsideChunk applies the three rewrite patterns to a chunk
// of prose. Two regions are preserved verbatim:
//
//   - URL-shaped tokens (any whitespace-delimited run containing
//     `://`).
//   - Markdown link-path references `](path)` where the path is NOT
//     an active-tree path (e.g., `archive/...`, external URLs).
//     Active-tree paths matching `work/<kind>/<id>-...` are
//     rewritten by linkPathPattern; the same regex's failure to
//     match an archive-prefixed path means we must NOT then run the
//     bare-id pattern inside that region — else `G-001` inside
//     `(work/gaps/archive/G-001-foo.md)` would still be canonicalized.
//
// Strategy: split the chunk into two stripes — regions covered by a
// markdown link-path (the parens directly following `]`) versus
// everything else. Inside a link-path region, run only the
// linkPathPattern rewrite (which canonicalizes active-tree paths and
// leaves archive/external paths alone). Outside, run the URL-token
// + composite + bare-id passes.
func rewriteOutsideChunk(chunk string) string {
	var out strings.Builder
	out.Grow(len(chunk))
	regions := splitLinkPathRegions(chunk)
	for _, reg := range regions {
		if reg.inLinkPath {
			// Inside a `](...)` link-path. Apply only the link-path
			// rewrite (which has its own active-vs-archive predicate).
			// Leave the literal as-is otherwise.
			out.WriteString(linkPathPattern.ReplaceAllStringFunc(reg.text, func(match string) string {
				groups := linkPathPattern.FindStringSubmatch(match)
				if len(groups) != 5 {
					return match //coverage:ignore defensive: regex shape pins group count
				}
				kindDir := groups[1]
				if kindDir == "archive" {
					return match //coverage:ignore guard against future grammar drift; current pattern can't reach
				}
				prefix := groups[2]
				digits := groups[3]
				tail := groups[4]
				return "(work/" + kindDir + "/" + padToCanonical(prefix, digits) + tail + ")"
			}))
			continue
		}
		// Outside any link-path region: tokenize by whitespace, skip
		// URL-shaped tokens, run composite + bare-id rewrites on the
		// rest. The composite-id pass is part of rewriteProseChunk;
		// link-path rewrite there is a no-op because no `(` regions
		// reach this branch.
		tokens := tokenizeBySpace(reg.text)
		for _, tok := range tokens {
			if tok.isSpace {
				out.WriteString(tok.text)
				continue
			}
			if strings.Contains(tok.text, "://") {
				// URL-shaped token — preserve verbatim per AC-3.
				out.WriteString(tok.text)
				continue
			}
			out.WriteString(rewriteProseChunk(tok.text))
		}
	}
	return out.String()
}

// linkPathRegion is a contiguous run inside or outside a markdown
// link-path `](...)` literal. inLinkPath=true regions include the
// surrounding `(` and `)` so the linkPathPattern can match them
// directly; outside regions are pure prose.
type linkPathRegion struct {
	text       string
	inLinkPath bool
}

// splitLinkPathRegions walks s and splits it into alternating
// in-link-path and outside-link-path regions. A link-path region
// starts at `](` (immediately after the `]`) and ends at the matching
// `)`. Nesting and escapes are not handled — markdown's link-path
// grammar disallows unescaped `)` inside the path, and the spec's
// inputs don't include escaped link paths.
func splitLinkPathRegions(s string) []linkPathRegion {
	var out []linkPathRegion
	var buf strings.Builder
	i := 0
	for i < len(s) {
		// Look for `](` starting at i.
		idx := strings.Index(s[i:], "](")
		if idx < 0 {
			buf.WriteString(s[i:])
			break
		}
		abs := i + idx
		// Everything up to (but not including) `]` goes into the
		// outside region. We also include the `]` itself in outside,
		// since it's not part of the link-path region.
		buf.WriteString(s[i : abs+1])
		out = append(out, linkPathRegion{text: buf.String(), inLinkPath: false})
		buf.Reset()
		// Now find the matching `)`. Start at the `(` immediately
		// after `]`.
		closeRel := strings.Index(s[abs+2:], ")")
		if closeRel < 0 {
			// Unbalanced — treat the rest of the string as outside,
			// per "conservative: don't rewrite" approach for malformed
			// markdown.
			out = append(out, linkPathRegion{text: s[abs+1:], inLinkPath: false})
			break
		}
		closeAbs := abs + 2 + closeRel
		// link-path region includes `(` and `)`.
		out = append(out, linkPathRegion{text: s[abs+1 : closeAbs+1], inLinkPath: true})
		i = closeAbs + 1
	}
	if buf.Len() > 0 {
		out = append(out, linkPathRegion{text: buf.String(), inLinkPath: false})
	}
	return out
}

// spaceToken is one segment of a chunk: either whitespace or a
// non-whitespace run. Tokenizing this way preserves the original
// spacing in the output so prose round-trips byte-identical.
type spaceToken struct {
	text    string
	isSpace bool
}

// tokenizeBySpace splits s into alternating whitespace and
// non-whitespace tokens. Whitespace runs and non-whitespace runs are
// each single tokens with isSpace set accordingly. Pure: no
// allocations beyond the output slice.
func tokenizeBySpace(s string) []spaceToken {
	if s == "" {
		return nil
	}
	var out []spaceToken
	var buf strings.Builder
	curSpace := isSpaceRune(rune(s[0]))
	for _, r := range s {
		isSp := isSpaceRune(r)
		if isSp != curSpace {
			out = append(out, spaceToken{text: buf.String(), isSpace: curSpace})
			buf.Reset()
			curSpace = isSp
		}
		buf.WriteRune(r)
	}
	if buf.Len() > 0 {
		out = append(out, spaceToken{text: buf.String(), isSpace: curSpace})
	}
	return out
}

// isSpaceRune reports whether r is a Unicode whitespace character.
// Inlined to avoid pulling in unicode for the small ASCII set the
// markdown rewriter cares about.
func isSpaceRune(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// rewriteProseChunk applies the bare-id and composite-id rewrite
// patterns to a chunk of out-of-link prose. Markdown link-paths are
// handled by the caller (rewriteOutsideChunk) which strips them out
// before this function sees them, so the link-path pattern is not
// applied here.
//
// Order: composite first (so `M-22/AC-1` rewrites the milestone
// portion before the bare-id pattern would re-match the same digits),
// then bare ids.
//
// The padder used here is `padToCanonical` rather than
// `entity.Canonicalize` because Canonicalize requires the input to
// match its kind's grammar, and the kind grammars (`^M-\d{3,}$`,
// `^G-\d{3,}$`, etc.) reject narrow legacy ids below the per-kind
// minimum (`M-22` fails milestone grammar's 3-digit minimum). The
// regex pins the input space to `<prefix>-<digits>` already, so we
// can pad directly without re-validating against per-kind grammar.
func rewriteProseChunk(s string) string {
	// 1. Composite ids: `M-22/AC-1` → `M-0022/AC-1`.
	s = compositeIDPattern.ReplaceAllStringFunc(s, func(match string) string {
		groups := compositeIDPattern.FindStringSubmatch(match)
		if len(groups) != 3 {
			return match //coverage:ignore regex returns matched groups when the pattern matches; defensive
		}
		return padToCanonical("M", groups[1]) + "/" + groups[2]
	})

	// 2. Bare ids: `E-22` → `E-0022`. The `\b` boundaries prevent
	//    matching inside longer ids (`E-220`, `E-2200`).
	s = bareIDPattern.ReplaceAllStringFunc(s, func(match string) string {
		groups := bareIDPattern.FindStringSubmatch(match)
		if len(groups) != 3 {
			return match //coverage:ignore defensive: regex shape pins group count
		}
		return padToCanonical(groups[1], groups[2])
	})
	return s
}

// padToCanonical zero-pads digits to entity.CanonicalPad width and
// concatenates `<prefix>-<padded>`. Returns the input verbatim when
// the existing digit count already meets or exceeds the canonical
// width (so canonical inputs are no-ops and idempotent).
//
// Why not call entity.Canonicalize directly: that helper requires the
// id to match the per-kind grammar (`^M-\d{3,}$` etc.), and narrow
// legacy ids below the grammar's per-kind minimum (e.g., `M-22`,
// `G-9`) fail validation and pass through unchanged. The verb is
// migrating exactly those forms, so we operate on the raw `<prefix>-
// <digits>` pair the regex already extracted.
func padToCanonical(prefix, digits string) string {
	if len(digits) >= entityCanonicalPad {
		return prefix + "-" + digits
	}
	// Strip leading zeros (digits is from regex `[0-9]+` so atoi-safe).
	// Then format with the canonical zero-pad width.
	n := 0
	for _, r := range digits {
		n = n*10 + int(r-'0')
	}
	return fmt.Sprintf("%s-%0*d", prefix, entityCanonicalPad, n)
}

// entityCanonicalPad mirrors entity.CanonicalPad. Hoisted to a
// package-private const so the body-rewrite engine doesn't pull in
// the entity package's regex-validated helper for a width-only
// operation.
const entityCanonicalPad = entity.CanonicalPad

// rewidthCommitBody renders the per-kind rename count and rewrite
// count summary for the commit message body. Per ADR-0008, the body
// "lists per-kind rename counts, reference-rewrite counts."
func rewidthCommitBody(renames []renamePair, rewrites []rewidthRewrite) string {
	if len(renames) == 0 && len(rewrites) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Per ADR-0008: canonicalize narrow-width entity ids to 4-digit form.\n")
	sb.WriteString("\n")
	if len(renames) > 0 {
		// Group by kind prefix for the per-kind summary.
		byPrefix := map[string]int{}
		for _, r := range renames {
			byPrefix[detectPrefix(r.from)]++
		}
		sb.WriteString("Renames:\n")
		for _, p := range []string{"E-", "M-", "G-", "D-", "C-", "ADR-"} {
			if n := byPrefix[p]; n > 0 {
				fmt.Fprintf(&sb, "  %s  %d file(s)\n", p, n)
			}
		}
	}
	if len(rewrites) > 0 {
		fmt.Fprintf(&sb, "\nBody rewrites: %d file(s)\n", len(rewrites))
	}
	return sb.String()
}

// detectPrefix returns the entity-prefix portion of a path's
// filename (e.g., `work/epics/E-22-foo/epic.md` → `E-`,
// `work/gaps/G-9-foo.md` → `G-`). Returns empty when no recognizable
// prefix appears in the basename.
//
// detectPrefix is invoked only on rename pairs (which always carry
// an entity-id-shaped basename — `E-22-foo`, `M-77-bar.md`, etc.), so
// the basename's prefix is enough; no parent-walk fallback is needed.
func detectPrefix(path string) string {
	base := filepath.Base(path)
	for _, p := range []string{"ADR-", "E-", "M-", "G-", "D-", "C-"} {
		if strings.HasPrefix(base, p) {
			return p
		}
	}
	return ""
}
