package doctor

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/skills"
)

// appendGuidanceImportReport adds the CLAUDE.md guidance-import advisory
// to the doctor output (M-0165). Advisory only — never increments the
// problem count.
//
// It is emitted only when guidance wiring is enabled (the default; opt
// out via aiwf.yaml `guidance.wire_claudemd: false`) AND the materialized
// fragment (.claude/aiwf-guidance.md) exists: if CLAUDE.md imports it,
// reports "ok"; if not, reports `claudemd-guidance-unwired` naming the
// exact fix (`aiwf update`, which self-heals the import per ADR-0018).
// When the consumer opted out, or the fragment is absent, nothing is
// reported.
func appendGuidanceImportReport(in []string, problemsIn []Problem, rootDir string) (lines []string, problems []Problem) {
	problems = problemsIn
	// Respect the opt-out: a consumer who disabled wiring should not be nagged.
	if cfg, err := config.Load(rootDir); err == nil && !cfg.WireClaudeMd() {
		return in, problems
	}
	guidancePath := filepath.Join(rootDir, filepath.FromSlash(skills.GuidanceFile))
	if _, err := os.Stat(guidancePath); err != nil {
		return in, problems // fragment absent → nothing to wire
	}
	importLine := "@" + skills.GuidanceFile
	if claudeMd, err := os.ReadFile(filepath.Join(rootDir, "CLAUDE.md")); err == nil &&
		initrepo.GuidanceMarkerLineIdx(strings.Split(string(claudeMd), "\n"), importLine) != -1 {
		return append(in, label("guidance:")+"ok (CLAUDE.md imports the aiwf guidance fragment)"), problems
	}
	val := "claudemd-guidance-unwired: advisory — " + skills.GuidanceFile + " exists but CLAUDE.md does not import it; run `aiwf update` to wire it"
	problems = append(problems, Problem{Severity: SeverityWarn, Message: val})
	return append(in, label("guidance:")+val), problems
}
