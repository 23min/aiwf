---
id: G-0055
title: Milestone creation does not require a TDD policy declaration
status: addressed
discovered_in: E-0014
addressed_by_commit:
    - 09fd0ccf675fb985b6a040c13b56433bcab5d5cc
---

## What's missing

`aiwf add milestone` has no `--tdd` flag. The TDD policy can only be set by hand-editing the milestone's frontmatter to add `tdd: required` (or `advisory`) — a tree-shape change that the `aiwf-add` skill explicitly forbids. When `tdd:` is absent, the kernel silently treats the milestone as `tdd: none` (per [`docs/pocv3/design/design-decisions.md`](../../docs/pocv3/design/design-decisions.md) §"Acceptance criteria and TDD"), and the audit cascade never engages:

- `aiwf add ac` does not seed `tdd_phase: red`.
- `aiwf promote --phase` is never required.
- `acs-tdd-audit` does not fire on `status: met`.
- The I3 governance render's Build/Tests tabs have no phase trailers to surface.

There is no `aiwf check` finding for "milestone has no `tdd:` declared" and no required-flag chokepoint at creation time. So the operator (human or LLM) has to *remember* to declare TDD policy out-of-band — and there is no surface that surfaces the omission.

## Why it matters

This violates CLAUDE.md's own "framework correctness must not depend on the LLM's behavior" principle for the entire TDD axis. Concrete symptom in this repo: every milestone in **E-0014** (M-0049 through M-0055) was created with no `tdd:` field and walked every AC `open -> met` directly, with no red/green/refactor/done phase events recorded in `git log`. M-0061 just reproduced the same pattern today without anyone noticing — because the kernel said nothing.

The gap is a chokepoint gap, not a TDD-policy gap. The kernel's intent ("TDD opt-in per milestone") is fine; the failure is that opt-in defaults to opt-out *without an explicit choice ever being made or recorded*.

## Possible remedies

Three layered fixes, increasing in invasiveness:

1. **Add `--tdd <required|advisory|none>` to `aiwf add milestone` and make it required at creation time.** No default value — the operator must state the policy. This is the cheapest change and the highest-leverage one: the policy decision becomes a single explicit act recorded in the same commit that creates the milestone.
2. **Add an `aiwf check` finding `milestone-tdd-undeclared` (warning by default; error under a project-policy opt-in).** Catches existing milestones and any future hand-edits that strip the field. Backstop for #1.
3. **Promote-time guard.** A milestone cannot transition `draft -> in_progress` without a `tdd:` value present. This makes the chokepoint impossible to skip even via direct file write, at the cost of extra friction on quick prototyping.

Layer #1 alone closes the immediate gap; #2 and #3 are defense-in-depth against drift.

## Opt-out: how someone could deliberately skip TDD

The kernel already has the right primitive: **`tdd: none` is the explicit opt-out.** The problem is not that opt-out is missing — it's that absence currently masquerades as opt-out. Once the `--tdd` flag is in place (remedy #1) and `aiwf.yaml` carries a `tdd.default`, opting out becomes:

```bash
aiwf add milestone --epic E-NN --tdd none --title "Documentation pass"
```

The choice is loud (visible in the verb invocation), durable (recorded in the milestone's own frontmatter), and auditable (the commit's `aiwf-verb: add` trailer ties the policy decision to a named actor on a known date). A reviewer scanning `git log --grep "aiwf-verb: add"` for `--tdd none` milestones gets the full opt-out census trivially.

Optional refinement: **`--tdd-reason "<text>"`** when `--tdd none` is chosen, written into the milestone body or a frontmatter `tdd_reason:` field. Cost is one flag; benefit is the audit trail explains *why* (e.g., "docs-only milestone", "config refresh, no behavior change"). Deferrable; the load-bearing change is making the choice mandatory.

## Project-level default and the upgrade migration

The project-level default lives in `aiwf.yaml` as `tdd.default: required | advisory | none` and **ships as `required`**. Rationale: aiwf's intended use case is engineering work where TDD is the norm; making the default `none` would silently reproduce the very gap this entry documents. Projects that prefer a different posture (docs-heavy repos, infra-as-code, config registries) override the default explicitly.

Resolution semantics — important so the migration doesn't retro-break existing planning trees:

- `aiwf add milestone` resolves `--tdd` at *creation time* and writes the resolved value into the new milestone's own frontmatter. The `aiwf.yaml` default is the fallback when `--tdd` is omitted; if both are absent (e.g., `aiwf.yaml` predates the setting), the verb refuses with a clear error pointing the operator at the new flag.
- Once the milestone is created, its frontmatter `tdd:` is the source of truth. Existing milestones with no `tdd:` field (E-0014's M-0049..M-0055, every pre-G055 milestone in this repo) stay grandfathered as `tdd: none` and are *not* retroactively re-audited. Future audits read the milestone's own field, never the project default.

This split is what lets us ship `tdd.default: required` without lighting up `acs-tdd-audit` errors across the historical tree.

### Upgrade / update behavior

`aiwf upgrade` already calls `aiwf update` as its post-install step, so `aiwf update` is the single chokepoint that has to add the setting:

- **`aiwf init`** writes `tdd.default: required` into the freshly created `aiwf.yaml` for new consumer repos. The seeded file's comment explains the value and how to override per-milestone (`--tdd none`) or repo-wide (edit the field).
- **`aiwf update`** detects an `aiwf.yaml` that lacks the `tdd.default` key and inserts it at top level with value `required`, preserving surrounding comments and key order. Idempotent: a second run is a no-op. If the key is already present (any value), `aiwf update` leaves it alone — humans who set `none` or `advisory` deliberately are not overridden.
- **Output is loud about the change.** When `aiwf update` adds the key, the human-readable output prints a clearly separated section, e.g.:

  ```
  aiwf.yaml:
    + tdd.default: required   (new in vX.Y.Z)
        New milestones now require an explicit --tdd value. The project default
        applies when --tdd is omitted. Override repo-wide by editing this field;
        opt out per-milestone with `aiwf add milestone --tdd none ...`.
  ```

  The `--format=json` envelope mirrors this in `result.changes[]` with the same fields (`path`, `key`, `value`, `note`) so CI scripts and downstream tooling can detect the migration without parsing prose.

- **No-change runs are also surfaced** (one-line acknowledgement that `tdd.default` is already set), so an operator who reruns `aiwf update` after manually editing the value sees confirmation rather than silence.

This makes the policy shift visible at exactly the moment a consumer repo absorbs it — neither buried in release notes nor announced only the next time someone runs `aiwf add milestone` and gets surprised by a refusal.
