package verb

import (
	"errors"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// TestSubstituteNamePart pins M-0270/AC-1 (F2): rename.go's
// substituteSlug and reallocate.go's substituteID collapse onto one
// shared helper, substituteNamePart, parameterized by a mode enum
// covering the two fixed call shapes those two callers need — the
// no-second-hyphen fallback is the one place they genuinely diverge
// (F2's "verified nuance": rename appends the new slug, reallocate
// discards and replaces).
func TestSubstituteNamePart(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		nameArg     string
		replacement string
		mode        nameSubstitution
		want        string
		wantErr     bool
	}{
		{
			name:        "rename: existing slug is replaced, id prefix kept",
			nameArg:     "E-0019-old-slug",
			replacement: "new-slug",
			mode:        substituteSlugMode,
			want:        "E-0019-new-slug",
		},
		{
			name:        "rename: bare id with no slug gains one by appending",
			nameArg:     "E-0001",
			replacement: "new-slug",
			mode:        substituteSlugMode,
			want:        "E-0001-new-slug",
		},
		{
			name:        "rename: multi-hyphen slug is preserved wholesale as the tail being replaced",
			nameArg:     "E-0019-old-slug-with-many-hyphens",
			replacement: "new-slug",
			mode:        substituteSlugMode,
			want:        "E-0019-new-slug",
		},
		{
			name:        "reallocate: existing slug is kept, id prefix is replaced",
			nameArg:     "E-0019-old-slug",
			replacement: "E-0042",
			mode:        substituteIDMode,
			want:        "E-0042-old-slug",
		},
		{
			name:        "reallocate: bare id with no slug has nothing to preserve, replacement returned bare",
			nameArg:     "E-0001",
			replacement: "E-0042",
			mode:        substituteIDMode,
			want:        "E-0042",
		},
		{
			name:        "reallocate: multi-hyphen slug is preserved wholesale as the kept tail",
			nameArg:     "E-0019-old-slug-with-many-hyphens",
			replacement: "E-0042",
			mode:        substituteIDMode,
			want:        "E-0042-old-slug-with-many-hyphens",
		},
		{
			name:        "no hyphen at all is an error regardless of mode",
			nameArg:     "noHyphenAtAll",
			replacement: "x",
			mode:        substituteSlugMode,
			wantErr:     true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := substituteNamePart(tc.nameArg, tc.replacement, tc.mode)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("substituteNamePart(%q, %q, %v) = %q, want error",
						tc.nameArg, tc.replacement, tc.mode, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("substituteNamePart(%q, %q, %v) unexpected error: %v",
					tc.nameArg, tc.replacement, tc.mode, err)
			}
			if got != tc.want {
				t.Errorf("substituteNamePart(%q, %q, %v) = %q, want %q",
					tc.nameArg, tc.replacement, tc.mode, got, tc.want)
			}
		})
	}
}

// TestRewriteEntityName pins rewriteEntityName's own contract
// directly (the seam renamePaths/reallocatePaths now share): the
// kind-switch between directory-based and file-based entities, and
// error propagation from substitute in both switch arms. The happy
// paths are also exercised indirectly by TestRename_*/TestReallocate_*
// via the real substituteNamePart-backed callers; these tests pin the
// shared helper's contract in isolation, including the
// substitute-errors branch that a well-formed entity path never
// triggers in production (defensive, per the branch-coverage rule).
func TestRewriteEntityName(t *testing.T) {
	t.Parallel()
	okSubstitute := func(name string) (string, error) { return "new-" + name, nil }
	errSubstitute := func(name string) (string, error) { return "", errors.New("substitute failed") }

	t.Run("directory-based kind moves the containing directory", func(t *testing.T) {
		t.Parallel()
		e := &entity.Entity{Kind: entity.KindEpic, Path: "work/epics/E-0001-old-slug/epic.md"}
		source, dest, err := rewriteEntityName(e, okSubstitute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if source != "work/epics/E-0001-old-slug" {
			t.Errorf("source = %q, want %q", source, "work/epics/E-0001-old-slug")
		}
		if dest != "work/epics/new-E-0001-old-slug" {
			t.Errorf("dest = %q, want %q", dest, "work/epics/new-E-0001-old-slug")
		}
	})

	t.Run("directory-based kind propagates a substitute error", func(t *testing.T) {
		t.Parallel()
		e := &entity.Entity{Kind: entity.KindContract, Path: "work/contracts/C-0001-old-slug/contract.md"}
		_, _, err := rewriteEntityName(e, errSubstitute)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("file-based kind moves the entity file, stripping .md before substitute", func(t *testing.T) {
		t.Parallel()
		e := &entity.Entity{Kind: entity.KindGap, Path: "work/gaps/G-0001-old-slug.md"}
		source, dest, err := rewriteEntityName(e, okSubstitute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if source != e.Path {
			t.Errorf("source = %q, want %q", source, e.Path)
		}
		if dest != "work/gaps/new-G-0001-old-slug.md" {
			t.Errorf("dest = %q, want %q", dest, "work/gaps/new-G-0001-old-slug.md")
		}
	})

	t.Run("file-based kind propagates a substitute error", func(t *testing.T) {
		t.Parallel()
		e := &entity.Entity{Kind: entity.KindGap, Path: "work/gaps/G-0001-old-slug.md"}
		_, _, err := rewriteEntityName(e, errSubstitute)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
