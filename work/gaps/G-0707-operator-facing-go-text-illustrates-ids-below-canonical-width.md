---
id: G-0707
title: Operator-facing Go text illustrates ids below canonical width
status: addressed
addressed_by_commit:
    - 2e4e1e1dd
---
## What's missing

Text that aiwf prints from Go string literals illustrates ids and id placeholders
below the canonical four-digit width the allocator emits.

- **Command help examples**, in the command packages under `internal/cli/`: `E-01`
  in `add`, `promote`, `cancel`, `history` and `show`; `E-02` in `cancel`; `M-007`
  in `add ac`, `promote`, `show`, `history`, `edit-body`, `rename`, `reallocate` and
  `move`; `E-04` in `move`; `E-13` in `list`'s example and its `--parent` flag;
  `E-14` in `authorize`; `C-001` in `contract bind` and `contract unbind`; `M-001`,
  `M-002` and `M-003` in `milestone depends-on`, and `M-003` in `milestone tdd`;
  `E-21`, `E-22` and `M-077` in `retitle`.
- **Root help**, the verb list `printHelp` renders (`internal/cli/root.go`): `E-13`,
  and the composite placeholder written `M-NNN/AC-N` in the `promote` and `show`
  lines.
- **Verb messages**: the refusals `internal/verb` returns name composite ids as
  `M-NNN/AC-N` (`ac.go`, `editbody.go`) and a milestone id as `M-NNN`
  (`milestone_depends_on.go`).

The narrow placeholders in the `entity-body-empty` hints of
`internal/check/hint.go` belong to G-0538, which carries the rest of that file's
text. Narrow ids that help text uses as citations rather than illustrations, such
as `M-057`, `G-055` and `M-076` in `aiwf add`'s flag help, are G-0538's population
and not this gap's.

Measured on Linux in bash, with an `aiwf` binary built from `3f5e58aa9`, in a fresh
scratch repository after `aiwf init` and `aiwf add epic`, by running the
`aiwf add` example as printed:

    $ aiwf add milestone --epic E-01 --tdd required --title 'Bootstrap Cobra'
      -> M-0001, exit 0; frontmatter: parent: E-01
    $ aiwf check
      -> 7 findings (0 errors, 7 warnings): entity-body-empty ×5,
         milestone-draft-incomplete-acs, provenance-untrailered-scope-undefined
    $ aiwf promote E-01 active
      -> subject "aiwf promote E-01 proposed -> active"; trailer aiwf-entity: E-0001

With `Bootstrap the CLI under E-01.` written into the milestone's `## Goal`,
`aiwf check --format=json` reports `body-prose-id/narrow-width`, warning:
`M-0001 body prose contains id "E-01" below canonical width — write E-0001, the
entity it resolves to`.

Expected: no illustrative id or placeholder in operator-facing text below canonical
width. Observed: the ids and placeholders listed above.

## Why it matters

Help text and check hints are where an operator, and an assistant following the
discoverability rule, learn the shape of an id, and these teach the narrow one. An
operator who copies an example writes the narrow form into the tree: in frontmatter
the verb stores it and no check reports it, while the same spelling in prose draws a
`body-prose-id` warning, so the help teaches a form one surface flags and another
accepts.

No check reads these strings for width. The width checks judge an entity's own id,
entity body prose, the documentation corpus, and, through `skill-body-id`, the
embedded skill markdown; a Go string literal is none of those.

## Related

G-0481 set the goal that no narrow id appears as an example in any shipped or
normative surface, and audited the embedded markdown, `README.md` and
`docs/workflows.md`. G-0538 covers internal-id citations in the same
operator-facing text, regardless of width. G-0712 records the stored half: verbs
write the ids they are handed into frontmatter, `aiwf.yaml` and commit subjects at
the width typed.
