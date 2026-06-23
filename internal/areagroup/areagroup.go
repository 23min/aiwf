// Package areagroup partitions a slice of items into ordered per-area
// groups — the single source of the workstream-area partition logic shared
// by the status, roadmap, and html render surfaces (E-0043, M-0175).
package areagroup

// DefaultComplementLabel is the built-in label for the untagged/undeclared
// complement when `aiwf.yaml: areas.default` is unset (M-0175/AC-1).
const DefaultComplementLabel = "Uncategorized"

// Group is one area partition: the declared area key it represents ("" for
// the untagged/undeclared complement), a display label, and the items that
// belong to it.
type Group[T any] struct {
	Area  string
	Label string
	Items []T
}

// Partition groups items by effective area into ordered Groups, the single
// source of the area-partition logic shared by the status, roadmap, and html
// renderers (E-0043, M-0175/AC-1). areaOf yields an item's effective area
// ("" = untagged); members is the declared `aiwf.yaml: areas.members` set;
// defaultLabel is `areas.default` (a built-in fallback is used when empty).
//
// Ordering and emptiness policy (M-0175/AC-5):
//   - Declared areas appear in members order, each carrying its items in
//     input order. A declared area with zero items is suppressed.
//   - The complement — items whose area is "" (untagged) OR not a declared
//     member (mis-tagged; the M-0172 area-unknown check is that backstop) —
//     is ALWAYS appended last (Area "") under defaultLabel, even when empty,
//     so a grouped view always shows where un-triaged work lives.
//
// Callers render flat (today's output) when members is empty; this helper is
// only invoked once an areas block exists.
func Partition[T any](items []T, areaOf func(T) string, members []string, defaultLabel string) []Group[T] {
	memberSet := make(map[string]bool, len(members))
	for _, m := range members {
		memberSet[m] = true
	}
	byMember := make(map[string][]T, len(members))
	var complement []T
	for _, it := range items {
		a := areaOf(it)
		if a != "" && memberSet[a] {
			byMember[a] = append(byMember[a], it)
		} else {
			complement = append(complement, it)
		}
	}

	groups := make([]Group[T], 0, len(members)+1)
	for _, m := range members {
		if len(byMember[m]) == 0 {
			continue // suppress empty declared area
		}
		groups = append(groups, Group[T]{Area: m, Label: m, Items: byMember[m]})
	}
	label := defaultLabel
	if label == "" {
		label = DefaultComplementLabel
	}
	groups = append(groups, Group[T]{Area: "", Label: label, Items: complement})
	return groups
}
