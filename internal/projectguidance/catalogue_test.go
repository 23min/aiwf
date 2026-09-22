package projectguidance

import (
	"errors"
	"strings"
	"testing"
)

// The grammar is defined by engineering-guidance/CATALOGUE.md.
func TestParseCatalogue_RejectsMalformedMetadata(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, old, replacement string }{
		{"invalid json", fixtureCatalogue, "{"},
		{"trailing json", fixtureCatalogue, fixtureCatalogue + " {}"},
		{"unknown root field", `"packs":`, `"extra":true,"packs":`},
		{"unknown pack field", `"id":"go/cobra"`, `"extra":true,"id":"go/cobra"`},
		{"wrong field case", `"detect":`, `"Detect":`},
		{"missing patterns", `,"detect":["go.mod","*.go"]`, ""},
		{"null patterns", `["go.mod","*.go"]`, "null"},
		{"null after description", `"description":"Go CLI guidance"`, `"description":"Go CLI guidance","description":null`},
		{"wrong type", `"Go CLI guidance"`, "3"},
		{"blank description", `"Go CLI guidance"`, `"  "`},
		{"invalid id", `"go/cobra"`, `"Go/cobra"`},
		{"duplicate id", `"id":"explicit"`, `"id":"go/cobra"`},
		{"empty packs", fixtureCatalogue, `{"packs":[]}`},
		{"empty files", `["packs/go/cobra/guide.md"]`, "[]"},
		{"duplicate files", `["packs/go/cobra/guide.md"]`, `["packs/go/cobra/guide.md","packs/go/cobra/guide.md"]`},
		{"outside pack", `packs/go/cobra/guide.md`, `packs/explicit/guide.md`},
		{"parent path", `packs/go/cobra/guide.md`, `packs/go/cobra/../guide.md`},
		{"dot path", `packs/go/cobra/guide.md`, `packs/go/cobra/./guide.md`},
		{"empty segment", `packs/go/cobra/guide.md`, `packs/go/cobra//guide.md`},
		{"absolute path", `packs/go/cobra/guide.md`, `/packs/go/cobra/guide.md`},
		{"backslash", `packs/go/cobra/guide.md`, `packs/go/cobra/a\\b.md`},
		{"wrong extension", `packs/go/cobra/guide.md`, `packs/go/cobra/guide.txt`},
		{"control path", `packs/go/cobra/guide.md`, `packs/go/cobra/a\n.md`},
		{"duplicate pattern", `["go.mod","*.go"]`, `["go.mod","go.mod"]`},
		{"empty pattern", `["go.mod","*.go"]`, `[""]`},
		{"recursive pattern", `["go.mod","*.go"]`, `["**.go"]`},
		{"path pattern", `["go.mod","*.go"]`, `["src/*.go"]`},
		{"bracket pattern", `["go.mod","*.go"]`, `["[ab].go"]`},
		{"dot pattern", `["go.mod","*.go"]`, `["."]`},
		{"parent pattern", `["go.mod","*.go"]`, `[".."]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseCatalogue([]byte(strings.Replace(fixtureCatalogue, tc.old, tc.replacement, 1)))
			if !errors.Is(err, ErrInvalidSource) {
				t.Fatalf("got %v; want invalid source", err)
			}
		})
	}
}

func TestParseCatalogue_AcceptsGrammar(t *testing.T) {
	t.Parallel()
	raw := strings.Replace(fixtureCatalogue, `["go.mod","*.go"]`, `["A_0-?.go","*","x.y"]`, 1)
	raw = strings.Replace(raw, "guide.md", "Guide notes.md", 1)
	got, err := parseCatalogue([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got.Packs[0].Files[0] != "packs/go/cobra/Guide notes.md" {
		t.Fatalf("changed path: %+v", got)
	}
}

func TestParseCatalogue_InvalidUTF8(t *testing.T) {
	t.Parallel()
	if _, err := parseCatalogue([]byte{0xff}); !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("got %v", err)
	}
}
