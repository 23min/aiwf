package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_Missing_ReturnsErrNotFound(t *testing.T) {
	root := t.TempDir()
	_, err := Load(root)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestAllocateTrunkRef_DefaultWhenUnset(t *testing.T) {
	cases := []struct {
		name string
		cfg  *Config
	}{
		{"nil receiver", nil},
		{"empty allocate block", &Config{}},
		{"explicit empty trunk", &Config{Allocate: Allocate{Trunk: ""}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ref, explicit := tc.cfg.AllocateTrunkRef()
			if ref != DefaultAllocateTrunk {
				t.Errorf("ref = %q, want %q", ref, DefaultAllocateTrunk)
			}
			if explicit {
				t.Error("explicit = true, want false (unset)")
			}
		})
	}
}

func TestAllocateTrunkRef_ExplicitlyConfigured(t *testing.T) {
	cfg := &Config{Allocate: Allocate{Trunk: "refs/remotes/upstream/master"}}
	ref, explicit := cfg.AllocateTrunkRef()
	if ref != "refs/remotes/upstream/master" {
		t.Errorf("ref = %q, want %q", ref, "refs/remotes/upstream/master")
	}
	if !explicit {
		t.Error("explicit = false, want true (set in config)")
	}
}

func TestLoad_AllocateTrunkRoundTrip(t *testing.T) {
	root := t.TempDir()
	contents := []byte("aiwf_version: 0.1.0\nallocate:\n  trunk: refs/remotes/origin/develop\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ref, explicit := cfg.AllocateTrunkRef()
	if ref != "refs/remotes/origin/develop" {
		t.Errorf("ref = %q, want %q", ref, "refs/remotes/origin/develop")
	}
	if !explicit {
		t.Error("explicit = false, want true (parsed from yaml)")
	}
}

func TestLoad_TypicalFile(t *testing.T) {
	root := t.TempDir()
	// Post-I2.5 typical file: no `actor:` key. Identity is runtime-
	// derived; aiwf.yaml carries only policy.
	contents := []byte("aiwf_version: 0.1.0\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AiwfVersion != "0.1.0" {
		t.Errorf("aiwf_version = %q, want 0.1.0", cfg.AiwfVersion)
	}
	if cfg.LegacyActor != "" {
		t.Errorf("LegacyActor = %q, want empty (no actor: key in source)", cfg.LegacyActor)
	}
	if len(cfg.Hosts) != 0 {
		t.Errorf("hosts should be empty, got %v", cfg.Hosts)
	}
}

func TestLoad_LegacyActorIsTolerated(t *testing.T) {
	// Backwards compat: pre-I2.5 repos still carry `actor:` in their
	// aiwf.yaml. Load must succeed (the field is ignored for runtime
	// identity resolution) and the value must surface on Config.LegacyActor
	// so `aiwf doctor` can render its deprecation note.
	root := t.TempDir()
	contents := []byte("aiwf_version: 0.1.0\nactor: human/peter\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LegacyActor != "human/peter" {
		t.Errorf("LegacyActor = %q, want human/peter", cfg.LegacyActor)
	}
}

func TestLoad_LegacyMalformedActorIsHarmless(t *testing.T) {
	// A malformed legacy `actor:` is no longer a parse error — the
	// field is ignored for runtime resolution. This keeps repos that
	// were previously misconfigured loadable so the user can run
	// `aiwf doctor` and remove the field.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("aiwf_version: 0.1.0\nactor: human peter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v (legacy actor should be tolerated)", err)
	}
	if cfg.LegacyActor != "human peter" {
		t.Errorf("LegacyActor = %q, want raw legacy value", cfg.LegacyActor)
	}
}

func TestLoad_WithHosts(t *testing.T) {
	root := t.TempDir()
	contents := []byte("aiwf_version: 0.1.0\nhosts: [claude-code]\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Hosts) != 1 || cfg.Hosts[0] != "claude-code" {
		t.Errorf("got %v", cfg.Hosts)
	}
}

func TestLoad_MissingVersion(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "aiwf_version") {
		t.Errorf("expected aiwf_version-required error, got %v", err)
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(":::not yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse error, got %v", err)
	}
}

