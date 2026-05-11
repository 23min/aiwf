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

func TestLoad_TreeBlockRoundTrip(t *testing.T) {
	root := t.TempDir()
	contents := []byte(strings.Join([]string{
		"aiwf_version: 0.1.0",
		"tree:",
		"  strict: true",
		"  allow_paths:",
		"    - work/templates/*.md",
		"    - work/scratch/foo.md",
		"",
	}, "\n"))
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Tree.Strict {
		t.Error("Tree.Strict = false, want true")
	}
	want := []string{"work/templates/*.md", "work/scratch/foo.md"}
	if len(cfg.Tree.AllowPaths) != len(want) {
		t.Fatalf("AllowPaths = %v, want %v", cfg.Tree.AllowPaths, want)
	}
	for i, w := range want {
		if cfg.Tree.AllowPaths[i] != w {
			t.Errorf("AllowPaths[%d] = %q, want %q", i, cfg.Tree.AllowPaths[i], w)
		}
	}
}

func TestLoad_TreeBlockDefaults(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Tree.Strict {
		t.Error("Tree.Strict default should be false (warn-only)")
	}
	if len(cfg.Tree.AllowPaths) != 0 {
		t.Errorf("Tree.AllowPaths default should be empty, got %v", cfg.Tree.AllowPaths)
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
	// Post-G47 typical file: no `actor:` key (I2.5) and no
	// `aiwf_version:` key (G47). aiwf.yaml carries only policy.
	contents := []byte("hosts: [claude-code]\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LegacyAiwfVersion != "" {
		t.Errorf("LegacyAiwfVersion = %q, want empty (no aiwf_version: key in source)", cfg.LegacyAiwfVersion)
	}
	if cfg.LegacyActor != "" {
		t.Errorf("LegacyActor = %q, want empty (no actor: key in source)", cfg.LegacyActor)
	}
	if len(cfg.Hosts) != 1 || cfg.Hosts[0] != "claude-code" {
		t.Errorf("hosts = %v, want [claude-code]", cfg.Hosts)
	}
}

// TestLoad_LegacyAiwfVersionIsTolerated (G47): pre-G47 repos still
// carry `aiwf_version:` in their aiwf.yaml. Load must succeed and
// the value surfaces on Config.LegacyAiwfVersion so doctor can
// render its deprecation note and update can strip it.
func TestLoad_LegacyAiwfVersionIsTolerated(t *testing.T) {
	root := t.TempDir()
	contents := []byte("aiwf_version: 0.1.0\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LegacyAiwfVersion != "0.1.0" {
		t.Errorf("LegacyAiwfVersion = %q, want 0.1.0", cfg.LegacyAiwfVersion)
	}
}

func TestLoad_LegacyActorIsTolerated(t *testing.T) {
	// Backwards compat: pre-I2.5 repos still carry `actor:` in their
	// aiwf.yaml. Load must succeed (the field is ignored for runtime
	// identity resolution) and the value must surface on Config.LegacyActor
	// so `aiwf doctor` can render its deprecation note.
	root := t.TempDir()
	contents := []byte("hosts: [claude-code]\nactor: human/peter\n")
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
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("hosts: [claude-code]\nactor: human peter\n"), 0o644); err != nil {
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
	contents := []byte("hosts: [claude-code]\n")
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

// TestLoad_EmptyFileIsOK (G47): aiwf_version is no longer required.
// An aiwf.yaml with nothing (or only optional fields) loads fine.
func TestLoad_EmptyFileIsOK(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err != nil {
		t.Errorf("empty aiwf.yaml should load fine post-G47, got: %v", err)
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
	cfg := &Config{}
	if err := Write(root, cfg); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		t.Fatal(err)
	}
	// Post-G47 fresh write: no actor: (I2.5), no aiwf_version: (G47).
	// Both fields are derivable from elsewhere; storing them was dead weight.
	if strings.Contains(string(got), "actor:") {
		t.Errorf("actor: present in default-Write output (post-I2.5 must omit it): %q", got)
	}
	if strings.Contains(string(got), "aiwf_version:") {
		t.Errorf("aiwf_version: present in default-Write output (post-G47 must omit it): %q", got)
	}
}

func TestWrite_RefusesOverwrite(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("# pre-existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{}
	err := Write(root, cfg)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected refuse-overwrite, got %v", err)
	}
}

// TestStatusMdAutoUpdate_Default: no `status_md:` block in the file.
// Getter returns true (the framework's default-on opt-out semantics).
func TestStatusMdAutoUpdate_Default(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName),
		[]byte("hosts: [claude-code]\n"), 0o644); err != nil {
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
		[]byte("status_md: {}\n"), 0o644); err != nil {
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
		[]byte("status_md:\n  auto_update: false\n"), 0o644); err != nil {
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
		[]byte("status_md:\n  auto_update: true\n"), 0o644); err != nil {
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
	cfg := &Config{}
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

// TestStripLegacyAiwfVersion_RemovesField (G47): a pre-G47 aiwf.yaml
// carrying a top-level `aiwf_version:` key gets the line removed;
// surrounding content stays byte-identical.
func TestStripLegacyAiwfVersion_RemovesField(t *testing.T) {
	root := t.TempDir()
	contents := []byte("aiwf_version: 0.1.0\nhosts: [claude-code]\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := StripLegacyAiwfVersion(root)
	if err != nil {
		t.Fatalf("StripLegacyAiwfVersion: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	got, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		t.Fatal(err)
	}
	want := "hosts: [claude-code]\n"
	if string(got) != want {
		t.Errorf("file after strip:\n got  %q\n want %q", got, want)
	}
}

// TestStripLegacyAiwfVersion_NoFieldIsNoOp: a post-G47 file with
// no aiwf_version: stays byte-for-byte unchanged (idempotent on
// every `aiwf update`).
func TestStripLegacyAiwfVersion_NoFieldIsNoOp(t *testing.T) {
	root := t.TempDir()
	contents := []byte("# project config\nhosts: [claude-code]\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := StripLegacyAiwfVersion(root)
	if err != nil {
		t.Fatalf("StripLegacyAiwfVersion: %v", err)
	}
	if changed {
		t.Errorf("changed = true, want false (no aiwf_version: present)")
	}
	got, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, contents) {
		t.Errorf("file mutated despite no aiwf_version: line:\n got  %q\n want %q", got, contents)
	}
}

// TestStripLegacyAiwfVersion_MissingFile: brownfield-safe — strip
// runs on `aiwf update` even before init has scaffolded the yaml.
func TestStripLegacyAiwfVersion_MissingFile(t *testing.T) {
	root := t.TempDir()
	changed, err := StripLegacyAiwfVersion(root)
	if err != nil {
		t.Errorf("StripLegacyAiwfVersion on missing file: %v", err)
	}
	if changed {
		t.Errorf("changed = true on missing file, want false")
	}
}

// TestStripLegacyAiwfVersion_IgnoresIndentedField: an indented
// `aiwf_version:` line (a key inside some other mapping) is left
// alone — strip targets only the documented top-level legacy field.
func TestStripLegacyAiwfVersion_IgnoresIndentedField(t *testing.T) {
	root := t.TempDir()
	contents := []byte("hosts: [claude-code]\nfuture_block:\n  aiwf_version: 0.1.0\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := StripLegacyAiwfVersion(root)
	if err != nil {
		t.Fatalf("StripLegacyAiwfVersion: %v", err)
	}
	if changed {
		t.Errorf("changed = true, want false (only top-level aiwf_version: should match)")
	}
}

// TestLoad_DoctorRecommendedPlugins_AbsentIsEmpty: the field is
// optional. When `aiwf.yaml` carries no `doctor:` block at all,
// Config.Doctor.RecommendedPlugins is empty. M-070/AC-1 + AC-4 —
// kernel-neutral default.
func TestLoad_DoctorRecommendedPlugins_AbsentIsEmpty(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Doctor.RecommendedPlugins) != 0 {
		t.Errorf("Doctor.RecommendedPlugins = %v, want empty", cfg.Doctor.RecommendedPlugins)
	}
}

// TestLoad_DoctorRecommendedPlugins_ExplicitEmpty: `[]` is identical
// in effect to absence — empty slice, no checks fire downstream.
func TestLoad_DoctorRecommendedPlugins_ExplicitEmpty(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("doctor:\n  recommended_plugins: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Doctor.RecommendedPlugins) != 0 {
		t.Errorf("Doctor.RecommendedPlugins = %v, want empty", cfg.Doctor.RecommendedPlugins)
	}
}

// TestLoad_DoctorRecommendedPlugins_RoundTrip: a populated list
// loads in order with each entry preserved verbatim.
func TestLoad_DoctorRecommendedPlugins_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want []string
	}{
		{
			name: "single entry",
			yaml: "doctor:\n  recommended_plugins:\n    - aiwf-extensions@ai-workflow-rituals\n",
			want: []string{"aiwf-extensions@ai-workflow-rituals"},
		},
		{
			name: "multiple entries",
			yaml: "doctor:\n  recommended_plugins:\n    - aiwf-extensions@ai-workflow-rituals\n    - wf-rituals@ai-workflow-rituals\n",
			want: []string{"aiwf-extensions@ai-workflow-rituals", "wf-rituals@ai-workflow-rituals"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, FileName), []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(root)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if len(cfg.Doctor.RecommendedPlugins) != len(tc.want) {
				t.Fatalf("Doctor.RecommendedPlugins = %v, want %v", cfg.Doctor.RecommendedPlugins, tc.want)
			}
			for i, w := range tc.want {
				if cfg.Doctor.RecommendedPlugins[i] != w {
					t.Errorf("[%d] = %q, want %q", i, cfg.Doctor.RecommendedPlugins[i], w)
				}
			}
		})
	}
}

// TestLoad_DoctorBlock_FallsThrough_WhenRecommendedPluginsAbsent:
// the `doctor:` block exists but has no `recommended_plugins` field
// (or has it explicitly null). Pre-check returns nil; typed unmarshal
// proceeds; cfg.Doctor.RecommendedPlugins is empty. Covers the
// `!present || raw == nil` branch of preCheckTypedShape that the
// fully-absent fixtures don't exercise (they take the !ok branch
// one step earlier).
func TestLoad_DoctorBlock_FallsThrough_WhenRecommendedPluginsAbsent(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		// doctor block with a different, currently-unknown field.
		// yaml.Unmarshal ignores unknown fields by default, so the
		// load succeeds; the pre-check's `!present` branch fires.
		{name: "no recommended_plugins key", yaml: "doctor:\n  some_other_future_field: ignored\n"},
		// recommended_plugins key but value omitted: parses as nil;
		// the pre-check's `raw == nil` branch fires.
		{name: "recommended_plugins null", yaml: "doctor:\n  recommended_plugins:\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, FileName), []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(root)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if len(cfg.Doctor.RecommendedPlugins) != 0 {
				t.Errorf("Doctor.RecommendedPlugins = %v, want empty", cfg.Doctor.RecommendedPlugins)
			}
		})
	}
}

