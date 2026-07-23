package check

// hintTable maps a finding's Code+Subcode to a one-line "what to do
// about it" hint. Render layers append `— hint: <hint>` to the
// human-readable line; JSON consumers see the same string in the
// `hint` field.
//
// Keep hints actionable and verb-led ("run X", "set Y", "remove Z").
// Avoid restating the failure — the message already does that.
//
// Every hint names the exact remediation command — a backtick-delimited
// `aiwf ...` or `git ...` invocation (with placeholder ids) — so an LLM
// or operator reading the finding has the fix in hand and never has to
// grep source or guess a verb. Config-only remediations that no single
// verb performs still name `aiwf check` / `aiwf contract verify` as the
// re-validation step. The `finding-hints-name-command` chokepoint
// (internal/check/hint_test.go) reddens on any hint that names none.
var hintTable = map[string]string{
	"load-error": "repair the YAML frontmatter (delimited by `---`) by hand and re-run `aiwf check`, or `git rm <path>` if the file is not an aiwf entity",
	"ids-unique": "run `aiwf reallocate <path>` on one of the duplicates to renumber it",
	// G-0379: the bare "ids-unique" hint above is wrong for this subcode
	// in the common case — a rename landed on trunk (`aiwf retitle`/`aiwf
	// rename`) after the branch forked, invisible to the branch's stale
	// copy (G-0378). Reallocating there doesn't resolve anything; it
	// renames the branch's otherwise-correct copy, producing a genuine
	// duplicate that then needs a manual revert. Lead with the stale-
	// branch check before recommending reallocate, which remains correct
	// only for a genuinely unrelated same-id entity (ADR-0031).
	"ids-unique/trunk-collision":        "first check whether the trunk-side path is a rename of this branch's entity since the branch forked, e.g. `git log --diff-filter=R --follow origin/main -- <trunk-path>` (put the trunk ref before the pathspec, or the check runs against HEAD and misses the rename), or just attempt `git merge origin/main` (or your configured trunk ref) and see whether it resolves the divergence cleanly; if so, merge/rebase trunk into the branch instead of reallocating. Only run `aiwf reallocate <path>` when the two paths are confirmed to be genuinely unrelated entities, not a rename",
	"case-paths":                        "rename one of the colliding paths via `aiwf rename` so they differ in more than just case (case-insensitive filesystems treat them as the same dir)",
	"frontmatter-shape":                 "add the missing frontmatter field by hand and re-run `aiwf check`; if the id itself is malformed, renumber via `aiwf reallocate <path>` so it emits at canonical width",
	"id-path-consistent":                "renumber via `aiwf reallocate <path>` (rewrites both sides + updates references), rename the slug via `aiwf rename` if only the slug drifted, or correct the side that's wrong by hand if you're certain which",
	"status-valid":                      "correct the `status:` field in the frontmatter to one of the kind's allowed states by hand, then re-run `aiwf check`; `aiwf promote` can't move an entity whose current status is unrecognized (it reads that status to compute the transition and refuses)",
	"priority-valid":                    "correct the `priority:` value by hand to one of urgent, high, medium, low — or remove it — and re-run `aiwf check`",
	"priority-not-applicable":           "remove the `priority:` field by hand (only gap and decision carry their own priority) and re-run `aiwf check`",
	"refs-resolve/unresolved":           "confirm the target with `aiwf show <target-id>`, then correct the spelling in the referencing frontmatter (or remove the reference) and re-run `aiwf check`",
	"refs-resolve/wrong-kind":           "replace the reference with an id of the expected kind — list candidates via `aiwf list --kind <kind>` — then re-run `aiwf check`",
	"refs-resolve/unresolved-milestone": "the composite id's parent milestone does not exist; verify with `aiwf show M-NNNN`, or create it via `aiwf add milestone --epic E-NNNN --tdd <policy> --title \"...\"`",
	"refs-resolve/unresolved-ac":        "the parent milestone exists but has no AC with that id; add it via `aiwf add ac <milestone-id> --title \"...\"`, or correct the reference and re-run `aiwf check`",
	// M-0259/AC-2: the target is real but lives only on another local
	// branch or remote-tracking ref — non-blocking, per ADR-0030. No
	// fix needed; it resolves on its own once the source branch merges,
	// or escalates to unresolved if that branch is deleted/abandoned.
	"refs-resolve/cross-branch-pending": "no action needed — the target exists on another local or remote-tracking branch and will resolve locally once that branch merges; run `git fetch` if the branch is a teammate's not-yet-fetched remote work",
	// M-0259/AC-3 (D-0036): the id exists on more than one ref with
	// DIFFERENT content. Non-blocking, like cross-branch-pending —
	// divergence is ambiguous between an in-flight edit on one of the
	// branches (the common case, especially across worktrees of this
	// repo, which share local branch refs) and a genuine duplicate-mint
	// collision; the latter is still caught, just later, by the
	// blocking ids-unique/trunk-collision check once both copies land
	// in a shared tree.
	"refs-resolve/cross-branch-collision": "compare content at each ref (e.g. `git show <ref>:<path>`) — if it's an in-flight edit on an unmerged branch, no action needed, it resolves on merge; if the two refs genuinely allocated different entities under the same id, reconcile by hand (rename one side via `aiwf reallocate`, or merge and resolve the conflict)",

	// G-0184: body-prose-id chokepoint. The check scans entity body
	// prose (frontmatter is covered by refs-resolve) for id-shaped
	// tokens that are either malformed or unallocated. The hints point
	// to the canonical fix per subcode; both shapes are silenced by
	// wrapping the token in backticks when the prose is discussing id
	// syntax rather than referencing a real entity. The bare-code hint
	// is the catch-all when the subcode lookup misses.
	"body-prose-id":                      "the body prose contains an id-shaped token that is either malformed or unallocated; fix it with `aiwf edit-body <id>` — reference real entities by their canonical id (e.g. `M-0001`) and wrap hypothetical or syntax-discussion tokens in backticks",
	"body-prose-id/malformed-shape":      "the body prose contains an id-shaped token that is not a valid id (letter suffix, uppercase placeholder, or narrow-numeric form); fix it with `aiwf edit-body <id>`. If it references a real entity, use the canonical id (`M-0001`, not `M-1` or `M-NNNN`); if it is discussing id syntax, wrap it in backticks. Conversational sequential labels (`M-1`, `M-2`) belong in chat, not committed prose — replace with the allocator-assigned canonical id once the entity exists.",
	"body-prose-id/unresolved":           "the body prose references a well-formed id that resolves to no entity; fix it with `aiwf edit-body <id>` — check the spelling, or wrap in backticks if the prose is discussing a hypothetical id shape rather than a real reference",
	"body-prose-id/unresolved-milestone": "the composite id's parent milestone does not exist; fix the prose with `aiwf edit-body <id>` — check the spelling or remove the reference",
	"body-prose-id/unresolved-ac":        "the parent milestone exists but has no AC with that id; fix the prose with `aiwf edit-body <id>` — check the AC number, or add the AC entry via `aiwf add ac <milestone-id> --title \"...\"`",
	// M-0259/AC-2: the mirror of refs-resolve/cross-branch-pending for
	// prose tokens — non-blocking, per ADR-0030.
	"body-prose-id/cross-branch-pending": "no action needed — the id exists on another local or remote-tracking branch and will resolve locally once that branch merges; run `git fetch` if the branch is a teammate's not-yet-fetched remote work",
	// M-0259/AC-3 (D-0036): the mirror of refs-resolve/cross-branch-collision
	// for prose tokens.
	"body-prose-id/cross-branch-collision": "compare content at each ref (e.g. `git show <ref>:<path>`) — if it's an in-flight edit on an unmerged branch, no action needed, it resolves on merge; if the two refs genuinely allocated different entities under the same id, reconcile by hand (rename one side via `aiwf reallocate`, or merge and resolve the conflict)",
	// G-0299 / M-0227: skill-body-id chokepoint. Shipped consumer surfaces
	// (every *.md under embedded{,-rituals,-guidance}/ plus statusline
	// comments) must cite no real entity id (the mirror image of
	// body-prose-id). The fix is a canonical placeholder or a design/ADR
	// doc-link, not a real id.
	"skill-body-id":              "a shipped surface cites a real entity id, which is meaningless in a consumer repo and rots as the entity changes; replace it with a canonical `<prefix>-NNNN` placeholder or a shape-description (or cite a design/ADR doc as a markdown link, so the id rides in the destination while the visible text stays descriptive), then re-run `aiwf check`",
	"no-cycles/depends_on":       "break the cycle by resetting one milestone's dependencies via `aiwf milestone depends-on <milestone-id> --on <remaining-ids>` (or `--clear` to empty it)",
	"depends-on-cancelled":       "retarget the dependency via `aiwf milestone depends-on <milestone-id> --on <remaining-ids>` (or `--clear` to empty it), or cancel the dependent milestone too if it's no longer needed",
	"no-cycles/supersedes":       "break the loop in the supersedes/superseded_by chain — inspect it with `aiwf history ADR-NNNN`, correct the errant `supersedes:`/`superseded_by:` frontmatter entry by hand, then re-run `aiwf check`",
	"titles-nonempty":            "set a non-empty title via `aiwf retitle <id> \"...\"`",
	"adr-supersession-mutual":    "record the supersession through the verb so both sides stay in sync: `aiwf promote ADR-NNNN superseded --superseded-by ADR-MMMM` writes the reciprocal `supersedes:` automatically",
	"gap-addressed-has-resolver": "name the resolver atomically with the status change via `aiwf promote <id> addressed --by <milestone-id>` (or `--by-commit <sha>` when a specific commit closed it), or step back with `aiwf promote <id> open` / `aiwf promote <id> wontfix`",

	// G-0155: misset core.worktree silently redirects every git op
	// against the wrong worktree. The hint points at the precise
	// remediation (unset the override) — that's the right move in the
	// overwhelmingly common case; bare-repo workflows that intentionally
	// set core.worktree are rare and the operator who set it will know
	// to disregard the hint.
	"git-config-core-worktree-misset": "run `git config --local --unset core.worktree` from the repo root (only override if your workflow specifically requires it, e.g. bare repos — see gap G-0155)",

	// M-0094: start-epic preflight signal per G-0063. The aiwfx-start-epic
	// skill consumes this finding to surface "no work queued" before
	// activation; post-activation, drafting the next milestone (or wrapping
	// the epic) clears it.
	"epic-active-no-drafted-milestones": "draft the next milestone with `aiwf add milestone --epic E-NN --tdd <policy> --title \"...\"`, or wrap the epic if all planned work is in flight or done — the rule is the start-epic preflight signal from G-0063",

	// M-083 AC-1: tree mid-migration warning. Fires only on the
	// mixed-active-tree case; uniform-narrow and uniform-canonical
	// stay silent per ADR-0008's "Drift control" subsection.
	"entity-id-narrow-width": "the active tree mixes narrow and canonical id widths; run `aiwf rewidth --apply` to complete the canonical-width migration (no commit until you re-invoke with `--apply`)",

	// M-0086: ADR-0004 §"Reversal" forbids relocation as the
	// remediation. The remediation is to revert the hand-edit, not
	// to move the file out of archive. There is no reverse-archive verb.
	"archived-entity-not-terminal": "revert the hand-edit so the status returns to a terminal value (`git checkout -- <path>` if the change is uncommitted), then re-run `aiwf check`; if the entity genuinely needs revisiting, open a new one that references it via `aiwf add <kind> --title \"...\"` (ADR-0004 §Reversal)",
	// M-0086: terminal-entity-not-archived is the pending-sweep
	// finding. Advisory by default; the M-0088 threshold knob will
	// promote it to blocking past N for opted-in consumers.
	"terminal-entity-not-archived": "run `aiwf archive --dry-run` to preview the sweep, then `aiwf archive --apply` to commit the move; advisory until `archive.sweep_threshold` is set in aiwf.yaml",
	// M-0086: archive-sweep-pending is the per-tree aggregate.
	// Hidden when the count is zero. The hint matches its leaf
	// counterpart so an operator reading either reaches the same
	// remediation.
	"archive-sweep-pending": "run `aiwf archive --dry-run` to preview the sweep, then `aiwf archive --apply` to commit; the aggregate's count comes from the per-file `terminal-entity-not-archived` findings",

	// G-0393: standing backstop for the epic-terminal-promote guard.
	// The epic itself is already terminal (nothing to do there); the
	// remediation is disposing each listed child milestone. Two
	// distinct root causes reach this finding (G-0398): a genuine
	// bypass of the promote/cancel guards (hand-edit, pre-guard
	// binary), OR `aiwf add milestone`/`aiwf import` creating a fresh
	// milestone under an epic that was already terminal — that path
	// has no dedicated guard yet, so this finding is standing in for
	// one. The hint below doesn't guess which; the remediation is the
	// same either way.
	"epic-terminal-non-terminal-children": "bring each listed child milestone to a terminal status via `aiwf promote <milestone-id> done` or `aiwf cancel <milestone-id>` — the epic is already terminal, so no epic-side action is needed. If this fired right after `aiwf promote`/`aiwf cancel` on the epic itself, that shouldn't be possible — both already refuse to reach this state (G-0393/D-0003) — so re-check for a hand-edit or a pre-guard binary. If it fired right after `aiwf add milestone`/`aiwf import`, the epic was simply already terminal before the milestone was created; there is no dedicated guard against that yet (G-0398)",

	"acs-shape/id":                         "fix the AC's id to match `AC-N` (position+1; cancelled entries still count) by correcting the `acs:` frontmatter, then re-run `aiwf check`",
	"acs-shape/title":                      "set a non-empty AC title via `aiwf retitle <milestone-id>/AC-N \"...\"`",
	"acs-shape/status":                     "correct the AC's status in the `acs:` frontmatter to a legal value by hand, then re-run `aiwf check`; `aiwf promote <milestone-id>/AC-N <status>` can't transition an AC whose current status is unrecognized",
	"acs-shape/tdd-phase":                  "the AC's tdd_phase value isn't in the allowed set; advance it via `aiwf promote <milestone-id>/AC-N --phase <red|green|refactor|done>`, or clear it by hand — absence is always legal, only an invalid value fires this",
	"acs-shape/tdd-policy":                 "declare the milestone's TDD policy at creation via `aiwf add milestone --tdd <required|advisory|none>`; for an existing milestone, set `tdd:` in the frontmatter by hand and re-run `aiwf check` (there is no post-create --tdd verb)",
	"acs-body-coherence/missing-heading":   "add a `### AC-<N> — <title>` heading in the milestone body via `aiwf edit-body <milestone-id>`, or drop the AC from the `acs:` frontmatter",
	"acs-body-coherence/orphan-heading":    "register the AC in the milestone's `acs:` frontmatter via `aiwf add ac <milestone-id> --title \"...\"`, or remove the stray body heading via `aiwf edit-body <milestone-id>`",
	"acs-body-coherence/duplicate-heading": "delete the extra `### AC-<N>` heading via `aiwf edit-body <milestone-id>`; keep exactly one per AC",
	"acs-empty-body":                       "write prose under the named AC's `### AC-N` heading via `aiwf edit-body <milestone-id>`; a title-only stub is not a real contract for that criterion once the milestone is in_progress or done",
	"acs-tdd-audit":                        "advance the AC's tdd_phase to `done` via `aiwf promote <id>/AC-N --phase done`, or relax the milestone's tdd: setting",
	"acs-tdd-tests-missing":                "re-run the TDD cycle through `aiwf promote <id>/AC-N --phase ... --tests \"pass=N fail=N skip=N\"`, or set `tdd.require_test_metrics: false` in aiwf.yaml to silence the warning",
	"acs-title-prose":                      "shorten the AC title via `aiwf retitle <milestone-id>/AC-N \"...\"` and move the detail prose into the body under `### AC-N` via `aiwf edit-body <milestone-id>`; titles render as one big heading",
	"milestone-done-incomplete-acs":        "promote each open AC via `aiwf promote <milestone-id>/AC-N <met|deferred|cancelled>`, or override with `aiwf promote <milestone-id> done --force --reason \"...\"` (the standing check still surfaces this)",
	"milestone-done-zero-acs":              "advisory only — a done milestone with no acceptance criteria is a legitimate end state; add one via `aiwf add ac <milestone-id> --title \"...\"` first if this was unintentional",
	"milestone-cancelled-incomplete-acs":   "promote each open AC via `aiwf promote <milestone-id>/AC-N <met|deferred|cancelled>`; `aiwf promote`/`aiwf cancel` already refuse this transition (with no --force override) through normal use, so this state means the verb layer was bypassed — a hand-edit is the usual cause",
	"milestone-draft-incomplete-acs":       "add the milestone's acceptance criteria at plan time via `aiwf add ac <milestone-id> --title \"...\"` (and fill each AC body); a warning only — a draft milestone is legitimately mid-planning, so no action is needed if you are still scoping it",

	// M-066 entity-body-empty: each kind's load-bearing body sections
	// must contain non-empty prose. AC bodies have a verb-side shortcut
	// (`aiwf add ac --body-file` from M-067); other kinds rely on
	// `aiwf edit-body` until the analogous flag for those verbs lands
	// (G-066). The bare-code hint is the catch-all when the subcode
	// (kind tag) doesn't have its own entry yet.
	"entity-body-empty":           "write prose for the named body section via `aiwf edit-body <id>`; for ACs, `aiwf add ac --body-file` (M-067) can scaffold the body during create",
	"entity-body-empty/ac":        "fill the AC body under `### AC-N` via `aiwf edit-body M-NNN`; on create, `aiwf add ac --body-file` (M-067) scaffolds the body in the same atomic commit",
	"entity-body-empty/epic":      "write prose for the named section in the epic body via `aiwf edit-body E-NN`; per-section detail belongs in the body, not the title",
	"entity-body-empty/milestone": "write prose for the named section in the milestone body via `aiwf edit-body M-NNN`; the per-AC detail goes under each `### AC-N` heading",
	"entity-body-empty/gap":       "write prose for the named section in the gap body via `aiwf edit-body G-NNN`; explain what's missing and why it matters so future readers understand the friction",
	"entity-body-empty/adr":       "write prose for the named section in the ADR body via `aiwf edit-body ADR-NNNN`; Context/Decision/Consequences are the load-bearing record",
	"entity-body-empty/decision":  "write prose for the named section in the decision body via `aiwf edit-body D-NNN`; Question/Decision/Reasoning are the load-bearing record",
	"entity-body-empty/contract":  "write prose for the named section in the contract body via `aiwf edit-body C-NNN`; Purpose/Stability are the load-bearing record",

	// G-0268 milestone-tdd-undeclared: the milestone has no tdd: policy
	// and absent is silently treated as tdd: none. New milestones get
	// the policy from the required `--tdd` flag at create time; an
	// existing/grandfathered milestone is fixed by a frontmatter edit
	// (there is no post-create --tdd verb — tdd is creation-set).
	"milestone-tdd-undeclared": "declare the milestone's TDD policy — create with `aiwf add milestone --tdd <required|advisory|none>`, or for an existing milestone add `tdd: required` (or advisory/none) to the frontmatter; absent `tdd:` is silently treated as `tdd: none`",

	// M-0172 area-unknown: the entity's `area` value is present but not
	// a member of the aiwf.yaml: areas set (typo protection). Absence is
	// never flagged and the rule is inert when no areas block exists, so
	// the only remediation paths are fixing the typo, declaring the value,
	// or removing the field.
	"area-unknown": "the entity's `area` is not in the declared set — retag it to a real member with `aiwf set-area <id> <member>`, add the value under `areas.members` in aiwf.yaml if it's a legitimate new workstream, or clear it with `aiwf set-area <id> --clear`; absence and an absent `areas` block are never flagged",

	// M-0178 area-required: the entity has no `area` but the consumer opted
	// into strictness via `aiwf.yaml: areas.required: true`. The remediation
	// is the M-0183 tag verb (`aiwf set-area <id> <member>`) or relaxing the
	// knob. Distinct from area-unknown (present-⇒-declared); this is
	// present-at-all.
	"area-required": "the entity has no `area` but `aiwf.yaml: areas.required` is set — tag it with `aiwf set-area <id> <member>` (a declared member of areas.members), or remove `areas.required` from aiwf.yaml if untagged entities are acceptable",

	// M-0180 area-dead-glob: a declared area's `paths:` glob matches no real
	// file or directory — dead config (a renamed / deleted / typo'd path).
	// Per-glob; warning by default, error under areas.required. The
	// remediation is fixing the glob, recreating the path, or dropping it.
	"area-dead-glob": "an `aiwf.yaml` area path glob matches no file or directory — correct the glob under that member's `paths:`, recreate the moved/renamed directory, or remove the dead glob, then re-run `aiwf check`",

	// M-0180 area-overlap: two declared areas' `paths:` globs both claim the
	// same directory — ambiguous attribution. Warning by default, error under
	// areas.required. The remediation is to make the globs disjoint.
	"area-overlap": "two `aiwf.yaml` areas claim the same directory — narrow one member's `paths:` glob so each directory belongs to at most one area, then re-run `aiwf check` (overlap makes the path-based area checks ambiguous)",

	// M-0185 area-unslotted: an immediate child directory of a declared
	// coverage root (aiwf.yaml: areas.coverage_roots) is claimed by no area's
	// `paths:` glob — an unslotted project. Warning by default, error under
	// areas.required. The remediation is to slot it into an area, narrow the
	// coverage root, or drop the root.
	"area-unslotted": "a directory under an `aiwf.yaml` coverage root is claimed by no area's `paths:` glob — slot it into a member's `paths:`, or remove the coverage root if that subtree is not a project-tiling scope, then re-run `aiwf check`; absence of a coverage root makes this check inert",

	// M-0185 area-coverage-root-missing: a declared coverage root resolves to
	// no directory (typo, deleted, or a file) — dead config, the coverage
	// analogue of area-dead-glob. A silently-skipped dead root gives false
	// confidence that coverage is active.
	"area-coverage-root-missing": "an `aiwf.yaml` coverage-root entry points at no directory — correct the path to the real coverage-scope directory, or remove the dead entry, then re-run `aiwf check`; a dead root silently disables coverage for that scope",

	// M-0185 area-coverage-no-paths: coverage_roots is declared but no area
	// declares `paths:`, so the path oracle is dormant and coverage is inert.
	// Surfaced rather than silently no-op'd.
	"area-coverage-no-paths": "`aiwf.yaml` coverage_roots is declared but no area declares `paths:`, so coverage has nothing to match against and is inert — add `paths:` to a member (areas.members[].paths), or remove the coverage roots if path-based coverage isn't wanted yet, then re-run `aiwf check`",

	// M-0181 area-mistag: an entity's linked commits (via the aiwf-entity
	// trailer) touched only a DIFFERENT area's `paths:` territory than the one
	// the entity is tagged to. Warning only — never escalated, because
	// cross-cutting work is legitimate. The remediation is to retag the entity
	// (`aiwf set-area`) or, if the work really is cross-cutting, acknowledge it.
	"area-mistag": "the entity's `area` tag and its commits disagree — its work landed entirely in another area's `paths:` territory; fix the tag with `aiwf set-area <id> <member>`, or if the work is genuinely cross-cutting, acknowledge it via `aiwf acknowledge mistag <id> --reason \"...\"`",

	// M-0130/AC-5: fsm-history-consistent fires when a status-change
	// commit bypasses the kernel's FSM in a way the per-subcode predicate
	// catches. Three subcodes cover the territory: illegal-transition
	// (change outside the FSM), forced-untrailered (sovereign-act-shape
	// change without aiwf-force), manual-edit (legal FSM step but no
	// aiwf-verb trailer at all). Hints land here ahead of M-0130/AC-2/3/4
	// per the A2 sequencing decision: PolicyFindingCodesHaveHints is
	// one-directional (fires on emitted codes lacking hints, not on hints
	// lacking codes), so landing the hints first is safe.
	"fsm-history-consistent/illegal-transition": "the status change is not a legal step in the kind's FSM and the commit has no `aiwf-force:` trailer; re-route through `aiwf promote <id> <to>` (which only accepts legal moves), or wield sovereign override via `aiwf <verb> --force --reason \"...\"` when the exceptional flip is genuinely warranted",
	"fsm-history-consistent/forced-untrailered": "the status change matches a sovereign-act shape (e.g., epic `proposed → active`) that requires explicit override but the commit has no `aiwf-force:` trailer; re-run the mutation as `aiwf <verb> <id> --force --reason \"...\"` so the override lands in the trailers, or undo the change via the corresponding inverse verb",
	"fsm-history-consistent/manual-edit":        "the status change has no `aiwf-verb:` trailer (manual `git commit` bypassed the kernel); re-route through the appropriate verb (`aiwf promote`, `aiwf cancel`), or record the exceptional flip via `aiwf <verb> --audit-only --reason \"...\"` after correcting the file by hand — the audit-only commit clears the finding",
	// M-0137/AC-4: history-walk-error subcode. The M-0130 walker
	// silently swallowed walker failures; M-0137 surfaces them as
	// findings so one transient subprocess error doesn't wipe the
	// rule's output (per CLAUDE.md §Engineering principles).
	"fsm-history-consistent/history-walk-error": "the walker hit a real failure reading the named entity's commit history (subprocess crash, blob read error, context cancelled mid-walk); other entities' findings are still surfaced alongside per the partial-preservation contract. Re-run `aiwf check` to confirm whether the failure is transient; if it repeats, inspect git's reachable-objects health (`git fsck`) or the consumer-repo permissions on `.git/objects/`",

	"contract-config/missing-entity":        "create a contract entity for this id (`aiwf add contract`), or remove the entry from aiwf.yaml.contracts.entries[]",
	"contract-config/missing-schema":        "correct the `schema:` path under `contracts` in aiwf.yaml (or create the file at that location), then re-run `aiwf contract verify`",
	"contract-config/missing-fixtures":      "correct the `fixtures:` path under `contracts` in aiwf.yaml (or create the directory), then re-run `aiwf contract verify`",
	"contract-config/no-binding":            "bind the contract via `aiwf contract bind`, or accept it as a registry-only record",
	"contract-config/path-escape":           "make the `schema:`/`fixtures:` paths under `contracts` in aiwf.yaml resolve inside the repo (drop `..` segments and out-of-repo symlinks), then re-run `aiwf contract verify`",
	"contract-config/validator-unavailable": "install the validator via `aiwf contract recipe install <name>` (or install its binary on this machine), or set `contracts.strict_validators: false` in aiwf.yaml to demote this to a warning team-wide",
	"fixture-rejected":                      "make the schema accept this fixture, or `git rm` the fixture out of valid/, then re-run `aiwf contract verify`",
	"fixture-accepted":                      "tighten the schema to reject this fixture, or `git mv` it into valid/, then re-run `aiwf contract verify`",
	"evolution-regression":                  "revert the schema change (`git checkout -- <schema>`) or migrate the historical fixture, then re-run `aiwf contract verify`",
	"validator-error":                       "every valid fixture failed — the schema or the validator invocation is likely broken; fix the `command:` under `contracts.validators` in aiwf.yaml (or the schema) and re-run `aiwf contract verify`",
	"environment":                           "install the validator via `aiwf contract recipe install <name>`, or fix its `command:` under `contracts.validators` in aiwf.yaml",

	// I2.5 provenance standing rules. These fire on commit history,
	// not on tree state — hints point to the verb / repair path that
	// would have produced a coherent commit.
	"provenance-trailer-incoherent":                     "amend the offending commit via `git commit --amend` so its trailer set obeys the required-together / mutually-exclusive rules in `docs/design/provenance-model.md`",
	"provenance-force-non-human":                        "`--force` requires `aiwf-actor: human/...`; have a human re-run the mutation as `aiwf <verb> <id> --force --reason \"...\"`, or drop the force and re-route through the normal verb",
	"provenance-actor-malformed":                        "set `git config user.email` to a valid address and re-run via `aiwf doctor`; the actor trailer is derived from `<localpart>` of the email",
	"provenance-principal-non-human":                    "`aiwf-principal:` must be `human/<id>` (agents and bots cannot be principals); re-run the verb with `--principal human/<id>`, or amend the trailer via `git commit --amend`",
	"provenance-on-behalf-of-non-human":                 "`aiwf-on-behalf-of:` must name a human principal; read the originating authorize commit with `aiwf history <scope-entity>` and amend the trailer via `git commit --amend`",
	"provenance-authorized-by-malformed":                "`aiwf-authorized-by:` must be 7–40 hex (the SHA of the authorize commit); copy it from `aiwf history <scope-entity>`",
	"provenance-authorization-missing":                  "the SHA does not name an `aiwf-verb: authorize / aiwf-scope: opened` commit; find it with `aiwf history <scope-entity>` and use the full SHA",
	"provenance-authorization-out-of-scope":             "the scope-entity does not reach the target via the reference graph; open a scope on the right entity with `aiwf authorize <id> --to <agent>`, or run the verb on something the existing scope already reaches",
	"provenance-authorization-ended":                    "the scope was already ended (terminal-promote or revoke); open a fresh scope with `aiwf authorize <id> --to <agent>`",
	"provenance-no-active-scope":                        "an `ai/...` actor needs an active authorization; run `aiwf authorize <id> --to <agent>` before retrying the verb",
	"provenance-audit-only-non-human":                   "`--audit-only` is a sovereign act; only humans may backfill audit trails (have a human invoke `aiwf <verb> --audit-only --reason ...`)",
	"provenance-untrailered-entity-commit":              "the commit modified this entity via plain `git commit`; two recovery paths: (1) `aiwf acknowledge illegal <sha> --for-entity <id> --reason \"...\"` — SHA-verified per-(commit, entity) ack, the kernel walks `git diff-tree` to confirm the binding (G-0231 item 3); (2) `aiwf promote <id> <state> --audit-only --reason \"...\"` or `aiwf cancel <id> --audit-only --reason \"...\"` — per-entity blanket, no SHA binding. Use (1) for body-edit acks where the SHA is real and the kernel should verify; use (2) for status flips where the per-entity blanket fits. Either clears the matching finding on the next push.",
	"provenance-untrailered-entity-commit/squash-merge": "the squash-merge from the GitHub UI dropped the original commits' aiwf-verb trailers; switch the repo's merge strategy to rebase-merge or `--no-ff` merge for branches that touch entity files, OR run `aiwf <verb> <id> --audit-only --reason \"...\"` per entity touched to backfill the audit trail",
	"provenance-untrailered-scope-undefined":            "the audit range is undefined; configure an upstream (`git push -u origin <branch>`) or pass `aiwf check --since <ref>` to opt back in",

	// G-0150: trailer-verb-unknown fires when a commit's `aiwf-verb:`
	// value is neither in the running binary's Cobra verb tree nor in
	// the recognized ritual-verb allowlist — typically an LLM-fabricated
	// value (e.g. `aiwf-verb: implement`) on a hand-rolled Conventional-
	// Commits commit. Known ritual lifecycle verbs (e.g. `wrap-epic`) are
	// allowlisted (G-0180) and do not fire.
	"trailer-verb-unknown": "the commit's `aiwf-verb:` value is not a registered top-level verb or subverb, nor a recognized ritual verb; if it's a typo (or an LLM fabrication), `git commit --amend` and drop the trailer — plain `feat(...)` / `fix(...)` commits don't need an `aiwf-verb:` line; if it's a new ritual verb, add it to the ritualVerbs allowlist in internal/check/trailer_verb_unknown.go",

	// M-0160/AC-4: id-rename-untrailered fires when a commit between
	// merge-base(HEAD, trunk) and HEAD renames an id-bearing entity
	// file without an aiwf-verb trailer in the rename-class closed
	// set. The canonical resolution is `aiwf reallocate` — it
	// rewrites the frontmatter, walks the tree to rewrite every
	// cross-reference to the old id, and stamps the proper
	// `aiwf-verb: reallocate` + `aiwf-prior-entity:` trailers so
	// `aiwf history` bridges old→new. Sovereign-human override via
	// `aiwf acknowledge illegal` is the post-hoc silencing path for
	// renames that were deliberate.
	"id-rename-untrailered": "the commit renamed an id-bearing entity file without an `aiwf-verb` trailer in the rename-class set (retitle/rename/reallocate/archive/move). Canonical resolution: run `aiwf reallocate <new-id-or-path>` to record the renumber with the proper trailer set — that rewrites cross-references and bridges `aiwf history` from the old id; alternatively, if the original rename was deliberate sovereign-human work, run `aiwf acknowledge illegal <sha> --reason \"<text>\"` to silence this specific commit's finding without rewriting history. See CLAUDE.md §\"Id-collision resolution at merge time\".",

	// Verb-emitted findings (from internal/verb/).
	"unexpected-tree-file": "remove the file with `git rm <path>` or move it outside `work/`; if it genuinely belongs there, add a glob to `tree.allow_paths` in aiwf.yaml — but tree-shape changes (new entities, renames, status transitions) go through `aiwf <verb>`, not direct writes",

	"slug-dropped-chars":  "the title contained non-ASCII runes that the slug omits; rename via `aiwf rename` if the resulting slug isn't what you want",
	"import-duplicate-id": "the manifest declares the same id more than once; deduplicate the entries before re-running `aiwf import`",
	"import-collision":    "the manifest's explicit id is already taken by an existing entity; re-run with `aiwf import --on-collision skip|update`, or change the manifest's id",

	// G-0185: roadmap-case-collision fires when more than one
	// case-variant of the generated ROADMAP.md artifact exists at the
	// repo root. Only physically possible on a case-sensitive filesystem;
	// `aiwf render roadmap --write` reconciles to a single existing
	// variant but cannot pick between two, so the renderer leaves this
	// advisory for the operator to resolve.
	"roadmap-case-collision": "remove one case-variant of the roadmap file (`git rm`) so a single canonical ROADMAP.md (or the lowercase convention the repo already uses) remains at the repo root",

	// isolation-escape — AI-actor commit on a branch that doesn't
	// match the active scope's recorded aiwf-branch:. Three sovereign
	// override paths leave a clean audit trail; the hint names all
	// three so an operator who hits the finding sees a single place
	// that lists every legitimate way out. acknowledge-illegal is
	// listed first as the canonical kernel-native path — separate
	// empty commit, no history rewrite, traces via `aiwf history`
	// through the aiwf-force-for trailer; the cherry-pick and amend
	// paths remain documented for the cases where a human re-author
	// or in-place sovereign override is the right shape. Lineage via
	// `aiwf history` covers M-0106 (original 2-path hint) and
	// M-0159/AC-9 (acknowledge-illegal addition).
	"isolation-escape":                    "the AI-actor commit landed on a branch that doesn't match the active scope's recorded `aiwf-branch:`. Override paths: (a) canonical: run `aiwf acknowledge illegal <sha> --reason \"<text>\"` as a human actor — records a separate audit-trail commit (aiwf-verb: acknowledge-illegal + aiwf-force-for: <sha>) that silences the finding without rewriting the original commit; (b) re-author via `git cherry-pick -x <sha>` — preserves the marker and changes the committer to a human, suppressing the finding; (c) amend the violating commit with `git commit --amend --trailer 'aiwf-force: <reason>'` and an `aiwf-actor: human/<id>` trailer to record the sovereign override. See E-0030 epic body §\"Sovereign override surface\" for the audit trail each path produces.",
	"isolation-escape-oracle-failure":     "advisory: the branch-choreography oracle could not resolve one or more refs, so isolation-escape could not be checked for the affected commits. It fails shut on correctness (no false positives) and fails open on coverage, so this is operator visibility, not a blocker — inspect the named ref (`git show-ref <ref>`) and re-run `aiwf check` once it resolves. See D-0019 for the contract.",
	"isolation-escape-shallow-clone":      "the repository is a shallow clone, so the per-commit branch map is left empty and isolation-escape can't run (a total-coverage gap). Unshallow with `git fetch --unshallow`, or in CI use `actions/checkout@vN` with `fetch-depth: 0`.",
	"isolation-escape-orphaned-ai-commit": "an AI-actor commit was orphaned by a non-fast-forward update (force-push) on a ritual branch, so the kernel can't tell whether it was on the correct branch. Review the commit; if it was deliberate sovereign-human work, record it via `aiwf acknowledge illegal <sha> --reason \"<text>\"` as a human actor.",
	"promote-on-wrong-branch":             "an activating-promote (e.g. `aiwf promote E-NNNN active` / `aiwf promote M-NNNN in_progress`) landed on a branch other than the entity's expected parent branch, contrary to ADR-0010 (sovereign acts land on the parent branch before the ritual branch is cut). Land future activations on the parent branch; if this placement was deliberate, silence the commit via `aiwf acknowledge illegal <sha> --reason \"<text>\"` as a human actor, or add an `aiwf-force: <reason>` trailer to the promote commit.",
}

// HintFor returns the canonical action hint for a given code+subcode.
// Returns "" when no hint is registered. Verb-side findings (e.g.,
// reallocate-body-reference) call this so the human-facing suggestion
// stays in one place.
func HintFor(code, subcode string) string {
	if subcode != "" {
		if h, ok := hintTable[code+"/"+subcode]; ok {
			return h
		}
	}
	return hintTable[code]
}

// applyHints fills in Hint on every finding from the hint table.
// Findings whose Hint is already set are left alone, so callers can
// override the default by setting Hint at construction time.
func applyHints(findings []Finding) {
	for i := range findings {
		f := &findings[i]
		if f.Hint != "" {
			continue
		}
		f.Hint = HintFor(f.Code, f.Subcode)
	}
}
