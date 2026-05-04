package htmlrender

import (
	"bytes"
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
	// 1 index + 1 epic + 2 milestones + 1 gap + 1 ADR + 1 decision +
	// 1 contract = 8 HTML files.
	if res.FilesWritten != 8 {
		t.Errorf("FilesWritten = %d, want 8", res.FilesWritten)
	}

	// Each expected file is on disk.
	for _, name := range []string{
		"index.html", "E-01.html", "M-001.html", "M-002.html",
		"G-001.html", "ADR-0001.html", "D-001.html", "C-001.html",
		"assets/style.css",
	} {
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

	for _, rel := range []string{
		"index.html", "E-01.html", "M-001.html", "M-002.html",
		"G-001.html", "ADR-0001.html", "D-001.html", "C-001.html",
		"assets/style.css",
	} {
		a := readFile(t, filepath.Join(out1, rel))
		b := readFile(t, filepath.Join(out2, rel))
		if a != b {
			t.Errorf("non-deterministic output for %s; len1=%d len2=%d", rel, len(a), len(b))
		}
	}
}

// TestRender_BodyMarkdownRendersAsHTML is the G36 seam test:
// markdown body content (lists, links, fenced code, inline code,
// emphasis) renders as real HTML on the gap/ADR/decision/contract
// pages, not as escaped raw text. Pre-G36 the templates emitted
// `<p>{{.}}</p>` which HTML-escaped the source and produced
// pages full of literal `*foo*` / backtick noise.
//
// The fixture writes one entity per kind whose body exercises the
// load-bearing markdown shapes (list, fenced code, link, inline
// code), then asserts the rendered HTML contains the corresponding
// HTML elements.
func TestRender_BodyMarkdownRendersAsHTML(t *testing.T) {
	root := t.TempDir()
	gapsDir := filepath.Join(root, "work", "gaps")
	mustMkdir(t, gapsDir)
	mustWrite(t, filepath.Join(gapsDir, "G-077-mdcheck.md"),
		"---\nid: G-077\ntitle: Markdown check\nstatus: open\n---\n\n## Symptoms\n\n- list item one\n- list item two\n\nUse `aiwf check` first.\n\nSee [the docs](https://example.com).\n\n## Repro\n\n```go\nfmt.Println(\"hi\")\n```\n")

	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	// htmlrender's default resolver returns no Sections (body access
	// is a cmd-side concern); to exercise the markdown helper through
	// a page render we need a resolver that surfaces sections. Use
	// bodyAwareResolver, which mirrors cmd-side wiring without the
	// git/history dependencies.
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Render(Options{
		OutDir: out,
		Tree:   tr,
		Root:   root,
		Data:   bodyAwareResolver{tree: tr, root: root},
	}); err != nil {
		t.Fatalf("Render with body-aware resolver: %v", err)
	}
	html := readFile(t, filepath.Join(out, "G-077.html"))

	for _, want := range []string{
		"<ul>",
		"<li>list item one</li>",
		"<code>aiwf check</code>",
		`<a href="https://example.com">the docs</a>`,
		"<pre>",
		"fmt.Println",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("G-077.html missing %q\n--- snippet ---\n%s", want, snippetAround(html, want))
		}
	}
	// Negative: literal markdown source must NOT appear (it would
	// mean the section was emitted via the old escaped path).
	for _, forbidden := range []string{
		"- list item one",
		"```go",
	} {
		if strings.Contains(html, forbidden) {
			t.Errorf("G-077.html contains raw markdown %q (escaped, not rendered)", forbidden)
		}
	}
}

// snippetAround returns a 200-char window around the first occurrence
// of want in s; falls back to the first 200 chars when want is absent.
// Used to keep error messages readable when an HTML page is several
// KB long.
func snippetAround(s, want string) string {
	idx := strings.Index(s, want)
	if idx < 0 {
		if len(s) > 200 {
			return s[:200] + "…"
		}
		return s
	}
	start := idx - 100
	if start < 0 {
		start = 0
	}
	end := idx + 100
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}

// bodyAwareResolver is a test-only PageDataResolver that wraps
// defaultResolver but overrides EntityData to read the body from
// disk and parse sections via entity.ParseBodySectionsOrdered, the
// same wiring cmd/aiwf's renderResolver uses. Lets the htmlrender
// package's own tests exercise the markdown round-trip without
// pulling in cmd-side dependencies.
type bodyAwareResolver struct {
	tree *tree.Tree
	root string
}

func (r bodyAwareResolver) IndexData() (*IndexData, error) {
	return defaultResolver{tree: r.tree}.IndexData()
}

