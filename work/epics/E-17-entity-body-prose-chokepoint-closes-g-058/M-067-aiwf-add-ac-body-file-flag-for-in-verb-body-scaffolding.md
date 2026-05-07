---
id: M-067
title: aiwf add ac --body-file flag for in-verb body scaffolding
status: in_progress
parent: E-17
tdd: required
acs:
    - id: AC-1
      title: --body-file <path> populates AC body in same atomic commit
      status: met
      tdd_phase: done
    - id: AC-2
      title: Multi-AC form pairs --body-file positionally with --title
      status: open
      tdd_phase: done
    - id: AC-3
      title: Mismatched --body-file / --title counts refuse pre-allocation
      status: open
      tdd_phase: red
    - id: AC-4
      title: Body file with leading --- frontmatter refused
      status: open
      tdd_phase: red
    - id: AC-5
      title: --body-file - reads from stdin (only with single --title)
      status: open
      tdd_phase: red
    - id: AC-6
      title: Omitting --body-file leaves body empty (today's behavior)
      status: open
      tdd_phase: red
    - id: AC-7
      title: Subprocess integration test covers single, multi, stdin, refusal
      status: open
      tdd_phase: red
    - id: AC-8
      title: aiwf-add skill documents the new --body-file flag
      status: open
      tdd_phase: red
---

## Goal

Extend `aiwf add ac` with a `--body-file <path>` flag that scaffolds the AC's body section (the prose under `### AC-N — <title>`) in the same atomic commit that creates the AC. Multi-AC form pairs `--body-file` positionally with `--title` so a single invocation can populate several fully-formed ACs at once. Reduces friction so the right thing (non-empty body) is also the easy thing — without forcing operators who genuinely have nothing yet to write to provide one (the [M-066](M-066-aiwf-check-finding-entity-body-empty.md) `entity-body-empty` finding is the chokepoint; this milestone is friction reduction). Scoped to the AC verb only — generalizing the flag to other entity-creation verbs is captured as [G-066](../../gaps/G-066-aiwf-add-epic-milestone-gap-adr-decision-contract-verbs-lack-body-file-flag-for-in-verb-body-scaffolding-only-aiwf-add-ac-will-gain-it-via-m-067-leaving-the-other-six-entity-creation-verbs-reliant-on-post-add-aiwf-edit-body.md).

## Approach

The whole-entity `--body-file` already exists for `aiwf add <kind>` (per the `aiwf-add` skill's documented flags) — extend the same loader to the AC subcommand with positional pairing semantics: the Nth `--body-file` populates the body of the Nth `--title`. Mismatched counts (more `--body-file` than `--title`, or interleaved-but-unequal) refuse with a usage error before allocation.

The body file content is appended after the verb-scaffolded `### AC-N — <title>` heading; the file must not contain its own AC heading (the verb owns that). A leading `---` is refused (same rule as the whole-entity `--body-file` flag, per the existing skill docs). `-` (single dash) reads from stdin; only valid when exactly one `--title` is provided (stdin can't be split positionally).

When `--body-file` is omitted for some or all ACs, the verb's behavior is unchanged from today: bare heading scaffolded, body left empty. The `entity-body-empty` check finding from [M-066](M-066-aiwf-check-finding-entity-body-empty.md) catches the empty case at validation time.

## Acceptance criteria

### AC-1 — --body-file <path> populates AC body in same atomic commit

`aiwf add ac M-NNN --title "..." --body-file ./body.md` results in a new AC whose body section under `### AC-N — <title>` contains the contents of `body.md`. The body content lands in the same atomic commit that creates the AC — there is no interim state where the AC exists with an empty body. The verb appends the file content directly after the scaffolded heading line; no transformation, no markdown re-rendering. Code: flag bind in `cmd/aiwf/add_cmd.go`'s ac subcommand; reuse the existing whole-entity `--body-file` loader rather than writing a parallel reader.

### AC-2 — Multi-AC form pairs --body-file positionally with --title

`aiwf add ac M-NNN --title "T1" --body-file b1.md --title "T2" --body-file b2.md` creates AC-N (with body from `b1.md`) and AC-N+1 (with body from `b2.md`) in one atomic commit. The pairing is positional: the Nth `--body-file` populates the body of the Nth `--title`. Both flags repeat in any order as long as the per-flag count is equal (see AC-3 for the unequal case). Order across flags is not significant; the verb sorts both into invocation order before pairing.

### AC-3 — Mismatched --body-file / --title counts refuse pre-allocation

When the count of `--title` differs from the count of `--body-file` *and* `--body-file` is provided at all, the verb exits with code 2 (usage error) before allocating any ids, acquiring the repo lock, or touching disk. Error message names: the observed counts (`got 3 titles, 2 body files`), the pairing rule (positional, equal counts required), and notes that omitting `--body-file` entirely is also valid (per AC-6). The "all-or-nothing" rule keeps the surface predictable — partial provision (some ACs with body, others without) requires running the verb twice or using `--body-file /dev/null` for the empty-on-purpose cases.

### AC-4 — Body file with leading --- frontmatter refused

A body file whose first non-blank line is `---` exits the verb with code 2 (usage error) before allocation. Same rule as the existing whole-entity `--body-file` flag (per the `aiwf-add` skill: "the file must contain body content only — leading `---` is refused"). Rationale: the AC body is appended after a heading the verb owns, so any frontmatter in the body file would land in the wrong place and silently break document structure. Error message names the offending file path and the rule.

### AC-5 — --body-file - reads from stdin (only with single --title)

`aiwf add ac M-NNN --title "T" --body-file -` reads the body content from stdin (consistent with the existing whole-entity `--body-file` `-` shorthand). When more than one `--title` is provided, `--body-file -` exits with code 2 (usage error) — stdin is a single stream and cannot be split positionally, and silently routing it to "the first AC" would surprise the operator. Error message names the constraint and recommends using files for multi-AC invocations.

### AC-6 — Omitting --body-file leaves body empty (today's behavior)

When `--body-file` is omitted entirely (any number of `--title` flags), the verb's behavior is unchanged from today: the AC frontmatter is allocated and the bare `### AC-N — <title>` heading is scaffolded with no body content. The check finding from M-066 is the chokepoint that catches this case at validation time. Pinning today's behavior as an explicit AC keeps the multi-AC flow (e.g. quick scaffolding while the operator is still figuring out what each AC means) viable — the friction-reducing flag is opt-in, not mandatory.

### AC-7 — Subprocess integration test covers single, multi, stdin, refusal

A binary-level test (`go build -o $TMP/aiwf ./cmd/aiwf` then `exec.Command(...)` per CLAUDE.md "test the seam") in `cmd/aiwf/binary_integration_test.go` exercises five cases against a fresh tempdir milestone:

1. **Single** — `--title T1 --body-file b1.md`; AC has title T1 and body content from `b1.md`.
2. **Multi** — two `--title` and two `--body-file`; both ACs created in one commit, bodies correctly paired.
3. **Stdin** — single `--title` with `--body-file -`, body content piped via stdin.
4. **Mismatched counts** — three `--title` and two `--body-file`; exit 2, no AC created.
5. **Frontmatter rejection** — `--body-file` pointing at a file that starts with `---`; exit 2, no AC created.

Each case asserts both the exit code and the produced milestone file's frontmatter `acs[]` and body sections, byte-exact against expected fixtures.

### AC-8 — aiwf-add skill documents the new --body-file flag

The `aiwf-add` skill source (`internal/skillsembed/aiwf-add/SKILL.md` or the equivalent generation site) gains, in the `aiwf add ac` section: a description of `--body-file` (positional pairing rule, stdin shorthand, leading-`---` rejection), an example invocation showing single and multi-AC forms, and a cross-reference to M-066's `entity-body-empty` finding so the operator understands the body is not optional in the long run. Verified by the discoverability policy test (per G-021).

## Work log

### AC-1 — --body-file <path> populates AC body in same atomic commit

Bound `--body-file` (StringArrayVar) on `aiwf add ac`; threaded through to `verb.AddACBatch` via new `bodies [][]byte` parameter. Body content lands under the scaffolded `### AC-N — <title>` heading in the same atomic commit. · commit f92a2e3 · tests pass=2 fail=0 skip=0
