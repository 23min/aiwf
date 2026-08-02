---
id: G-0514
title: skill-body-id tells CLI metavariables and non-id acronyms to become placeholders
status: open
discovered_in: M-0287
---
## What's missing

`skill-body-id` classifies any id-shaped token that is neither a real id nor the
canonical letter-N placeholder as a placeholder defect, and tells the operator to
"write the canonical `<prefix>-NNNN` placeholder". For a genuine narrow
placeholder that instruction is right. For a token that was never an id, it is
nonsense.

Measured on the shipped tree, these currently fire under that message:

- **CLI metavariables** — `M-id`, `E-id`, `C-id`, from `<M-id>` / `<E-id>` /
  `<C-id>` in command synopses. The correct "fix" is to leave them alone.
- **Distinct-placeholder conventions** — `M-PPPP`, `M-QQQQ`, `M-PPP`, `M-QQQ`,
  used where a passage needs two *different* milestone ids and the canonical form
  cannot express distinctness.
- **Deliberately-exhibited bad shapes** — `M-a`, `M-alpha`, `M-007a`,
  `M-NNN/AC-X`, in passages documenting what the rules reject.
- **Non-id acronyms** — `ADR-NEW`, `ADR-OPSPEC`.

The rule's own grammar admits the whole class: `idTokenPattern` matches
`<kind-prefix>-<word>`, so ordinary hyphenated English (`E-mail`, `C-style`,
`D-Bus`, `M-x`) classifies the same way. None of those appear in the tree today,
but nothing stops one landing.

## Why it matters

The classification is defensible — none of these is a valid id, and the shipped
surfaces are the one corpus where a stray id-shape is worth surfacing. The
*remediation* is what breaks: an operator handed `M-id` is told to write
`M-NNNN`, which would corrupt a correct command synopsis.

This lands on the sweep as concrete friction. A sweep driven by the rule's output
must resolve each of these by hand and decide, per token, whether it is debris or
a convention the rule cannot recognize — and the message actively misdirects that
judgment. The distinct-placeholder convention is the sharpest case: canonical
form genuinely cannot express "two different milestones", so the rule is asking
for something impossible.

Distinct from the narrow-placeholder work, which is about width. This is about
tokens that are not placeholders at all and are being told they are.

## Resolution shape

Options, in rough order of appetite:

- **Split the message by sub-shape.** A token whose suffix is a letter-N form of
  the wrong width gets "widen it"; anything else gets a message that names the
  ambiguity and asks the author to confirm it is not an id. Cheapest, and it
  removes the actively-wrong instruction without changing what fires.
- **Sanction the distinct-placeholder convention** so a passage needing two ids
  has a canonical way to say so, and the rule stops asking for the impossible.
- **Narrow the token grammar** so a suffix that is neither digits nor a letter-N
  run stops matching. Removes the false-positive class outright, at the cost of no
  longer catching a genuinely malformed id-shape in a shipped surface.

Decide against the sweep's actual worklist rather than in the abstract — the
sweep is where the population is enumerated, and its per-token judgments are the
evidence for which option is right.

## Where to fix

- `internal/check/skill_body_id.go` — `classifySkillToken` and
  `skillTokenMessage`, where the sub-shape distinction would live.
- `internal/check/body_prose_id.go` — `idTokenPattern`, the shared grammar that
  admits the class. Shared with `body-prose-id`, so narrowing it is not a local
  change.
- `internal/check/hint.go` — the remediation the operator actually reads.

## Related

- G-0481 — the narrow-id audit. Its per-tier counts are digit and letter-N
  populations; this class is neither, which is why the audit did not surface it.
