---
id: G-037
title: Cross-branch id collisions split the audit trail; allocator is local-tree only
status: addressed
addressed_by_commit:
  - 271f514
  - b9d73d8
  - c5a98c1
  - a6e8067
  - 685f288
---

`entity.AllocateID` (`internal/entity/allocate.go:43`) walks the caller's working tree and picks `max+1`. The doc comment at lines 34-37 names this as deliberate ("cross-branch coordination is by design out of scope; collisions are caught by the ids-unique check and resolved with `aiwf reallocate`"). The design *predicted* the collision class but the resolution path was sized for a single-entity oops, not for "two parallel sessions both did real work under the same id."

**Concrete reproducer (flowtime-vnext):** main allocated `G-035 = "Promote InvariantAnalyzer warnings to CI gate"` (commit `01960ab`). Branch `milestone/M-066-edge-flow-authority-decision` independently allocated `G-035 = "Pre-aiwf v1 framework docs survived migration..."` (commit `95e4b18`). Both branches did real work under that id. Merge attempt surfaced the collision via a side-effect path (STATUS.md regen, not the entity files themselves — they have different slugs and would have merged silently into a tree with two G-035s, only caught by pre-push `aiwf check`).

**Why this is severe:**

1. *Detection is too late to be cheap.* By the time the merge happens, both branches have committed real work under the same id and both have been discussed with humans / AI under that name. Retraction cost scales with how long the branches diverged before noticing.
2. *Reallocate splits the audit trail.* Whichever branch loses, its pre-rename commits forever reference an id that, post-reallocate, means something else. `git log --grep "aiwf-entity: G-035"` returns commits from both branches under one id that now means two entities. The framework's "git log is the audit log" promise has an unsignalled hole in the multi-branch case.
3. *The path-conflict surface is a symptom, not the bug.* The two G-035 files have different slugs — git doesn't conflict on the entity files themselves. Whatever did conflict (in this case STATUS.md regen) just happened to be where the deeper id-collision became visible. Without it, the merge would silently produce a tree with two G-035s.

**Resolution:** Specified in [`design/id-allocation.md`](design/id-allocation.md) and shipped in two layers:

1. **Layer (a) — trunk-aware allocator + cross-tree `ids-unique`** (commit `271f514`). The allocator reads the working tree and the configured trunk ref (default `refs/remotes/origin/main`, overridable via `aiwf.yaml: allocate.trunk`). On a missing ref with no remotes the read is silently skipped (sandbox repos); on a missing ref *with* remotes the verb fails with a clear message — no silent fallback. `ids-unique` reads the trunk ref too, so a cross-tree collision surfaces as a normal pre-push finding (subcode `trunk-collision`). No `--against` flag, no merge simulation.
2. **Layer (b) — `prior_ids` frontmatter + reallocate trunk-ancestry tiebreaker + history chain walk** (commits `b9d73d8`, `c5a98c1`, plus integration scenario `a6e8067`). `aiwf reallocate` appends the old id to a `prior_ids: []` frontmatter list on the renumbered entity. When two entities collide on an id, the verb resolves the renumber target via `git merge-base --is-ancestor` against the trunk ref — the side already in trunk keeps the id; if ancestry can't decide, the verb refuses with a clear diagnostic and asks for a path. `aiwf history` resolves any id (current or prior) through `tree.Tree.ResolveByCurrentOrPriorID`, expands the queried id through the entity's `PriorIDs` chain, and runs one `git log` grep over `aiwf-entity:` and `aiwf-prior-entity:` for the union — pre-rename, rename, and post-rename commits arrive as one chronological timeline. The doc was reconciled to match the shipped reality (both surfaces ship; trailer is the git-log-readable source, frontmatter is the tree-readable source) in commit `685f288`.

The design deliberately omits origin-pinning, a counter-branch push-CAS allocator, surrogate identities, and an all-refs walk — each was considered and judged more code than this gap requires. The migration verb (`aiwf migrate-lineage`) for backfilling `prior_ids` from `aiwf-prior-entity:` trailers in pre-G37 reallocate history stays unbuilt-by-design: no consumer currently has the kind of legacy reallocate history that would benefit, and the verb earns its own follow-up if one surfaces.

Severity: **High**. Audit-trail integrity is one of the framework's central correctness stories ("git log is the audit log"); the multi-branch case had an unsignalled hole. The reproducer was real and recent, not theoretical.

---

<a id="g38"></a>
