package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeValidatorCLI writes the same fake validator the contractverify
// tests use into dir and returns the absolute path. Skips on Windows.
func fakeValidatorCLI(t *testing.T, dir string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("contract verify CLI tests use a /bin/sh script; skipping on Windows")
	}
	path := filepath.Join(dir, "fake-validator.sh")
	body := `#!/bin/sh
fixture="$1"
[ -f "$fixture" ] || { echo "fixture not found: $fixture" >&2; exit 2; }
case "$(head -c 4 "$fixture")" in
  PASS) exit 0 ;;
  *) echo "rejected: $fixture" >&2; exit 1 ;;
esac
`
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("writing fake validator: %v", err)
	}
	return path
}

// TestRun_ContractVerifyClean exercises the `aiwf contract verify`
// dispatcher end-to-end: init the repo, register a contract entity,
// write a `contracts:` block + on-disk fixtures, and assert exit
// status is exitOK with no findings.
func TestRun_ContractVerifyClean(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "contract", "--title", "Public API", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("add contract: %d", rc)
	}

	script := fakeValidatorCLI(t, root)
	writeFixtureFile(t, root, "fixtures/v1/valid/a.json", "PASS")
	writeFixtureFile(t, root, "fixtures/v1/invalid/b.json", "FAIL")
	mustWriteFile(t, filepath.Join(root, "schema.cue"), "")

	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(`aiwf_version: 0.1.0
actor: human/test
contracts:
  validators:
    fake:
      command: `+script+`
      args:
        - "{{fixture}}"
  entries:
    - id: C-001
      validator: fake
      schema: schema.cue
      fixtures: fixtures
`), 0o644); err != nil {
		t.Fatal(err)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"contract", "verify", "--root", root}); rc != exitOK {
			t.Errorf("got rc=%d, want %d", rc, exitOK)
		}
	})
	out := string(captured)
	if !strings.Contains(out, "ok") {
		t.Errorf("expected an 'ok' line; got:\n%s", out)
	}
}

// TestRun_ContractVerifyReportsFixtureRejected exercises the failure
// path: a valid fixture that the fake validator rejects produces an
// error-severity finding and a non-zero exit code.
func TestRun_ContractVerifyReportsFixtureRejected(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "contract", "--title", "Public API", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("add contract: %d", rc)
	}

	script := fakeValidatorCLI(t, root)
	// Two valid fixtures: one passes, one fails. (Both failing would
	// trigger the validator-error reclassification, which we cover
	// in the contractverify package tests.)
	writeFixtureFile(t, root, "fixtures/v1/valid/good.json", "PASS")
	writeFixtureFile(t, root, "fixtures/v1/valid/bad.json", "FAIL")
	mustWriteFile(t, filepath.Join(root, "schema.cue"), "")

	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(`aiwf_version: 0.1.0
actor: human/test
contracts:
  validators:
    fake:
      command: `+script+`
      args:
        - "{{fixture}}"
  entries:
    - id: C-001
      validator: fake
      schema: schema.cue
      fixtures: fixtures
`), 0o644); err != nil {
		t.Fatal(err)
	}

	captured := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			if rc := run([]string{"contract", "verify", "--root", root}); rc != exitFindings {
				t.Errorf("got rc=%d, want %d (findings)", rc, exitFindings)
			}
		})
	})
	_ = captured
}

// TestRun_ContractVerifyReportsConfigMissingSchema verifies the
// contract-config check fires for a binding whose schema path
// doesn't resolve.
func TestRun_ContractVerifyReportsConfigMissingSchema(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "contract", "--title", "Public API", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("add contract: %d", rc)
	}

	script := fakeValidatorCLI(t, root)
	if err := os.MkdirAll(filepath.Join(root, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	// schema.cue intentionally not created.

	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(`aiwf_version: 0.1.0
actor: human/test
contracts:
  validators:
    fake:
      command: `+script+`
      args:
        - "{{fixture}}"
  entries:
    - id: C-001
      validator: fake
      schema: schema.cue
      fixtures: fixtures
`), 0o644); err != nil {
		t.Fatal(err)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"contract", "verify", "--root", root}); rc != exitFindings {
			t.Errorf("got rc=%d, want %d (findings)", rc, exitFindings)
		}
	})
	out := string(captured)
	if !strings.Contains(out, "contract-config") || !strings.Contains(out, "schema") {
		t.Errorf("expected a contract-config/missing-schema finding in output:\n%s", out)
	}
}

// captureStderr is a sibling of captureStdout used to silence noise
// during verify-failure runs (the renderer writes findings to stdout
// but verb-level errors go to stderr).
func captureStderr(t *testing.T, fn func()) []byte {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	doneCh := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 0, 4096)
		tmp := make([]byte, 1024)
		for {
			n, err := r.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
			}
			if err != nil {
				break
			}
		}
		doneCh <- buf
	}()
	fn()
	_ = w.Close()
	return <-doneCh
}

// writeFixtureFile creates a fixture file at <root>/<rel> with the
// given content, making any parent directories as needed.
func writeFixtureFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mustWriteFile writes content to path, creating parent dirs.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
