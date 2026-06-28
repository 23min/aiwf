package policies

import "strings"

// PolicyEmptyDiffCommitsCarryMarker asserts that every Go file in
// the verb package containing `AllowEmpty: true` (or `AllowEmpty
// = true`) also references one of the empty-commit markers somewhere
// in the same file. An empty-diff commit with no marker is
// indistinguishable from a no-op verb call to a reader of `git log`
// — exactly the audit-trail hole G24 closed.
//
// The accepted markers correspond to the deliberately-empty verb
// shapes the kernel ships:
//   - `aiwf authorize` (TrailerScope: opened/paused/resumed/ended)
//   - `aiwf <verb> --audit-only` (TrailerAuditOnly + reason)
//   - `aiwf acknowledge illegal` (TrailerForceFor: <historical-sha>)
//   - `aiwf acknowledge mistag` (M-0181/AC-6) — the per-entity sovereign
//     ack. Its target rides on `aiwf-entity` (shared with non-empty
//     entity verbs, so not a distinguishing empty-commit marker on its
//     own), so the distinguishing marker is its UNIQUE verb value
//     `acknowledge-mistag` — present only in this verb's file. Accepting
//     that literal keeps the check tight (no broadening via the shared
//     entity trailer) while recognizing the new shape.
//
// File scope (vs function scope) accounts for verbs that delegate
// trailer assembly to a helper in the same file (e.g.
// auditOnlyTrailers). A regression where AllowEmpty leaks into a
// new file with no marker reference still surfaces here.
func PolicyEmptyDiffCommitsCarryMarker(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/verb/") {
			continue
		}
		body := string(f.Contents)
		if !strings.Contains(body, "AllowEmpty: true") && !strings.Contains(body, "AllowEmpty = true") {
			continue
		}
		if strings.Contains(body, "TrailerScope") ||
			strings.Contains(body, "TrailerAuditOnly") ||
			strings.Contains(body, "TrailerForceFor") ||
			strings.Contains(body, `"acknowledge-mistag"`) {
			continue
		}
		offsets := FindAllOffsets(f.Contents, "AllowEmpty: true")
		if len(offsets) == 0 {
			offsets = FindAllOffsets(f.Contents, "AllowEmpty = true")
		}
		line := 1
		if len(offsets) > 0 {
			line = LineOf(f.Contents, offsets[0])
		}
		out = append(out, Violation{
			Policy: "empty-diff-commits-carry-marker",
			File:   f.Path,
			Line:   line,
			Detail: `file uses Plan.AllowEmpty = true but references none of the empty-commit markers (TrailerScope, TrailerAuditOnly, TrailerForceFor, or the "acknowledge-mistag" verb value); an unmarked empty-diff commit is indistinguishable from a no-op`,
		})
	}
	return out, nil
}
