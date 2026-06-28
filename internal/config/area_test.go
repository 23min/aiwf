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
		if cfg.Areas.Members[i].Name != w {
			t.Errorf("Areas.Members[%d].Name = %q, want %q", i, cfg.Areas.Members[i].Name, w)
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
			wantSub: "neither a string",
		},
		{
			name:    "null member",
			block:   []string{"areas:", "  members:", "    - platform", "    - ~"},
			wantSub: "neither a string",
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

// TestLoad_AreasMemberNamedGlobalRejected pins M-0184/AC-5(a): a member
// declared with the reserved name `global` is rejected at config-load time
// with an error naming the reserved value — areas.members may not declare
// the cross-cutting sentinel (ADR-0021).
func TestLoad_AreasMemberNamedGlobalRejected(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	contents := []byte("areas:\n  members:\n    - platform\n    - global\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	if err == nil {
		t.Fatal("Load: want error for a member named the reserved `global`, got nil")
	}
	if !strings.Contains(err.Error(), "global") || !strings.Contains(err.Error(), "reserved") {
		t.Errorf("error = %q, want it to name the reserved %q value", err.Error(), "global")
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
	if len(cfg.Areas.Members) != 2 || cfg.Areas.Members[0].Name != "42" || cfg.Areas.Members[1].Name != "platform" {
		t.Errorf("Members = %v, want [42 platform]", cfg.Areas.Members)
	}
}

// TestConfig_CoverageRoots_ParsesAndValidates pins M-0185/AC-1: the
// areas.coverage_roots knob decodes as a string list, and validate() rejects an
// empty / whitespace-padded / non-repo-relative entry at config load (the
// Tier-1 gate). `.` is a valid root (the repo root as coverage scope); a
// leading slash or a `..` segment is rejected so the single-level enumeration
// the M-0185 check performs cannot escape the tree.
func TestConfig_CoverageRoots_ParsesAndValidates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		yaml      string
		wantErr   bool
		wantField string
		wantRoots []string
	}{
		{
			name:      "absent is nil",
			yaml:      "areas:\n  members:\n    - platform\n",
			wantRoots: nil,
		},
		{
			name:      "explicit empty list normalizes to nil",
			yaml:      "areas:\n  members:\n    - platform\n  coverage_roots: []\n",
			wantRoots: nil,
		},
		{
			name:      "valid list parses in order",
			yaml:      "areas:\n  members:\n    - platform\n  coverage_roots:\n    - projects\n    - services/backend\n",
			wantRoots: []string{"projects", "services/backend"},
		},
		{
			name:      "dot is a valid root",
			yaml:      "areas:\n  members:\n    - platform\n  coverage_roots:\n    - \".\"\n",
			wantRoots: []string{"."},
		},
		{
			name:      "empty entry rejected",
			yaml:      "areas:\n  members:\n    - platform\n  coverage_roots:\n    - \"\"\n",
			wantErr:   true,
			wantField: "empty",
		},
		{
			name:      "whitespace-padded entry rejected",
			yaml:      "areas:\n  members:\n    - platform\n  coverage_roots:\n    - \" projects \"\n",
			wantErr:   true,
			wantField: "whitespace",
		},
		{
			name:      "absolute path rejected",
			yaml:      "areas:\n  members:\n    - platform\n  coverage_roots:\n    - /abs\n",
			wantErr:   true,
			wantField: "repo-relative",
		},
		{
			name:      "dotdot segment rejected",
			yaml:      "areas:\n  members:\n    - platform\n  coverage_roots:\n    - ../escape\n",
			wantErr:   true,
			wantField: "repo-relative",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, FileName), []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(root)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (cfg=%+v)", cfg)
				}
				if !strings.Contains(err.Error(), tc.wantField) {
					t.Errorf("error %q does not name %q", err, tc.wantField)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			got := cfg.Areas.CoverageRoots
			if len(got) != len(tc.wantRoots) {
				t.Fatalf("CoverageRoots = %v, want %v", got, tc.wantRoots)
			}
			for i, w := range tc.wantRoots {
				if got[i] != w {
					t.Errorf("CoverageRoots[%d] = %q, want %q", i, got[i], w)
				}
			}
		})
	}
}

// TestConfig_AreasBlock_RejectsUnknownKey pins M-0185/AC-2: an unknown
// top-level key in the areas block (e.g. a `coverage_rootz:` typo) is a
// load-time error naming the bad key — the areas-block-level strict-key guard
// mirroring G-0287's member-level guard. Without it, yaml.v3's non-strict
// decode would silently drop the typo'd key and the operator's intent (here:
// `coverage_roots`) would vanish unflagged.
func TestConfig_AreasBlock_RejectsUnknownKey(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	contents := []byte("areas:\n  members:\n    - platform\n  coverage_rootz:\n    - projects\n")
	if err := os.WriteFile(filepath.Join(root, FileName), contents, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	if err == nil {
		t.Fatal("Load: want error for an unknown areas key, got nil")
	}
	if !strings.Contains(err.Error(), "coverage_rootz") || !strings.Contains(err.Error(), "unknown key") {
		t.Errorf("error = %q, want it to name the unknown key %q", err.Error(), "coverage_rootz")
	}
}
