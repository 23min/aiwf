---
id: D-0049
title: Red-first ordering gate is red-only; drop the green gate
status: accepted
---
M-0276 implements D-0047 point 1's red-first ordering gate. Implementation and
the wrap design review found the **green** half unsound; this decision drops it,
leaving a **red-only** gate, and refines D-0047 point 1 accordingly.

## Why the green gate fails

`--phase green` was specified to refuse unless a non-test (implementation) path
is dirty — "no implementation to have turned the test green." But that checks
whether implementation was *written*, which is orthogonal to what green *means*
(the test now passes). The two diverge for a **test-only AC** — a regression pin
or characterization test that passes against existing code with no new
implementation. Under an active gate such an AC can **never** reach `--phase
green` without `--force`; there is no honest non-force path (unlike the
new-symbol / compile-stub case, which promotes red before the stub). Test-only
ACs are common — M-0275/AC-3 and M-0276/AC-5 were both test-only.

The green gate's only unique catch — "green promoted with zero code written" —
is indistinguishable from a legitimate test-only AC, so it is unreliable as well
as false-refusing.

## The decision

The red-first ordering gate is **red-only**. `--phase red` refuses when a
non-test path is already dirty or nothing is dirty (unchanged). `--phase green`
is no longer gated. The red gate carries the entire ordering guarantee — no
implementation before the test — which is the property D-0047 set out to
enforce; the green gate added friction and an unreliable check without
strengthening it.

## Scope

- Refines D-0047 point 1 (red-first ordering enforcement) to red-only.
- M-0276/AC-4 (the green gate) is cancelled; AC-5 (force override) and AC-7
  (skill documentation) narrow to the red gate.
- The config surface (AC-1), the gitops dirty-path helper (AC-2), the red gate
  (AC-3), `--force` overridability (AC-5), and the planning-file exclusion
  (AC-6) are unaffected.
