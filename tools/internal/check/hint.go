package check

// hintTable maps a finding's Code+Subcode to a one-line "what to do
// about it" hint. Render layers append `— hint: <hint>` to the
// human-readable line; JSON consumers see the same string in the
// `hint` field.
//
// Keep hints actionable and verb-led ("run X", "set Y", "remove Z").
// Avoid restating the failure — the message already does that.
var hintTable = map[string]string{
	"load-error":                        "fix the file's structure (YAML frontmatter delimited by `---`), or remove the file if it's not an aiwf entity",
	"ids-unique":                        "run `aiwf reallocate <path>` on one of the duplicates to renumber it",
	"case-paths":                        "rename one of the colliding paths via `aiwf rename` so they differ in more than just case (case-insensitive filesystems treat them as the same dir)",
	"frontmatter-shape":                 "set the missing field, or correct the id format to match the kind's pattern",
	"id-path-consistent":                "renumber via `aiwf reallocate <path>` (rewrites both sides + updates references), rename the slug via `aiwf rename` if only the slug drifted, or correct the side that's wrong by hand if you're certain which",
	"status-valid":                      "use one of the allowed statuses listed above",
	"refs-resolve/unresolved":           "check the spelling, or remove the reference if the target was deleted",
	"refs-resolve/wrong-kind":           "use a reference of the expected kind",
	"refs-resolve/unresolved-milestone": "the composite id's parent milestone does not exist; check the spelling or create the milestone",
	"refs-resolve/unresolved-ac":        "the parent milestone exists but has no AC with that id; add it to acs[] or fix the reference",
	"no-cycles/depends_on":              "remove one edge in the cycle to keep the milestone DAG acyclic",
	"no-cycles/supersedes":              "remove the loop in the supersedes/superseded_by chain",
	"titles-nonempty":                   "set a non-empty `title:` in the frontmatter",
	"adr-supersession-mutual":           "add this ADR to the other ADR's `supersedes:` list, or remove the back-reference",
	"gap-resolved-has-resolver":         "list the resolving milestone(s) in `addressed_by:`, or revert the status to `open`/`wontfix`",

	"acs-shape/id":                       "fix the AC's id to match `AC-N` and equal its position+1 (cancelled entries count toward position)",
	"acs-shape/title":                    "set a non-empty `title:` on the AC entry",
	"acs-shape/status":                   "use one of the allowed AC statuses listed above",
	"acs-shape/tdd-phase":                "set tdd_phase to one of red|green|refactor|done (required when the milestone is tdd: required)",
	"acs-shape/tdd-policy":               "set the milestone's tdd: to one of required|advisory|none (or omit to default to none)",
	"acs-body-coherence/missing-heading": "add a `### AC-<N> — <title>` heading in the milestone body for this AC, or remove it from acs[]",
	"acs-body-coherence/orphan-heading":  "add the AC to the milestone's frontmatter acs[], or remove the body heading",
	"acs-tdd-audit":                      "advance the AC's tdd_phase to `done` via `aiwf promote <id>/AC-N --phase done`, or relax the milestone's tdd: setting",
	"acs-tdd-tests-missing":              "re-run the TDD cycle through `aiwf promote <id>/AC-N --phase ... --tests \"pass=N fail=N skip=N\"`, or set `tdd.require_test_metrics: false` in aiwf.yaml to silence the warning",
	"acs-title-prose":                    "shorten the AC title to a single short label and move the detail prose into the body section under `### AC-N`; titles render as one big heading",
	"milestone-done-incomplete-acs":      "promote the open ACs to met / deferred / cancelled, or use --force --reason to override (the standing check still surfaces this)",

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

	// I2.5 provenance standing rules. These fire on commit history,
	// not on tree state — hints point to the verb / repair path that
	// would have produced a coherent commit.
	"provenance-trailer-incoherent":          "rewrite or amend the offending commit so the trailer set obeys the required-together / mutually-exclusive rules in `docs/pocv3/design/provenance-model.md`",
	"provenance-force-non-human":             "`--force` requires `aiwf-actor: human/...`; have a human invoke the verb directly, or drop the force",
	"provenance-actor-malformed":             "set `git config user.email` to a valid address and re-run via `aiwf doctor`; the actor trailer is derived from `<localpart>` of the email",
	"provenance-principal-non-human":         "`aiwf-principal:` must be `human/<id>`; agents and bots cannot be principals",
	"provenance-on-behalf-of-non-human":      "`aiwf-on-behalf-of:` must name a human principal; rebuild the trailer from the originating authorize commit's `aiwf-actor:` value",
	"provenance-authorized-by-malformed":     "`aiwf-authorized-by:` must be 7–40 hex (the SHA of the authorize commit); copy it from `aiwf history <scope-entity>`",
	"provenance-authorization-missing":       "the SHA does not name an `aiwf-verb: authorize / aiwf-scope: opened` commit; check for typos or use the full SHA",
	"provenance-authorization-out-of-scope":  "the scope-entity does not reach the target via the reference graph; either authorize the right entity or run the verb on something the existing scope already reaches",
	"provenance-authorization-ended":         "the scope was already ended (terminal-promote or revoke); open a fresh scope with `aiwf authorize <id> --to <agent>`",
	"provenance-no-active-scope":             "an `ai/...` actor needs an active authorization; run `aiwf authorize <id> --to <agent>` before retrying the verb",
	"provenance-audit-only-non-human":        "`--audit-only` is a sovereign act; only humans may backfill audit trails (have a human invoke `aiwf <verb> --audit-only --reason ...`)",
	"provenance-untrailered-entity-commit":   "the commit modified this entity via plain `git commit`; backfill its audit trail with `aiwf cancel <id> --audit-only --reason \"...\"` or `aiwf promote <id> <state> --audit-only --reason \"...\"` — one audit-only per entity; the matching finding clears on the next push",
	"provenance-untrailered-scope-undefined": "the audit range is undefined; configure an upstream (`git push -u origin <branch>`) or pass `aiwf check --since <ref>` to opt back in",

	// Verb-emitted findings (from tools/internal/verb/).
	"slug-dropped-chars":  "the title contained non-ASCII runes that the slug omits; rename via `aiwf rename` if the resulting slug isn't what you want",
	"import-duplicate-id": "the manifest declares the same id more than once; deduplicate the entries before re-running `aiwf import`",
	"import-collision":    "the manifest's explicit id is already taken by an existing entity; re-run with `--on-collision skip|update`, or change the manifest's id",
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
