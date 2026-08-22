---
id: C-NNNN
title: <what the contract governs>
status: proposed         # aiwf contract statuses: proposed | accepted | deprecated | retired | rejected
linked_adrs: []          # optional: the ADR ids that motivate this contract
---

<!-- How to use this file. A contract is born complete: `aiwf add` refuses to create
     one whose sections are empty, so the body is written first and lands in the
     create commit, not after it.

       1. Copy this file and delete the `---` block above — it is field reference,
          and `aiwf add` writes the real frontmatter itself. Body content passed with
          its own frontmatter is refused, since the two blocks would concatenate into
          a file the loader cannot parse.
       2. Fill the two sections below.
       3. `aiwf add contract --title "<title>" --body-file <your-file>`, adding
          `--linked-adr`, `--validator`, `--schema` and `--fixtures` to bind the
          bundle in the same commit.

     `aiwf edit-body C-NNNN` is for revising the body later. Delete this comment. -->

This entity is the **registry record** for a contract, not the contract itself.
The authoritative schema, its valid and invalid fixtures, and the worked example
live on disk and are bound to this record by `aiwf add contract` or
`aiwf contract bind`; `aiwf contract verify` is what checks them. The
`aiwf-contract` skill walks the whole bundle. What belongs here is why the bundle
exists and how far it is allowed to move.

## Purpose

What the schema captures, and who is on each end of it. Name the producer and the
consumer — something with only one side is a data format, not a contract, and it
does not need this record.

Then name the failure a free-form interface would carry here: the malformed
payload that would otherwise reach production, the field two services would
disagree about, the shape that would drift once nobody was checking it.

## Stability

The evolution posture, as one of three words followed by a sentence:

- **frozen** — nothing changes; a different shape means a different contract.
- **additive-only** — new optional fields are allowed; nothing is removed,
  renamed, or narrowed.
- **breaking-allowed-with-migration** — breaking changes are permitted when they
  ship with a documented migration.

Then say what triggers a version bump, so an author changing the schema later can
tell from this record alone whether the change they intend is one this contract
permits.
