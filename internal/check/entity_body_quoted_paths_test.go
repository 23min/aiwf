package check

import "testing"

func TestWalkDroppedBodySections_QuotedPaths(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, slug string }{
		{"quote", `has "quote"`},
		{"backslash", `has\backslash`},
		{"tab", "has\ttab"},
		{"newline", "has\nnewline"},
		{"carriage return", "has\rcarriage"},
		{"record separator", "has\x1erecord"},
		{"field separator", "has\x1ffield"},
		{"combined separators", "has\x1e\x1f\n\t\"\\end"},
		{"unicode", "café"},
		{"header marker", "commit fake header"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := "work/gaps/G-0001-" + tc.slug + ".md"
			full := gapFile("G-0001", "", whatsMissing, whyItMatters)
			partial := gapFile("G-0001", "", whatsMissing)
			t.Run("credits drop before later commit", func(t *testing.T) {
				t.Parallel()
				f := newWalkerFixture(t)
				base := f.put(path, full, "seed")
				f.writeFile("commit fake header\n\x1e\x1f", "unrelated path in the drop commit")
				drop := f.put(path, partial, "drop")
				f.put("later.txt", "later", "later")
				assertDropped(t, []DroppedBodySection{{SHA: drop, Path: path, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
			})
			t.Run("follows rename after drop", func(t *testing.T) {
				t.Parallel()
				f := newWalkerFixture(t)
				base := f.put(path, full, "seed")
				drop := f.put(path, partial, "drop")
				f.run("git", "mv", path, gapRenamed)
				f.commit("rename")
				assertDropped(t, []DroppedBodySection{{SHA: drop, Path: gapRenamed, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
			})
			t.Run("follows rename into quoted path", func(t *testing.T) {
				t.Parallel()
				f := newWalkerFixture(t)
				base := f.put(gapPath, full, "seed")
				drop := f.put(gapPath, partial, "drop")
				f.run("git", "mv", gapPath, path)
				f.commit("rename")
				assertDropped(t, []DroppedBodySection{{SHA: drop, Path: path, EntityID: "G-0001", Section: whyItMatters}}, walkFrom(t, f, base))
			})
			t.Run("preserves imported omission", func(t *testing.T) {
				t.Parallel()
				f := newWalkerFixture(t)
				base := f.put("seed.txt", "seed", "seed")
				f.put(path, partial, "import", "aiwf-verb: import", "aiwf-entity: G-0001")
				f.put(path, partial+"More prose.\n", "edit")
				assertDropped(t, nil, walkFrom(t, f, base))
			})
			t.Run("preserves trunk omission", func(t *testing.T) {
				t.Parallel()
				f := newWalkerFixture(t)
				base := f.put(path, full, "seed")
				trunk := f.put(path, partial, "drop on trunk")
				f.put(path, partial+"More prose.\n", "edit")
				assertDropped(t, nil, walkWithTrunk(t, f, base, trunk))
			})
			t.Run("preserves starting omission after rename", func(t *testing.T) {
				t.Parallel()
				f := newWalkerFixture(t)
				base := f.put(path, partial, "seed")
				f.run("git", "mv", path, gapRenamed)
				f.put(gapRenamed, partial+"More prose.\n", "rename and edit")
				assertDropped(t, nil, walkFrom(t, f, base))
			})
		})
	}
}

func TestWalkDroppedBodySections_EmptyRange(t *testing.T) {
	t.Parallel()
	f := newWalkerFixture(t)
	base := f.put(gapPath, gapFile("G-0001", "", whatsMissing), "seed")
	assertDropped(t, nil, walkFrom(t, f, base))
}
