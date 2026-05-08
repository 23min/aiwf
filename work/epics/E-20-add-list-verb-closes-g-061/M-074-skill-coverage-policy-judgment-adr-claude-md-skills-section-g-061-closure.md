---
id: M-074
title: skill-coverage policy, judgment ADR, CLAUDE.md skills section, G-061 closure
status: in_progress
parent: E-20
tdd: required
acs:
    - id: AC-1
      title: skill-coverage policy file exists, modeled on config_fields_discoverable
      status: met
      tdd_phase: done
    - id: AC-2
      title: Policy enforces non-empty name and description on every embedded skill
      status: open
      tdd_phase: red
    - id: AC-3
      title: Policy enforces skill name matches dir and aiwf-<topic> convention
      status: open
      tdd_phase: red
    - id: AC-4
      title: Policy enforces every top-level verb is documented or in allowlist
      status: open
      tdd_phase: red
    - id: AC-5
      title: Policy enforces every aiwf <verb> mention in skills resolves to a real verb
      status: open
      tdd_phase: red
    - id: AC-6
      title: Allowlist has rationale per entry; show entry rationale references follow-up gap
      status: met
      tdd_phase: done
    - id: AC-7
      title: Follow-up gap for aiwf-show skill exists
      status: met
      tdd_phase: done
    - id: AC-8
      title: Skills judgment ADR allocated and proposed
      status: met
      tdd_phase: done
    - id: AC-9
      title: CLAUDE.md gains Skills policy section and What's enforced row
      status: met
      tdd_phase: done
    - id: AC-10
      title: G-061 promoted to terminal status with closing commit citing this epic
      status: met
      tdd_phase: done
    - id: AC-11
      title: G-085 doc-sweep and gap closure
      status: met
      tdd_phase: done
---

# M-074 — skill-coverage policy, judgment ADR, CLAUDE.md skills section, G-061 closure

## Goal

Lock the AI-discoverability surface for skills against drift via a kernel policy, capture the judgment rule that drove this epic's split-skill design as an ADR, weave both into `CLAUDE.md`'s authoritative ruleset, file the deferred-skill follow-up gap for `aiwf show`, and close G-061 with a commit that cites this epic as its resolution.

## Context

