//go:build linux

package cliutil_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/initcmd"
	"github.com/23min/aiwf/internal/cli/update"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/skills"
	"github.com/23min/aiwf/internal/testsupport"
)

func TestGuidanceSelection_RealTerminalThroughInitAndUpdate(t *testing.T) {
	// Serial: replaces process stdin with an actual terminal.
	testsupport.IsolateGuidanceEnvironment(t)
	for _, verb := range []string{"init", "update", "init-no-prompt", "update-no-prompt"} {
		t.Run(verb, func(t *testing.T) {
			master := terminalStdin(t)
			root, source := t.TempDir(), testsupport.GuidanceSource(t, "*.xyz")
			if err := gitops.Init(t.Context(), root); err != nil {
				t.Fatal(err)
			}

			for name, content := range map[string]string{config.FileName: "hosts: []\nguidance:\n  source: " + source + "\n", "source.xyz": "source"} {
				if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := fmt.Fprintln(master, "s"); err != nil {
				t.Fatal(err)
			}
			var rc int
			output := testutil.CaptureStderr(t, func() {
				if verb == "init" || verb == "init-no-prompt" {
					rc = initcmd.Run(root, "", false, true, false, "", false, false, verb == "init-no-prompt", nil, nil)
				} else {
					rc = runUpdateCommand(t, root, verb == "update-no-prompt")
				}
			})
			if rc != cliutil.ExitOK {
				t.Fatalf("%s exit %d", verb, rc)
			}
			if verb == "init-no-prompt" || verb == "update-no-prompt" {
				if !bytes.Contains(output, []byte("Suggested guidance sample/base")) {
					t.Fatalf("no-prompt omitted suggestion: %s", output)
				}
				if _, err := os.Stat(filepath.Join(root, ".guidance", "index.md")); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("no-prompt adopted policy: %v", err)
				}
				return
			}
			installed, err := os.ReadFile(filepath.Join(root, ".guidance", "packs", "sample", "base", "guide.md"))
			if err != nil || string(installed) != "# Sample guidance\nInitial upstream content.\n" {
				t.Fatalf("terminal selection not installed: %s, %v", installed, err)
			}
		})
	}
}

// TestHookConsent_RealTerminalThroughUpdate drives update's hook-consent gate
// on a real terminal with an answer ("y") already waiting for each shipped hook. Without --no-prompt the
// gate reads it and records the hook; with --no-prompt the same waiting answer
// is never read and the hook stays undecided.
func TestHookConsent_RealTerminalThroughUpdate(t *testing.T) {
	// Serial: replaces process stdin with an actual terminal.
	testsupport.IsolateGuidanceEnvironment(t)
	for _, tc := range []struct {
		name        string
		noPrompt    bool
		wantDecided bool
	}{
		{name: "prompt", noPrompt: false, wantDecided: true},
		{name: "no-prompt", noPrompt: true, wantDecided: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			master := terminalStdin(t)
			root := t.TempDir()
			if err := gitops.Init(t.Context(), root); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, config.FileName), []byte("hosts: [claude-code]\nguidance:\n  enabled: false\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := fmt.Fprintln(master, "y"); err != nil {
				t.Fatal(err)
			}
			hooks := skills.ShippedHooks
			var rc int
			testutil.CaptureStderr(t, func() {
				rc = runUpdateCommand(t, root, tc.noPrompt)
			})
			if rc != cliutil.ExitOK {
				t.Fatalf("update exit %d", rc)
			}
			cfg, err := config.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			for _, hook := range hooks {
				enabled, decided := cfg.HookDecision(hook.Name)
				if decided != tc.wantDecided || (decided && !enabled) {
					t.Fatalf("HookDecision(%s) = (enabled %v, decided %v), want decided %v", hook.Name, enabled, decided, tc.wantDecided)
				}
			}
		})
	}
}

// runUpdateCommand runs `aiwf update` through its command-line parsing, so
// the --no-prompt flag is exercised from argv through to the prompts.
func runUpdateCommand(t *testing.T, root string, noPrompt bool) int {
	t.Helper()
	args := []string{"--root", root}
	if noPrompt {
		args = append(args, "--no-prompt")
	}
	cmd := update.NewCmd()
	cmd.SetArgs(args)
	err := cmd.Execute()
	var exitErr *cliutil.ExitError
	switch {
	case err == nil:
		return cliutil.ExitOK
	case errors.As(err, &exitErr):
		return exitErr.Code
	default:
		t.Fatalf("update: %v", err)
		return cliutil.ExitInternal
	}
}

// terminalStdin replaces os.Stdin with the slave side of a fresh
// pseudo-terminal for the rest of the test and returns the master side,
// which the test writes answers into.
func terminalStdin(t *testing.T) *os.File {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { master.Close() })
	if err = unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	number, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stdin
	os.Stdin = slave
	t.Cleanup(func() {
		os.Stdin = previous
		slave.Close()
	})
	return master
}
