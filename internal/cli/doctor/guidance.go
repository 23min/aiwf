package doctor

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/skills"
)

// appendGuidanceImportReport adds the CLAUDE.md guidance-import advisory
// to the doctor output (M-0165). Advisory only — never increments the
// problem count.
//
// It is emitted only when the materialized guidance fragment
// (.claude/aiwf-guidance.md) exists: if CLAUDE.md imports it, reports
// "ok"; if not, reports the `claudemd-guidance-unwired` advisory naming
// the exact fix command. When the fragment is absent there is nothing to
// wire, so nothing is reported.
func appendGuidanceImportReport(in []string, rootDir string) []string {
	guidancePath := filepath.Join(rootDir, filepath.FromSlash(skills.GuidanceFile))
	if _, err := os.Stat(guidancePath); err != nil {
		return in // fragment absent → nothing to wire (AC-2)
	}
	importLine := "@" + skills.GuidanceFile
	if claudeMd, err := os.ReadFile(filepath.Join(rootDir, "CLAUDE.md")); err == nil && strings.Contains(string(claudeMd), importLine) {
		return append(in, label("guidance:")+"ok (CLAUDE.md imports the aiwf guidance fragment)")
	}
	return append(in, label("guidance:")+"claudemd-guidance-unwired: advisory — "+skills.GuidanceFile+" exists but CLAUDE.md does not import it; run `aiwf init` to wire it")
}
