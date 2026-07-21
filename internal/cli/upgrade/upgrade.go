// Package upgrade implements the `aiwf upgrade` verb (per-verb subpackage of M-0116;
// cmd/aiwf/main.go newRootCmd wires it via NewCmd).
package upgrade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/logger"
	"github.com/23min/aiwf/internal/version"
)

// NewCmd builds `aiwf upgrade`: a one-command flow that fetches
// a newer (or specified) aiwf binary via `go install` and re-execs
// the new binary to refresh the consumer repo's framework artifacts
// via `aiwf update`.
//
// Without flags, installs `<package>@latest`; with --version, pins to
// the supplied semver. With --check, prints the current/target
// comparison and exits without installing anything.
//
// Test seams (env-var honored, undocumented for users):
//   - AIWF_GO_BIN: path to the `go` binary (default: lookpath "go").
//     Tests substitute a shell shim that fakes `go install`.
//   - AIWF_NO_REEXEC: when set to a non-empty value, skip the
//     syscall.Exec into the new binary after install. Lets tests
//     verify install succeeds without overlaying the test process.
func NewCmd() *cobra.Command {
	var (
		root      string
		target    string
		checkOnly bool
	)
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Fetch a newer aiwf binary via go install and refresh artifacts",
		Example: `  # Upgrade to latest published release
  aiwf upgrade

  # Pin to a specific tag
  aiwf upgrade --version v0.6.0

  # Print the current/target comparison without installing
  aiwf upgrade --check`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(root, target, checkOnly))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root for the post-install `aiwf update` step (default: cwd)")
	cmd.Flags().StringVar(&target, "version", "latest", "version to install: a semver tag (e.g. v0.2.0) or 'latest'")
	cmd.Flags().BoolVar(&checkOnly, "check", false, "print the current/target comparison and exit without installing")
	return cmd
}

