---
id: G-0449
title: 'Remove untracked dead code: entity.SubKindFromID and resolver var-guards'
status: addressed
addressed_by_commit:
    - 43f45f7c
---
## What's missing

Two untracked pieces of production-unused code surfaced by a `deadcode -test` sweep (the whole-program reachability lens that `unused`, being package-scoped, does not provide):

- **`entity.SubKindFromID`** ([`internal/entity/entity.go`](../../internal/entity/entity.go) ~line 352) — an exported helper reachable only from its own `TestSubKindFromID`; no production call site. It returns a hardcoded sub-kind label that nothing reads.
- **`internal/cli/render/resolver.go` speculative guards** — a `var _ = errorf` guard (~line 795) keeping an unused `errorf` adapter alive, and a `var _ = strings.TrimSpace` guard (~line 805) keeping an otherwise-dead `strings` import alive. Both are commented "reserved for future use"; neither has a call site.

These differ from the `branch-not-found` dead surface (owned by G-0417 / D-0018, coupled to a spec-table repoint): they are unowned, uncoupled, and carry no retention decision.

## Why it matters

Speculative-reserved code with no call site is a YAGNI violation the repo's own engineering principles name explicitly ("no speculative interfaces, no 'might need it later' knobs"; "stubs and TODOs in shipped code are a smell"). The `var _ =` guards specifically defeat the `unused` linter — they exist *to suppress* the signal that would otherwise flag the reservations, so nothing mechanical catches them; only a reachability sweep does. The cost is small but real: a reader must reason about why an unused adapter and a lone import survive.

## Resolution shape

Delete `SubKindFromID` and its test; delete the two `var _ =` guards, the `errorf` function, and the now-unneeded `strings` import in `resolver.go`. `go build` + `go test ./...` + `golangci-lint` confirm nothing depended on them. One focused `wf-patch`.

## Where to fix

- [`internal/entity/entity.go`](../../internal/entity/entity.go) — remove `SubKindFromID` (+ the doc comment referencing it).
- [`internal/entity/entity_test.go`](../../internal/entity/entity_test.go) — remove `TestSubKindFromID`.
- [`internal/cli/render/resolver.go`](../../internal/cli/render/resolver.go) — remove both guards, `errorf`, and the `strings` import.

## Related

- `wf-codebase-health` / `deadcode -test` — the pass that surfaced this.
- G-0417 — the *tracked, coupled* dead-code sibling (`branch-not-found` surface); deliberately out of this gap's scope.
