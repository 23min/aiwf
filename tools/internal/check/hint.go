package check

// hintTable maps a finding's Code+Subcode to a one-line "what to do
// about it" hint. Render layers append `— hint: <hint>` to the
// human-readable line; JSON consumers see the same string in the
// `hint` field.
//
// Keep hints actionable and verb-led ("run X", "set Y", "remove Z").
// Avoid restating the failure — the message already does that.
var hintTable = map[string]string{
	"load-error":                "fix the file's structure (YAML frontmatter delimited by `---`), or remove the file if it's not an aiwf entity",
	"ids-unique":                "run `aiwf reallocate <path>` on one of the duplicates to renumber it",
	"frontmatter-shape":         "set the missing field, or correct the id format to match the kind's pattern",
	"status-valid":              "use one of the allowed statuses listed above",
	"refs-resolve/unresolved":   "check the spelling, or remove the reference if the target was deleted",
	"refs-resolve/wrong-kind":   "use a reference of the expected kind",
	"no-cycles/depends_on":      "remove one edge in the cycle to keep the milestone DAG acyclic",
	"no-cycles/supersedes":      "remove the loop in the supersedes/superseded_by chain",
	"titles-nonempty":           "set a non-empty `title:` in the frontmatter",
	"adr-supersession-mutual":   "add this ADR to the other ADR's `supersedes:` list, or remove the back-reference",
	"gap-resolved-has-resolver": "list the resolving milestone(s) in `addressed_by:`, or revert the status to `open`/`wontfix`",
	"reallocate-body-reference": "update the prose to use the new id; aiwf rewrites only frontmatter, not body text",
}

// HintFor returns the canonical action hint for a given code+subcode.
// Returns "" when no hint is registered. Verb-side findings (e.g.,
// reallocate-body-reference) call this so the human-facing suggestion
// stays in one place.
func HintFor(code, subcode string) string {
	if subcode != "" {
		if h, ok := hintTable[code+"/"+subcode]; ok {
			return h
		}
	}
	return hintTable[code]
}

// applyHints fills in Hint on every finding from the hint table.
// Findings whose Hint is already set are left alone, so callers can
// override the default by setting Hint at construction time.
func applyHints(findings []Finding) {
	for i := range findings {
		f := &findings[i]
		if f.Hint != "" {
			continue
		}
		f.Hint = HintFor(f.Code, f.Subcode)
	}
}
