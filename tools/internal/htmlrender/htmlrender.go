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
//
// Data is a per-page resolver supplied by the caller. The renderer
// calls Data.IndexData() once, Data.EpicData(id) for each epic, and
// Data.MilestoneData(id) for each milestone. Returning nil for any
// of these triggers the empty-state path in the template — the
// page still renders, with a "no data" line.
//
// Splitting the data resolution out keeps htmlrender free of git /
// history walking. The cmd/aiwf side (which already has those
// helpers wired for `aiwf show`) builds the page data once per id
// and hands it in.
type Options struct {
	OutDir string
	Tree   *tree.Tree
	Root   string
	Scope  string
	Data   PageDataResolver
}

// PageDataResolver is the per-page data provider Render consults.
// Implementations build the typed view models from whatever sources
// they need (frontmatter, git log, scope FSM); the renderer walks
// the tree, calls the resolver per entity, and applies templates.
//
// A nil Resolver triggers a minimal default that returns just the
// frontmatter shape — useful for the htmlrender package's own tests
// and for any caller who only wants id/title/status without history.
type PageDataResolver interface {
	IndexData() (*IndexData, error)
	EpicData(id string) (*EpicData, error)
	MilestoneData(id string) (*MilestoneData, error)
	// EntityData returns the page payload for the four kinds with
	// no specialized template (gap, ADR, decision, contract). The
	// renderer routes those kinds through a shared entity template;
	// epic and milestone pages keep their dedicated resolvers above.
	// A nil return (with no error) skips the entity's page —
	// resolvers should only do this on a kind mismatch, otherwise
	// every linked entity is expected to have a page.
	EntityData(id string) (*EntityData, error)
	// StatusData returns the project-status page payload. A nil
	// return (with no error) means "skip the status page" — the
	// renderer will not emit status.html and the sidebar will
	// suppress the link. The default resolver returns nil so the
	// htmlrender package's own tests don't need git access.
	StatusData() (*StatusData, error)
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
	resolver := opts.Data
	if resolver == nil {
		resolver = defaultResolver{tree: opts.Tree}
	}

	count := 0
	if err := renderIndex(opts, tmpls, resolver); err != nil {
		return Result{}, err
	}
	count++

	if status, err := resolver.StatusData(); err != nil {
		return Result{}, fmt.Errorf("StatusData: %w", err)
	} else if status != nil {
		if err := renderStatus(opts, tmpls, status); err != nil {
			return Result{}, err
		}
		count++
	}

	for _, e := range sortedByID(opts.Tree.ByKind(entity.KindEpic)) {
		if err := renderEpic(opts, tmpls, resolver, e.ID); err != nil {
			return Result{}, err
		}
		count++
	}
	for _, m := range sortedByID(opts.Tree.ByKind(entity.KindMilestone)) {
		if err := renderMilestone(opts, tmpls, resolver, m.ID); err != nil {
			return Result{}, err
		}
		count++
	}
	for _, kind := range []entity.Kind{entity.KindADR, entity.KindGap, entity.KindDecision, entity.KindContract} {
		for _, e := range sortedByID(opts.Tree.ByKind(kind)) {
			if err := renderEntity(opts, tmpls, resolver, e.ID); err != nil {
				return Result{}, err
			}
			count++
		}
	}

	return Result{
		FilesWritten: count,
		ElapsedMs:    time.Since(start).Milliseconds(),
	}, nil
}

// renderIndex writes the top-level index.html. Pulls IndexData from
// the resolver; nil indicates "no data" — the template renders an
// empty-state line.
func renderIndex(opts Options, tmpls *template.Template, resolver PageDataResolver) error {
	data, err := resolver.IndexData()
	if err != nil {
		return fmt.Errorf("IndexData: %w", err)
	}
	if data == nil {
		data = &IndexData{Title: "Overview"}
	}
	if data.Title == "" {
		data.Title = "Overview"
	}
	return executeToFile(tmpls, "index.tmpl", filepath.Join(opts.OutDir, "index.html"), data)
}

// renderEpic writes one epic page. Resolver builds the typed
// EpicData from the tree + git history.
func renderEpic(opts Options, tmpls *template.Template, resolver PageDataResolver, id string) error {
	data, err := resolver.EpicData(id)
	if err != nil {
		return fmt.Errorf("EpicData(%s): %w", id, err)
	}
	if data == nil || data.Epic == nil {
		return fmt.Errorf("EpicData(%s) returned no Epic ref", id)
	}
	return executeToFile(tmpls, "epic.tmpl", filepath.Join(opts.OutDir, data.Epic.FileName), data)
}

// renderStatus writes the status.html page. Called only when the
// resolver returned a non-nil StatusData; the page summarises in-
// flight epics, open decisions, gaps, warnings, and recent activity.
func renderStatus(opts Options, tmpls *template.Template, data *StatusData) error {
	return executeToFile(tmpls, "status.tmpl", filepath.Join(opts.OutDir, "status.html"), data)
}

// renderMilestone writes one milestone page. The Manifest tab's
// AC list, Build tab's phase timelines, Tests tab's metrics, and
// Provenance tab's scopes/timeline all come from the resolver.
func renderMilestone(opts Options, tmpls *template.Template, resolver PageDataResolver, id string) error {
	data, err := resolver.MilestoneData(id)
	if err != nil {
		return fmt.Errorf("MilestoneData(%s): %w", id, err)
	}
	if data == nil || data.Milestone == nil {
		return fmt.Errorf("MilestoneData(%s) returned no Milestone ref", id)
	}
	return executeToFile(tmpls, "milestone.tmpl", filepath.Join(opts.OutDir, data.Milestone.FileName), data)
}

// renderEntity writes one gap / ADR / decision / contract page
// through the shared entity template. The four kinds have less
// structured rendering than epic/milestone — no AC tables, no
// phase timelines, no scope FSM — so a single template walking
// each `## ` body section in source order covers all of them.
// G35 fix.
func renderEntity(opts Options, tmpls *template.Template, resolver PageDataResolver, id string) error {
	data, err := resolver.EntityData(id)
	if err != nil {
		return fmt.Errorf("EntityData(%s): %w", id, err)
	}
	if data == nil || data.Entity == nil {
		return fmt.Errorf("EntityData(%s) returned no Entity ref", id)
	}
	return executeToFile(tmpls, "entity.tmpl", filepath.Join(opts.OutDir, data.Entity.FileName), data)
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
