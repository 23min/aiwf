package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/23min/ai-workflow-v2/tools/internal/version"
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

	if installErr := runGoInstall(context.Background(), installArg); installErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf upgrade: %v\n", installErr)
		return exitInternal
	}

	newBinary, err := installedBinaryPath(context.Background(), pkg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf upgrade: install succeeded, but locating the new binary failed: %v\n", err)
		fmt.Fprintln(os.Stderr, "                run `aiwf update` manually to refresh consumer artifacts")
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

// runGoInstall shells out to `go install <arg>` with stdout/stderr
// streamed through to the user. Returns a wrapped error on non-zero
// exit. The `go` binary path is honored from AIWF_GO_BIN when set,
// otherwise discovered via exec.LookPath.
func runGoInstall(ctx context.Context, arg string) error {
	goBin, err := goBinaryPath()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, goBin, "install", arg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("`go install %s`: %w", arg, err)
	}
	return nil
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
func goBinDir(ctx context.Context) (string, error) {
	goBin, err := goBinaryPath()
	if err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, goBin, "env", "GOBIN", "GOPATH").Output()
	if err != nil {
		return "", fmt.Errorf("`go env GOBIN GOPATH`: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("unexpected `go env` output: %q", string(out))
	}
	gobin := strings.TrimSpace(lines[0])
	gopath := strings.TrimSpace(lines[1])
	if gobin != "" {
		return gobin, nil
	}
	if gopath == "" {
		return "", errors.New("`go env GOPATH` is empty")
	}
	return filepath.Join(gopath, "bin"), nil
}

// reexecUpdate overlays the current process with the new binary
// running `aiwf update --root <rootDir>`. On success, this function
// does not return — control transfers to the new binary.
var reexecUpdate = func(newBinary, rootDir string) error {
	args := []string{filepath.Base(newBinary), "update", "--root", rootDir}
	return syscall.Exec(newBinary, args, os.Environ())
}
