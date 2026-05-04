---
name: aiwf-add
description: Use when the user wants to add a new aiwf entity (epic, milestone, ADR, gap, decision, or contract) or an acceptance criterion under an existing milestone. Runs `aiwf add` so the id allocation, frontmatter, and commit happen mechanically.
---

# aiwf-add

The `aiwf add` verb creates a new entity (or acceptance criterion) and produces exactly one git commit. Skills like this one are advisory; the binary is authoritative.

## When to use

The user wants to record a new piece of planning state — a new epic, a milestone under an existing epic, an ADR, a discovered gap, a decision, a contract, or an acceptance criterion (AC) under an existing milestone.

## What to run

```bash
aiwf add <kind> --title "<title>" [kind-specific flags]
aiwf add ac <milestone-id> --title "<title>"
```

The six kinds and their required flags:

| Kind | Required flags | Notes |
|---|---|---|
| epic | `--title` | Allocates `E-NN`. |
| milestone | `--title`, `--epic <E-id>` | Lives under the epic's directory. |
| adr | `--title` | Allocates `ADR-NNNN` under `docs/adr/`. |
| gap | `--title` | Optional `--discovered-in <id>`. |
| decision | `--title` | Optional `--relates-to <id,id,...>`. |
| contract | `--title` | Allocates `C-NNN` and creates `work/contracts/C-NNN-<slug>/contract.md`. Optional `--linked-adr <id,id,...>` records the motivating ADRs. Pass `--validator <name> --schema <path> --fixtures <path>` together to also bind the contract in aiwf.yaml within the same commit. |
| ac | `--title`, positional milestone id | Allocates `AC-N` per-milestone (max+1 across the full `acs[]` including cancelled). Appends to the milestone's frontmatter `acs[]` and scaffolds a `### AC-N — <title>` body heading. The milestone file is rewritten in place — no separate AC file. |

## What aiwf does

1. Allocates the next free id by scanning the working tree and the configured trunk ref (default `refs/remotes/origin/main`; override via `aiwf.yaml: allocate.trunk`). For ACs the scan is the milestone's `acs[]`. The trunk read is silently skipped when the repo has no remotes configured; an explicitly-configured trunk ref that doesn't resolve is a hard error so the operator notices.
2. Writes the new entity file with proper frontmatter (`id`, `title`, `status` set to the kind's initial status). For ACs, appends to the parent milestone's `acs[]` and scaffolds the body heading.
3. When the parent milestone is `tdd: required`, an AC is seeded with `tdd_phase: red` — the only legal entry phase under the FSM. Otherwise `tdd_phase` is left absent.
4. Validates the projected tree before touching disk; if a finding would be introduced, aborts with no changes.
5. Creates one commit carrying `aiwf-verb: add`, `aiwf-entity: <id>` (composite `M-NNN/AC-N` for ACs), `aiwf-actor: <actor>` trailers. When the operator is non-human (`ai/<id>`, `bot/<id>`), the kernel additionally requires a `--principal human/<id>` flag and stamps `aiwf-principal:` on the commit. If an active authorization scope (see `aiwf-authorize`) covers the new entity's parent / references, `aiwf-on-behalf-of:` and `aiwf-authorized-by:` are added too.

## Provenance flags

| Flag | When |
|---|---|
| `--actor <role>/<id>` | Override the runtime-derived identity (default: `human/<localpart-of-git-config-user.email>`). Rarely needed by hand. |
| `--principal human/<id>` | **Required** when `--actor` is non-human (`ai/...`, `bot/...`); **forbidden** when `--actor` is `human/...`. The principal is who is accountable. |

If the LLM is invoked turn-by-turn by a human (HITL / tool mode), pass `--actor ai/<id> --principal human/<id>`. For autonomous work, the human first runs `aiwf authorize <scope-entity> --to ai/<id>`; then agent verbs work the same way and the kernel matches the scope automatically.

## Don't

- Don't hand-edit frontmatter to "skip allocation" — the id allocator + commit trailer chain is what makes history queryable.
- Don't pre-create the directory; `aiwf add` does that.
- Don't pass `--actor` unless the user asked for a specific actor; the default (derived from git config user.email) is correct.
- Don't omit `--principal` when invoking as a non-human actor — the verb refuses with a `provenance-trailer-incoherent` finding.
- Don't manually edit the milestone's `acs[]` to "fix" a gap from a cancelled AC — AC ids are position-stable. After cancelling AC-2, the next `aiwf add ac` allocates AC-3, not a recycled AC-2.
