---
id: G-0404
title: Running the stress-test harness is undocumented; make stress doesn't exist
status: addressed
addressed_by_commit:
    - 03e17c887f05939113f5cc3f20a1649099a7ea8f
---
## What's missing

`cmd/stresstest/main.go`'s own doc comment says the harness is
"built and invoked by hand (see `make stress`)" — but no `stress`
target exists in the Makefile. Confirmed directly: `grep -n stress
Makefile` returns nothing. The comment has presumably been dead since
it was written; nothing currently makes it fail loudly (it's prose,
not a reference anything mechanical checks).

Beyond that one broken pointer, there is no documentation of the
harness anywhere an operator or an LLM session would naturally look:

- No skill under `.claude/skills/` covers it (checked: no file
  matches `stress` or `stresstest`).
- No mention in `CLAUDE.md`.
- No `README.md` section.

Right now, "run a stress test" (or "run the full scenario catalog")
has no discoverable answer short of grepping `cmd/stresstest/run.go`
and `registry.go` to reconstruct the invocation and its flags by
reading source — confirmed by doing exactly that during E-0062's own
wrap.

## Why it matters

`cmd/stresstest` is deliberately dev-only tooling for this repo (per
E-0062's own scope: "Lives in its own tree, not part of the shipped
aiwf binary"), so it doesn't fall under the shipped-skill coverage
policy (`internal/policies/skill_coverage.go`) that keeps the real
`aiwf` CLI AI-discoverable — that policy has no reason to know this
binary exists. But the repo's own general principle ("if an AI must
grep source to learn a capability, it's undocumented") still applies
to repo-development tooling; it just ships in `CLAUDE.md` instead of
a skill, per the repo's own "consumer-operating vs repo-development
guidance" split. Nothing currently fills that home for this harness.

## Direction

Two independent fixes, not mutually exclusive:

- Add a real `make stress` target to the Makefile — a thin wrapper
  around `go run ./cmd/stresstest run --scenario all --repeat N` with
  a sane default repeat count — so the doc comment's own promise
  becomes true.
- Document the harness in `CLAUDE.md`'s repo-development section:
  the one-line invocation, the `--scenario`/`--repeat`/`--out`/
  `--module-root` flags, `go run ./cmd/stresstest list` to enumerate
  scenario names, and the one documented caveat (the `head-drift`
  scenario is expected-red until G-0269 ships its own fix).

## Scope

`Makefile` (new `stress` target), `CLAUDE.md` (new documentation
section, repo-development guidance — does not ship to consumers).
Small, low-risk, no production code touched.