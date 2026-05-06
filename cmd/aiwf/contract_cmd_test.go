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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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

// TestRun_CheckIncludesContractFindings: `aiwf check` (the entity-tree
// validator) must also surface contract-config and verify-pass
// findings when bindings exist. This is the pre-push integration:
// the same hook fires both kinds of validation.
func TestRun_CheckIncludesContractFindings(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "contract", "--title", "Public API", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("add contract: %d", rc)
	}

	script := fakeValidatorCLI(t, root)
	// One valid PASS, one valid FAIL → fixture-rejected expected.
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

	captured := captureStdout(t, func() {
		if rc := run([]string{"check", "--root", root}); rc != exitFindings {
			t.Errorf("got rc=%d, want %d (findings)", rc, exitFindings)
		}
	})
	out := string(captured)
	if !strings.Contains(out, "fixture-rejected") {
		t.Errorf("expected `fixture-rejected` finding from `aiwf check`:\n%s", out)
	}
}

// TestRun_CheckSkipsTerminalContracts: a rejected/retired contract
// is excluded from verify, so a binding pointing at a fixture that
// would otherwise fail does not block `aiwf check`.
func TestRun_CheckSkipsTerminalContracts(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "contract", "--title", "Old API", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("add contract: %d", rc)
	}
	if rc := run([]string{"cancel", "--root", root, "--actor", "human/test", "C-001"}); rc != exitOK {
		t.Fatalf("cancel C-001: %d", rc)
	}

	script := fakeValidatorCLI(t, root)
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

	captured := captureStdout(t, func() {
		if rc := run([]string{"check", "--root", root}); rc != exitOK {
			t.Errorf("got rc=%d, want %d (terminal contract skipped)", rc, exitOK)
		}
	})
	out := string(captured)
	if strings.Contains(out, "fixture-rejected") {
		t.Errorf("terminal contract should not produce fixture findings:\n%s", out)
	}
}

// TestRun_ContractVerifyReportsConfigMissingSchema verifies the
// contract-config check fires for a binding whose schema path
// doesn't resolve.
func TestRun_ContractVerifyReportsConfigMissingSchema(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
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

// TestRun_ContractVerify_ValidatorUnavailableIsWarning is the
// load-bearing test for G3: a binding whose validator binary is
// missing must NOT block `aiwf contract verify` (and therefore
// must not block the pre-push hook). The output must contain a
// validator-unavailable warning, and the exit code must be exitOK.
func TestRun_ContractVerify_ValidatorUnavailableIsWarning(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "contract", "--title", "Public API", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("add contract: %d", rc)
	}
	mustWriteFile(t, filepath.Join(root, "schema.cue"), "")
	if err := os.MkdirAll(filepath.Join(root, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Bind to a deliberately-missing validator binary.
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(`aiwf_version: 0.1.0
actor: human/test
contracts:
  validators:
    cue-missing:
      command: /nonexistent/cue-12345
      args: []
  entries:
    - id: C-001
      validator: cue-missing
      schema: schema.cue
      fixtures: fixtures
`), 0o644); err != nil {
		t.Fatal(err)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"contract", "verify", "--root", root}); rc != exitOK {
			t.Errorf("got rc=%d, want %d (warning should NOT block)", rc, exitOK)
		}
	})
	out := string(captured)
	if !strings.Contains(out, "validator-unavailable") {
		t.Errorf("expected validator-unavailable in output:\n%s", out)
	}
	if !strings.Contains(out, "cue-missing") {
		t.Errorf("expected validator name in message:\n%s", out)
	}
}

