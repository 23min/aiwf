// Package contractverify runs a consumer repo's contract validators
// against the configured fixtures and reports verdicts. It is the
// engine half of contract verification; the CLI verb (`aiwf contract
// verify`) and the pre-push integration (`aiwf check`) are thin
// callers.
//
// Two passes per contract:
//
//   - Verify pass: the *current* fixture-tree version (lexicographically
//     highest directory name) is exercised. Every file in `valid/` is
//     expected to pass; every file in `invalid/` is expected to fail.
//   - Evolve pass: every *non-current* version's `valid/` fixtures are
//     run against the HEAD schema. Failures here flag silent breakage
//     introduced by a schema change that broke historical compatibility.
//
// The engine is validator-agnostic. It executes the user-declared
// command + args (with four documented substitution variables) and
// makes one judgment: exit code 0 means accepted; non-zero means
// rejected. stdout and stderr are captured as opaque text and surfaced
// in finding details so the user can see what their tool said.
//
// One reclassification keeps noise down: when *every* `valid/` fixture
// for a contract is rejected, the schema itself is the more likely
// culprit. The per-fixture findings collapse into a single
// `validator-error` for the contract.
package contractverify

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/aiwfyaml"
	"github.com/23min/ai-workflow-v2/tools/internal/contractconfig"
)

// Result is one verdict produced by the verify or evolve pass.
// Code corresponds to the documented finding codes in §10 of the
// contracts plan; see the constants below for the closed set.
//
// Detail carries the raw stderr/stdout of the validator invocation
// for fixture-level findings, or a free-form explanation for
// contract-level findings.
type Result struct {
	Code        string
	Message     string
	Detail      string
	EntityID    string
	Version     string
	FixturePath string
}

// Finding code constants. The set is closed and matches §10 of
// docs/poc-contracts-plan.md.
const (
	CodeFixtureRejected     = "fixture-rejected"
	CodeFixtureAccepted     = "fixture-accepted"
	CodeEvolutionRegression = "evolution-regression"
	CodeValidatorError      = "validator-error"
	CodeEnvironment         = "environment"
)

// Options configures one verify-and-evolve pass over the consumer
// repo's contract bindings.
type Options struct {
	// RepoRoot is the consumer repo root. Validators run with this
	// as their working directory; schema and fixture paths in
	// Contracts are resolved relative to it.
	RepoRoot string
	// Contracts is the parsed `contracts:` block from aiwf.yaml.
	// Pass a non-nil value with empty Entries for a no-op pass.
	Contracts *aiwfyaml.Contracts
	// SkipIDs lists contract entity ids that should be excluded from
	// verification. Callers fill this with ids whose status is
	// `rejected` or `retired`; the package itself has no entity
	// awareness.
	SkipIDs map[string]bool
}

// Run executes the verify and evolve passes for every binding in
// opts.Contracts.Entries (skipping ids listed in opts.SkipIDs) and
// returns the resulting findings. The slice is sorted: contract id
// first, then code, then version, then fixture path — so output is
// stable regardless of filesystem walk order.
//
// Run never returns a Go error: every failure mode (missing binary,
// missing fixture path, bad exit code) becomes a Result.
func Run(ctx context.Context, opts Options) []Result {
	var out []Result
	if opts.Contracts == nil {
		return out
	}
	// Refuse to invoke a validator on any entry whose configured paths
	// escape the repo root (contractconfig has already raised the
	// path-escape finding via aiwf check; here we silently skip the
	// validator invocation). This is the load-bearing guarantee for
	// G1: a corrupted aiwf.yaml can never cause a validator to run on
	// out-of-repo content.
	resolved, _ := contractconfig.Resolve(opts.RepoRoot, opts.Contracts.Entries)
	for i, e := range opts.Contracts.Entries {
		if opts.SkipIDs[e.ID] {
			continue
		}
		if resolved[i].Skip {
			continue
		}
		v, ok := opts.Contracts.Validators[e.Validator]
		if !ok {
			// Validator-name resolution is a structural property of
			// the contracts: block; aiwfyaml.Validate already enforces
			// it. Treat a missed lookup here as a defensive guard
			// rather than a user-facing finding.
			continue
		}
		out = append(out, runOne(ctx, opts.RepoRoot, e, v, resolved[i].FixturesPath)...)
	}
	sortResults(out)
	return out
}

