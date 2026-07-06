---
id: M-0235
title: 'Generalized hook registry: aiwf.yaml-declared, persisted consent'
status: in_progress
parent: E-0059
tdd: required
acs:
    - id: AC-1
      title: 'aiwf.yaml hooks: schema + aiwf.example.yaml regen'
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf init gates undecided hooks via TTY prompt / --enable-hook flag
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf update gates only newly-introduced hooks; syncs decided hooks silently
      status: open
      tdd_phase: red
    - id: AC-4
      title: 'Hooks settings writer: no-clobber, .bak backup, multi-event-array composition'
      status: open
      tdd_phase: red
    - id: AC-5
      title: New hooks materialization category + aiwf doctor drift reporting
      status: open
      tdd_phase: red
---

## Goal

Build the infrastructure so any Claude Code hook aiwf ships (`SessionStart`,
`SubagentStart`, …) can be materialized into a consumer's `.claude/hooks/`
and activated in the consumer's `.claude/settings.json`, gated by a per-hook
decision recorded in `aiwf.yaml`'s new `hooks:` table — persisted once,
shared across a team's clones, rather than re-asked per invocation per
developer.

## Context

ADR-0015 gates the statusline's settings.json edit on a per-invocation TTY
prompt / `--wire-settings` flag, with no persisted record. ADR-0018
generalizes the underlying risk-calibrated-consent principle to a second
artifact (`CLAUDE.md`), using a different, fully-automatic mechanism suited
to that edit's near-zero risk. Neither fits a hook, which changes runtime
behavior and needs its own per-hook consent that persists rather than being
re-litigated every run. ADR-0032 captures this as the third instance of the
family; this milestone implements it. No concrete hook ships here — that is
the next milestone's job, registered against what this one builds.

## Acceptance criteria

