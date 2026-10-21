# Writing docs

## Authoring an ADR

The FSM (`proposed → accepted | rejected`; `accepted → superseded`) and `aiwf promote` are the only surfaces that constrain ADR status — no bespoke per-ADR test pins.

## Documentation hierarchy

Every active `docs/` subtree carries one of the authority tiers below, so a reader — human or LLM — can weight a file's content correctly from its path alone, without reading every file to infer intent.

- **Normative** — current-truth, kept in lockstep with the code: `docs/adr/` (terminal-status ADRs move to `docs/adr/archive/` under the same forget-by-default convention as the Archival tier below), `docs/design/`, `docs/reference/` (generated references), `docs/architecture.md`, `docs/overview.md`, `docs/workflows.md`, `docs/skill-author-guide.md`.
- **Forward-looking** — captured ideas awaiting promotion to a real epic/gap entity: `docs/initiatives/` (an initiative whose every thread has been promoted to a tracked entity moves to `docs/initiatives/archive/` with `status: realized`, under the same forget-by-default convention as the Archival tier below — promotion, not closure of every descendant, is the bar).
- **Observational** — dated measurements of the repo, true as of the date or commit they name and worked down rather than kept in lockstep with the code: `docs/audits/`.
- **Exploratory** — synthesis/thesis/proposal genre, not kernel-binding regardless of internal rigor: `docs/explorations/` (including [`explorations/loom/loom-by-example.md`](../../docs/explorations/loom/loom-by-example.md), [`loom-light-plan.md`](../../docs/explorations/loom/loom-light-plan.md), `explorations/surveys/`), `docs/research/`, `docs/working-paper.md`.
- **Archival** — frozen historical snapshot per [the archive convention](../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md); cross-references inside it are not fixed when their targets move (same forget-by-default convention as `CHANGELOG.md`'s append-only exemption): `docs/archive/` (including `docs/archive/pocv3/`, and `docs/archive/migration/`, entry point [`from-prior-systems.md`](../../docs/archive/migration/from-prior-systems.md) — it specifies the `aiwf import` manifest contract, and that verb still ships, so a reader can reach the surface it documents).
