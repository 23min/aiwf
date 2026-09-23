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
	"github.com/23min/aiwf/internal/testsupport"
)

func TestGuidanceSelection_RealTerminalThroughInitAndUpdate(t *testing.T) {
	// Serial: replaces process stdin with an actual terminal.
	testsupport.IsolateGuidanceEnvironment(t)
	for _, verb := range []string{"init", "update", "init-no-prompt"} {
		t.Run(verb, func(t *testing.T) {
			master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
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
			defer slave.Close()
			previous := os.Stdin
			os.Stdin = slave
			defer func() { os.Stdin = previous }()
			root, source := t.TempDir(), testsupport.GuidanceSource(t, "*.xyz")
			if err = gitops.Init(t.Context(), root); err != nil {
				t.Fatal(err)
			}

			for name, content := range map[string]string{config.FileName: "hosts: []\nguidance:\n  source: " + source + "\n", "source.xyz": "source"} {
				if err = os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = fmt.Fprintln(master, "s"); err != nil {
				t.Fatal(err)
			}
			var rc int
			output := testutil.CaptureStderr(t, func() {
				if verb != "update" {
					rc = initcmd.Run(root, "", false, true, false, "", false, false, verb == "init-no-prompt", nil, nil)
				} else {
					rc = update.Run(root, false, "", false, false, false, false, nil, nil)
				}
			})
			if rc != cliutil.ExitOK {
				t.Fatalf("%s exit %d", verb, rc)
			}
			if verb == "init-no-prompt" {
				if !bytes.Contains(output, []byte("Suggested guidance sample/base")) {
					t.Fatalf("no-prompt omitted suggestion: %s", output)
				}
				if _, err = os.Stat(filepath.Join(root, ".guidance", "index.md")); !errors.Is(err, os.ErrNotExist) {
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