// Run executes `aiwf upgrade`. Returns one of the cliutil.Exit* codes.
func Run(root, target string, checkOnly bool) (code int) {
	pkg := version.PackagePath()
	if pkg == "" {
		cliutil.Errorln("aiwf upgrade: package path unavailable from build info — run `go install <pkg>@latest` manually")
		return cliutil.ExitInternal
	}

	current := version.Current()
	cliutil.Printf("current:  %s\n", RenderVersionLabel(current))

	resolved, latestErr := ResolveTarget(target)
	switch {
	case latestErr == nil:
		cliutil.Printf("target:   %s\n", RenderVersionLabel(resolved))
	case errors.Is(latestErr, version.ErrProxyDisabled):
		cliutil.Printf("target:   %s (proxy disabled — go install will resolve at install time)\n", target)
	default:
		cliutil.Printf("target:   %s (proxy lookup failed: %v)\n", target, latestErr)
		cliutil.Print(proxyLookupFailedHint(pkg))
	}

	if latestErr == nil {
		switch version.Compare(current, resolved) {
		case version.SkewEqual:
			cliutil.Println("status:   already at target, nothing to do")
			return cliutil.ExitOK
		case version.SkewAhead:
			cliutil.Println("status:   binary is ahead of target (downgrade)")
		case version.SkewBehind:
			cliutil.Println("status:   upgrade available")
		default:
			cliutil.Println("status:   skew unknown (devel or pre-release on either side)")
		}
		if hint := proxyStaleHint(current, resolved); hint != "" {
			cliutil.Print(hint)
		}
	}

	if checkOnly {
		return cliutil.ExitOK
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		cliutil.Errorf("aiwf upgrade: %v\n", err)
		return cliutil.ExitUsage
	}

	// Minted once here rather than at the tail, per M-0238/AC-5 (see
	// cancel.Run's identical comment) — upgrade has no --actor flag, so
	// actor is bound empty. installSucceeded guards the deferred
	// failure emit below: once install.completed has fired, a later
	// reexec failure must not also emit install.failed for the same
	// invocation (see the emission below for why the two are
	// deliberately decoupled from Run's own final exit code).
	diagLog, closeDiagLog := cliutil.ResolveLogger(rootDir, os.Getenv)
	defer func() { _ = closeDiagLog() }()
	if diagLog.Enabled(context.Background(), slog.LevelInfo) {
		diagLog = logger.WithVerb(diagLog, "upgrade", target, "", logger.NewRunID())
	}
	var installSucceeded bool
	defer func() {
		if !installSucceeded && code != cliutil.ExitOK {
			cliutil.EmitVerbOutcome(diagLog, "install", code, "")
		}
	}()

	installArg := pkg + "@" + target
	cliutil.Printf("\nrunning:  go install %s\n", installArg)

	if stderrBuf, installErr := runGoInstall(context.Background(), installArg); installErr != nil {
		cliutil.Errorf("aiwf upgrade: %v\n", installErr)
		// G46: detect "module found but does not contain package" — the
		// signature of a release that relocated the cmd package within
		// the module. Without remediation the user sees only the raw
		// `go install` error and has to figure out the recovery
		// command themselves.
		if missingPkg, ok := PathChangedFromStderr(stderrBuf); ok {
			printPackagePathChangedHint(pkg, target, missingPkg)
		}
		return cliutil.ExitInternal
	}

	newBinary, err := installedBinaryPath(context.Background(), pkg)
	if err != nil {
		cliutil.Errorf("aiwf upgrade: install succeeded, but locating the new binary failed: %v\n", err)
		if hint := InstallLocationHint(pkg); hint != "" {
			cliutil.Errorf("                the new binary is most likely at %s\n", hint)
			cliutil.Errorf("                run `%s update --root %s` to refresh consumer artifacts\n", hint, rootDir)
		} else {
			cliutil.Errorln("                run `aiwf update` manually to refresh consumer artifacts")
		}
		return cliutil.ExitInternal
	}

	// Install genuinely succeeded (a binary exists on disk); this is
	// upgrade's "install.completed" moment. Whether the reexec below
	// into `aiwf update` also succeeds is a separate, subsequent
	// concern — installSucceeded stops the deferred failure emit above
	// from also firing install.failed if reexec fails later.
	installSucceeded = true
	cliutil.EmitVerbOutcome(diagLog, "install", cliutil.ExitOK, "")

	// G-0134: ad-hoc sign on Darwin to dodge the Sonoma 14.8.x
	// syspolicyd crash on unsigned Mach-O binaries. Warn-and-continue
	// on failure: the binary still runs unsigned (just risks the
	// syspolicyd crash on stale state); failing the upgrade for a
	// codesign hiccup would be worse UX than a hint that lets the
	// operator sign manually later.
	if signErr := signDarwinBinary(newBinary); signErr != nil {
		cliutil.Errorf("aiwf upgrade: %v\n", signErr)
		cliutil.Errorf("                manually sign with: codesign -s - -f %s\n", newBinary)
		cliutil.Errorln("                continuing (binary works unsigned but may trigger syspolicyd on stale state)")
	}

	if os.Getenv("AIWF_NO_REEXEC") != "" {
		cliutil.Printf("install succeeded; new binary at %s\n", newBinary)
		cliutil.Println("AIWF_NO_REEXEC set — skipping re-exec into `aiwf update`")
		return cliutil.ExitOK
	}

	cliutil.Printf("re-exec:  %s update --root %s\n", newBinary, rootDir)
	if err := reexecUpdate(newBinary, rootDir); err != nil {
		cliutil.Errorf("aiwf upgrade: re-exec failed: %v\n", err)
		cliutil.Errorln("                run `aiwf update` manually to refresh consumer artifacts")
		return cliutil.ExitInternal
	}
	// reexecUpdate replaces this process; we never return from here on
	// success.
	//coverage:ignore Reached only when reexecUpdate's syscall.Exec
	// returns without overlaying the process — kernel-level oddity.
	return cliutil.ExitInternal
}

// proxyStaleHint returns a multi-line hint when the running binary's
// pseudo-version base is newer than the resolved target — the
// signature of a freshly-pushed tag that the Go module proxy CDN has
// not yet propagated to every edge. Empty string when no hint applies.
//
// The Go module proxy's edge cache propagates new tags non-uniformly;
// a tag visible from one POP may take several minutes to surface at
// another. The pseudo-version base (e.g. v0.8.1 in
// v0.8.1-0.<date>-<sha>) is the proxy's *predicted* next tag for the
// binary's commit; when the resolved latest is older than that base,
// the user is almost certainly hitting stale-edge propagation rather
// than a real downgrade. The hint points at the GOPROXY=direct
// workaround and the "retry in a few minutes" path.
//
// Closes G-0149.
func proxyStaleHint(current, resolved version.Info) string {
	if current.Tagged {
		return ""
	}
	base, ok := version.PseudoBase(current.Version)
	if !ok {
		return ""
	}
	baseInfo := version.Parse(base)
	if version.Compare(baseInfo, resolved) != version.SkewAhead {
		return ""
	}
	return fmt.Sprintf(
		"hint:     pseudo-base %s is newer than target %s; the Go module\n"+
			"          proxy CDN may not have propagated the freshest tag yet.\n"+
			"          retry in a few minutes, or set GOPROXY=direct to bypass.\n",
		base, resolved.Version)
}