Two policies in `internal/policies/` already enforce that AI-discoverable surfaces don't go undocumented: `PolicyFindingCodesAreDiscoverable` (`discoverability.go`) and `PolicyConfigFieldsAreDiscoverable` (`config_fields_discoverable.go`). Both share the `Violation` shape, `readDiscoverabilityChannels` haystack, and the small allowlist-with-rationale pattern. Skill coverage of verbs is the third leg of the same surface and has been un-policed: a verb can ship without skill coverage, and a skill can reference a verb that doesn't exist (G-061's exact failure mode). This milestone closes that loop. The judgment rule that drove the split-skill design (per-verb default for mutating verbs, topical multi-verb when concept-shaped, no skill when --help suffices, discoverability priority justifies splitting within an otherwise topical group) lives in an ADR — judgment-shaped, not mechanically evaluable. The ADR is the *why*; the policy is the enforced *what*.

## Acceptance criteria

### AC-1 — skill-coverage policy file exists, modeled on config_fields_discoverable

### AC-2 — Policy enforces non-empty name and description on every embedded skill

### AC-3 — Policy enforces skill name matches dir and aiwf-<topic> convention

### AC-4 — Policy enforces every top-level verb is documented or in allowlist

### AC-5 — Policy enforces every aiwf <verb> mention in skills resolves to a real verb

### AC-6 — Allowlist has rationale per entry; show entry rationale references follow-up gap

### AC-7 — Follow-up gap for aiwf-show skill exists

### AC-8 — Skills judgment ADR allocated and proposed

### AC-9 — CLAUDE.md gains Skills policy section and What's enforced row

### AC-10 — G-061 promoted to terminal status with closing commit citing this epic

### AC-11 — G-085 doc-sweep and gap closure

Five sites (CLAUDE.md, three under `docs/pocv3/`, one gap body) advertise the non-existent `aiwf status --kind gap` command. M-072 shipped the canonical replacement (`aiwf list --kind gap`); this AC sweeps the prose and closes the gap. Mechanical search-and-replace across:

- `CLAUDE.md:3`
- `docs/pocv3/README.md:11`
- `docs/pocv3/architecture.md:184`
- `docs/pocv3/archive/gaps-pre-migration.md:3`
- `work/gaps/G-078-*.md:9`

After the doc edits land in the same commit (or commit pair) as the AC-9 CLAUDE.md edit, `aiwf promote G-085 <terminal>` closes the gap with a body citing E-20 — same shape as AC-10's G-061 closure.

## Constraints

- Same precedent: `internal/policies/skill_coverage.go` follows the shape of `config_fields_discoverable.go` exactly — same `Violation` struct, same `readDiscoverabilityChannels` haystack helper, same allowlist-with-rationale-comment pattern. No new framework primitives in `internal/policies/`.
- Mechanical vs. judgment split is non-negotiable. The policy contains *only* mechanically evaluable invariants. Judgment lives in the ADR. The two artifacts cross-reference each other; neither smuggles the other's role.
- AC-4's allowlist must carry a one-line rationale comment per entry in source, exactly like `excluded["actor"]` in `config_fields_discoverable.go:52`. The entry for `show` references the follow-up gap by id.
- AC-7 (follow-up gap) is allocated via `aiwf add gap` — not hand-crafted. The gap's body explains *why* `show` warrants its own skill (body-rendering branches, composite-id handling, AI assistants reach for it constantly) and is filed under `discovered_in: M-074`.
- AC-8 (judgment ADR) is allocated via `aiwf add adr` and lives under `docs/adr/ADR-NNNN-*.md`. Status `proposed` at minimum; ratification is not a blocker for this milestone's `done`.
- AC-10's closing commit is produced by `aiwf promote G-061 <terminal>`; the commit's `aiwf-entity:` trailer references G-061 and the body cites this epic. Do not hand-craft the closure commit.
- AC-11's G-085 closure follows the same pattern as AC-10: `aiwf promote G-085 <terminal>` after the five doc-sweep edits land. The doc sweep is mechanical (a single `aiwf status --kind gap` → `aiwf list --kind gap` substitution at each of the five sites); confirm each site renders correctly afterwards (especially `docs/pocv3/archive/gaps-pre-migration.md`, which is otherwise frozen content).

## Design notes

- Policy implementation outline (refine at start-milestone):
  1. Walk `internal/skills/embedded/aiwf-*/SKILL.md` files; parse frontmatter; assert `name:` non-empty and matches the directory; assert `description:` non-empty; assert name matches `aiwf-<topic>` regex.
  2. Walk `cmd/aiwf/*.go` for top-level Cobra `Use:` strings; build the verb set.
  3. For each verb: assert it appears in some skill body OR appears in the allowlist with a rationale comment.
  4. For each skill body: extract every backticked `aiwf <verb>` mention; assert each resolves to a verb in the verb set.
- Allowlist initial entries (refine at start-milestone): `init`, `doctor`, `update`, `upgrade`, `version`, `verbs`, `schema`, `template`, `whoami`, `import`, `move`, `edit-body`, `cancel`, `roadmap`, `recipes`, `help`, `show`. Each carries a rationale; `show` carries `"deferred — see G-NNN"` where G-NNN is the follow-up gap from AC-7. The other entries are stable "ops verb" or "trivially documented in --help" rationales.
- ADR title (suggested, refine at allocation): "Skills policy: per-verb default; topical multi-verb when concept-shaped; no skill when --help suffices; discoverability priority justifies splitting".
- ADR body covers: the four cases (per-verb / topical / no-skill / split-within-topical); the precedents (`aiwf-contract` for topical bundling, `aiwf-status` and `aiwf-list` after this epic for the split-within-topical case); the principle that skill descriptions enumerate natural-language *query phrasings*, not just verb names; the cross-reference to `internal/policies/skill_coverage.go` as the mechanical companion.
- CLAUDE.md *Skills policy* section (~10 lines): summarize the four cases, point at the ADR for the *why* and the policy file for the enforced *what*. Add one row to the *What's enforced and where* table: rule = "Every verb has skill coverage or an allowlist entry; every `aiwf <verb>` mention in a skill resolves", chokepoint = `internal/policies/skill_coverage.go` test, status = "Blocking via CI test".
- G-061 closure: `aiwf promote G-061 addressed` (or whichever terminal status `gap` uses; `entity.IsTerminal(KindGap, ...)` from M-072's helper resolves it). The closing reason cites E-20.

## Surfaces touched

- `internal/policies/skill_coverage.go` (new)
- `internal/policies/policies_test.go` (test entry)
- `docs/adr/ADR-NNNN-*.md` (new — judgment rule)
- `CLAUDE.md` (Skills policy section + What's enforced row + AC-11 doc-sweep edit)
- `docs/pocv3/README.md` (AC-11 — `aiwf status --kind gap` → `aiwf list --kind gap`)
- `docs/pocv3/architecture.md` (AC-11 — same substitution)
- `docs/pocv3/archive/gaps-pre-migration.md` (AC-11 — same substitution)
- `work/gaps/G-078-*.md` (AC-11 — same substitution in body prose)
- `work/gaps/G-NNN-*.md` (new — follow-up gap for aiwf-show skill)
- G-061 (status promotion only)
- G-085 (status promotion only — AC-11)

## Out of scope

- A new `aiwf-show` embedded skill. The whole point of the deferred entry plus follow-up gap is that this milestone records the absence rather than papers over it. The actual skill ships in a future milestone.
- Migration of this policy into a future `P-NNN` under the `aiwf-rituals` bundle when policy-model.md's opt-in module lands. That migration is name-only and out of scope here.
- Closure of G-068 (discoverability policy misses dynamic finding subcodes). Different policy, different fix shape.
- Any change to the existing `PolicyFindingCodesAreDiscoverable` or `PolicyConfigFieldsAreDiscoverable`.

## Dependencies

- M-072 — the contract-skill drift fix must have landed so AC-5's "every `aiwf <verb>` resolves" check doesn't trip on stale `aiwf list contracts` mentions in `aiwf-contract/SKILL.md`.
- M-073 — the new `aiwf-list` skill must exist so the policy's verb-coverage check passes (without it, `list` would have no skill and would need an allowlist entry, defeating the M-073 work).
- E-14 conventions for `internal/policies/` test wiring (`policies_test.go` runPolicy pattern).

## Coverage notes

- (filled at wrap)

## References

- E-20 epic spec (this milestone's parent).
- G-061 — the gap this milestone closes.
- G-085 — sibling drift case to G-061: five sites advertise `aiwf status --kind gap`, a non-existent flag; M-072 ships `aiwf list --kind gap` and AC-11 sweeps the prose. Closed by AC-11.
- `internal/policies/discoverability.go` — `PolicyFindingCodesAreDiscoverable`. Precedent for haystack and Violation shape.
- `internal/policies/config_fields_discoverable.go` — `PolicyConfigFieldsAreDiscoverable`. Precedent for the allowlist-with-rationale-comment pattern (see `excluded` map at line 52).
- `internal/policies/policies.go` — the package's `Violation` and `WalkGoFiles` primitives this milestone reuses.
- `docs/pocv3/design/policy-model.md` — future opt-in policy module; the ADR notes the migration story (name-only) but this milestone does not depend on the module landing.
- CLAUDE.md kernel principles cited verbatim by the ADR: *"kernel functionality must be AI-discoverable"*, *"the framework's correctness must not depend on the LLM's behavior"*.

---

## Work log

(filled during implementation)

## Decisions made during implementation

- (none — all decisions are pre-locked above)

## Validation

(pasted at wrap)

## Deferrals

- (none)

## Reviewer notes

- (filled at wrap)
