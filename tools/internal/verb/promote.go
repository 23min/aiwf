package verb

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// Promote advances an entity's status. The transition is validated
// against the kind's FSM (entity.ValidateTransition) before any
// projection runs, so unknown statuses and illegal jumps are rejected
// with a clear error rather than as a `status-valid` finding.
//
// reason is optional free-form prose explaining *why* the transition
// happens. When non-empty, it lands in the commit body (between
// subject and trailers) so future readers can see the why, not just
// the what. Empty reason produces a body-less commit (today's shape).
//
// Returns a Go error for "couldn't even start": id not found, illegal
// transition. Tree-level findings caused by the change are returned as
// a Result with non-empty Findings.
func Promote(t *tree.Tree, id, newStatus, actor, reason string) (*Result, error) {
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	if err := entity.ValidateTransition(e.Kind, e.Status, newStatus); err != nil {
		return nil, err
	}

	modified := *e
	modified.Status = newStatus

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}

	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	subject := fmt.Sprintf("aiwf promote %s %s -> %s", id, e.Status, newStatus)
	return plan(&Plan{
		Subject: subject,
		Body:    reason,
		Trailers: []gitops.Trailer{
			{Key: "aiwf-verb", Value: "promote"},
			{Key: "aiwf-entity", Value: id},
			{Key: "aiwf-actor", Value: actor},
		},
		Ops: []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	}), nil
}

// Cancel promotes an entity to its kind's terminal-cancel status —
// `cancelled` for epic/milestone, `rejected` for adr/decision,
// `wontfix` for gap, `retired` for contract. Errors when the entity is
// already in a terminal state or when the kind is unknown.
//
// reason is optional free-form prose; when non-empty, it lands in the
// commit body so the cancellation's "why" is preserved for future
// readers. Empty reason matches today's body-less behaviour.
func Cancel(t *tree.Tree, id, actor, reason string) (*Result, error) {
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	target := entity.CancelTarget(e.Kind)
	if target == "" {
		return nil, fmt.Errorf("kind %q has no cancel target", e.Kind)
	}
	if e.Status == target {
		return nil, fmt.Errorf("%s is already %s", id, target)
	}

	modified := *e
	modified.Status = target

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}

	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	subject := fmt.Sprintf("aiwf cancel %s -> %s", id, target)
	return plan(&Plan{
		Subject: subject,
		Body:    reason,
		Trailers: []gitops.Trailer{
			{Key: "aiwf-verb", Value: "cancel"},
			{Key: "aiwf-entity", Value: id},
			{Key: "aiwf-actor", Value: actor},
		},
		Ops: []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	}), nil
}

// readBody reads the body bytes from an existing entity file. Returns
// an empty body if the file lacks frontmatter (a freshly-edited file
// with no closing ---). Used by promote/cancel/reallocate to preserve
// body prose during frontmatter rewrites.
func readBody(root, relPath string) ([]byte, error) {
	full := filepath.Join(root, relPath)
	content, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", relPath, err)
	}
	_, body, ok := entity.Split(content)
	if !ok {
		return []byte{}, nil
	}
	return body, nil
}
