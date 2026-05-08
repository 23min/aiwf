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
