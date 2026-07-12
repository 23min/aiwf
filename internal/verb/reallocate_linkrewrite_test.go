package verb_test

// reallocate_linkrewrite_test.go — M-0248/AC-1 real-tree integration
// test for routing `aiwf reallocate`'s path-link rewriting through
// the shared link-destination rewrite primitive (M-0245), rather than
// the blind id-token substring replace that also happens to touch
// text that isn't a real entity-path link.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestReallocate_LinkDestinationPrecision pins M-0248/AC-1: a real
// markdown link to the renumbered entity's old path is rewritten to
// its new path via the shared primitive; a URL-shaped link
// destination that merely contains the old id as a substring (not a
// repo-relative entity reference) stays byte-identical, since neither
// the primitive nor the bare-id pass treats a link-path region as
// fair game unless it actually resolves to a moved entity; and a bare
// old-id mention inside a code span is rewritten by the separate
// id-token pass, not the link primitive.
//
// The URL case is the load-bearing assertion: the pre-M-0248
// mechanism (a blind `\bG-0001\b` substring replace over the whole
// body) cannot tell a real entity-path link from an id-shaped
// substring inside an unrelated URL, and corrupts the latter. The
// region-aware primitive treats it as out of scope on both counts —
// not a link to a moved entity, and (once routed through the shared
// primitive) not a bare token either.
func TestReallocate_LinkDestinationPrecision(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target gap", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	target := r.tree().ByID("G-0001")
	if target == nil {
		t.Fatal("G-0001 missing")
	}
	targetPath := target.Path

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Linking gap", testActor, verb.AddOptions{
		BodyOverride: []byte("## What's missing\n\n" +
			"See [the target](" + targetPath + ") for context, " +
			"an unrelated [tracker link](https://example.com/issues/G-0001) that must stay byte-identical, " +
			"and a bare mention `G-0001` in a code span.\n\n" +
			"## Why it matters\n\nFixture.\n"),
	}))
	linking := r.tree().ByID("G-0002")
	if linking == nil {
		t.Fatal("G-0002 missing")
	}
	linkingPath := linking.Path

	res, err := verb.Reallocate(r.ctx, r.tree(), "G-0001", testActor)
	if err != nil {
		t.Fatal(err)
	}
	newID, _ := res.Metadata["new_id"].(string)
	if newID == "" {
		t.Fatal("expected new_id in metadata")
	}

	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatal(applyErr)
	}

	newTarget := r.tree().ByID(newID)
	if newTarget == nil {
		t.Fatalf("%s missing after reallocate", newID)
	}

	body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(linkingPath)))
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)

	if !strings.Contains(got, "("+newTarget.Path+")") {
		t.Errorf("real link destination not rewritten to %s:\n%s", newTarget.Path, got)
	}
	if !strings.Contains(got, "(https://example.com/issues/G-0001)") {
		t.Errorf("URL-shaped destination must stay byte-identical:\n%s", got)
	}
	if !strings.Contains(got, "`"+newID+"`") {
		t.Errorf("bare code-span mention not rewritten to %s:\n%s", newID, got)
	}
}
