package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// G-0241 seam test: the edit-body dispatcher must load via
// cliutil.LoadTreeWithTrunk (not bare tree.Load) so the verb-time
// body-prose-id scan sees TrunkIDs. The unit tier
// (TestBodyProseID_TrunkTier_G0241 in internal/check) proves the
// scanner's trunk tier works when TrunkIDs is populated; this test
// proves the dispatcher actually populates it — the seam a unit test
// cannot catch if the dispatcher regresses to a trunk-blind load.
//
// Scenario (same two-clone shape as the G37 allocator tests):
//
//  1. Clone A adds gap G-0001 and pushes. Origin now has G-0001.
//  2. Clone B (cloned before A's gap landed) runs `git fetch` so
//     refs/remotes/origin/main sees G-0001, but B's working tree
//     does not have the file.
//  3. Clone B adds its own gap (allocator picks G-0002, trunk-aware)
//     and runs `aiwf edit-body G-0002` with a body referencing
//     G-0001. Pre-G-0241 the verb refused the write
//     (body-prose-id/unresolved); now the trunk tier resolves it.
//
// The negative control inside the same fixture pins that the trunk
// tier did not widen anything: a body referencing a truly-unknown id
// still refuses.
func TestIntegrationG0241_EditBodyResolvesTrunkOnlyID(t *testing.T) {
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	bare := makeBareOrigin(t)
	cloneA := makeClone(t, bare, "A")
	aiwfInitClone(t, cloneA, binDir)

	// B clones BEFORE A's gap exists, so the gap never enters B's
	// working tree.
	cloneB := makeSiblingClone(t, bare, "B")

	aiwfAddGap(t, cloneA, binDir, "Trunk side gap")
	pushAll(t, cloneA)

	// B learns about G-0001 via refs only.
	fetchOrigin(t, cloneB)

	aiwfAddGap(t, cloneB, binDir, "Branch side gap")

	bodyDir := t.TempDir()

	// Negative control first (no write side effects): a truly-unknown
	// id must still refuse, proving the trunk tier didn't go
	// permissive across the board.
	unknownBody := "## What's missing\n\nDepends on G-9999 which exists nowhere.\n\n## Why it matters\n\nIt must refuse.\n"
	unknownFile := filepath.Join(bodyDir, "unknown.md")
	if err := os.WriteFile(unknownFile, []byte(unknownBody), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}
	out, err := testutil.RunBin(t, cloneB, binDir, nil, "edit-body", "G-0002", "--body-file", unknownFile)
	if err == nil {
		t.Fatalf("edit-body with truly-unknown G-9999 should refuse; output:\n%s", out)
	}
	if !strings.Contains(out, "G-9999") {
		t.Errorf("refusal output should name the unresolved token G-9999; got:\n%s", out)
	}

	// Positive: G-0001 is trunk-only from B's perspective — the write
	// must proceed.
	// The prose deliberately avoids any other id-shaped token: the
	// fixture clone's tree knows nothing beyond G-0001 (trunk) and
	// G-0002 (local), so e.g. naming this gap's own id in the prose
	// would itself fire unresolved — the scanner doing its job.
	trunkBody := "## What's missing\n\nDepends on the trunk-side gap G-0001 which this branch has not merged yet.\n\n## Why it matters\n\nBefore the trunk tier landed, the verb-time body-prose-id scan refused this write.\n"
	trunkFile := filepath.Join(bodyDir, "trunk.md")
	if werr := os.WriteFile(trunkFile, []byte(trunkBody), 0o644); werr != nil {
		t.Fatalf("write body file: %v", werr)
	}
	out, err = testutil.RunBin(t, cloneB, binDir, nil, "edit-body", "G-0002", "--body-file", trunkFile)
	if err != nil {
		t.Fatalf("edit-body referencing trunk-only G-0001 should succeed (G-0241); got %v:\n%s", err, out)
	}

	// The body landed on disk with the trunk reference intact.
	gapPath := findEntityPath(t, cloneB, filepath.Join("work", "gaps"), "G-0002-")
	if gapPath == "" {
		t.Fatal("G-0002 gap file not found in clone B")
	}
	content, err := os.ReadFile(filepath.Join(cloneB, gapPath))
	if err != nil {
		t.Fatalf("reading %s: %v", gapPath, err)
	}
	if !strings.Contains(string(content), "G-0001") {
		t.Errorf("edited body should reference G-0001; file content:\n%s", content)
	}
}

// TestIntegrationG0241_ImportResolvesTrunkOnlyID is the import-side
// twin of the edit-body seam test above: the import dispatcher also
// switched from bare tree.Load to cliutil.LoadTreeWithTrunk, so a
// manifest entry whose body references a trunk-only id must import
// cleanly while a truly-unknown reference still refuses.
func TestIntegrationG0241_ImportResolvesTrunkOnlyID(t *testing.T) {
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)

	bare := makeBareOrigin(t)
	cloneA := makeClone(t, bare, "A")
	aiwfInitClone(t, cloneA, binDir)
	cloneB := makeSiblingClone(t, bare, "B")

	aiwfAddGap(t, cloneA, binDir, "Trunk side gap")
	pushAll(t, cloneA)
	fetchOrigin(t, cloneB)

	manifestDir := t.TempDir()
	manifest := func(name, ref string) string {
		path := filepath.Join(manifestDir, name)
		body := `version: 1
entities:
  - kind: gap
    id: G-0500
    frontmatter: {title: "Imported gap", status: open}
    body: |
      ## What's missing

      Depends on the gap ` + ref + ` per the import manifest.

      ## Why it matters

      Import must not refuse trunk-known references.
`
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
		return path
	}

	// Negative control: truly-unknown reference still refuses.
	out, err := testutil.RunBin(t, cloneB, binDir, nil, "import", manifest("unknown.yaml", "G-9999"))
	if err == nil {
		t.Fatalf("import with truly-unknown G-9999 should refuse; output:\n%s", out)
	}
	if !strings.Contains(out, "G-9999") {
		t.Errorf("refusal output should name the unresolved token G-9999; got:\n%s", out)
	}

	// Positive: trunk-only G-0001 resolves through the import
	// dispatcher's trunk-stamped tree.
	out, err = testutil.RunBin(t, cloneB, binDir, nil, "import", manifest("trunk.yaml", "G-0001"))
	if err != nil {
		t.Fatalf("import referencing trunk-only G-0001 should succeed (G-0241); got %v:\n%s", err, out)
	}
	gapPath := findEntityPath(t, cloneB, filepath.Join("work", "gaps"), "G-0500-")
	if gapPath == "" {
		t.Fatal("imported G-0500 gap file not found in clone B")
	}
}
