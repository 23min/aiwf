package verb

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/23min/ai-workflow-v2/tools/internal/aiwfyaml"
	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/config"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// AddOptions carries the per-kind extra arguments to Add. Only the
// fields relevant to the kind are read; others are ignored.
type AddOptions struct {
	// Milestone: id of the parent epic. Required.
	EpicID string
	// Gap: optional reference to the milestone or epic where the gap
	// was discovered.
	DiscoveredIn string
	// Decision: optional list of entity ids the decision relates to.
	RelatesTo []string
	// Contract: optional list of ADR ids motivating the contract.
	LinkedADRs []string
	// Contract: when all three of BindValidator/BindSchema/BindFixtures
	// are non-empty, Add atomically appends the binding to
	// aiwf.yaml.contracts.entries[] in the same commit. Partial
	// triplets are an error; the verb is all-or-nothing on the bind.
	BindValidator string
	BindSchema    string
	BindFixtures  string
	// Contract: when atomic-bind is requested, AiwfDoc must be the
	// editable aiwf.yaml document and AiwfContracts the parsed
	// contracts: block (nil ok if absent). The CLI dispatcher loads
	// these only when the bind flags are present.
	AiwfDoc       *aiwfyaml.Doc
	AiwfContracts *aiwfyaml.Contracts
}

// Add creates a new entity of the given kind. Allocates the next free
// id, builds the entity, projects it onto the tree, runs `aiwf check`
// against the projection, and either returns findings (no changes
// staged) or a Plan that the orchestrator applies.
//
// For contracts with all three Bind* options set, Add additionally
// splices the binding into aiwf.yaml.contracts.entries[] and the
// returned Plan carries a second OpWrite so the entity creation and
// the binding land as a single commit.
//
// Returns a Go error only when arguments are malformed (missing
// required option, parent epic not found, contract-only flag on
// non-contract kind, partial bind triplet). Tree-integrity issues
// arising from the addition are returned as findings, not errors.
func Add(ctx context.Context, t *tree.Tree, kind entity.Kind, title, actor string, opts AddOptions) (*Result, error) {
	_ = ctx // reserved for future IO; verbs are currently pure-projection and IO happens in Apply
	if title == "" {
		return nil, fmt.Errorf("--title is required")
	}
	if err := validateAddOptsForKind(kind, opts); err != nil {
		return nil, err
	}
	id := entity.AllocateID(kind, t.Entities)
	slug, dropped := entity.SlugifyDetailed(title)
	if slug == "" {
		return nil, fmt.Errorf("title %q produces an empty slug; try a different title", title)
	}
	var slugNotices []check.Finding
	if len(dropped) > 0 {
		slugNotices = append(slugNotices, slugDroppedFinding(id, title, slug, dropped))
	}

	path, err := newEntityPath(t, kind, id, slug, opts)
	if err != nil {
		return nil, err
	}

	e := &entity.Entity{
		Kind:   kind,
		ID:     id,
		Title:  title,
		Status: initialStatus(kind),
		Path:   path,
	}
	applyAddOpts(e, opts)

	ops, err := buildAddOps(e)
	if err != nil {
		return nil, err
	}

	proj := projectAdd(t, e)
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	if kind == entity.KindContract && opts.BindValidator != "" {
		bindOps, err := atomicContractBind(id, opts)
		if err != nil {
			return nil, err
		}
		ops = append(ops, bindOps...)
	}

	subject := fmt.Sprintf("aiwf add %s %s %q", kind, id, title)
	return &Result{
		Findings: slugNotices,
		Plan: &Plan{
			Subject: subject,
			Trailers: []gitops.Trailer{
				{Key: "aiwf-verb", Value: "add"},
				{Key: "aiwf-entity", Value: id},
				{Key: "aiwf-actor", Value: actor},
			},
			Ops: ops,
		},
	}, nil
}

// slugDroppedFinding builds the warning surfaced when SlugifyDetailed
// drops non-ASCII runes from the title. It travels with the
// successful plan so the user sees the slug they actually got and
// can rename later if needed.
func slugDroppedFinding(id, title, slug string, dropped []rune) check.Finding {
	return check.Finding{
		Code:     "slug-dropped-chars",
		Severity: check.SeverityWarning,
		EntityID: id,
		Message: fmt.Sprintf(
			"title %q contains non-ASCII characters that the slug omits (%s); slug is %q",
			title, runeListString(dropped), slug,
		),
	}
}

