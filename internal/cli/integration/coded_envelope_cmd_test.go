package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// codedEnvelope mirrors the M-0143 / D-0013 envelope shape: status with
// an additive error:{code,message} object. Only the fields the ACs
// assert are modeled — the parse fails if the binary emits anything but
// a single well-formed JSON object on stdout.
type codedEnvelope struct {
	Tool   string `json:"tool"`
	Status string `json:"status"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Result *struct {
		Subject string `json:"subject"`
	} `json:"result"`
}

// runSplit runs the built binary in root with separate stdout/stderr
// capture and returns (stdout, stderr, exitCode). The JSON envelope is
// emitted on stdout; capturing it apart from stderr lets the test parse
// stdout cleanly and assert stderr is empty in JSON mode (D-0013: JSON
// output is a single clean envelope on stdout).
func runSplit(t *testing.T, root, bin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=aiwf-test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=aiwf-test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("running aiwf %v: %v\nstderr:\n%s", args, err, errb.String())
		}
		return out.String(), errb.String(), ee.ExitCode()
	}
	return out.String(), errb.String(), 0
}

// setupRepoWithEpic inits a fresh repo and adds one epic (E-0001,
// status proposed). Returns (root, binDir).
func setupRepoWithEpic(t *testing.T) (root, bin string) {
	t.Helper()
	bin = testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)
	root = t.TempDir()
	if out, err := testutil.RunGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Platform"); err != nil {
		t.Fatalf("aiwf add epic: %v\n%s", err, out)
	}
	return root, bin
}

// TestCodedEnvelope_VerbRefusal_AC2 is M-0143/AC-2: a mutating verb that
// returns a Coded error, run with --format=json, emits an envelope with
// status:"error", error.code = the structured code (structural field
// access, not substring), error.message set, and exits 1 (D-0013, C2:
// legality refusal unifies with the check-time exit). Two distinct verbs
// / codes prove the surfacing is uniform (the A2 payoff), not special-
// cased to one verb.
func TestCodedEnvelope_VerbRefusal_AC2(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	root, bin := setupRepoWithEpic(t)

	t.Run("fsm-transition-illegal via promote", func(t *testing.T) {
		// E-0001 is `proposed`; proposed->done is FSM-illegal -> FSMTransitionError.
		stdout, stderr, code := runSplit(t, root, bin, "promote", "E-0001", "done", "--format=json")
		if code != 1 {
			t.Errorf("exit = %d, want 1 (ExitFindings, C2)\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
		}
		if stderr != "" {
			t.Errorf("JSON mode must write nothing to stderr; got:\n%s", stderr)
		}
		var env codedEnvelope
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("stdout is not a single JSON envelope: %v\nstdout:\n%s", err, stdout)
		}
		if env.Status != "error" {
			t.Errorf("status = %q, want \"error\"", env.Status)
		}
		if env.Error == nil {
			t.Fatalf("envelope has no error object:\n%s", stdout)
		}
		if env.Error.Code != "fsm-transition-illegal" {
			t.Errorf("error.code = %q, want \"fsm-transition-illegal\"", env.Error.Code)
		}
		if env.Error.Message == "" {
			t.Error("error.message is empty")
		}
	})

	t.Run("authorize-kind-not-allowed via authorize", func(t *testing.T) {
		// Authorizing a non-epic/milestone scope -> AuthorizeKindError.
		if out, err := testutil.RunBin(t, root, filepath.Dir(bin), nil, "add", "gap", "--title", "Scope target"); err != nil {
			t.Fatalf("aiwf add gap: %v\n%s", err, out)
		}
		stdout, stderr, code := runSplit(t, root, bin, "authorize", "G-0001", "--to", "ai/claude", "--format=json")
		if code != 1 {
			t.Errorf("exit = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
		}
		if stderr != "" {
			t.Errorf("JSON mode must write nothing to stderr; got:\n%s", stderr)
		}
		var env codedEnvelope
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("stdout is not a single JSON envelope: %v\nstdout:\n%s", err, stdout)
		}
		if env.Status != "error" || env.Error == nil {
			t.Fatalf("want status:error with an error object; got status=%q error=%v", env.Status, env.Error)
		}
		if env.Error.Code != "authorize-kind-not-allowed" {
			t.Errorf("error.code = %q, want \"authorize-kind-not-allowed\"", env.Error.Code)
		}
	})
}

// TestSuccessEnvelope_FormatJSON covers the success path of the M-0143
// rollout (D-0013): a legal mutating verb run with --format=json emits an
// ok envelope with result:{subject} on stdout and exits 0. This exercises
// the emitSuccess JSON branch the error-path ACs (AC-2/AC-3) do not, and
// pins the success-side envelope shape the decision specifies.
func TestSuccessEnvelope_FormatJSON(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	root, bin := setupRepoWithEpic(t)

	// E-0001 is `proposed`; proposed->active is a legal transition.
	stdout, stderr, code := runSplit(t, root, bin, "promote", "E-0001", "active", "--format=json")
	if code != 0 {
		t.Errorf("exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("JSON mode must write nothing to stderr; got:\n%s", stderr)
	}
	var env codedEnvelope
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("stdout is not a single JSON envelope: %v\nstdout:\n%s", err, stdout)
	}
	if env.Status != "ok" {
		t.Errorf("status = %q, want \"ok\"", env.Status)
	}
	if env.Error != nil {
		t.Errorf("success envelope must have no error object; got %+v", env.Error)
	}
	if env.Result == nil || env.Result.Subject == "" {
		t.Errorf("success envelope must carry result.subject; got %+v", env.Result)
	}
}

// TestCodedEnvelope_NonCodedError_AC3 is M-0143/AC-3: a non-Coded verb
// error (unknown entity id) run with --format=json still produces a
// well-formed envelope — status:"error", error.message set, error.code
// empty/omitted — and exits 2 (ExitUsage, unchanged for non-coded
// errors). This proves the error object is additive and the code field
// is optional.
func TestCodedEnvelope_NonCodedError_AC3(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	root, bin := setupRepoWithEpic(t)

	stdout, stderr, code := runSplit(t, root, bin, "promote", "E-9999", "active", "--format=json")
	if code != 2 {
		t.Errorf("exit = %d, want 2 (ExitUsage; non-coded error)\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("JSON mode must write nothing to stderr; got:\n%s", stderr)
	}
	var env codedEnvelope
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("stdout is not a single JSON envelope: %v\nstdout:\n%s", err, stdout)
	}
	if env.Status != "error" {
		t.Errorf("status = %q, want \"error\"", env.Status)
	}
	if env.Error == nil {
		t.Fatalf("envelope has no error object:\n%s", stdout)
	}
	if env.Error.Message == "" {
		t.Error("error.message is empty")
	}
	if env.Error.Code != "" {
		t.Errorf("error.code = %q, want empty (non-coded error)", env.Error.Code)
	}
}
