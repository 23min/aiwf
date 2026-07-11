package policies

import (
	"strings"
)

// historyRewriteSubstrings are the git invocations that rewrite or
// destroy history. None should appear in production code paths;
// the kernel's audit-trail guarantee depends on history being
// append-only. Tests legitimately use a few of these (e.g. force
// pushes against a temp upstream); they are exempt.
var historyRewriteSubstrings = []string{
	`"push", "--force"`,
	`"push", "-f"`,
	`"--force-with-lease"`,
	`"reset", "--hard"`,
	`"rebase"`,
	`"commit", "--amend"`,
	`"--amend"`,
	`"filter-branch"`,
	`"filter-repo"`,
	`"replace"`,
}

// historyRewriteAllowlist exempts files by repo-relative forward-slash
// path whose history-rewrite invocations target a scenario's own
// disposable, throwaway fixture repo — never this repo's own audit
// trail — with a one-line rationale kept alongside the exemption.
var historyRewriteAllowlist = map[string]string{
	// M-0243/AC-4's force-override-durability scenario rebases a
	// disposable fixture repo (dropping just an acknowledge-illegal
	// commit while keeping the originally-flagged one reachable) to
	// reproduce the same reachability effect a force-push produces —
	// against a repo this scenario itself creates and discards, never
	// this repo's own history.
	"internal/stresstest/force_override_durability.go": "rebases a disposable fixture repo the scenario itself creates, never this repo's own audit trail",
}

// PolicyNoHistoryRewrites flags any non-test, non-policies-package
// Go file that contains a substring matching a known
// history-rewriting git invocation. The list is conservative — a
// match in a code-comment or unrelated string would also fire,
// which we accept as a forcing function: ambiguous spellings get
// renamed.
//
// Why a substring match rather than AST? The shape we care about
// is the literal `git ... <flag>` invocation; once it appears in
// the source, it's discoverable, and review knows to reject it.
// AST analysis would have to track every wrapper that proxies to
// exec.Command anyway.
func PolicyNoHistoryRewrites(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, f := range files {
		if _, allowed := historyRewriteAllowlist[f.Path]; allowed {
			continue
		}
		for _, sub := range historyRewriteSubstrings {
			offsets := FindAllOffsets(f.Contents, sub)
			for _, off := range offsets {
				out = append(out, Violation{
					Policy: "no-history-rewrites",
					File:   f.Path,
					Line:   LineOf(f.Contents, off),
					Detail: "production code references " + strings.Trim(sub, "\"") +
						"; the kernel's audit trail must be append-only",
				})
			}
		}
	}
	return out, nil
}
