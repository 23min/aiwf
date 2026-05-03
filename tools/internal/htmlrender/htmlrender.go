// Package htmlrender produces a static-site governance render of an
// aiwf planning tree: one index page, one page per epic, one page per
// milestone. Output is a directory of self-contained HTML files plus
// a single embedded stylesheet — no JS, no runtime, no external
// assets.
//
// This package is the read-side renderer for I3 step 5's templates;
// step 3 (this file) lays the seams. Templates and CSS live under
// embedded/ and are pulled in via go:embed in embed.go. A minimal
// placeholder template ships now so callers can verify the
// directory layout, link integrity, and determinism (render twice →
// byte-identical output) before the real templates land in step 5.
//
// The package is deterministic by construction:
//   - sorted iteration over the tree's entities (no map range);
//   - no wall-clock timestamps in output (all dates derive from the
//     entity's commit metadata, captured by the caller);
//   - sorted directory enumeration where applicable.
//
// See docs/pocv3/plans/governance-html-plan.md §8 "Determinism" for
// the load-bearing rules.
package htmlrender

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// Options is the input to Render. OutDir is the absolute directory
// the renderer writes into (callers who pass a relative path should
// resolve it first). Tree is the loaded planning tree; Root is the
// repo root used for any per-entity body reads. Scope, when non-empty,
// limits the render to one entity id and its referenced children
// (step-4 plumbing — step 3 ignores it but the field is reserved so
// the seam is in place).
type Options struct {
	OutDir string
	Tree   *tree.Tree
	Root   string
	Scope  string
}

// Result reports what the render produced. FilesWritten is the count
// of HTML files emitted (excluding the stylesheet, which always
// writes once); ElapsedMs is wall-clock time for the render. Both
// surface in the verb's JSON envelope per I3 step 4.
type Result struct {
	FilesWritten int
	ElapsedMs    int64
}

// Render produces the static-site output under opts.OutDir. The
// directory is created if it does not exist and existing files at
// the same paths are overwritten.
//
// The renderer is a pure function of opts.Tree (and the per-entity
// body files it reads). Two calls with identical inputs produce
// byte-identical files — the determinism test in step 4 pins this.
func Render(opts Options) (Result, error) {
	start := time.Now()
	if opts.Tree == nil {
		return Result{}, fmt.Errorf("htmlrender.Render: opts.Tree is required")
	}
	if opts.OutDir == "" {
		return Result{}, fmt.Errorf("htmlrender.Render: opts.OutDir is required")
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("creating %s: %w", opts.OutDir, err)
	}
	assetsDir := filepath.Join(opts.OutDir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("creating assets dir: %w", err)
	}
	if err := writeAssetFile(filepath.Join(assetsDir, "style.css"), embeddedStyleCSS); err != nil {
		return Result{}, err
	}

	tmpls, err := loadTemplates()
	if err != nil {
		return Result{}, err
	}

	count := 0
	if err := renderIndex(opts, tmpls); err != nil {
		return Result{}, err
	}
	count++

	for _, e := range sortedByID(opts.Tree.ByKind(entity.KindEpic)) {
		if err := renderEpic(opts, tmpls, e); err != nil {
			return Result{}, err
		}
		count++
	}
	for _, m := range sortedByID(opts.Tree.ByKind(entity.KindMilestone)) {
		if err := renderMilestone(opts, tmpls, m); err != nil {
			return Result{}, err
		}
		count++
	}

	return Result{
		FilesWritten: count,
		ElapsedMs:    time.Since(start).Milliseconds(),
	}, nil
}

// renderIndex writes the top-level index.html. Step-3 placeholder
// content: the list of epics with their ids and titles. Step 5
// replaces with the full epics + AC met-rollup table.
func renderIndex(opts Options, tmpls *template.Template) error {
	type epicRow struct {
		ID, Title, Status, FileName string
	}
	data := struct {
		Title string
		Epics []epicRow
	}{
		Title: "Governance",
	}
	for _, e := range sortedByID(opts.Tree.ByKind(entity.KindEpic)) {
		data.Epics = append(data.Epics, epicRow{
			ID:       e.ID,
			Title:    e.Title,
			Status:   e.Status,
			FileName: idToFileName(e.ID),
		})
	}
	return executeToFile(tmpls, "index.tmpl", filepath.Join(opts.OutDir, "index.html"), data)
}

// renderEpic writes one epic page. Step-3 placeholder content: the
// epic's id/title/status plus its child milestones (id + title only).
// Step 5 fills in the dependency DAG, linked entities, history.
func renderEpic(opts Options, tmpls *template.Template, e *entity.Entity) error {
	type milestoneRow struct {
		ID, Title, Status, FileName string
	}
	var milestones []milestoneRow
	for _, m := range sortedByID(opts.Tree.ByKind(entity.KindMilestone)) {
		if m.Parent != e.ID {
			continue
		}
		milestones = append(milestones, milestoneRow{
			ID:       m.ID,
			Title:    m.Title,
			Status:   m.Status,
			FileName: idToFileName(m.ID),
		})
	}
	data := struct {
		Epic       *entity.Entity
		Milestones []milestoneRow
	}{Epic: e, Milestones: milestones}
	return executeToFile(tmpls, "epic.tmpl", filepath.Join(opts.OutDir, idToFileName(e.ID)), data)
}

// renderMilestone writes one milestone page. Step-3 placeholder
// content: the milestone's id/title/status plus its ACs (id + title).
// Step 5 fills in the six tabs (Overview, Manifest, Build, Tests,
// Commits, Provenance).
func renderMilestone(opts Options, tmpls *template.Template, m *entity.Entity) error {
	type acRow struct {
		ID, Title, Status, TDDPhase string
	}
	acs := make([]acRow, 0, len(m.ACs))
	for _, ac := range m.ACs {
		acs = append(acs, acRow{
			ID:       ac.ID,
			Title:    ac.Title,
			Status:   ac.Status,
			TDDPhase: ac.TDDPhase,
		})
	}
	parentFile := ""
	if m.Parent != "" {
		parentFile = idToFileName(m.Parent)
	}
	data := struct {
		Milestone  *entity.Entity
		ACs        []acRow
		ParentFile string
	}{Milestone: m, ACs: acs, ParentFile: parentFile}
	return executeToFile(tmpls, "milestone.tmpl", filepath.Join(opts.OutDir, idToFileName(m.ID)), data)
}

// sortedByID returns a copy of entities sorted by id. Used so every
// iteration site producing output uses the same canonical order;
// directly ranging over Tree.ByKind would be order-stable today but
// the contract isn't.
func sortedByID(entities []*entity.Entity) []*entity.Entity {
	out := append([]*entity.Entity(nil), entities...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// executeToFile renders tmplName with data and writes the result to
// path. Truncates any existing file. Failure to render or write
// surfaces as a wrapped error naming the path.
func executeToFile(tmpls *template.Template, tmplName, path string, data any) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	if err := tmpls.ExecuteTemplate(f, tmplName, data); err != nil {
		_ = f.Close()
		return fmt.Errorf("rendering %s into %s: %w", tmplName, path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", path, err)
	}
	return nil
}

// writeAssetFile writes a static asset file with stable permissions,
// truncating any existing file at path.
func writeAssetFile(path string, content []byte) error {
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
