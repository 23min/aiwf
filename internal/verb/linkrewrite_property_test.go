package verb

// linkrewrite_property_test.go — M-0245/AC-3 property tests for
// RewriteLinkDestinations. Two properties, sampled over generated
// tree layouts and move sets:
//
//   - Idempotence: running the primitive twice on a body yields the
//     same output as running it once.
//   - Resolution correctness: every deliberately-rewritten
//     destination resolves — via an independent resolver written
//     fresh in this file, not RewriteLinkDestinations' own
//     resolveLinkDestination — to its move's To path.
//
// Per wf-property-test, these sample the input space; they do not
// prove the properties for all inputs. Determinism: each property
// drives testing/quick with a fixed-seed *rand.Rand, so a green run
// is reproducible.

import (
	"bytes"
	"fmt"
	"math/rand"
	"path"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
)

const (
	propMaxMoves        = 3
	propMaxNoiseLinks   = 3
	propertyRewriteRuns = 1500
)

// propKindRoots is a small, deliberately narrow set of entity root
// dirs so randomly-generated paths collide (and therefore exercise
// real rewrite cases) often enough to be useful.
var propKindRoots = []string{"work/epics", "work/gaps", "work/decisions", "docs/adr"}

// craftedLink is one deliberately-generated link the property test
// tracks by a unique marker label, so it can find the link's
// destination again in the rewritten output without re-parsing
// markdown. rootRelative is the flavor the link was GENERATED with —
// carried explicitly rather than re-derived from the output string,
// so the resolution-correctness property actually exercises flavor
// preservation instead of accepting any string that happens to look
// root-relative (every EntityMove.To in this generator does, by
// construction — re-deriving flavor from shape alone made the
// property vacuous against a real regression: see the M-0245/AC-3
// work log for the caught case).
type craftedLink struct {
	label        string
	moveIdx      int // index into linkRewriteInput.moves this link targets
	rootRelative bool
}

// linkRewriteInput is one generated RewriteLinkDestinations call: a
// linking-file path, a small set of entity moves, and a body
// containing one deliberately-crafted link per move (root-relative or
// recomputed-relative, chosen at random) plus prose/link noise.
type linkRewriteInput struct {
	linkingFile string
	moves       []EntityMove
	body        string
	crafted     []craftedLink
}

// Generate implements testing/quick.Generator.
func (linkRewriteInput) Generate(r *rand.Rand, _ int) reflect.Value {
	linkingFile := randEntityPath(r)

	nMoves := r.Intn(propMaxMoves + 1)
	moves := make([]EntityMove, 0, nMoves)
	used := map[string]bool{linkingFile: true}
	for len(moves) < nMoves {
		from := randEntityPath(r)
		to := randEntityPath(r)
		if used[from] || used[to] || from == to {
			continue
		}
		used[from] = true
		used[to] = true
		moves = append(moves, EntityMove{From: from, To: to})
	}

	var sb strings.Builder
	var crafted []craftedLink
	for i, mv := range moves {
		sb.WriteString(randProse(r))
		label := fmt.Sprintf("MARKER%d", i)
		dest := mv.From
		rootRelative := true
		if r.Intn(2) == 1 {
			// Relative flavor: recompute against linkingFile's dir.
			// relativeFromDir is production code, reused here only to
			// construct a *valid input* destination string — the
			// property's assertion oracle (testResolvePath) takes the
			// flavor as a known parameter rather than re-deriving it,
			// so it stays independent of this call. Skip the
			// degenerate same-dir case (relativeFromDir returns ".")
			// since it can't carry a filename and would make the
			// marker unresolvable.
			if rel := relativeFromDir(path.Dir(linkingFile), mv.From); rel != "." {
				dest = rel
				rootRelative = false
			}
		}
		fmt.Fprintf(&sb, "[%s](%s)\n", label, dest)
		crafted = append(crafted, craftedLink{label: label, moveIdx: i, rootRelative: rootRelative})
	}

	nNoise := r.Intn(propMaxNoiseLinks + 1)
	for i := 0; i < nNoise; i++ {
		sb.WriteString(randProse(r))
		fmt.Fprintf(&sb, "[noise-%d](%s)\n", i, randEntityPath(r))
	}
	sb.WriteString(randProse(r))

	return reflect.ValueOf(linkRewriteInput{
		linkingFile: linkingFile,
		moves:       moves,
		body:        sb.String(),
		crafted:     crafted,
	})
}

const proseAlphabet = "the quick fix for gap and epic notes context here today soon later work item task"

