package cliutil

import (
	"io"
	"log/slog"
	"time"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/logger"
)

// ResolveLogger resolves this invocation's diagnostic-logging
// configuration from AIWF_LOG*/env vars and rootDir's aiwf.yaml
// logging: block (ADR-0017) via getenv (os.Getenv in production; a fake
// map in tests) and returns a ready-to-use logger plus a closer that is
// always safe to defer-call.
//
// A resolve or destination-open failure never surfaces to the caller: it
// falls back to a discard logger, since diagnostic logging must never
// affect a verb's own behavior or exit code. A missing or unreadable
// aiwf.yaml is tolerated the same way — treated as an absent logging:
// block, not an error. internal/logger cannot import internal/config
// (the reverse direction is legal, but logger sits below config in the
// tier order), so config.Logging.ToYAMLConfig() is the conversion
// point to logger's own decode-target shape.
func ResolveLogger(rootDir string, getenv func(string) string) (log *slog.Logger, closeLog func() error) {
	var yamlCfg logger.YAMLConfig
	if cfg, cfgErr := config.Load(rootDir); cfgErr == nil && cfg != nil {
		yamlCfg = cfg.Logging.ToYAMLConfig()
	}
	cfg, err := logger.ResolveConfig(getenv, yamlCfg)
	if err != nil {
		return logger.New(logger.Config{}, nil), noopClose
	}
	w, err := logger.OpenDestination(cfg, time.Now(), getenv)
	if err != nil {
		return logger.New(logger.Config{}, nil), noopClose
	}
	l := logger.New(cfg, w)
	// "stderr" resolves to the real, shared os.Stderr (internal/logger's
	// own contract) — it must never be closed, unlike a real opened file.
	if cfg.Destination == "stderr" {
		return l, noopClose
	}
	if closer, ok := w.(io.Closer); ok {
		return l, closer.Close
	}
	return l, noopClose
}

// ResolveTraceLogger is ResolveLogger with logging forced enabled at
// debug level (M-0239/AC-3's --trace flag): an operator passing
// --trace gets phase timing for this one invocation without needing
// AIWF_LOG set separately. Destination and format still resolve
// normally (AIWF_LOG_FILE/AIWF_LOG_FORMAT, then aiwf.yaml, then the
// XDG-state-home default) — --trace only forces the invocation on and
// its level floor to debug, never redirects it.
//
// logger.ResolveConfig short-circuits to a bare zero Config (Enabled:
// false, Format/Destination unresolved) whenever no level is supplied
// from any source (ADR-0017 Decision #2's default-off state) — it
// never even looks at AIWF_LOG_FORMAT/AIWF_LOG_FILE in that case.
// Patching Enabled/Level on that zero Config afterward would still
// carry an empty Destination, silently discarding the operator's own
// AIWF_LOG_FILE. forcedGetenv supplies a synthetic "debug" for
// AIWF_LOG only when the real environment has no level set, so
// ResolveConfig takes its normal fully-resolving path and every other
// key (format, destination) still reads the real environment/yaml
// unchanged.
func ResolveTraceLogger(rootDir string, getenv func(string) string) (log *slog.Logger, closeLog func() error) {
	var yamlCfg logger.YAMLConfig
	if cfg, cfgErr := config.Load(rootDir); cfgErr == nil && cfg != nil {
		yamlCfg = cfg.Logging.ToYAMLConfig()
	}
	forcedGetenv := func(key string) string {
		if key == "AIWF_LOG" && getenv(key) == "" && yamlCfg.Level == "" {
			return "debug"
		}
		return getenv(key)
	}
	cfg, err := logger.ResolveConfig(forcedGetenv, yamlCfg)
	if err != nil {
		// A malformed AIWF_LOG_FORMAT (or aiwf.yaml logging.format)
		// value doesn't get to silently defeat --trace: fall back to
		// an always-valid config (debug, text) while still honoring
		// AIWF_LOG_FILE/aiwf.yaml's destination, since that half of
		// the input was never invalid. Discarding it here (as a bare
		// logger.Config{Enabled: true} once did) would have silently
		// dropped the operator's own AIWF_LOG_FILE on an unrelated
		// format typo.
		destination := getenv("AIWF_LOG_FILE")
		if destination == "" {
			destination = yamlCfg.Destination
		}
		cfg = logger.Config{Enabled: true, Level: slog.LevelDebug, Format: "text", Destination: destination}
	}
	cfg.Enabled = true
	if cfg.Level > slog.LevelDebug {
		cfg.Level = slog.LevelDebug
	}
	w, err := logger.OpenDestination(cfg, time.Now(), getenv)
	if err != nil {
		return logger.New(logger.Config{}, nil), noopClose
	}
	l := logger.New(cfg, w)
	if cfg.Destination == "stderr" {
		return l, noopClose
	}
	if closer, ok := w.(io.Closer); ok {
		return l, closer.Close
	}
	return l, noopClose //coverage:ignore unreachable here specifically: OpenDestination only ever returns a nil, non-Closer io.WriteCloser when cfg.Enabled is false (its own early return), but the Enabled=true forced two statements above makes this call's Enabled always true — ResolveLogger's identical fallback IS reachable (via its own genuinely-disabled path), this one structurally isn't
}

func noopClose() error { return nil }
