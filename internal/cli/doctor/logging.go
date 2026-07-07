package doctor

import (
	"fmt"
	"os"
	"strings"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/logger"
)

// appendLoggingReport appends the `logging:` line: the currently
// active, fully-resolved diagnostic-logging configuration (ADR-0017),
// naming which tier — env, yaml, or default — supplied each field, so
// an operator can confirm what's on without reading source. cfg may be
// nil (config load failed above); the yaml tier is then treated as
// absent, matching every other cfg-derived doctor line's nil tolerance.
//
// Disabled is the documented default-off state, never a problem. An
// invalid AIWF_LOG*/logging: value is a real misconfiguration and is
// reported as an error-severity problem, same as the other config-
// derived lines in this report.
func appendLoggingReport(lines []string, problems []Problem, cfg *config.Config) ([]string, []Problem) {
	var yamlCfg logger.YAMLConfig
	if cfg != nil {
		yamlCfg = cfg.Logging.ToYAMLConfig()
	}
	resolved, sources, err := logger.ResolveConfigWithSources(os.Getenv, yamlCfg)
	if err != nil {
		val := err.Error()
		lines = append(lines, label("logging:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	if !resolved.Enabled {
		lines = append(lines, label("logging:")+"disabled (opt in via AIWF_LOG or aiwf.yaml's logging: block)")
		return lines, problems
	}
	val := fmt.Sprintf(
		"enabled level=%s format=%s destination=%s (level: %s, format: %s, destination: %s)",
		strings.ToLower(resolved.Level.String()), resolved.Format, destinationDisplay(resolved.Destination),
		sources.Level, sources.Format, sources.Destination,
	)
	lines = append(lines, label("logging:")+val)
	return lines, problems
}

// destinationDisplay renders the resolved destination for the doctor
// line: the empty string means "use the default XDG-state-home daily
// file" (internal/logger's own OpenDestination contract), which reads
// as blank rather than a meaningful value if shown verbatim.
func destinationDisplay(d string) string {
	if d == "" {
		return "(default XDG-state-home file)"
	}
	return d
}
