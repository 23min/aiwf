package policies

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PolicyEmbeddedRitualsNoRetiredTrackingDoc asserts that the embedded
// ritual snapshot (internal/skills/embedded-rituals/, the single
// source of truth per ADR-0016) does not instruct the retired v1
// separate tracking-doc convention.
//
// Two rules, applied per line to every file under the snapshot:
//
//  1. No `work/tracking/` path reference, anywhere. The directory is
//     retired; an agent following an instruction that names it
//     recreates it in the consumer repo, and nothing kernel-side
//     objects (verified in G-0245 — a stray file under
//     `work/tracking/` produces zero `aiwf check` findings).
//  2. Any "tracking doc" / "tracking-doc" mention must carry "v1" on
//     the same line — the mechanical definition of "explicit
//     v1-historical context." Retirement statements ("The v1 separate
//     tracking doc is gone") pass; instruction-shaped phrasing
//     ("finalize the tracking doc") fails.
//
// Why this exists: the vendored snapshot shipped self-contradictory
// at M-0148 — five artifacts instructed the retired convention while
// three others declared it gone. Which convention an agent followed
// depended on which artifact it happened to read — the "guarantee
// depends on the LLM's behavior" failure class. G-0224 was the same
// defect class at nit level; G-0245 is the recurrence that crossed
// the stated threshold for a mechanical chokepoint.
//
// Pins G-0245 fix-shape item 2.
func PolicyEmbeddedRitualsNoRetiredTrackingDoc(root string) ([]Violation, error) {
	snapshotRel := filepath.Join("internal", "skills", "embedded-rituals")
	trackingDoc := regexp.MustCompile(`(?i)tracking[ -]docs?\b`)

	var vs []Violation
	err := filepath.WalkDir(filepath.Join(root, snapshotRel), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		for i, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, "work/tracking/") {
				vs = append(vs, Violation{
					Policy: "embedded-rituals-no-retired-tracking-doc",
					File:   rel,
					Line:   i + 1,
					Detail: "embedded ritual content must not reference the retired `work/tracking/` directory (G-0245); point at the milestone spec's frontmatter `acs[]` / `## Work log` / `## Decisions made during implementation` instead",
				})
				continue
			}
			if trackingDoc.MatchString(line) && !strings.Contains(strings.ToLower(line), "v1") {
				vs = append(vs, Violation{
					Policy: "embedded-rituals-no-retired-tracking-doc",
					File:   rel,
					Line:   i + 1,
					Detail: "\"tracking doc\" mention without \"v1\" on the same line — the retired convention may only appear in explicit v1-historical context (G-0245)",
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return vs, nil
}
