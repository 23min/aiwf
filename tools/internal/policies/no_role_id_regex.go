package policies

import (
	"regexp"
	"strings"
)

// rolePatternSubstrings detect ad-hoc regex constructions that
// match the `<role>/<id>` shape — duplicating gitops.roleIDPattern
// in another file. The kernel's actor / principal validation
// lives in gitops.ValidateTrailer; replicating the regex elsewhere
// invites drift (e.g., one site allows underscores, another
// doesn't).
//
// We flag conservative shapes that almost certainly mean
// "role/id". False positives (e.g., a regex matching paths) are
// possible — the policy is advisory enough that whitelisting via
// rename is acceptable.
var rolePatternSubstrings = []*regexp.Regexp{
	regexp.MustCompile(`regexp\.MustCompile\([^)]*\^\[\^/\]\+/\[\^/\]\+`),
	regexp.MustCompile(`regexp\.MustCompile\([^)]*\^\[\^\\\\s/\]\+/\[\^\\\\s/\]\+`),
	regexp.MustCompile(`regexp\.MustCompile\([^)]*\(\?:human\|ai\|bot\)`),
}

// PolicyRoleIDRegexCentralized flags any ad-hoc regex constructed
// outside the gitops package that matches the role/id shape. The
// canonical place is gitops; everywhere else should call a helper
// or reuse the constant.
func PolicyRoleIDRegexCentralized(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, f := range files {
		if strings.HasPrefix(f.Path, "tools/internal/gitops/") {
			continue
		}
		for _, pat := range rolePatternSubstrings {
			matches := pat.FindAllIndex(f.Contents, -1)
			for _, m := range matches {
				out = append(out, Violation{
					Policy: "role-id-regex-centralized",
					File:   f.Path,
					Line:   LineOf(f.Contents, m[0]),
					Detail: "ad-hoc role/id regex; route validation through gitops.ValidateTrailer or a shared helper",
				})
			}
		}
	}
	return out, nil
}