func (r bodyAwareResolver) EpicData(id string) (*EpicData, error) {
	return defaultResolver{tree: r.tree}.EpicData(id)
}

func (r bodyAwareResolver) MilestoneData(id string) (*MilestoneData, error) {
	return defaultResolver{tree: r.tree}.MilestoneData(id)
}

func (r bodyAwareResolver) StatusData() (*StatusData, error) {
	return nil, nil
}

func (r bodyAwareResolver) EntityData(id string) (*EntityData, error) {
	data, err := defaultResolver{tree: r.tree}.EntityData(id)
	if err != nil || data == nil {
		return data, err
	}
	e := r.tree.ByID(id)
	if e == nil {
		return data, nil
	}
	abs := filepath.Join(r.root, e.Path)
	body, err := os.ReadFile(abs)
	if err != nil {
		return data, nil
	}
	// Strip frontmatter — find the second `---` line.
	frontDelim := []byte("---\n")
	if idx := bytes.Index(body, frontDelim); idx >= 0 {
		rest := body[idx+len(frontDelim):]
		if end := bytes.Index(rest, []byte("\n---\n")); end >= 0 {
			body = rest[end+len("\n---\n"):]
		}
	}
	for _, s := range entity.ParseBodySectionsOrdered(body) {
		data.Sections = append(data.Sections, BodySectionView{
			Slug:    s.Slug,
			Heading: s.Heading,
			Content: s.Content,
		})
	}
	return data, nil
}

// TestRender_NonEpicMilestoneKinds_GetPages is the G35 seam test:
// gap, ADR, decision, and contract entities each get their own
// HTML page rendered through the shared entity template. Pre-G35
// these kinds were referenced from the index but had no pages, so
// every link 404'd. The test pins one page per kind, asserting the
// kind kicker and the entity title are present in the right place.
func TestRender_NonEpicMilestoneKinds_GetPages(t *testing.T) {
	root := writeFixtureTree(t)
	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Render(Options{OutDir: out, Tree: tr, Root: root}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	cases := []struct {
		file  string
		kind  string
		title string
	}{
		{"G-001.html", "gap", "Flaky build"},
		{"ADR-0001.html", "adr", "Tooling choice"},
		{"D-001.html", "decision", "Release cadence"},
		{"C-001.html", "contract", "API contract"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			html := readFile(t, filepath.Join(out, tc.file))
			// Kicker carries the kind label; title is in the H1.
			if !strings.Contains(html, tc.kind+" · ") {
				t.Errorf("%s missing kind kicker %q", tc.file, tc.kind)
			}
			if !strings.Contains(html, "<h1>"+tc.title+"</h1>") {
				t.Errorf("%s missing <h1>%s</h1>", tc.file, tc.title)
			}
			// Sidebar link back to the index.
			if !strings.Contains(html, `href="index.html"`) {
				t.Errorf("%s missing index link", tc.file)
			}
		})
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
// milestones, the first milestone carrying 2 ACs, plus one entity
// of each non-epic/non-milestone kind (gap, ADR, decision, contract)
// so the per-kind rendering is exercised. Returns the repo root.
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

	gapsDir := filepath.Join(root, "work", "gaps")
	mustMkdir(t, gapsDir)
	mustWrite(t, filepath.Join(gapsDir, "G-001-flaky-build.md"),
		"---\nid: G-001\ntitle: Flaky build\nstatus: open\n---\n\n## What's missing\n\nAn investigation.\n\n## Why it matters\n\nCI is unreliable.\n")

	adrDir := filepath.Join(root, "docs", "adr")
	mustMkdir(t, adrDir)
	mustWrite(t, filepath.Join(adrDir, "ADR-0001-tooling.md"),
		"---\nid: ADR-0001\ntitle: Tooling choice\nstatus: accepted\n---\n\n## Context\n\nNeed a tool.\n\n## Decision\n\nUse it.\n\n## Consequences\n\nLife is good.\n")

	decisionDir := filepath.Join(root, "work", "decisions")
	mustMkdir(t, decisionDir)
	mustWrite(t, filepath.Join(decisionDir, "D-001-cadence.md"),
		"---\nid: D-001\ntitle: Release cadence\nstatus: open\n---\n\n## Question\n\nWhen do we ship?\n\n## Decision\n\nMonthly.\n\n## Reasoning\n\nMatches QA.\n")

	contractSubdir := filepath.Join(root, "work", "contracts", "C-001-api")
	mustMkdir(t, contractSubdir)
	mustWrite(t, filepath.Join(contractSubdir, "contract.md"),
		"---\nid: C-001\ntitle: API contract\nstatus: accepted\n---\n\n## Purpose\n\nDescribe the API.\n\n## Stability\n\nLocked.\n")
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
