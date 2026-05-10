---
id: M-0062
title: tdd flag on aiwf add milestone with project-default fallback
status: draft
parent: E-0016
tdd: required
acs:
    - id: AC-1
      title: --tdd flag writes resolved value to milestone frontmatter
      status: open
      tdd_phase: red
    - id: AC-2
      title: Project default from aiwf.yaml tdd.default applies when --tdd omitted
      status: open
      tdd_phase: red
    - id: AC-3
      title: Verb refuses when neither --tdd nor tdd.default is set
      status: open
      tdd_phase: red
    - id: AC-4
      title: Invalid --tdd values exit with code 2 and usage error
      status: open
      tdd_phase: red
    - id: AC-5
      title: --tdd value tab-completes the closed set
      status: open
      tdd_phase: red
    - id: AC-6
      title: 'aiwf add ac still seeds tdd_phase: red for tdd: required parents'
      status: open
      tdd_phase: red
    - id: AC-7
      title: Subprocess integration test covers all four resolution paths
      status: open
      tdd_phase: red
    - id: AC-8
      title: aiwf-add skill documents --tdd flag and resolution order
      status: open
      tdd_phase: red
---

## Goal

Add `--tdd required|advisory|none` to `aiwf add milestone`, the load-bearing chokepoint for the epic. The verb resolves the policy in this order: explicit flag > `aiwf.yaml: tdd.default` > refuse with a clear error pointing the operator at the new flag. The resolved value is written to the new milestone's frontmatter and becomes the per-milestone source of truth that future audits and verbs read from.

After this milestone, `aiwf add ac` against a `tdd: required` parent continues to seed `tdd_phase: red` (existing behavior — no change required). The `aiwf-add` skill documents the new flag and the resolution order so an LLM following the skill produces well-specified milestones.

## Approach

The flag is added to the existing Cobra command in `cmd/aiwf/add_cmd.go`; the resolver lives next to the existing `aiwf.yaml` consumer code (`internal/configyaml/` or the package that owns the loaded config struct) and is called from the verb body before the entity is allocated. Static completion via `cobra.FixedCompletions` for the closed set, registered the same way the existing `--format`/`--status` completions are. Subprocess integration test exercises every resolution path including the refusal cases, per CLAUDE.md "test the seam." Aggressively reuse the existing project-config load path — do not introduce a parallel reader.

The error message for the no-default-no-flag case is part of the contract: it must name the flag (`--tdd`), the closed-set values, the config field (`aiwf.yaml: tdd.default`), and recommend `--tdd required` for code milestones.

## Acceptance criteria

### AC-1 — --tdd flag writes resolved value to milestone frontmatter

`aiwf add milestone --tdd required --epic E-NN --title "..."` results in a new milestone file whose frontmatter contains `tdd: required` (same for `advisory` and `none`). The field lands in the same atomic commit that creates the milestone — there is no interim state where the milestone exists without its policy. Closed-set values are case-sensitive (`Required` is rejected, see AC-4). The `tdd:` line appears in the frontmatter at a stable position relative to other fields (consistent with how `acs:` is inserted today by `aiwf add ac`). Code: flag bind in `cmd/aiwf/add_cmd.go`; serialization in the existing milestone-write path under `internal/entity/`.

### AC-2 — Project default from aiwf.yaml tdd.default applies when --tdd omitted

With `aiwf.yaml` containing `tdd.default: required`, invoking `aiwf add milestone --epic E-NN --title "..."` (no `--tdd` flag) writes `tdd: required` into the new milestone's frontmatter as if the flag had been passed explicitly. The resolved value, not the literal string `default`, is written — the milestone's frontmatter records the *materialized* policy, not a reference to the project config, so subsequent edits to `aiwf.yaml` do not retroactively change historical milestones. Same behavior for project defaults of `advisory` and `none`. The resolver is called from the verb body before id allocation (see AC-3 for the no-default branch).

### AC-3 — Verb refuses when neither --tdd nor tdd.default is set

With `aiwf.yaml` lacking `tdd.default` (or no `aiwf.yaml` at all), invoking `aiwf add milestone --epic E-NN --title "..."` without `--tdd` exits with code 2 (usage error) **before** allocating any id, acquiring the repo lock, or touching disk. The error message is part of the contract and names: the `--tdd` flag, the three closed-set values (`required`, `advisory`, `none`), the `aiwf.yaml: tdd.default` field, and a recommendation that code milestones use `--tdd required`. The integration test (AC-7) asserts the message contains each of those tokens. Refusal happens in `PreRunE` so the no-default case is uniform with the invalid-value case.