// proxyLookupFailedHint returns operator guidance shown when the
// pre-flight latest-version lookup to the Go module proxy fails
// (commonly a `context deadline exceeded` timeout reaching
// proxy.golang.org's `/@v/list`). It is advisory: the subsequent
// `go install …@latest` still runs and can succeed via GOPROXY's
// `,direct` fallback even when the direct proxy GET timed out. If
// `go install` also fails, the three remediations below apply.
func proxyLookupFailedHint(pkg string) string {
	return fmt.Sprintf(
		"hint:     the latest-version lookup to the Go proxy failed; `go install` will\n"+
			"          still try (and may resolve via GOPROXY's `,direct` fallback). If it\n"+
			"          also fails: retry once the proxy warms, pin a version with\n"+
			"          `go install %s@vX.Y.Z`, or bypass the proxy with\n"+
			"          `GOPROXY=direct aiwf upgrade`.\n",
		pkg)
}

// RenderVersionLabel formats an Info for human display: the version
// itself plus a parenthetical state ("tagged", "working-tree build",
// "pseudo-version"). The `+dirty` suffix that Go appends to working-
// tree builds with uncommitted changes is always rendered as a
// working-tree build, regardless of the base version shape.
func RenderVersionLabel(info version.Info) string {
	switch {
	case info.Version == version.DevelVersion:
		return info.Version + " (working-tree build)"
	case strings.HasSuffix(info.Version, "+dirty"):
		return info.Version + " (working-tree build)"
	case info.Tagged:
		return info.Version + " (tagged)"
	default:
		return info.Version + " (pseudo-version)"
	}
}

// ResolveTarget turns the --version flag value into a concrete Info.
// The literal "latest" routes through the module proxy; an explicit
// semver is taken at face value (the install step is the eventual
// authority on whether the tag exists).
func ResolveTarget(target string) (version.Info, error) {
	if target != "latest" {
		return version.Parse(target), nil
	}
	return version.Latest(context.Background())
}

// runGoInstall shells out to `go install <arg>` with stdout streamed
// through to the user and stderr tee'd to both the user *and* a
// captured buffer. The buffer lets the caller introspect the failure
// text — needed for G46's "module found but does not contain
// package" detection — without depriving the user of the live
// stderr stream. The `go` binary path is honored from AIWF_GO_BIN
// when set, otherwise discovered via exec.LookPath.
//
// On non-zero exit returns the captured stderr (for caller
// introspection) and the wrapped error. On success returns ("", nil).
func runGoInstall(ctx context.Context, arg string) (capturedStderr string, err error) {
	goBin, err := goBinaryPath()
	if err != nil {
		return "", err
	}
	var stderrBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, goBin, "install", arg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	cmd.Stdin = os.Stdin
	if runErr := cmd.Run(); runErr != nil {
		return stderrBuf.String(), fmt.Errorf("`go install %s`: %w", arg, runErr)
	}
	return "", nil
}

// pathChangedRE matches the Go toolchain's "module found but does
// not contain package" failure (cmd/go/internal/modload/import.go).
// Captures the missing package subpath so we can echo it back to
// the user in the remediation hint.
//
//	module github.com/owner/repo@v0.4.0 found (v0.4.0), but does not contain package github.com/owner/repo/old/sub/path
//
// The capture group is the missing package path.
var pathChangedRE = regexp.MustCompile(`module .+ found .+, but does not contain package (\S+)`)

// PathChangedFromStderr scans `go install` stderr for the package-
// path-change signature. Returns the missing package path and true
// when matched; ("", false) otherwise. Filed under G46.
func PathChangedFromStderr(stderr string) (string, bool) {
	m := pathChangedRE.FindStringSubmatch(stderr)
	if len(m) < 2 {
		return "", false
	}
	return m[1], true
}

