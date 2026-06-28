package check

import (
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// TestAreaPathChecks_InertWithoutPaths pins M-0180/AC-5: a label-only / legacy
// string-form config (no member declares paths) fires neither dead-glob nor
// overlap. The two path-axis checks share one inertness contract — without
// paths there is no oracle to police — so the E-0043 backward-compat configs
// keep validating byte-for-byte after the path-axis checks landed. Asserting
// both checks in one place guards against a future change making only one of
// them fire on a paths-less member.
func TestAreaPathChecks_InertWithoutPaths(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// A real directory is present, so silence here is "no paths to police",
	// not "nothing on disk".
	mkAreaDir(t, root, "projects/app-a")
	// Two label-only members — the legacy string form decodes to Name set,
	// Paths nil.
	areas := []AreaPaths{
		{Name: "app-a", Paths: nil},
		{Name: "app-b", Paths: nil},
	}
	if got := AreaDeadGlob(&tree.Tree{Root: root}, areas); len(got) != 0 {
		t.Errorf("dead-glob must be inert without paths, got %+v", got)
	}
	if got := AreaOverlap(&tree.Tree{Root: root}, areas); len(got) != 0 {
		t.Errorf("overlap must be inert without paths, got %+v", got)
	}
}
