# Skill author guide

For AI skill scaffolders (and the humans tuning them) writing skills that touch aiwf state. The goal of this doc is to make it cheap to do the right thing and visibly painful to do the wrong thing — so the scaffold's first attempt usually validates, instead of producing something `aiwf check` rejects.

If you only read one section: jump to "[The five rules](#the-five-rules)". The rest of the doc derives from those rules.

---

## Verb cheat-sheet

What a skill is allowed to call, what each verb does, and whether it produces a commit. Verbs that produce a commit also write the standard trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`) automatically — your skill never assembles those by hand.

| Verb                            | Effect                                                                              | Commits |
|--------------------------------- | ----------------------------------------------------------------------------------- | ------- |
| `aiwf status`                    | Project snapshot: in-flight epics, open decisions, gaps, recent activity.           | no      |
| `aiwf check`                     | Validate the planning tree; emit findings.                                          | no      |
| `aiwf history <id>`              | One line per event for an entity (reads `git log` via trailers).                    | no      |
| `aiwf schema [kind]`             | Frontmatter contract per kind: required/optional fields, allowed statuses, refs.    | no      |
| `aiwf template [kind]`           | Body section template per kind (the shape `aiwf add` would scaffold).               | no      |
| `aiwf whoami`                    | Resolved actor + source label.                                                      | no      |
| `aiwf add <kind> --title "..."`  | Create a new entity. Allocates id, builds frontmatter+body, writes trailers.        | yes     |
| `aiwf promote <id> <new-status>` | Advance an entity's status; checked against the kind's legal transitions.           | yes     |
| `aiwf cancel <id>`               | Promote to the kind's terminal-cancel status.                                       | yes     |
| `aiwf rename <id> <new-slug>`    | Rename the file/dir slug; id preserved; references unaffected.                      | yes     |
| `aiwf move <M-id> --epic <E-id>` | Move a milestone to a different epic; id preserved.                                 | yes     |
| `aiwf reallocate <id-or-path>`   | Renumber the entity; rewrite refs in others; both sides change atomically.          | yes     |
| `aiwf import <manifest>`         | Bulk-create entities from a YAML/JSON manifest.                                     | yes     |
| `aiwf render roadmap --write`    | Regenerate `ROADMAP.md`.                                                            | yes     |
| `aiwf contract bind / unbind`    | Manage `aiwf.yaml.contracts.entries[]`.                                             | yes     |

Skills generally **do not** call: `aiwf init`, `aiwf update`, `aiwf doctor`, `aiwf contract recipe install`, or any flag that the user wouldn't approve in-flow. Those are administrative; ask the user to run them.

For everything else, `aiwf help` is the authoritative reference.

---

## Worked example: `aiwfx-snapshot-status`

A small illustrative skill: capture the current project status as a permanent record, by creating a decision entity that points at "this is where we were on date X." Useful at major milestones, before a risky refactor, or just as a periodic check-in. The example covers all five rules in one ~30-line skill.

### What the user types

> "Take a snapshot of where we are right now."

### The skill (SKILL.md)

```markdown
---
name: aiwfx-snapshot-status
description: Use when the user asks for a snapshot, checkpoint, or "where are we now" record. Captures the current aiwf project state and creates a permanent decision entity referencing the current moment, so the snapshot is recoverable later via `aiwf history`.
---

# aiwfx-snapshot-status

Captures the current project state as a `D-NNN` decision entity, so the
moment is permanently visible in `aiwf status`, `aiwf history`, and
`git log`.

## When to invoke

The user asks for a "snapshot", "checkpoint", "where are we now",
"capture state", or describes wanting an audit-trail entry for a
moment-in-time view of the project.

## Steps

### 1. Read the current state

```bash
aiwf status
```

Show the output to the user verbatim. They are about to record this.

### 2. Confirm

Ask: *"Record this as a D-NNN decision (title: 'Status snapshot — <YYYY-MM-DD>')?"*

If the user wants a custom title or wants to add a one-line note,
collect it before proceeding.

### 3. Look up the decision schema (optional, but a good habit)

```bash
aiwf schema decision --format=json
```

This confirms `decision` accepts the fields you intend to set and lists
its allowed statuses. For an `aiwf add`, this is informational — the
verb owns the schema. For any subsequent edit you might be tempted to
do, this is required reading.

### 4. Create the decision

```bash
aiwf add decision --title "Status snapshot — 2026-05-01"
```

aiwf allocates the next D-NNN id, scaffolds the body using the decision
body template (`aiwf template decision`), commits with structured
trailers, and reports the new id.

**Do not** hand-edit the new file's frontmatter to add `date:`,
`captured_by:`, or any other field aiwf doesn't recognize. The commit
date is in `git log`; the actor is in the `aiwf-actor:` trailer; both
are recoverable via `aiwf history D-NNN`. Anything more is duplication
that drifts.

### 5. Validate

```bash
aiwf check
```

If `aiwf check` reports zero findings, the snapshot is recorded
cleanly. If it reports findings, surface them to the user — the skill
either introduced something invalid (revert the verb) or surfaced a
pre-existing issue (which the user may want to fix or defer).

## What this skill does NOT do

- Does **not** write a body for the decision describing the snapshot
  text. The git log + `aiwf history` is the snapshot. Embedding the
  status output in the decision body would be redundant and would
  drift the moment something downstream changes.
- Does **not** edit the new decision's frontmatter after `aiwf add`.
  The verb produced a coherent entity; touching it again is a smell.
- Does **not** commit anything itself. Every commit comes from `aiwf
  add`, which writes the right trailers.
