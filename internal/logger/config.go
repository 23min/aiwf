// Package logger provides aiwf's opt-in, default-off diagnostic-log
// surface: a thin wrapper around log/slog whose configuration comes
// from three AIWF_LOG* env vars, then aiwf.yaml's logging: block,
// then a no-op default (ADR-0017).
package logger

import (
	"fmt"
	"log/slog"
)

// levelNames is the closed set of level values ADR-0017 Decision #3
// accepts from either AIWF_LOG or the logging.level yaml key.
var levelNames = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

// YAMLConfig is the decoded shape of aiwf.yaml's optional top-level
// logging: block (ADR-0017 Decision #3). All three keys are optional;
// an absent key decodes to the empty string.
type YAMLConfig struct {
	Level       string `yaml:"level"`
	Format      string `yaml:"format"`
	Destination string `yaml:"destination"`
}

// Config is the fully-resolved diagnostic-logging configuration for
// one aiwf invocation.
type Config struct {
	// Enabled is false when neither AIWF_LOG nor aiwf.yaml's
	// logging.level was set — the default-off state (ADR-0017
	// Decision #2). Level/Format/Destination are the zero value
	// when Enabled is false.
	Enabled bool
	Level   slog.Level
	// Format is "text" or "json".
	Format string
	// Destination is "" (use the default XDG-state-home path),
	// "stderr", or an absolute path.
	Destination string
}

// ResolveConfig applies ADR-0017's env-beats-yaml-beats-default
// precedence, independently per setting: AIWF_LOG/AIWF_LOG_FORMAT/
// AIWF_LOG_FILE each beat the corresponding aiwf.yaml logging: key,
// which beats the default. getenv is injected so callers (and tests)
// control the environment read; yamlCfg is the zero value when
// aiwf.yaml has no logging: block.
//
// Logging is enabled only when a level is supplied from either
// source — setting only format or destination without a level never
// opts in, matching ADR-0017 Decision #2's default-off state.
//
// Returns an error when a supplied level or format value (from
// either source) is not one of the closed sets ADR-0017 defines.
func ResolveConfig(getenv func(string) string, yamlCfg YAMLConfig) (Config, error) {
	levelSrc := firstNonEmpty(getenv("AIWF_LOG"), yamlCfg.Level)
	if levelSrc == "" {
		return Config{}, nil
	}
	level, ok := levelNames[levelSrc]
	if !ok {
		return Config{}, fmt.Errorf("logging: invalid level %q (want one of debug, info, warn, error)", levelSrc)
	}

	format := firstNonEmpty(getenv("AIWF_LOG_FORMAT"), yamlCfg.Format, "text")
	if format != "text" && format != "json" {
		return Config{}, fmt.Errorf("logging: invalid format %q (want text or json)", format)
	}

	return Config{
		Enabled:     true,
		Level:       level,
		Format:      format,
		Destination: firstNonEmpty(getenv("AIWF_LOG_FILE"), yamlCfg.Destination),
	}, nil
}

// firstNonEmpty returns the first non-empty string in vals, or "" when
// all are empty.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
