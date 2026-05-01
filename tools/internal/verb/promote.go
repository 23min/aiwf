package verb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
// the what. Empty reason produces a body-less commit.
//
// force=true relaxes the FSM transition rule so any-to-any moves are
// permitted; coherence (closed-set membership of the target status,
// id format, ref resolution) still runs via projection findings, so
// promoting to an unknown status is still rejected. Force requires a
// non-empty reason; the caller (cmd dispatcher) is responsible for
// enforcing that. When force is set, the standard trailers gain
// `aiwf-force: <reason>` so the audit trail is queryable.
//
// Returns a Go error for "couldn't even start": id not found, illegal
// transition (when not forced). Tree-level findings caused by the
// change are returned as a Result with non-empty Findings.
func Promote(ctx context.Context, t *tree.Tree, id, newStatus, actor, reason string, force bool) (*Result, error) {
	_ = ctx
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	if !force {
		if err := entity.ValidateTransition(e.Kind, e.Status, newStatus); err != nil {
			return nil, err
		}
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
		Subject:  subject,
		Body:     reason,
		Trailers: transitionTrailers("promote", id, actor, reason, force),
		Ops:      []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
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
//
// force=true emits an `aiwf-force: <reason>` trailer alongside the
// standard ones so the cancellation is auditable as a forced action.
// Cancel has no FSM transition rule to relax (it always sets status to
// the kind's terminal-cancel target), so force is purely an audit
// signal here. The "already at target" guard remains in place even
// under force — there is no diff to write. Force requires a non-empty
// reason; the caller is responsible for enforcing that.
func Cancel(ctx context.Context, t *tree.Tree, id, actor, reason string, force bool) (*Result, error) {
	_ = ctx
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
		Subject:  subject,
		Body:     reason,
		Trailers: transitionTrailers("cancel", id, actor, reason, force),
		Ops:      []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	}), nil
}

// transitionTrailers builds the standard trailer block for a status-
// changing verb. The `aiwf-force` trailer is appended only when force
// is true; its value is the trimmed reason (which the dispatcher has
// already verified is non-empty). The standard trailers come first so
// downstream readers (`aiwf history`) find them in a stable order.
func transitionTrailers(verbName, id, actor, reason string, force bool) []gitops.Trailer {
	trailers := []gitops.Trailer{
		{Key: "aiwf-verb", Value: verbName},
		{Key: "aiwf-entity", Value: id},
		{Key: "aiwf-actor", Value: actor},
	}
	if force {
		trailers = append(trailers, gitops.Trailer{Key: "aiwf-force", Value: strings.TrimSpace(reason)})
	}
	return trailers
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
