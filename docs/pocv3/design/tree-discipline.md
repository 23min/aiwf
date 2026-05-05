# Tree discipline

`work/` is aiwf's domain. The kernel's invariants — id uniqueness, FSM-correct status transitions, atomic commits, repo lock, structured trailers, STATUS regeneration — depend on the entity tree being mutated through verbs, not hand-written files. An LLM (or a human) writing directly under `work/` can silently break any of them.

This doc records the rule and the mechanical guarantee that backs it. See also [`design-decisions.md`](design-decisions.md) for the kernel's load-bearing properties and [`id-allocation.md`](id-allocation.md) for the allocator's contract that direct writes would bypass.

## The rule

Two operations, two answers:

1. **Tree-shape changes** — creating an entity, renaming, status transitions, id reallocation, adding ACs — go through `aiwf <verb>`. The verb owns id allocation, frontmatter shape, FSM correctness, atomicity, locking, trailers. Never write a *new* file under `work/` by hand; use `aiwf add <kind>`.
2. **Body-prose edits to existing entity files** — the markdown under the frontmatter — are allowed mechanically. There is no `aiwf edit` verb (deliberate; YAGNI). The commit will be untrailered, which is the [G24](../gaps.md#g24) audit surface; reconcile with `aiwf adopt` or commit through a verb that touches the entity for an unrelated reason. Frontmatter must not change as part of a body-prose edit.

Anything else under `work/` that is not a recognized entity file is a stray and surfaces as an `unexpected-tree-file` finding from `aiwf check`.

## Recognized paths

The loader recognizes exactly these shapes (per [`entity.PathKind`](../../../tools/internal/entity/entity.go)):

| Kind     | Path                                              |
|----------|---------------------------------------------------|
| epic     | `work/epics/E-NN-<slug>/epic.md`                  |
| milestone| `work/epics/E-NN-<slug>/M-NNN-<slug>.md`          |
| gap      | `work/gaps/G-NNN-<slug>.md`                       |
| decision | `work/decisions/D-NNN-<slug>.md`                  |
| contract | `work/contracts/C-NNN-<slug>/contract.md`         |
| ADR      | `docs/adr/ADR-NNNN-<slug>.md`                     |

Two carve-outs:

- **`docs/adr/`** is conventionally permissive — READMEs, templates, and other markdown live there alongside ADR files. The tree-discipline check is *not* applied here.
- **Contract subdirectories** (`work/contracts/C-NNN-<slug>/`) auto-exempt their auxiliary files: schema files, fixtures trees, etc. The check filters them out because the contract binding in `aiwf.yaml` legitimately references those paths.

## The mechanical check

`aiwf check` calls `check.TreeDiscipline(tree, allow, strict)` after the standard rule chain. For each path the loader recorded under `work/*` that `entity.PathKind` did not recognize:

- if the path is inside a contract directory → exempt;
- else if the path matches any glob in `aiwf.yaml: tree.allow_paths` → exempt;
- else → emit one `unexpected-tree-file` finding.

Severity is **warning** by default. Setting `aiwf.yaml: tree.strict: true` promotes it to **error**, which means the pre-push hook blocks the push. Strict mode is the right setting for any consumer where the LLM is doing real work; the warning default exists so adopting aiwf doesn't immediately turn an existing repo's incidental files into push blockers.

### Configuration

```yaml
# aiwf.yaml
tree:
  strict: true                  # promote unexpected-tree-file from warning to error
  allow_paths:                  # globs (filepath.Match), repo-relative, forward-slash
    - work/templates/*.md
    - work/scratch/**           # NOTE: ** is not supported by filepath.Match;
                                # `*` does not cross slashes
```

The PoC uses `filepath.Match` semantics (single-level `*`, single-char `?`, character classes). For deeper trees, list each glob explicitly per directory level, or accept the warning rather than allow-listing wholesale.

## Why mechanical, not advisory

The kernel principle "framework correctness must not depend on the LLM's behavior" rules out a pure-skill approach. A skill that says "don't write to `work/` directly" is the convenient version of the rule, materialized into the consumer repo via `aiwf init` for AI-discoverability. The check is the *guarantee*: if the LLM forgets the skill, the next push fails. If the LLM never reads the skill, the next push fails. The skill is for ergonomics; the check is for invariants.

## Why no `aiwf edit` verb

Body-prose edits are the one legitimate reason to touch an entity file directly. Adding an `aiwf edit <id>` verb that wraps the edit + trailer would close the last hole, but at the cost of either (a) routing every body-prose change through a verb that the LLM has to learn and the user has to invoke, or (b) building an in-process editor harness. The PoC's answer is to leave body edits as a bare `git` operation reconciled by `aiwf adopt` after the fact. Revisit if the audit warning becomes the dominant noise source.

## Where this lives

- **Doctrine** — this file. Canonical.
- **AI-discoverable** — folded into the [`aiwf-add`](../../../tools/internal/skills/embedded/aiwf-add/SKILL.md) and [`aiwf-check`](../../../tools/internal/skills/embedded/aiwf-check/SKILL.md) skills. No new skill — see "guard against skill sprawl" in the same conversation that produced this doc.
- **Mechanical** — `tools/internal/check/tree_discipline.go`, called from `runCheck` at [`tools/cmd/aiwf/main.go`](../../../tools/cmd/aiwf/main.go).
- **Configuration** — `tree.allow_paths` and `tree.strict` in `aiwf.yaml`, parsed by [`tools/internal/config/config.go`](../../../tools/internal/config/config.go).
- **Consumer's `CLAUDE.md`** — *not* aiwf's responsibility. The kernel ships the embedded skills (gitignored, refreshed on `aiwf update`) and the check; the consumer's hand-written `CLAUDE.md` is theirs alone.