// runOne handles one binding end-to-end. The flow is:
//
//  1. Resolve the validator binary; on miss, emit one `environment`
//     finding and skip every fixture for this contract.
//  2. Enumerate version directories; if there are none (or the
//     fixtures dir is missing/empty), emit nothing — the contract is
//     "registered but bundle in flight" per §8.
//  3. Run verify pass on the current version.
//  4. Run evolve pass on every non-current version.
//  5. Reclassify per-fixture findings into a single `validator-error`
//     when *every* valid fixture in the verify pass was rejected.
func runOne(ctx context.Context, repoRoot string, e aiwfyaml.Entry, v aiwfyaml.Validator, fixturesPath string) []Result {
	if _, err := exec.LookPath(v.Command); err != nil {
		return []Result{{
			Code:     CodeEnvironment,
			EntityID: e.ID,
			Message:  fmt.Sprintf("validator %q (command %q) not found on PATH", e.Validator, v.Command),
			Detail:   err.Error(),
		}}
	}

	versions, err := enumerateVersions(fixturesPath)
	if err != nil {
		// Fixtures directory is missing or unreadable — surface as a
		// per-contract config-style result so the user sees something.
		// (The richer `contract-config` finding lives in `aiwf check`,
		// where path-existence is its own concern; here we report the
		// fact of skipping.)
		return nil
	}
	if len(versions) == 0 {
		return nil
	}

	// Lexicographically highest = "current" per §8.
	current := versions[len(versions)-1]

	var out []Result
	verifyResults, validValidatorFailed, validValidatorTotal := verifyPass(ctx, repoRoot, e, v, fixturesPath, current)
	out = append(out, verifyResults...)

	for _, ver := range versions {
		if ver == current {
			continue
		}
		out = append(out, evolvePass(ctx, repoRoot, e, v, fixturesPath, ver)...)
	}

	// Reclassification: if every valid fixture in the verify pass was
	// rejected, collapse those rejections into one validator-error.
	if validValidatorTotal > 0 && validValidatorFailed == validValidatorTotal {
		out = collapseToValidatorError(out, e.ID, current)
	}
	return out
}

// verifyPass runs the validator over the current version's valid
// and invalid fixture sets. It returns the resulting findings plus
// the (failed, total) tally of valid-fixture rejections so the
// caller can decide whether to reclassify them as validator-error.
func verifyPass(ctx context.Context, repoRoot string, e aiwfyaml.Entry, v aiwfyaml.Validator, fixturesPath, version string) (results []Result, validFailed, validTotal int) {
	validDir := filepath.Join(fixturesPath, version, "valid")
	invalidDir := filepath.Join(fixturesPath, version, "invalid")

	validFixtures := walkFixtures(validDir)
	invalidFixtures := walkFixtures(invalidDir)

	validTotal = len(validFixtures)

	for _, f := range validFixtures {
		exit, detail := runValidator(ctx, repoRoot, e, v, version, f)
		if exit == 0 {
			continue
		}
		validFailed++
		results = append(results, Result{
			Code:        CodeFixtureRejected,
			EntityID:    e.ID,
			Version:     version,
			FixturePath: relPath(repoRoot, f),
			Message:     fmt.Sprintf("valid fixture %q rejected by schema", relPath(repoRoot, f)),
			Detail:      detail,
		})
	}
	for _, f := range invalidFixtures {
		exit, detail := runValidator(ctx, repoRoot, e, v, version, f)
		if exit != 0 {
			continue
		}
		results = append(results, Result{
			Code:        CodeFixtureAccepted,
			EntityID:    e.ID,
			Version:     version,
			FixturePath: relPath(repoRoot, f),
			Message:     fmt.Sprintf("invalid fixture %q accepted by schema", relPath(repoRoot, f)),
			Detail:      detail,
		})
	}
	return results, validFailed, validTotal
}

// evolvePass runs the validator over a single non-current version's
// valid fixtures, expecting all of them to still pass against the
// HEAD schema. Failures emit `evolution-regression`.
func evolvePass(ctx context.Context, repoRoot string, e aiwfyaml.Entry, v aiwfyaml.Validator, fixturesPath, version string) []Result {
	validDir := filepath.Join(fixturesPath, version, "valid")
	fixtures := walkFixtures(validDir)
	var out []Result
	for _, f := range fixtures {
		exit, detail := runValidator(ctx, repoRoot, e, v, version, f)
		if exit == 0 {
			continue
		}
		out = append(out, Result{
			Code:        CodeEvolutionRegression,
			EntityID:    e.ID,
			Version:     version,
			FixturePath: relPath(repoRoot, f),
			Message:     fmt.Sprintf("historical valid fixture %q fails HEAD schema", relPath(repoRoot, f)),
			Detail:      detail,
		})
	}
	return out
}

// collapseToValidatorError replaces every CodeFixtureRejected result
// for entityID at version with a single CodeValidatorError result.
// Other codes (fixture-accepted, evolution-regression, environment)
// pass through unchanged.
func collapseToValidatorError(results []Result, entityID, version string) []Result {
	var (
		out      []Result
		consumed []Result
	)
	for _, r := range results {
		if r.EntityID == entityID && r.Version == version && r.Code == CodeFixtureRejected {
			consumed = append(consumed, r)
			continue
		}
		out = append(out, r)
	}
	if len(consumed) == 0 {
		return results
	}
	out = append(out, Result{
		Code:     CodeValidatorError,
		EntityID: entityID,
		Version:  version,
		Message:  fmt.Sprintf("every valid fixture for %s rejected; the schema or validator invocation is likely broken", entityID),
		Detail:   summarizeDetails(consumed),
	})
	return out
}

