package check

// Real-tree assertions for the G-0299 skill-body id discipline. These run
// the rule (and the placeholder-canonicality scan) over this repo's actual
// shipped skill bodies, so they pin the full-sweep (AC-4) and
// placeholder-normalization (AC-3) deliverables rather than synthetic
// fixtures. They live in package check (white-box) to reuse proseMask and
// the id patterns — the same machinery the production rule uses, so the
// test cannot drift from the rule's notion of "prose" or "real id".

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// repoRootForTest resolves the module root from this test file's compiled
// path (internal/check/ → ../..), matching the fsm_history_hints_test.go
// idiom. The test binary is built from the working tree, so this points at
// the skill bodies under test (worktree or main checkout alike).
func repoRootForTest(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

// skillBody is one shipped skill body located for a real-tree scan.
type skillBody struct {
	relPath string
	body    []byte
}

// collectSkillBodies walks the embedded skill-source trees under root and
// returns each SKILL.md's post-frontmatter body. Fails the test if a tree
// is missing — these are the shipped artifacts; their absence is a bug.
func collectSkillBodies(t *testing.T, root string) []skillBody {
	t.Helper()
	var out []skillBody
	for _, dir := range skillScanDirs {
		base := filepath.Join(root, dir)
		if _, err := os.Stat(base); err != nil {
			t.Fatalf("skill source tree %s missing: %v", dir, err)
		}
		err := fs.WalkDir(os.DirFS(base), ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
				return err
			}
			raw, readErr := os.ReadFile(filepath.Join(base, p))
			if readErr != nil {
				return readErr
			}
			body := raw
			if _, b, ok := entity.Split(raw); ok {
				body = b
			}
			out = append(out, skillBody{relPath: filepath.Join(dir, p), body: body})
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", dir, err)
		}
	}
	if len(out) == 0 {
		t.Fatal("no SKILL.md bodies found under the embedded skill trees")
	}
	return out
}

// TestSkillBodyID_RealEmbeddedTreeIsClean (AC-4) asserts the full sweep
// landed: no shipped skill body cites a real entity id. It scans each body
// independently via ScanSkillBodyID rather than through skillBodyIDReference,
// so it is a second code path onto the same property.
func TestSkillBodyID_RealEmbeddedTreeIsClean(t *testing.T) {
	t.Parallel()
	root := repoRootForTest(t)
	var msgs []string
	for _, sb := range collectSkillBodies(t, root) {
		for _, f := range ScanSkillBodyID(sb.body, sb.relPath) {
			msgs = append(msgs, fmt.Sprintf("%s:%d %s", f.Path, f.Line, f.Message))
		}
	}
	if len(msgs) != 0 {
		sort.Strings(msgs)
		shown := msgs
		if len(shown) > 25 {
			shown = shown[:25]
		}
		t.Fatalf("%d real-id citation(s) remain in shipped skill bodies (sweep incomplete):\n%s",
			len(msgs), strings.Join(shown, "\n"))
	}
}

// canonicalPlaceholder matches the one allowed placeholder shape: the
// canonical-width letter-N form (`G-NNNN`, `M-NNNN/AC-N`). Anything that
// looks like an id but is neither a real digit-bearing id nor this shape is
// a non-canonical placeholder the normalization (AC-3) must have removed.
var canonicalPlaceholder = regexp.MustCompile(`^(?:E|M|G|D|C|ADR)-NNNN(?:/AC-N)?$`)

// TestSkillBodyID_PlaceholdersAreCanonical (AC-3) asserts placeholder
// normalization landed: every id-shaped token in skill-body prose is either
// a real digit-bearing id (AC-4's concern, scanned out separately) or the
// canonical letter-N placeholder — no narrow widths (`E-NN`), idiosyncratic
// shapes (`G-XYZ`), or pseudo-arithmetic (`C-NNN+1`). proseMask exempts
// code/link carriers so regex examples and command snippets don't trip it.
func TestSkillBodyID_PlaceholdersAreCanonical(t *testing.T) {
	t.Parallel()
	root := repoRootForTest(t)
	var bad []string
	for _, sb := range collectSkillBodies(t, root) {
		masked := proseMask(sb.body)
		seen := map[string]bool{}
		for _, m := range idTokenPattern.FindAllString(masked, -1) {
			if seen[m] {
				continue
			}
			seen[m] = true
			if strictBareIDPattern.MatchString(m) || strictCompositeIDPattern.MatchString(m) {
				continue // a real digit-bearing id: AC-4's concern, not a placeholder
			}
			if canonicalPlaceholder.MatchString(m) {
				continue
			}
			bad = append(bad, fmt.Sprintf("%s: non-canonical placeholder %q", sb.relPath, m))
		}
	}
	if len(bad) != 0 {
		sort.Strings(bad)
		t.Fatalf("%d non-canonical placeholder(s) in skill-body prose (normalize to <prefix>-NNNN):\n%s",
			len(bad), strings.Join(bad, "\n"))
	}
}
