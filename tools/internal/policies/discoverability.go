package policies

import (
	"os"
	"path/filepath"
	"strings"
)

// PolicyFindingCodesAreDiscoverable asserts that every provenance
// finding code defined in tools/internal/check/provenance.go
// appears in either the aiwf-check embedded skill OR the binary's
// printHelp output. The CLAUDE.md "kernel functionality must be
// AI-discoverable" principle: a finding code that only exists in
// source is, by definition, undocumented.
//
// We scope the check to provenance-* codes because those are the
// I2.5 surface; pre-existing finding codes are already covered by
// the legacy aiwf-check skill content. Extending the policy to
// "every finding code anywhere" is a fine future tightening.
func PolicyFindingCodesAreDiscoverable(root string) ([]Violation, error) {
	provenanceCodes, err := readProvenanceCodes(root)
	if err != nil {
		return nil, err
	}
	skillPath := filepath.Join(root, "tools", "internal", "skills", "embedded", "aiwf-check", "SKILL.md")
	skillBytes, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, err
	}
	helpPath := filepath.Join(root, "tools", "cmd", "aiwf", "main.go")
	helpBytes, err := os.ReadFile(helpPath)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, code := range provenanceCodes {
		if strings.Contains(string(skillBytes), code) {
			continue
		}
		if strings.Contains(string(helpBytes), code) {
			continue
		}
		out = append(out, Violation{
			Policy: "finding-codes-are-discoverable",
			File:   "tools/internal/skills/embedded/aiwf-check/SKILL.md",
			Detail: code + " appears in tools/internal/check/provenance.go but is not mentioned in the aiwf-check skill or printHelp",
		})
	}
	return out, nil
}

// readProvenanceCodes reads tools/internal/check/provenance.go and
// returns every string-valued constant whose value starts with
// "provenance-". Defensive: returns an empty slice if the file is
// not where we expect.
func readProvenanceCodes(root string) ([]string, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	consts := loadCheckCodeConstants(files)
	var out []string
	for _, v := range consts {
		if strings.HasPrefix(v, "provenance-") {
			out = append(out, v)
		}
	}
	return out, nil
}
