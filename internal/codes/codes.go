// Package codes defines the typed descriptor for kernel error/finding
// codes that carry their structural class intrinsically (D-0011). A
// legality-pertinent code declares itself as a [Code] value with
// [ClassLegality]; the class is a property of the code value, so the
// closed legality set is enumerable from the declarations themselves —
// no parallel allowlist or central registry can drift from it.
//
// The package is a leaf: it imports nothing from the module.
package codes

// Class names the structural category a kernel code belongs to. It
// distinguishes verb-time legality refusals (named by illegal spec
// cells) from integrity findings that report tree/state inconsistency.
type Class int

const (
	// ClassStructural marks integrity findings: frontmatter shape, id
	// collision, ref resolution, provenance, contract verification. The
	// zero value, so a bare code defaults to structural.
	ClassStructural Class = iota
	// ClassLegality marks verb-time FSM / precondition refusals named by
	// illegal spec cells (e.g. an FSM transition the kind forbids).
	ClassLegality
)

// Code is a typed kernel-code descriptor: a stable string ID paired
// with the structural [Class] it belongs to. The ID is what message and
// JSON consumers see (unchanged from the bare-string era); the Class is
// the marker the legality enumeration derives from.
type Code struct {
	// ID is the stable, machine-readable code string.
	ID string
	// Class is the code's structural category.
	Class Class
}
