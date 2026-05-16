package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActorPattern(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			got := actorPattern.MatchString(tt.s)
			if got != tt.want {
				t.Errorf("actorPattern.Match(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestResolveActor_ExplicitValid(t *testing.T) {
	t.Parallel()
	got, err := resolveActor("human/peter", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "human/peter" {
		t.Errorf("got %q", got)
	}
}

func TestResolveActor_ExplicitInvalid(t *testing.T) {
	t.Parallel()
	for _, bad := range []string{"human:peter", "human / peter", "no-slash", ""} {
		t.Run(bad, func(t *testing.T) {
			t.Parallel()
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

// TestResolveActor_LegacyConfigActorIgnored: pre-I2.5 repos with an
// `actor:` key in aiwf.yaml must NOT be consulted for runtime
// resolution. Identity is now flag-or-git-config only. The legacy
// field surfaces only via aiwf doctor's deprecation note.
func TestResolveActor_LegacyConfigActorIgnored(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("aiwf_version: 0.1.0\nactor: human/from-config\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Isolate git env so the host's identity doesn't bleed in.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"),
		[]byte("[user]\n\temail = git-user@example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveActor("", root)
	if err != nil {
		t.Fatalf("resolveActor: %v", err)
	}
	// Must come from git, not from the legacy aiwf.yaml.actor.
	if got != "human/git-user" {
		t.Errorf("got %q, want human/git-user (legacy aiwf.yaml.actor must be ignored)", got)
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

// TestResolveActor_FlagOverridesGitConfig: --actor wins over git
// config user.email. This is the load-bearing precedence rule;
// without it, an LLM harness can never act as `ai/claude` from a
// developer machine where git is configured for the human.
func TestResolveActor_FlagOverridesGitConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"),
		[]byte("[user]\n\temail = peter@example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveActor("ai/claude", "")
	if err != nil {
		t.Fatalf("resolveActor: %v", err)
	}
	if got != "ai/claude" {
		t.Errorf("got %q, want ai/claude (--actor must override git config)", got)
	}
}

// TestResolveActor_MalformedGitEmail: git config user.email is set
// but has no @ separator (a degenerate but technically allowed git
// state). Falls through to the no-actor error rather than producing
// a malformed `human/<entire-string>` identity.
func TestResolveActor_MalformedGitEmail(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"),
		[]byte("[user]\n\temail = no-at-sign\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := resolveActor("", "")
	if err == nil {
		t.Fatal("expected error from malformed git config user.email, got nil")
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
