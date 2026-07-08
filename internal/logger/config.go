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

// YAMLConfig is the argument shape ResolveConfig accepts for aiwf.yaml's
// optional top-level logging: block (ADR-0017 Decision #3). All three
// fields are optional; a zero value means "absent." internal/config.Logging
// (config.Logging.ToYAMLConfig()) is what actually decodes a real
// aiwf.yaml file today — internal/logger can't import internal/config
// (see internal/config.Logging's doc comment), so this is a separately
// declared, structurally identical type callers convert into.
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
//
// A thin wrapper over ResolveConfigWithSources for callers (most of
// them) that only need the merged result, not which tier supplied it.
func ResolveConfig(getenv func(string) string, yamlCfg YAMLConfig) (Config, error) {
	cfg, _, err := ResolveConfigWithSources(getenv, yamlCfg)
	return cfg, err
}

// FieldSource names which precedence tier supplied a resolved field's
// value.
type FieldSource string

const (
	// SourceEnv means the corresponding AIWF_LOG* env var supplied the value.
	SourceEnv FieldSource = "env"
	// SourceYAML means aiwf.yaml's logging: block supplied the value.
	SourceYAML FieldSource = "yaml"
	// SourceDefault means neither source was set for this field.
	SourceDefault FieldSource = "default"
)

// Sources reports, per field, which precedence tier supplied the
// value ResolveConfigWithSources actually used — for an operator-
// facing surface (`aiwf doctor`) that explains why a given value won,
// not just what the merged result is. The zero value (empty
// FieldSource on every field) is what a disabled Config carries: with
// no level from either source, format/destination are never resolved
// at all, so labeling them "default" would misleadingly imply they
// were consulted.
type Sources struct {
	Level       FieldSource
	Format      FieldSource
	Destination FieldSource
}

// ResolveConfigWithSources is ResolveConfig plus the per-field
// Sources breakdown. See ResolveConfig's doc comment for the
// precedence rule and error conditions; this is the same resolution,
// not a second implementation of it.
func ResolveConfigWithSources(getenv func(string) string, yamlCfg YAMLConfig) (Config, Sources, error) {
	levelSrc, levelSource := resolveWithSource(getenv("AIWF_LOG"), yamlCfg.Level)
	if levelSrc == "" {
		return Config{}, Sources{}, nil
	}
	level, ok := levelNames[levelSrc]
	if !ok {
		return Config{}, Sources{}, fmt.Errorf("logging: invalid level %q (want one of debug, info, warn, error)", levelSrc)
	}

	format, formatSource := resolveWithSource(getenv("AIWF_LOG_FORMAT"), yamlCfg.Format)
	if format == "" {
		format = "text"
	}
	if format != "text" && format != "json" {
		return Config{}, Sources{}, fmt.Errorf("logging: invalid format %q (want text or json)", format)
	}

	destination, destSource := resolveWithSource(getenv("AIWF_LOG_FILE"), yamlCfg.Destination)

	return Config{
			Enabled:     true,
			Level:       level,
			Format:      format,
			Destination: destination,
		}, Sources{
			Level:       levelSource,
			Format:      formatSource,
			Destination: destSource,
		}, nil
}

// resolveWithSource returns the env value if non-empty (SourceEnv),
// else the yaml value if non-empty (SourceYAML), else "" (SourceDefault
// — the caller applies its own default, if any, for that field).
func resolveWithSource(envVal, yamlVal string) (value string, source FieldSource) {
	if envVal != "" {
		return envVal, SourceEnv
	}
	if yamlVal != "" {
		return yamlVal, SourceYAML
	}
	return "", SourceDefault
}
