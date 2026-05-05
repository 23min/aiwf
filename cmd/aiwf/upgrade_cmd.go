package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/23min/ai-workflow-v2/internal/version"
)

// runUpgrade handles `aiwf upgrade`: a one-command flow that fetches
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
func runUpgrade(args []string) int {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root for the post-install `aiwf update` step (default: cwd)")
	target := fs.String("version", "latest", "version to install: a semver tag (e.g. v0.2.0) or 'latest'")
	checkOnly := fs.Bool("check", false, "print the current/target comparison and exit without installing")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	pkg := version.PackagePath()
	if pkg == "" {
		fmt.Fprintln(os.Stderr, "aiwf upgrade: package path unavailable from build info — run `go install <pkg>@latest` manually")
		return exitInternal
	}

	current := version.Current()
	fmt.Printf("current:  %s\n", renderVersionLabel(current))

	resolved, latestErr := resolveTarget(*target)
	switch {
	case latestErr == nil:
		fmt.Printf("target:   %s\n", renderVersionLabel(resolved))
	case errors.Is(latestErr, version.ErrProxyDisabled):
		fmt.Printf("target:   %s (proxy disabled — go install will resolve at install time)\n", *target)
	default:
		fmt.Printf("target:   %s (proxy lookup failed: %v)\n", *target, latestErr)
	}

	if latestErr == nil {
		switch version.Compare(current, resolved) {
		case version.SkewEqual:
			fmt.Println("status:   already at target, nothing to do")
			return exitOK
		case version.SkewAhead:
			fmt.Println("status:   binary is ahead of target (downgrade)")
		case version.SkewBehind:
			fmt.Println("status:   upgrade available")
		default:
			fmt.Println("status:   skew unknown (devel or pre-release on either side)")
		}
	}

	if *checkOnly {
		return exitOK
	}

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf upgrade: %v\n", err)
		return exitUsage
	}

	installArg := pkg + "@" + *target
	fmt.Printf("\nrunning:  go install %s\n", installArg)

	if stderrBuf, installErr := runGoInstall(context.Background(), installArg); installErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf upgrade: %v\n", installErr)
		// G46: detect "module found but does not contain package" — the
		// signature of a release that relocated the cmd package within
		// the module. Without remediation the user sees only the raw
		// `go install` error and has to figure out the recovery
		// command themselves.
		if missingPkg, ok := pathChangedFromStderr(stderrBuf); ok {
			printPackagePathChangedHint(pkg, *target, missingPkg)
		}
		return exitInternal
	}

	newBinary, err := installedBinaryPath(context.Background(), pkg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf upgrade: install succeeded, but locating the new binary failed: %v\n", err)
		if hint := installLocationHint(pkg); hint != "" {
			fmt.Fprintf(os.Stderr, "                the new binary is most likely at %s\n", hint)
			fmt.Fprintf(os.Stderr, "                run `%s update --root %s` to refresh consumer artifacts\n", hint, rootDir)
		} else {
			fmt.Fprintln(os.Stderr, "                run `aiwf update` manually to refresh consumer artifacts")
		}
		return exitInternal
	}

	if os.Getenv("AIWF_NO_REEXEC") != "" {
		fmt.Printf("install succeeded; new binary at %s\n", newBinary)
		fmt.Println("AIWF_NO_REEXEC set — skipping re-exec into `aiwf update`")
		return exitOK
	}

	fmt.Printf("re-exec:  %s update --root %s\n", newBinary, rootDir)
	if err := reexecUpdate(newBinary, rootDir); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf upgrade: re-exec failed: %v\n", err)
		fmt.Fprintln(os.Stderr, "                run `aiwf update` manually to refresh consumer artifacts")
		return exitInternal
	}
	// reexecUpdate replaces this process; we never return from here on
	// success.
	//coverage:ignore Reached only when reexecUpdate's syscall.Exec
	// returns without overlaying the process — kernel-level oddity.
	return exitInternal
}

// renderVersionLabel formats an Info for human display: the version
// itself plus a parenthetical state ("tagged", "working-tree build",
// "pseudo-version"). The `+dirty` suffix that Go appends to working-
// tree builds with uncommitted changes is always rendered as a
// working-tree build, regardless of the base version shape.
func renderVersionLabel(info version.Info) string {
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

// resolveTarget turns the --version flag value into a concrete Info.
// The literal "latest" routes through the module proxy; an explicit
// semver is taken at face value (the install step is the eventual
// authority on whether the tag exists).
func resolveTarget(target string) (version.Info, error) {
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

// pathChangedFromStderr scans `go install` stderr for the package-
// path-change signature. Returns the missing package path and true
// when matched; ("", false) otherwise. Filed under G46.
func pathChangedFromStderr(stderr string) (string, bool) {
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
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "aiwf upgrade: hint — the install path may have changed in the target release.")
	fmt.Fprintf(os.Stderr, "  This binary's upgrade verb tried `%s@%s`, but the target tag's\n", pkg, target)
	fmt.Fprintf(os.Stderr, "  module no longer contains that package subpath (`%s`).\n", missingPkg)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Recovery:")
	fmt.Fprintln(os.Stderr, "    1. Find the new install path in the target release's CHANGELOG.")
	fmt.Fprintln(os.Stderr, "       https://github.com/23min/ai-workflow-v2/blob/main/CHANGELOG.md")
	fmt.Fprintf(os.Stderr, "    2. Run `go install <new-path>@%s` manually.\n", target)
	fmt.Fprintln(os.Stderr, "    3. Run `aiwf update` in your consumer repo to refresh artifacts.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  After that, future `aiwf upgrade` runs from the new binary will work end-to-end.")
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
	binDir, err := goBinDir(ctx)
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

// goBinDir returns the directory `go install` writes binaries into.
// Resolution order matches `go install`'s own logic: GOBIN if set,
// else GOPATH/bin (where GOPATH defaults to $HOME/go).
//
// Each variable is queried in its own `go env` call rather than as
// `go env GOBIN GOPATH`. The combined form returns one line per name
// with empty values rendered as a blank line, and the leading blank
// for an unset GOBIN was being silently consumed by strings.TrimSpace
// — see G39 in docs/pocv3/gaps.md for the upgrade-flow regression.
func goBinDir(ctx context.Context) (string, error) {
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

// installLocationHint returns a best-guess absolute path to the
// binary `go install <pkg>` would have produced, derived from the
// caller's environment without invoking `go env`. Used only to help
// the user recover after locateBinary failed; never load-bearing.
// Returns an empty string when neither GOBIN/GOPATH nor a home
// directory can be resolved.
func installLocationHint(pkg string) string {
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
