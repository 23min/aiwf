package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActorPattern(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"human/peter", true},
		{"claude/opus-4.7", true},
		{"gpt/4o", true},
		{"foo/bar/baz", false},   // multiple slashes -> two slashes; \S+/\S+ matches "foo/bar" + "/baz" not whole-string anchor
		{"human:peter", false},   // no slash
		{"human / peter", false}, // whitespace
		{"/peter", false},        // empty role
		{"peter/", false},        // empty identifier
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := actorPattern.MatchString(tt.s)
			if got != tt.want {
				t.Errorf("actorPattern.Match(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestResolveActor_ExplicitValid(t *testing.T) {
	got, err := resolveActor("human/peter", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "human/peter" {
		t.Errorf("got %q", got)
	}
}

func TestResolveActor_ExplicitInvalid(t *testing.T) {
	for _, bad := range []string{"human:peter", "human / peter", "no-slash", ""} {
		t.Run(bad, func(t *testing.T) {
			_, err := resolveActor(bad, "")
			if bad == "" {
				// Empty falls through to git-config derivation; not an "invalid" path.
				return
			}
			if err == nil || !strings.Contains(err.Error(), "must match") {
				t.Errorf("expected format error for %q, got %v", bad, err)
			}
		})
	}
}

// TestResolveActor_FromConfig verifies that an aiwf.yaml's actor wins
// over git-config derivation when no --actor flag is passed.
func TestResolveActor_FromConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("aiwf_version: 0.1.0\nactor: human/from-config\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveActor("", root)
	if err != nil {
		t.Fatalf("resolveActor: %v", err)
	}
	if got != "human/from-config" {
		t.Errorf("got %q, want human/from-config", got)
	}
}

// TestResolveActor_ConfigMalformed_Errors propagates a parse error so
// the user is not silently dropped to a git-config-derived fallback
// after writing a broken aiwf.yaml.
func TestResolveActor_ConfigMalformed_Errors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(":::not yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := resolveActor("", root)
	if err == nil {
		t.Fatal("expected error from malformed aiwf.yaml")
	}
}

// TestResolveActor_DerivedFromGitConfig sets up an isolated git env
// (HOME pointing at a tmpdir with a .gitconfig containing user.email)
// and verifies that resolveActor("") derives `human/<localpart>`.
func TestResolveActor_DerivedFromGitConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	gitconfig := filepath.Join(home, ".gitconfig")
	if err := os.WriteFile(gitconfig, []byte("[user]\n\temail = peter@example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveActor("", "")
	if err != nil {
		t.Fatalf("resolveActor: %v", err)
	}
	if got != "human/peter" {
		t.Errorf("got %q, want human/peter", got)
	}
}

// TestResolveActor_NoConfigErrors verifies the no-info path: explicit
// is empty and git config user.email is unset.
func TestResolveActor_NoConfigErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// Intentionally no .gitconfig.

	_, err := resolveActor("", "")
	if err == nil {
		t.Fatal("expected error when neither --actor nor git config is set")
	}
	if !strings.Contains(err.Error(), "no actor") {
		t.Errorf("error %q should mention 'no actor'", err.Error())
	}
}
