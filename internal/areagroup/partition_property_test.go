package areagroup

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

// This file holds generative property tests over Partition (M-0176). The
// example-based tests in areagroup_test.go pin the cases the author had in
// mind; these fabricate thousands of inputs per run and assert the partition
// invariants across all of them, turning "Partition never silently drops or
// duplicates an item" from a hoped-for property into a mechanical floor — the
// Tier-0 base under E-0044's trust claim. They sample the input space; they do
// not prove the invariant for all inputs.
//
// Determinism: each property drives testing/quick with a fixed-seed
// *rand.Rand, so a green run is reproducible and any counterexample is stable
// (no wall-clock dependence, per the repo's test discipline).
//
// Precondition respected by the generator: members is the declared
// aiwf.yaml: areas.members set, which config validation guarantees is unique
// and free of empty/whitespace entries (internal/config Areas.validate). The
// generator therefore emits only unique, non-empty member names — feeding
// duplicates would be an out-of-contract input, not a real counterexample.

const (
	maxPartitionMembers = 5
	maxPartitionItems   = 12
	propertyMaxCount    = 2000
)

// areaAlphabet is deliberately small so randomly-generated item areas collide
// with declared member names often enough to exercise the declared-bucket
// path, while still producing values that fall outside the member set and land
// in the complement.
const areaAlphabet = "abcde"

// partitionInput is one generated Partition call: a unique-member set, a slice
// of items each carrying an effective area, and a complement default label.
type partitionInput struct {
	members      []string
	items        []item
	defaultLabel string
}

// Generate implements testing/quick.Generator, fabricating a valid Partition
// input: 0..maxPartitionMembers unique non-empty members, 0..maxPartitionItems
// items whose areas span declared / undeclared / untagged, and a default label
// that is sometimes empty (to exercise the built-in fallback).
func (partitionInput) Generate(r *rand.Rand, _ int) reflect.Value {
	nMembers := r.Intn(maxPartitionMembers + 1)
	members := make([]string, 0, nMembers)
	seen := make(map[string]bool, nMembers)
	for len(members) < nMembers {
		name := randArea(r)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		members = append(members, name)
	}

	nItems := r.Intn(maxPartitionItems + 1)
	items := make([]item, nItems)
	for i := range items {
		items[i] = item{id: fmt.Sprintf("E-%04d", i), area: pickArea(r, members)}
	}

	label := ""
	if r.Intn(2) == 0 {
		label = randArea(r)
	}
	return reflect.ValueOf(partitionInput{members: members, items: items, defaultLabel: label})
}

// randArea draws a short string over a small alphabet; length 0 yields "" (the
// untagged value).
func randArea(r *rand.Rand) string {
	n := r.Intn(4) // 0..3
	b := make([]byte, n)
	for i := range b {
		b[i] = areaAlphabet[r.Intn(len(areaAlphabet))]
	}
	return string(b)
}

// pickArea chooses an item's effective area, biased to cover all three
// categories: untagged (""), a declared member, and an arbitrary value that is
// usually undeclared (but may coincide with a member — also valid).
func pickArea(r *rand.Rand, members []string) string {
	switch r.Intn(4) {
	case 0:
		return "" // untagged → complement
	case 1:
		if len(members) > 0 {
			return members[r.Intn(len(members))] // declared
		}
		return ""
	default:
		return randArea(r) // arbitrary; usually undeclared → complement
	}
}

// propertyConfig builds a deterministic testing/quick config seeded so runs
// are reproducible.
func propertyConfig(seed int64) *quick.Config {
	return &quick.Config{
		MaxCount: propertyMaxCount,
		Rand:     rand.New(rand.NewSource(seed)),
	}
}

func memberSetOf(members []string) map[string]bool {
	set := make(map[string]bool, len(members))
	for _, m := range members {
		set[m] = true
	}
	return set
}

