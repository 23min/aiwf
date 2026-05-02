package entity

// ForwardRef describes one outbound reference from an entity. Field is
// the YAML key the reference came from (e.g., "parent", "addressed_by");
// Target is the referenced id (bare or composite); AllowedKinds is the
// closed set of kinds the target may resolve to, taken from the kind's
// schema. An empty AllowedKinds means any kind is allowed (open-target
// fields like gap.addressed_by and decision.relates_to).
//
// ForwardRef is the published shape used by both the validator (check
// package) and the reference-graph index (tree package). Callers that
// only need the (referrer, target) pair can ignore Field and AllowedKinds.
type ForwardRef struct {
	Field        string
	Target       string
	AllowedKinds []Kind
}

// ForwardRefs returns every outbound reference the entity carries, one
// ForwardRef per (field, target) pair. Multi-cardinality fields produce
// one entry per target. Empty / absent fields produce no entries.
//
// The function is the single source of truth for "what references does
// an entity make" — consulted by check.refsResolve to validate them and
// by tree.Load to invert them into the reverse-ref index. A drift-
// detection test in the check package pins the result against the per-
// kind schema table.
func ForwardRefs(e *Entity) []ForwardRef {
	if e == nil {
		return nil
	}
	var refs []ForwardRef
	switch e.Kind {
	case KindMilestone:
		if e.Parent != "" {
			refs = append(refs, ForwardRef{Field: "parent", Target: e.Parent, AllowedKinds: []Kind{KindEpic}})
		}
		for _, dep := range e.DependsOn {
			refs = append(refs, ForwardRef{Field: "depends_on", Target: dep, AllowedKinds: []Kind{KindMilestone}})
		}
	case KindADR:
		for _, sup := range e.Supersedes {
			refs = append(refs, ForwardRef{Field: "supersedes", Target: sup, AllowedKinds: []Kind{KindADR}})
		}
		if e.SupersededBy != "" {
			refs = append(refs, ForwardRef{Field: "superseded_by", Target: e.SupersededBy, AllowedKinds: []Kind{KindADR}})
		}
	case KindGap:
		if e.DiscoveredIn != "" {
			refs = append(refs, ForwardRef{Field: "discovered_in", Target: e.DiscoveredIn, AllowedKinds: []Kind{KindMilestone, KindEpic}})
		}
		for _, addr := range e.AddressedBy {
			refs = append(refs, ForwardRef{Field: "addressed_by", Target: addr})
		}
	case KindDecision:
		for _, rel := range e.RelatesTo {
			refs = append(refs, ForwardRef{Field: "relates_to", Target: rel})
		}
	case KindContract:
		for _, a := range e.LinkedADRs {
			refs = append(refs, ForwardRef{Field: "linked_adrs", Target: a, AllowedKinds: []Kind{KindADR}})
		}
	}
	return refs
}
