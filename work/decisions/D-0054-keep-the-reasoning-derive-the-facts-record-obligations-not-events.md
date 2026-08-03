---
id: D-0054
title: Keep the reasoning, derive the facts; record obligations, not events
status: accepted
relates_to:
    - G-0526
    - D-0053
---
## Question

Recording is what gives an assistant a memory. A fix nobody wrote down is
re-derived; reasoning nobody wrote down is re-argued, usually worse. Against
that, the apparatus-growth measurements in
[`growth.md`](../../docs/design/growth.md) are unambiguous: the corpus grows
faster than what it governs, and most of the rules that produce it can only add.

Both are true, so "record less" is not the answer and neither is "record
everything". What is the rule that decides what gets recorded, and where?

## Decision

Two rules, and a cost model that explains why they are the right two.

**Keep the reasoning. Derive the facts.** Before recording something as prose,
ask whether a check, a frontmatter field, or a git trailer could hold it
instead. If one could, prose is a second copy of a fact that already has an
owner, and it is the copy nothing re-derives. If none could — a judgment, a
rejected alternative, why the obvious approach fails — it is irreplaceable and
worth its words.

**Record obligations, not events.** An unresolved intention needs an entity,
because git cannot record what has not happened yet. A completed act does not:
the commit holds it, the trailers make it queryable, and where a defect was
pinned, the check is a better record than prose about the defect could be.

**Records live in three tiers, priced very differently.** Always-loaded prose
(`CLAUDE.md`, the guidance fragment, skill frontmatter) costs on every turn.
Entity bodies are retrieved on demand and cost per query, scaling with entity
*count* rather than word count. Commits and archived entities cost nothing until
asked for. Growth in the lower tiers is close to free; growth in the top tier is
the real tax. So the objective is not less recording but recording at the right
tier, with things falling down the tiers as they age.

## Reasoning

The first rule is a generalisation of one this project already shipped. The
reference-phrasing rule says a hand-written count is a second source of truth
for a fact the tree can move, and that the fix is to keep the reasoning and drop
the arithmetic. That is exactly right, and a count is not special: every fact a
check, a field, or a trailer could hold has the same property. The rule was
derived for numbers and never generalised, and the un-generalised cases are
where the growth is.

The second rule explains a measurement that otherwise looks like indiscipline.
Half of all closed gaps closed within a day of opening, at a median of six
commits each — work already in flight when it was filed. Those were not
obligations. They were finished acts, recorded in a container built to hold
unfinished ones, while git already held them with trailers. The wrap ritual's
own anti-pattern list names this as ledger padding; nothing enforces that, and
several mandates in the same file push the other way.

The tier model is why the conclusion is not simply "record less". Entity bodies
are the largest surface by word count and among the cheapest by per-turn cost.
`CLAUDE.md` is small by comparison and is paid on every turn — which is why it
is the one surface that has been cut back hard, and why it re-grew afterwards. A
word budget applied uniformly would compress the cheap tier and leave the
expensive one untouched.

What the two rules protect is the thing recording is actually for. An assistant
resuming cold does not need a narration of what happened; it needs the current
state, which is derivable, and the reasoning behind choices that are not
re-derivable from state. Prose spent on the first crowds out the second, and the
reader cannot tell them apart once both are on the page.

The test is self-applying, which is the reason to trust it. This decision passes
it: its content is a judgment about a tradeoff, and no check or field could
carry the argument. A same-day gap describing a fix that landed with its own
regression test fails it, because the test says the same thing and fails when it
stops being true.

## Follow-ups

Three changes follow, and their routing differs by whether the obligation has to
outlive the session. Each either narrows an existing mandate or extends an
existing ban; none adds a gate, because H3's warning applies to its own remedy.

- **A fifth disposition in the wrap ritual's review handling** — *already fixed,
  pinned by a named check* — which produces no gap. A narrowing of an existing
  mandate, the same shape as the cheap-fix test that preceded it.
- **The reasoning-versus-facts rule in the guidance fragment**, generalising the
  reference-phrasing clause from counts to facts. Ban-shaped, so it costs once.
- **The uniform prose obligation at entity creation.** Requiring a full set of
  body sections when an entity is opened prices every entity as though it will
  live for months. This one changes a kernel check's semantics for every
  consumer, so it needs its own decision record and its own scoped work — it is
  not a text edit, and this decision deliberately does not settle it.
