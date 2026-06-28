package aiwfyaml

import (
	"strings"
	"testing"
)

// TestRenameAreaMember_PreservesEverythingExceptTheToken pins M-0195/AC-1: the
// surgical rename replaces ONLY the renamed member's name scalar in the source
// bytes, leaving every other byte — comments inside the areas block, sibling
// keys (`required`, `default`), `paths`, member form, indentation — verbatim.
// The new name is re-emitted through yamlScalar (quoted only when it must be).
func TestRenameAreaMember_PreservesEverythingExceptTheToken(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		src      string
		old, new string
		want     string
	}{
		{
			name: "string-form member, sibling required and inner comments preserved",
			src: "# top\n" +
				"areas:\n" +
				"  members:\n" +
				"    # the platform app\n" +
				"    - platform\n" +
				"    - billing\n" +
				"  required: true\n",
			old: "platform", new: "infra",
			want: "# top\n" +
				"areas:\n" +
				"  members:\n" +
				"    # the platform app\n" +
				"    - infra\n" +
				"    - billing\n" +
				"  required: true\n",
		},
		{
			name: "object-form member, paths and required preserved",
			src: "areas:\n" +
				"  members:\n" +
				"    - name: app-a\n" +
				"      paths:\n" +
				"        - \"projects/app-a/**\"\n" +
				"    - billing\n" +
				"  required: true\n",
			old: "app-a", new: "app-alpha",
			want: "areas:\n" +
				"  members:\n" +
				"    - name: app-alpha\n" +
				"      paths:\n" +
				"        - \"projects/app-a/**\"\n" +
				"    - billing\n" +
				"  required: true\n",
		},
		{
			name: "inline comment on the member line preserved (and not rewritten)",
			src: "areas:\n" +
				"  members:\n" +
				"    - platform  # the platform project\n",
			old: "platform", new: "infra",
			want: "areas:\n" +
				"  members:\n" +
				"    - infra  # the platform project\n",
		},
		{
			name: "double-quoted old name replaced canonically",
			src:  "areas:\n  members:\n    - \"platform\"\n",
			old:  "platform", new: "infra",
			want: "areas:\n  members:\n    - infra\n",
		},
		{
			name: "single-quoted old name replaced canonically",
			src:  "areas:\n  members:\n    - 'platform'\n",
			old:  "platform", new: "infra",
			want: "areas:\n  members:\n    - infra\n",
		},
		{
			name: "new name needing quoting is quoted",
			src:  "areas:\n  members:\n    - platform\n",
			old:  "platform", new: "true",
			want: "areas:\n  members:\n    - \"true\"\n",
		},
		{
			// A double-quoted name carrying an escaped quote (`"a\"b"` decodes to
			// a"b) exercises the backslash-escape arm of the close-quote scan: the
			// whole quoted token must be located and replaced, not truncated at the
			// inner quote.
			name: "double-quoted name with an escaped quote",
			src:  "areas:\n  members:\n    - \"a\\\"b\"\n",
			old:  "a\"b", new: "renamed",
			want: "areas:\n  members:\n    - renamed\n",
		},
		{
			// A single-quoted name carrying a doubled quote (`'a''b'` decodes to
			// a'b) exercises the '' escape arm of the close-quote scan.
			name: "single-quoted name with a doubled quote",
			src:  "areas:\n  members:\n    - 'a''b'\n",
			old:  "a'b", new: "renamed",
			want: "areas:\n  members:\n    - renamed\n",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, _, err := ReadBytes([]byte(tc.src))
			if err != nil {
				t.Fatalf("ReadBytes: %v", err)
			}
			if err := doc.RenameAreaMember(tc.old, tc.new); err != nil {
				t.Fatalf("RenameAreaMember: %v", err)
			}
			if got := string(doc.Bytes()); got != tc.want {
				t.Errorf("surgical rename mismatch\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// TestRenameAreaMember_Errors pins the refusal paths: no areas block, and an
// old name that is not a declared member. The verb pre-validates both, but the
// guard keeps the API honest.
func TestRenameAreaMember_Errors(t *testing.T) {
	t.Parallel()
	t.Run("no areas block", func(t *testing.T) {
		t.Parallel()
		doc, _, err := ReadBytes([]byte("hosts: [claude-code]\n"))
		if err != nil {
			t.Fatalf("ReadBytes: %v", err)
		}
		if err := doc.RenameAreaMember("platform", "infra"); err == nil {
			t.Fatal("expected an error on a doc with no areas block")
		}
	})
	t.Run("member not declared", func(t *testing.T) {
		t.Parallel()
		doc, _, err := ReadBytes([]byte("areas:\n  members:\n    - platform\n"))
		if err != nil {
			t.Fatalf("ReadBytes: %v", err)
		}
		if err := doc.RenameAreaMember("ghost", "infra"); err == nil {
			t.Fatal("expected an error when the old name is not a declared member")
		}
	})
	// aiwfyaml's ReadBytes detects the `areas:` key for the byte-range without
	// validating its shape (config.Load does that, upstream of the verb), so
	// these malformed-but-detectable blocks reach the navigation guards.
	t.Run("areas is not a mapping", func(t *testing.T) {
		t.Parallel()
		doc, _, err := ReadBytes([]byte("areas: not-a-mapping\n"))
		if err != nil {
			t.Fatalf("ReadBytes: %v", err)
		}
		if err := doc.RenameAreaMember("platform", "infra"); err == nil {
			t.Fatal("expected an error when areas is not a mapping")
		}
	})
	t.Run("areas has no members key", func(t *testing.T) {
		t.Parallel()
		doc, _, err := ReadBytes([]byte("areas:\n  required: true\n"))
		if err != nil {
			t.Fatalf("ReadBytes: %v", err)
		}
		if err := doc.RenameAreaMember("platform", "infra"); err == nil {
			t.Fatal("expected an error when areas has no members")
		}
	})
	t.Run("members is not a sequence", func(t *testing.T) {
		t.Parallel()
		doc, _, err := ReadBytes([]byte("areas:\n  members: oops\n"))
		if err != nil {
			t.Fatalf("ReadBytes: %v", err)
		}
		if err := doc.RenameAreaMember("platform", "infra"); err == nil {
			t.Fatal("expected an error when members is not a sequence")
		}
	})
}

// TestReadBytes_DetectsAreasWithoutContracts pins that the areas block
// is detected even when no contracts: block is present — the contracts
// path returns early on a missing contracts key, so areas detection
// must not depend on it.
func TestReadBytes_DetectsAreasWithoutContracts(t *testing.T) {
	t.Parallel()
	src := `hosts: [claude-code]
areas:
  members:
    - platform
    - billing
`
	doc, contracts, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if contracts != nil {
		t.Errorf("contracts = %+v, want nil", contracts)
	}
	// RenameAreaMember succeeding proves the areas block was located.
	if err := doc.RenameAreaMember("platform", "infra"); err != nil {
		t.Fatalf("RenameAreaMember after areas-only read: %v", err)
	}
	if !strings.Contains(string(doc.Bytes()), "- infra") {
		t.Errorf("areas block not rewritten:\n%s", doc.Bytes())
	}
}

// TestRenameAreaMember_PreservesTrailingKeysAndComments pins that a top-level
// key and its comment AFTER the areas block — and the in-block `default:` —
// survive the surgical rename: only the member-name token changes.
func TestRenameAreaMember_PreservesTrailingKeysAndComments(t *testing.T) {
	t.Parallel()
	src := `areas:
  members:
    - platform
    - billing
  default: untagged

# trailing comment belongs to html
html:
  out_dir: site
`
	doc, _, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if err := doc.RenameAreaMember("platform", "infra"); err != nil {
		t.Fatalf("RenameAreaMember: %v", err)
	}
	got := string(doc.Bytes())
	for _, want := range []string{"default: untagged", "# trailing comment belongs to html", "html:", "out_dir: site", "- infra"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q after rename:\n%s", want, got)
		}
	}
	if strings.Contains(got, "- platform") {
		t.Errorf("old member still present:\n%s", got)
	}
}

// TestRenameAreaMember_AreasBeforeContracts pins that when both blocks exist,
// renaming an area member leaves the later contracts block intact — the
// surgical splice touches only the member-name token.
func TestRenameAreaMember_AreasBeforeContracts(t *testing.T) {
	t.Parallel()
	src := `areas:
  members:
    - platform
    - billing
contracts:
  validators: {}
  entries: []
`
	doc, _, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if err := doc.RenameAreaMember("platform", "infra"); err != nil {
		t.Fatalf("RenameAreaMember: %v", err)
	}
	got := string(doc.Bytes())
	for _, want := range []string{"- infra", "contracts:", "validators: {}", "entries: []"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q after areas rename:\n%s", want, got)
		}
	}
}
