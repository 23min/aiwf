package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoad_AreasBlockRoundTrip pins AC-2 of M-0171: aiwf.yaml accepts an
// `areas` block (closed member set + optional default display label) and Load
// surfaces it on the typed Config.
func TestLoad_AreasBlockRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	contents := []byte(strings.Join([]string{
		"areas:",
		"  members:",
		"    - platform",
		"    - tooling",
		"  default: Uncategorized",
		"",
	}, "\n"))
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{"platform", "tooling"}
	if len(cfg.Areas.Members) != len(want) {
		t.Fatalf("Areas.Members = %v, want %v", cfg.Areas.Members, want)
	}
	for i, w := range want {
		if cfg.Areas.Members[i] != w {
			t.Errorf("Areas.Members[%d] = %q, want %q", i, cfg.Areas.Members[i], w)
		}
	}
	if cfg.Areas.Default != "Uncategorized" {
		t.Errorf("Areas.Default = %q, want %q", cfg.Areas.Default, "Uncategorized")
	}
}

// TestLoad_AreasBlockAbsent pins that an aiwf.yaml with no areas block loads
// clean with an empty Areas (the field is inert — AC-4).
func TestLoad_AreasBlockAbsent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Areas.Members) != 0 {
		t.Errorf("Areas.Members = %v, want empty", cfg.Areas.Members)
	}
	if cfg.Areas.Default != "" {
		t.Errorf("Areas.Default = %q, want empty", cfg.Areas.Default)
	}
}

// TestLoad_AreasBlockMalformed pins AC-2's validation: a malformed areas block
// is rejected at config-load time with a clear error. One case per validation
// branch: empty member, duplicate member, default-equal-to-member.
func TestLoad_AreasBlockMalformed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		block   []string
		wantSub string
	}{
		{
			name:    "empty member",
			block:   []string{"areas:", "  members:", "    - platform", "    - \"\""},
			wantSub: "empty member",
		},
		{
			name:    "duplicate member",
			block:   []string{"areas:", "  members:", "    - platform", "    - platform"},
			wantSub: "duplicate member",
		},
		{
			name:    "default is a member",
			block:   []string{"areas:", "  members:", "    - platform", "  default: platform"},
			wantSub: "must not also be a member",
		},
		{
			name:    "non-string member",
			block:   []string{"areas:", "  members:", "    - platform", "    - 42"},
			wantSub: "not a string",
		},
		{
			name:    "null member",
			block:   []string{"areas:", "  members:", "    - platform", "    - ~"},
			wantSub: "not a string",
		},
		{
			name:    "members not a sequence",
			block:   []string{"areas:", "  members: oops"},
			wantSub: "", // any decode error; exact yaml text is unstable
		},
		{
			name:    "whitespace-padded member",
			block:   []string{"areas:", "  members:", "    - platform", "    - \"  tooling  \""},
			wantSub: "whitespace",
		},
		{
			name:    "whitespace-only default",
			block:   []string{"areas:", "  members:", "    - platform", "  default: \"   \""},
			wantSub: "whitespace-only",
		},
		{
			name:    "whitespace-padded default",
			block:   []string{"areas:", "  members:", "    - platform", "  default: \" platform \""},
			wantSub: "whitespace",
		},
		{
			name:    "default without members",
			block:   []string{"areas:", "  default: Uncategorized"},
			wantSub: "no members",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			contents := []byte(strings.Join(append(tc.block, ""), "\n"))
			if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(root); err == nil {
				t.Fatal("Load: want error for malformed areas block, got nil")
			} else if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantSub)
			}
		})
	}
}

// TestLoad_AreasBlockQuotedNumericMemberAccepted pins that the non-string
// rejection (AC-2) keys on the YAML node type, not on appearance: a quoted
// numeric is a string member and is accepted, while only an unquoted non-string
// scalar (yaml !!int/!!bool/!!null) is rejected.
func TestLoad_AreasBlockQuotedNumericMemberAccepted(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	contents := []byte("areas:\n  members:\n    - \"42\"\n    - platform\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Areas.Members) != 2 || cfg.Areas.Members[0] != "42" || cfg.Areas.Members[1] != "platform" {
		t.Errorf("Members = %v, want [42 platform]", cfg.Areas.Members)
	}
}
