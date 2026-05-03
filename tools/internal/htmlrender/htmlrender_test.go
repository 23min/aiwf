package htmlrender

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// TestRender_FixtureTree_FilesAndLinks renders a small fixture tree
// (1 epic, 2 milestones, 2 ACs) into a tempdir and verifies:
//   - every entity gets one HTML file at the expected path;
//   - the index page links to every epic; epic page links to every
//     milestone; milestone page links back to its parent.
//
// This is the load-bearing seam test for the step-3 render
// skeleton — it exercises sortedByID, idToFileName, and the
// embed.FS template loading pipeline as a single round-trip.
func TestRender_FixtureTree_FilesAndLinks(t *testing.T) {
	root := writeFixtureTree(t)

	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	out := filepath.Join(t.TempDir(), "site")
	res, err := Render(Options{OutDir: out, Tree: tr, Root: root})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// 1 index + 1 epic + 2 milestones = 4 HTML files.
	if res.FilesWritten != 4 {
		t.Errorf("FilesWritten = %d, want 4", res.FilesWritten)
	}

	// Each expected file is on disk.
	for _, name := range []string{"index.html", "E-01.html", "M-001.html", "M-002.html", "assets/style.css"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("missing output %s: %v", name, err)
		}
	}

	// Index links to E-01.
	indexHTML := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(indexHTML, `href="E-01.html"`) {
		t.Errorf("index.html missing link to E-01.html\n%s", indexHTML)
	}

	// Epic page links to both milestones.
	epicHTML := readFile(t, filepath.Join(out, "E-01.html"))
	for _, mid := range []string{"M-001.html", "M-002.html"} {
		if !strings.Contains(epicHTML, `href="`+mid+`"`) {
			t.Errorf("E-01.html missing link to %s\n%s", mid, epicHTML)
		}
	}

	// Milestone page links back to parent epic.
	mHTML := readFile(t, filepath.Join(out, "M-001.html"))
	if !strings.Contains(mHTML, `href="E-01.html"`) {
		t.Errorf("M-001.html missing link to parent E-01.html\n%s", mHTML)
	}

	// AC anchors land as id attributes.
	if !strings.Contains(mHTML, `id="ac-1"`) {
		t.Errorf("M-001.html missing AC-1 anchor\n%s", mHTML)
	}

	// Link integrity: every internal href on a rendered page must
	// resolve to a file we wrote (or be an in-page #ac- fragment).
	verifyLinkIntegrity(t, out)
}

// TestRender_DeterministicAcrossInvocations renders the same
// fixture tree into two separate dirs and asserts byte-identical
// output. Pins the I3 plan §8 "Determinism" rule for step 3 — the
// real determinism gate is in step 4, but the renderer's own
// behavior must already satisfy it.
func TestRender_DeterministicAcrossInvocations(t *testing.T) {
	root := writeFixtureTree(t)
	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}

	out1 := filepath.Join(t.TempDir(), "site1")
	out2 := filepath.Join(t.TempDir(), "site2")
	if _, err := Render(Options{OutDir: out1, Tree: tr, Root: root}); err != nil {
		t.Fatalf("Render(out1): %v", err)
	}
	if _, err := Render(Options{OutDir: out2, Tree: tr, Root: root}); err != nil {
		t.Fatalf("Render(out2): %v", err)
	}

	for _, rel := range []string{"index.html", "E-01.html", "M-001.html", "M-002.html", "assets/style.css"} {
		a := readFile(t, filepath.Join(out1, rel))
		b := readFile(t, filepath.Join(out2, rel))
		if a != b {
			t.Errorf("non-deterministic output for %s; len1=%d len2=%d", rel, len(a), len(b))
		}
	}
}

// TestRender_EmptyTree_StillProducesIndex: a tree with no epics
// must still produce a valid index.html (with an empty-state line)
// so the deployment pipeline always has something to ship.
func TestRender_EmptyTree_StillProducesIndex(t *testing.T) {
	root := setupEmptyTree(t)
	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Render(Options{OutDir: out, Tree: tr, Root: root}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	html := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(html, "No epics yet") {
		t.Errorf("empty-state index missing fallback line\n%s", html)
	}
}

func TestIDToFileName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "index.html"},
		{"E-01", "E-01.html"},
		{"M-007", "M-007.html"},
		{"ADR-0042", "ADR-0042.html"},
		{"G-099", "G-099.html"},
		{"D-003", "D-003.html"},
		{"C-100", "C-100.html"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := idToFileName(tc.in); got != tc.want {
				t.Errorf("idToFileName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestACAnchor(t *testing.T) {
	cases := map[string]string{
		"AC-1":  "ac-1",
		"AC-12": "ac-12",
	}
	for in, want := range cases {
		if got := ACAnchor(in); got != want {
			t.Errorf("ACAnchor(%q) = %q, want %q", in, got, want)
		}
	}
}

// hrefRE extracts the href attribute value from an anchor tag. We do
// not need a full HTML parser — the templates produce a small,
// well-known shape.
var hrefRE = regexp.MustCompile(`href="([^"]+)"`)

// verifyLinkIntegrity walks every rendered HTML file and asserts
// every `href="..."` either points at a file that exists in outDir,
// at a `#`-fragment within the same page, or at a stylesheet.
func verifyLinkIntegrity(t *testing.T, outDir string) {
	t.Helper()
	walkErr := filepath.WalkDir(outDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".html") {
			return nil
		}
		body := readFile(t, path)
		for _, m := range hrefRE.FindAllStringSubmatch(body, -1) {
			href := m[1]
			if strings.HasPrefix(href, "#") {
				continue
			}
			// Strip query string + fragment before checking file
			// existence. The cache-busting `?v=<hash>` on the
			// stylesheet href is a browser-cache hint, not a path
			// component.
			fileOnly := href
			if i := strings.IndexAny(fileOnly, "?#"); i >= 0 {
				fileOnly = fileOnly[:i]
			}
			target := filepath.Join(outDir, fileOnly)
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s: broken link to %q (%v)", filepath.Base(path), href, err)
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk %s: %v", outDir, walkErr)
	}
}

// writeFixtureTree builds a small but complete tree: 1 epic with 2
// milestones, the first milestone carrying 2 ACs. Returns the repo
// root.
func writeFixtureTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	epicDir := filepath.Join(root, "work", "epics", "E-01-foundations")
	mustMkdir(t, epicDir)
	mustWrite(t, filepath.Join(epicDir, "epic.md"),
		"---\nid: E-01\ntitle: Foundations\nstatus: active\n---\n\n## Goal\n\nbuild things\n")
	mustWrite(t, filepath.Join(epicDir, "M-001-first.md"),
		`---
id: M-001
title: First
status: in_progress
parent: E-01
acs:
    - id: AC-1
      title: AC one
      status: open
    - id: AC-2
      title: AC two
      status: open
---

## Goal
ship it
`)
	mustWrite(t, filepath.Join(epicDir, "M-002-second.md"),
		"---\nid: M-002\ntitle: Second\nstatus: draft\nparent: E-01\n---\n\n## Goal\n\nlater\n")
	return root
}

func setupEmptyTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "work", "epics"))
	return root
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// avoid an unused-import warning; entity, exec are referenced by
// other helpers later in step 4 but for step 3 keep the imports
// tight by referencing them at least once.
var (
	_ = entity.KindEpic
	_ = exec.Command
)
