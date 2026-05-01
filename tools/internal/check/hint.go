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

	"contract-config/missing-entity":        "create a contract entity for this id (`aiwf add contract`), or remove the entry from aiwf.yaml.contracts.entries[]",
	"contract-config/missing-schema":        "fix the `schema:` path in aiwf.yaml.contracts.entries[], or create the file at that location",
	"contract-config/missing-fixtures":      "fix the `fixtures:` path in aiwf.yaml.contracts.entries[], or create the directory",
	"contract-config/no-binding":            "bind the contract via `aiwf contract bind`, or accept it as a registry-only record",
	"contract-config/path-escape":           "ensure schema and fixtures paths in aiwf.yaml resolve inside the repo; check for `..` segments or out-of-repo symlinks",
	"contract-config/validator-unavailable": "install the validator binary on this machine, or set `contracts.strict_validators: false` in aiwf.yaml to demote this to a warning team-wide",
	"fixture-rejected":                      "make the schema accept this fixture, or remove the fixture from valid/",
	"fixture-accepted":                      "tighten the schema to reject this fixture, or move it to valid/",
	"evolution-regression":                  "revert the schema change or migrate the historical fixture",
	"validator-error":                       "every valid fixture failed; the schema or validator invocation is likely broken",
	"environment":                           "install the validator binary or fix `command:` in aiwf.yaml.contracts.validators",
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
