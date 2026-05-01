package config

import (
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

func TestLoad_TypicalFile(t *testing.T) {
	root := t.TempDir()
	contents := []byte("aiwf_version: 0.1.0\nactor: human/peter\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AiwfVersion != "0.1.0" || cfg.Actor != "human/peter" {
		t.Errorf("got %+v", cfg)
	}
	if len(cfg.Hosts) != 0 {
		t.Errorf("hosts should be empty, got %v", cfg.Hosts)
	}
}

func TestLoad_WithHosts(t *testing.T) {
	root := t.TempDir()
	contents := []byte("aiwf_version: 0.1.0\nactor: human/peter\nhosts: [claude-code]\n")
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

func TestLoad_InvalidActor(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("aiwf_version: 0.1.0\nactor: human peter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "actor") {
		t.Errorf("expected actor format error, got %v", err)
	}
}

func TestLoad_MissingActor(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("aiwf_version: 0.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "actor") {
		t.Errorf("expected actor-required error, got %v", err)
	}
}

func TestLoad_MissingVersion(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("actor: human/peter\n"), 0o644); err != nil {
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
	cfg := &Config{AiwfVersion: "0.1.0", Actor: "human/peter"}
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
	if !strings.Contains(string(got), "actor: human/peter") {
		t.Errorf("actor missing in output: %q", got)
	}
}

func TestWrite_RefusesOverwrite(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("# pre-existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{AiwfVersion: "0.1.0", Actor: "human/peter"}
	err := Write(root, cfg)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected refuse-overwrite, got %v", err)
	}
}

func TestWrite_RejectsInvalidConfig(t *testing.T) {
	root := t.TempDir()
	if err := Write(root, &Config{AiwfVersion: "0.1.0", Actor: "broken format"}); err == nil {
		t.Error("expected validation error, got nil")
	}
}

// TestStatusMdAutoUpdate_Default: no `status_md:` block in the file.
// Getter returns true (the framework's default-on opt-out semantics).
func TestStatusMdAutoUpdate_Default(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName),
		[]byte("aiwf_version: 0.1.0\nactor: human/peter\n"), 0o644); err != nil {
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
		[]byte("aiwf_version: 0.1.0\nactor: human/peter\nstatus_md: {}\n"), 0o644); err != nil {
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
		[]byte("aiwf_version: 0.1.0\nactor: human/peter\nstatus_md:\n  auto_update: false\n"), 0o644); err != nil {
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
		[]byte("aiwf_version: 0.1.0\nactor: human/peter\nstatus_md:\n  auto_update: true\n"), 0o644); err != nil {
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
	cfg := &Config{AiwfVersion: "0.1.0", Actor: "human/peter"}
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