func TestWrite_FreshDir(t *testing.T) {
	root := t.TempDir()
	cfg := &Config{AiwfVersion: "0.1.0"}
	if err := Write(root, cfg); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "aiwf_version: 0.1.0") {
		t.Errorf("aiwf_version missing in output: %q", got)
	}
	// Identity is no longer stored — `aiwf init` must never emit an
	// `actor:` line on a fresh write.
	if strings.Contains(string(got), "actor:") {
		t.Errorf("actor: present in default-Write output (post-I2.5 must omit it): %q", got)
	}
}

func TestWrite_RefusesOverwrite(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("# pre-existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{AiwfVersion: "0.1.0"}
	err := Write(root, cfg)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected refuse-overwrite, got %v", err)
	}
}

func TestWrite_RejectsInvalidConfig(t *testing.T) {
	// Post-I2.5 the only required field is aiwf_version. Empty-version
	// must reject; empty-everything-else must succeed.
	root := t.TempDir()
	if err := Write(root, &Config{AiwfVersion: ""}); err == nil {
		t.Error("expected validation error on empty aiwf_version, got nil")
	}
}

// TestStatusMdAutoUpdate_Default: no `status_md:` block in the file.
// Getter returns true (the framework's default-on opt-out semantics).
func TestStatusMdAutoUpdate_Default(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName),
		[]byte("aiwf_version: 0.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.StatusMdAutoUpdate() {
		t.Errorf("default: StatusMdAutoUpdate() = false, want true")
	}
	if cfg.StatusMd.AutoUpdate != nil {
		t.Errorf("StatusMd.AutoUpdate = %v, want nil (absent)", *cfg.StatusMd.AutoUpdate)
	}
}

// TestStatusMdAutoUpdate_BlockEmpty: `status_md:` is present but
// carries no fields. The getter still falls back to the default.
func TestStatusMdAutoUpdate_BlockEmpty(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName),
		[]byte("aiwf_version: 0.1.0\nstatus_md: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.StatusMdAutoUpdate() {
		t.Errorf("block-empty: StatusMdAutoUpdate() = false, want true")
	}
}

// TestStatusMdAutoUpdate_ExplicitFalse: the load-bearing opt-out
// case. The getter returns false; the round-trip preserves the
// explicit setting.
func TestStatusMdAutoUpdate_ExplicitFalse(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName),
		[]byte("aiwf_version: 0.1.0\nstatus_md:\n  auto_update: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.StatusMdAutoUpdate() {
		t.Errorf("explicit-false: StatusMdAutoUpdate() = true, want false")
	}
	if cfg.StatusMd.AutoUpdate == nil || *cfg.StatusMd.AutoUpdate {
		t.Errorf("StatusMd.AutoUpdate = %v, want &false", cfg.StatusMd.AutoUpdate)
	}
}

// TestStatusMdAutoUpdate_ExplicitTrue: an explicit `auto_update: true`
// opts in (matches the default but is preserved on round-trip so the
// user's intent isn't dropped).
func TestStatusMdAutoUpdate_ExplicitTrue(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName),
		[]byte("aiwf_version: 0.1.0\nstatus_md:\n  auto_update: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.StatusMdAutoUpdate() {
		t.Errorf("explicit-true: StatusMdAutoUpdate() = false, want true")
	}
	if cfg.StatusMd.AutoUpdate == nil || !*cfg.StatusMd.AutoUpdate {
		t.Errorf("StatusMd.AutoUpdate = %v, want &true", cfg.StatusMd.AutoUpdate)
	}
}

// TestWrite_OmitsStatusMdByDefault: a Config with no explicit
// status_md setting must not emit a `status_md:` block — preserving
// the file-shape guarantee that "default behavior" is also "default
// file shape" (no surprise YAML on `aiwf init`).
func TestWrite_OmitsStatusMdByDefault(t *testing.T) {
	root := t.TempDir()
	cfg := &Config{AiwfVersion: "0.1.0"}
	if err := Write(root, cfg); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "status_md") {
		t.Errorf("status_md present in default-Write output: %q", got)
	}
}