// TestPartition_Property_TotalAndDisjoint pins M-0176/AC-1: for any input,
// every item lands in exactly one output group — none dropped, none
// duplicated, none fabricated (count-in == count-out, with each input id
// appearing exactly once across the flattened output).
func TestPartition_Property_TotalAndDisjoint(t *testing.T) {
	t.Parallel()
	var note string
	property := func(in partitionInput) bool {
		groups := Partition(in.items, areaOf, in.members, in.defaultLabel)
		counts := make(map[string]int, len(in.items))
		total := 0
		for _, g := range groups {
			for _, it := range g.Items {
				counts[it.id]++
				total++
			}
		}
		if total != len(in.items) {
			note = fmt.Sprintf("count-out %d != count-in %d", total, len(in.items))
			return false
		}
		for _, it := range in.items {
			if counts[it.id] != 1 {
				note = fmt.Sprintf("item %s appears in %d groups, want exactly 1", it.id, counts[it.id])
				return false
			}
		}
		if len(counts) != len(in.items) {
			note = fmt.Sprintf("output carries %d distinct ids, input had %d (fabricated id)", len(counts), len(in.items))
			return false
		}
		return true
	}
	if err := quick.Check(property, propertyConfig(1)); err != nil {
		t.Errorf("totality/disjointness: %s\n%v", note, err)
	}
}

// TestPartition_Property_ComplementCorrect pins M-0176/AC-2: exactly one group
// carries the complement marker (Area ""), it holds exactly the items whose
// area is "" or not a declared member (in input order), and its label is the
// configured default — or the built-in fallback when the default is empty.
func TestPartition_Property_ComplementCorrect(t *testing.T) {
	t.Parallel()
	var note string
	property := func(in partitionInput) bool {
		members := memberSetOf(in.members)
		var expected []item
		for _, it := range in.items {
			if it.area == "" || !members[it.area] {
				expected = append(expected, it)
			}
		}
		groups := Partition(in.items, areaOf, in.members, in.defaultLabel)
		complementCount := 0
		var comp Group[item]
		for _, g := range groups {
			if g.Area == "" {
				complementCount++
				comp = g
			}
		}
		if complementCount != 1 {
			note = fmt.Sprintf("found %d complement groups (Area \"\"), want exactly 1", complementCount)
			return false
		}
		if !reflect.DeepEqual(ids(comp.Items), ids(expected)) {
			note = fmt.Sprintf("complement items = %v, want %v", ids(comp.Items), ids(expected))
			return false
		}
		wantLabel := in.defaultLabel
		if wantLabel == "" {
			wantLabel = DefaultComplementLabel
		}
		if comp.Label != wantLabel {
			note = fmt.Sprintf("complement label = %q, want %q", comp.Label, wantLabel)
			return false
		}
		return true
	}
	if err := quick.Check(property, propertyConfig(2)); err != nil {
		t.Errorf("complement correctness: %s\n%v", note, err)
	}
}

// TestPartition_Property_DeclaredOrderAndComplementLast pins M-0176/AC-3:
// declared areas appear in members order carrying their items in input order,
// a declared area with no items is suppressed, each declared group labels
// itself with its area, and the complement is always the final group.
func TestPartition_Property_DeclaredOrderAndComplementLast(t *testing.T) {
	t.Parallel()
	var note string
	property := func(in partitionInput) bool {
		members := memberSetOf(in.members)
		byMember := make(map[string][]item)
		for _, it := range in.items {
			if it.area != "" && members[it.area] {
				byMember[it.area] = append(byMember[it.area], it)
			}
		}
		var wantDeclared []string
		for _, m := range in.members {
			if len(byMember[m]) > 0 {
				wantDeclared = append(wantDeclared, m)
			}
		}

		groups := Partition(in.items, areaOf, in.members, in.defaultLabel)
		if len(groups) == 0 {
			note = "no groups emitted; complement must always be present"
			return false
		}
		if last := groups[len(groups)-1]; last.Area != "" {
			note = fmt.Sprintf("last group Area = %q, want complement \"\"", last.Area)
			return false
		}
		declared := groups[:len(groups)-1]
		if len(declared) != len(wantDeclared) {
			note = fmt.Sprintf("emitted %d declared groups, want %d (suppression)", len(declared), len(wantDeclared))
			return false
		}
		for i, g := range declared {
			if g.Area != wantDeclared[i] {
				note = fmt.Sprintf("declared group %d Area = %q, want %q (members order)", i, g.Area, wantDeclared[i])
				return false
			}
			if g.Label != g.Area {
				note = fmt.Sprintf("declared group %q Label = %q, want Label == Area", g.Area, g.Label)
				return false
			}
			if !reflect.DeepEqual(ids(g.Items), ids(byMember[g.Area])) {
				note = fmt.Sprintf("declared group %q items = %v, want %v (input order)", g.Area, ids(g.Items), ids(byMember[g.Area]))
				return false
			}
		}
		return true
	}
	if err := quick.Check(property, propertyConfig(3)); err != nil {
		t.Errorf("declared order / suppression / complement-last: %s\n%v", note, err)
	}
}