```

### What the example demonstrates

Each step lines up with one of the five rules below:

- Step 1 reads via `aiwf status` — the framework's read surface.
- Step 3 looks up the contract via `aiwf schema decision` (Rule 2).
- Step 4 calls `aiwf add` — verb-first (Rule 1) — which produces the commit with proper trailers (Rule 4).
- Step 4's "do not hand-edit frontmatter" enforces Rule 3.
- Step 5 runs `aiwf check` at the exit gate (Rule 5).

The skill is ~30 lines and trivially auditable.

---

## The five rules

1. **Verb-first.** If aiwf has a verb for what you want to do, call the verb. Don't compose frontmatter by hand, don't write commit trailers by hand, don't allocate ids by hand. The verb takes care of all of it atomically. Rule of thumb: if your skill writes to a file under `work/` or `docs/adr/` directly, you are probably wrong — there is a verb for that.

2. **The schema is published.** Run `aiwf schema [kind]` (text or `--format=json`) before assuming a field is valid. Don't guess. If you find yourself wanting a field that isn't in the schema (`completed:`, `decided_by:`, `priority:`, `target_date:` — all real cases that have caused real bugs), the answer is **never** to add it to frontmatter anyway. The answer is one of: (a) put it in the body prose, (b) record it as a separate entity that links back, (c) accept that aiwf doesn't carry it.

3. **The body template is published.** Run `aiwf template [kind]` to see the section headers `aiwf add` would scaffold. If your skill creates auxiliary files for an entity (a `wrap.md` next to an epic, a tracking doc, a release notes file), match the body shape of the kind it relates to or invent a deliberate convention — but do not silently invent a fourth `## Section` header that aiwf might one day expect to be missing.

4. **Trailers come from verbs.** Every aiwf-mutating commit must carry `aiwf-verb:`, `aiwf-entity:`, and `aiwf-actor:` trailers — `aiwf history` depends on them, and `aiwf check` does not enforce them at runtime. The only safe way to get them right is to call an aiwf verb. If your skill ever needs to write a commit by hand (rare), include the three trailers explicitly; consult an existing verb's commit message for the shape.

5. **`aiwf check` at the exit gate.** Before your skill returns success, run `aiwf check`. Findings count as failure, even if no Go error was raised. The pre-push hook will catch them anyway, but catching them at skill-exit means the user sees them in-context (when they can fix them cheaply) rather than at push time (when they're already several actions deep).

---

## Boilerplate skill template

Copy and adapt. Fill the bracketed sections; delete the rest.

```markdown
---
name: aiwfx-<verb>-<noun>
description: Use when <user-intent-trigger>. <One-sentence summary of what the skill does and why it exists>.
---

# aiwfx-<verb>-<noun>

<One-paragraph framing: what success looks like for this skill, what
state it leaves behind, and what the user should expect.>

## When to invoke

<Bullet list of phrases or situations that should trigger this skill,
phrased as the user would say them. Be generous — the cost of an
unwanted invocation is low; the cost of a missed invocation is the
user repeating themselves.>

## Steps

### 1. <Read or confirm step>

<Run a read-only verb (`aiwf status`, `aiwf check`, `aiwf history`,
`aiwf schema`, `aiwf template`) to gather what's needed. Show output
to the user. Confirm intent before mutating.>

### 2. <Lookup step, if needed>

```bash
aiwf schema <kind>     # what fields the kind has
aiwf template <kind>   # what body sections aiwf scaffolds
```

<Skip this step if the skill only calls verbs that own the schema
themselves (`aiwf add`, `aiwf promote`). Include it whenever the
skill writes any text into an aiwf-adjacent file.>

### 3. <Mutate via a verb>

```bash
aiwf <verb> <args>
```

<One verb per atomic mutation. Multiple mutations = multiple commits.
Do not batch mutations into a single hand-written commit.>

### 4. Validate

```bash
aiwf check
```

<Surface findings to the user. Findings = the skill broke something
or surfaced an existing issue.>

## What this skill does NOT do

- Does not <hand-edit frontmatter / write commits / allocate ids /
  invent fields outside the schema>.
- Does not <whatever class of action is tempting but wrong for this
  skill's domain>.
```

---

## Common scaffolder mistakes

These are real classes of bugs that have shipped in real skills, each closed in either aiwf or the rituals plugin. The list is short on purpose — most other mistakes are caught by `aiwf check`.

- **Adding an unknown frontmatter field** (e.g. `completed: 2026-04-30` on an epic, or `decided_by: <person>` on a decision). aiwf's schema is strict; the field is rejected and the file fails to parse. Fixed in aiwf via [G14](gaps.md) (parse failures no longer cascade) and [G15](gaps.md) (`aiwf schema` published). Avoid by following Rule 2.
- **Writing a commit by hand without trailers.** `aiwf history` then comes up empty for that entity; the moment becomes invisible. Avoid by following Rule 4 — call a verb.
- **Renaming a file by `mv` instead of `aiwf rename`.** References to the entity break silently because nothing rewrote them. Use `aiwf rename` for slug changes; `aiwf reallocate` for id changes.
- **Putting state in frontmatter that should be in body prose.** Dates, narrative explanations, decided-by labels, completion notes — these are body content. Frontmatter is for fields aiwf reads. Verified by [G16](gaps.md) (path/id-consistency) and the schema verb.

---

## When this guide is wrong

Treat this doc as the contract aiwf publishes for skill authors. If a verb's behavior diverges from what's described here, the doc is wrong, not the verb — file a gap entry.
