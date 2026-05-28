package cliutil

// AnnotationRegisteredVerbs is the key on the root Cobra command's
// Annotations map under which `NewRootCmd` records the newline-
// delimited list of verbs the binary explicitly registers. The
// `trailer-verb-unknown` finding (G-0150) reads this annotation at
// RunE time to distinguish verbs aiwf intentionally exposes from
// commands Cobra auto-adds (`help`, `completion`) during `Execute`.
//
// The annotation is the single source of truth: when the explicit
// set is enumerated, no domain-specific knowledge of Cobra's
// auto-add list is needed. New verbs added to `NewRootCmd` flow
// into the annotation automatically; Cobra auto-adds (which run
// later, during `Execute`) are absent by construction.
//
// Defined in cliutil so both `internal/cli` (which writes it) and
// `internal/cli/check` (which reads it) can reference the same key
// without an import cycle.
//
// Closes G-0150.
const AnnotationRegisteredVerbs = "aiwf:registered-verbs"
