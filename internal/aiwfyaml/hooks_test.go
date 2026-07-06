package aiwfyaml

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHooks_NoBlock(t *testing.T) {
	t.Parallel()
	d, _, err := ReadBytes([]byte(baseConfig))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	got, err := d.Hooks()
	if err != nil {
		t.Fatalf("Hooks: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Hooks() = %#v, want empty map for a file with no hooks: block", got)
	}
}

// TestHooks_DecodesExistingBlock pins the undecided-vs-decided distinction
// (ADR-0032): an entry present but omitting `enabled:` must not surface as
// a decision either way — only entries with an explicit enabled: value do.
func TestHooks_DecodesExistingBlock(t *testing.T) {
	t.Parallel()
	src := `hosts: [claude-code]
hooks:
  hook-a:
    enabled: true
  hook-b:
    enabled: false
  hook-c: {}
`
	d, _, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	got, err := d.Hooks()
	if err != nil {
		t.Fatalf("Hooks: %v", err)
	}
	want := map[string]bool{"hook-a": true, "hook-b": false}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Hooks() mismatch (-want +got):\n%s", diff)
	}
}

// TestHooks_RejectsUnknownFieldInEntry pins the KnownFields(true) strict
// decode: a hooks: entry carrying a key other than enabled: fails rather
// than silently ignoring it, mirroring decodeContracts' strictness.
func TestHooks_RejectsUnknownFieldInEntry(t *testing.T) {
	t.Parallel()
	src := `hooks:
  hook-a:
    enabled: true
    unknown_field: nonsense
`
	d, _, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if _, err := d.Hooks(); err == nil {
		t.Fatal("Hooks() = nil error, want a decode error for the unknown field")
	}
}

// TestReadBytes_DetectsHooksWithoutContracts mirrors
// TestReadBytes_DetectsAreasWithoutContracts: the hooks: block is detected
// independently of whether a contracts: block is present.
func TestReadBytes_DetectsHooksWithoutContracts(t *testing.T) {
	t.Parallel()
	src := `hosts: [claude-code]
hooks:
  hook-a:
    enabled: true
`
	d, contracts, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if contracts != nil {
		t.Errorf("contracts = %+v, want nil", contracts)
	}
	got, err := d.Hooks()
	if err != nil {
		t.Fatalf("Hooks: %v", err)
	}
	if !got["hook-a"] {
		t.Errorf("Hooks() = %#v, want hook-a: true", got)
	}
}

func TestSetHooks_AppendsWhenAbsent(t *testing.T) {
	t.Parallel()
	d, _, err := ReadBytes([]byte(baseConfig))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	in := map[string]bool{"hook-a": true, "hook-b": false}
	d.SetHooks(in)
	got := string(d.Bytes())
	if !strings.HasPrefix(got, baseConfig) {
		t.Errorf("base content lost; got:\n%s", got)
	}
	if !strings.Contains(got, "hooks:") {
		t.Errorf("hooks: block missing; got:\n%s", got)
	}

	d2, _, err := ReadBytes(d.Bytes())
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	back, err := d2.Hooks()
	if err != nil {
		t.Fatalf("Hooks: %v", err)
	}
	if diff := cmp.Diff(in, back); diff != "" {
		t.Errorf("round-trip mismatch (-want +got):\n%s", diff)
	}
}

// TestSetHooks_AppendsAddsNewlineWhenSourceLacksTrailingNewline pins the
// branch appendHooks takes when the existing source's last byte is not
// already a newline (e.g. a hand-edited aiwf.yaml with no trailing
// newline) — the append must still produce a valid, separated block
// rather than concatenating onto the same line.
func TestSetHooks_AppendsAddsNewlineWhenSourceLacksTrailingNewline(t *testing.T) {
	t.Parallel()
	src := "aiwf_version: 0.1.0" // deliberately no trailing newline
	d, _, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	d.SetHooks(map[string]bool{"hook-a": true})
	got := string(d.Bytes())
	if !strings.HasPrefix(got, src+"\n") {
		t.Errorf("expected a newline inserted after the no-newline source; got:\n%q", got)
	}
	if !strings.Contains(got, "hooks:") {
		t.Errorf("hooks: block missing:\n%s", got)
	}
}

func TestSetHooks_PreservesOuterCommentsAndOrder(t *testing.T) {
	t.Parallel()
	src := `# Top-of-file comment
aiwf_version: 0.1.0
actor: human/peter # actor comment
hosts: [claude-code]

hooks:
  hook-a:
    enabled: false
`
	d, _, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	d.SetHooks(map[string]bool{"hook-a": true})
	got := string(d.Bytes())

	idx := strings.Index(src, "hooks:")
	if idx < 0 {
		t.Fatal("source has no hooks: token (test setup wrong)")
	}
	wantPrefix := src[:idx]
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("outer content changed.\nwant prefix:\n%q\ngot:\n%q", wantPrefix, got[:min(len(got), len(wantPrefix)+64)])
	}
	if !strings.Contains(got, "enabled: true") {
		t.Errorf("updated decision missing from output:\n%s", got)
	}
	if strings.Contains(got, "enabled: false") {
		t.Errorf("stale decision still present:\n%s", got)
	}
}

func TestSetHooks_ReplaceMidFile(t *testing.T) {
	t.Parallel()
	src := `aiwf_version: 0.1.0
actor: human/peter
hooks:
  hook-a:
    enabled: false
hosts:
  - claude-code
`
	d, _, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	d.SetHooks(map[string]bool{"hook-a": true})
	got := string(d.Bytes())

	if !strings.Contains(got, "hosts:\n  - claude-code\n") {
		t.Errorf("trailing hosts: block damaged:\n%s", got)
	}
	if !strings.Contains(got, "enabled: true") {
		t.Errorf("updated decision missing:\n%s", got)
	}
	if strings.Contains(got, "enabled: false") {
		t.Errorf("stale decision retained:\n%s", got)
	}
}

func TestSetHooks_RoundTripIsStable(t *testing.T) {
	t.Parallel()
	in := map[string]bool{"hook-a": true, "hook-b": false}
	d, _, err := ReadBytes([]byte(baseConfig))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	d.SetHooks(in)
	first := append([]byte(nil), d.Bytes()...)

	d2, _, err := ReadBytes(first)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	back, err := d2.Hooks()
	if err != nil {
		t.Fatalf("Hooks: %v", err)
	}
	d2.SetHooks(back)
	if !cmp.Equal(first, d2.Bytes()) {
		t.Errorf("second SetHooks not stable.\nfirst:\n%s\nsecond:\n%s", first, d2.Bytes())
	}
}

func TestSetHooks_EmptyDecisionsMap(t *testing.T) {
	t.Parallel()
	d, _, err := ReadBytes([]byte(baseConfig))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	d.SetHooks(map[string]bool{})
	d2, _, err := ReadBytes(d.Bytes())
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	got, err := d2.Hooks()
	if err != nil {
		t.Fatalf("Hooks: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Hooks() = %#v, want empty", got)
	}
}
