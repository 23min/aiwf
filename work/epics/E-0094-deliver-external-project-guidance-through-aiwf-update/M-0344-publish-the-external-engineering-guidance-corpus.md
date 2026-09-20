---
id: M-0344
title: Publish the external engineering guidance corpus
status: in_progress
parent: E-0094
tdd: advisory
acs:
    - id: AC-1
      title: The catalogue describes packs without aiwf language code
      status: met
    - id: AC-2
      title: The initial corpus preserves existing engineering guidance
      status: met
    - id: AC-3
      title: The source can be consumed without aiwf tooling
      status: open
---
## Goal

Establish the plain Markdown corpus and minimal catalogue that both legacy distribution and aiwf delivery can consume, without changing downstream loading behavior.

## Closes

- (none)

## Context

E-0094 selects `23min/engineering-guidance` as the canonical source. The maintained ai-dotfiles checkout contains the source material and detector. Inventory those sources before packaging; the selected repository's remote availability remains unverified.

## Acceptance criteria

### AC-1 — The catalogue describes packs without aiwf language code

Catalogue entries identify Markdown documents, applicability descriptions, and simple marker/path/extension patterns. Validate unique ids, resolvable documents, safe paths, and names that disclose opinionated tooling; no executable hooks or dependency parsing. References: external catalogue and its validation fixtures; `/workspaces/ai-dotfiles/bin/dotfiles-sync` supplies existing detection rules to inventory.

### AC-2 — The initial corpus preserves existing engineering guidance

Map all existing engineering and language guidance to the external corpus, including code-health and the material used by aiwf. Compare against a recorded source revision, allowing only documented packaging, naming, and reference changes; personal collaboration, approval, machine, and session rules remain outside the corpus. References: ai-dotfiles `guidance/`, engineering-related skill sources, and the external corpus inventory. The comparison establishes migration fidelity, not permanent prose pins.

### AC-3 — The source can be consumed without aiwf tooling

The default branch supplies the documented catalogue and Markdown, with no aiwf runtime, package installer, registry, or planning tree needed to read them. Verify the local repository first and record remote retrieval after publication approval, including command, expectation, observation, and source revision. References: the external repository's README and catalogue. Do not claim the remote exists from a local fixture.

## Constraints

Keep the catalogue declarative and small. Preserve wording. Do not introduce new languages to meet an arbitrary coverage target. The installed index ownership marker and minimum routing semantics must be specified for the next delivery, including empty selections; avoid a general protocol framework.

## Design notes

E-0094 owns the agreed delivery behavior; D-0089 records the proposed ownership boundary. Choose the exact schema and pack-to-path mapping during this milestone, before implementing consumers.

## Surfaces touched

External repository Markdown, catalogue, README, and validation fixtures; ai-dotfiles source inventory.

## Out of scope

Changing global instructions, migrating any consumer, or implementing the aiwf updater.

## Dependencies

None. Publication requires separate approval.

## References

- E-0094 — delivery scope and compatibility requirements.

---

## Local validation evidence

Observation on 2026-09-20 in the Linux x86_64 development container, with Node.js
v22.23.2 and Git 2.54.0. The corpus is in `/workspaces/engineering-guidance`;
its initial local commit is `26a55ecac2396d6ecd439a95ad4843dec65b1b25`.
The tested working copy includes the link-check correction, identified by the
SHA-256 of `__tests__/catalogue.test.mjs`: `e79e0833b3e013215d9f682a27448ef637ac1f497bd0e344a495877a869534ef`.
No remote is configured and no remote retrieval is claimed.

### Catalogue and local documents

Command, from the corpus checkout:

```sh
node --test __tests__/catalogue.test.mjs
node --check __tests__/catalogue.test.mjs
git diff --check HEAD
```

Expected: structural checks pass, test source parses, and the working diff has no
whitespace errors. Observed: four tests passed with zero failures; syntax and diff
checks exited zero. The assertions check catalogue shape and unique ids, regular
owned nonempty Markdown files, normalized paths, and delivered link destinations.
They are corpus-maintenance checks, not evidence that the future aiwf consumer
validates external input or that a host follows the installed instructions.

Temporary-copy probes reject duplicate ids, executable-hook fields, recursive
globs, missing/empty documents, parent or doubled path separators, symlink files
and directories, and undelivered link targets. The corrected link extraction
covers inline titles and reference definitions, including destinations on the
following line; delivered rubric destinations remain accepted. These are measured
examples, not a claim of complete Markdown parsing.

### Import fidelity

Run from any directory with the two local checkouts available:

```sh
python3 - /home/vscode/ai-dotfiles /workspaces/engineering-guidance <<'PYTHON'
from pathlib import Path
import hashlib,json,subprocess,sys
source,corpus=map(Path,sys.argv[1:])
record=json.loads((corpus/'IMPORT.json').read_text())
revision=record['source_commit']
paths=subprocess.check_output(['git','-C',str(source),'ls-tree','-r','--name-only',revision],text=True).splitlines()
expected={p for p in paths if p.startswith('guidance/') and p.endswith('.md') and p not in record['excluded']}
expected.add('skills/code-health/SKILL.md')
assert expected=={row['source'] for row in record['files']}
assert len(record['files'])==len(expected)
catalogue=json.loads((corpus/'catalogue.json').read_text())
assert {row['destination'] for row in record['files']}=={p for pack in catalogue['packs'] for p in pack['files']}
for row in record['files']:
    original=subprocess.check_output(['git','-C',str(source),'show',revision+':'+row['source']])
    assert hashlib.sha256(original).hexdigest()==row['source_sha256']
    output=original
    for edit in row['packaging_edits']:
        assert output.count(edit['from'].encode())==1
        output=output.replace(edit['from'].encode(),edit['to'].encode())
    assert output==(corpus/row['destination']).read_bytes(),row['destination']
print('PASS: source coverage, catalogue coverage, source hashes, and transformed byte equality')
PYTHON
```

Expected: the pinned source inventory equals the declared import inventory after
personal-content exclusion, all catalogue files are accounted for, source hashes
match Git blobs, and migrated bytes differ only by recorded packaging edits.
Observed: `PASS: source coverage, catalogue coverage, source hashes, and transformed byte equality`.
The source revision and per-file mappings are in the corpus's `IMPORT.json`.
This comparison is migration evidence, not a permanent wording freeze.

### Local portability and publication boundary

A temporary `git clone --local --no-hardlinks` of the initial corpus commit passed
its structural tests and remained clean. This establishes local Git consumption,
not the remote default-branch retrieval required by AC-3. AC-3 remains open pending
separate publication approval and a recorded remote-clone observation.

The corpus has no build step, dependency installation, or configured linter.
Node.js is needed only to run its maintenance checks. Root-document links, heading
levels, and TODO scans passed the scoped documentation review. Imported guidance
is covered by the byte comparison and pack-local link check.

Independent full-package review approves the local catalogue, source preservation,
and plain-file consumption; remote retrieval remains unverified. The reviewer
independently repeated source-fidelity, malformed-input, and link-destination
checks and confirmed the evidence above. No live TDD phase transitions were
recorded; this evidence does not claim a timestamped red/green/done progression.

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
