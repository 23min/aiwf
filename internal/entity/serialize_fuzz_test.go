package entity

import (
	"bytes"
	"strings"
	"testing"
	"unicode"
)

// FuzzSlugify drives Slugify with arbitrary strings and checks structural
// invariants the production code commits to. Filed under G44 item 1.
func FuzzSlugify(f *testing.F) {
	for _, seed := range []string{
		"",
		"hello world",
		"Café au Lait",
		"!!!---!!!",
		"  leading and trailing  ",
		"a",
		"123",
		"こんにちは",
		"mixed-CASE_with.PUNCT",
		"M-001 (something)",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		slug, dropped := SlugifyDetailed(in)

		// Output is ASCII-only.
		for _, r := range slug {
			if r > unicode.MaxASCII {
				t.Fatalf("non-ASCII rune %q in slug %q (input %q)", r, slug, in)
			}
		}
		// No leading or trailing hyphen.
		if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
			t.Fatalf("slug %q has leading/trailing hyphen (input %q)", slug, in)
		}
		// No consecutive hyphens.
		if strings.Contains(slug, "--") {
			t.Fatalf("slug %q has consecutive hyphens (input %q)", slug, in)
		}
		// Output characters are restricted to [a-z0-9-].
		for _, r := range slug {
			ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
			if !ok {
				t.Fatalf("disallowed rune %q in slug %q (input %q)", r, slug, in)
			}
		}
		// Idempotence: re-slugifying a slug yields the same slug.
		if again, _ := SlugifyDetailed(slug); again != slug {
			t.Fatalf("not idempotent: SlugifyDetailed(%q)=%q, second pass=%q", in, slug, again)
		}
		// Dropped runes are non-ASCII alphanumerics by contract; punctuation
		// drops are silent.
		for _, r := range dropped {
			if r <= unicode.MaxASCII {
				t.Fatalf("dropped rune %q is ASCII; input %q", r, in)
			}
		}
		// Slugify is the no-detail wrapper.
		if Slugify(in) != slug {
			t.Fatalf("Slugify and SlugifyDetailed disagree on %q", in)
		}
	})
}

// FuzzSplit drives Split with arbitrary byte input and checks that it
// never panics and that the structural invariants on returned slices
// hold. Filed under G44 item 1.
func FuzzSplit(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		[]byte(""),
		[]byte("no frontmatter at all\n"),
		[]byte("---\n---\n"),
		[]byte("---\nid: M-001\ntitle: foo\n---\nbody here\n"),
		[]byte("---\nbroken: yaml: shape\n---\nbody\n"),
		[]byte("---\r\nid: G-001\r\n---\r\nwindows line endings\r\n"),
		[]byte("\xef\xbb\xbf---\nid: E-01\n---\nutf8 BOM\n"),
		[]byte("---\nno closing delim\n"),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in []byte) {
		fm, body, ok := Split(in)
		if !ok {
			// On failure, both outputs must be nil.
			if fm != nil || body != nil {
				t.Fatalf("ok=false but fm=%q body=%q", fm, body)
			}
			return
		}
		// On success: combined output bytes are reachable from the
		// input. We don't require byte-equality (Split strips the
		// delimiter lines and the optional BOM), but we do require
		// that the body suffix appears verbatim in the input.
		if !bytes.Contains(in, body) {
			t.Fatalf("body %q not found verbatim in input %q", body, in)
		}
		// Frontmatter must not contain a `---` line on its own — that
		// would mean the splitter consumed the closing delimiter into
		// fm rather than terminating on it.
		for _, line := range bytes.Split(fm, []byte("\n")) {
			if bytes.Equal(bytes.TrimRight(line, "\r"), []byte("---")) {
				t.Fatalf("frontmatter contains a `---` line: fm=%q", fm)
			}
		}
	})
}
