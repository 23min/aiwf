---
id: G-0209
title: Ritual step ordering is advisory only; no kernel enforcement
status: open
discovered_in: M-0158
---
M-0104 and M-0105 deliver structural updates to
[`aiwfx-start-epic`](../../internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md)
and
[`aiwfx-start-milestone`](../../internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md)
that prescribe ADR-0010's branch model: sovereign acts on `main`
(or parent epic branch), then cut the workflow branch.

The milestones' tests pin SKILL.md doc-shape (heading order, key
markers). **They do NOT pin that an AI assistant actually follows
the sequence.**

## What's enforced vs not

| Step | Enforced by | Failure mode |
|---|---|---|
| `aiwf promote E-NN active` requires `human/` actor | M-0095 kernel rule | Catches AI attempting to promote |
| Promote happens before branch cut | **NOTHING** | AI cuts branch first, then promotes — kernel doesn't notice |
| `aiwf authorize` requires ritual branch context | M-0103 preflight | Catches AI dispatching without context |
| Authorize happens on parent branch (not cut branch yet) | **NOTHING** | AI cuts then authorizes — kernel doesn't notice |
| Branch cut happens AFTER promote+authorize | **NOTHING** | Same as above |

The kernel's existing chokepoints (M-0095, M-0103, M-0106's
post-hoc detection) catch SOME deviations from the SKILL.md
flow but not the ordering itself.

## Concrete failure mode

An AI assistant misreads the SKILL.md (or operates from a stale
version) and cuts `epic/E-NN-foo` first, THEN promotes E-NN active
ON the epic branch, THEN authorizes on the epic branch.

The promote commit lands on the epic branch instead of main.
The authorize commit lands on the epic branch (passes M-0103
because the operator is on a ritual branch).

`aiwf history` shows: `aiwf promote E-NN active` and
`aiwf authorize E-NN --to ai/X` on `epic/E-NN-foo`. **No kernel
check fires.** The operator only notices when reading the history
post-hoc.

## What's needed

Either:

1. **A new kernel finding** (`promote-on-wrong-branch` or similar)
   that detects sovereign-promote commits NOT on `main` (or the
   configured parent) and warns.
2. **A pre-promote precondition check** that refuses
   `aiwf promote E-NN active` from a ritual-shape current branch.
3. **Document the limitation explicitly** in the SKILL.md
   constraints section + the epic body's "what's NOT enforced".

Option 1 is the strongest; option 3 is the smallest fix. Option 2
might cause friction for legitimate scenarios (e.g., re-activating
an epic from its branch).

## Why parked

The M-0158 honest-scope audit surfaced this. The SKILL.md is
advisory; the kernel doesn't enforce the ritual step ordering.
Address as part of the real-world hardening milestone — at minimum,
document the gap; ideally, add a kernel check.

## Related: the epic's framing

The E-0030 epic positions this work as "branch model chokepoint",
implying kernel enforcement. The SKILL.md updates are framed as
part of the chokepoint. The kernel's actual enforcement is narrower
than the framing suggests. The honest framing: M-0102+M-0103+M-0106
build kernel chokepoints; M-0104+M-0105 deliver SKILL doc updates
that AN AI ASSISTANT MAY FOLLOW.