// printPackagePathChangedHint surfaces the G46 structured
// remediation. The user just saw `go install` fail with a path-
// change message buried in toolchain wording; this prints a
// kernel-friendly explanation pointing at CHANGELOG and the manual
// re-install command they need.
//
// pkg is the install path the running binary's upgrade verb tried
// (i.e., the path that no longer exists in target).
// target is the version string passed on the command line ("latest"
// or an explicit "vX.Y.Z").
// missingPkg is the subpath that go install said is missing.
func printPackagePathChangedHint(pkg, target, missingPkg string) {
	cliutil.Errorln()
	cliutil.Errorln("aiwf upgrade: hint — the install path may have changed in the target release.")
	cliutil.Errorf("  This binary's upgrade verb tried `%s@%s`, but the target tag's\n", pkg, target)
	cliutil.Errorf("  module no longer contains that package subpath (`%s`).\n", missingPkg)
	cliutil.Errorln()
	cliutil.Errorln("  Recovery:")
	cliutil.Errorln("    1. Find the new install path in the target release's CHANGELOG.")
	cliutil.Errorln("       https://github.com/23min/aiwf/blob/main/CHANGELOG.md")
	cliutil.Errorf("    2. Run `go install <new-path>@%s` manually.\n", target)
	cliutil.Errorln("    3. Run `aiwf update` in your consumer repo to refresh artifacts.")
	cliutil.Errorln()
	cliutil.Errorln("  After that, future `aiwf upgrade` runs from the new binary will work end-to-end.")
}

// goBinaryPath returns the path to the `go` toolchain binary,
// honoring the AIWF_GO_BIN test seam.
func goBinaryPath() (string, error) {
	if env := os.Getenv("AIWF_GO_BIN"); env != "" {
		return env, nil
	}
	path, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("locating `go` on PATH: %w (install Go from https://go.dev/dl/, or set AIWF_GO_BIN to a working binary)", err)
	}
	return path, nil
}

// installedBinaryPath returns the absolute path to the binary that
// `go install` just produced for pkg. Tries `go env GOBIN` first;
// falls back to `$GOPATH/bin` (then `$HOME/go/bin`). The binary name
// is the last segment of pkg.
func installedBinaryPath(ctx context.Context, pkg string) (string, error) {
	binDir, err := GoBinDir(ctx)
	if err != nil {
		return "", err
	}
	name := filepath.Base(pkg)
	full := filepath.Join(binDir, name)
	if _, err := os.Stat(full); err != nil {
		return "", fmt.Errorf("expected binary at %s: %w", full, err)
	}
	return full, nil
}

// GoBinDir returns the directory `go install` writes binaries into.
// Resolution order matches `go install`'s own logic: GOBIN if set,
// else GOPATH/bin (where GOPATH defaults to $HOME/go).
//
// Each variable is queried in its own `go env` call rather than as
// `go env GOBIN GOPATH`. The combined form returns one line per name
// with empty values rendered as a blank line, and the leading blank
// for an unset GOBIN was being silently consumed by strings.TrimSpace
// — see G39 in docs/archive/pocv3/gaps-pre-migration.md for the upgrade-flow regression.
func GoBinDir(ctx context.Context) (string, error) {
	goBin, err := goBinaryPath()
	if err != nil {
		return "", err
	}
	gobin, err := goEnv(ctx, goBin, "GOBIN")
	if err != nil {
		return "", err
	}
	if gobin != "" {
		return gobin, nil
	}
	gopath, err := goEnv(ctx, goBin, "GOPATH")
	if err != nil {
		return "", err
	}
	if gopath == "" {
		return "", errors.New("`go env GOPATH` is empty")
	}
	return filepath.Join(gopath, "bin"), nil
}

// goEnv runs `go env <name>` and returns the trimmed value. Empty
// output (an unset variable like GOBIN with no override) returns an
// empty string with no error — the caller decides how to fall back.
func goEnv(ctx context.Context, goBin, name string) (string, error) {
	out, err := exec.CommandContext(ctx, goBin, "env", name).Output()
	if err != nil {
		return "", fmt.Errorf("`go env %s`: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// InstallLocationHint returns a best-guess absolute path to the
// binary `go install <pkg>` would have produced, derived from the
// caller's environment without invoking `go env`. Used only to help
// the user recover after locateBinary failed; never load-bearing.
// Returns an empty string when neither GOBIN/GOPATH nor a home
// directory can be resolved.
func InstallLocationHint(pkg string) string {
	name := filepath.Base(pkg)
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		return filepath.Join(gobin, name)
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return filepath.Join(gopath, "bin", name)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "go", "bin", name)
	}
	return ""
}

// reexecUpdate overlays the current process with the new binary
// running `aiwf update --root <rootDir>`. On success, this function
// does not return — control transfers to the new binary.
var reexecUpdate = func(newBinary, rootDir string) error {
	args := []string{filepath.Base(newBinary), "update", "--root", rootDir}
	return syscall.Exec(newBinary, args, os.Environ())
}