### AC-4 — Invalid --tdd values exit with code 2 and usage error

`aiwf add milestone --tdd yes --epic ...` (or any value outside `required|advisory|none`, including `Required`, `REQUIRED`, empty string) exits with code 2 and a Cobra usage error naming the flag and the closed set. No id is allocated; no file is touched; no lock is acquired. Validation runs in `PreRunE` via the same closed-set predicate the resolver uses (single source of truth — no parallel validator). The Cobra-rendered error includes both the rejected value and the allowed set, matching the format other closed-set flags (`--status`, `--format`) produce on bad input.

### AC-5 — --tdd value tab-completes the closed set

In a shell with completion sourced (`source <(aiwf completion zsh)` or bash equivalent), `aiwf add milestone --tdd <TAB>` enumerates exactly `required`, `advisory`, `none` and nothing else. Wired through `cmd.RegisterFlagCompletionFunc("tdd", cobra.FixedCompletions([]string{"required","advisory","none"}, cobra.ShellCompDirectiveNoFileComp))` — the same pattern used for `--status` and `--format` (see M-0053). The completion-drift policy test (`cmd/aiwf/completion_drift_test.go`) either passes unchanged (its enumeration generalizes over flag-completion bindings) or gains an explicit entry; under no circumstances does the new flag land without completion wiring (per the kernel's auto-completion design principle in CLAUDE.md).

### AC-6 — aiwf add ac still seeds tdd_phase: red for tdd: required parents

Regression check on the existing `aiwf add ac` behavior. After creating a milestone via `aiwf add milestone --tdd required --epic E-NN --title "..."`, running `aiwf add ac M-XX --title "..."` produces an AC whose frontmatter has `tdd_phase: red` — the only legal entry phase under the FSM when the parent milestone is `tdd: required` (per `aiwf-add` skill: "an AC is seeded with `tdd_phase: red` — the only legal entry phase under the FSM"). No code change is expected for this AC; it exists to pin the contract between the new `--tdd` resolution and the pre-existing AC-seeding behavior so a future refactor of either side cannot silently break the chain.

### AC-7 — Subprocess integration test covers all four resolution paths

A binary-level test (`go build -o $TMP/aiwf ./cmd/aiwf` then `exec.Command(...)` per CLAUDE.md "test the seam") in `cmd/aiwf/binary_integration_test.go` (the existing `version`-verb pattern) exercises:

1. Explicit `--tdd required`, no project default — milestone frontmatter contains `tdd: required`, exit 0.
2. Omitted `--tdd`, project default `required` — milestone frontmatter contains `tdd: required`, exit 0.
3. Omitted `--tdd`, no project default — exit 2; stderr contains the four contract tokens from AC-3 (`--tdd`, `tdd.default`, the three closed-set values).
4. Invalid `--tdd value` (e.g., `yes`) — exit 2; no milestone file is created; no id is consumed (the next `aiwf add milestone` allocates the same id the failed run would have).

The test runs against a fresh tempdir per case (clean planning tree), seeded only with the minimum needed (`aiwf init` for cases 2 / 3 / 4; nothing for case 1's no-yaml setup).

### AC-8 — aiwf-add skill documents --tdd flag and resolution order

The `aiwf-add` skill source (`internal/skillsembed/aiwf-add/SKILL.md` or the equivalent generation site) gains:

- A row in (or after) the milestone required-flags table listing `--tdd <required|advisory|none>` as required, with a footnote naming `aiwf.yaml: tdd.default` as the project-level fallback.
- A "TDD policy" subsection naming the resolution order (explicit flag > project default > refusal) and the explicit-opt-out shape (`--tdd none`).
- A "Don't" entry: do not omit `--tdd` and rely on a missing project default — the verb refuses; the right move is to set `aiwf.yaml: tdd.default` once at the repo level (per `aiwf init` / `aiwf update`) or to pass `--tdd` explicitly per invocation.

The change is verified by the discoverability policy test (`internal/policies/PolicyFindingCodesAreDiscoverable` and the broader skill-doc enumeration from G-0021) which already enforces that every kernel surface is mentioned in at least one channel CLAUDE.md names.

