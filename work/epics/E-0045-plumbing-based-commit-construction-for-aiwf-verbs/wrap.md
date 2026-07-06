# Epic wrap — E-0045

**Date:** 2026-07-06
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0045-plumbing-based-commit-construction-for-aiwf-verbs
**Merge commit:** 57ad1e3f

## Milestones delivered

- M-0186 — gitops commit primitive via temp-index and commit-tree (merged fe38dff9) — `done`
- M-0187 — Opt-in gaps inbox on a never-checked-out ref — `cancelled` (indefinitely deferred pending decision; see Handoff)

## Summary

Retired aiwf's fragile `git stash push --staged` + `git commit` verb-commit isolation (G-0275's silent half-states on staged renames) in favor of a plumbing-based commit-construction primitive: a temp-index `read-tree`/`update-index`/`write-tree`/`commit-tree`/`update-ref` pipeline that never reads or writes the live index or worktree. Every mutating verb (`verb.Apply`) is retrofitted onto it via one exported seam, `gitops.CommitVerbChange`, bundling commit construction, the best-effort post-commit hook, and post-commit index reconciliation. `commit.gpgsign` parity, the shape-validation relocation (pre-commit hook drops for verb commits; frontmatter-shape enforcement was independent of it all along, per ADR-0029), and a structural policy pinning the seam as the sole commit-construction path all shipped as part of the six-AC milestone. The second, opt-in consumer this substrate was built to support (M-0187, the gaps-inbox) is deferred rather than built now — a repo-wide measurement (`docs/initiatives/id-lifecycle.md`) found the id-collision friction it would solve is small (~3.4% collision rate, already absorbed by `E-0052` + `aiwf reallocate`) relative to the engineering cost, with a named, checkable trigger for revisiting.

## ADRs ratified

- ADR-0029 — Verb shape correctness comes from pre-write projection, not type-safe construction or a git hook.

## Decisions captured

- D-0029 — Unify applyTx rollback into a single LIFO undo journal.

## Follow-ups carried forward

- G-0374 — Worktree creation never materializes aiwf skills/agents/guidance (found during M-0186, unrelated to its own scope).
- G-0375 — Test fixtures leak ambient global git config into commit-based tests (a pre-existing, repo-wide exposure `CommitTree`/AC-4 extended to but did not create; per-key fix scoped out of this epic).
- G-0377 — `Apply`'s staged-conflict guard is coarser than a directory-move's actual writes (found during the M-0186 wrap review; narrow, not corruption in practice today).
- G-0281 — Opt-in gaps inbox: file gaps via plumbing onto a never-checked-out ref. Stays `open` (not cancelled) — it's the *design*, independent of whether M-0187 builds it now; the deferral is about timing, not the idea.

## Handoff

The shared commit-construction primitive (`gitops.CommitTree`/`ReconcilePaths`/`CommitVerbChange`) is ready for a second consumer whenever one arrives — that reusability was itself an AC (M-0186/AC-5), independently verified by a `wf-rethink` design audit at wrap time. M-0187 is deliberately left `cancelled` rather than building it now: `docs/initiatives/id-lifecycle.md`'s "Recommendation" section is the reasoning trail, with a named trigger (reallocate rate climbing meaningfully above the measured ~3-4%, or its bursts stopping being tied to identifiable concurrent-work episodes) for when to revisit — not yet a ratified decision, so a future reopening of G-0281 should start from a real ADR or `D-NNN`-shaped decision, not just this wrap note.

## Doc findings

`wf-doc-lint`, scoped to the epic's full change-set (`git diff main..HEAD`, 27 files): `ADR-0029`, `docs/initiatives/id-lifecycle.md`, `D-0029`, `epic.md`, `M-0186`'s spec, `M-0187`'s spec, `G-0375`, `G-0377`, this artefact.

- **Broken code references:** none. Every backticked symbol these docs cite resolves in the current source tree.
- **Removed-feature docs:** none spurious — `StashStaged`/`StashPop`/etc. are described only as historically retired, matching `no_stash_test.go`'s structural enforcement.
- **Orphan files:** none new beyond the two benign `discovered_in`-only gaps already noted at M-0186's own wrap (structural links, not dead docs).
- **Documentation TODOs:** none found.
- One stale-reference class was caught and fixed *before* this sweep, not by it: `docs/initiatives/id-lifecycle.md` described G-0372's two safe performance fixes as a future to-do; they had already shipped. Corrected in a dedicated commit (`90f9cd8d`) prior to this wrap.

Clean.
