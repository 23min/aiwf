package policies_test

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

func TestDevcontainerCodexInstall(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		existing   bool
		extension  bool
		npmFailure string
		wantExit   int
	}{
		{name: "missing npm binary is installed"},
		{name: "extension binary does not replace npm installation", extension: true},
		{name: "existing npm binary is retained", existing: true},
		{name: "prefix failure stops initialization", npmFailure: "prefix", wantExit: 42},
		{name: "install failure stops initialization", npmFailure: "install", wantExit: 43},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			prefix := filepath.Join(dir, "npm prefix")
			for _, path := range []string{bin, filepath.Join(prefix, "bin")} {
				if err := os.MkdirAll(path, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range []string{"git", "make", "golangci-lint", "gofumpt", "goimports", "govulncheck", "gitleaks", "claude"} {
				devcontainerExecutable(t, filepath.Join(bin, name), "#!/bin/sh\nexit 0\n")
			}
			devcontainerExecutable(t, filepath.Join(bin, "curl"), "#!/bin/sh\nexit 99\n")
			devcontainerExecutable(t, filepath.Join(bin, "go"), "#!/bin/sh\nif [ \"$1\" = env ]; then printf '%s\\n' \"$FIXTURE_DIR/go\"; fi\n")
			devcontainerExecutable(t, filepath.Join(bin, "aiwf"), "#!/bin/sh\nprintf 'aiwf %s\\n' \"$*\" >> \"$FIXTURE_DIR/calls\"\n")
			devcontainerExecutable(t, filepath.Join(bin, "npm"), `#!/bin/sh
set -eu
case "$*" in
  'prefix -g')
    [ "$NPM_FAILURE" != prefix ] || exit 42
    printf '%s\n' "$FIXTURE_DIR/npm prefix"
    ;;
  'install -g @openai/codex')
    [ "$NPM_FAILURE" != install ] || exit 43
    printf 'install\n' >> "$FIXTURE_DIR/calls"
    cp "$FIXTURE_DIR/npm-codex" "$FIXTURE_DIR/npm prefix/bin/codex"
    ;;
  *) exit 98 ;;
esac
`)
			const codex = "#!/bin/sh\nprintf 'npm codex fixture\\n'\n"
			devcontainerExecutable(t, filepath.Join(dir, "npm-codex"), codex)
			if tc.existing {
				devcontainerExecutable(t, filepath.Join(prefix, "bin", "codex"), codex)
			}
			if tc.extension {
				devcontainerExecutable(t, filepath.Join(bin, "codex"), "#!/bin/sh\nprintf 'extension codex fixture\\n'\n")
			}
			env := []string{"HOME=" + dir, "PATH=" + bin + ":/usr/bin:/bin", "FIXTURE_DIR=" + dir, "NPM_FAILURE=" + tc.npmFailure}
			for range 2 {
				cmd := exec.CommandContext(t.Context(), "bash", filepath.Join(repoRootForHook(t), ".devcontainer", "init.sh"))
				cmd.Dir, cmd.Env = dir, env
				out, err := cmd.CombinedOutput()
				var exitErr *exec.ExitError
				exitCode := 0
				if errors.As(err, &exitErr) {
					exitCode = exitErr.ExitCode()
				} else if err != nil {
					t.Fatal(err)
				}
				if exitCode != tc.wantExit {
					t.Fatalf("initialization exit = %d, want %d:\n%s", exitCode, tc.wantExit, out)
				}
			}
			calls, err := os.ReadFile(filepath.Join(dir, "calls"))
			if tc.wantExit != 0 {
				if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("failed npm operation must stop before aiwf: calls=%q, err=%v", calls, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			wantCalls := "aiwf init --no-prompt\naiwf init --no-prompt\n"
			if !tc.existing {
				wantCalls = "install\n" + wantCalls
			}
			if string(calls) != wantCalls {
				t.Errorf("calls = %q, want %q", calls, wantCalls)
			}
			got, err := os.ReadFile(filepath.Join(prefix, "bin", "codex"))
			if err != nil || string(got) != codex {
				t.Fatalf("npm binary not retained: content=%q, err=%v", got, err)
			}
		})
	}
}

func TestDevcontainerCodexStateMount(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bin, mounts := filepath.Join(dir, "bin"), filepath.Join(dir, "mounts")
	for _, path := range []string{bin, mounts} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	// Redirect only the process's /tmp mount destinations into this fixture;
	// mkdir, chmod and the underlying symlink operation use real filesystems.
	devcontainerExecutable(t, filepath.Join(bin, "ln"), `#!/bin/sh
set -eu
[ "$1" = -sfn ]
case "$3" in /tmp/.*-mount) ;; *) exit 98 ;; esac
exec /bin/ln -sfn "$2" "$FIXTURE_DIR/mounts/${3##*/}"
`)
	root := repoRootForHook(t)
	for run := range 2 {
		cmd := exec.CommandContext(t.Context(), "bash", filepath.Join(root, ".devcontainer", "initialize.sh"))
		cmd.Dir = dir
		cmd.Env = []string{"HOME=" + dir, "PATH=" + bin + ":/usr/bin:/bin", "FIXTURE_DIR=" + dir}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("initialize host: %v\n%s", err, out)
		}
		state := filepath.Join(dir, ".codex-linux")
		info, err := os.Stat(state)
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() || info.Mode().Perm() != 0o700 {
			t.Fatalf("state must be a private directory, got %v", info.Mode())
		}
		marker := filepath.Join(state, "session-marker")
		if run == 0 {
			if err := os.WriteFile(marker, []byte("retained session"), 0o600); err != nil {
				t.Fatal(err)
			}
		} else if got, err := os.ReadFile(marker); err != nil || string(got) != "retained session" {
			t.Fatalf("initialization changed existing state: %q, %v", got, err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, ".devcontainer", "devcontainer.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Mounts []string `json:"mounts"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, mount := range config.Mounts {
		fields := make(map[string]string)
		for _, part := range strings.Split(mount, ",") {
			key, value, _ := strings.Cut(part, "=")
			fields[key] = value
		}
		if fields["target"] != "/home/vscode/.codex" {
			continue
		}
		found = true
		if fields["type"] != "bind" {
			t.Fatalf("Codex state must be bind-mounted: %s", mount)
		}
		if !strings.HasPrefix(fields["source"], "/tmp/") {
			t.Fatalf("mount source must use the host's prepared /tmp path: %s", mount)
		}
		got, err := filepath.EvalSymlinks(filepath.Join(mounts, strings.TrimPrefix(fields["source"], "/tmp/")))
		if err != nil || got != filepath.Join(dir, ".codex-linux") {
			t.Fatalf("mount does not resolve to host state: %q, %v", got, err)
		}
	}
	if !found {
		t.Fatal("Codex state mount missing")
	}
}

func devcontainerExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := testsupport.WriteExecutable(path, []byte(body)); err != nil {
		t.Fatal(err)
	}
}