// TestLoad_DoctorRecommendedPlugins_RejectsMalformed: every malformed
// shape rejects the load with a clear error pointing at the field
// path. Covers entry-without-@, non-string entry, and non-list value.
// Each error message must name `doctor.recommended_plugins` so a
// reader can locate the offending key without grepping the source.
func TestLoad_DoctorRecommendedPlugins_RejectsMalformed(t *testing.T) {
	cases := []struct {
		name        string
		yaml        string
		errContains string
	}{
		{
			name:        "entry missing @",
			yaml:        "doctor:\n  recommended_plugins:\n    - just-a-name\n",
			errContains: "doctor.recommended_plugins",
		},
		{
			name:        "entry empty marketplace",
			yaml:        "doctor:\n  recommended_plugins:\n    - aiwf-extensions@\n",
			errContains: "doctor.recommended_plugins",
		},
		{
			name: "entry empty name",
			// `@`-prefixed value must be YAML-quoted to parse. The
			// validation below should catch the empty-name shape.
			yaml:        "doctor:\n  recommended_plugins:\n    - \"@ai-workflow-rituals\"\n",
			errContains: "doctor.recommended_plugins",
		},
		{
			name:        "non-list value",
			yaml:        "doctor:\n  recommended_plugins: aiwf-extensions@ai-workflow-rituals\n",
			errContains: "doctor.recommended_plugins",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, FileName), []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(root)
			if err == nil {
				t.Fatalf("Load: nil error, want failure naming %q", tc.errContains)
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("Load error = %q, want to contain %q", err.Error(), tc.errContains)
			}
		})
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

