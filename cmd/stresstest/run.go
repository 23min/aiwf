package main

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/stresstest"
)

func newRunCmd() *cobra.Command {
	var (
		moduleRoot   string
		outDir       string
		repeat       int
		scenarioName string
	)
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Build the aiwf binary under test and run one or all of the real catalog scenarios",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRun(cmd.Context(), moduleRoot, outDir, repeat, scenarioName, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&moduleRoot, "module-root", ".", "aiwf module root to build the binary under test from")
	cmd.Flags().StringVar(&outDir, "out", "", "directory for the build output, scenario temp dirs, and the raw-report file (defaults to a fresh temp dir, printed on completion)")
	cmd.Flags().IntVar(&repeat, "repeat", 1, "number of times to repeat the scenario")
	cmd.Flags().StringVar(&scenarioName, "scenario", "", fmt.Sprintf("scenario to run: one of %s", strings.Join(scenarioNames(), ", ")))
	_ = cmd.MarkFlagRequired("scenario")
	_ = cmd.RegisterFlagCompletionFunc("scenario", cobra.FixedCompletions(scenarioNames(), cobra.ShellCompDirectiveNoFileComp))
	return cmd
}

// unknownScenarioError reports that name is not a registered catalog
// entry, naming the full valid set so the operator doesn't have to
// consult source or --help to recover.
func unknownScenarioError(name string) error {
	valid := append([]string{}, scenarioNames()...)
	sort.Strings(valid)
	return fmt.Errorf("unknown --scenario %q; want one of: %s", name, strings.Join(valid, ", "))
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

// runRun builds the aiwf binary under test, then runs the named
// catalog scenario --repeat times against it, logging each attempt to
// a raw-report JSONL file under outDirFlag (or a fresh temp dir if
// empty). scenarioName is validated against the registry before any
// I/O — a bad --scenario, like a bad --out, never wastes a compile.
func runRun(ctx context.Context, moduleRoot, outDirFlag string, repeat int, scenarioName string, out io.Writer) error {
	if repeat <= 0 {
		return fmt.Errorf("repeat count must be positive, got %d", repeat)
	}
	entry, ok := lookupScenario(scenarioName)
	if !ok {
		return unknownScenarioError(scenarioName)
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
	rt := scenarioRuntime{aiwfBin: bin}
	if needsLockHolder(scenarioName) {
		lockHolderBin, buildErr := stresstest.BuildLockHolder(ctx, moduleRoot, outDir)
		if buildErr != nil { //coverage:ignore BuildLockHolder's own failure path is covered at its source (internal/stresstest/binary_test.go's TestBuildLockHolder_ErrorsOnBuildFailure) — a moduleRoot bad enough to fail this build already fails BuildBinary above it in the same call, so this specific branch isn't independently triggerable through runRun
			return fmt.Errorf("building lockholder binary under test: %w", buildErr)
		}
		rt.lockHolderBin = lockHolderBin
	}

	results, err := stresstest.RunRepeated(entry.Build(rt), outDir, repeat, nextSeed, rw)
	if err != nil { //coverage:ignore not portably triggerable: every registered scenario's Setup/Run failure mode is a genuine environmental fault (a bad binary path, a disk fault) already exercised at its own source in internal/stresstest; forcing one here, or forcing rw.WriteEvent to fail mid-write, needs either sabotaging the freshly built binary or an already-open fd to fail, neither reproducible without an unsafe/fragile test
		return fmt.Errorf("running scenario %s: %w", entry.Name, err)
	}

	passCount := 0
	for _, r := range results {
		if r.Passed {
			passCount++
		}
	}
	_, _ = fmt.Fprintf(out, "stresstest run: %s: %d/%d attempts passed; raw report at %s\n", entry.Name, passCount, len(results), reportPath)
	return nil
}

// nextSeed returns a fresh pseudo-random seed for one repeat attempt.
func nextSeed() int64 { return rand.Int64() } //nolint:gosec // G404: replay needs a seedable source; crypto/rand can't be seeded, and this isn't a security context
