package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// loadAreas writes an aiwf.yaml carrying the given areas-block body lines
// (each already indented under `areas:`) to a temp dir and returns the loaded
// Config. It exercises the real Load → UnmarshalYAML → Validate path.
func loadAreas(t *testing.T, body string) (*Config, error) {
	t.Helper()
	root := t.TempDir()
	contents := "areas:\n" + body
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(contents), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	return Load(root)
}

// TestAreas_UnmarshalDualForm pins AC-1 (M-0179): areas.members decodes each
// member shape into the right Member, in declaration order. The form
// enumeration is spec-sourced from the M-0179 AC plan §AC-1 (string-only,
// object-with-paths, object-no-paths, object-explicit-empty, mixed) — one row
// per declared form. The full []Member slice (name, paths, order) is asserted
// with go-cmp; the explicit-empty and absent-paths rows both assert Paths is
// the canonical nil, not a distinct empty slice.
func TestAreas_UnmarshalDualForm(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want []Member
	}{
		{
			name: "string-only",
			body: "  members:\n    - app-a\n    - billing\n",
			want: []Member{{Name: "app-a"}, {Name: "billing"}},
		},
		{
			name: "object-with-paths",
			body: "  members:\n    - name: app-a\n      paths:\n        - projects/app-a/**\n",
			want: []Member{{Name: "app-a", Paths: []string{"projects/app-a/**"}}},
		},
		{
			name: "object-no-paths",
			body: "  members:\n    - name: app-a\n",
			want: []Member{{Name: "app-a"}},
		},
		{
			name: "object-explicit-empty",
			body: "  members:\n    - name: app-a\n      paths: []\n",
			want: []Member{{Name: "app-a"}}, // explicit [] normalizes to nil, == absent
		},
		{
			name: "mixed",
			body: "  members:\n    - app-a\n    - name: billing\n      paths:\n        - svc/billing/**\n",
			want: []Member{{Name: "app-a"}, {Name: "billing", Paths: []string{"svc/billing/**"}}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := loadAreas(t, tc.body)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if diff := cmp.Diff(tc.want, cfg.Areas.Members); diff != "" {
				t.Errorf("Members mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestAreas_MemberNames pins the derived single-source accessor: nil for an
// empty member set (matching the prior []string nil-when-absent semantics) and
// the names in declaration order for a populated set.
func TestAreas_MemberNames(t *testing.T) {
	t.Parallel()
	if got := (Areas{}).MemberNames(); got != nil {
		t.Errorf("MemberNames() on empty Areas = %v, want nil", got)
	}
	a := Areas{Members: []Member{{Name: "app-a", Paths: []string{"x/**"}}, {Name: "billing"}}}
	want := []string{"app-a", "billing"}
	if diff := cmp.Diff(want, a.MemberNames()); diff != "" {
		t.Errorf("MemberNames() mismatch (-want +got):\n%s", diff)
	}
}

// TestAreas_StringFormParity pins AC-2 (M-0179): the legacy E-0043 string form
// parses byte-for-byte as before. Every member's Paths is nil, MemberNames()
// equals the input list in declaration order, and default/required validation
// semantics are unchanged under the label+location migration.
func TestAreas_StringFormParity(t *testing.T) {
	t.Parallel()

	t.Run("nil paths and MemberNames parity", func(t *testing.T) {
		t.Parallel()
		cfg, err := loadAreas(t, "  members:\n    - platform\n    - tooling\n    - billing\n")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		for i, m := range cfg.Areas.Members {
			if m.Paths != nil {
				t.Errorf("Members[%d] (%q) Paths = %v, want nil", i, m.Name, m.Paths)
			}
		}
		want := []string{"platform", "tooling", "billing"}
		if diff := cmp.Diff(want, cfg.Areas.MemberNames()); diff != "" {
			t.Errorf("MemberNames mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("default validates against string form unchanged", func(t *testing.T) {
		t.Parallel()
		cfg, err := loadAreas(t, "  members:\n    - platform\n  default: Uncategorized\n")
		if err != nil {
			t.Fatalf("Load with default: %v", err)
		}
		if cfg.Areas.Default != "Uncategorized" {
			t.Errorf("Default = %q, want Uncategorized", cfg.Areas.Default)
		}
	})

	t.Run("required:true validates against string form unchanged", func(t *testing.T) {
		t.Parallel()
		cfg, err := loadAreas(t, "  members:\n    - platform\n  required: true\n")
		if err != nil {
			t.Fatalf("Load with required: %v", err)
		}
		if !cfg.Areas.Required {
			t.Error("Required = false, want true")
		}
	})

	t.Run("required:true with no members still rejected", func(t *testing.T) {
		t.Parallel()
		if _, err := loadAreas(t, "  required: true\n"); err == nil {
			t.Fatal("Load: want error for required:true with no members, got nil")
		}
	})
}

// TestAreas_RejectsMalformed pins AC-3 (M-0179): a malformed member is rejected
// at decode (UnmarshalYAML) or validate() with a distinct error naming the
// offending member/path. One row per arm a1–a5 plus the cross-form-duplicate
// row. The decode arms (a2, a4) assert the wrapped member context, not the bare
// yaml.v3 text; the validate arms (a1, a3, a5) assert the rule message.
func TestAreas_RejectsMalformed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		body    string
		wantSub string
	}{
		{
			name:    "a1 object member with empty name",
			body:    "  members:\n    - name: \"\"\n      paths:\n        - x/**\n",
			wantSub: "empty member",
		},
		{
			name:    "a1 object member with absent name",
			body:    "  members:\n    - paths:\n        - x/**\n",
			wantSub: "empty member",
		},
		{
			name:    "a2 paths is not a list (decode, wrapped)",
			body:    "  members:\n    - name: app-a\n      paths: notalist\n",
			wantSub: "app-a", // wrapped decode error names the member
		},
		{
			name:    "a2 paths not a list with no name (decode, index locator)",
			body:    "  members:\n    - paths: notalist\n",
			wantSub: "members[0]:", // no name key → index is the operator's locator
		},
		{
			name:    "a3 empty path entry",
			body:    "  members:\n    - name: app-a\n      paths:\n        - \"\"\n",
			wantSub: "empty path entry",
		},
		{
			name:    "a3 whitespace-padded path entry",
			body:    "  members:\n    - name: app-a\n      paths:\n        - \"  x/**  \"\n",
			wantSub: "whitespace",
		},
		{
			name:    "a4 non-!!str scalar member (decode)",
			body:    "  members:\n    - app-a\n    - 42\n",
			wantSub: "neither a string",
		},
		{
			name:    "a4 sequence member (decode)",
			body:    "  members:\n    - app-a\n    - [nested, list]\n",
			wantSub: "neither a string",
		},
		{
			name:    "a5 cross-form duplicate name",
			body:    "  members:\n    - app-a\n    - name: app-a\n      paths:\n        - x/**\n",
			wantSub: "duplicate member",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := loadAreas(t, tc.body)
			if err == nil {
				t.Fatalf("Load: want error for malformed member, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantSub)
			}
		})
	}
}
