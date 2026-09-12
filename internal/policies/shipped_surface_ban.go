package policies

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// lineBan is one rule a shipped-surface ban applies to a line: the pattern
// naming the banned shape, the substring that clears a match, and what to
// report where nothing clears it.
//
// escape is matched case-insensitively as a substring of the whole line, in
// whatever case it is written here; empty means the rule has no escape and
// every match reports. An escape clears the rule carrying it and no other, so
// the search continues to the rules after it. Order within a []lineBan is
// significant — see runShippedBan.
type lineBan struct {
	pattern *regexp.Regexp
	escape  string
	detail  string
}

// runShippedBan applies bans to every line of every file aiwf ships to a
// consumer — the trees under internal/skills/ whose names begin "embedded" —
// and reports at most one violation per line: the first rule in bans that
// catches it. Order therefore decides how a line is described, not whether it
// reports, so a line naming a banned shape outright is reported as that rather
// than as the weaker phrase which also matches it.
//
// Scope is every embedded tree rather than any one of them. A retired
// convention is followed from wherever an agent reads it, and which artifact
// that is depends only on which one it happened to open, so a ban that reads
// one tree leaves the same instruction live in the others. Siblings under
// internal/skills/ are aiwf's own source rather than surfaces a consumer
// receives: the walk descends into the embedded trees alone and reads no file
// sitting directly in internal/skills/.
//
// A tree it cannot read is an error rather than a clean verdict. A ban
// reporting nothing for bytes it never saw is worse than no ban, because the
// clean verdict is what stops the next reader looking.
//
// newViolation is a constructor rather than a value so the caller's
// `Policy: "<id>"` literal is evaluated once per firing rather than once per
// call. PolicyFiringFixturePresence decides a policy is unproven by finding
// that line uncovered, which it can only do while the line runs on a firing
// and not on a clean tree; a Violation passed here as a plain value would be
// built on every call and leave the line permanently lit, so the gate would
// pass for these two policies whether or not any test drove them.
//
// The literal also sits in the policy's own file rather than here, so that
// gate attributes the line to the policy it belongs to. That placement is not
// PolicyViolationPolicyIDLiteral's doing — it requires only that the field be
// a string literal, anywhere under internal/policies.
//
// TestRunShippedBan_BuildsTheViolationOncePerFiring holds the per-firing half:
// hoisting the constructor call out of the loop is otherwise invisible.
func runShippedBan(root string, bans []lineBan, newViolation func() Violation) ([]Violation, error) {
	skillsDir := filepath.Join(root, "internal", "skills")
	var vs []Violation
	err := filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !strings.HasPrefix(d.Name(), "embedded") && filepath.Dir(path) == skillsDir {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Dir(path) == skillsDir {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil { //coverage:ignore defensive: the walk starts at a path built from root, so every entry it yields is already rooted there
			return err
		}
		rel = filepath.ToSlash(rel)
		for i, line := range strings.Split(string(b), "\n") {
			ban, ok := firstBan(bans, line)
			if !ok {
				continue
			}
			v := newViolation()
			v.File, v.Line, v.Detail = rel, i+1, ban.detail
			vs = append(vs, v)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", skillsDir, err)
	}
	return vs, nil
}

// firstBan returns the first rule in bans that catches line — one whose
// pattern matches and whose escape, if it has one, is absent.
func firstBan(bans []lineBan, line string) (lineBan, bool) {
	lower := strings.ToLower(line)
	for _, ban := range bans {
		if !ban.pattern.MatchString(line) {
			continue
		}
		if ban.escape != "" && strings.Contains(lower, strings.ToLower(ban.escape)) {
			continue
		}
		return ban, true
	}
	return lineBan{}, false
}