// summarizeDetails joins the captured details from the consumed
// per-fixture findings into a single multi-line block. Useful for
// the user when the validator-error reclassification fires; without
// the underlying stderr the user has nothing to act on.
func summarizeDetails(consumed []Result) string {
	var b strings.Builder
	for i, r := range consumed {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		fmt.Fprintf(&b, "[%s]\n%s", r.FixturePath, r.Detail)
	}
	return b.String()
}

// runValidator executes the validator once for one fixture file and
// returns the exit code and a captured stdout+stderr block.
//
// On exec failure (binary disappeared between the LookPath check and
// the call, IO error, ctx cancellation), runValidator returns a
// non-zero exit code and a synthesized detail explaining the cause.
// The caller treats that the same as a validator rejection; the
// reclassification step turns "all valid rejected" into
// validator-error, which is the right code for those cases.
func runValidator(ctx context.Context, repoRoot string, e aiwfyaml.Entry, v aiwfyaml.Validator, version, fixture string) (exitCode int, detail string) {
	args := substitute(v.Args, e, version, fixture)
	cmd := exec.CommandContext(ctx, v.Command, args...)
	cmd.Dir = repoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	detail = combineStdStreams(stdout.Bytes(), stderr.Bytes())
	if err == nil {
		return 0, detail
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), detail
	}
	// Non-exit error (couldn't start, ctx cancelled, etc.).
	return 1, fmt.Sprintf("exec error: %v\n%s", err, detail)
}

// substitute applies the four documented substitution variables to
// every element of args. Empty inputs are returned as a fresh empty
// slice so the caller can mutate without aliasing.
func substitute(args []string, e aiwfyaml.Entry, version, fixture string) []string {
	out := make([]string, len(args))
	r := strings.NewReplacer(
		"{{schema}}", e.Schema,
		"{{fixture}}", fixture,
		"{{contract_id}}", e.ID,
		"{{version}}", version,
	)
	for i, a := range args {
		out[i] = r.Replace(a)
	}
	return out
}

// walkFixtures returns every regular file directly inside dir.
// Subdirectories are not recursed into per §8 ("natural enforcement,
// not a second engine"). A missing directory yields a nil slice.
func walkFixtures(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, en := range entries {
		if !en.Type().IsRegular() {
			continue
		}
		out = append(out, filepath.Join(dir, en.Name()))
	}
	sort.Strings(out)
	return out
}

// enumerateVersions returns every immediate subdirectory of fixturesDir
// in ascending lexicographic order. Used by the orchestrator to pick
// the "current" version (highest name) and to drive the evolve pass.
//
// A non-existent fixturesDir returns an error; callers treat that
// the same as "no versions" and skip the contract.
func enumerateVersions(fixturesDir string) ([]string, error) {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		return nil, err
	}
	var versions []string
	for _, en := range entries {
		if en.IsDir() {
			versions = append(versions, en.Name())
		}
	}
	sort.Strings(versions)
	return versions, nil
}

// combineStdStreams joins captured stdout and stderr into a single
// detail block, labeling each section. Empty streams are omitted so
// the typical "validator wrote only to stderr" case isn't padded
// with "[stdout]" headers for nothing.
func combineStdStreams(stdout, stderr []byte) string {
	stdout = bytes.TrimRight(stdout, "\n")
	stderr = bytes.TrimRight(stderr, "\n")
	switch {
	case len(stdout) == 0 && len(stderr) == 0:
		return ""
	case len(stdout) == 0:
		return string(stderr)
	case len(stderr) == 0:
		return string(stdout)
	}
	return fmt.Sprintf("[stdout]\n%s\n[stderr]\n%s", stdout, stderr)
}

// relPath returns the path of f relative to root, falling back to
// the absolute path on conversion failure. Used in finding messages
// so output is portable across machines.
func relPath(root, f string) string {
	r, err := filepath.Rel(root, f)
	if err != nil {
		return filepath.ToSlash(f)
	}
	return filepath.ToSlash(r)
}

// sortResults orders findings by entity id, code, version, and
// fixture path. Stable across runs so test snapshots and CI logs
// stay diff-friendly.
func sortResults(rs []Result) {
	sort.SliceStable(rs, func(i, j int) bool {
		if rs[i].EntityID != rs[j].EntityID {
			return rs[i].EntityID < rs[j].EntityID
		}
		if rs[i].Code != rs[j].Code {
			return rs[i].Code < rs[j].Code
		}
		if rs[i].Version != rs[j].Version {
			return rs[i].Version < rs[j].Version
		}
		return rs[i].FixturePath < rs[j].FixturePath
	})
}