Tracked in frontmatter `acs[]` and detailed in the `### AC-1` … `### AC-5` sections
below. AC-1 and AC-2 are landed; AC-3 through AC-5 remain drafted here as
prose hints (not yet kernel state) pending their own TDD cycles.

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac M-0235 --title "..."`. -->

- **AC-3 candidate** — On an existing `aiwf.yaml`, `aiwf update` gates only
  hooks absent from the `hooks:` map (introduced by a newer aiwf version);
  every already-decided hook syncs silently every run — materialize +
  wire when `true`, remove + unwire when `false` — with no re-prompt.
- **AC-4 candidate** — The hooks settings writer targets the shared
  `.claude/settings.json`, preserves every unrelated key, refuses to clobber
  an existing non-aiwf entry for the same event (no-clobber, `.bak` before
  edit — mirroring `WireStatuslineSettings`), and composes correctly across
  multiple hook-event arrays (`SessionStart`, `SubagentStart`, `PreToolUse`,
  …) without duplicating entries on repeat runs.
- **AC-5 candidate** — A new "hooks" materialization category (parallel to
  the existing skills/agents/templates categories) embeds hook scripts;
  `aiwf doctor` reports drift (missing / stale / unwired / still-undecided)
  the same way it already does for rituals.

### AC-1 — aiwf.yaml hooks: schema + aiwf.example.yaml regen

`aiwf.yaml`'s schema gains `Config.Hooks map[string]Hook` (`hooks.<name>.enabled:
true|false`), a tristate `*bool` mirroring `StatusMd.AutoUpdate`: absence of the
map key, or an entry present but omitting `enabled:`, both read as undecided —
never as an implicit decline. `Config.HookDecision(name)` is the single getter
consumers use, returning `(enabled, decided bool)`. `aiwf.example.yaml`
regenerates to document the block (verified against the real `aiwf init` output,
not just the unit test), following the map-of-struct pattern `agents:` already
established and ADR-0027's generated-example convention.

Evidence: `TestHookDecision_*` (6 cases: no block, name absent, entry-present-
enabled-absent, explicit true/false, nil receiver) in
`internal/config/config_test.go`; `TestSchema_EnumeratesEveryYAMLField`,
`TestGenerateExample_ProducesValidReparseableYAML`,
`TestGenerateExample_HooksExampleItemUndecidedVerbatim`,
`TestAcceptedKeys_MembershipChecks` in `internal/config/schema_test.go`.

### AC-2 — aiwf init gates undecided hooks via TTY prompt / --enable-hook flag

`aiwf init` gates every hook in the shipped registry (`internal/skills.HookDef`
/ `ShippedHooks`, empty until M-0236 registers its first entry): a hook named
via the repeatable `--enable-hook <name>` flag is enabled without prompting
(the non-TTY consent escape hatch, mirroring `--wire-settings`); with a TTY
present it prompts `[y/N]` naming the hook and its one-line effect (default
declines); absent both, it silently declines. The gate (`cliutil.GateHookDecisions`)
is a pure decision function taking the registry as an explicit parameter, so
tests exercise it with a synthetic registry rather than depending on a real
hook existing.

The decision lands in the freshly-written `aiwf.yaml` via a new step in
`aiwf init`'s pipeline, running after `initrepo.Init` has already written the
file — not by populating `Config.Hooks` before `config.Write`'s marshal, which
would have silently dropped the full commented schema reference the moment
any hook carried a decision (`yaml.Marshal(cfg)` would no longer trim to `"{}"`,
skipping the `GenerateExample()` substitution). Instead the gate persists via
a new surgical `hooks:` block reader/writer in `internal/aiwfyaml`
(`Doc.Hooks()`/`Doc.SetHooks()`), mirroring the existing `contracts:`
whole-block splice so every other byte of the file survives untouched.

Evidence: `TestGateHookDecisions_*` (6 cases) in
`internal/cli/cliutil/hooks_test.go`; `TestHooks_*`/`TestSetHooks_*` (11 cases,
including the `hasHooks` detection, unknown-field rejection, and no-trailing-
newline append path) in `internal/aiwfyaml/hooks_test.go`; `TestRun_*` (4 cases,
including the dry-run-skips-gating and empty-registry-no-op cases) and
`TestNewCmd_EnableHookFlagParsesAndReachesRun` (the real Cobra flag-parsing
seam, not just a direct `Run` call) in `internal/cli/initcmd/initcmd_test.go`;
`TestGateAndPersistHookDecisions_MissingAiwfYamlReturnsInternal` in
`internal/cli/initcmd/gate_test.go`.

### AC-3 — aiwf update gates only newly-introduced hooks; syncs decided hooks silently

### AC-4 — Hooks settings writer: no-clobber, .bak backup, multi-event-array composition

### AC-5 — New hooks materialization category + aiwf doctor drift reporting

## Constraints

- Never write `enabled: true` for a hook that hasn't been explicitly
  consented — the TTY-prompt / explicit-flag gate runs before the first
  write for every undecided hook, no exceptions.
- Settings target is the shared `.claude/settings.json`, never
  `.settings.local.json` — hooks are unconditionally materialized once
  enabled, unlike the personal opt-in statusline (ADR-0015).
- Neither ADR-0015's nor ADR-0018's own code paths change; this ships as an
  independent, parallel mechanism scoped to hooks only.

## Design notes

- ADR-0032 locks the mechanism this milestone implements: aiwf.yaml-declared
  `hooks:` map, per-hook consent gate on first decision only, shared
  `.claude/settings.json` target, no-clobber `.bak`-guarded writer.

## Out of scope

- The concrete `worktree-materialization-check` hook's own detection logic,
  script, and policy test — the next milestone, riding on this one's
  registry.
- Migrating the existing `.claude/hooks/validate-agent-isolation.sh`
  (G-0099) into this registry — tracked as a follow-up gap, not implemented
  here.

## Dependencies

- None within this epic — independent of M-0233/M-0234.

## References

- ADR-0032 — the consent mechanism this milestone implements.
- ADR-0015 / ADR-0018 — the sibling instances of the risk-calibrated-consent
  family this decision extends.
- ADR-0027 — the generated-`aiwf.example.yaml` convention this milestone's
  schema change follows.
- G-0374 — the gap this epic closes.

## Work log

### AC-1 — aiwf.yaml hooks: schema + aiwf.example.yaml regen

Landed the `hooks:` map-of-struct schema field, `Hook` tristate struct, and
`Config.HookDecision` getter, following the existing `agents:` map-of-struct
pattern · commit 31ff89a9 · tests 9/9 new (6 `TestHookDecision_*` + 3
schema/example tests), full repo suite green, `make lint` clean, branch-
coverage audit and adversarial mutation probe (3/3 mutants caught) both
clean, real-binary `aiwf init` output manually inspected for the generated
`aiwf.example.yaml` hooks block.

### AC-2 — aiwf init gates undecided hooks via TTY prompt / --enable-hook flag

Landed `internal/skills.HookDef`/`ShippedHooks` (empty registry), the
`hooks:` block reader/writer in `internal/aiwfyaml` (mirroring the existing
`contracts:` surgical splice), `cliutil.GateHookDecisions` (reusing the
statusline's existing `promptYN`/TTY detection), and the `--enable-hook`
flag + wiring in `aiwf init` · commits 5459e35d, ecae87d9 · tests 22/22 new,
full repo suite green, `make lint` clean. Branch-coverage audit found and
closed two real gaps beyond the obvious: a `blockByteRange` error path in
the new hooks-detection code and the `gateAndPersistHookDecisions` failure-
propagation line in `Run`, both marked `//coverage:ignore` with a traced
rationale (not just asserted). Ran the actual mechanized `make
coverage-gate` (not just manual `go tool cover` reasoning) and it caught one
real miss my own analysis wrongly assumed was an accepted, unflagged
precedent — the interactive-prompt branch — fixed in a second, separate
commit rather than folded into the first. Adversarial mutation probe: 5/5
mutants caught (including the CLI-level `!dryRun` gate inversion).

## Decisions made during implementation

- (none — all decisions are pre-locked in ADR-0032 / this spec's Design notes)