// M-0088 AC-1 — archive.sweep_threshold schema.
//
// The knob is the consumer-controlled drift escalation per ADR-0004
// §"Drift control" layer (2). The default is **unset** (permissive);
// when set, exceeding the count flips the `archive-sweep-pending`
// finding to blocking. A tristate `*int` distinguishes "no threshold"
// (nil) from "threshold of 0" (every pending sweep blocks). Mirrors
// the `StatusMd.AutoUpdate *bool` precedent.

// TestArchiveSweepThreshold_DefaultUnset: no `archive:` block in the
// file. The Archive struct field is zero-value; the threshold pointer
// is nil; ArchiveSweepThreshold() returns 0, false (the "unset"
// signal a check rule reads to skip the escalation).
func TestArchiveSweepThreshold_DefaultUnset(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName),
		[]byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Archive.SweepThreshold != nil {
		t.Errorf("Archive.SweepThreshold = %v, want nil (absent)", *cfg.Archive.SweepThreshold)
	}
	if n, set := cfg.ArchiveSweepThreshold(); set || n != 0 {
		t.Errorf("ArchiveSweepThreshold() = (%d, %v), want (0, false)", n, set)
	}
}

// TestArchiveSweepThreshold_ExplicitZero: `archive.sweep_threshold: 0`
// is a legitimate value — the consumer wants every pending sweep to
// block. Distinguished from "unset" via the *int tristate and the
// "set" return of the getter.
func TestArchiveSweepThreshold_ExplicitZero(t *testing.T) {
	root := t.TempDir()
	contents := []byte("archive:\n  sweep_threshold: 0\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Archive.SweepThreshold == nil {
		t.Fatal("Archive.SweepThreshold = nil, want &0")
	}
	if *cfg.Archive.SweepThreshold != 0 {
		t.Errorf("*Archive.SweepThreshold = %d, want 0", *cfg.Archive.SweepThreshold)
	}
	n, set := cfg.ArchiveSweepThreshold()
	if !set {
		t.Error("ArchiveSweepThreshold() set = false, want true")
	}
	if n != 0 {
		t.Errorf("ArchiveSweepThreshold() n = %d, want 0", n)
	}
}

