---
id: M-067
title: aiwf add ac --body-file flag for in-verb body scaffolding
status: draft
parent: E-17
tdd: required
---

## Goal

Extend `aiwf add ac` with a `--body-file <path>` flag that scaffolds the AC's body section (the prose under `### AC-N — <title>`) in the same atomic commit that creates the AC. Multi-AC form pairs `--body-file` positionally with `--title` so a single invocation can populate several fully-formed ACs at once. Reduces friction so the right thing (non-empty body) is also the easy thing — without forcing operators who genuinely have nothing yet to write to provide one (the [M-066](M-066-aiwf-check-finding-acs-body-empty.md) check finding is the chokepoint; this milestone is friction reduction).

## Approach

The whole-entity `--body-file` already exists for `aiwf add <kind>` (per the `aiwf-add` skill's documented flags) — extend the same loader to the AC subcommand with positional pairing semantics: the Nth `--body-file` populates the body of the Nth `--title`. Mismatched counts (more `--body-file` than `--title`, or interleaved-but-unequal) refuse with a usage error before allocation.

The body file content is appended after the verb-scaffolded `### AC-N — <title>` heading; the file must not contain its own AC heading (the verb owns that). A leading `---` is refused (same rule as the whole-entity `--body-file` flag, per the existing skill docs). `-` (single dash) reads from stdin; only valid when exactly one `--title` is provided (stdin can't be split positionally).

When `--body-file` is omitted for some or all ACs, the verb's behavior is unchanged from today: bare heading scaffolded, body left empty. The check finding from [M-066](M-066-aiwf-check-finding-acs-body-empty.md) catches the empty case at validation time.

## Acceptance criteria
