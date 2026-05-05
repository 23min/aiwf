package verb

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/manifest"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// On-collision modes for the Import verb.
const (
	OnCollisionFail   = "fail"
	OnCollisionSkip   = "skip"
	OnCollisionUpdate = "update"
)

// ImportOptions controls how Import handles edge cases.
type ImportOptions struct {
	// OnCollision selects behavior when a manifest entry has an
	// explicit id that already exists in the tree. Empty defaults
	// to "fail".
	OnCollision string
}

// ImportResult is what Import returns. Either Findings is non-empty
// (validation rejected the projection; nothing should be applied) or
// Plans is non-empty (one plan in single-commit mode, N plans in
// per-entity mode; the orchestrator applies each in order).
type ImportResult struct {
	Findings []check.Finding
	Plans    []*Plan
}

// plannedEntry is the per-entry working state Import threads through
// the pipeline: the source manifest entry, its resolved kind/id, and
// (for updates) the existing entity being replaced.
type plannedEntry struct {
	idx      int             // manifest index, for diagnostics
	entry    *manifest.Entry // source entry (read-only)
	kind     entity.Kind     // resolved kind
	id       string          // resolved id (post-allocation)
	isUpdate bool            // replace existing entity vs create new
	existing *entity.Entity  // when isUpdate, the entity being replaced
}

