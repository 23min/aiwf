package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

func TestRunCommitMsg_PassThroughKernelVerb(t *testing.T) {
	t.Parallel()
	path := writeMsg(t, "feat(check): add the thing\n\nLong-form rationale.\n\naiwf-verb: promote\naiwf-entity: M-0001\n")
	var buf bytes.Buffer
	code := runCommitMsg(path, map[string]struct{}{"promote": {}, "add": {}}, &buf)
	if code != cliutil.ExitOK {
		t.Errorf("kernel verb: code=%d want %d; stderr=%q", code, cliutil.ExitOK, buf.String())
	}
}

// ritualVerbs derive from the embedded snapshot per G-0190; not stubbed.
func TestRunCommitMsg_PassThroughRitualVerb(t *testing.T) {
	t.Parallel()
	path := writeMsg(t, "chore(epic): wrap E-0030\n\naiwf-verb: wrap-epic\naiwf-entity: E-0030\n")
	var buf bytes.Buffer
	code := runCommitMsg(path, map[string]struct{}{"promote": {}, "add": {}}, &buf)
	if code != cliutil.ExitOK {
		t.Errorf("ritual verb: code=%d want %d; stderr=%q", code, cliutil.ExitOK, buf.String())
	}
}

func TestRunCommitMsg_NoTrailer(t *testing.T) {
	t.Parallel()
	path := writeMsg(t, "chore: refactor the thing\n\nSome rationale.\n")
	var buf bytes.Buffer
	code := runCommitMsg(path, map[string]struct{}{"promote": {}}, &buf)
	if code != cliutil.ExitOK {
		t.Errorf("no aiwf-verb: code=%d want %d; stderr=%q", code, cliutil.ExitOK, buf.String())
	}
}

// G-0218 canonical case: `aiwf-verb: merge` (a git concept, not a
// Cobra verb, not a ritual). Refusal + stderr names the value and
// the closed-set guidance.
func TestRunCommitMsg_RefusesFabricated(t *testing.T) {
	t.Parallel()
	path := writeMsg(t, "chore(epic): merge milestone\n\naiwf-verb: merge\naiwf-entity: M-0160\n")
	var buf bytes.Buffer
	code := runCommitMsg(path, map[string]struct{}{"promote": {}, "wrap-epic": {}}, &buf)
	if code == cliutil.ExitOK {
		t.Errorf("fabricated trailer: code=%d want non-zero", code)
	}
	stderr := buf.String()
	if !strings.Contains(stderr, "merge") {
		t.Errorf("stderr missing 'merge':\n%s", stderr)
	}
	if !strings.Contains(stderr, "ritualVerbs") {
		t.Errorf("stderr missing closed-set guidance:\n%s", stderr)
	}
}

// Two distinct fabricated values surface together (not just the first).
func TestRunCommitMsg_RefusesMultipleFabricated(t *testing.T) {
	t.Parallel()
	path := writeMsg(t, "wip\n\naiwf-verb: implement\naiwf-verb: bogus\n")
	var buf bytes.Buffer
	code := runCommitMsg(path, map[string]struct{}{"promote": {}}, &buf)
	if code == cliutil.ExitOK {
		t.Errorf("multi-fabricated: code=%d want non-zero", code)
	}
	stderr := buf.String()
	if !strings.Contains(stderr, "implement") || !strings.Contains(stderr, "bogus") {
		t.Errorf("stderr missing one of the fabricated values:\n%s", stderr)
	}
}

// BUG-1 / GAP-2: a registered + a fabricated trailer on the same
// commit refuses (the registered one passes, the fabricated one is
// still surfaced). Without this case a future "exit OK as soon as
// one trailer is recognized" regression would slip through.
func TestRunCommitMsg_RefusesMixedRegisteredAndFabricated(t *testing.T) {
	t.Parallel()
	path := writeMsg(t, "wip\n\naiwf-verb: promote\naiwf-verb: bogus\n")
	var buf bytes.Buffer
	code := runCommitMsg(path, map[string]struct{}{"promote": {}}, &buf)
	if code == cliutil.ExitOK {
		t.Errorf("mixed registered+fabricated: code=%d want non-zero; stderr=%q", code, buf.String())
	}
	if !strings.Contains(buf.String(), "bogus") {
		t.Errorf("stderr missing 'bogus':\n%s", buf.String())
	}
}

