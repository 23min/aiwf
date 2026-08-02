package check

// Real-tree assertions for the skill-body id discipline: they run the
// production rule over this repo's actual shipped surfaces rather than over
// synthetic fixtures, so they pin the shipped bytes.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
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

// TestSkillBodyID_RealEmbeddedTreeIsClean scans each shipped skill body
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
		t.Fatalf("shipped skill bodies carry %d id-shaped defects:\n  %s",
			len(msgs), strings.Join(msgs, "\n  "))
	}
}

// TestSkillBodyID_WholeShippedTreeClean drives the production check over the
// repo root, so it exercises the registered walkers — the whole-file *.md scan
// and the statusline #-comment scan — against the same rules the pre-push hook
// runs. Placeholder canonicality is included: skillTokenMessage owns that
// property over every *.md whole-file, frontmatter included.
//
// An in-memory tree rooted at the repo suffices: the two walkers key only on
// t.Root, and the entity-driven checks see no entities, so the only findings
// that can surface are skill-body-id.
func TestSkillBodyID_WholeShippedTreeClean(t *testing.T) {
	t.Parallel()
	root := repoRootForTest(t)
	var msgs []string
	for _, f := range Run(&tree.Tree{Root: root}, nil) {
		if f.Code == CodeSkillBodyID {
			msgs = append(msgs, fmt.Sprintf("%s:%d %s", f.Path, f.Line, f.Message))
		}
	}
	if len(msgs) != 0 {
		t.Fatalf("shipped surfaces carry %d id-shaped defects:\n  %s",
			len(msgs), strings.Join(msgs, "\n  "))
	}
}