// Import processes a manifest against the existing tree and returns
// either findings (validation failed) or plans (validation passed,
// orchestrator should apply). Pure with respect to the filesystem;
// no writes happen here.
//
// The processing pipeline is:
//
//  1. resolve --on-collision against entries with explicit ids that
//     already exist in the tree (fail/skip/update).
//  2. detect intra-manifest duplicate explicit ids (always an error).
//  3. allocate auto-id entries from max(existing ∪ reserved) + 1.
//  4. resolve paths for every entry (new or updating an existing one).
//  5. project the tree (add or replace per entry) with PlannedFiles
//     populated for every OpWrite path.
//  6. run projectionFindings; abort with findings if any error level
//     issues are introduced.
//  7. assemble plans according to the manifest's commit mode.
func Import(ctx context.Context, t *tree.Tree, m *manifest.Manifest, actor string, opts ImportOptions) (*ImportResult, error) {
	_ = ctx
	if opts.OnCollision == "" {
		opts.OnCollision = OnCollisionFail
	}
	switch opts.OnCollision {
	case OnCollisionFail, OnCollisionSkip, OnCollisionUpdate:
		// ok
	default:
		return nil, fmt.Errorf("--on-collision: unknown value %q (want fail, skip, or update)", opts.OnCollision)
	}

	// Build an id → existing-entity map for collision detection.
	existing := make(map[string]*entity.Entity, len(t.Entities))
	for _, e := range t.Entities {
		existing[e.ID] = e
	}

	// Step 1+2: resolve explicit-id collisions and dedupe within
	// manifest. Build the set of entries to act on (after skips).
	var plannedEntries []plannedEntry
	reserved := make(map[entity.Kind]map[string]bool) // kind → set of reserved explicit ids
	var collisionFindings []check.Finding

	// Pre-scan manifest twice: first for explicit ids, then for auto.
	for i := range m.Entities {
		e := &m.Entities[i]
		k := entity.Kind(e.Kind)

		if e.IsAuto() {
			continue
		}
		// Intra-manifest duplicate?
		if reserved[k] == nil {
			reserved[k] = map[string]bool{}
		}
		if reserved[k][e.ID] {
			collisionFindings = append(collisionFindings, check.Finding{
				Code:     "import-duplicate-id",
				Severity: check.SeverityError,
				EntityID: e.ID,
				Message:  fmt.Sprintf("manifest declares id %s more than once", e.ID),
			})
			continue
		}
		reserved[k][e.ID] = true

		ex, hit := existing[e.ID]
		if !hit {
			plannedEntries = append(plannedEntries, plannedEntry{idx: i, entry: e, kind: k, id: e.ID})
			continue
		}
		switch opts.OnCollision {
		case OnCollisionFail:
			collisionFindings = append(collisionFindings, check.Finding{
				Code:     "import-collision",
				Severity: check.SeverityError,
				EntityID: e.ID,
				Path:     ex.Path,
				Message:  fmt.Sprintf("id %s already exists in tree (re-run with --on-collision=skip or --on-collision=update)", e.ID),
			})
		case OnCollisionSkip:
			// drop silently from action list
		case OnCollisionUpdate:
			plannedEntries = append(plannedEntries, plannedEntry{idx: i, entry: e, kind: k, id: e.ID, isUpdate: true, existing: ex})
		}
	}

	if len(collisionFindings) > 0 {
		return &ImportResult{Findings: collisionFindings}, nil
	}

	// Step 3: allocate auto ids. Per kind, max ID is taken over
	// (existing entities of that kind ∪ already-reserved explicit
	// ids in the manifest). New auto-allocated ids are also added to
	// reserved as we go so two `auto` entries for the same kind get
	// distinct ids.
	highest := computeHighestPerKind(t.Entities, reserved)
	for i := range m.Entities {
		e := &m.Entities[i]
		if !e.IsAuto() {
			continue
		}
		k := entity.Kind(e.Kind)
		highest[k]++
		id := formatID(k, highest[k])
		plannedEntries = append(plannedEntries, plannedEntry{idx: i, entry: e, kind: k, id: id})
	}

	// Step 4: build entities and resolve paths. Path resolution may
	// reference an epic that's also in the manifest (forward ref),
	// so build a transient lookup over (existing ∪ planned) ids.
	plannedByID := make(map[string]*plannedEntry, len(plannedEntries))
	for i := range plannedEntries {
		plannedByID[plannedEntries[i].id] = &plannedEntries[i]
	}

	entities := make([]*entity.Entity, len(plannedEntries))
	ops := make([]FileOp, 0, len(plannedEntries))
	plannedPaths := make([]string, 0, len(plannedEntries))
	for i := range plannedEntries {
		pe := &plannedEntries[i]
		ent, opErr := buildEntityFromEntry(pe, t, plannedByID)
		if opErr != nil {
			return nil, fmt.Errorf("manifest entry %d (%s/%s): %w", pe.idx, pe.kind, pe.id, opErr)
		}
		entities[i] = ent
		content, sErr := serializeFromEntry(ent, pe.entry.Body)
		if sErr != nil {
			return nil, fmt.Errorf("manifest entry %d (%s): %w", pe.idx, pe.id, sErr)
		}
		ops = append(ops, FileOp{Type: OpWrite, Path: ent.Path, Content: content})
		plannedPaths = append(plannedPaths, filepath.ToSlash(ent.Path))
	}

	// Step 5: project. New entries are appended; update entries
	// replace by id.
	proj := projectImport(t, plannedEntries, entities, plannedPaths)

	// Step 6: validate.
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return &ImportResult{Findings: fs}, nil
	}

	// Step 7: assemble plans.
	plans := buildImportPlans(m, plannedEntries, entities, ops, actor)
	return &ImportResult{Plans: plans}, nil
}

// computeHighestPerKind returns the largest existing id number per
// kind, including ids reserved by explicit-id manifest entries. Used
// as the starting point for auto-allocation.
func computeHighestPerKind(es []*entity.Entity, reserved map[entity.Kind]map[string]bool) map[entity.Kind]int {
	out := make(map[entity.Kind]int, len(entity.AllKinds()))
	for _, k := range entity.AllKinds() {
		out[k] = 0
	}
	for _, e := range es {
		if n := parseIDInt(e.Kind, e.ID); n > out[e.Kind] {
			out[e.Kind] = n
		}
	}
	for k, ids := range reserved {
		for id := range ids {
			if n := parseIDInt(k, id); n > out[k] {
				out[k] = n
			}
		}
	}
	return out
}