// BUG-1: `aiwf-verb: ` (trailing space, empty value) is the canonical
// "started typing the trailer and didn't finish" — must refuse, not
// silently pass. The post-hoc trailer-verb-unknown rule skips empties
// because it walks already-landed history; the hook is prescriptive
// at composition time and the asymmetry is the chokepoint's point.
func TestRunCommitMsg_RefusesEmptyValue(t *testing.T) {
	t.Parallel()
	path := writeMsg(t, "wip\n\naiwf-verb: \n")
	var buf bytes.Buffer
	code := runCommitMsg(path, map[string]struct{}{"promote": {}}, &buf)
	if code == cliutil.ExitOK {
		t.Errorf("empty value: code=%d want non-zero; stderr=%q", code, buf.String())
	}
	if !strings.Contains(buf.String(), "malformed") {
		t.Errorf("stderr missing empty-value remediation hint:\n%s", buf.String())
	}
}

// GAP-5: trailer keys are case-sensitive per gitops.ValidateTrailer.
// A miscased `Aiwf-Verb:` is silently ignored by THIS hook (it's
// policed by the trailer-keys policy elsewhere). Pin the contract
// so a future "case-insensitive match" change can't drift the two
// recognition surfaces apart.
func TestRunCommitMsg_IgnoresMiscasedKey(t *testing.T) {
	t.Parallel()
	path := writeMsg(t, "wip\n\nAiwf-Verb: bogus\n")
	var buf bytes.Buffer
	code := runCommitMsg(path, map[string]struct{}{"promote": {}}, &buf)
	if code != cliutil.ExitOK {
		t.Errorf("miscased key: code=%d want ExitOK (case-sensitivity is the kernel-wide contract); stderr=%q", code, buf.String())
	}
}

// GAP-3: non-UTF-8 bytes in the message file don't crash the
// validator and pass through silently when no aiwf-verb is present.
func TestRunCommitMsg_NonUTF8(t *testing.T) {
	t.Parallel()
	body := []byte{0xc3, 0x28, '\n', '\n', 'b', 'o', 'd', 'y', '\n'}
	path := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	var buf bytes.Buffer
	code := runCommitMsg(path, map[string]struct{}{"promote": {}}, &buf)
	if code != cliutil.ExitOK {
		t.Errorf("non-utf8: code=%d want ExitOK; stderr=%q", code, buf.String())
	}
}

func TestRunCommitMsg_EmptyPath(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	code := runCommitMsg("", map[string]struct{}{"promote": {}}, &buf)
	if code != cliutil.ExitUsage {
		t.Errorf("empty path: code=%d want %d", code, cliutil.ExitUsage)
	}
}

func TestRunCommitMsg_MissingFile(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	code := runCommitMsg(filepath.Join(t.TempDir(), "no-such"), map[string]struct{}{"promote": {}}, &buf)
	if code != cliutil.ExitUsage {
		t.Errorf("missing file: code=%d want %d", code, cliutil.ExitUsage)
	}
}

// Body prose that LOOKS like an aiwf-verb trailer (e.g. a commit
// message DISCUSSING `aiwf-verb: implement` as an example) must
// not be parsed as a trailer. Without `git interpret-trailers
// --parse`, the canonical "commit message explaining the G-0218
// fix" — which mentions fabricated values in its body before the
// real trailer block — would be falsely refused. This pins the
// extraction's trailer-block-only semantics.
func TestRunCommitMsg_BodyProseIsNotTrailerBlock(t *testing.T) {
	t.Parallel()
	path := writeMsg(t, "chore(check): close G-0218\n\n"+
		"Prior to this change, the kernel allowed fabricated\n"+
		"values in the trailer slot like:\n"+
		"\n"+
		"    aiwf-verb: merge\n"+
		"    aiwf-verb: implement\n"+
		"\n"+
		"— both fabrications. This patch lands the chokepoint.\n"+
		"\n"+
		"aiwf-verb: promote\n"+
		"aiwf-entity: G-0218\n")
	var buf bytes.Buffer
	code := runCommitMsg(path, map[string]struct{}{"promote": {}}, &buf)
	if code != cliutil.ExitOK {
		t.Errorf("body-prose with example-trailers + real trailer block: code=%d want %d (the prose lines must not be parsed as trailers); stderr=%q", code, cliutil.ExitOK, buf.String())
	}
}

func TestNewCmd_HasCommitMsgFlag(t *testing.T) {
	t.Parallel()
	cmd := NewCmd()
	if cmd.Flags().Lookup("commit-msg") == nil {
		t.Error("flag --commit-msg missing on aiwf check")
	}
}

func writeMsg(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write commit-msg fixture: %v", err)
	}
	return path
}
