---
id: G-0184
title: aiwf check misses invented id-shaped tokens; no rule against fabricating ids
status: addressed
addressed_by_commit:
    - 92e4ebaf
---
## What's missing

Nothing — mechanical or advisory — stops an operator (human or LLM) from
writing an **invented, never-allocated id-shaped token** into an entity body,
frontmatter, or planning prose, and nothing catches it at the commit boundary.

Two concrete holes:

1. **Malformed id shapes are invisible to `aiwf check`.** The kernel's id
   patterns are anchored and digit-required (e.g. milestone `^M-\d{3,}$` in
   `internal/entity/entity.go`). A token like `M-a` / `M-b` (letters, no
   digits) does not match the id shape, so the reference-resolution findings
   (`refs-resolve`, `unresolved-milestone`) never treat it as an id at all — it
   slips below the radar. A well-formed-but-unallocated reference (`M-0999`)
   may or may not be caught depending on whether the body-prose reference
   scanner covers the section it appears in; the malformed shape is uncaught
   regardless.

2. **No standing convention forbids inventing ids.** CLAUDE.md documents id
   stability, collision resolution, and `reallocate` at length, but carries no
   rule of the form "never emit an id-shaped token you didn't get from the
   allocator (`aiwf add`) or read from the tree; refer to not-yet-allocated
   entities descriptively." The planning skills (`aiwfx-plan-epic`,
   `aiwfx-plan-milestones`) likewise don't state it at the moment of risk.

The triggering instance: while sequencing not-yet-allocated milestones in an
epic-planning conversation, the assistant labeled them `M-a … M-e` — a generic
"Phase A/B" pattern — despite having just read the `epic-spec.md` template
(`M-NNNN` placeholders) and the `aiwf-add` skill's explicit "Milestone ids are
global (`M-NNNN`), not epic-scoped." Had those labels been written into the epic
spec and committed, `aiwf check` would not have flagged them.

## Why it matters

The slip is one keystroke from a committed artifact: an invented label in an
epic's *Milestones* section, a fabricated `depends_on:` target, a gap body
referencing an id that was never allocated. Once committed it pollutes the
planning tree's referential integrity exactly where the kernel's value
proposition is "ids are the primary key and references stay live" — and the
one tool meant to police that (`aiwf check`) is blind to the malformed-shape
case.

It also lands squarely on the framework's load-bearing principle: *correctness
must not depend on the LLM's behavior.* The error is most likely from an LLM
operating in a fresh session, where any in-conversation "lesson" from a prior
session is gone and the same training prior (treat short id-like labels as
casual handles) re-fires. A resolution that lives only in an agent's working
memory does not survive context reset; only a tracked entity with a mechanical
chokepoint does. CLAUDE.md's own meta-rule (the merge-collision discipline
note) prescribes exactly this path: operator-discipline gaps that recur earn a
kernel-side check — "file a gap if you see this happen."

## Direction

Layered fix, ordered by how much each depends on LLM behavior (mechanical =
guarantee; advisory = odds):

- **Chokepoint (the guarantee): an `aiwf check` finding** that flags
  id-shaped tokens in entity bodies/frontmatter that do not resolve to an
  allocated entity — *including malformed shapes* like `M-a` (token matches a
  loose `<known-prefix>-<rest>` shape but not the strict allocator pattern, or
  matches the strict pattern but resolves to no entity). LLM-independent;
  catches the error at the commit boundary regardless of session or operator.
  Real design cost is false-positive scoping: the templates legitimately
  contain `M-NNNN` placeholders and the skills contain `M-NNN` examples, so the
  rule must exempt template/skill/example contexts (or scope strictly to
  committed `work/**` and `docs/adr/**` entity files, never skill/template
  text). This is the piece that actually answers "the risk remains in a new
  session."

- **Advisory backstop, every session: a CLAUDE.md standing rule** — "Never
  emit an id-shaped token (`E-`/`M-`/`G-`/`D-`/`C-`/`ADR-` + anything) you did
  not get from the allocator (`aiwf add`) or read from the tree. Refer to
  not-yet-allocated entities descriptively; the verb assigns the id." Loads on
  every session; raises odds, not a guarantee.

- **Advisory backstop, at the moment of risk: the same instruction in
  `aiwfx-plan-epic` / `aiwfx-plan-milestones`** — when sequencing milestones
  that don't exist yet, describe them; do not assign placeholder labels.

The mechanical check is the answer to the durability challenge; the advisory
lines reduce how often the check has to fire. File as the first observed
instance per CLAUDE.md's "file a gap if you see this happen" rule.
