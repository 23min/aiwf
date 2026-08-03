---
id: G-0480
title: No check rule detects frontmatter laundering already in history
status: wontfix
priority: medium
discovered_in: E-0075
---
## What's missing

Every guard under consideration for the frontmatter write-scope problem is a
*precondition*: it refuses or reports before a verb commits. Nothing detects
laundering that is already committed.

Three routes produce history a precondition cannot reach:

- **Commits that predate the guard.** Whatever the tree already carries stays
  carried, unreported.
- **Merge commits.** The untrailered-entity-commit audit skips a multi-parent
  commit whose subject does not look like a squash merge, on the reasoning that
  the feature branch's own commits carry trailers and were audited by that
  branch's pre-push hook. A verb cannot produce a merge commit — the
  in-progress-operation guard refuses while a merge, cherry-pick, revert or
  rebase is running — but a human resolving a conflict can, and that commit is
  unaudited.
- **The nested-path vector, if the precondition ends up entity-scoped.** A guard
  comparing only the verb's named entity misses a nested milestone's frontmatter
  riding along inside a parent epic's directory move. In that branch this rule is
  the only backstop that exists.

The existing provenance audit cannot fill the role. It skips any commit carrying
a non-empty `aiwf-verb:` trailer, and it derives touched entities from changed
*paths* rather than from frontmatter, so which fields moved is invisible to it.
That is precisely the shape laundering takes: a correctly-trailered commit whose
frontmatter diff belongs to no verb.

## Why it matters

The damage laundering does is to the record, and the record is what
`aiwf history` reads. A field changed under another verb's trailer is attributed
to an act that did not make it, so a reader concludes the wrong current value and
the wrong last actor. A precondition stops new instances; it never tells an
operator that the tree they already hold is misattributed, or which entity to
look at.

The stakes are not uniform across fields. A laundered `priority` misleads a
reader. A laundered `status` on a path-changing route evades
`fsm-history-consistent/illegal-transition`, whose walker deliberately skips a
commit that both renames and changes status. A laundered `tdd` decides whether
`acs-tdd-audit` fires at all. Detection after the fact is what separates "fixed
going forward" from "fixed".

## Scope

A check rule that walks history and reports a commit whose frontmatter diff is
not accounted for by its own `aiwf-verb:` trailer.

Four things are open at filing:

- **Severity.** An error blocks a push over history that is already written,
  where the remedies are acknowledgment or rewriting. A warning is carryable but
  normalizes easily into background noise.
- **Where the walk starts.** A full-history walk on every run has a cost the
  pre-push hook pays; a `--since`-scoped walk is cheap but silent by default,
  which is the shape that already makes the provenance audit's scope-gating easy
  to miss.
- **How a verb's legitimate frontmatter diff is characterized.** The rule has to
  know which fields each verb owns in order to tell a laundered field from an
  owned one. No such mapping exists today.
- **Acknowledgment.** Laundering already in history may be legitimate to carry
  forward, which points at the existing `aiwf acknowledge illegal` shape rather
  than a new one.

## Resolution options

1. **A history-walking rule keyed on verb-owned fields.** Complete, and the only
   option covering the merge route. The verb-to-owned-fields mapping is the bulk
   of the work.
2. **A narrower rule keyed on the fields whose laundering defeats another
   check** — `status` and `tdd`. Cheaper, and it covers the cases that silence a
   blocking rule, but it leaves `priority`-class misattribution unreported.
3. **Nothing, relying on the precondition alone.** Defensible only if the
   precondition lands committed-path-scoped. Under entity scope it leaves the
   nested vector with no detector at all, and it covers no merge commit either
   way.

Option 1 is the lean if the verb-to-owned-fields mapping gets built for the
precondition anyway, since a committed-path-scoped guard needs something close to
it. If the precondition ships without such a mapping, option 2 is the
proportionate first step and option 1 stays open.
