package projectguidance

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// renderHostFile renders a fictional host instruction file: handwritten text,
// then the aiwf:guidance block carrying guidanceBody, then the routing
// block carrying routeBody.
func renderHostFile(handwritten, guidanceBody, routeBody string) string {
	return handwritten + "\n" +
		GuidanceStartMarker + "\n" + guidanceBody + "\n" + GuidanceEndMarker + "\n\n" +
		routeStart + "\n" + routeBody + "\n" + routeEnd + "\n"
}

// TestRoutedDocuments pins what the router counts as a route: a markdown
// link in any CommonMark form — titled, angle-bracketed, reference-style,
// percent-escaped — resolved against the router's directory, or the
// repository root for a leading slash, anchor dropped. A link with a
// scheme, a link to something other than markdown, and a link leaving the
// repository are not routes.
func TestRoutedDocuments(t *testing.T) {
	t.Parallel()
	router := "Read [a](../docs/a.md#part), [b](b.md), [web](https://example.com/c.md),\n" +
		"[img](../x.png), [out](../../outside.md), [t](../docs/t.md \"titled\"),\n" +
		"[angle](<../docs/my doc.md>), [root](/docs/r.md), [esc](../docs/sp%20ace.md) and [ref][r].\n\n" +
		"[r]: ../docs/ref.md\n"
	want := []string{"docs/a.md", ".guidance/b.md", "docs/t.md", "docs/my doc.md", "docs/r.md", "docs/sp ace.md", "docs/ref.md"}
	if diff := cmp.Diff(want, RoutedDocuments(router)); diff != "" {
		t.Errorf("RoutedDocuments (-want +got):\n%s", diff)
	}
}

// TestHandwrittenText pins that a host file's managed blocks are removed,
// and that malformed markers of one family — a stray marker included —
// leave that family's block handwritten while the other family's block is
// still removed.
func TestHandwrittenText(t *testing.T) {
	t.Parallel()
	if HandwrittenText(renderHostFile("mine\n", "generated v1", "route v1")) != HandwrittenText(renderHostFile("mine\n", "generated v2", "route v2")) {
		t.Error("a change inside the managed blocks must leave the handwritten text unchanged")
	}
	if HandwrittenText(renderHostFile("mine\n", "generated", "route")) == HandwrittenText(renderHostFile("yours\n", "generated", "route")) {
		t.Error("a change outside the managed blocks must change the handwritten text")
	}
	for name, broken := range map[string]string{
		"no end marker": "mine\n" + GuidanceStartMarker + "\nno end marker\n",
		"stray marker":  "mine\n" + GuidanceStartMarker + "\n@x.md\n" + GuidanceEndMarker + "\n" + GuidanceMarkerPrefix + "note -->\n",
	} {
		if got := HandwrittenText(broken); got != broken {
			t.Errorf("%s: HandwrittenText = %q, want the whole file", name, got)
		}
	}
	strayBesideRoute := "mine\n" + GuidanceStartMarker + "\n@x.md\n" + GuidanceEndMarker + "\n" + GuidanceMarkerPrefix + "note -->\n" +
		routeStart + "\nroute\n" + routeEnd + "\n"
	want := "mine\n" + GuidanceStartMarker + "\n@x.md\n" + GuidanceEndMarker + "\n" + GuidanceMarkerPrefix + "note -->\n\n"
	if got := HandwrittenText(strayBesideRoute); got != want {
		t.Errorf("a stray guidance marker beside a routing block: HandwrittenText = %q, want %q", got, want)
	}
}