// parseIDInt is the package-local mirror of entity.parseIDNumber. The
// allocator helper there is unexported so we recreate it here.
func parseIDInt(k entity.Kind, id string) int {
	prefix := idPrefix(k)
	if !strings.HasPrefix(id, prefix) {
		return 0
	}
	rest := id[len(prefix):]
	n := 0
	for _, c := range rest {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func idPrefix(k entity.Kind) string {
	switch k {
	case entity.KindEpic:
		return "E-"
	case entity.KindMilestone:
		return "M-"
	case entity.KindADR:
		return "ADR-"
	case entity.KindGap:
		return "G-"
	case entity.KindDecision:
		return "D-"
	case entity.KindContract:
		return "C-"
	}
	return ""
}

// formatID builds an id string for kind k with the canonical pad
// width applied to n. Mirrors entity.AllocateID's formatting.
func formatID(k entity.Kind, n int) string {
	pad := canonicalPadFor(k)
	return fmt.Sprintf("%s%0*d", idPrefix(k), pad, n)
}

func canonicalPadFor(k entity.Kind) int {
	switch k {
	case entity.KindEpic:
		return 2
	case entity.KindADR:
		return 4
	default:
		return 3
	}
}

// buildEntityFromEntry materializes a manifest entry into an
// entity.Entity, with path resolved and the resolved id stamped onto
// the frontmatter. Forward refs to manifest-declared epics resolve
// via plannedByID.
func buildEntityFromEntry(pe *plannedEntry, t *tree.Tree, plannedByID map[string]*plannedEntry) (*entity.Entity, error) {
	// Frontmatter from manifest, with id forced to the resolved one.
	fm := make(map[string]any, len(pe.entry.Frontmatter)+1)
	for k, v := range pe.entry.Frontmatter {
		fm[k] = v
	}
	fm["id"] = pe.id

	// Marshal to YAML, then decode into Entity with KnownFields so
	// unrecognized frontmatter fields are caught.
	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling frontmatter: %w", err)
	}
	var ent entity.Entity
	dec := yaml.NewDecoder(bytes.NewReader(yamlBytes))
	dec.KnownFields(true)
	if err := dec.Decode(&ent); err != nil {
		return nil, fmt.Errorf("decoding frontmatter: %w", err)
	}
	ent.Kind = pe.kind

	// Path resolution.
	if pe.isUpdate {
		ent.Path = pe.existing.Path
		return &ent, nil
	}

	slug := entity.Slugify(ent.Title)
	if slug == "" {
		return nil, fmt.Errorf("title %q produces empty slug", ent.Title)
	}
	switch pe.kind {
	case entity.KindEpic:
		ent.Path = filepath.Join("work", "epics", pe.id+"-"+slug, "epic.md")
	case entity.KindMilestone:
		parent := ent.Parent
		if parent == "" {
			return nil, fmt.Errorf("milestone requires `parent`")
		}
		parentDir, perr := lookupEpicDir(parent, t, plannedByID)
		if perr != nil {
			return nil, perr
		}
		ent.Path = filepath.Join(parentDir, pe.id+"-"+slug+".md")
	case entity.KindADR:
		ent.Path = filepath.Join("docs", "adr", pe.id+"-"+slug+".md")
	case entity.KindGap:
		ent.Path = filepath.Join("work", "gaps", pe.id+"-"+slug+".md")
	case entity.KindDecision:
		ent.Path = filepath.Join("work", "decisions", pe.id+"-"+slug+".md")
	case entity.KindContract:
		ent.Path = filepath.Join("work", "contracts", pe.id+"-"+slug, "contract.md")
	default:
		return nil, fmt.Errorf("unsupported kind %q", pe.kind)
	}
	return &ent, nil
}

// lookupEpicDir resolves a parent epic id to its directory. The epic
// may already exist in the tree, or it may be declared earlier in the
// same manifest. The returned path is repo-relative.
func lookupEpicDir(epicID string, t *tree.Tree, plannedByID map[string]*plannedEntry) (string, error) {
	if ex := t.ByID(epicID); ex != nil {
		if ex.Kind != entity.KindEpic {
			return "", fmt.Errorf("parent %q is not an epic (it's a %s)", epicID, ex.Kind)
		}
		return filepath.Dir(ex.Path), nil
	}
	pe, ok := plannedByID[epicID]
	if !ok {
		return "", fmt.Errorf("parent %q does not exist in tree or manifest", epicID)
	}
	if pe.kind != entity.KindEpic {
		return "", fmt.Errorf("parent %q is not an epic (it's a %s)", epicID, pe.kind)
	}
	slug := entity.Slugify(asString(pe.entry.Frontmatter["title"]))
	if slug == "" {
		return "", fmt.Errorf("parent %q has empty title; cannot derive directory", epicID)
	}
	return filepath.Join("work", "epics", epicID+"-"+slug), nil
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// serializeFromEntry composes the file bytes for one entry. The body
// is written verbatim from the manifest; if it does not start with a
// newline, one is prepended so there is always whitespace between the
// closing `---` and the first body line.
func serializeFromEntry(ent *entity.Entity, body string) ([]byte, error) {
	bodyBytes := []byte(body)
	if len(bodyBytes) > 0 && bodyBytes[0] != '\n' {
		bodyBytes = append([]byte{'\n'}, bodyBytes...)
	}
	if len(bodyBytes) == 0 {
		bodyBytes = []byte("\n")
	}
	return entity.Serialize(ent, bodyBytes)
}

// projectImport returns a tree where every plannedEntry has been
// either appended (new) or replaced (update) by its corresponding
// entity.
func projectImport(t *tree.Tree, plans []plannedEntry, ents []*entity.Entity, paths []string) *tree.Tree {
	updateByID := make(map[string]*entity.Entity)
	var appends []*entity.Entity
	for i := range plans {
		if plans[i].isUpdate {
			updateByID[plans[i].id] = ents[i]
		} else {
			appends = append(appends, ents[i])
		}
	}

	proj := *t
	proj.Entities = make([]*entity.Entity, 0, len(t.Entities)+len(appends))
	for _, e := range t.Entities {
		if rep, ok := updateByID[e.ID]; ok {
			proj.Entities = append(proj.Entities, rep)
			continue
		}
		proj.Entities = append(proj.Entities, e)
	}
	proj.Entities = append(proj.Entities, appends...)
	proj.PlannedFiles = withPlanned(t.PlannedFiles, paths)
	return &proj
}

// buildImportPlans groups operations into plans according to the
// manifest's commit mode.
func buildImportPlans(m *manifest.Manifest, plans []plannedEntry, ents []*entity.Entity, ops []FileOp, actor string) []*Plan {
	mode := m.EffectiveCommitMode()
	if mode == manifest.CommitPerEntity {
		out := make([]*Plan, len(plans))
		for i := range plans {
			pe := &plans[i]
			subject := fmt.Sprintf("aiwf import %s %s %q", pe.kind, pe.id, ents[i].Title)
			if pe.isUpdate {
				subject = fmt.Sprintf("aiwf import update %s %s %q", pe.kind, pe.id, ents[i].Title)
			}
			out[i] = &Plan{
				Subject: subject,
				Trailers: []gitops.Trailer{
					{Key: gitops.TrailerVerb, Value: "add"},
					{Key: gitops.TrailerEntity, Value: pe.id},
					{Key: gitops.TrailerActor, Value: actor},
				},
				Ops: []FileOp{ops[i]},
			}
		}
		return out
	}

	subject := m.Commit.Message
	if subject == "" {
		subject = fmt.Sprintf("aiwf import %d entities", len(plans))
	}
	return []*Plan{{
		Subject: subject,
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "import"},
			{Key: gitops.TrailerActor, Value: actor},
		},
		Ops: ops,
	}}
}