// randProse returns a short line of filler words plus a trailing
// newline. The alphabet deliberately excludes markdown-special
// characters (`]`, `(`, `)`, backtick) so generated noise can never
// accidentally form a link or code span.
func randProse(r *rand.Rand) string {
	words := strings.Fields(proseAlphabet)
	n := r.Intn(4)
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString(words[r.Intn(len(words))])
		sb.WriteByte(' ')
	}
	sb.WriteByte('\n')
	return sb.String()
}

const slugAlphabet = "abcdefghijklmnopqrstuvwxyz"

func randSlug(r *rand.Rand) string {
	n := 3 + r.Intn(5)
	b := make([]byte, n)
	for i := range b {
		b[i] = slugAlphabet[r.Intn(len(slugAlphabet))]
	}
	return string(b)
}

// randEntityPath fabricates a repo-relative entity file path under a
// random known root, with 0-2 extra nested directories (simulating
// epic subdirectories) and a small id range so paths collide often
// enough to exercise real moves.
func randEntityPath(r *rand.Rand) string {
	root := propKindRoots[r.Intn(len(propKindRoots))]
	depth := r.Intn(3)
	parts := []string{root}
	for i := 0; i < depth; i++ {
		parts = append(parts, fmt.Sprintf("sub%d", r.Intn(3)))
	}
	parts = append(parts, fmt.Sprintf("X-%04d-%s.md", r.Intn(5), randSlug(r)))
	return strings.Join(parts, "/")
}

// testResolvePath is an independent resolver mirroring
// RewriteLinkDestinations' documented resolution rule (root-relative
// destinations compare as-is; everything else resolves against the
// linking file's directory), reimplemented here rather than calling
// resolveLinkDestination so the property test isn't asserting the
// primitive against itself. rootRelative is the flavor known from
// generation — NOT re-derived from dest's shape. Every EntityMove.To
// in this file happens to start with a recognized entity root (it
// comes from randEntityPath), so shape-based re-detection would
// treat any output as root-relative and silently stop checking
// whether a relative destination was actually recomputed — the
// property survived a real newDestination regression under that
// design (see M-0245/AC-3 work log) until switched to this form.
func testResolvePath(dest, linkingFile string, rootRelative bool) string {
	if rootRelative {
		return path.Clean(dest)
	}
	return path.Clean(path.Join(path.Dir(linkingFile), dest))
}

func propertyConfig(seed int64) *quick.Config {
	return &quick.Config{
		MaxCount: propertyRewriteRuns,
		Rand:     rand.New(rand.NewSource(seed)),
	}
}

// TestRewriteLinkDestinations_Property_Idempotent pins M-0245/AC-3's
// idempotence half: rewriting an already-rewritten body is a no-op.
func TestRewriteLinkDestinations_Property_Idempotent(t *testing.T) {
	t.Parallel()
	var note string
	property := func(in linkRewriteInput) bool {
		out1 := RewriteLinkDestinations([]byte(in.body), in.linkingFile, in.moves)
		out2 := RewriteLinkDestinations(out1, in.linkingFile, in.moves)
		if !bytes.Equal(out1, out2) {
			note = fmt.Sprintf("not idempotent:\nfirst:  %q\nsecond: %q", out1, out2)
			return false
		}
		return true
	}
	if err := quick.Check(property, propertyConfig(11)); err != nil {
		t.Errorf("idempotence: %s\n%v", note, err)
	}
}

// TestRewriteLinkDestinations_Property_RewrittenDestinationsResolveToNewPath
// pins M-0245/AC-3's resolution-correctness half: every deliberately
// crafted link (found by its unique marker label) resolves, under the
// independent testResolvePath oracle, to its move's To path.
func TestRewriteLinkDestinations_Property_RewrittenDestinationsResolveToNewPath(t *testing.T) {
	t.Parallel()
	var note string
	property := func(in linkRewriteInput) bool {
		out := string(RewriteLinkDestinations([]byte(in.body), in.linkingFile, in.moves))
		for _, c := range in.crafted {
			marker := "[" + c.label + "]("
			idx := strings.Index(out, marker)
			if idx < 0 {
				note = fmt.Sprintf("marker %q not found in output %q", c.label, out)
				return false
			}
			start := idx + len(marker)
			end := strings.Index(out[start:], ")")
			if end < 0 {
				note = fmt.Sprintf("marker %q: unterminated destination in %q", c.label, out)
				return false
			}
			gotDest := out[start : start+end]
			want := in.moves[c.moveIdx].To
			if resolved := testResolvePath(gotDest, in.linkingFile, c.rootRelative); resolved != want {
				note = fmt.Sprintf("marker %q: destination %q resolves to %q, want %q", c.label, gotDest, resolved, want)
				return false
			}
		}
		return true
	}
	if err := quick.Check(property, propertyConfig(12)); err != nil {
		t.Errorf("resolution correctness: %s\n%v", note, err)
	}
}
