---
id: G-0788
title: A scope opened under a shared id goes to the wrong entity after reallocate
status: open
---
## What's missing

After `aiwf reallocate` resolves an id collision, a scope opened under the shared id is read by
that id alone, so it goes to whichever entity a reader starts from rather than the one it was
opened on. The agent's gate gives every such scope to the renumbered entity:
`loadActiveScopesForActor` (`internal/cli/cliutil/provenance.go:146`) maps each opener's
`aiwf-entity` forward with `ResolveCurrentEntityID` (`internal/cli/cliutil/scopes.go:208`,
called at `provenance.go:176`), which follows the reallocate's `aiwf-prior-entity` without
regard to which of the two entities the opener named. The readers that start from the id that
stayed put replay every commit carrying it (`LoadEntityScopes`, `scopes.go:39`), so they take the
renumbered entity's scopes as that entity's own: `aiwf show` (`internal/cli/show/scopes.go:49`)
and `aiwf authorize`'s pause, resume and end (`internal/cli/authorize/authorize.go:409`).

Measured 2026-11-24 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a
tree whose code is `main` at f068fa5a1 (`aiwf version`:
`v0.42.1-0.20261124212924-f068fa5a1f83+dirty`), in a disposable repository `a`: `aiwf init`
with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs. `b` is a
clone of `a`; each adds a milestone on `epic/E-0001-engine` before seeing the other's, so both
get `M-0002`. Ids in the block name entities in that repository.

```
b$ aiwf add milestone --epic E-0001 --title "Side work" --tdd none
b$ aiwf authorize M-0002 --to ai/side --reason side --branch milestone/M-0002-side-work     # 398f829
a$ aiwf add milestone --epic E-0001 --title "Kept work" --tdd none
a$ aiwf authorize M-0002 --to ai/kept --reason kept --branch milestone/M-0002-kept-work     # 048a22e
a$ git fetch <b> epic/E-0001-engine && git merge --no-edit FETCH_HEAD                       # 8ff2c31
a$ aiwf reallocate work/epics/E-0001-engine/M-0002-side-work.md
aiwf reallocate M-0002 -> M-0003
a$ aiwf show M-0002
M-0002 · Kept work · status: draft · tdd: none
  …
  Scopes (2):
    398f829  human/t → ai/side  state: active   opened 2026-11-24  events: 1
    048a22e  human/t → ai/kept  state: active   opened 2026-11-24  events: 1
a$ aiwf retitle M-0002 "Kept, by ai/kept" --actor ai/kept --principal human/t
aiwf retitle: provenance refused: actor "ai/kept": target M-0002 is outside the reach of every active scope (provenance-authorization-out-of-scope)    # exit 1
a$ aiwf retitle M-0003 "Side, by ai/kept" --actor ai/kept --principal human/t
aiwf retitle M-0003 -> "Side, by ai/kept"                                       # exit 0
a$ git log -1 --format=%B | grep -E "aiwf-(entity|authorized-by)"
aiwf-entity: M-0003
aiwf-authorized-by: 048a22e20241c670ef0b026d132a3d0f92472ddf
a$ aiwf check --since 8ff2c31                                                    # exit 0, no provenance finding
a$ aiwf authorize M-0002 --end --scope 398f829 --reason over
aiwf authorize M-0002 --end --scope 398f829 (ai/side)                           # exit 0
a$ aiwf retitle M-0003 "Side, by ai/side" --actor ai/side --principal human/t
aiwf retitle: provenance refused: actor "ai/side" has no active scope authorizing this act (provenance-no-active-scope)    # exit 1
```

`398f829` was opened in `b` on "Side work", which is now `M-0003`; `048a22e` was opened in `a` on
"Kept work", which kept `M-0002`.

## Why it matters

After a collision is repaired, an agent delegated on the entity that kept the id is refused on that
entity and allowed on the renumbered one, whose delegation it never received. Nothing reports it:
the verb succeeds and `aiwf check` passes, though the commit's `aiwf-authorized-by` names a scope
opened on the other entity.

A human working on the entity that kept the id acts on the other entity's delegations. `aiwf show`
lists them beside its own, and `aiwf authorize <id> --end` ends one: above, ending a scope listed
on `M-0002` withdrew `ai/side`'s delegation on `M-0003`.

## Relates to

G-0748 records the same mix-up in `aiwf history`, G-0434 in `aiwf check`'s resolution of prior
ids, and G-0745 in the references `aiwf reallocate` rewrites: each reads an id that two entities
have held as if one had. G-0776 is the renumbered entity missing the scopes opened under its own
old id; this gap is the entity that kept the id taking on the renumbered entity's scopes, and the
agent's gate sending each to the wrong one. G-0787 records the scope commits written while both
entities still carry the id, which no reader can attribute.
