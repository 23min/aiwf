package main

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/stresstest"
)

func newRunCmd() *cobra.Command {
	var (
		moduleRoot string
		outDir     string
		repeat     int
	)
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Build the aiwf binary under test and run the placeholder scenario",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRun(cmd.Context(), moduleRoot, outDir, repeat, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&moduleRoot, "module-root", ".", "aiwf module root to build the binary under test from")
	cmd.Flags().StringVar(&outDir, "out", "", "directory for the build output, scenario temp dirs, and the raw-report file (defaults to a fresh temp dir, printed on completion)")
	cmd.Flags().IntVar(&repeat, "repeat", 1, "number of times to repeat the scenario")
	return cmd
}

// resolveOutDir returns an absolute, existing directory for a run's
// output: a fresh temp directory when outDir is empty, or outDir made
// absolute and created otherwise.
func resolveOutDir(outDir string) (string, error) {
	if outDir == "" {
		dir, err := os.MkdirTemp("", "stresstest-run-")
		if err != nil {
			return "", fmt.Errorf("creating run output dir: %w", err)
		}
		return dir, nil
	}
	abs, err := filepath.Abs(outDir)
	if err != nil { //coverage:ignore not portably triggerable: filepath.Abs on a relative path only fails if os.Getwd() fails, which requires the process's cwd to be removed out from under it — unsafe to simulate under parallel tests
		return "", fmt.Errorf("resolving --out to an absolute path: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", fmt.Errorf("creating out dir %s: %w", abs, err)
	}
	return abs, nil
}

// runRun builds the aiwf binary under test, then runs the placeholder
// scenario --repeat times against it, logging each attempt to a
// raw-report JSONL file under outDirFlag (or a fresh temp dir if
// empty). The raw-report file is opened before the binary is built,
// so a bad --out never wastes a compile.
func runRun(ctx context.Context, moduleRoot, outDirFlag string, repeat int, out io.Writer) error {
	if repeat <= 0 {
		return fmt.Errorf("repeat count must be positive, got %d", repeat)
	}

	outDir, err := resolveOutDir(outDirFlag)
	if err != nil {
		return err
	}

	reportPath := filepath.Join(outDir, "report.jsonl")
	rw, err := stresstest.OpenReportWriter(reportPath)
	if err != nil {
		return fmt.Errorf("opening raw-report file: %w", err)
	}
	defer func() { _ = rw.Close() }()

	bin, err := stresstest.BuildBinary(ctx, moduleRoot, outDir)
	if err != nil {
		return fmt.Errorf("building aiwf binary under test: %w", err)
	}

	newScenario := func(seed int64) stresstest.Scenario {
		return &placeholderScenario{aiwfBin: bin}
	}

	results, err := stresstest.RunRepeated(newScenario, outDir, repeat, nextSeed, rw)
	if err != nil { //coverage:ignore not portably triggerable: newScenario is hardcoded to the always-passing placeholderScenario (git init + `aiwf check`, exit code ignored) and rw is a real, already-open file — forcing this path needs either sabotaging git itself or an already-open fd to fail mid-write, neither reproducible without an unsafe/fragile test
		return fmt.Errorf("running scenario: %w", err)
	}

	passCount := 0
	for _, r := range results {
		if r.Passed {
			passCount++
		}
	}
	_, _ = fmt.Fprintf(out, "stresstest run: %d/%d attempts passed; raw report at %s\n", passCount, len(results), reportPath)
	return nil
}

// nextSeed returns a fresh pseudo-random seed for one repeat attempt.
func nextSeed() int64 { return rand.Int64() } //nolint:gosec // G404: replay needs a seedable source; crypto/rand can't be seeded, and this isn't a security context

// placeholderScenario runs `aiwf check` against a freshly git-init'd
// empty repo and always passes — the trivial scenario M-0240's own
// constraint calls for; no real catalog scenario ships until M-0241+.
type placeholderScenario struct {
	aiwfBin string
}

func (p *placeholderScenario) Setup(dir string) error {
	return exec.Command("git", "init", "-q", dir).Run()
}

func (p *placeholderScenario) Run(dir string) error {
	cmd := exec.Command(p.aiwfBin, "check")
	cmd.Dir = dir
	_ = cmd.Run() // exit code intentionally ignored: this placeholder always passes
	return nil
}

func (p *placeholderScenario) Verify(_ string) []stresstest.Violation {
	return nil
}