// runeListString renders a list of runes as a comma-separated,
// quoted, deduplicated list for inclusion in a finding message.
func runeListString(rs []rune) string {
	seen := make(map[rune]bool, len(rs))
	var out []string
	for _, r := range rs {
		if seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, fmt.Sprintf("%q", string(r)))
	}
	result := ""
	for i, s := range out {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

// validateAddOptsForKind enforces that contract-only flags
// (LinkedADRs, BindValidator/Schema/Fixtures) are not passed for
// other kinds, and that the bind triplet is all-or-nothing.
func validateAddOptsForKind(kind entity.Kind, opts AddOptions) error {
	if kind != entity.KindContract {
		if len(opts.LinkedADRs) > 0 {
			return fmt.Errorf("--linked-adr is only valid for kind=contract")
		}
		if opts.BindValidator != "" || opts.BindSchema != "" || opts.BindFixtures != "" {
			return fmt.Errorf("--validator, --schema, --fixtures are only valid for kind=contract")
		}
		return nil
	}
	bindCount := 0
	for _, s := range []string{opts.BindValidator, opts.BindSchema, opts.BindFixtures} {
		if s != "" {
			bindCount++
		}
	}
	if bindCount != 0 && bindCount != 3 {
		return fmt.Errorf("contract bind requires all of --validator, --schema, and --fixtures (got %d/3)", bindCount)
	}
	if bindCount == 3 && opts.AiwfDoc == nil {
		return fmt.Errorf("contract add+bind requires aiwf.yaml; run 'aiwf init' first")
	}
	return nil
}

// atomicContractBind splices the new binding into aiwf.yaml's
// contracts: block and returns the OpWrite that lands the spliced
// bytes. The verb has already validated that the entity will be
// created with id; here we only check that the validator is declared
// (the standard ContractBind pre-write rule).
func atomicContractBind(id string, opts AddOptions) ([]FileOp, error) {
	next := cloneContracts(opts.AiwfContracts)
	if _, ok := next.Validators[opts.BindValidator]; !ok {
		return nil, fmt.Errorf("validator %q not declared; install via 'aiwf contract recipe install %s' or 'aiwf contract recipe install --from <path>'", opts.BindValidator, opts.BindValidator)
	}
	for _, en := range next.Entries {
		if en.ID == id {
			return nil, fmt.Errorf("binding for %s already exists; this is a freshly-allocated id, indicating a programming error in Add", id)
		}
	}
	next.Entries = append(next.Entries, aiwfyaml.Entry{
		ID:        id,
		Validator: opts.BindValidator,
		Schema:    opts.BindSchema,
		Fixtures:  opts.BindFixtures,
	})
	if err := opts.AiwfDoc.SetContracts(next); err != nil {
		return nil, fmt.Errorf("updating aiwf.yaml: %w", err)
	}
	return []FileOp{{Type: OpWrite, Path: config.FileName, Content: opts.AiwfDoc.Bytes()}}, nil
}

// newEntityPath computes the relative path the new entity will live at.
func newEntityPath(t *tree.Tree, kind entity.Kind, id, slug string, opts AddOptions) (string, error) {
	switch kind {
	case entity.KindEpic:
		return filepath.Join("work", "epics", id+"-"+slug, "epic.md"), nil
	case entity.KindMilestone:
		if opts.EpicID == "" {
			return "", fmt.Errorf("milestone requires --epic <epic-id>")
		}
		epic := t.ByID(opts.EpicID)
		if epic == nil {
			return "", fmt.Errorf("--epic %q does not exist", opts.EpicID)
		}
		if epic.Kind != entity.KindEpic {
			return "", fmt.Errorf("--epic %q is not an epic (it's a %s)", opts.EpicID, epic.Kind)
		}
		epicDir := filepath.Dir(epic.Path)
		return filepath.Join(epicDir, id+"-"+slug+".md"), nil
	case entity.KindADR:
		return filepath.Join("docs", "adr", id+"-"+slug+".md"), nil
	case entity.KindGap:
		return filepath.Join("work", "gaps", id+"-"+slug+".md"), nil
	case entity.KindDecision:
		return filepath.Join("work", "decisions", id+"-"+slug+".md"), nil
	case entity.KindContract:
		return filepath.Join("work", "contracts", id+"-"+slug, "contract.md"), nil
	}
	return "", fmt.Errorf("unsupported kind %q", kind)
}

// applyAddOpts copies kind-specific options from opts onto the entity.
func applyAddOpts(e *entity.Entity, opts AddOptions) {
	switch e.Kind {
	case entity.KindMilestone:
		e.Parent = opts.EpicID
	case entity.KindGap:
		if opts.DiscoveredIn != "" {
			e.DiscoveredIn = opts.DiscoveredIn
		}
	case entity.KindDecision:
		if len(opts.RelatesTo) > 0 {
			e.RelatesTo = append([]string(nil), opts.RelatesTo...)
		}
	case entity.KindContract:
		if len(opts.LinkedADRs) > 0 {
			e.LinkedADRs = append([]string(nil), opts.LinkedADRs...)
		}
	}
}

// buildAddOps composes the file operations needed to land the new
// entity: a single OpWrite of the entity file with serialized
// frontmatter and the kind's body template.
func buildAddOps(e *entity.Entity) ([]FileOp, error) {
	body := entity.BodyTemplate(e.Kind)
	content, err := entity.Serialize(e, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", e.ID, err)
	}
	return []FileOp{{Type: OpWrite, Path: e.Path, Content: content}}, nil
}
