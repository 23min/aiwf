package check

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// Citation names one record whose body mentions a given entity id, and
// where in that record the first mention sits.
type Citation struct {
	// ID is the citing entity's id.
	ID string
	// Path is the citing entity's repo-relative path.
	Path string
	// Line is the file-relative line of the first mention.
	Line int
}

// CitersOf returns the still-live records whose bodies mention id,
// ordered by id.
//
// This answers a question rather than judging one: it reports that a
// record names the entity, not that anything the record says about it
// is wrong. Whether a mention is a premise that a closure falsifies or
// a past-tense sentence that closing makes *more* accurate is a
// reading, and no rule here attempts it — the caller hands the list to
// whoever is closing the entity, who is the one person holding the
// context to tell those apart.
//
// Three exclusions, each because the record could not be acted on:
// archived files, records already at a terminal status (a closed record
// citing a closed record is history, and editing it would make it less
// true rather than more), and the named entity's own body.
//
// Mentions inside code spans and fenced blocks count. body-prose-id
// masks those out, because there a backticked id-shape is how a body
// legitimately discusses id syntax rather than naming an entity. The
// question here is the opposite one — an id in a command example names
// the same entity a sentence does — so this walks the wider mask, the
// one the shipped-surface rule uses for the same reason. Over-inclusion
// costs a glance; a citation missed at the only moment anyone is
// looking costs the whole notice.
func CitersOf(t *tree.Tree, id string) []Citation {
	target := entity.Canonicalize(id)
	var out []Citation
	for _, e := range t.Entities {
		if entity.Canonicalize(e.ID) == target {
			continue
		}
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		if entity.IsTerminal(e.Kind, e.Status) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(t.Root, e.Path))
		if err != nil { //coverage:ignore the loader read this same path to build the entity moments earlier; a failure here needs the file to vanish or lose permissions mid-walk.
			continue
		}
		_, body, ok := entity.Split(raw)
		if !ok { //coverage:ignore a file without parseable frontmatter never becomes a t.Entities member — it loads as a stub or a load-error finding instead.
			continue
		}
		if line, found := firstMention(body, target); found {
			// Body-relative to file-relative: the body is a suffix of
			// the file, so the frontmatter is everything ahead of it and
			// its newlines are not in the body slice. Taking the offset
			// by length rather than by searching for the body keeps this
			// exact where a frontmatter block quotes the body verbatim.
			line += bytes.Count(raw[:len(raw)-len(body)], []byte{'\n'})
			out = append(out, Citation{ID: e.ID, Path: e.Path, Line: line})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// firstMention reports the body-relative line of the first id-shaped
// token in body that names target.
//
// A composite token names its parent as well as itself: a body resting
// on `M-NNNN/AC-N` rests on the milestone that owns it, so closing the
// milestone bears on that sentence. The reverse does not hold — closing
// one criterion says nothing about a body citing the milestone — so the
// widening runs in one direction only.
func firstMention(body []byte, target string) (int, bool) {
	masked := proseAndCodeMask(body)
	for _, m := range idTokenPattern.FindAllStringIndex(masked, -1) {
		if !names(masked[m[0]:m[1]], target) {
			continue
		}
		return 1 + bytes.Count(body[:m[0]], []byte{'\n'}), true
	}
	return 0, false
}

// names reports whether the id-shaped token refers to target, treating a
// composite token as naming its parent.
func names(tok, target string) bool {
	if entity.Canonicalize(tok) == target {
		return true
	}
	parent, _, ok := entity.ParseCompositeID(tok)
	return ok && entity.Canonicalize(parent) == target
}
