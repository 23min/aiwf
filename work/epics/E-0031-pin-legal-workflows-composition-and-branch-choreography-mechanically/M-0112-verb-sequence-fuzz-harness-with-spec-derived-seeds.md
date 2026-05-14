---
id: M-0112
title: Verb-sequence fuzz harness with spec-derived seeds
status: draft
parent: E-0031
depends_on:
    - M-0108
    - M-0109
tdd: required
---
## Goal

Build a `go test -fuzz`-style verb-sequence fuzz harness. Seeds derive from the spec's transition graph (per E-0031 constraint "fuzz seeds are spec-derived"). Commits a seed corpus. Branch-state fuzzing is a stretch goal; in-tree single-branch fuzz is the floor.

## Context

M-0108 ships the spec; M-0109 ships the integration harness. This milestone composes both — read the spec's transition graph, generate random walks across legal verb sequences, run each walk against the integration harness, assert tree-level invariants hold after every step. Catches "sequences we didn't hand-think-of" — the failure class hand-written integration tests miss by construction.

## Approach

Fuzz target at `internal/workflows/verb_sequence_fuzz_test.go`. The fuzz function reads the spec's transition graph (re-using M-0108's structured-markdown parser), uses fuzz input bytes to choose a starting workflow and a step count, walks N steps choosing legal verbs at each transition, asserts tree-level invariants after each. Seed corpus committed under `internal/workflows/testdata/fuzz/FuzzVerbSequence/`. Branch-state fuzz (the stretch goal) extends the harness with branch transitions; punt-to-follow-on-gap if it overruns the milestone.

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac M-0112 --title "..."`. -->

## Surfaces touched

- `internal/workflows/verb_sequence_fuzz_test.go` (new)
- `internal/workflows/testdata/fuzz/FuzzVerbSequence/` (seed corpus)
- `.github/workflows/fuzz.yml` (extend existing fuzz workflow per CLAUDE.md G44 item 1)

## Out of scope

- Branch-state fuzz over the choreography layer — stretch goal; punt to follow-on gap if it overruns.

## Dependencies

- M-0108 (spec drives seeds)
- M-0109 (harness drives execution)