// TestRun_ContractVerify_StrictValidators_IsError: opt-in strict
// mode upgrades validator-unavailable from warning to error, so
// teams that DO want to enforce validator presence on every
// machine can keep that behavior.
func TestRun_ContractVerify_StrictValidators_IsError(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "contract", "--title", "Public API", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("add contract: %d", rc)
	}
	mustWriteFile(t, filepath.Join(root, "schema.cue"), "")
	if err := os.MkdirAll(filepath.Join(root, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(`aiwf_version: 0.1.0
actor: human/test
contracts:
  strict_validators: true
  validators:
    cue-missing:
      command: /nonexistent/cue-12345
      args: []
  entries:
    - id: C-001
      validator: cue-missing
      schema: schema.cue
      fixtures: fixtures
`), 0o644); err != nil {
		t.Fatal(err)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"contract", "verify", "--root", root}); rc != exitFindings {
			t.Errorf("got rc=%d, want %d (strict mode must block)", rc, exitFindings)
		}
	})
	out := string(captured)
	if !strings.Contains(out, "validator-unavailable") {
		t.Errorf("expected validator-unavailable in output:\n%s", out)
	}
	if !strings.Contains(out, "error") {
		t.Errorf("expected error severity in output:\n%s", out)
	}
}

// TestRun_ContractEndToEnd_FullChain exercises the documented
// onboarding flow as one end-to-end test:
//
//  1. init the repo
//  2. install a custom validator via `recipe install --from <path>`
//  3. add a contract atomically with --validator/--schema/--fixtures
//     (so the entity creation and the binding land in one commit)
//  4. write the schema and fixture files
//  5. verify (clean)
//  6. corrupt a fixture so the validator rejects it
//  7. verify (findings) and confirm `aiwf check` also reports them
//     (pre-push integration parity)
//  8. unbind and verify clean again
func TestRun_ContractEndToEnd_FullChain(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}

	// --- Step: install a custom validator from a YAML file. ---
	script := fakeValidatorCLI(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte(`name: fake
command: `+script+`
args:
  - "{{fixture}}"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := run([]string{"contract", "recipe", "install", "--from", customPath, "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("recipe install --from: %d", rc)
	}

	// --- Step: prepare schema + valid fixture on disk. ---
	mustWriteFile(t, filepath.Join(root, "schema.cue"), "")
	writeFixtureFile(t, root, "fixtures/v1/valid/good.json", "PASS")

	// --- Step: atomic add+bind. The verb must produce a single
	// commit that creates the contract entity AND adds the binding
	// to aiwf.yaml. We verify both artifacts after the run.
	if rc := run([]string{
		"add", "contract", "--title", "Public API", "--root", root, "--actor", "human/test",
		"--validator", "fake", "--schema", "schema.cue", "--fixtures", "fixtures",
	}); rc != exitOK {
		t.Fatalf("add contract atomic: %d", rc)
	}
	if _, err := os.Stat(filepath.Join(root, "work", "contracts", "C-001-public-api", "contract.md")); err != nil {
		t.Errorf("contract entity file missing: %v", err)
	}
	yamlBytes, err := os.ReadFile(filepath.Join(root, "aiwf.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(yamlBytes), "C-001") {
		t.Errorf("aiwf.yaml does not carry the new binding:\n%s", yamlBytes)
	}

	// --- Step: verify is clean. ---
	if rc := run([]string{"contract", "verify", "--root", root}); rc != exitOK {
		t.Errorf("verify (initial clean): rc=%d", rc)
	}
	if rc := run([]string{"check", "--root", root}); rc != exitOK {
		t.Errorf("check (initial clean): rc=%d", rc)
	}

	// --- Step: corrupt the fixture so the validator rejects it. ---
	writeFixtureFile(t, root, "fixtures/v1/valid/good.json", "FAIL")

	if rc := run([]string{"contract", "verify", "--root", root}); rc != exitFindings {
		t.Errorf("verify after corrupt: rc=%d, want %d", rc, exitFindings)
	}
	if rc := run([]string{"check", "--root", root}); rc != exitFindings {
		t.Errorf("check after corrupt (pre-push integration): rc=%d, want %d", rc, exitFindings)
	}

	// --- Step: unbind. The contract entity stays; verification stops. ---
	if rc := run([]string{"contract", "unbind", "--root", root, "--actor", "human/test", "C-001"}); rc != exitOK {
		t.Fatalf("unbind: %d", rc)
	}
	if rc := run([]string{"contract", "verify", "--root", root}); rc != exitOK {
		t.Errorf("verify after unbind: rc=%d (binding gone, should be silent)", rc)
	}

	// Confirm the entity still exists despite the unbind.
	if rc := run([]string{"check", "--root", root}); rc == exitInternal {
		t.Errorf("check after unbind returned internal error rc=%d", rc)
	}
}

// TestRun_ContractRecipeInstallIsIdempotent: running install twice
// with the same recipe must succeed both times, with the second run
// printing a no-op message and not creating a second commit.
func TestRun_ContractRecipeInstallIsIdempotent(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}

	script := fakeValidatorCLI(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte(`name: fake
command: `+script+`
args:
  - "{{fixture}}"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if rc := run([]string{"contract", "recipe", "install", "--from", customPath, "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("recipe install (first): %d", rc)
	}
	captured := captureStdout(t, func() {
		if rc := run([]string{"contract", "recipe", "install", "--from", customPath, "--root", root, "--actor", "human/test"}); rc != exitOK {
			t.Errorf("recipe install (second, idempotent): %d", rc)
		}
	})
	if !strings.Contains(string(captured), "unchanged") {
		t.Errorf("idempotent install did not print no-op message:\n%s", captured)
	}
}

// TestRun_ContractRecipeRemoveRefuses_WhenBindingExists: the verb
// must error out (exit 2) and name the offending binding.
func TestRun_ContractRecipeRemoveRefusesWhenBindingExists(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	script := fakeValidatorCLI(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte(`name: fake
command: `+script+`
args: ["{{fixture}}"]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := run([]string{"contract", "recipe", "install", "--from", customPath, "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("recipe install: %d", rc)
	}
	if rc := run([]string{"add", "contract", "--title", "API", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("add contract: %d", rc)
	}
	mustWriteFile(t, filepath.Join(root, "schema.cue"), "")
	writeFixtureFile(t, root, "fixtures/v1/valid/a.json", "PASS")
	if rc := run([]string{
		"contract", "bind", "--root", root, "--actor", "human/test", "C-001",
		"--validator", "fake", "--schema", "schema.cue", "--fixtures", "fixtures",
	}); rc != exitOK {
		t.Fatalf("bind: %d", rc)
	}

	// Now try to remove the validator. Must refuse, naming C-001.
	captured := captureStderr(t, func() {
		if rc := run([]string{"contract", "recipe", "remove", "--root", root, "--actor", "human/test", "fake"}); rc != exitUsage {
			t.Errorf("recipe remove with binding: rc=%d, want %d", rc, exitUsage)
		}
	})
	if !strings.Contains(string(captured), "C-001") {
		t.Errorf("remove error did not name the binding:\n%s", captured)
	}
}

// TestRun_ContractAddPartialBindFlagsRejected: --validator alone (or
// any 1-of-3 / 2-of-3 of the bind flags) must error out as a usage
// problem and leave no entity behind.
func TestRun_ContractAddPartialBindFlagsRejected(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	script := fakeValidatorCLI(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte(`name: fake
command: `+script+`
args: ["{{fixture}}"]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := run([]string{"contract", "recipe", "install", "--from", customPath, "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("recipe install: %d", rc)
	}
	if rc := run([]string{
		"add", "contract", "--title", "API", "--root", root, "--actor", "human/test",
		"--validator", "fake",
	}); rc != exitUsage {
		t.Errorf("partial-triplet add: rc=%d, want %d", rc, exitUsage)
	}
	// Confirm no entity was created.
	if _, err := os.Stat(filepath.Join(root, "work", "contracts", "C-001-api", "contract.md")); err == nil {
		t.Errorf("contract entity created despite usage error")
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
