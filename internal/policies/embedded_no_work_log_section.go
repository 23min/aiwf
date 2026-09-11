package policies

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// workLogViolation builds this policy's violations from one literal, so the
// firing-fixture inventory sees a single policy id and every report carries a
// line.
func workLogViolation(rel string, line int, format string, args ...any) Violation {
	return Violation{
		Policy: "embedded-no-work-log-section",
		File:   rel,
		Line:   line,
		Detail: fmt.Sprintf(format, args...),
	}
}

var (
	// workLogAsSection matches the section named as a section: a heading, a
	// backticked heading, or the JSON body key `aiwf show` derives from it.
	// No sentence about a consumer's own habits produces either shape.
	workLogAsSection = regexp.MustCompile(`(?i)##\s+work[ _-]log\b|work_log\b`)

	// workLogMention matches the phrase in any form.
	workLogMention = regexp.MustCompile(`(?i)work[ _-]log\b`)
)

// PolicyEmbeddedNoWorkLogSection asserts that no surface aiwf ships names a
// `## Work log` section of the milestone spec. The section is retired: what it
// held is `acs[]`, the TDD phase ladder, and `aiwf history M-NNNN/AC-<N>`,
// which lists the implementation commit by its entity trailer.
//
// The ban is what makes the retirement hold. Removing the section from the
// template, the rituals and the agent cards leaves nothing asserting it stays
// removed — an exact revert of that removal passes every other check in this
// repo, and three single-line edits each restore the convention on their own: a
// heading placed above the template's ownership map, an instruction bullet in
// either milestone ritual, or any mention in the engineering-skill tree the
// section-ownership policies do not scan.
//
// Two rules, applied per line to every file under internal/skills/embedded*:
//
//  1. No `## Work log` heading and no `work_log` body key, anywhere and
//     unconditionally. Naming the section as a section is the reintroduction.
//  2. Any other "work log" mention must be conditional on the reader's own
//     project, which is mechanically a line carrying "if the project". A skill
//     asking whether a project keeps its own work log alongside a diff passes;
//     an instruction to fill one in an aiwf milestone spec does not. The escape
//     is the conditional rather than a bare mention of a project, because
//     shipped prose is consumer-scoped throughout and names one constantly — an
//     instruction reintroducing the section reads naturally as "append a work
//     log entry to the project's spec", which the bare form would clear.
//
// Scope is every embedded tree rather than the ritual snapshot alone, because
// one of the stale references this ban was built from sat outside that
// snapshot: the `aiwf show` verb skill's body-key list and its worked example
// both named the section.
//
// The mirror of PolicyEmbeddedRitualsNoRetiredTrackingDoc, which bans the
// retired v1 tracking-doc convention on the same reasoning: a retired
// convention an agent can still read somewhere is one it will still follow, and
// which convention it follows depends on which artifact it happened to open.
func PolicyEmbeddedNoWorkLogSection(root string) ([]Violation, error) {
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
			switch {
			case workLogAsSection.MatchString(line):
				vs = append(vs, workLogViolation(rel, i+1,
					"a shipped surface names the retired `## Work log` section of the milestone spec; what it held is `acs[]`, the TDD phase ladder, and `aiwf history M-NNNN/AC-<N>`, which lists the implementation commit by its entity trailer"))
			case workLogMention.MatchString(line) && !strings.Contains(strings.ToLower(line), "if the project"):
				vs = append(vs, workLogViolation(rel, i+1,
					"\"work log\" mention that is not conditional on the reader's own project — the milestone spec's `## Work log` section is retired, so a mention here reads as an instruction to fill one; a sentence about a consumer's own habit opens \"if the project\""))
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", skillsDir, err)
	}
	return vs, nil
}
