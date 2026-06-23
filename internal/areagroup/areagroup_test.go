package areagroup

import (
	"reflect"
	"testing"
)

// item is a minimal test value: an id and an effective area.
type item struct {
	id   string
	area string
}

func areaOf(i item) string { return i.area }

func ids(items []item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.id
	}
	return out
}

// TestPartition_Basic pins M-0175/AC-1: declared areas appear in members
// order, each carrying its own items; the untagged/undeclared complement
// is appended last under the default label.
func TestPartition_Basic(t *testing.T) {
	t.Parallel()
	items := []item{
		{"E-0001", "platform"},
		{"E-0002", "billing"},
		{"E-0003", ""},         // untagged → complement
		{"E-0004", "platform"}, // second platform
		{"E-0005", "legacy"},   // undeclared (not a member) → complement
	}
	got := Partition(items, areaOf, []string{"platform", "billing"}, "Uncategorized")

	if len(got) != 3 {
		t.Fatalf("got %d groups, want 3 (platform, billing, complement): %+v", len(got), got)
	}
	// platform first (members order), with both platform items.
	if got[0].Area != "platform" || got[0].Label != "platform" {
		t.Errorf("group[0] = {Area:%q Label:%q}, want platform/platform", got[0].Area, got[0].Label)
	}
	if want := []string{"E-0001", "E-0004"}; !reflect.DeepEqual(ids(got[0].Items), want) {
		t.Errorf("platform items = %v, want %v", ids(got[0].Items), want)
	}
	// billing second.
	if got[1].Area != "billing" {
		t.Errorf("group[1].Area = %q, want billing", got[1].Area)
	}
	if want := []string{"E-0002"}; !reflect.DeepEqual(ids(got[1].Items), want) {
		t.Errorf("billing items = %v, want %v", ids(got[1].Items), want)
	}
	// complement last: Area "" marker, default label, holds untagged AND
	// undeclared items (M-0172's area-unknown check is the mis-tag backstop).
	c := got[2]
	if c.Area != "" || c.Label != "Uncategorized" {
		t.Errorf("complement = {Area:%q Label:%q}, want \"\"/Uncategorized", c.Area, c.Label)
	}
	if want := []string{"E-0003", "E-0005"}; !reflect.DeepEqual(ids(c.Items), want) {
		t.Errorf("complement items = %v, want %v (untagged + undeclared)", ids(c.Items), want)
	}
}

// TestPartition_SuppressesEmptyDeclared pins M-0175/AC-5 (first half): a
// declared area with zero items is omitted entirely.
func TestPartition_SuppressesEmptyDeclared(t *testing.T) {
	t.Parallel()
	items := []item{{"E-0001", "platform"}}
	got := Partition(items, areaOf, []string{"platform", "billing", "tooling"}, "Uncategorized")
	// billing and tooling are empty → suppressed; platform + complement remain.
	if len(got) != 2 {
		t.Fatalf("got %d groups, want 2 (platform + complement): %+v", len(got), got)
	}
	if got[0].Area != "platform" {
		t.Errorf("group[0].Area = %q, want platform", got[0].Area)
	}
	if got[1].Area != "" {
		t.Errorf("group[1].Area = %q, want complement", got[1].Area)
	}
}

// TestPartition_AlwaysShowsComplement pins M-0175/AC-5 (second half): the
// complement is always present, even when empty (every item is tagged).
func TestPartition_AlwaysShowsComplement(t *testing.T) {
	t.Parallel()
	items := []item{{"E-0001", "platform"}, {"E-0002", "billing"}}
	got := Partition(items, areaOf, []string{"platform", "billing"}, "Uncategorized")
	if len(got) != 3 {
		t.Fatalf("got %d groups, want 3 (platform, billing, empty complement)", len(got))
	}
	c := got[len(got)-1]
	if c.Area != "" || len(c.Items) != 0 {
		t.Errorf("last group = {Area:%q items:%d}, want empty complement", c.Area, len(c.Items))
	}
}

// TestPartition_DefaultLabelFallback pins M-0175/AC-1: an unset default
// label falls back to the built-in DefaultComplementLabel; a configured
// label is used verbatim.
func TestPartition_DefaultLabelFallback(t *testing.T) {
	t.Parallel()
	items := []item{{"E-0001", ""}}
	fallback := Partition(items, areaOf, []string{"platform"}, "")
	if got := fallback[len(fallback)-1].Label; got != DefaultComplementLabel {
		t.Errorf("empty default → complement label %q, want %q", got, DefaultComplementLabel)
	}
	configured := Partition(items, areaOf, []string{"platform"}, "Backlog")
	if got := configured[len(configured)-1].Label; got != "Backlog" {
		t.Errorf("configured default → complement label %q, want Backlog", got)
	}
}
