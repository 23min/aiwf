package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// skillCheckPath is the repo-relative path of the aiwf-check skill whose
// Findings tables document the finding codes `aiwf check` emits.
const skillCheckPath = "internal/skills/embedded/aiwf-check/SKILL.md"

// findingDocOptOut lists emitted codes deliberately absent from the
// aiwf-check skill, each with a one-line rationale. Reserved for
// synthetic codes that never surface to a user; a real user-facing code
// belongs in the skill, not here.
var findingDocOptOut = map[string]string{
	"a-err":  "synthetic code used only in check-layer test fixtures; never surfaced to a user",
	"z-warn": "synthetic code used only in check-layer test fixtures; never surfaced to a user",
}

// skillDocRowPattern matches a Findings-table row whose first cell is a
// backticked finding code — `| `code` |` or `| `code/subcode` |`. The
// match is scoped to the table-row shape (leading pipe, backticked
// first cell) rather than a flat backtick grep, so an incidental
// backticked mention in prose does not count as a documented entry.
var skillDocRowPattern = regexp.MustCompile("(?m)^\\|\\s*`([a-z][a-z0-9-]*(?:/[a-z0-9-]+)?)`\\s*\\|")

// PolicyFindingCodesDocumentedInSkill asserts the aiwf-check skill body
// documents every finding code the check layer can emit. The emitted set
// is the union of string-constant codes and typed codespkg.Code{ID:…}
// descriptors used at Finding{} construction sites, enumerated through
// the shared emittedFindingCodeSites walker (the same source of truth
// PolicyFindingCodesHaveHints reads). A code emitted with no matching
// Findings-table entry — and not in the rationale-annotated opt-out —
// yields a violation, so a new finding code can't ship undocumented.
//
// The property is an aiwf-repo development invariant (it enumerates Go
// Code* declarations by AST), meaningless in a consumer tree where
// internal/check is absent and the skill is materialized rather than
// authored — so it lives here as a CI-tier Go policy test, not an
// aiwf check finding.
func PolicyFindingCodesDocumentedInSkill(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	documented := loadSkillDocumentedCodes(root)

	var out []Violation
	reported := map[string]struct{}{}
	for _, sc := range emittedFindingCodeSites(files) {
		// The aiwf-check skill documents codes `aiwf check` surfaces —
		// i.e. those emitted from the check layer. Verb-layer findings
		// (e.g. import-collision from `aiwf import`, slug-dropped-chars
		// from `aiwf add`) ride the shared emitted set (the hint policy
		// still requires them a hint) but are surfaced by their own verb,
		// not `aiwf check`, so they are out of this skill's scope.
		if !isCheckLayerFile(sc.File) {
			continue
		}
		if _, opt := findingDocOptOut[sc.Code]; opt {
			continue
		}
		key := sc.Code
		if sc.Subcode != "" {
			key = sc.Code + "/" + sc.Subcode
		}
		if _, ok := documented[key]; ok {
			continue
		}
		// Subcode-less fallback: a bare `code` row in the skill covers
		// every subcode of that code, mirroring the hint-table fallback.
		if sc.Subcode != "" {
			if _, ok := documented[sc.Code]; ok {
				continue
			}
		}
		if _, dup := reported[key]; dup {
			continue
		}
		reported[key] = struct{}{}
		out = append(out, Violation{
			Policy: "finding-codes-documented-in-skill",
			File:   sc.File,
			Line:   sc.Line,
			Detail: "finding code " + strconv.Quote(key) + " is emitted here but has no entry in the aiwf-check skill (" +
				skillCheckPath + "); add a row to its Findings table, or opt it out with a rationale in findingDocOptOut",
		})
	}
	return out, nil
}

// isCheckLayerFile reports whether a repo-relative path is part of the
// check layer whose finding codes `aiwf check` surfaces.
func isCheckLayerFile(path string) bool {
	return strings.HasPrefix(path, "internal/check/") ||
		strings.HasPrefix(path, "internal/cli/check/")
}

// loadSkillDocumentedCodes reads the aiwf-check skill and returns the set
// of finding codes documented as Findings-table rows. The skill is
// markdown, so it is read directly rather than via WalkGoFiles (which
// returns only .go files). Falls back to an empty set if the file isn't
// present, so the policy fires loudly rather than silently passing when
// the skill is missing.
func loadSkillDocumentedCodes(root string) map[string]struct{} {
	docs := map[string]struct{}{}
	data, err := os.ReadFile(filepath.Join(root, skillCheckPath))
	if err != nil {
		return docs
	}
	for _, m := range skillDocRowPattern.FindAllSubmatch(data, -1) {
		docs[string(m[1])] = struct{}{}
	}
	return docs
}
