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
// block, not an error — since internal/config cannot import
// internal/logger (config sits above logger in the tier order), the
// yaml block's three fields are copied across as plain strings.
func ResolveLogger(rootDir string, getenv func(string) string) (log *slog.Logger, closeLog func() error) {
	var yamlCfg logger.YAMLConfig
	if cfg, cfgErr := config.Load(rootDir); cfgErr == nil && cfg != nil {
		yamlCfg = logger.YAMLConfig{
			Level:       cfg.Logging.Level,
			Format:      cfg.Logging.Format,
			Destination: cfg.Logging.Destination,
		}
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

func noopClose() error { return nil }