// TestArchiveSweepThreshold_ExplicitPositive: the load-bearing
// consumer-tuning case. `archive.sweep_threshold: 5` is set; the
// getter returns (5, true); a check rule reading the value will
// escalate the aggregate finding when count > 5.
func TestArchiveSweepThreshold_ExplicitPositive(t *testing.T) {
	root := t.TempDir()
	contents := []byte("archive:\n  sweep_threshold: 5\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Archive.SweepThreshold == nil {
		t.Fatal("Archive.SweepThreshold = nil, want &5")
	}
	if *cfg.Archive.SweepThreshold != 5 {
		t.Errorf("*Archive.SweepThreshold = %d, want 5", *cfg.Archive.SweepThreshold)
	}
	n, set := cfg.ArchiveSweepThreshold()
	if !set {
		t.Error("ArchiveSweepThreshold() set = false, want true")
	}
	if n != 5 {
		t.Errorf("ArchiveSweepThreshold() n = %d, want 5", n)
	}
}

// TestArchiveSweepThreshold_NilReceiver: getter on a nil Config
// returns the "unset" signal. Mirrors AllocateTrunkRef's
// nil-tolerance — callers in `cmd/aiwf/main.go::runCheckCmd` may
// reach the getter before cfg is loaded (or when Load returned
// ErrNotFound), and the getter must not panic.
func TestArchiveSweepThreshold_NilReceiver(t *testing.T) {
	var cfg *Config
	n, set := cfg.ArchiveSweepThreshold()
	if set || n != 0 {
		t.Errorf("nil-receiver ArchiveSweepThreshold() = (%d, %v), want (0, false)", n, set)
	}
}

// TestArchiveSweepThreshold_BlockEmpty: `archive:` block present but
// carries no `sweep_threshold:`. Mirrors TestStatusMdAutoUpdate_BlockEmpty:
// the block-empty case must still resolve to the default.
func TestArchiveSweepThreshold_BlockEmpty(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName),
		[]byte("archive: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Archive.SweepThreshold != nil {
		t.Errorf("Archive.SweepThreshold = %v, want nil (block-empty)", *cfg.Archive.SweepThreshold)
	}
	if n, set := cfg.ArchiveSweepThreshold(); set || n != 0 {
		t.Errorf("block-empty ArchiveSweepThreshold() = (%d, %v), want (0, false)", n, set)
	}
}

// TestArchive_BlockRoundTrip: parse → marshal → parse holds the
// `archive.sweep_threshold` value through the full Config life-
// cycle. Mirrors the existing TestLoad_TreeBlockRoundTrip pattern.
func TestArchive_BlockRoundTrip(t *testing.T) {
	root1 := t.TempDir()
	contents := []byte("archive:\n  sweep_threshold: 12\n")
	if err := os.WriteFile(filepath.Join(root1, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root1)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Archive.SweepThreshold == nil || *cfg.Archive.SweepThreshold != 12 {
		t.Fatalf("parse: SweepThreshold = %v, want &12", cfg.Archive.SweepThreshold)
	}

	// Marshal → Write to a fresh dir → Load again. The second Load
	// is the round-trip pin.
	root2 := t.TempDir()
	if wErr := Write(root2, cfg); wErr != nil {
		t.Fatalf("Write: %v", wErr)
	}
	written, err := os.ReadFile(filepath.Join(root2, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(written, []byte("sweep_threshold")) {
		t.Errorf("Write output missing sweep_threshold:\n%s", written)
	}
	cfg2, err := Load(root2)
	if err != nil {
		t.Fatalf("Load (round-trip): %v", err)
	}
	if cfg2.Archive.SweepThreshold == nil || *cfg2.Archive.SweepThreshold != 12 {
		t.Errorf("round-trip: SweepThreshold = %v, want &12", cfg2.Archive.SweepThreshold)
	}
}

// TestWrite_OmitsArchiveByDefault: a default Config must not emit
// an `archive:` block on Write — mirrors the StatusMd default-shape
// guarantee. Otherwise `aiwf init` would surprise the operator
// with a knob they didn't set.
func TestWrite_OmitsArchiveByDefault(t *testing.T) {
	root := t.TempDir()
	cfg := &Config{}
	if err := Write(root, cfg); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "archive") {
		t.Errorf("archive present in default-Write output: %q", got)
	}
}