// TestStripLegacyActor_RemovesField: a pre-I2.5 aiwf.yaml carrying
// a top-level `actor:` key gets the line removed; aiwf_version and
// surrounding content stay byte-identical (only that line drops).
func TestStripLegacyActor_RemovesField(t *testing.T) {
	root := t.TempDir()
	contents := []byte("aiwf_version: 0.1.0\nactor: human/peter\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := StripLegacyActor(root)
	if err != nil {
		t.Fatalf("StripLegacyActor: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	got, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		t.Fatal(err)
	}
	want := "aiwf_version: 0.1.0\n"
	if string(got) != want {
		t.Errorf("file after strip:\n got  %q\n want %q", got, want)
	}
}

// TestStripLegacyActor_NoFieldIsNoOp: a typical post-I2.5 file
// with no actor line stays byte-for-byte unchanged. Comments
// survive — that's the whole reason we line-strip rather than
// YAML round-trip.
func TestStripLegacyActor_NoFieldIsNoOp(t *testing.T) {
	root := t.TempDir()
	contents := []byte("# project config\naiwf_version: 0.1.0\nhosts: [claude-code]\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := StripLegacyActor(root)
	if err != nil {
		t.Fatalf("StripLegacyActor: %v", err)
	}
	if changed {
		t.Errorf("changed = true, want false (no actor: present)")
	}
	got, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, contents) {
		t.Errorf("file mutated despite no actor: line:\n got  %q\n want %q", got, contents)
	}
}

// TestStripLegacyActor_PreservesComments: a comment block above
// the actor line is retained; the strip only drops the actor
// line itself.
func TestStripLegacyActor_PreservesComments(t *testing.T) {
	root := t.TempDir()
	contents := []byte("# the project's config\naiwf_version: 0.1.0\n# legacy identity\nactor: human/peter\nhosts: [claude-code]\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := StripLegacyActor(root)
	if err != nil {
		t.Fatalf("StripLegacyActor: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	got, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		t.Fatal(err)
	}
	want := "# the project's config\naiwf_version: 0.1.0\n# legacy identity\nhosts: [claude-code]\n"
	if string(got) != want {
		t.Errorf("file after strip:\n got  %q\n want %q", got, want)
	}
}

// TestStripLegacyActor_MissingFile: when aiwf.yaml is absent the
// strip is a no-op (changed=false, no error). Lets `aiwf update`
// run on a brownfield branch with no aiwf.yaml at the root.
func TestStripLegacyActor_MissingFile(t *testing.T) {
	root := t.TempDir()
	changed, err := StripLegacyActor(root)
	if err != nil {
		t.Errorf("StripLegacyActor on missing file: %v", err)
	}
	if changed {
		t.Errorf("changed = true on missing file, want false")
	}
}

// TestStripLegacyActor_IgnoresIndentedActor: an indented `actor:`
// line (i.e., a key inside some other mapping) is left alone. The
// strip targets only the documented top-level legacy field.
func TestStripLegacyActor_IgnoresIndentedActor(t *testing.T) {
	root := t.TempDir()
	contents := []byte("aiwf_version: 0.1.0\nfuture_block:\n  actor: human/peter\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := StripLegacyActor(root)
	if err != nil {
		t.Fatalf("StripLegacyActor: %v", err)
	}
	if changed {
		t.Errorf("changed = true, want false (only top-level actor: should match)")
	}
	got, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, contents) {
		t.Errorf("indented actor: line was clobbered:\n got  %q\n want %q", got, contents)
	}
}

func TestActorPattern(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"human/peter", true},
		{"claude/opus-4.7", true},
		{"foo/bar/baz", false},
		{"human:peter", false},
		{"human / peter", false},
		{"/peter", false},
		{"peter/", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := ActorPattern.MatchString(tt.s); got != tt.want {
			t.Errorf("%q: got %v, want %v", tt.s, got, tt.want)
		}
	}
}
