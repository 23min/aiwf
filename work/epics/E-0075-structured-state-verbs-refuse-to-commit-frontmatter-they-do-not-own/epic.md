---
id: E-0075
title: Structured-state verbs refuse to commit frontmatter they do not own
status: proposed
---
## Goal

Stop a mutating verb from committing frontmatter it does not own.

Every structured-state verb reads its entity from the loaded tree, re-serializes
the whole frontmatter, and commits. Nothing compares that frontmatter against HEAD
first, so a hand-edited field rides into the next verb's commit under that verb's
own `aiwf-verb:` trailer. Measured: a gap's priority set to `high` through
`aiwf set-priority`, hand-edited to `low`, then `aiwf retitle` run — the retitle
commit carried the priority change, `aiwf check` reported nothing, and
`aiwf history` still showed `set-priority high` as the last priority act.

The consequence is a commit that misattributes a structural change to a verb that
did not make it, and it defeats `provenance-untrailered-entity-commit` — the
finding whose whole purpose is catching hand-edited frontmatter. That rule holds
only while the operator commits the edit themselves; run any legitimate verb
instead and the edit arrives inside a properly trailered commit.

Addresses G-0466, which generalizes G-0463 from `edit-body --body-file` to every
structured-state verb.

## Scope

A precondition shared by every structured-state verb — `promote`, `cancel`, `move`,
`retitle`, `rename`, `reallocate`, `set-priority`, `set-area`, `milestone tdd`,
`milestone depends-on` — plus `edit-body --body-file`, which is G-0463's narrower
instance of the same defect.

Three decisions have to land before implementation, and they are the reason this is
an epic rather than a patch:

1. **Refuse or warn.** `edit-body`'s bless mode already refuses on a frontmatter
   diff and points at the structured verbs; that message exists and can be reused.
   Refusing is instructive but blocks a workflow that currently succeeds.
2. **Whether an escape hatch exists.** `--force` is sovereign and human-only. A
   precondition with no override may be correct here, since the remedy —
   `git checkout` the file, or route the edit through its owning verb — is cheap
   and always available.
3. **The never-committed entity.** An entity created and not yet committed has no
   HEAD version to compare, so the comparison has to degrade rather than fail.
   This is the case that broke AC-8's first test in M-0281, and it will break this
   one the same way if it is not decided up front.

## Out of scope

- **Adding a HEAD conjunct to the same-state NoOp guards.** Measured: the harm sits
  on the *mutating* path. Against the same dirty tree a same-state re-run converges,
  commits nothing, and leaves the hand-edit where `git status` still shows it. A
  HEAD conjunct in the NoOp guards would harden that harmless path and leave the
  harmful one laundering unchanged.
- **A check rule for laundering already in history.** Worth considering later — it
  would catch what is already committed, which a precondition cannot — but it fires
  after the push and answers a different question.
- **The same-state convergence remainder** (G-0458, G-0459, G-0460): same layer,
  different axis, its own epic.
