---
title: Gap truth audit — evidence
status: captured
date: 2026-08-23
---

# Gap truth audit — evidence

Supporting evidence for
[`gap-truth-audit.md`](gap-truth-audit.md). Two things the summary cannot carry:
every high-severity finding as its auditor recorded it, and the recommended
disposition for all 143 subjects.

Findings are reproduced verbatim — claim, measurement, command and quoted fragment
in whatever shape the auditor used. Medium and low-severity detail is deliberately
absent: the summary carries per-subject counts, and any single gap can be
re-measured in minutes with the protocol the summary describes.

Ages exactly as the summary does. Every entry is either fixed and deleted, or still
true.

## High-severity findings

Grouped by subject, most-cited first within each. Linked from the summary's
findings ledger.

### G-0060

**`dead-premise`**

- **Claim:** *"'Patch' appears in the consumer-facing rituals (the optional `wf-rituals` plugin's TDD cycle / code-review / doc-lint surface) … The kernel says nothing about it."*
- **Measured:** `wf-rituals` is no longer an optional marketplace plugin, and patch is no longer a mention inside three other skills — it is its own 16-step ritual. ADR-0014 (`accepted`, added + promoted 2026-05-29) §1 rules the rituals are embedded in the binary and materialized by `aiwf init`/`update`. `<aiwf> update --help` offers no plugin-selection flag. `wf-patch/SKILL.md` was vendored into the embedded snapshot on 2026-05-29 (`8a2c8acdf`), and `.claude/skills/wf-patch` exists as a materialized artifact in this repo alongside eight other `wf-*` skills. G-0060's "What's missing" text predates all of it (last touched at the 2026-05-11 rename `9df927060`; the 2026-07-22 edit `765e8da78` appended the Investigation section without revisiting these bullets).
- **Command:** `<aiwf> show ADR-0014 --format=json`; `sed -n '1,45p' docs/adr/ADR-0014*.md`; `<aiwf> update --help`; `git log --diff-filter=A --format='%h %ad %s' --date=short -- internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md`; `ls .claude/skills | grep '^wf-'`; `git log --format='%h %ad %s' --date=short -- work/gaps/G-0060-patch-ritual-loosely-defined.md`
- **Quoted:** ADR-0014 §1: *"The rituals are **embedded in the engine binary via `go:embed`** and written into the consumer repo by `aiwf init` / `aiwf update`, using the same marker-managed, gitignored, wipe-and-rewrite pipeline that already ships the verb skills."*

**`false-claim`**

- **Claim:** *"No branch model. A patch is whatever-shaped work on whatever-shaped branch, with no kernel-level guidance."*
- **Measured:** The kernel carries a patch branch grammar in Go and a closed rung ladder that includes `patch` as a rung. `internal/branchparse/branchparse.go:30` is `^(?:epic/([Ee]-\d+)|milestone/([Mm]-\d+)|patch/([Gg]-\d+))(?:-|$)`, and `branchparse.go:127-132` declares `legalRungPairs` = `{trunk→epic, epic→milestone, milestone→patch, epic→patch}` with the comments *"milestone → patch — wf-patch under a milestone"* and *"epic → patch — wf-patch directly under an epic"*. Driven live: in a fixture, `<aiwf> authorize E-0001 --to ai/claude` succeeds from branch `patch/G-0001-fix-widget` and is refused from branch `randomthing` with *"current checkout \"randomthing\" does not match a ritual shape … (epic/E-NNNN-&lt;slug&gt; / milestone/M-NNNN-&lt;slug&gt; / patch/g-NNNN-&lt;slug&gt;)"*. ADR-0010 (`accepted`, 2026-05-11/12) line 54 carries the branch-namespace table row for `wf-patch`, and `wf-patch/SKILL.md` Constraints pins `patch/G-NNNN-<short-slug>`. `internal/cli/worktree/worktree.go:51` uses `aiwf worktree add patch/G-0100-fix` as the verb's own help example.
- **Command:** fixture `git init`; `<aiwf> add epic --title "Some epic" --body "## Goal\n\nDo a thing." --root $S --actor human/peter`; `git checkout -b patch/G-0001-fix-widget`; `<aiwf> authorize E-0001 --to ai/claude --actor human/peter --reason delegate --root $S`; then the same from `randomthing`. Plus `sed -n '105,135p' internal/branchparse/branchparse.go`; `grep -n patch docs/adr/ADR-0010*.md`
- **Quoted:** `branchparse.go:119-120`: *"- milestone → patch  — wf-patch under a milestone. / - epic → patch       — wf-patch directly under an epic, skipping an intermediate milestone"*

**`false-claim`**

- **Claim:** *"**`aiwf check` has no patch-shaped invariants to enforce.** Without a defined shape, `check` cannot say anything is wrong."*
- **Measured:** `aiwf check` treats `patch/` as one of exactly three ritual namespaces it polices. `internal/check/reflog_walk.go:168-175` (`ritualShape`) matches `epic/`, `milestone/`, `patch/`, and `listRitualHeads` (`reflog_walk.go:137`) filters the branch set it walks for force-pushed-away AI commits through it; `internal/cli/check/isolation_escape_oracle.go:319` names the same set. The consuming finding is `isolation-escape` (`internal/check/isolation_escape.go:34`, `codes.ClassBranchChoreography`), which fires when an AI actor's commit lands off its scope's recorded branch. So a patch branch has a defined shape *and* a check rule that judges commits on it. `internal/check/id_rename_untrailered.go:36` calls this "the branch-policing finding set alongside isolation-escape". Reading of source only — I did not stage a force-pushed orphan to drive the rule.
- **Command:** `awk 'NR>=125 && NR<=175' internal/check/reflog_walk.go`; `sed -n '1,45p' internal/check/isolation_escape.go`; `grep -rn 'patch/' --include='*.go' internal/ cmd/ | grep -v _test`
- **Quoted:** `reflog_walk.go:158-159`: *"ritualShape reports whether the branch name sits in an aiwf ritual namespace (epic/, milestone/, patch/)."*

**`false-claim`**

- **Claim:** *"No relationship to gaps, ADRs, or contracts. A patch that closes a gap has no formal way to record `closes G-NNN` in a way `aiwf check` can validate."*
- **Measured:** The closure route exists, is verb-guarded, and *is* partly check-validated. Driven end-to-end in a fixture reproducing the `wf-patch` shape (branch `patch/G-0001-fix-thing` → commit → `git merge --no-ff` → `aiwf promote G-0001 addressed --by-commit <merge-sha>`): the promote succeeded and wrote `addressed_by_commit: [d61c164…]` into the gap's frontmatter. The verb refuses every degenerate form — no resolver → *"promoting a gap to \"addressed\" requires --by &lt;entity-id&gt; or --by-commit &lt;sha&gt; so the gap-addressed-has-resolver rule is satisfied"*; a fabricated SHA → *"does not resolve to a commit in this repo"*; a real-but-unreachable SHA → *"resolves to a commit not reachable from HEAD"*. On the check side, `gapAddressedHasResolver` (`internal/check/check.go:980-995`) fires at **warning** when an addressed gap carries neither resolver. **What is genuinely missing is narrower than the claim:** `aiwf check` validates only *presence*, not the value — a hand-edited `addressed_by_commit: [not-a-sha-at-all]` produced no finding at all in the fixture.
- **Command:** fixture; `<aiwf> promote G-0001 addressed --by-commit $MERGESHA --root $S --actor human/peter`; `cat work/gaps/G-0001-thing-is-broken.md`; `<aiwf> promote G-0002 addressed …` (no resolver / bogus sha / unreachable sha); hand-write `addressed_by_commit: [not-a-sha-at-all]` then `<aiwf> check --root $S --format=json`
- **Quoted:** the body's own later Investigation section already concedes it: *"A patch closing more than one gap is *not* structurally blocked today: `aiwf promote G-NNNN addressed --by-commit <sha>` … can run once per gap against the same merge SHA"* — the two sections contradict each other.

### G-0068

**`contradicted-by-code`**

- **Claim:** "Only `/ac` is a literal in source. The other six are derived via
  `string(e.Kind)` and bypass the policy entirely." — i.e. `entity-body-empty/ac`
  *is* required to be discoverable.
- **Measured:** It is not. In a clone of HEAD I replaced the only haystack
  occurrence of `entity-body-empty/ac`
  (`internal/skills/embedded/aiwf-check/SKILL.md`) with `entity-body-empty/zz` and
  ran the policy: **PASS, zero violations**. Positive control on the same clone:
  removing `contract-config/no-binding` (whose site has a *literal* `Code:
  "contract-config"`) fires immediately —
  "`contract-config/no-binding` is a finding code in the kernel but is not
  mentioned in any AI-discoverable channel". Third control: removing the bare
  string `entity-body-empty` everywhere in the haystack (which also removes all
  seven composites) yields **exactly one** violation, for the bare code only.
  Cause: `loadCheckCodeLiterals` short-circuits on
  `code := stringFieldValue(cl, "Code"); if code == "" …  return true` before it
  reaches the `Subcode` branch, and `entity_body.go`:179 emits
  `Code: CodeEntityBodyEmpty` — an `*ast.Ident`, which `stringFieldValue` rejects.
  `internal/check/` has **0** literal `Code: "…"` sites and 91 constant ones; the
  only literal-`Code` sites left repo-wide are 4 in
  `internal/contractcheck/contractcheck.go`, all `contract-config`.
- **Command:** `git clone --shared --single-branch /workspaces/aiwf <clone>`;
  `sed -i 's|entity-body-empty/ac`|entity-body-empty/zz`|g' internal/skills/embedded/aiwf-check/SKILL.md`;
  `go test -run 'TestPolicy_FindingCodesAreDiscoverable' -count=1 ./internal/policies/` → `ok`;
  same with `contract-config/no-binding` → `FAIL`;
  same with every `entity-body-empty` occurrence → `FAIL` with one violation;
  `grep -rnE 'Code: *"' internal/check/ --include="*.go" | grep -v _test.go | wc -l` → 0
- **Quoted:** body — "Only `/ac` is a literal in source."; source —
  `discoverability.go`:107-113, "code := stringFieldValue(cl, \"Code\") / if code
  == \"\" || !looksLikeFindingCode(code) { return true } … if sub :=
  stringFieldValue(cl, \"Subcode\"); sub != \"\"".

**`stale-claims`**

- **Claim:** "Discovered during M-0066/AC-6's RED-first sanity check: deleting the
  SKILL.md row produced exactly one violation (`entity-body-empty/ac`), surfacing
  the asymmetry."
- **Measured:** True when written, false now. At `56ad4b841^` the site read
  `Code: "entity-body-empty"` (a literal), so the composite did enter the required
  set; `56ad4b841` (2026-05-31, "chore(check): type finding-code constants and
  enforce adoption (closes G-0129)") converted it to `Code: CodeEntityBodyEmpty`.
  Re-running the same experiment against HEAD today produces **zero** violations
  for `/ac` (above). G-0068's body has not been substantively edited since
  2026-05-10.
- **Command:**
  `git show 56ad4b841^:internal/check/entity_body.go | grep -n "Code:\|Subcode:"` →
  `Code: "entity-body-empty"` / `Subcode: string(e.Kind)` / `Code:
  "entity-body-empty"` / `Subcode: "ac"`; `git show 56ad4b841:…` → `Code:
  CodeEntityBodyEmpty` ×2; `git log --format='%h %ad %s' --date=short -- work/gaps/G-0068*.md`
- **Quoted:** M-0066's wrap, which is the source of the sentence — "deleted the
  row entirely. `TestPolicy_FindingCodesAreDiscoverable` failed red on
  `entity-body-empty/ac` — exactly one violation."

### G-0070

**`dead-premise`**

- **Claim:** "The `recommended-plugin-not-installed` finding-code string appears verbatim in
  the output (so a script can grep for it), but the structured `finding.data` payload doesn't
  exist as a queryable surface — a caller would have to regex the install command out of the
  prose continuation line."
- **Measured:** The check does not exist, so nothing appears in the output. `$A doctor`
  against `/workspaces/aiwf` prints 19 report lines and no such string.
  Tree-wide, `recommended-plugin-not-installed` survives in non-test Go at exactly one place:
  `internal/policies/m0202_devcontainer_onboarding.go:44`, where it is listed among
  `retiredMarkers` and the docblock at `:21` calls it "the retired aiwf doctor warning …
  that no longer exists". `grep -rn "recommended_plugins\|RecommendedPlugins" --include="*.go"
  internal/ cmd/` finds one hit, in `internal/workflows/spec/antirules.go:65`, a prose
  statement — `internal/config/config.go` has no `Doctor` field. Retired by `2ab09ac8b`
  ("feat(doctor): retire marketplace channel — verify materialized rituals + de-dupe guard
  (M-0152)", 2026-05-29), machinery removed by `debf7f5ff` ("chore: remove marketplace-plugin
  transitional machinery (G-0194)", 2026-05-31). The governing decision, `$A show D-0016` →
  "Retire doctor.recommended_plugins; verify materialized rituals + de-dupe guard",
  **accepted**.
- **Command:** `$A doctor` · `grep -rn "recommended-plugin-not-installed" --include="*.go"
  --include="*.md" .` · `grep -rn "recommended_plugins\|RecommendedPlugins" --include="*.go"
  internal/ cmd/` · `git log --oneline -S 'recommended-plugin-not-installed' -- internal/` ·
  `$A show D-0016`
- **Quoted:** as above.

**`dead-premise`**

- **Claim:** "When implemented, M-0070's AC-3 contract is automatically satisfied without
  spec changes."
- **Measured:** It cannot be. `$A show M-0070` → `done`, archived, AC-3 "Each missing plugin
  emits one warning with install command" **met**. Its body demands "JSON envelope output
  carries the same structured info (`finding.code: \"recommended-plugin-not-installed\"`,
  `finding.data: {plugin, marketplace, install_command}`)". The check that would emit that
  finding no longer exists (previous finding), so a `--format=json` on doctor would satisfy
  nothing — automatically or otherwise. The quotation itself is faithful: M-0070's AC-3 body
  carries that sentence verbatim.
- **Command:** `$A show M-0070` · `grep -n "finding.code" work/epics/archive/E-0018-*/M-0070-*.md`
- **Quoted:** as above.

### G-0073

**`dead-premise`**

- **Claim:** two of the five "concrete cross-kind cases the kernel can't represent today", plus one fix-shape arm, rest on ADR-0003 and the `finding` kind.
- **Measured:** ADR-0003 is **`rejected`** and archived (`docs/adr/archive/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md`), reversed by ADR-0045 (`accepted`, 2026-08-19). E-0019's *Dependencies* section — the very prose G-0073 cites — now reads *"**The F-NNNN storage model** ([ADR-0003]) — **withdrawn.** aiwf carries six entity kinds and `finding` is not one … The two implementation epics this list anticipated … neither was filed"*, and lists ADR-0004 as *"accepted, and the convention has shipped. **Met.**"*. So the bullet *"E-0019's Dependencies prose lists ADR-0003 and ADR-0004 as required"* is false on both halves — one is withdrawn, the other is met. The bullet *"**Implementation-epic chains.** Once ADR-0003 is ratified, an implementation epic for `finding` is filed"* names a future that has been ruled out. Fix-shape item 1's *"plus future finding"* and the predicate table's row *"| Finding (future) | `resolved` |"* are dead for the same reason.
- **Command:** `<aiwf> show ADR-0003 --format=json`; `<aiwf> show ADR-0045 --format=json`; `<aiwf> show E-0019 --format=json`; `grep -n -A12 '^## Dependencies' work/epics/E-0019-.../epic.md`
- **Quoted:** ADR-0045 Decision: *"aiwf carries six entity kinds. `finding` is not added."*

**`false-claim`**

- **Claim (2026-08-12, first instance):** *"The epic-spec template `aiwf update` materializes carries a `depends_on` entry in its frontmatter block, annotated as optional prior epic ids, so an author following the template writes it on an epic — as one did."* And the closing sentence: *"the epic template advertises the field and the schema drops it, and either one moving settles that much."*
- **Measured:** **the template moved.** `db843cead` (2026-08-22, one day before HEAD, `aiwf-entity: G-0617`) deleted the line `depends_on: []           # optional: prior epic ids; e.g. [E-NNNN]` from `internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/epic-spec.md`. `grep -n depends_on` on that file returns nothing; the frontmatter block is now `id / title / status` plus a pointer comment to `aiwf schema epic`. G-0617 ("Template frontmatter restates each kind's vocabulary, unchecked") is `addressed` and archived. The gap's own stated settlement condition is met.
- **Command:** `grep -n "depends_on" internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/epic-spec.md` (no output); `git log -S 'depends_on' --oneline -- …/epic-spec.md`; `git show db843cead -- …/epic-spec.md`; `<aiwf> show G-0617 --format=json`
- **Quoted:** `db843cead`'s message: *"The epic template also offered a depends_on field the kind does not have… Measured against a fixture tree, an epic carrying it draws no finding at all… No epic in a tree carries it, so removing the line migrates nothing."*

**`false-claim`**

- **Claim (2026-08-12, second instance):** *"E-0084's spec resolves that question 'with E-0083 before either epic's first milestone lands'. E-0083's spec resolves it 'in the rule's own milestone'. Both statements are prose, neither is wrong on its own terms, and nothing reconciles them: whichever epic starts first settles the question alone."*
- **Measured:** the two specs **agree today**, and have since 17 seconds after this section was committed. G-0073's edit is `f202250c1` at `2026-08-12 13:13:41 +0000`; `9e9835ae1` (`aiwf edit-body E-0083`) landed at `2026-08-12 13:13:58 +0000` and replaced the quoted resolution path. E-0083's open-questions table row 120 now reads *"Shared with E-0084, which adds the membership rule ADR-0043 decides. Settled jointly before either epic's first milestone lands, not inside whichever starts first — ADR-0043 names the question and neither epic owns it alone."* E-0084's constraint (line 74) and table row 104 match. E-0083 additionally added a risk row (line 128) that **cites this gap by id** for the general point. So the *instance* is settled; the *argument it supports* — no `depends_on` edge expresses "neither of us lands before we settle this together" — survives, and is now sourced from E-0083 rather than from a disagreement that no longer exists.
- **Command:** `git log --format='%h %ad %s' --date=iso -- work/gaps/G-0073-*.md`; `git show -s --format='%h %ad' --date=iso 9e9835ae1`; `grep -n -B3 -A6 "finding code\|E-0084" work/epics/E-0083-*/epic.md`; `grep -n -B2 -A6 "E-0083" work/epics/E-0084-*/epic.md`
- **Quoted:** E-0083 spec line 128: *"Both specs now name it as jointly owned and settled before either epic's first milestone lands. Nothing mechanical holds them to it — the constraint is prose on both sides, which G-0073 records as the case no `depends_on` edge expresses."*

### G-0110

**`contradicted-by-code`**

- **Claim (title and opening sentence):** "The `--diff <ref>` flag on `gremlins
  unleash` … excludes mutants in files that are *entirely new in the branch*,
  not just lines unchanged versus the diff target."
- **Measured:** false as stated. In a disposable single-package module at module
  root, `gremlins unleash --dry-run --diff main .` marked the **new** file
  `b.go`'s mutants `RUNNABLE` and correctly `SKIPPED` the unchanged `a.go`:

  ```
  SKIPPED CONDITIONALS_NEGATION at a.go:4:7
  SKIPPED CONDITIONALS_BOUNDARY at a.go:4:7
  RUNNABLE CONDITIONALS_NEGATION at b.go:4:7
  RUNNABLE CONDITIONALS_BOUNDARY at b.go:4:7
  Runnable: 2, Not covered: 0
  ```

  The real discriminator is the **path argument**, not file newness. In a second
  disposable module with the package at `internal/check/`:

  - `gremlins unleash --dry-run ./internal/check` → `Runnable: 4` (both files).
  - `gremlins unleash --dry-run --diff main ./internal/check` → **`Runnable: 0`**,
    every mutant `SKIPPED` — including `b.go` (new) *and* `a.go` after I
    **modified** it on the branch, which the gap's diagnosis predicts should stay
    runnable.
  - `gremlins unleash --dry-run --diff main .` on that same repo → `Runnable: 4`,
    with paths reported module-root-relative (`internal/check/a.go`), versus
    scan-root-relative (`a.go`) under the subpath run.

  So `--diff` compares module-root-relative git paths against scan-root-relative
  mutant paths; below the module root nothing ever matches and everything is
  skipped. The gap's own evidence is consistent with this and inconsistent with
  its own diagnosis: "192 SKIPPED mutants and 0 runnable" on a package with
  several changed files could not happen if only *new* files were excluded.
- **Command:** `gremlins unleash --dry-run --diff main .` and
  `gremlins unleash --dry-run --diff main ./internal/check` in two `git init`
  fixtures under the scratch dir; `go version -m /go/bin/gremlins` →
  `mod github.com/go-gremlins/gremlins v0.6.0`, matching
  `.github/workflows/mutate-hunt.yml:145`
  (`go install …/cmd/gremlins@v0.6.0`).
- **Quoted:** "excludes mutants in files that are *entirely new in the branch*,
  not just lines unchanged versus the diff target"

**`dead-premise`**

- **Claim:** "With `--diff` broken for new files, the operator either runs the
  full package (slow, noisy triage) or skips mutation testing entirely (loses the
  evidence). … operators need a working diff-scoped mutation run for future
  milestones."
- **Measured:** a working diff-scoped mutation run shipped. `make mutate-diff`
  exists (`Makefile:244-245` → `scripts/mutate-diff.sh`), was built for G-0267
  (`aiwf show G-0267` → `addressed`, "No diff-scoped mutation check on changed
  code or new tests"), and does **not** use `--diff` at all: it resolves
  `git merge-base origin/main HEAD`, computes changed `internal/` packages from
  `git diff --name-only "$base" -- '*.go'` **plus**
  `git ls-files --others --exclude-standard -- '*.go'` (untracked new files
  explicitly included, `scripts/mutate-diff.sh:56-61`), and then runs
  `gremlins unleash --workers 1 --timeout-coefficient "$COEFF" --output "$report"
  "$pkg"` per package (line 86) — no `--diff` flag. Two workarounds therefore
  exist that the gap does not name: the shipped `make mutate-diff`, and simply
  invoking `--diff` from the module root.
- **Command:** `grep -n "mutate-diff" Makefile`; `cat -n scripts/mutate-diff.sh`;
  `aiwf show G-0267`.
- **Quoted:** "the operator either runs the full package (slow, noisy triage) or
  skips mutation testing entirely"

### G-0111

**`already-addressed`**

- **Claim:** Concern 4 — "Wrap doesn't close the epic's named resolution gaps … The mechanism is missing on the epic surface." Restated in *Why it matters* as "**no mechanism to land the gap-status flips the epic claims to deliver**".
- **Measured:** The mechanism shipped on 2026-07-21 in `9cad2af3e` ("fix(ritual): close gaps a milestone's own prose claims to fix at wrap (G-0431)"). `aiwfx-wrap-epic/SKILL.md:24` now carries Precondition 6 — "Neither the epic's own spec nor any milestone's left a gap open that it explicitly claims to fix" — with the disposition rules at line 26; `aiwfx-wrap-milestone/SKILL.md:26` identifies the gaps at step 1 and line 241 runs `aiwf promote G-NNNN addressed --by-commit <sha>` at step 13, ahead of the milestone's own promote-done. The owning gap **G-0431** ("Milestone/epic wrap never closes gaps their own prose claims to fix") is `addressed`. This is precisely the "Skill-driven" option G-0111 itself enumerates; only the "Declarative" option (a `closes:`/`addressed_by:` field on the resolver's frontmatter) is unbuilt — `$A schema` shows milestone references are still only `parent` and `depends_on`.
- **Command:** `git log --oneline -S "explicitly claims to fix" -- internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/`; `git log -1 --format='%h %ci %s%n%b' 9cad2af3e`; `$A show G-0431 --format json`; `grep -n "" internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md | sed -n '17,27p'`; `grep -n "addressed --by-commit\|explicitly claims" internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md`
- **Quoted:** body — "The mechanism is missing on the epic surface." / shipped skill — "6. Neither the epic's own spec nor any milestone's left a gap open that it explicitly claims to fix."

**`dead-path`**

- **Claim:** Resolution path — "Cross-repo coupling pattern from M-0090 / M-0096 applies — author the skill body in `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md` and copy at wrap."
- **Measured:** `internal/policies/testdata` does not exist. The canonical authoring location is pinned as a Go constant: `internal/policies/aiwfx_wrap_epic_test.go:19` — `const aiwfxWrapEpicFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md"`, whose own doc comment (line 11-12) calls it "the canonical authoring location for the `aiwfx-wrap-epic` skill body — the embedded ritual snapshot". The "copy at wrap" half is dead too: ADR-0016 (`accepted`) archives the upstream rituals repo and ADR-0014 (`accepted`) makes the embedded snapshot the single source. A reader following this path writes into a directory that does not exist and looks for a repo that takes no changes.
- **Command:** `ls -d internal/policies/testdata`; `grep -n "aiwfxWrapEpicFixturePath" internal/policies/aiwfx_wrap_epic_test.go`; `$A show ADR-0016 --format json`; `$A show ADR-0014 --format json`
- **Quoted:** "author the skill body in `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md` and copy at wrap"

### G-0121

**`stale-claim`**

- **Claim:** "G-0567 is one such disagreement, live and unfixed."
- **Measured:** `$A show G-0567` → "status: addressed · priority: high · archived", with
  `addressed_by_commit: 29eb2a94cdb1551e88d92f1dfa20e00ca658f100` =
  `fix(check): apply one aiwf.yaml severity policy at every check.Run call site`, dated
  2026-08-09 — one day after G-0121's note was written (`d7677502d`, 2026-08-08). A reader
  acting on this sentence would go looking for a live violating state that no longer exists,
  which matters because the body's own argument says an agreement property "must be
  demonstrated failing against a real violating state".
- **Command:** `$A show G-0567`; `git log -1 --format='%H %ad %s' --date=short 29eb2a94c`;
  `git log -1 --format='%ad' --date=short d7677502d`
- **Quoted:** "Then size a cross-surface agreement property as its own piece of work, against
  a real violating state — G-0567 is one such disagreement, live and unfixed."

### G-0161

**`contradicted-by-code`**

- **Claim:** "**ANTI-0007** (no kernel branch-of-verb rule): invoke a mutating
  verb on any branch (main / feature / detached HEAD), confirm verb succeeds with
  no branch-specific check."
- **Measured:** the verb refuses. In the fixture, on `main`, with milestone
  M-0001 under epic E-0001:
  `$A promote M-0001 in_progress` → exit 2,
  `aiwf promote: aiwf promote M-0001 in_progress: refusing to land on "main" — this activation is expected on "epic/E-0001-fixture-epic-one" (a concurrent session checked out a different branch here? see G-0269); … or use --force --reason "..." to override`.
  The guard is `requireExpectedBranchForActivatingTransition`,
  `internal/verb/promote_branch_guard.go:37-54`, landed `ffe373de2` (2026-07-11,
  "feat(verb): refuse epic/milestone activation promotes on the wrong branch
  (G-0269)") — i.e. *after* `antirules.go` was authored (`e5ad99e54`,
  2026-05-19). There is also a check-time finding `promote-on-wrong-branch`
  (`internal/check/promote_on_wrong_branch.go`, hint at
  `internal/check/hint.go:357`) and an 11-illegal-cell branch-choreography rule
  layer at `internal/workflows/spec/branch/rules.go`. The sketch as written would
  fail; ANTI-0007's Statement ("There is NO kernel rule about which branch a verb
  is legal on") is false today.
- **Command:** in the fixture — `$A add epic --title …`, `$A promote E-0001 active`,
  `$A add milestone --epic E-0001 --tdd none --title …`, `$A promote M-0001 in_progress`;
  then `grep -rn 'refusing to land on' --include='*.go' internal/` and
  `git log --oneline --date=short -- internal/verb/promote_branch_guard.go`
- **Quoted (antirules.go:59):** "There is NO kernel rule about which branch a verb is legal on."

**`contradicted-by-code`**

- **Claim:** "**ANTI-0012** (epic → active with zero milestones legal): promote an
  epic with zero child milestones to `active`, confirm no
  `epic-active-no-drafted-milestones` finding."
- **Measured:** the finding **does** fire. After `$A add epic` + `$A promote
  E-0001 active` (exit 0, so the *transition* half of ANTI-0012 holds), `$A check
  --format=json` reported
  `warning | epic-active-no-drafted-milestones | epic E-0001 is active but has no milestones at status draft`.
  The rule's trigger is "no child at status draft", which zero milestones
  satisfies — `internal/check/epic_active_drafts.go:32-44`, and its own comment
  (lines 20-25) says "the trigger is 'zero drafts among children'". That logic is
  byte-identical to its first commit (`62ec2c2de`, 2026-05-11), which predates
  `antirules.go` (2026-05-19) — so ANTI-0012's second sentence was wrong at
  authoring, not drifted. The sketch as written would fail.
- **Command:** fixture: `$A add epic`, `$A promote E-0001 active`, `$A check --format=json`;
  then `cat -n internal/check/epic_active_drafts.go`,
  `git show 62ec2c2de:internal/check/epic_active_drafts.go`
- **Quoted (antirules.go:89):** "Distinct from the epic-active-no-drafted-milestones warning, which fires only when at least one milestone exists and ALL are drafts."

**`contradicted-by-code`**

- **Claim:** "**ANTI-0001** (no `≥1 AC` requirement): create a milestone with 0
  ACs, confirm `aiwf check` produces no `acs-required` finding."
- **Measured:** two problems.
  (1) `acs-required` is a finding code that **has never existed**. It appears
  nowhere in the repo except inside G-0161's own body, and `git log -S
  'acs-required'` returns exactly one commit — `9fdc8424b aiwf edit-body G-0161`.
  A test written to this sketch would pass vacuously.
  (2) The kernel now *does* police AC existence, under different names. Check
  time: a 0-AC draft milestone produces
  `warning | milestone-draft-incomplete-acs | draft milestone M-0001 has zero acceptance criteria; add them at plan time (aiwf add ac) so the contract is visible before the milestone lands on main`.
  Verb time: `requireNonEmptyACsAtMilestoneStart`
  (`internal/verb/promote.go:567-575`, landed `c6752d7f9` "feat(promote): refuse
  zero-AC milestone at draft->in_progress (M-0268/AC-1)") refuses
  draft→in_progress with "milestone has no acceptance criteria … or pass --force
  to override". ANTI-0001's Statement ("ACs are optional") is therefore narrower
  than it reads, and the sketch is the worst kind of oracle — one that stays
  green while its subject is breached.
- **Command:** fixture: `$A add milestone --epic E-0001 --tdd none --title …`,
  `$A check --format=json`, `$A promote M-0001 in_progress`; then
  `grep -rn 'acs-required' --include='*.go' --include='*.md' .`,
  `git log --oneline -S 'acs-required'`,
  `sed -n '545,580p' internal/verb/promote.go`
- **Quoted (antirules.go:23):** "A milestone is NOT required to have >=1 AC. ACs are optional."

### G-0168

**`contradicted-by-code`**

- **Claim:** The table row `| milestone | `tdd:` | `aiwf add milestone --tdd …` | **none** |`,
  and the surrounding sentence "Several frontmatter fields are set at entity-creation time
  … but have **no post-creation mutation verb**."
- **Measured:** `aiwf milestone tdd <M-id> --policy none|advisory|required [--reason …]`
  exists, is in the root `--help` banner, is tab-completable, and works. Driving it in the
  fixture: `M-0001` frontmatter went `tdd: advisory` → `tdd: none`, and the commit carried
  `aiwf-verb: milestone-tdd` / `aiwf-entity: M-0001` / `aiwf-actor: human/audit`. A re-run
  converged: `M-0001 tdd policy is already "none"; nothing to change`, exit 0. It landed in
  `3e1e350ff feat(milestone): add tdd policy-mutation verb (M-0277/AC-1)` (2026-07-24), under
  M-0277 (`done`) in E-0071 (`… milestone-tdd-policy-mutation-verb-g-0168`, archived). G-0168's
  last `edit-body` was 2026-07-22, two days before, so the body never heard.
- **Command:**
  `$AB __complete milestone ""` →  `depends-on`, `tdd`;
  `$AB milestone tdd --help`;
  `$AB milestone tdd M-0001 --root <fx1> --policy none --actor human/audit --reason "audit probe"` → exit 0;
  `git -C <fx1> log -1 --format='%s%n%b'`;
  `git log --oneline -S 'milestone tdd' -- cmd/aiwf/ internal/`
- **Quoted:** "| milestone | `tdd:` | `aiwf add milestone --tdd required\|advisory\|none` | **none** |"

**`dead-advice`**

- **Claim:** `## Workaround (current)` — "the operator hand-edits the YAML and commits
  manually with a descriptive but **fictional** `aiwf-verb:` trailer naming the field
  (e.g., `aiwf-verb: edit-frontmatter`, `aiwf-verb: retdd`)."
- **Measured:** That commit cannot be made in any repo with aiwf's hooks installed. The
  `commit-msg` hook (`# aiwf:commit-msg`, installed by `aiwf init`/`update`, present at
  `/workspaces/aiwf/.git/hooks/commit-msg`) `exec`s `aiwf check --commit-msg "$1"`, which
  refuses an `aiwf-verb:` value outside the live Cobra tree. Measured against a message
  carrying `aiwf-verb: edit-frontmatter`: exit **1**, with
  `aiwf check: commit-msg refuses aiwf-verb trailer value(s): ["edit-frontmatter"]`. The
  control (`aiwf-verb: milestone-tdd`) exits 0. Beyond the hook, `RunTrailerVerbUnknown`
  emits at `SeverityError` for any post-`HookInstallSHA` commit. The closed set is derived
  from the running binary's command tree (`internal/cli/check/verbs.go`), so there is no
  registry a fabricated name could be added to.
- **Command:**
  `printf 'chore(plan): …\n\naiwf-verb: edit-frontmatter\naiwf-entity: M-0001\naiwf-actor: human/audit\n' > /tmp/msg1.txt; $AB check --root <fx1> --commit-msg /tmp/msg1.txt; echo $?` → 1
  `… aiwf-verb: milestone-tdd … ; $AB check --root <fx1> --commit-msg /tmp/msg2.txt; echo $?` → 0
  `tail -5 /workspaces/aiwf/.git/hooks/commit-msg`
- **Quoted:** "Until the verbs exist, the operator hand-edits the YAML and commits manually
  with a descriptive but **fictional** `aiwf-verb:` trailer naming the field"

**`dead-premise`**

- **Claim:** `### Prerequisites for closing` — "**G-0285** (root `--help` banner drift) — the
  banner omits the entire `milestone` namespace today, so the new subverb would be invisible
  there until G-0285's chokepoint lands. This same omission is *why* the verb felt missing:
  `aiwf milestone depends-on` already ships but the banner hides it." and "**G-0284**
  (skill-coverage subverb blind spot) — namespace subverbs escape the skill-coverage policy
  today, so the new verb could ship uncovered."
- **Measured:** Both premises are false now, and both gaps are `addressed`. The root banner
  carries two `milestone` lines verbatim:
  `milestone depends-on <M-id> --on <id,id,...> | --clear …` and
  `milestone tdd <M-id> --policy <none|advisory|required> …`. `skill_coverage.go` carries
  `checkSubverbCoverage` — its own comment reads "checkSubverbCoverage is G-0284's fix:
  every runnable command reached through one or more namespace parents … G-0284: earlier
  this axis didn't exist". `aiwf milestone tdd` is documented in
  `internal/skills/embedded/aiwf-add/SKILL.md:166-167`.
- **Command:**
  `$AB --help | grep milestone`;
  `$AB show G-0284` → `addressed`; `$AB show G-0285` → `addressed`;
  `grep -n "subverb" internal/policies/skill_coverage.go`;
  `grep -rn 'aiwf milestone tdd' internal/skills/`
- **Quoted:** "the banner omits the entire `milestone` namespace today"

**`already-addressed`**

- **Claim:** `### The check-rule half has landed; a residual remains for the verb` — "The
  `tdd:` verb must instead **refuse with an actionable hint** naming the offending
  met-phaseless ACs, leaving the operator to resolve each deliberately."
- **Measured:** Shipped, exactly as specified. In the fixture, M-0002 (`tdd: none`) with
  `AC-1` at `status: met` and no `tdd_phase`, flipping to `required` returns **exit 2** with
  `aiwf milestone tdd: cannot set tdd: required — the following met AC(s) have no tdd_phase:
  done: AC-1; requiring TDD now would strand them. …`. `internal/verb/milestone_tdd.go`
  carries the guard under the comment "Refuse-with-hint: a flip to `required` must not strand
  an already-`met` AC that has no `tdd_phase: done`." The surrounding claims all measure true
  and are now descriptions of shipped code rather than requirements: an absent phase is legal
  until `met` (`acsShape` comment: "Absence is always legal (G-0286)"; G-0286 is `addressed`);
  a met-phaseless AC is `SeverityWarning` under `advisory` and `SeverityError` under `required`
  (`acsTDDAudit`, and observed as a warning in the fixture); the phase enum is
  `{"red","green","refactor","done"}` with no not-started member
  (`internal/entity/entity.go:184`); the sovereign tier is keyed on `(Kind, From, To)` status
  edges and holds one entry (`internal/entity/sovereign.go`), pinned by
  `TestSovereignActShapes_AllFSMLegal` (`internal/entity/sovereign_test.go:109`), and
  `milestone-tdd` is not in it.
- **Command:**
  `$AB add ac M-0002 --root <fx1> --title "…"`; `$AB promote M-0002/AC-1 met --root <fx1>`;
  `$AB milestone tdd M-0002 --root <fx1> --policy required --actor human/audit; echo $?` → 2;
  `$AB milestone tdd M-0002 --root <fx1> --policy advisory` then `$AB check --root <fx1> --verbose | grep tdd-audit` → warning;
  `sed -n '241,287p' internal/check/acs.go`; `sed -n '1,50p' internal/entity/sovereign.go`
- **Quoted:** "The `tdd:` verb must instead **refuse with an actionable hint** naming the
  offending met-phaseless ACs"

### G-0169

**`retired-verb-cited-present-tense`**

- **Claim:** "**Mutating, bespoke output path:** `aiwf import` (multi-entity import) and
  `aiwf rewidth` (id-width migration) emit their own multi-line/multi-commit output rather
  than routing through `FinishVerb`, so the uniform rollout did not reach them."
- **Measured:** Both halves are false at HEAD.
  `$A rewidth --help` → `aiwf: unknown command "rewidth" for "aiwf"`; `rewidth` does not
  appear in `$A __complete ""`'s 34-entry root command list (retired by ADR-0039, per the
  protocol appendix and confirmed here by its absence).
  `aiwf import` both **has** the flag and **does** route through the chokepoint:
  `$A import /nonexistent.yaml --format=json --root /tmp` →
  `{"tool":"aiwf",…,"status":"error","findings":[],"error":{"message":"reading manifest
  /nonexistent.yaml: open /nonexistent.yaml: no such file or directory"},"metadata":
  {"correlation_id":"e62f9fd1a75a609c"}}`, and `internal/cli/importcmd/importcmd.go` calls
  `cliutil.FinishVerbOutcome` at 11 sites (`:77 :83 :96 :122 :133 :137 :146 :156 :213`).
  The gap's own Notes section says as much; the "What's missing" list was never updated to
  match.
- **Command:** `$A rewidth --help` · `$A __complete "" | grep -i rewidth` ·
  `$A import /nonexistent.yaml --format=json --root /tmp` ·
  `grep -rn "FinishVerb\|OutputFormat" internal/cli/importcmd/*.go | grep -v _test`
- **Quoted:** as above.

**`contradicted-by-code`**

- **Claim:** "A CI consumer scripting `aiwf import --format=json` or `aiwf render roadmap
  --format=json` gets \"unknown flag\"."
- **Measured:** False for `import`, true for `render roadmap`.
  `$A import /nonexistent.yaml --format=json --root /tmp` → a well-formed envelope (above).
  `$A render roadmap --format=json` → `aiwf: unknown flag: --format`.
  `$A import --help` lists `--format string   output format: text or json (default "text")`
  and `--pretty`.
- **Command:** `$A import /nonexistent.yaml --format=json --root /tmp` ·
  `$A render roadmap --format=json` · `$A import --help | grep -i format`
- **Quoted:** as above.

### G-0212

**`already-shipped`**

- **Claim:** the six classes are catalogued "from history evidence +
  reasoning", each posed as an open question for a future epic to examine
  ("Under what verb-sequence combinations…?", "What happens to the scope's
  resolution?", "…are untested in combinatorial scenarios").
- **Measured:** M-0243 "Named scenarios from G-0212 and G-0269" is `done` under
  E-0062 with **all five ACs met** on 2026-07-09, and its ACs are titled by
  G-0212 item number. The scenarios exist and name the gap in their own source
  headers:

  | item | scenario file | what it found |
  |---|---|---|
  | 1 reallocate races | `internal/stresstest/parallel_branch_reallocate.go` | drives the full collision→`aiwf check`→`aiwf reallocate`→fast-forward-push sequence to clean resolution |
  | 2 edit-body races | `internal/stresstest/cross_worktree_edit_body_race.go` | **contradicts the gap** — see next finding |
  | 3 archive-during-scope | `internal/stresstest/archive_during_active_scope.go` | "G-0212 item 3's own literal fear … was already confirmed unfounded"; the probe surfaced G-0393 instead, now fixed |
  | 4 concurrent invocations | `internal/stresstest/concurrent_id_allocation.go` (M-0241/AC-2), `concurrent_writer_at_scale.go`, `cross_worktree_id_race.go`, `concurrent_milestone_race.go`, `concurrent_move.go` | real `aiwf` subprocesses raced against one working copy; repolock mutual exclusion asserted |
  | 5 force-push vs ack | `internal/stresstest/force_override_durability.go` | revival confirmed real → filed **G-0395**, since `addressed` via **D-0034** (`accepted`) + `findDanglingAckHint` (`internal/check/fsm_history_consistent.go:198`) |
  | 6 cherry-pick of force-amend | same file | confirmed, and recorded as the **current by-design trust model**, not a bug |

  A reader taking the body as current would re-scope work that shipped six
  weeks after the gap was filed. The gap body has not been touched since —
  its only two commits are the add (2026-06-02) and a `set-priority`
  (2026-07-17).
- **Command:** `grep -rn "G-0212" --include='*.go' --include='*.md' . | grep -v
  '^./work/gaps/G-0212' | grep -v '.claude/worktrees'` · `<binary> show M-0243`
  · `sed -n '30,100p'
  work/epics/archive/E-0062-correctness-stress-harness/M-0243-named-scenarios-from-g-0212-and-g-0269.md`
  · `sed -n '1,60p' internal/stresstest/archive_during_active_scope.go` ·
  `sed -n '1,60p' internal/stresstest/force_override_durability.go` ·
  `<binary> show G-0395; <binary> show D-0034; <binary> show G-0393` ·
  `git log --format='%h %ad %s' --date=short --follow -- work/gaps/G-0212-*.md`
- **Quoted:** "Known classes (from history evidence + reasoning):" … "The audit
  catalogs known + plausible scenarios, then drives a future epic to
  mechanically prevent each class."

**`contradicted-by-code`**

- **Claim (item 2):** "Two operators run `aiwf edit-body` on the same entity in
  different worktrees within minutes. **Last writer wins** per git's normal
  semantics, but the lost edits leave no audit trail."
- **Measured:** the scenario built to test exactly this states the opposite as
  an empirical result: "Empirically confirmed (manual git experiment, not a
  guess): merging one operator's branch into the other's **ALWAYS produces a
  genuine git conflict, never a silent last-writer-wins overwrite** —
  `edit-body` replaces the whole body field, so two different edits to the same
  field are, structurally, two changes to the same lines. This is a better
  outcome than G-0212 feared (maximally observable, not silent)." Its oracle
  asserts some trace of *both* operators' edits survives however the merge
  resolves. The premise "lost edits leave no audit trail" is therefore false as
  stated.
- **Command:** `sed -n '1,45p' internal/stresstest/cross_worktree_edit_body_race.go`
- **Quoted:** "Last writer wins per git's normal semantics, but the lost edits
  leave no audit trail."

**`contradicted-by-code`**

- **Claim (item 4):** "The repolock (`internal/repolock/`) serializes per-repo
  verb invocations **within a process**, but cross-process invocations on the
  same repo via subprocess fan-out are **untested** in combinatorial
  scenarios."
- **Measured:** both halves are false. (a) repolock is an **inter-process**
  lock by construction — `syscall.Flock(int(f.Fd()), syscall.LOCK_EX|LOCK_NB)`
  on `<root>/.git/aiwf.lock` (`internal/repolock/repolock_unix.go:72`), and the
  package doc says "serializes mutating aiwf invocations on the same
  repository … a POSIX advisory file lock (flock(2))"; its sentinel error text
  is literally "another aiwf **process** is running on this repo"
  (`repolock_unix.go:40`). The `internal/repolock` package predates the gap
  (present since the 2026-05-05 layout reorg `a13713213`), so this was already
  false at filing. (b) Cross-process invocations are now tested:
  `ConcurrentIDAllocationScenario` "launches n real `aiwf add <kind>`
  subprocesses against ONE working copy … and confirms repolock's mutual
  exclusion holds: no two attempts ever allocate the same id", plus four more
  subprocess-driving scenarios. Those all landed 2026-07-09 or later, i.e.
  after filing — so this half rotted rather than being born wrong.
- **Command:** `sed -n '1,30p;60,90p' internal/repolock/repolock_unix.go` ·
  `sed -n '1,40p' internal/stresstest/concurrent_id_allocation.go` ·
  `for f in concurrent_id_allocation.go concurrent_milestone_race.go
  concurrent_writer_at_scale.go cross_worktree_id_race.go
  parallel_branch_reallocate.go; do git log --diff-filter=A --format='%h %ad'
  --date=short -- internal/stresstest/$f | tail -1; done` ·
  `git log --diff-filter=A --format='%h %ad %s' --date=short -- 'internal/repolock/*.go' | tail -3`
- **Quoted:** "The repolock (`internal/repolock/`) serializes per-repo verb
  invocations within a process, but cross-process invocations on the same repo
  via subprocess fan-out are untested in combinatorial scenarios."

### G-0217

**`false-claim`**

- **Claim:** "The current message fires in BOTH cases with identical wording" — case 1
  being an `in_progress` milestone with the wrap ritual genuinely pending. The
  proposed-fix table repeats it, labelling the `in_progress` row "(current behavior)".
- **Measured:** An `in_progress` milestone whose branch is ahead of trunk emits **no
  label at all**. I built a fixture (epic E-0001 active, milestone M-0001 `in_progress`,
  worktree on `milestone/M-0001-test-milestone`, branch 2 commits ahead of `main`) and
  ran the audit binary: the worktree rendered as ordinary in-flight work —
  `→ M-0001 — Test milestone [in_progress]  (driven)` with its AC row and nothing else.
  Promoting AC-1 to `met` and M-0001 to `done` (branch then 4 ahead) produced
  `WRAP PENDING — driver done but branch ahead of trunk by 4 commits; merge to trunk
  before removing`. The source agrees: `renderStaleSection` is reached only when
  `v.Stale`, and `v.Stale = isTerminalStatus(e.Kind, e.Status)`
  (`internal/cli/status/worktrees.go:211`), whose set is
  done/cancelled/wontfix/rejected/addressed/retired/superseded/deprecated —
  `in_progress` is not in it. The only other route to `Stale`, `mergedStaleOverride`,
  requires `aheadOfTrunk == 0`, which the wrap-pending arm's `v.AheadOfTrunk > 0` guard
  excludes. This was already true when the gap was filed: G-0153, which introduced the
  stale-arm split, was `addressed` 2026-05-24, ten days before G-0217 (2026-06-03).
- **Command:** fixture under `…/audit/tmp/C2-3/wt`: `git init -b main`; `aiwf add epic/milestone/ac`; `aiwf edit-body`; `git worktree add -b milestone/M-0001-test-milestone`; `aiwf promote M-0001 in_progress --force --reason fixture`; `…/audit/aiwf status --worktrees` → no label; then `aiwf promote M-0001/AC-1 met`, `aiwf promote M-0001 done`; `…/audit/aiwf status --worktrees` → label. Source: `awk '/^func isTerminalStatus\(/,/^}/' internal/cli/status/worktrees.go`
- **Quoted:** "The current message fires in BOTH cases with identical wording, and the phrase \"WRAP PENDING\" strongly suggests case 1 (operator forgot to wrap)."

### G-0254

**`dead-premise`**

- **Claim:** "**`aiwf check` rule (pre-push + CI)** — the authoritative
  chokepoint… **CI's `aiwf check` always runs**, catching `--no-verify` bypass
  and uninstalled-hook clones." and, in Fix shape, "the `aiwf check` rule is the
  load-bearing one".
- **Measured:** **no workflow invokes `aiwf check`, and none ever has.**
  `grep -rn "aiwf check" .github/workflows/ ` returns nothing;
  `git log --oneline -S 'aiwf check' -- .github/workflows/` returns **zero
  commits**, so the string has never appeared in a workflow file. The only aiwf
  invocation in CI is `make selfcheck` → `aiwf doctor --self-check`
  (`.github/workflows/go.yml:326`), which drives verbs against a *temporary*
  repo, not this tree. `make ci` is `vet lint test-cov coverage-gate-only
  selfcheck` (`Makefile:268`). Independently corroborated by G-0536 (`open`,
  priority `high`), whose §"What's missing" states the same and is itself in
  this TODO cluster. Consequence: an `aiwf check` rule for `Co-Authored-By`
  would be pre-push-only and `--no-verify`-bypassable — the exact weakness the
  body assigns to the `commit-msg` hook while calling the check rule "the
  always-on guarantee".
- **Command:** `grep -rn "aiwf check" .github/workflows/ Makefile`;
  `git log --oneline -S 'aiwf check' -- .github/workflows/`;
  `grep -n "name: \|run: " .github/workflows/go.yml`;
  `grep -n -A10 '^ci:' Makefile`;
  `sed -n '1,40p' work/gaps/G-0536-aiwf-check-has-no-ci-backstop-the-primary-oracle-is-per-clone-opt-in.md`
- **Quoted:** body lines 78–81 and 91–92; against G-0536:9-12 "No workflow under
  `.github/workflows/` invokes it: `make ci` is `vet lint test-cov
  coverage-gate-only selfcheck`, and `selfcheck` drives `aiwf doctor
  --self-check` against a temporary repo rather than against this one's planning
  tree."

### G-0282

**`dead-premise`**

- **Claim:** the gated-annotation extension "is not independently buildable yet
  … that toggle verb does not exist: `tdd:` is currently set-once at `aiwf add
  --tdd` (`internal/cli/add/add.go`) with no post-create mutation verb".
- **Measured:** the verb exists. `aiwf milestone tdd --help` →
  "Set a milestone's TDD policy after creation… `--policy string  the TDD policy
  to set: none | advisory | required`". It landed 2026-07-24 in `3e1e350ff
  feat(milestone): add tdd policy-mutation verb (M-0277/AC-1)` — three days
  after this gap's last edit (2026-07-21). `M-0277` is `done` under `E-0071`.
  Better still for the gap's purpose: the case it wanted to pin is now live and
  observable. In a disposable fixture, downgrading a `tdd: required` milestone
  with no `--reason` and no `--force` succeeds at exit 0:
  `aiwf milestone tdd M-0001 -> advisory`, frontmatter `tdd: advisory`, trailers
  `aiwf-verb: milestone-tdd` / `aiwf-entity: M-0001` / `aiwf-actor: human/auditor`.
  So the extension is unblocked, not blocked.
- **Command:**
  ```
  <audit>/aiwf milestone tdd --help
  git log --oneline --diff-filter=A -- internal/verb/milestone_tdd.go   # 3e1e350ff
  git show -s --format='%H %ad %s' --date=short 3e1e350ff               # 2026-07-24
  # disposable fixture $W/fix1, HOME redirected
  <audit>/aiwf add milestone --epic E-0001 --title "Fixture milestone" --tdd required --actor human/auditor
  <audit>/aiwf milestone tdd M-0001 --policy advisory --actor human/auditor   # exit 0
  ```
- **Quoted:** "The `gated`-annotation extension proposed above is not
  independently buildable yet. It is motivated by the `tdd: required → advisory`
  toggle, and that toggle verb does not exist".

**`contradicted-by-record`**

- **Claim:** the `required → advisory` downgrade is "exactly the
  integrity-loosening act aiwf routes through a sovereign `--reason` (and human
  actor) elsewhere", so "a verb can have a perfectly *present* inverse that is
  nonetheless **governance-weighted in one direction**", and "A `tdd:` policy
  verb that lets `required → advisory` through without `--reason` would then be
  either mis-declared or caught mechanically."
- **Measured:** G-0168 — the sibling gap this section itself names — settled the
  opposite, in a section dated the same day (2026-06-26) but whose text landed
  `2026-07-22` in `90cb4702c aiwf edit-body G-0168`, one day after G-0282's last
  edit. Its heading is literally "### tdd: is a uniform ordinary mutator —
  direction carries no gating". Its closing line: *"The undo of 'set tdd
  required' is simply 'set tdd advisory' — a plain self-inverse, not a
  governance-weighted one."* The shipped verb implements that: no `--force`, an
  optional `--reason`, and (measured above) a clean exit-0 downgrade with
  neither. So the gap's motivating third instance is a decided non-instance, and
  a registry asserting a `gated` annotation on it would go red against a
  deliberate design.
- **Command:**
  ```
  sed -n '205,250p' work/gaps/G-0168-kernel-lacks-mutation-verbs-for-set-at-create-frontmatter-fields.md
  git log --oneline -S 'not a governance-weighted one' -- work/gaps/G-0168-*.md   # 90cb4702c
  git log -1 --format='%H %ad %s' --date=short 90cb4702c                          # 2026-07-22
  ```
- **Quoted (G-0282):** "the downgrade *weakens* it — exactly the
  integrity-loosening act aiwf routes through a sovereign `--reason` (and human
  actor) elsewhere (`acknowledge-illegal`, `--force`)."
- **Quoted (G-0168, the record that carries the opposite):** "that semantic
  difference does **not** warrant a mechanical gating difference. `aiwf
  milestone tdd` is a plain frontmatter mutator, treated exactly like `milestone
  depends-on`, `set-priority`, and `set-area`: any actor …, an optional
  `--reason`, standard trailers, and no directional or sovereign carve-out
  either way."

**`retired-verb`**

- **Claim (twice):** "`rewidth` is one-way but does NOT say so in `--help` — the
  policy would flag it", and — load-bearing for scheduling — "a concrete seed
  fixture already on hand: `rewidth` is one-way but its `--help`/Long text does
  not say so (unlike `archive`, which cites ADR-0004), a real present-day
  violation the base registry would catch immediately."
- **Measured:** `rewidth` no longer exists. `<audit>/aiwf rewidth --help` →
  `aiwf: unknown command "rewidth" for "aiwf"`. `internal/verb/rewidth.go` and
  `internal/cli/rewidth/` were deleted in `db307fdc7` (2026-08-03,
  M-0290/AC-1+AC-2), and `internal/cli/root_test.go:162` now pins
  `const retired = "rewidth"`. The governing record is **ADR-0039 (accepted)**,
  "Retire the rewidth verb; ADR-0008's migration clauses lapse", whose Decision
  section reads: "Retire the verb: `internal/verb/rewidth.go`,
  `internal/cli/rewidth/`, `padToCanonical`, and the command's registration are
  deleted." The gap's only named present-day violation, and therefore its stated
  reason the base registry is worth scheduling now, is gone.
- **Command:**
  ```
  <audit>/aiwf rewidth --help
  grep -rn 'rewidth' cmd internal | head
  git log --oneline --diff-filter=D -- internal/verb/rewidth.go    # db307fdc7
  git show -s --format='%H %ad %s' --date=short db307fdc7          # 2026-08-03
  sed -n '1,40p' docs/adr/ADR-0039-retire-the-rewidth-verb-adr-0008-s-migration-clauses-lapse.md
  ```
- **Quoted:** "a concrete seed fixture already on hand: `rewidth` is one-way but
  its `--help`/Long text does not say so … a real present-day violation the base
  registry would catch immediately."

### G-0311

**`false-claim`**

- **Claim:** *"the kernel forces it into three separate epics wired by `depends_on`, with no entity that names 'the subtitle feature.'"*
- **Measured:** epics cannot be wired by `depends_on` at all. `<aiwf> schema epic` prints `optional fields: (none)` / `reference fields: (none)`. `entity.ForwardRefs`' default arm (`internal/entity/refs.go:63-66`) states *"KindEpic and any future kind without outbound refs falls through to an empty list"*. In a fixture, an epic hand-carrying `depends_on: [ADR-0001]` drew **zero** findings and `<aiwf> show E-0001 --format=json` carried no `depends_on` key — accepted, stored, read by nothing. `<aiwf> epic depends-on E-0001 --on E-0002` → `aiwf: unknown command "epic" for "aiwf"`. The real fallback is unwired sibling epics plus prose, which is worse than the sentence claims — the gap's case survives and strengthens, but the sentence tells a reader a mechanism exists that does not.
- **Command:** `<aiwf> schema epic`; fixture with `depends_on` on an epic; `<aiwf> check --root $S --format=json`; `<aiwf> show E-0001 --root $S --format=json`; `<aiwf> epic depends-on E-0001 --on E-0002 --root $S`; `sed -n '59,69p' internal/entity/refs.go`
- **Quoted:** as above.

### G-0328

**`contradicted-by-code`**

- **Claim:** "There is no **standing** test that reproduces the byte-identity
  claim mechanically."
- **Measured:** three standing, untagged tests do exactly this, in
  `internal/cli/integration/check_summary_binary_test.go`:
  `TestBinary_CheckVerbose_ByteIdenticalToBaseline` (line 55, raw byte equality
  on text output), `TestBinary_CheckJSON_ByteIdenticalToBaseline` (line 89) and
  `TestBinary_CheckJSONPretty_ByteIdenticalToBaseline` (line 119). They build the
  real binary, run it against a frozen synthetic `testdata/` fixture
  (`internal/check/testdata/messy`, copied to a tempdir), and compare against
  goldens under `internal/cli/integration/testdata/m0089/` captured from the
  pre-M-0089 binary at SHA `5523e99`. No `//go:build` tag, so they are in the
  ordinary `go test ./...` path. They landed 2026-05-18 … 2026-05-20 —
  **before** G-0328 was filed on 2026-06-30 — so the sentence was already too
  broad when written.
  I reproduced the comparison with the audit binary rather than running the test:
  25 findings, findings array JSON-equal to `testdata/m0089/json.golden`,
  metadata equal modulo `root`.
  **What genuinely is missing:** the fixture carries no `.git`
  (`find internal/check/testdata/messy -maxdepth 2 -name '.git*'` → nothing; the
  test's own header says the copy exists "so the test runs outside any .git
  context (provenance audit and trunk-collision checks are skipped
  accordingly)"). Consistently, the golden's 25 findings carry only tree-shape
  codes — `adr-supersession-mutual`, `archive-sweep-pending`, `body-prose-id`,
  `epic-active-no-drafted-milestones`, `frontmatter-shape`,
  `gap-addressed-has-resolver`, `id-path-consistent`, `ids-unique`,
  `milestone-draft-incomplete-acs`, `no-cycles`, `refs-resolve`, `status-valid`,
  `terminal-entity-not-archived`, `titles-nonempty` — and **no** history-walking
  code at all. So the existing comparator cannot see the rules M-0216 changed.
- **Command:**
  `grep -rn 'byte-identical\|ByteIdentical' --include='*_test.go' internal/`;
  `sed -n '1,140p' internal/cli/integration/check_summary_binary_test.go`;
  `cp -r internal/check/testdata/messy $S && $A check --format=json --root $S > actual.json`
  then a python `json.dumps(..., sort_keys=True)` comparison against
  `internal/cli/integration/testdata/m0089/json.golden`;
  `git log --oneline --date=short -- internal/cli/integration/check_summary_binary_test.go`
- **Quoted (body):** "There is no **standing** test that reproduces the byte-identity claim mechanically."

### G-0333

**`false-absence-claim`**

- **Claim:** The Tier-1/Tier-2 boundary "is stated in no AI-discoverable channel
  (CLAUDE.md, `docs/design/provenance-model.md`, or `--force --help`), yet it
  determines what a sovereign override can and cannot do."
- **Measured:** False, and false at the moment of filing. Six surfaces carry it;
  five predate the gap's `add` commit `f099ba43e` (2026-07-01 15:01:56 +0000):
  1. **`aiwf promote --help`** — the channel the gap names. At filing,
     `internal/cli/promote/promote.go:72` read *"skip the FSM transition rule
     (requires --reason); coherence checks still run"*. Today (`:70`) it reads
     *"…; sovereign, so the actor must be human/… ; coherence checks still run and
     the standing audit keeps reporting"*.
  2. **`internal/skills/embedded/aiwf-promote/SKILL.md:45`** (since `1f2cd91da`,
     2026-05-02): *"`--force` relaxes only the FSM transition rule — coherence
     checks (status in closed set, refs resolve, AC body coherence) still run."*
  3. **`internal/skills/embedded-rituals/…/wf-tdd-cycle/SKILL.md:116`** (since
     `cc949954b`, 2026-07-01 14:57:56 +0000 — four minutes before the gap's own
     `add` commit): *"Force relaxes only the status/phase FSM *transition* check;
     the audit runs as a projection finding **regardless of `--force`**, so there
     is no `--force met` shortcut."* This is the very M-0199 correction the gap
     itself credits two paragraphs earlier.
  4. **`docs/design/design-decisions.md:148`** (Normative tier): *"Force relaxes
     only the *transition* rule; coherence checks (id format, closed-set
     membership, ref resolution) still run."*
  5. **`docs/design/legal-workflows-first-principles.md:370`**, R-FP-0145: *"Force
     relaxes only the transition rule, not the coherence rules."*
  6. **`internal/check/hint.go`**, `forceCaveatSentence` (landed `527caf70d`,
     2026-08-05 under M-0293/AC-2 — one day *after* this body's last edit):
     *"`--force` overrides only the precondition this finding names, not the
     finding itself: it is sovereign, so it requires a `human/...` actor, and
     every other check still runs."* `HintFor` appends it to every hint offering
     `--force --reason`. I observed it live in verb output (see the isolation
     measurement below).

  Only two of the gap's three named channels hold up: `CLAUDE.md` mentions
  `--force` five times and never states the boundary (lines 16, 25, 32, 67, 220),
  and `docs/design/provenance-model.md` §"The `--force` rule" (:126-134) is
  entirely about the human-only constraint.
- **Command:**
  `git show f099ba43e:internal/cli/promote/promote.go | grep -n 'force", false'`;
  `…/audit/aiwf promote --help | grep -- --force`;
  `sed -n '45p;47p' internal/skills/embedded/aiwf-promote/SKILL.md`;
  `sed -n '116p' internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-tdd-cycle/SKILL.md`;
  `git log --format='%h %ad %s' --date=iso -S 'does not get you around it' -- internal/skills/`;
  `git log -1 --format='%h %ad' --date=iso f099ba43e`;
  `grep -rn 'relaxes only' docs/`;
  `sed -n '488,505p' internal/check/hint.go`;
  `grep -n -i force CLAUDE.md docs/design/provenance-model.md`
- **Quoted:** "The boundary — \"`--force` overrides local FSM / preconditions, not tree-invariant error findings\" — is stated in no AI-discoverable channel (CLAUDE.md, `docs/design/provenance-model.md`, or `--force --help`), yet it determines what a sovereign override can and cannot do. This violates the kernel's \"kernel functionality must be AI-discoverable\" principle." — and its restatement lower down, "What remains here is the boundary itself: a reader who wants to know which rules `--force` overrides has no document to consult, and infers it from whichever hint they happen to read." (body lines 40-42). Both sentences carry the same false absence, so an `edit-body` must reach both.

### G-0370

**`dead-premise`**

- **Claim:** "Needs a hand-written pinning test under `internal/policies/` (the
  `skill-edit-structural-test-backstop` mechanical gate is scoped to
  `*/SKILL.md` paths only, so it will not fire automatically on this file —
  follow the same manual-pinning convention already used for other
  non-`SKILL.md` ritual content…)."
- **Measured:** The named gate no longer exists.
  `internal/policies/skill_edit_structural_test_backstop.go` was deleted by
  `667c8e0fc` ("feat(policies): ask a skill edit who owns it, not whether a test
  names it (M-0312)"). Its only surviving mention in Go source is
  `internal/policies/skill_edit_provenance_backstop_test.go:386`, in a list of
  *retired* signatures the successor test asserts must be **absent** from
  CLAUDE.md. Worse for the instruction: the manual-pinning convention it points
  at was itself retired one day later. D-0070 (`accepted`) retires
  prose-presence assertions over shipped surfaces, and
  `internal/policies/shipped_prose_assertion.go:23-27` lists
  `"embedded-guidance"` in `shippedSurfaceMarkers` — the guidance fragment is in
  scope. `namesShippedPath` / `exprNamesShippedPath` (`:790-810`) fire on any
  test whose `ReadFile` path literal contains that marker and which then
  compares the bytes against a test-authored phrase, which is precisely the test
  G-0370 asks for. The exemption class that could rescue it,
  `triggerPhraseExemptions`
  (`internal/policies/shipped_prose_assertion_allowlist.go:17-38`), is scoped in
  its own doc comment to "those in a skill's `## When to use` section and its
  `description:` frontmatter" — neither of which the always-on fragment is.
  The repo's own record confirms this is live and unsettled: M-0312's spec says
  "G-0370 was left alone deliberately. Its instruction looks like the same case,
  but the content it pins is dispatch trigger phrases, which D-0070 preserves —
  and whether that exception reaches the always-on guidance fragment, as opposed
  to a skill's own `## When to use`, is a question D-0070 does not answer."
- **Command:**
  `grep -rn "skill-edit-structural-test-backstop" . --exclude-dir=.git` ;
  `git log --oneline --diff-filter=D -- internal/policies/skill_edit_structural_test_backstop.go` ;
  `sed -n '15,30p' internal/policies/shipped_prose_assertion.go` ;
  `sed -n '785,812p' internal/policies/shipped_prose_assertion.go` ;
  `cat internal/policies/shipped_prose_assertion_allowlist.go` ;
  `grep -n -A6 "G-0370" work/epics/archive/E-0087-…/M-0312-…md` ;
  `$A show D-0070` (→ `accepted`)
- **Quoted:** from `internal/policies/shipped_prose_assertion_allowlist.go:17-19`
  — "triggerPhraseExemptions cover the phrasings that decide whether an
  assistant reaches for a skill at all — those in a skill's `## When to use`
  section and its `description:` frontmatter."

### G-0398

**`dead-premise`**

- **Claim:** "Discovered indirectly: `internal/stresstest/verb_sequence.go`'s
  `TestVerbSequenceScenario_FullWalkAcrossAllKindsPasses` property test creates one entity per
  kind and walks each one's status independently. Because it creates and fully random-walks the
  epic (which can land it on `done`/`cancelled`) *before* creating the milestone that needs the
  epic as `--epic` parent, the test occasionally hits this exact refusal by accident and treats it
  as a hard scenario failure."
- **Measured:** The ordering was inverted four days after this gap was written. `Run` in
  `internal/stresstest/verb_sequence.go:112-143` now creates the milestone *immediately after* the
  epic and *before* any walk step, and the code comment at lines 133-139 states the consequence in
  terms of this very gap: *"this add can no longer trip the epic-terminal-non-terminal-children
  refusal G-0398 describes, because the epic it names has had no opportunity yet to reach a
  terminal or archived status."* The change is `e2efb07c1` *"fix(stresstest): create the
  verb-sequence walker's milestone right after its epic (G-0401)"* (2026-07-14); `aiwf show G-0401`
  reports `addressed` and archived. A secondary reference slip: the named test does not live in
  the named file — `TestVerbSequenceScenario_FullWalkAcrossAllKindsPasses` is at
  `internal/stresstest/verb_sequence_test.go:632`, while `verb_sequence.go` holds `Run`.
- **Command:**
  ```
  sed -n '95,160p' internal/stresstest/verb_sequence.go
  grep -rn "TestVerbSequenceScenario_FullWalkAcrossAllKindsPasses" internal/stresstest/
  git log --format='%h %ad %s' --date=short -S 'can no longer trip the epic-terminal-non-terminal-' -- internal/stresstest/verb_sequence.go
  aiwf show G-0401
  ```
- **Quoted:** `verb_sequence.go:99-102`: *"the milestone is created immediately after the epic —
  while the epic is still freshly "proposed", guaranteed non-terminal — and only then are both
  walked, rather than "walk the epic to completion, then create the milestone" the way
  entity.AllKinds() order would otherwise suggest."*

### G-0412

**`stale-claim`**

- **Claim:** the wording "has since been copied verbatim into `internal/cli/renamearea`,
  `internal/cli/setarea`, and every M-0252/M-0253 file that ignores this branch (add, cancel,
  promote, reallocate, rename, retitle, milestone, update, editbody)".
- **Measured:** 11 of the 12 named files carry neither the string nor a `ResolveRoot(` call
  today. Only `internal/cli/update/update.go` still does. `be463fdcf` (M-0279/AC-2,
  2026-07-25, "route verb preludes through shared prelude helpers") collapsed the inline
  prelude in 21 files — including archive, renamearea, setarea, add, cancel, promote,
  reallocate, rename, retitle, milestone, editbody — onto `cliutil.ResolvePrelude` /
  `ResolvePreludeEnvelope`. A reader following the body's list would find nine dead ends.
- **Command:**
  `for f in archive/archive.go renamearea/renamearea.go setarea/setarea.go add/add.go cancel/cancel.go promote/promote.go reallocate/reallocate.go rename/rename.go retitle/retitle.go milestone/milestone.go update/update.go editbody/editbody.go; do p="internal/cli/$f"; echo "$p : exact=$(grep -c 'ResolveRoot only fails on missing aiwf.yaml' $p) : calls=$(grep -c 'ResolveRoot(' $p)"; done`
  → every line `exact=0 : calls=0` except `internal/cli/update/update.go : exact=1 : calls=1`;
  `git show --stat be463fdcf`
- **Quoted:** "has since been copied verbatim into `internal/cli/renamearea`,
  `internal/cli/setarea`, and every M-0252/M-0253 file that ignores this branch"

### G-0442

**`contradicted-by-code`**

- **Claim:** "There is no verb to **amend, add to, or clear** either field afterward
  — to credit a second resolver on an already-addressed gap, or correct a
  `superseded_by` pointer, an operator must hand-edit the YAML and commit manually."
  Reinforced by the table's "Amend / clear afterward? — **none**" for both rows and
  by "Each is written once, at the `open → addressed` / `accepted → superseded` step."
- **Measured:** three of the four sub-claims are false.
  - *Backfill, unforced.* `…/aiwf promote G-0003 addressed --force --reason "…"`
    (no resolver) left `addressed_by` absent; then
    `…/aiwf promote G-0003 addressed --by E-0001` — **no `--force`** — printed
    `aiwf promote G-0003 addressed -> addressed`, exit 0, and the file gained
    `addressed_by:\n    - E-0001`. A post-transition write to the field, through a
    verb, with no sovereign act.
  - *Add to.* `…/aiwf promote G-0003 addressed --by E-0001,E-0002 --force --reason "audit: credit a second resolver"`
    → exit 0, file now carries `addressed_by:\n    - E-0001\n    - E-0002`. That is
    exactly the body's own example — "credit a second resolver on an already-addressed
    gap" — done by a verb.
  - *Correct a `superseded_by` pointer.*
    `…/aiwf promote ADR-0001 superseded --superseded-by ADR-0003 --force --reason "audit: correct the supersession pointer"`
    → exit 0, `superseded_by: ADR-0002` became `superseded_by: ADR-0003`.
  - *Clear.* This one holds. `--by ""` and `--superseded-by ""` both converge:
    `G-0003 is already addressed; nothing to change` / `ADR-0001 is already
    superseded; nothing to change`, exit 0, field unchanged. `--clear` is rejected:
    `aiwf: unknown flag: --clear`. `…/aiwf help` lists no verb that touches either
    field.

  The verb states its own position explicitly when asked to re-point unforced:
  `aiwf promote: G-0001 is already addressed and already carries a resolver; this
  verb backfills an empty resolver, it does not re-point one — pass --force --reason
  to override`. Source: `internal/verb/promote.go:105–118` (the re-point refusal) and
  `:119–131` (`isBackfill`, the G-0096 carve-out).
- **Command:** the sequence above, in `…/tmp/C5-2/g0442` (a `git init` + `aiwf init`
  fixture with G-0001..G-0004, E-0001..E-0002, ADR-0001..ADR-0004)
- **Quoted:** body — "There is no verb to **amend, add to, or clear** either field
  afterward"; verb — "this verb backfills an empty resolver, it does not re-point
  one — pass --force --reason to override"

### G-0444

**`contradicted-by-code`**

- **Claim:** "`readHistoryChain` survives only in test comments under `internal/cli/integration/` (e.g. `canonicalize_history_test.go`, `show_cmd_test.go`), **not in production code**; the chain logic now lives inline in `history.go`." Restated in "Why it matters" ("the chain logic is inline") and made operative in "Suggested approach" ("describe the inline `PriorIDs`-chain handling rather than the **retired** `readHistoryChain`").
- **Measured:** `internal/entityview/historyevent.go:117` declares `func ReadHistoryChain(ctx context.Context, root string, chain []string) ([]HistoryEvent, error)` — production code, not a test file — and `internal/cli/history/history.go:100` calls it (`events, err := entityview.ReadHistoryChain(ctx, rootDir, chain)`). It landed in `5d331e61d` (2026-07-20, "feat(entityview): extract read-side helpers into a neutral package (M-0272)"), three days before G-0444 was filed (`beb1bc399`, 2026-07-23), so the claim was already false at filing. What *is* inline in `Run` is the chain **assembly** (lines 85–98: seed with the queried id, resolve via `ResolveByCurrentOrPriorID`, append `PriorIDs` and the canonical id) — not the "greps the union in one pass" step that `id-allocation.md:164` attributes to `readHistoryChain`, which is precisely what `ReadHistoryChain` still does. The doc line's *description* is therefore still accurate; only the spelling and the owning package moved.
- **Command:** `grep -rn '\breadHistoryChain\b' . ; grep -rn 'func .*History' --include='*.go' internal/ | grep -v _test; sed -n '100,150p' internal/entityview/historyevent.go; sed -n '76,102p' internal/cli/history/history.go; git log -S 'func ReadHistoryChain' --format='%h %ad %s' --date=short -- internal/entityview/`
- **Quoted:** `internal/entityview/historyevent.go:110–117` — "// ReadHistoryChain is ReadHistory's lineage-aware variant: it greps / git log for any aiwf-entity / aiwf-prior-entity trailer matching / any id in chain, dedupes by commit SHA, and returns a single / oldest-first chronological slice. Used by `aiwf history <id>` …"

### G-0445

**`contradicted-by-code`**

- **Claim:** "`docs/` is not an aiwf-managed path: in a consumer repo `docs/` may hold real shippable implementation."
- **Measured:** `docs/adr/` is an aiwf-managed path in *every* consumer. `$A --root <fx> add adr --title "Some architectural choice" --body …` created `docs/adr/ADR-0001-some-architectural-choice.md` and committed it (`git show --stat` names that single file). The loader walks `docs/adr` (`internal/tree/tree.go:288`), `aiwf add` routes ADRs there (`internal/verb/add.go:490`), `aiwf init` creates it (`internal/initrepo/initrepo.go:686`), `aiwf archive` sweeps to `docs/adr/archive/` (`internal/verb/archive.go:428`), and `internal/trunk/trunk.go:88,239` reads trunk state from `"work/"` **and** `"docs/adr/"`. The consequence is load-bearing for the gap's own remedy list: I measured that a dirty `docs/adr/ADR-0001-….md` alongside a dirty test path passes `--phase red` today (exit 0), and that any *non-excluded* dirty non-test path refuses (control: `src/component.jsx` → "refusing — red-first requires the test to change before the implementation … : src/component.jsx"). `isPlanningPath` reaches `docs/adr/` only via the `docs/` prefix (4-line function, read at `promote_phase_gate.go:78-85`), so the gap's option 2 — exclude `work/` only — would make every mid-edit ADR false-refuse a legitimate red promote.
- **Command:** `$A --root <fx> add adr --title "Some architectural choice" --body "Context, decision, consequences." --actor human/auditor`; `git -C <fx> log -1 --stat`; `$A --root <fx> promote M-0001/AC-6 --phase red --actor human/auditor` with `docs/adr/ADR-0001-….md` modified and `foo_test.go` untracked; `grep -rn '"docs' internal/ --include='*.go' | grep -v _test`
- **Quoted:** body — "But `docs/` is not an aiwf-managed path"; and "Options: … scope the exclusion to the entity tree only (`work/`) …"

### G-0448

**`contradicted-by-code`**

- **Claim:** "split by an accident of function signature rather than a declared boundary" / "Whether a rule lands in surface one or surface two is decided purely by 'does it need git/ctx/config' — not by any intentional layering."
- **Measured:** the boundary is declared in three independent places and is load-bearing.
  1. `internal/check` imports **no** config package: `grep -rn "aiwf/internal/config\|aiwf/internal/aiwfyaml" internal/check/*.go | grep -v _test` returns nothing. The CLI projects `aiwf.yaml` members into a config-agnostic `check.AreaPaths` with the comment "so the path-axis rules (dead-glob) stay free of any aiwf.yaml type, **the M-0171/AC-4 boundary**" (`internal/cli/check/check.go:196-198`). `aiwf show M-0171/AC-4` → status `met`, phase `done`.
  2. The purity is what two other surfaces are built on. `runShapeOnly` (the pre-commit hook path) runs one rule "because the full check.Run pipeline (trunk read, provenance walk, contract validation) is too slow and too noisy to fire on every commit" (`internal/cli/check/check.go:421-427`); `runFast` (the statusline health glyph) calls its two extra rules "pure in-memory passes, so they stay render-safe" (:379-381). `FSMHistoryConsistent` carries the same rationale explicitly: "Lives in the CLI layer rather than check.Run because the per-entity git walk is too expensive for the pre-commit hook's shape-only policy path" (:170-175).
  3. `PolicyLayeringDirection` (`internal/policies/layering_direction.go`) enforces the import direction `cmd → cli → verb → check/render/…` in CI.
- **Command:** `grep -rn "aiwf/internal/config\|aiwf/internal/aiwfyaml" internal/check/*.go` ; `sed -n '160,300p;330,400p;415,471p' internal/cli/check/check.go` ; `aiwf show M-0171/AC-4` ; `sed -n '1,60p' internal/policies/layering_direction.go`
- **Quoted:** "Whether a rule lands in surface one or surface two is decided purely by 'does it need git/ctx/config' — not by any intentional layering."

**`stale-count`**

- **Claim:** "Rules are dispatched from two parallel surfaces" (also in the title), with the git/history/area rules "individually invoked and appended in `internal/cli/check/check.go` (~lines 127-305)".
- **Measured:** four rule-invoking sites, and the largest git/history group is in a file the gap never names.
  - `internal/check/check.go:128` `Run` — 36 `append` statements at lines **130–181**.
  - `internal/cli/check/check.go:71` `Run` — appends at 113, 167, 177, 212, 217, 219, 225, 232, 239, 246, 255, 275, 283, 284.
  - **`internal/cli/check/provenance.go` `RunProvenanceCheck` — seven rules individually invoked at lines 67, 99, 107, 182, 208, 223, 250** (`RunProvenance`, `RunIsolationEscape`, `RunOrphanedAICommits`, `RunPromoteOnWrongBranch`, `RunIDRenameUntrailered`, `RunUntrailedAudit`, `RunTrailerVerbUnknown`). This file is the *only* non-test caller of each of those seven.
  - `internal/cli/check/check.go:352` `runFast` — `check.Run` plus appends at 382–383; `internal/cli/check/check.go:432` `runShapeOnly` — `check.TreeDiscipline` alone.
  Neither omission is drift: `internal/cli/check/provenance.go` was created 2026-05-18 (`108f38836`) and `runFast` 2026-06-28 (`5fd867d81`); `aiwf history G-0448` dates the gap to 2026-07-24.
- **Command:** `grep -n "findings = append(findings" internal/cli/check/check.go internal/check/check.go` ; `grep -rn "RunIDRenameUntrailered\|RunIsolationEscape\|RunPromoteOnWrongBranch\|RunProvenance(\|RunUntrailedAudit\|RunTrailerVerbUnknown\|RunOrphanedAICommits" --include="*.go" internal/ | grep -v _test | grep -v "^internal/check/"` ; `git log --oneline --diff-filter=A -- internal/cli/check/provenance.go`
- **Quoted:** "Git/history/area rules … are individually invoked and appended in `internal/cli/check/check.go` (~lines 127-305)."
- **Why it matters for a reader:** *Where to fix* names only `internal/check/check.go` and `internal/cli/check/check.go`. Converging those two and stopping leaves seven provenance rules outside the registry — producing exactly the half-wired state the gap's second bullet warns about.

### G-0472

**`dead-premise`**

- **Claim:** "The hook installers carry a cost of a different kind, and it is the
  largest thing on this page: their four members disagree about what a read fault
  means, three of them destroying a user's unreadable hook and reporting success.
  That is a wrong-output failure mode … it is tracked as G-0557, and it is the
  work this family actually wants." Related repeats it: "G-0557 — the hook
  installers' read-fault divergence: a live data-loss path".
- **Measured:** All four installers now refuse. `ensurePreHook:1350` has
  `case readErr != nil: return …fmt.Errorf("reading pre-push hook: %w", readErr)`;
  `ensurePreCommitHook:1454`, `ensureCommitMsgHook:1535` and
  `ensurePostCommitHook:1622` each carry
  `if readErr != nil && !errors.Is(readErr, fs.ErrNotExist)` returning a wrapped
  error, under comments citing G-0557 ("Refuse instead (G-0557)" ×2, "Refuse on
  both routes (G-0557)"). `G-0557` is `status: addressed`, archived, closed by
  `ae918ff9e` (2026-08-06) — "fix(init): refuse a git hook aiwf cannot read
  instead of overwriting it", whose body reads "All four now refuse an unreadable
  hook, the contract ensurePreHook already had." G-0472's last `edit-body` was
  2026-08-05, one day before.
- **Command:** `git log --oneline -S 'Refuse instead (G-0557)' -- internal/initrepo/initrepo.go` ·
  `git show --stat ae918ff9e` · `…/audit/aiwf show G-0557` ·
  per-installer `sed -n` over `internal/initrepo/initrepo.go`
- **Quoted:** "three of them destroying a user's unreadable hook and reporting
  success" — and the file's own current comment: "An existing hook whose content
  is unknown cannot be classified … Refuse on both routes (G-0557)."

### G-0478

**`stale-claim`**

- **Claim:** Two links in `docs/initiatives/quality-signal-and-cadence.md` *still* name
  `work/epics/E-0073-mutating-verb-ux-uniformity/epic.md` and its `M-0281` sibling, and
  "the second pair is the point".
- **Measured:** False now, and false when the sentence was last revised. Both links point at
  `work/epics/archive/…` at lines 305–306 of that document. They were repaired by
  `f2017eb56` "fix(docs): point initiative links at archived entity paths",
  **2026-08-02 20:25:59**. The gap's body was rewritten by `74ae8337e` (`aiwf edit-body
  G-0478`) on **2026-08-10 13:33:57** — eight days later — and the sentence survived. At
  `74ae8337e` the tree carried **zero** broken `docs/`→`work/` links.
- **Command:**
  `grep -n "E-0073\|M-0281" docs/initiatives/quality-signal-and-cadence.md` →
  `305:[E-0073]\(../../work/epics/archive/E-0073-…/epic.md\)`,
  `306:[M-0281]\(../../work/epics/archive/E-0073-…/M-0281-…md\)`;
  `git log --oneline -5 -S 'work/epics/archive/E-0073-mutating-verb-ux-uniformity/epic.md' -- docs/initiatives/quality-signal-and-cadence.md` → `f2017eb56`;
  `git log --format='%h %ad %s' --date=iso -- work/gaps/G-0478-*.md`;
  `python3 …/links_at.py 74ae8337e` → `links docs/->work/: 63 … broken: 0`.
- **Quoted:** "Two links in that same document still name
  `work/epics/E-0073-mutating-verb-ux-uniformity/epic.md` and its `M-0281` sibling. Both now
  live under `work/epics/archive/`. Nothing reported them; they were found only by walking
  the links directly."

**`dead-premise`**

- **Claim:** The recommended first move is "**Detection** … a check rule that resolves every
  relative markdown link whose target is a `.md` file under `work/`, and reports the ones
  naming no existing file."
- **Measured:** An accepted ADR declines exactly that, and a prior gap proposing it was
  already dispositioned. `docs/adr/ADR-0033-…:45` reads "Enforcement is at move-time only.
  No pre-push check rule is added for this concern, so the pre-push chokepoint's cost is
  unchanged." ADR-0033 is `accepted`. `G-0392` — "aiwf check: flag markdown path-links into
  entity files (archive strands them)" — is `addressed`/archived (`addressed_by_commit:
  aa57a1b6`) and its body proposed the same rule at the same chokepoint; E-0063 took the
  opposite stance and delegated the residue to advisory detection. E-0088's *Out of scope*
  repeats it: "**A pre-push check rule for link integrity.** ADR-0033's third bullet places
  enforcement at move time and declines to grow the pre-push chokepoint's cost." G-0478 cites
  neither ADR-0033 nor G-0392 anywhere in its body.
- **Command:** `grep -n "Enforcement is at move-time only" docs/adr/ADR-0033-*.md` → `45`;
  `aiwf show ADR-0033` → `status: accepted`; `aiwf show G-0392` → `status: addressed ·
  archived`; `head -30 work/gaps/archive/G-0392-*.md`;
  `grep -n "ADR-0033\|G-0392" work/gaps/G-0478-*.md` → no output.
- **Quoted (ADR-0033:45):** "Enforcement is at move-time only. No pre-push check rule is
  added for this concern, so the pre-push chokepoint's cost is unchanged."

### G-0504

**`dead-premise`**

- **Claim:** "G-0471 — the binary-versus-source staleness axis, addressed by
  E-0076. This gap is the artifact-versus-embed axis; neither detects the
  other." (the `## Related` section)
- **Measured:** E-0076 is `status: cancelled`, archived — `aiwf cancel E-0076 ->
  cancelled` on 2026-08-03. G-0471 is `status: open`, `priority: high`, never
  promoted. This body's last `edit-body` is `505c972`, 2026-08-13 — ten days
  after the cancel — so the claim was already false when the current text was
  written. A reader concludes the binary-vs-source axis has an owner; it has
  none.
- **Command:** `aiwf show E-0076 --root /workspaces/aiwf` ;
  `aiwf show G-0471 --root /workspaces/aiwf` ;
  `aiwf history G-0504 --root /workspaces/aiwf` ;
  `head -30 work/epics/archive/E-0076-*/epic.md`
- **Quoted (E-0076 body):** "Addresses G-0465, G-0471 and G-0474." — the epic
  that was to address it, `status: cancelled`.

### G-0508

**`undercount`**

- **Claim:** "Four policies scan `internal/verb` for package-level function
  declarations, and each carries its own copy of the walk", followed by a
  four-item list.
- **Measured:** **Eight** files in `internal/policies/` carry that walk. The four
  unnamed ones are:
  - `verbs_validate_then_write.go` (added **2026-05-05**, `a13713213`)
  - `commit_construction_seam.go` (added **2026-07-06**, `45d4c3ed1`) — prefix is
    `internal/verb/` *or* `internal/gitops/`
  - `verb_result_noop_invariant.go` (added **2026-07-27**, `1304bdd69`) — uses
    `WalkGoFiles(root, false)` deliberately, since it needs test bodies too
  - `coherence_guard_chokepoint.go` (added **2026-08-04**, `c0b45f3d7`) — walks
    the whole tree minus `internal/gitops/` and uses the `internal/verb/` prefix
    to classify rather than to skip

  Three of the four predate the gap (added 2026-08-01, `9c074bea7`), so this was
  an undercount at the moment of writing, not drift. A fourth arrived three days
  later. Scoped strictly to files using the prefix as a scan *filter*, the count
  is seven. `mint_ids_via_allocate.go` and `empty_diff.go` also carry
  `WalkGoFiles` + the same prefix filter but use `ast.Inspect` / a textual scan
  instead of the `Decls` loop, so they are outside the claim as stated;
  `acks_helper_lift.go` and `apply_callers_lock.go` scan other prefixes.
- **Command:**
  ```
  for f in internal/policies/*.go; do case "$f" in *_test.go) continue;; esac
    grep -q 'HasPrefix(f.Path, "internal/verb/")' "$f" && grep -q "astFile.Decls" "$f" \
      && grep -q "ast.FuncDecl" "$f" && grep -q "WalkGoFiles(" "$f" && echo "$f"; done | wc -l
  # → 8
  git log --diff-filter=A --format='%ad %h' --date=short -1 -- internal/policies/<each>.go
  ```
- **Quoted:** "Four policies scan `internal/verb` for package-level function
  declarations, and each carries its own copy of the walk"

### G-0514

**`dead-premise`**

- **Claim:** "Measured on the shipped tree, these currently fire under that message" — followed
  by four enumerated classes (`M-id`/`E-id`/`C-id`; `M-PPPP`/`M-QQQQ`/`M-PPP`/`M-QQQ`;
  `M-a`/`M-alpha`/`M-007a`/`M-NNN/AC-X`; `ADR-NEW`/`ADR-OPSPEC`).
- **Measured:** `skill-body-id` fires on **nothing** in the tree today — `aiwf check` reports
  `"status":"ok"` with `"findings":[]` and `findings: 0` over 1143 entities. Not one of the
  eleven named tokens appears anywhere under `internal/skills/` (grep returns zero lines for
  each). They were removed by the M-0288 sweep: `a0fe2d90c` (18:05:20Z), `17bc2c8b0`
  (18:20:55Z), `3ca397918` (18:35:32Z) — all on 2026-08-02, the same day the gap was filed at
  16:22:08Z (`897ec9cb4`). The rule itself is still wired (`internal/check/check.go:147,149`)
  and still classifies exactly as the body describes — confirmed by planting all eleven tokens
  plus four English words in a fixture `SKILL.md`, which produced 17 `skill-body-id` errors.
- **Command:**
  `…/audit/aiwf check --format=json`;
  `for t in 'M-PPPP' 'M-QQQQ' 'M-PPP' 'M-QQQ' 'M-a\b' 'M-alpha' 'M-007a' 'M-NNN/AC-X' 'ADR-NEW' 'ADR-OPSPEC'; do grep -rn -- "$t" internal/skills/; done`;
  `grep -rn -- '\(M\|E\|C\|G\|D\|ADR\)-id\b' internal/skills/embedded internal/skills/embedded-rituals internal/skills/embedded-guidance internal/skills/embedded-statusline`;
  `git log --oneline -S 'M-id' -- internal/skills/`;
  `for c in 3ca397918 17bc2c8b0 a0fe2d90c 897ec9c; do git log -1 --format='%h %cI %s' $c; done`;
  fixture: `…/audit/aiwf check --root . --format=json` against a hand-built repo carrying
  `internal/skills/embedded/skills/aiwf-demo/SKILL.md` with the eleven tokens.
- **Quoted:** "Measured on the shipped tree, these currently fire under that message:"

**`dead-premise`**

- **Claim:** "This lands on the sweep as concrete friction. A sweep driven by the rule's output
  must resolve each of these by hand… Decide against the sweep's actual worklist rather than in
  the abstract — the sweep is where the population is enumerated, and its per-token judgments
  are the evidence for which option is right."
- **Measured:** The sweep is M-0288 (`Sweep shipped surfaces to canonical placeholders and
  enforce at error severity`), status `done`, archived, all four ACs terminal, wrapped
  2026-08-02 — roughly two hours after the gap was filed. Its parent E-0078 is `done` and
  archived (wrapped 2026-08-03). There is no live sweep to decide against; the per-token
  judgments were made without the decision. Those judgments are now readable in the diffs, and
  they went the way the gap warned: `git show 3ca397918 -- internal/skills/embedded/aiwf-promote/SKILL.md`
  rewrites the command synopsis `aiwf promote <M-id>/AC-N` to `aiwf promote <M-NNNN>/AC-N`.
- **Command:** `…/audit/aiwf show M-0288`; `…/audit/aiwf show E-0078`;
  `git show 3ca397918 -- internal/skills/embedded/aiwf-promote/SKILL.md`
- **Quoted:** "Decide against the sweep's actual worklist rather than in the abstract — the
  sweep is where the population is enumerated"

### G-0517

**`false-premise`**

- **Claim:** "These are mostly citations of entities that were genuinely real
  at a narrow width, so the correct fix is widening each to the real canonical
  id — a different edit… with a lower payoff than cleaning the docs an
  assistant reads to learn the workflow." And the `## Resolution`: "Widen each
  citation to the entity's canonical id".
- **Measured:** Of the 113 raw narrow-id occurrences in the three surfaces, 71
  are real-shaped (all-digit suffix) and 42 are letter-N placeholders. I read
  every one of the 41 lines carrying a real-shaped narrow id. **Zero of them
  is a citation of the entity that id names.** Every one is worked-example
  fiction, the same genre the swept corpus was:
  - `docs/design/design-decisions.md:31` — "An id like `E-19`, once
    allocated, always means the same entity" (an example of *an* id);
    `:101` — `E-19-<slug>/`, `M-001-<slug>.md` shape examples; `:110` — a
    worked `reallocate` transcript (`aiwf history M-008` / `M-007`);
    `:149`/`:150` — trailer/composite-id examples.
  - `docs/design/design-lessons.md:13` — "Identities are stable names
    (`E-001`, `M-002`, `D-014`)". `docs/architecture.md:170` gives the same
    parenthetical as (`E-19`, `M-002`, `D-014`) — the two files disagree on
    the first element, which is itself evidence they are illustrations.
  - `docs/design/id-allocation.md:11,13,60,75,78,82,109` — the whole
    branch-collision worked example, `G-035` renumbered to `G-037`. G-0035 is
    really "HTML site only generates pages for epic and milestone…" and
    G-0037 is "Cross-branch id collisions split the audit trail…"; widening
    would turn the example into a false citation of both.
  - `docs/design/provenance-model.md:206,301,310-313,338,343,350-358,365-366,
    381,395` — Examples 3–6 of the authorization transcript, with fabricated
    dates (2026-04-30) and reasons ("implement the engine", "fixture work
    landed"). E-0003 is "Skills, history, hooks"; E-0009 is "Iteration I3 —
    Governance HTML render".
  - `docs/overview.md:126,129,130,143,144,145,148,150` — the mermaid flow and
    the `work/epics/E-01-discovery-and-ramp-up/` tree. E-0001 is "Foundations
    and aiwf check", not "discovery and ramp-up"; M-0001 is "Session 1
    deliverable", not "map the system".
  - `docs/design/legal-workflows-first-principles.md:290` is a third category
    the gap does not name at all: rule R-FP-0104 writes "(`E-22`, `M-007`)"
    *as examples of narrower legacy widths*. Narrowness is the sentence's
    subject, so neither widening nor a canonical placeholder preserves it.

  The already-swept corpus was fixed the opposite way, and the sweep commit
  says exactly why.
- **Command:**
  `git show ece70f010` (message + diff);
  `python3` enumeration of `\b(?:E|M|G|D|C|ADR)-[A-Za-z0-9_]+(?:/AC-[A-Za-z0-9_]+)?\b`
  over `docs/design/*.md docs/overview.md docs/architecture.md` filtering
  suffix-length < 4; `sed -n` on every reported line;
  `…/audit/aiwf show E-19 M-002 D-014 M-001 M-008 M-007 E-001 G-035 G-037 E-22 E-03 E-09 E-01`.
- **Quoted:** the sweep commit `ece70f010` reads *"112 narrow sites become
  canonical `<prefix>-NNNN` placeholders. Every one was tutorial fiction, so
  the placeholder is the fix rather than a widened number — widening would
  turn an invented id into a citation of whatever entity now holds it."*
  That reasoning applies unchanged to all 41 lines above.

### G-0523

**`overclaimed-attribution`**

- **Claim:** "The events are already wired, so this needs no consent surface
  beyond what ADR-0015 already governs."
- **Measured:** Both halves are false.
  (a) *The events are not already wired in a consumer repo.* A fresh
  non-interactive `aiwf init` leaves the hook **undecided**: the init ledger
  ends `aiwf init: hook "worktree-rituals-check.sh" — deferred (undecided — run
  \`aiwf doctor\`)`; `ls .claude/` shows only `agents  aiwf-guidance.md  skills
  templates` — no `hooks/` directory and no `settings.json`; `aiwf doctor`
  reports `hooks: drift: 1 undecided, 0 materialized-not-wired, 0
  wired-but-stale`. Only after `aiwf update --enable-hook
  worktree-rituals-check.sh` did `.claude/hooks/worktree-rituals-check.sh` and a
  `.claude/settings.json` with `SessionStart`/`SubagentStart` entries appear.
  This repo is wired only because its committed `aiwf.yaml` carries
  `hooks: worktree-rituals-check.sh: enabled: true`.
  (b) *ADR-0015 is not the governing record.* ADR-0032 (accepted, dated
  2026-07-06 — i.e. a month before this gap was filed on 2026-08-03) is, and it
  rules ADR-0015's mechanism unfit for hooks by name.
  The consequence for the fix shape is material: a guidance-delivery hook would
  ship default-OFF and be *silently refused* in exactly the headless agentic
  `aiwf init` that ADR-0018 went automatic to protect — a weaker channel than
  the one it is proposed to backstop.
- **Command:**
  `aiwf init --root <fixture>` (HOME redirected) ;
  `ls <fixture>/.claude/` ; `cat <fixture>/.claude/settings.json` (No such file) ;
  `aiwf doctor --root <fixture>` ;
  `aiwf update --root <fixture> --enable-hook worktree-rituals-check.sh` ;
  `ls <fixture>/.claude/hooks` ;
  `sed -n '1,120p' docs/adr/ADR-0032-materialized-hook-consent-persisted-per-hook-aiwf-yaml-registry.md`
- **Quoted (ADR-0032, Context):** "ADR-0015's per-invocation, unpersisted prompt
  does not scale past one feature — a second hook would mean a second bespoke
  flag, a third a third, which is the exact generalization ADR-0015's own
  consequences section warns against."
- **Quoted (ADR-0032, Decision):** "aiwf does not activate a materialized hook
  without consent, calibrated to hooks' own profile: **per-hook granularity,
  decided once, persisted and shared.** … **`aiwf init`** (no prior `aiwf.yaml`):
  every hook aiwf ships is undecided, so every one is gated before the fresh
  `aiwf.yaml` is written — a TTY `[y/N]` prompt naming the hook and its one-line
  effect (default declines), or, absent a TTY, silent refusal unless the operator
  passes `--enable-hook <name>`."

### G-0527

**`contradicted-by-code`**

- **Claim:** "Asking for the verb that does not exist reports success: `aiwf worktree remove
  <path>` -> exit 0, prints the parent help… That exit-code behavior is general to every parent
  verb rather than specific to this one — a bogus top-level verb correctly exits 2, while a bogus
  subverb under `contract`, `milestone`, `acknowledge` or `worktree` exits 0."
- **Measured:** False for all five cases. `aiwf worktree remove /tmp/nonexistent-abc-zzz` printed
  `aiwf: unknown command "remove" for "aiwf worktree"` and exited **2**, not 0, and printed no
  help. A bogus subverb under each of `contract`, `milestone`, `acknowledge`, `worktree` also
  exits 2. Fixed by `981115515` *"fix(cli): verb groups report a usage error instead of exiting 0
  on an unknown subverb"* (2026-08-04), which introduced
  `internal/cli/cliutil/verbgroup.go`'s `MarkVerbGroup` — its doc comment names the exact
  mechanism the gap describes ("Cobra returns flag.ErrHelp for a command that is not Runnable
  before it validates arguments… it answers a subverb it does not have with help and exit 0")
  and credits **G-0528**, which `aiwf show G-0528` reports as `addressed` and archived. All five
  verb groups (`contract`, `contract recipes`, `acknowledge`, `worktree`, `milestone`) route
  through `MarkVerbGroup`. G-0527 was authored 2026-08-03; the fix landed the next day and the
  body was never revisited.
- **Command:**
  ```
  aiwf worktree remove /tmp/nonexistent-abc-zzz      -> 'aiwf: unknown command "remove" for "aiwf worktree"'  exit=2
  aiwf bogusverb                                     -> exit=2
  for p in contract milestone acknowledge worktree; do aiwf $p bogussub; echo $?; done   -> 2 2 2 2
  aiwf show G-0528                                   -> status: addressed · archived
  git log --oneline --diff-filter=A -- internal/cli/cliutil/verbgroup.go -> 981115515
  git show -s --format='%h %ad %s' --date=short 981115515
  ```
- **Quoted:** From `verbgroup.go:29-34`: *"The RunE is load-bearing rather than cosmetic. Cobra
  returns flag.ErrHelp for a command that is not Runnable before it validates arguments, so a
  group whose behavior lives entirely in its children has no reachable Args constraint: it
  answers a subverb it does not have with help and exit 0. Supplying RunE makes the group
  Runnable, which is what lets the NoArgs below reject that name (G-0528)."*

**`claim-about-what-does-not-exist`**

- **Claim:** "Nothing removes one: teardown is `git worktree remove` plus `git branch -d`,
  unmentioned by any surface."
- **Measured:** False, and false at authoring time. Four surfaces name it, three of them shipped:
  - `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md:197,199` — an
    executable block running `git worktree remove "$PATCH_WT"` then `git branch -d <branch>`,
    with the ordering rationale spelled out. In the tree since `6c74894a7` (2026-07-04).
  - `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md:296,298`
    — the same pair for a milestone worktree.
  - `internal/skills/embedded/aiwf-status/SKILL.md:109` — *"cleanup: git worktree remove <path>"*,
    since `e25fd62da` (2026-05-20).
  - `internal/cli/status/worktrees.go:948,954` — `aiwf status` itself prints
    `… cleanup: git worktree remove <path>` for ABANDONED and SAFE-TO-REMOVE worktrees, i.e. a
    *runtime* surface, not only documentation. Since `e25fd62da` (2026-05-20).

  The gap's specific "you only discover it by trying" story is itself written down: the wf-patch
  skill says *"git refuses to delete a branch a worktree still holds — 'error: cannot delete
  branch … used by worktree at …'"*. I reproduced that refusal exactly
  (`git branch -d testbranch` → `error: cannot delete branch 'testbranch' used by worktree at
  …/fx2/.claude/worktrees/testbranch`, exit 1). What *is* true is the narrow version: neither
  `aiwf worktree add --help` nor `internal/skills/embedded/aiwf-worktree/SKILL.md` mentions
  teardown at all — I read both in full.
- **Command:**
  ```
  grep -rn "worktree remove\|branch -d" --exclude-dir=.git --exclude-dir=site .
  sed -n '185,205p' internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md
  sed -n '940,958p' internal/cli/status/worktrees.go
  git log --format='%h %ad %s' --date=short -S 'git branch -d <branch>' -- …/wf-patch/SKILL.md
  git log --format='%h %ad %s' --date=short -S 'cleanup: git worktree remove' -- internal/skills/embedded/aiwf-status/SKILL.md
  aiwf worktree add --help ; cat internal/skills/embedded/aiwf-worktree/SKILL.md
  git -C fx2 branch -d testbranch
  ```
- **Quoted:** wf-patch SKILL.md: *"Delete the local branch; remove the worktree if one was used.
  Order matters and the branch cannot go first: git refuses to delete a branch a worktree still
  holds…"*

### G-0529

**`contradicted-by-record`**

- **Claim:** "The failure is not hypothetical. E-0075 wrapped with an entry that
  omitted a user-visible refusal; a human noticed afterwards, and the omission
  was tracked and back-filled as G-0509. **The entry existed and was thin** —
  the shape a presence check cannot catch." And, resting on it: the second
  Direction bullet's "This is the property that catches a thin entry, and the
  one that would have caught G-0509."
- **Measured:** Wrong on all three particulars, and the timeline runs the other
  way.
  - **The entry did not exist.** G-0509's own body, which the gap cites, says:
    "`CHANGELOG.md` records none of it. The epic branch carries over 150 commits
    and not one touches the file." Its title is "E-0075's user-visible refusal
    is **absent** from CHANGELOG".
  - **The human noticed before the wrap, not after.** G-0509 was added by
    `1aa0f7099` at **2026-08-01 20:15:01**. E-0075's wrap-artefact commit is
    `6249148e2` at **2026-08-02 16:43:02**; the epic reached `done` at
    **2026-08-02 16:44:21**. G-0509 was promoted `addressed` at 16:44:10, one
    minute after the wrap wrote the entry and eleven seconds before the epic
    closed.
  - **There was no back-fill.** The commit G-0509 names in
    `addressed_by_commit` is `6249148e2` — the epic's own ordinary
    wrap-artefact commit (`aiwf-verb: wrap-epic`, +23 lines to `CHANGELOG.md`).
    G-0509's Scope section had explicitly asked for exactly that: "Write it at
    the epic wrap rather than per milestone."
  - **The entry is not thin.** `6249148e2` added a `### Changed — E-0075:` entry
    of three paragraphs covering the refusal, the per-candidate sweep decline,
    and the `edit-body --body-file` narrowing.
  The consequence for the gap's argument: G-0509's condition was *absence* of an
  entry for an epic heading to `done`, which is precisely what Direction bullet
  one ("Every epic that reached `done` is cited in `CHANGELOG.md`") computes.
  Attributing it to bullet two — the costlier surface-delta property — is
  backwards, and it is the sentence a reader would use to justify building the
  expensive one first.
- **Command:** `cat work/gaps/archive/G-0509-*.md` ;
  `git log --format="%h %ci %s" --all --grep="G-0509"` ;
  `git log --follow --diff-filter=A --format="%h %ci %s" -- work/gaps/archive/G-0509-*.md`
  → `1aa0f7099 2026-08-01 20:15:01` ; `$A history E-0075` ;
  `git show --stat 6249148e21f6ae2caa5967d3f24c991df33094f9` ;
  `git show 6249148e2 -- CHANGELOG.md`
- **Quoted:** G-0509's Problem section — "`CHANGELOG.md` records none of it. The
  epic branch carries over 150 commits and not one touches the file."

### G-0530

**`already-addressed`**

- **Claim:** "The classification of `entity-body-empty` in the shipped-surface table of
  `docs/design/growth.md` records it as a mandate. It is a ban… **The table needs that
  correction**, and the mechanism it obscures is worth stating in its place — a template that
  seeds headings and a ban that forces them filled compose into a mandate, while neither is one
  alone."
- **Measured:** The correction landed on 2026-08-04, 14 hours after the gap was filed, and it
  landed in the exact terms the gap proposed. `docs/design/growth.md:151-160` now reads:
  "**A template and a ban compose into a mandate neither one is.** `entity-body-empty` is
  classified above as a ban, and that is what it is: it fires only on a section that is *present
  and empty*, so a body omitting the heading altogether satisfies it and the check reports
  nothing. What produces the obligation is the pair: the spec template seeds the headings, and
  the ban then forces every seeded heading to be filled… **Tracked as G-0530.**"
  The claim was true at filing — `git show cdbc70eae:docs/design/growth.md` (2026-08-03T18:33Z,
  36 minutes before the gap) shows "**`entity-body-empty` is the highest-cost mandate in the
  shipped surface.**". The fixing commit is `2574341f7 docs: correct the shipped-surface
  classification and name what each gate loses` (2026-08-04T09:06:35Z).
- **Command:** `grep -n "entity-body-empty" docs/design/growth.md`; `sed -n '125,175p' docs/design/growth.md`;
  `git log --format='%h %cI %s' -S 'Tracked as G-0530' -- docs/design/growth.md`;
  `git show cdbc70eae:docs/design/growth.md | grep -n -A12 'entity-body-empty'`;
  `git log --format='%h %cI %s' -S 'shipped-surface table of' -- work/gaps/G-0530-*.md`
- **Quoted:** "The table needs that correction, and the mechanism it obscures is worth stating
  in its place"

**`count-does-not-rederive`**

- **Claim:** "`## Work log` is the sharpest case — the wrap ritual mandates one entry per
  acceptance criterion with its outcome and commit SHA, and **the section is empty in half the
  milestones that carry it**." Plus the table row placing `## Work log` among four sections whose
  "median word counts [are] 0, 14, 21 and 24 respectively."
- **Measured:** The wrap-ritual half re-derives exactly (see below). The emptiness half does not,
  under any of four readings, at either the stated measurement date or today:

  | reading | 2026-08-03 (`cfc0c0238`) | today (`da34c1009`) |
  |---|---|---|
  | section truly empty (subsections included) | 1 of 194 = **1%** | 6 of 210 = **3%** |
  | no `###` subsection at all | 36 of 194 = **19%** | 42 of 210 = **20%** |
  | no commit-SHA-shaped token anywhere | 44 of 194 = **23%** | 50 of 210 = **24%** |
  | no direct prose before the first subsection | 137 of 194 = **71%** | 151 of 210 = **72%** |

  Nothing is near 50%. The 0-word median re-derives *only* under the fourth reading — content
  directly under the heading, subsections excluded — which is the method the whole table uses
  (I reproduced 0 / 14 / 21 / 24 exactly at `cfc0c0238`, and 0 / 14 / 19.5 / 27 today). But that
  method is wrong for `## Work log` specifically, because the template *prescribes* subsections:
  the section's scaffold is `### AC-1 — <short title>`. Counted with its own prescribed content,
  `## Work log` has a median of 226.5 words (191.5 at the measurement date) and is the
  **third-largest section in the milestone spec**, behind `## Acceptance criteria` (227) and
  ahead of `## Reviewer notes` (218) — the section the gap holds up as "the largest". So the
  headline table lumps the spec's third-biggest section in with three genuinely thin ones.
- **Command:** Python over `work/epics/**/M-*.md` (317 files today; 290 at the snapshot
  extracted read-only via `git archive cfc0c0238 work/epics | tar -x`), splitting on `^## `
  and on `^#{1,6} ` respectively, stripping `<!--…-->` before counting. Section totals/medians
  computed both ways; per-reading emptiness counts as tabulated.
- **Quoted:** "the section is empty in half the milestones that carry it."

**`dead-premise`**

- **Claim:** "Gap is the largest population and has no template at all, so its structure is
  convention carried in guidance and skills rather than a file that can be edited."
- **Measured:** First half true — 620 gaps vs 317 milestones, 88 epics, 75 decisions, 46 ADRs,
  0 contracts (archive included). Second half **false since 2026-08-22**: a gap template ships at
  `internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/gap.md`, added by
  `711669bb9 feat(skills): ship gap and contract body templates` (2026-08-22T20:37:21Z — the day
  before HEAD). It is a full authoring template with `## What's missing` / `## Why it matters`
  guidance prose and a how-to-use comment. The gap's last `edit-body` was 2026-08-16, so it
  could not have known; the claim is nonetheless wrong now, and it is the load-bearing reason
  the body gives for treating the gap surface as unprunable.
- **Command:** `for k in epic milestone adr gap decision contract; do …/audit/aiwf list --kind $k --archived --format=json | python3 -c "import json,sys; print(len(json.load(sys.stdin)['result']))"; done`;
  `ls internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/`;
  `git log --format='%h %cI %s' --diff-filter=A -- …/templates/gap.md`
- **Quoted:** "Gap is the largest population and has no template at all"

### G-0535

**`dead-premise`**

- **Claim:** Option 3 — "**Give each an anti-orphan assertion**, as the sovereign
  policy now has — a live-tree test asserting the scanned prefix still holds
  subjects."
- **Measured:** The sovereign policy no longer exists. `internal/policies/sovereign.go`,
  `sovereign_test.go` and `sovereign_guard_predicate_test.go` were **deleted** on
  2026-08-06 in `72c914ecd` "fix(cli): retire the dispatcher-layer force guard
  (M-0293 review)". The claim was true when written: at `72c914ecd^` the file
  carried `TestSovereignDispatchers_ScopeIsNotOrphaned` and
  `const sovereignDispatcherPrefix = "internal/cli/"`. G-0535's body was last
  edited 2026-08-04 (`0c14fb5af`), two days before. The only anti-orphan test in
  the tree today is `TestPolicyApplyCallersAcquireLock_ScopeIsNotOrphaned`
  (`apply_callers_lock_test.go`:18). The retirement is recorded as **D-0061**,
  status `accepted` — "Retire the sovereign-dispatcher policy; the force rule
  lives at the apply seam" — i.e. the sovereign case resolved by the gap's own
  option **2**, not option 3.
- **Command:**
  `git log --oneline --diff-filter=D --name-only --format='%h %ad %s' --date=short -- 'internal/policies/*sovereign*'`;
  `git show 72c914ecd^:internal/policies/sovereign_test.go | grep -n "func Test"`;
  `grep -rn "ScopeIsNotOrphaned" internal/policies/*.go`; `aiwf show D-0061`
- **Quoted:** `legal-workflows-audit.md`:159 (R-AUDIT-0070) — "Enforced at
  `verb.Apply` over the fully-assembled trailer set, so every route is covered
  without being enumerated (ADR-0040) — **there is no dispatcher-layer assertion**,
  and D-0061 records why one cannot exist".

**`dead-premise`**

- **Claim:** "That is what separates these from G-0534, where `internal/verb`
  really does enforce the guarantee the dispatcher scan also claims, so **the
  redundancy question there is live**."
- **Measured:** G-0534 is `addressed` and archived; its redundancy question was
  settled by D-0061 (`accepted`, 2026-08-06) in favour of retiring the dispatcher
  scan. G-0476, the sibling sweep the Scope section cites, is likewise `addressed`
  and archived. A reader taking the contrast at face value would go looking for an
  open question that no longer exists.
- **Command:** `aiwf show G-0534`; `aiwf show G-0476`; `aiwf show D-0061`
- **Quoted:** as above.

### G-0536

**`dead-premise`**

- **Claim:** "On the tree as it stands the step reports errors on day one. Which
  view is authoritative is a question this gap inherits rather than answers;
  G-0556 holds it." (and, in the same paragraph, "an id minted on a local branch
  that was never pushed is unreachable from any CI checkout and resolves as an
  error there")
- **Measured:** false in both the strictest and the realistic reproduction.
  1. *Ref-less fixture* (the harshest case — one commit, no remote, no trunk, no
     sibling refs), populated with `work/`, `docs/`, `internal/skills/`,
     `aiwf.yaml`, `README.md`, `ROADMAP.md`, `CLAUDE.md`:
     `1 findings (0 errors, 1 warnings)` — the single warning being
     `provenance-untrailered-scope-undefined … provenance audit skipped`,
     which is the fixture having no upstream, not a tree fault.
  2. *CI-like checkout* — `git clone --shared /workspaces/aiwf`, every
     remote-tracking ref dropped except `origin/main`, upstream set, full
     history: `ok — no findings`, and `--since v0.32.0` (provenance audit over a
     real range) also `ok — no findings`.
  3. *The live repo*: `ok — no findings` (three consecutive runs).
  G-0556 is `status: addressed`, archived, `addressed_by_commit: 50a642903`
  (2026-08-06) — one day **after** this body's last `edit-body` (`055cabc7f`,
  2026-08-05), which is why the body never heard. ADR-0041 landed with it and
  split the cross-branch class: a remote-visible reference stays a warning, a
  local-only one becomes an error at the push boundary — so a tree that reaches
  trunk can no longer carry a reference CI cannot follow.
- **Command:**
  `aiwf check --root <refless-fixture>` ;
  `git clone --shared /workspaces/aiwf <ciclone>` + `git update-ref -d` on the
  extra remotes + `git branch --set-upstream-to=origin/main main` +
  `aiwf check --root <ciclone> --verbose` + `aiwf check --root <ciclone> --since v0.32.0` ;
  `aiwf check --root /workspaces/aiwf` ;
  `aiwf show G-0556 --root /workspaces/aiwf` ;
  `git log -1 --format='%h %ad %s' --date=short 50a642903` ;
  `aiwf history G-0536 --root /workspaces/aiwf`
- **Quoted (G-0556 §"What has landed"):** "`refs-resolve` and `body-prose-id`
  now split a cross-branch hit by the most visible ref that resolves it: any
  remote-tracking ref keeps `cross-branch-pending` at warning severity, local
  branch refs alone fire `cross-branch-local-only` at error, and a miss at every
  tier stays `unresolved`."
- **Residue the edit should keep:** G-0556 also says "What remains is the half
  this cannot reach … a tree copied into a ref-less repository resolves a
  published-but-unmerged id at no tier". That half is real but did **not** fire
  here — measurement 1 above is exactly that repository shape and reported zero
  errors, because the tree currently cites no unmerged id at all. The property is
  therefore true today and not guaranteed for every future tree.

### G-0539

**`contradicted-by-code`**

- **Claim:** "The pre-commit half prevents rather than detects, and needs one distinction to be safe: a verb's own commit also stages entity files. The verbs drive git themselves, so they can mark their own commits — an environment variable set across the commit call is the cheap version, with the hook skipping when it is present." And: "the second and third are verbs, covered by the same marker" (of `aiwf archive`'s sweep and `aiwf reallocate`'s rewrite).
- **Measured:** aiwf verb commits fire **no git hooks whatsoever** and stage nothing in the index, so a pre-commit rule reading `git diff --cached` would never see a verb's commit and needs no marker or exemption. In the fixture, with a logging `pre-commit.local` and `commit-msg.local` installed, five verb invocations produced five commits and zero hook firings; the plain `git commit` control fired both hooks.
  ```
  === aiwf add gap (verb commit) ===      aiwf add gap G-0003 "Hook probe gap"
  === aiwf edit-body (verb commit) ===    aiwf edit-body G-0001
  === aiwf promote (verb commit) ===      aiwf promote G-0002 open -> addressed
  === plain git commit control ===        ok — no findings
  --- hook log ---
  PRE-COMMIT FIRED: staged=[control.txt ]
  COMMIT-MSG FIRED: chore: control commit
  ```
  A second run over `aiwf archive --apply` and `aiwf reallocate G-0003` produced two commits and the log read `(no hook firings)`.
  Source corroboration at `internal/gitops/gitops.go:97-119`: `Commit` and `CommitAllowEmpty` are marked *"Test/porcelain-only (F7): no `aiwf` verb calls this at runtime"*, and verbs *"route through gitops.CommitVerbChange/CommitTree — the plumbing path (commit-tree + update-ref), **which fires no hooks** and needs no staged changes"*.
- **Command:**
  ```
  # in the fixture, .git/hooks/{pre-commit,commit-msg}.local append to /tmp/c18-verbhooks.log
  aiwf add gap --title "Hook probe gap" --body-file /tmp/body1.md
  aiwf edit-body G-0001 --body-file /tmp/body2.md
  aiwf promote G-0002 addressed --by-commit HEAD
  git add control.txt && git commit -m "chore: control commit"
  aiwf archive --apply ; aiwf reallocate G-0003
  cat /tmp/c18-verbhooks.log
  ```
- **Quoted:** body — *"needs one distinction to be safe: a verb's own commit also stages entity files"*. Source — *"the plumbing path (commit-tree + update-ref), which fires no hooks"*.

### G-0555

**`contradicted-by-measurement`**

- **Claim:** an age-based sweep cannot reclaim this, therefore only a code fix can — "a sweep
  holding to a 24-hour safety margin — the shortest margin that is safe when concurrent
  sessions may still hold a built binary — would have found nothing to delete… The leak
  regenerates faster than an age-based sweep can safely reclaim it, so any operational
  workaround either lags the growth or risks deleting a directory a running test is using.
  That leaves the code as the only place the problem can actually be solved."
- **Measured:** a 24-hour-margin sweep run right now would delete **1367 directories holding
  24.4 GB** — 75% of the 32.4 GB accumulated — while the last 24 hours account for only
  6.5 GB. The directories are *not* all younger than a safe cutoff; they span 19 consecutive
  days, from 2026-08-05 to today, with nothing having removed one.
- **Command:**
  ```
  find /tmp -maxdepth 1 -name 'aiwf-int-build-*'        -type d -mtime +1 | wc -l   → 1002
  find /tmp -maxdepth 1 -name 'aiwf-int-build-*'        -type d -mtime +1 -print0 \
      | du -shc --files0-from=- | tail -1                                           → 18G
  find /tmp -maxdepth 1 -name 'stresstest-shared-bin-*' -type d -mtime +1 | wc -l   → 365
  find /tmp -maxdepth 1 -name 'stresstest-shared-bin-*' -type d -mtime +1 -print0 \
      | du -shc --files0-from=- | tail -1                                           → 6.4G
  # last 24h, for contrast:
  find … -name 'aiwf-int-build-*'        -type d -mtime -1 | wc -l → 272  (4.8G)
  find … -name 'stresstest-shared-bin-*' -type d -mtime -1 | wc -l →  93  (1.7G)
  # spread across days:
  find /tmp -maxdepth 1 -name 'aiwf-int-build-*' -type d -printf '%TY-%Tm-%Td\n' | sort | uniq -c
    → 226 on 2026-08-05, 279 on 08-06, 69, 120, 1, 61, 28, 22, 4, 48, 20, 16, 48, 17,
      37, 5, 73, 190, 108 on 2026-08-23  (sums to 1372; every day represented)
  ```
- **Quoted:** "The leak regenerates faster than an age-based sweep can safely reclaim it, so
  any operational workaround either lags the growth or risks deleting a directory a running
  test is using. That leaves the code as the only place the problem can actually be solved."
- **Note on scope:** the *dated* half of the paragraph ("On the day measured, every one of
  these directories had been created within the preceding 24 hours") is honestly scoped and I
  cannot refute it — the oldest surviving `aiwf-int-build-*` directory is
  `2026-08-05 01:53`, consistent with there being nothing older on the day it was written.
  It is the tense-free generalisation drawn from it that measurement contradicts.

### G-0560

**`dead-premise`**

- **Claim:** "G-0559 gates it — the strings originate in `internal/entity/entity.go` and
  `aiwf schema` prints them, so widening the doc tables **while the emitter still says
  `E-NN`** leaves the published surface contradicting ADR-0008 and regrows the drift.
  Sequence is G-0559, then G-0517, then this gap's tables."
- **Measured:** G-0559 is `status: addressed`, `addressed_by_commit: b778bc9f32ad…`
  (`b778bc9f3`, "Merge patch/G-0559-derive-idformat: derive the advertised id shape from
  the allocator's own facts", 2026-08-06 15:13:50 +0000), and lives at
  `work/gaps/archive/G-0559-…`. `aiwf schema` now prints `E-NNNN / M-NNNN / ADR-NNNN /
  G-NNNN / D-NNNN / C-NNNN`, and `grep -n 'IDFormat:' internal/entity/entity.go` returns
  nothing — the literals are gone. The gate has cleared. G-0560's body was last edited
  `8e7d4df38` on 2026-08-15, nine days after G-0559 closed, so the sentence survived a
  revision.
- **Command:** `aiwf show G-0559 --format=json`; `aiwf schema | grep -i "id format"`;
  `grep -n 'IDFormat:' internal/entity/entity.go`;
  `git log -1 --format='%h %ci %s' b778bc9f32ad147df12bb83df3fa6c7c032175ce`;
  `git log --format='%h %ci %s' -- work/gaps/G-0560-*.md`
- **Quoted:** "G-0559 gates it — the strings originate in `internal/entity/entity.go` and
  `aiwf schema` prints them, so widening the doc tables while the emitter still says `E-NN`…"

**`dead-premise`**

- **Claim:** "**ADR-0003 is `accepted` and unimplemented** — an accepted decision to add a
  seventh entity kind, against a kernel that hardcodes six. … This is a planning-state
  disposition, not a doc fix, and **nothing tracks it**."
- **Measured:** `ADR-0003 | rejected | Add finding (F-NNN) as a seventh entity kind |
  docs/adr/archive/ADR-0003-…`. It was promoted `accepted -> rejected` by `052a39869`
  (2026-08-16 10:08:48 +0000) and swept to archive by `9717f5ac8` (2026-08-19) — one day
  after G-0560's last body edit. The disposition the gap asks for has been made. (The rest
  of the bullet still holds: no `KindFinding` anywhere in `internal/`, no `work/findings/`;
  ADR-0001 is still `proposed` and D-0037 `Defer ADR-0001, G-0281, and EMB pending a
  measured id-collision trigger` is `accepted`.)
- **Command:** `aiwf show ADR-0003 --format=json`;
  `git log --format='%h %ci %s' -- docs/adr/archive/ADR-0003-*.md docs/adr/ADR-0003-*.md`;
  `grep -rn "KindFinding" --include='*.go' internal`; `ls work/`
- **Quoted:** "ADR-0003 is an accepted architectural decision the kernel contradicts, with
  nothing in the Normative tier marking it as pending."

### G-0562

**`already-fixed`**

- **Claim:** "`writeWorktreeHookScript` in `internal/policies/worktree_rituals_check_hook_test.go`
  writes the hook script it is about to exec with a bare
  `os.WriteFile(path, skills.WorktreeRitualsCheckScript, 0o755)`, rather than through
  `testsupport.WriteExecutable`." — and, in *Why it matters*, "It fires."
- **Measured:** false at HEAD. `internal/policies/worktree_rituals_check_hook_test.go:28-34`
  now reads:
  ```go
  func writeWorktreeHookScript(t *testing.T, dir string) string {
      t.Helper()
      path := filepath.Join(dir, "worktree-rituals-check.sh")
      if err := testsupport.WriteExecutable(path, skills.WorktreeRitualsCheckScript); err != nil {
          t.Fatalf("writing hook script: %v", err)
      }
      return path
  }
  ```
  The exact line the gap quotes was replaced by commit
  `793b1ad97b89e7830b1048c7ca311cd014953d99` — *"test(policies): route the hook-script fixture
  through WriteExecutable"*, authored 2026-08-19, an ancestor of HEAD
  (`git merge-base --is-ancestor 793b1ad97 HEAD` → yes). Its diff is the one-line swap plus
  the `internal/testsupport` import. The pre-fix file contained exactly one `os.WriteFile`
  (at line 30), so that single change closes the whole subject of both this gap and G-0578.
  A whole-file run of the policy's own detector over the current file reports zero violations.
- **Command:**
  ```
  grep -n "writeWorktreeHookScript\|os.WriteFile\|WriteExecutable" internal/policies/worktree_rituals_check_hook_test.go
  git log --oneline -- internal/policies/worktree_rituals_check_hook_test.go
  git log -S 'testsupport.WriteExecutable(path, skills.WorktreeRitualsCheckScript)' --oneline -- internal/policies/worktree_rituals_check_hook_test.go
  git show 793b1ad97 -- internal/policies/worktree_rituals_check_hook_test.go
  git show 793b1ad97^:internal/policies/worktree_rituals_check_hook_test.go | grep -n "os.WriteFile"   # -> exactly one, line 30
  git merge-base --is-ancestor 793b1ad97 HEAD && echo yes
  # plus a focused whole-file run of detectBareExecutableWrites over every _test.go in the
  # tree, added to $C and deleted after:
  go test -run 'TestAuditC42_WholeTreeBareExecWrites' -count=1 -v ./internal/policies/
  ```
- **Quoted:** commit body of `793b1ad97`: "A bare os.WriteFile holds a writable descriptor
  while the file is open; a concurrent fork inherits it, and execve on a file held open for
  writing fails with ETXTBSY, so the fixture step fails before the property under test is
  evaluated. WriteExecutable holds syscall.ForkLock across the write, excluding exactly those
  forks."

**`duplicate`**

- **Claim:** (implicit) G-0562 and G-0578 are separate gaps.
- **Measured:** **They are duplicates — same file, same helper, same single call site.**
  G-0578 does not name the helper, but the pre-fix file held exactly one bare executable
  write (`os.WriteFile(path, skills.WorktreeRitualsCheckScript, 0o755)` at line 30), which is
  the body of `writeWorktreeHookScript`, so both bodies can only be describing that line.
  Both are `status: open`. Both are closed by `793b1ad97`.
  What differs is only the observation each records, and those are two genuinely distinct
  events: G-0562 was filed 2026-08-06 (`$A history G-0562`, commit `c76980a`) recording a
  `make check-fast` failure of
  `TestWorktreeRitualsCheckHook_NotAWorktreeExitsZeroSilently` at
  `worktree_rituals_check_hook_test.go:86`; G-0578 was filed 2026-08-11
  (`$A history G-0578`, commit `e91a5ef`) recording a `make ci` failure during M-0306's wrap
  — and M-0306's own history shows its wrap actions on 2026-08-10/11
  (`$A show M-0306` → `status: done`, parent `E-0081`, *Referenced by: G-0578*).
  Corroboration for G-0562's cited line number: `runWorktreeHook` calls `t.Helper()`, so a
  `t.Fatalf` inside it is reported at the *caller's* line; in the pre-fix file, the
  `runWorktreeHook(...)` call inside `TestWorktreeRitualsCheckHook_NotAWorktreeExitsZeroSilently`
  is at line 86 exactly. The observation is internally consistent with the file as it stood.
- **Command:**
  ```
  $A history G-0562 ; $A history G-0578 ; $A show G-0578 ; $A show M-0306
  git show 793b1ad97^:internal/policies/worktree_rituals_check_hook_test.go | sed -n '48,92p' | cat -n
  ```
- **Quoted:** G-0578: "`internal/policies/worktree_rituals_check_hook_test.go` writes the hook
  script under test with a bare `os.WriteFile` carrying an exec bit, where this repo's
  convention is `testsupport.WriteExecutable`."

### G-0564

**`dead-premise`**

- **Claim:** "Two of them — composition tests across verb chains, and tree-level
  post-conditions under arbitrary legal composition — are mechanized by E-0080."
  Repeated in `Related`: "G-0121 — the parent. E-0080 mechanizes its composition
  sub-gaps and closes it" and "E-0080 — the epic whose scope this gap is the
  complement of."
- **Measured:** `E-0080` status is **`cancelled`**, path
  `work/epics/archive/E-0080-…/epic.md`. It was cancelled `2026-08-08T10:17:04Z`
  in `2f9c389` with the reason "Premise falsified: the walker's oracle is an
  absolute allowlist, not monotonic, and an invariant shape does not make an
  agreement property cheap. Successor direction in G-0121; the composition trap
  that motivated the epic is G-0572." **All five** of its milestones
  (M-0300…M-0304) carry `status: cancelled`. `G-0121` is `open`, priority
  `high`, and its own body was edited twice on 2026-08-08 *after* the
  cancellation. G-0564 was last edited `2026-08-06T19:00:54Z` (`6d5daf3`) — two
  days before — and has not been touched since. TODO.md itself records the
  reversal 16 lines below G-0564's own entry ("E-0080 was the mechanized half
  and is cancelled, so nothing closes this at anyone's wrap"), so the file
  contradicts itself between line 167 and line 183.
- **Command:**
  ```
  <audit>/aiwf show E-0080 --format=json        # status: cancelled
  <audit>/aiwf history E-0080                   # cancel 2026-08-08, 2f9c389
  git show --stat 2f9c389
  grep -H '^status:' work/epics/archive/E-0080-*/M-*.md   # all five: cancelled
  <audit>/aiwf show G-0121 --format=json        # status: open, priority high
  <audit>/aiwf history G-0564                   # last edit 2026-08-06
  ```
- **Quoted:** "Two of them — composition tests across verb chains, and
  tree-level post-conditions under arbitrary legal composition — are mechanized
  by E-0080."

### G-0571

**`superseded-premise`**

- **Claim:** "The narrower option is a create-time refusal on `aiwf add --body-file` and `aiwf edit-body --body-file`, which fires only on new content and would raise none." — presented as an open option.
- **Measured:** `ADR-0043` (status `accepted`, dated 2026-08-11, i.e. after this body's last edit on 2026-08-08) has already settled the question, and settled it differently and more broadly. Its Decision section reads: *"Body-section membership is enforced at the seams where body bytes are written, and nowhere else."* — **Seam one** is *"A scan over the bytes a verb is about to write, called by every body-supplying verb, refusing the write at error severity for every kind"* (every body-supplying verb, not the two `--body-file` paths), and **Seam two** is *"A gate riding the commit range the provenance audit already resolves, scoped to entities whose body content differs between the range base and HEAD."* `E-0084` ("Enforce body-section membership at the write seams", `proposed`) implements it and names `- Closing G-0571.` The body mentions neither record. A reader acting on the gap's sentence would design a narrower gate than the one already ratified.
- **Command:** `…/audit/aiwf show ADR-0043 --format=json`; `cat docs/adr/ADR-0043-enforce-body-section-membership-at-the-write-seams-never-tree-wide.md`; `…/audit/aiwf list --kind epic`; `grep -n 'G-0571\|ADR-0043' work/epics/E-0084*/epic.md`.
- **Quoted:** gap — "The narrower option is a create-time refusal on `aiwf add --body-file` and `aiwf edit-body --body-file`". ADR-0043 — "**Seam one — the verb.** A scan over the bytes a verb is about to write, called by every body-supplying verb".

### G-0573

**`contradicted-by-code`**

- **Claim:** title, plus "It calls `check.Run` directly, and applies none of the four
  `aiwf.yaml` severity passes the full `aiwf check` applies afterwards."
- **Measured:** the guard applies all four. `internal/verb/common.go:143-148`:
  ```go
  func projectionFindings(original, projected *tree.Tree) []check.Finding {
      pre := check.Run(original, nil)
      post := check.Run(projected, nil)
      policy := severity.Load(original.Root)
      severity.Apply(pre, policy, original)
      severity.Apply(post, policy, projected)
  ```
  `severity.Apply` (`internal/severity/severity.go:98-103`) composes exactly the four
  passes — `ApplyTDDStrict`, `ApplyAreaRequiredStrict`, `ApplyDocsStrict`,
  `ApplyArchiveSweepThreshold`. The function's own doc comment now paraphrases this
  gap's sentence as the reason the call exists.
- **Command:** `grep -rn "check\.Run(" --include='*.go' internal/ cmd/ | grep -v _test`;
  `sed -n '110,200p' internal/verb/common.go`; `cat -n internal/severity/severity.go`
- **Quoted:** body: "applies none of the four `aiwf.yaml` severity passes"

**`contradicted-by-code`**

- **Claim:** "So a knob that escalates a finding to error severity is invisible to every
  verb."
- **Measured:** false — falsified against the binary. In a fixture with
  `tdd: strict: true`, `aiwf import` of a manifest whose milestone declares no `tdd:`
  policy **refused**: exit 1, no commit (HEAD unchanged), reporting
  `error milestone-tdd-undeclared`. The rule's default severity is warning, so the error
  is the knob reaching the guard. This is pinned in-tree too, by
  `TestProjectionGuard_TDDStrictRefusesAnUndeclaredMilestone`
  (`internal/verb/projection_severity_test.go:56`), whose complement
  `…TDDStrictOffLeavesTheImportAlone` pins the same manifest importing cleanly with the
  knob unset.
- **Command:** in `r2` (`aiwf.yaml` = `hooks: {}` + `tdd:\n  strict: true`):
  `$AIWF import seed.yaml` → `EXIT=1`, `error milestone-tdd-undeclared`, `git log --oneline -1` unchanged
- **Quoted:** "a knob that escalates a finding to error severity is invisible to every verb"

**`dead-premise`**

- **Claim:** implied throughout — that the surviving symptom is a live, fixable defect
  of the guard.
- **Measured:** ADR-0042 (`Retire tdd.strict; require a complete body at the readiness
  promote`, status **accepted**, dated 2026-08-09, promoted 2026-08-10) addresses this
  gap by name in §Consequences: "G-0573's behavioural half becomes moot rather than
  fixed: with no knob able to make an empty draft body an error, `aiwf add epic` has
  nothing to misreport. Its structural half — seven `check.Run` call sites held in no
  relation to each other — is already closed by the shared seam." The same section
  states "The verb-time projection guard's severity application becomes inert." The ADR
  also gives the real cause of the surviving symptom, which I confirmed in source:
  `entityBodyEmpty` does `os.ReadFile(filepath.Join(t.Root, e.Path))`
  (`internal/check/entity_body.go:147`) and `continue`s on a missing file, so a
  projected-but-unwritten entity yields nothing to escalate — a mechanism the gap body
  never names. `tdd.strict` itself has not yet been retired (`internal/config/schema.go:43`
  still declares it; `severity.Policy.TDDStrict` still exists), so the symptom is live
  today while the tracker for it is this dead body.
- **Command:** `$AIWF show ADR-0042`; `cat docs/adr/ADR-0042-retire-tdd-strict-*.md`;
  `sed -n '140,200p' internal/check/entity_body.go`; `grep -rn "strict" internal/config/schema.go`
- **Quoted:** ADR-0042: "G-0573's behavioural half becomes moot rather than fixed"

### G-0574

**`overclaimed-absence`**

- **Claim:** "What is missing is a decided answer to what makes two findings the same finding for the purpose of the guard … That decision is the prerequisite for both repairs above, and it is currently unwritten."
- **Measured:** `D-0046` — *Diff the shared contract gate by finding identity, not the full struct* — exists, status `accepted`, decided 2026-07-21 (i.e. 18 days before G-0574 was filed on 2026-08-08). It decides exactly this question for `internal/verb/contractgate.go`, names the identity subset (`Code`, `Severity`, `EntityID`, `Subcode`, `Path`), and excludes `Message` for precisely the reason G-0574 describes (a message that interpolates positional/derived state makes an unchanged finding read as introduced). Its **Consequences** section is not scoped to the contract gate: *"Any future consumer that diffs two `contractcheck.Run` (or similarly shaped) result sets for equality should treat `Message` (and any other `Run()`-computed prose field) as unsafe for an equality/identity key across two separate invocations whose input differs by more than the field under test."* `projectionFindings` is a similarly-shaped consumer of `check.Run`. The claim is not merely unsupported — a written decision points the opposite way and the body does not mention it.
- **Command:** `…/audit/aiwf show D-0046`; `cat work/decisions/D-0046-diff-the-shared-contract-gate-by-finding-identity-not-the-full-struct.md`; `…/audit/aiwf show G-0574` (add date 2026-08-08); `grep -rln "finding identity\|findingKey\|same finding" docs/ work/`
- **Quoted (gap):** "That decision is the prerequisite for both repairs above, and it is currently unwritten."
- **Quoted (D-0046):** "Key the diff on a `findingIdentity` subset — `Code`, `Severity`, `EntityID`, `Subcode`, `Path` — instead of the full `check.Finding` struct. `Message`, `Line`, `Hint`, and `Field` are excluded from the identity".

### G-0577

**`contradicted-by-code`**

- **Claim:** "…`IsTerminalACStatus` drives the AC cancel path's convergence guard, so flipping a terminal silently changes what `aiwf cancel` does with no test to catch it."
- **Measured:** Two tests catch it, in two packages. Patching `acTransitions["deferred"]` from `{}` to `{"open"}` in a clone: `TestCancelAC_TerminalStatus_ReturnsNoOp/deferred` fails at `ac_same_state_noop_test.go:48` with *"fixture assumes \"deferred\" is terminal in the AC FSM; it is not"*, and `TestIsLegalACTransition_AllPairs/deferred->open` fails at `transition_test.go:196` with *"IsLegalACTransition(\"deferred\", \"open\") = true, want false"*. Nothing about it is silent. (Nuance worth keeping: the first is a fixture-precondition `t.Fatalf`, so it fails *before* exercising `aiwf cancel`'s behaviour — but it fails CI, which is what "no test to catch it" denies.) The half of the sentence that is true: `IsTerminalACStatus` does drive `cancelAC`'s convergence guard — `internal/verb/ac.go:279`.
- **Command:** in the clone, patch `internal/entity/transition.go` then `go test -run 'TestCancelAC_TerminalStatus_ReturnsNoOp' -count=1 ./internal/verb/` and `go test -run 'TestIsLegalACTransition_AllPairs|TestIsTerminalACStatus' -count=1 ./internal/entity/`.
- **Quoted:** "flipping a terminal silently changes what `aiwf cancel` does with no test to catch it"

### G-0578

**`already-addressed`**

- **Claim:** "`internal/policies/worktree_rituals_check_hook_test.go` writes the hook
  script under test with a bare `os.WriteFile` carrying an exec bit, where this repo's
  convention is `testsupport.WriteExecutable`."
- **Measured:** False at HEAD. The file's only executable write is line 31,
  `if err := testsupport.WriteExecutable(path, skills.WorktreeRitualsCheckScript); err != nil {`,
  inside `writeWorktreeHookScript` at line 28. The remaining `0o755` literals in the file
  (lines 108, 139) are `os.MkdirAll` on a directory, not an executable stand-in. The real
  whole-tree detector run confirms it independently: `worktree_rituals_check_hook_test.go`
  appears zero times in its 90-site output. The fix is commit `793b1ad97`
  ("test(policies): route the hook-script fixture through WriteExecutable",
  2026-08-19 14:36 +0000), whose diff is exactly `-os.WriteFile(path, skills.WorktreeRitualsCheckScript, 0o755)`
  → `+testsupport.WriteExecutable(path, skills.WorktreeRitualsCheckScript)`. It is an
  ancestor of HEAD. It carries no `aiwf-entity:` trailer, which is why neither gap was
  promoted.
- **Command:** `grep -n "WriteFile\|WriteExecutable\|writeWorktreeHookScript\|0o755" internal/policies/worktree_rituals_check_hook_test.go`;
  `git log --oneline -- internal/policies/worktree_rituals_check_hook_test.go`;
  `git show 793b1ad97 -- internal/policies/worktree_rituals_check_hook_test.go`;
  `git merge-base --is-ancestor 793b1ad97 HEAD`;
  `AIWF_COVERAGE_BASE=$(git hash-object -t tree /dev/null) go test -run 'TestPolicy_TestExecutableWrite$' -count=1 ./internal/policies/` then `grep -c worktree_rituals`
- **Quoted:** body — "writes the hook script under test with a bare `os.WriteFile`
  carrying an exec bit"

**`duplicate`**

- **Claim:** implicit — the gap is filed as a fresh finding and references no sibling.
- **Measured:** G-0578 and G-0562 name **the same single call site**, not two.
  G-0562's body names `writeWorktreeHookScript` in
  `internal/policies/worktree_rituals_check_hook_test.go` and quotes the exact call
  `os.WriteFile(path, skills.WorktreeRitualsCheckScript, 0o755)`; G-0578 names the same
  file and the same write without naming the helper. Counting that file at G-0497's
  creation commit with the detector clone shows it held exactly **one** matching site,
  line 30 — so there is no second site for the two gaps to be about. G-0562 was filed
  `c76980a55` 2026-08-06 16:40; G-0578 `e91a5eff2` 2026-08-11 11:23 — five days later,
  with no cross-reference in either direction (`grep` for "G-0562" in G-0578 and vice
  versa: no hits). They are independent *sightings* — G-0562 records a `make check-fast`
  failure on 2026-08-06, G-0578 a `make ci` failure during M-0306's wrap on 2026-08-11 —
  of one defect. Both are subsumed by G-0497, whose `internal/policies` row of 8 included
  this site (verified: the clone lists `worktree_rituals_check_hook_test.go:30` among the
  8 at commit `105f2ab70`).
- **Command:** `cat -n work/gaps/G-0562-*.md`;
  `git log --follow --format='%h %ci %s' -- work/gaps/G-0578-*.md work/gaps/G-0562-*.md`;
  `git archive 105f2ab70 internal | tar -x -C at0497 && python3 detect_at.py -v | grep worktree_rituals`
- **Quoted:** G-0562 — "`writeWorktreeHookScript` in
  `internal/policies/worktree_rituals_check_hook_test.go` writes the hook script it is
  about to exec with a bare `os.WriteFile(path, skills.WorktreeRitualsCheckScript, 0o755)`"

### G-0580

**`dead-premise`**

- **Claim:** "The skill-edit **structural-test** backstop fails the profile-driven
  gate when a `SKILL.md` under the embedded rituals changes and **no test under
  `internal/policies/` references its path**."
- **Measured:** that predicate no longer exists. Commit `667c8e0fc`
  (`feat(policies): ask a skill edit who owns it, not whether a test names it
  (M-0312)`, 2026-08-21) **deleted**
  `internal/policies/skill_edit_structural_test_backstop.go` (−176) and
  `…_test.go` (−438) and added `skill_edit_provenance_backstop.go` (+248). The
  live predicate is a commit trailer: a commit touching a watched `SKILL.md` must
  carry an `aiwf-entity:` trailer resolving to a real entity. The current source
  says so in terms: "What it does NOT require is a policy test referencing the
  edited path. That was the predicate M-0196 shipped, and D-0071 retired it".
  `D-0071` resolves (`aiwf show D-0071` → "Enforce provenance, not content, at the
  skill-edit backstop · status: **accepted**"), and its Decision section reads
  "It no longer requires that a policy test reference the edited path." The repo
  additionally pins the retirement: `TestSkillEditProvenance_DocumentedInClaudeMd`
  fails if `CLAUDE.md`'s two enforcement sections contain "structural test",
  "structural-test", or "skill-edit-structural-test-backstop". G-0580 was filed
  2026-08-11 (`aiwf show G-0580`), so the description was correct when written and
  is now false.
- **Command:** `git show 667c8e0fc --stat`;
  `sed -n '15,60p' internal/policies/skill_edit_provenance_backstop.go`;
  `aiwf show D-0071`; `sed -n '370,400p' internal/policies/skill_edit_provenance_backstop_test.go`
- **Quoted (D-0071, Decision):** "The skill-edit backstop enforces **provenance**:
  an edit to a shipped surface must ride a commit whose `aiwf-entity` trailer names
  an owning entity that exists. It no longer requires that a policy test reference
  the edited path."

### G-0586

**`contradicted-by-code`**

- **Claim:** "It directs the reader to `aiwf show`, and no entity template carries a section in which an unsettled claim would appear."
- **Measured:** False, twice over. (a) The shipped epic template `internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/epic-spec.md` carries `## Open questions` — a table with columns `Question | Blocking? | Resolution path`. It has been there since `8a2c8acdf` (2026-05-29), i.e. before this gap was filed on 2026-08-15, so this is not drift. 54 entity files in `work/` currently carry a `## Open questions` section. (b) `aiwf show` does surface it: `aiwf show E-0086 --format=json` returns body key `open_questions`. The milestone template additionally carries `## Deferrals` ("Work this milestone deliberately punted") and `## Reviewer notes`, both surfaced by `aiwf show M-0312 --format=json` as `deferrals` / `reviewer_notes`. This is the load-bearing sentence for "the pointer does not close the hole either", and it does not hold.
- **Command:**
  - `for f in $(find internal/skills/embedded-rituals -ipath '*template*' -name '*.md'); do grep -n '^#' "$f"; done` → `epic-spec.md:51:## Open questions`
  - `sed -n '51,57p' internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/epic-spec.md`
  - `<audit>/aiwf show E-0086 --format=json | python3 -c "…"` → `body keys: [... 'open_questions', ...]`
  - `<audit>/aiwf show M-0312 --format=json | python3 -c "…"` → `[... 'deferrals', ... 'reviewer_notes', ...]`
  - `grep -rln "^## Open questions" work/ | wc -l` → `54`
  - `git log -S "## Open questions" --oneline --date=short -- …/templates/epic-spec.md` → `8a2c8acdf 2026-05-29`
- **Quoted:** template — *"## Open questions\n\n| Question | Blocking? | Resolution path |\n|---|---|---|\n| \<question\> | \<yes/no\> | \<where/when it gets resolved\> |"*

### G-0587

**`contradicted-by-code`**

- **Claim:** the title and the second paragraph — a shipped skill instructing a
  corpus-reading review "has to say where they are, and the only vocabulary available is the
  paths it is forbidden to name."
- **Measured:** shipped surfaces already name exactly that corpus by path, ship, and pass.
  `agents/planner.md:31` reads: "Project conventions: tech stack, constraints, prior
  decisions captured in ADRs (`docs/adr/`) and D-NNNN entries (`work/decisions/`)." That is
  the ADR corpus and the decision corpus, named by directory, in a shipped role card.
  Across the whole shipped tree, **32 distinct lines in 8 files** carry a backticked `docs/`
  path — including four lines citing a real aiwf repo doc *file* (`aiwf-add/SKILL.md:243`
  twice, `:276`, `aiwf-check/SKILL.md:165`, naming `docs/archive/pocv3/acs-and-tdd-plan.md`,
  `docs/design/design-decisions.md`, `docs/design/tree-discipline.md`, all of which exist).
  `aiwf check` over that tree reports **zero** findings.
- **Command:**
  `grep -rnoE '`[^`]*docs/[^`]*`' --include='*.md' internal/skills/` (36 tokens on 32 distinct
  lines); `sed -n '25,45p' internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/planner.md`;
  `/…/audit/aiwf check --format=json` → `{"status":"ok","findings":[], …"findings":0}`.
- **Quoted:** gap body — "A skill instructing that review has to say where they are, and the
  only vocabulary available is the paths it is forbidden to name." Shipped
  `agents/planner.md:31` — "prior decisions captured in ADRs (`docs/adr/`) and D-NNNN
  entries (`work/decisions/`)."

### G-0589

**`dead-premise`**

- **Claim:** "It will fork a third time at implementation. A shipped review pass has to
  tell a reviewer what to read, and its only options are to point at a hierarchy that
  omits the root files and cannot say *derived*, or to restate the classification again."
- **Measured:** No review pass will ship. `D-0069` — *Reject the dispatched reading
  pass; keep the lab rule* — was added and accepted 2026-08-19, four days after this
  gap was filed. The preflight initiative's own `### Promotion status` (dated
  2026-08-19) records "**Dropped** — T1, T2, T3, T6, T7, T8, T9, and with them the
  sweep…" and "Two implementation attempts, both terminal": `E-0085` is `cancelled`,
  `E-0086` is `done` "closed re-scoped to the lab rule". CLAUDE.md's
  preflight-implementation section was removed at `fe3862ee9` ("drop the
  preflight-implementation section, whose subject was rejected"). What survives is the
  lab *rule* only — "a claim is settled true only where a command, its expected result,
  its observed output and its environment sit together" — which names no document corpus.
- **Command:** `…/audit/aiwf show D-0069` ; `…/audit/aiwf show E-0085` ; `…/audit/aiwf show E-0086` ; `sed -n '/^### Promotion status/,/^## Open questions/p' docs/initiatives/milestone-preflight-as-independent-review.md` ; `git log --oneline -15 --since=2026-08-15 -- CLAUDE.md`
- **Quoted:** "It will fork a third time at implementation. A shipped review pass has to tell a reviewer what to read"

### G-0590

**`false-claim`**

- **Claim:** The HTML render path carries the cancel reason too, so `show` is the only surface that omits it — the field is "already rendered by two other surfaces".
- **Measured:** The HTML render omits it entirely. `htmlrender.HistoryRow` has no `Body` field at all; `internal/cli/render/resolver.go:666` populates `Reason: e.Reason` (the `aiwf-reason:` trailer, which no cancel commit carries — see the next finding), and no template renders even that: `epic.tmpl:80-93` renders a four-column Date/Verb/Detail/Actor table, and `milestone.tmpl:174-184` renders Date/Verb/Actor plus force/audit/scope chips. Rendering the site and grepping it confirms: the E-0058 cancel reason appears in no rendered page except `G-0590.html` (this gap's own body) and `ADR-0031.html`. Two surfaces *do* carry it — `aiwf history` text and the JSON envelopes — but not the one the body names.
- **Command:** `aiwf render --format=html --out <scratch>/site-c22 --scope E-0058` → `files_written: 1150`; then `grep -rl "adversarial reviews" <scratch>/site-c22` → `G-0590.html`, `ADR-0031.html` only; `sed -n '/Recent activity/,/<\/table>/p' E-0058.html` → three rows, no reason; `grep -rn "Reason\|\.Body" internal/htmlrender/embedded/*.tmpl`
- **Quoted:** "The HTML render path carries the same field. Only `show` omits it." / "already written, already parsed, and already rendered by two other surfaces."

### G-0594

**`overclaimed-attribution`**

- **Claim:** "A shipped skill defined the reading state *ambiguous* as the
  subject neither asserting nor denying a premise. **The specification's only
  gloss is** 'that a sentence carries a second reading' — a claim with two
  readings, not one with none."
- **Measured:** "the specification" resolves to
  `docs/initiatives/milestone-preflight-as-independent-review.md` — the quoted
  phrase exists at exactly one place in the whole tree, that file's line 317.
  That file carries a **second, explicit gloss** of the same term at line 333,
  inside a list introduced at line 329 by "The four are distinct, and the
  distinction is what makes each actionable:" — "**Ambiguous** — the text
  addresses the point and can be read more than one way. Nothing is false; a
  builder reading it the other way builds the other thing. The fix is to
  disambiguate the text, not to measure anything." So "only" is false. The
  definition was added by `01f3f37da` (2026-08-15 09:21:56), never removed
  (`git log --all -S` returns that one commit), and `01f3f37da` **is an ancestor
  of** `14e4f9a09`, the 2026-08-21 11:32 `edit-body` that wrote this sentence —
  so the fuller gloss was present and reachable when the claim was made. The
  gap's *conclusion* is right (line 333 says exactly "can be read more than one
  way", which is a claim with two readings, not one with none); its
  **attribution** is not. This is the failure class G-0594 documents, occurring
  inside G-0594.
- **Command:**
  `grep -n "ambiguous\|Ambiguous" docs/initiatives/milestone-preflight-as-independent-review.md`
  → lines 29, 306, 318, **333**, 556;
  `sed -n '326,345p' docs/initiatives/milestone-preflight-as-independent-review.md`;
  `grep -rn "second reading" work/ docs/ internal/`;
  `git log --all --oneline -S 'the text addresses the point and can be read more than one way' -- docs/initiatives/milestone-preflight-as-independent-review.md`;
  `git merge-base --is-ancestor 01f3f37da 14e4f9a09`
- **Quoted:** body line 84–86: "The specification's only gloss is \"that a
  sentence carries a second reading\" — a claim with two readings, not one with
  none." vs. the record's line 333: "**Ambiguous** — the text addresses the
  point and can be read more than one way."

### G-0600

**`contradicted-by-code`**

- **Claim:** "Stamping is uneven. **Only the guidance fragment carries a version today**, so a
  comparison covers one family and the rest stay silent. Extending the stamp to the other
  materialized families is the same change or an explicit non-goal, not an oversight to discover
  later." And, separately: "Comparing versions is not a plain ordering… A naive string or semver
  comparison gets the aiwf-repo case backwards, which is the case where the trap fires most."
- **Measured:** Both paragraphs are answered by shipped code the gap does not mention.
  The **statusline** carries a version stamp today —
  `internal/skills/embedded-statusline/statusline.sh:2`:
  `# aiwf-statusline version: __AIWF_VERSION__ — managed by aiwf; regenerated by 'aiwf update'`.
  It is read back — `statuslineVersionRE` (`internal/skills/statusline.go:26`) and
  `InstalledStatuslineVersion` (`:41`). And `decideStatuslineRefresh` (`:135`) already implements
  the exact never-downgrade contract the gap's Resolution shape asks for, including the
  can't-order case the gap raises as an open design question:
  `case version.SkewBehind: return RefreshActionSkipped, "installed %s is newer than this binary %s — not downgrading"`
  and `case version.SkewUnknown: … "cannot order versions (installed %q, binary %q) — not auto-refreshing"`.
  `version.Compare` (`internal/version/version.go:199`) returns `SkewUnknown` whenever either
  side is untagged or carries a pre-release/build suffix — so the dirty-dev-build case does not
  "get it backwards", it declines. Plain `aiwf update` invokes this
  (`internal/cli/update/update.go:229`, `skills.AutoRefreshStatusline(rootDir)`), and the
  statusline is the *only* materialized family whose writer reads the destination first — the
  sole `os.ReadFile` on a destination in `internal/skills/` outside the settings writers is
  `statusline.go:190`.
  Driven end to end in one fixture run, with a `v99.0.0`-stamped statusline and a
  `v99.0.0`-stamped guidance fragment side by side, the same `aiwf update` printed:

      updated    .claude/aiwf-guidance.md  (materialized from embedded guidance)
      skipped    statusline (project scope)  (cannot order versions (installed "v99.0.0", binary "v0.32.1-0.20260823122915-da34c1009627+dirty") — not auto-refreshing)

  After it: the statusline sentinel line survived and its stamp still read `v99.0.0`; the
  guidance sentinel clause was gone and the stamp had been rewritten to the binary's. The gap
  names as unprotected the one family that is protected.
- **Command:** fixture at `…/audit/tmp/C7-3/fx3`, `git init` + `aiwf.yaml` + one epic; a
  `.claude/aiwf-guidance.md` built from the embedded source with `__AIWF_VERSION__` → `v99.0.0`
  plus a sentinel clause, and a `.claude/statusline.sh` built the same way with a sentinel
  comment; then
  `env HOME=…/fakehome XDG_CONFIG_HOME=… XDG_STATE_HOME=… …/audit/aiwf update --root .`;
  plus `grep -rn "AutoRefreshStatusline\|ScaffoldStatusline" --include='*.go' internal/ cmd/`,
  `grep -rn "os.ReadFile" internal/skills/*.go`, `sed -n '110,155p' internal/skills/statusline.go`,
  `sed -n '190,225p' internal/version/version.go`.
- **Quoted:** "Only the guidance fragment carries a version today, so a comparison covers one
  family and the rest stay silent."

### G-0601

**`contradicted-by-code`**

- **Claim:** "`aiwf history <id>` renders a row only for a commit carrying `aiwf-verb`." (opening sentence) and "Every row that projection does render carries a verb." (end of §What's missing)
- **Measured:** The projection's drop rule is `verb == "" && actor == ""` — a commit carrying `aiwf-entity` + `aiwf-actor` and **no** `aiwf-verb` renders, with a blank verb column. 19 such commits exist on `main`. `aiwf history M-0158` renders one of them (`1951f02`, subject `feat(spec): M-0158/AC-1+AC-7+AC-8 — scaffold layer-4 branch package`) with an empty verb field. Both claims are false, and were false when the gap was filed (`internal/entityview/` has no commit in `667c8e0f..main`).
- **Command:**
  - `git log --format='%H%x1f%(trailers:key=aiwf-entity,valueonly,separator=|)%x1f%(trailers:key=aiwf-verb,valueonly,separator=|)%x1f%(trailers:key=aiwf-actor,valueonly,separator=|)' main | awk -F'\x1f' '$2!="" && $3=="" && $4!="" {c++} END{print c+0}'` → `19`
  - `<audit>/aiwf history M-0158 | head -30` → row for `1951f02` with no verb rendered
  - `sed -n '190,200p' internal/entityview/historyevent.go` (source, second-best) → `if verb == "" && actor == "" { continue }`
  - `git log --oneline 667c8e0f..main -- internal/entityview/` → empty
- **Quoted:** body — *"`aiwf history <id>` renders a row only for a commit carrying `aiwf-verb`."* / *"Every row that projection does render carries a verb."*; code — `internal/entityview/historyevent.go:195` `if verb == "" && actor == "" {` under the comment *"A genuine entity event always carries both."*

### G-0604

**`contradicted-by-code`**

- **Claim:** "Each spells the same walk — resolve by current id, fall back to prior ids, then scan the stub list."
- **Measured:** false for every site except the backstop. `internal/check` consults `prior_ids` nowhere in these walks — `grep` for `PriorIDs`/`ByPriorID`/`ResolveByCurrentOrPriorID` in non-test Go finds the prior-id arm only in `internal/tree/tree.go`, `internal/cli/history/history.go:87`, `internal/policies/skill_edit_provenance_backstop.go:169`, and (through `resolveViaPriorIDs`) `internal/check/provenance.go` + `promote_on_wrong_branch.go` — none of which is one of the cited copies. Confirmed behaviourally in a fixture: a `relates_to: D-0009` where `D-0009` lives only in another entity's `prior_ids:` yields `refs-resolve` / subcode `unresolved`, severity **error** — i.e. `refsResolve` does not fall back to prior ids. The same fixture with `D-0009` in body prose yields `body-prose-id` / `unresolved`, severity **error** — `BodyProseIDIndex` does not either. Expected, had the body been true: both silent.
- **Command:** fixture `…/tmp/C9-3/fix2` (`git init`; `work/decisions/D-0001-a.md` referencing `D-0009`; `work/decisions/D-0003-c.md` carrying `prior_ids: [D-0009]`; one commit) then `aiwf check --root …/fix2 --format=json --pretty`
- **Quoted:** "Each spells the same walk — resolve by current id, fall back to prior ids, then scan the stub list."
- **Why it matters for a reader:** the body's stated fix is "a single function that answers *is this id claimed by anything in the tree*". Built to the body's description it would carry a prior-ids fallback, and dropping it into `refsResolve` / `BodyProseIDIndex` / `idsUnique` / `DisputedTrunkIDs` would silently widen resolution at four rules, one of which (`idsUnique`) emits blocking errors.

### G-0606

**`false-count`**

- **Claim:** "The live instance is `m0211-guidance-operating-anchors` …" and "The cost of
  leaving it is small today, since one instance exists and it is documented. It grows the
  moment a second policy reaches for the shape, because the precedent will read as
  sanctioned."
- **Measured:** a second live instance exists and predates the first.
  `PolicyM0210TrailerCommitDrift` (`internal/policies/m0210_trailer_commit_drift.go`) reads
  every shipped ritual body under
  `internal/skills/embedded-rituals/plugins/*/skills/*/SKILL.md` (`:51-52`) and compares it
  against phrases written into the policy source:
  `m0210HasCaveat` → `strings.Contains(l, "variant casing") && strings.Contains(l, "trailer-keys policy")` (`:144`),
  `m0210HasIdentityRule` → `strings.Contains(l, "git config user.email") && strings.Contains(l, "do not hardcode")` (`:158`),
  `m0210HasStagedMerge` → `strings.Contains(body, "git merge --no-ff --no-commit")` (`:150`).
  Those are prose phrases over shipped prose, in a production policy, exactly the shape the
  gap describes. The file was added `2026-07-01` (`19aa526c5`), one day **before**
  `m0211_guidance_operating_anchors.go` (`d52f037ff`, `2026-07-02`), and both existed at the
  gap's own creation commit `a09bba763` (2026-08-22). Unlike m0211 it is **not** documented
  in CLAUDE.md as a deliberate anchor set — `grep -n "M0210" CLAUDE.md` returns nothing —
  so the gap's "and it is documented" does not cover it either.
- **Command:**
  ```
  grep -rln "embedded-guidance\|embedded-rituals\|embedded-statusline\|internal/skills/embedded" --include="*.go" internal/ cmd/ | grep -v _test.go
  grep -n "strings.Contains" internal/policies/m0210_trailer_commit_drift.go
  git log --format='%h %ad %s' --date=short -1 --diff-filter=A -- internal/policies/m0210_trailer_commit_drift.go
  git cat-file -e a09bba763:internal/policies/m0210_trailer_commit_drift.go
  ```
- **Quoted:** gap body — "The live instance is `m0211-guidance-operating-anchors`"; and
  "since one instance exists and it is documented".

### G-0613

**`contradicted-by-code`**

- **Claim:** "The wrap ritual is now the only surface naming a category set, so whichever
  way this settles, it settles in one place."
- **Measured:** Two shipped surfaces name a closed category set, not one.
  `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md:64`
  carries `### <Added|Changed|Fixed> — E-NNNN: <one-line summary>`, and
  `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md:68`
  carries "using a Keep-a-Changelog category as the heading: `### Added — G-NNNN: …`,
  `### Changed — G-NNNN: …`, or `### Fixed — G-NNNN: …`". The `wf-patch` set is not new
  — `git log -S` dates it to `7d9c2bc6f` (2026-07-05, "fix(rituals): wf-patch requires a
  CHANGELOG entry, every time (G-0365)"), six weeks before G-0613 was filed (`cde6e080e`,
  2026-08-22) and before its only body edit (`89d704d53`, same day). So the claim was
  false when written. A third file, `aiwfx-release/SKILL.md:81,85`, shows `### Changed`
  and `### Fixed` inside a fenced example of *already-written* entries — illustrative,
  not an enumerated set, so the gap's "the release ritual now names no category set at
  all" stands.
- **Command:**
  `grep -rn "### Added\|### Removed\|### Changed\|### Fixed\|Added|Changed" internal/skills/`;
  `awk 'NR>=60 && NR<=75' internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md`;
  `git log --format='%h %ci %s' -S 'Keep-a-Changelog category as the heading' -- internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md`;
  `git log --follow --format='%h %ci %s' -- work/gaps/G-0613-*.md`
- **Quoted:** body — "The wrap ritual is now the only surface naming a category set, so
  whichever way this settles, it settles in one place." · wf-patch SKILL.md:68 — "Add a
  new sub-section under `## [Unreleased]` in `CHANGELOG.md`, using a Keep-a-Changelog
  category as the heading: `### Added — G-NNNN: <one-line summary>`, `### Changed —
  G-NNNN: <one-line summary>`, or `### Fixed — G-NNNN: <one-line summary>`"

## Recommended dispositions

What a human should do with each subject, as its auditor recommended. These are recommendations against the tree at `da34c1009`, not decisions.

- **G-0060** — keep open at high priority, but `aiwf edit-body G-0060` to rewrite the "What's missing" and "Why it matters" sections. Specifically: drop the "optional `wf-rituals` plugin … TDD cycle / code-review / doc-lint surface" framing (ADR-0014 retired it); delete or invert the "No branch model" and "`aiwf check` has no patch-shaped invariants" bullets (both false — `internal/branchparse`, `legalRungPairs`, `isolation-escape`); rewrite the "no formal way to record `closes G-NNN`" bullet to say what is actually missing (the record exists on the *gap* side as `addressed_by_commit`; what is absent is a queryable patch-side record and any check of the SHA's meaning); fix `addressed_by: <commit-sha>` → `addressed_by_commit:`; refresh the three `file:line` citations; refresh the Go snippet against today's `Schema` type; and weigh ADR-0045 against option 2.
- **G-0068** — `aiwf edit-body G-0068`. The load-bearing correction: the composite is captured only when **both** `Code:` and `Subcode:` are string literals, and `internal/check/` now has **zero** literal `Code:` sites (all 91 use named constants since `56ad4b841`, 2026-05-31, "type finding-code constants and enforce adoption"). So `entity-body-empty/ac` is *not* in the required set either, and the "Static path" fix as written would change nothing — `stringFieldValue(cl, "Code")` returns `""` and the walker returns before it ever looks at `Subcode`. Retitle accordingly (the defect is "misses every composite subcode", not "misses the dynamic ones"). Worth recording as the cheap first half of the fix: the sibling collector `emittedFindingCodeSites` (`finding_hints.go`:137) already resolves constant-valued `Code:` via `resolveStringExpr`; routing `allCheckCodes` through it restores every literal-Subcode composite, leaving only the genuinely dynamic ones for the gap's static/convention choice. Also fix the `:133` line ref.
- **G-0070** — do **not** promote addressed — the flag is still absent. But the body needs a rewrite, not a patch: `aiwf edit-body G-0070` to strip the entire `recommended-plugin-not-installed` framing (paragraph 2, paragraph 3's first clause, and the closing sentence), re-derive the section enumeration from the current 21 `label()` strings plus `plugin-mount:`, replace the fabricated CLAUDE.md quotation with the real line-285 sentence, and state ADR-0026 as the record that answered the machine-readable- doctor-state question by a different route. Worth asking the human whether, with ADR-0026 in place and `--write-health` shipping, the remaining ask is still wanted.
- **G-0073** — keep open; `aiwf edit-body G-0073` to (a) rebuild the call-site table — the count is no longer six, one path no longer exists, and four of six line numbers moved; (b) delete or replace the three dead cross-kind examples (E-0019/ADR-0003/ADR-0001 and the "Implementation-epic chains" bullet), which ADR-0045 killed; (c) delete the "future finding" arms of fix-shape items 1 and 4; (d) rewrite the 2026-08-12 section — **both** of its instances have been settled since, the epic template on 2026-08-22 and the E-0083/E-0084 spec disagreement 17 seconds after the section was written — keeping only the *general* point (no `depends_on` edge expresses joint co-ordination), which E-0083's own risk table now cites this gap for; (e) fix `linked_adr` → `linked_adrs`; (f) drop the stale "(writer verb pending per G-0072)" and "G-0072 stays open" sentences.
- **G-0110** — `aiwf edit-body G-0110` (and `aiwf retitle`) to restate the defect as measured: `--diff` skips **every** mutant — new *and* modified — whenever the path argument is below the module root; running `--diff` from the module root works correctly, including for entirely-new files. Then drop the stale CLAUDE.md quote, and drop the "operators need a working diff-scoped mutation run" framing, since `make mutate-diff` (G-0267, `addressed`) shipped one that sidesteps `--diff` entirely. Also fix the TODO.md dependency claim.
- **G-0111** — `aiwf edit-body G-0111` to (a) delete or rewrite concern 4 as "the *declarative* half is still missing", crediting `9cad2af3e` / G-0431 for the skill-driven half; (b) replace the `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md` authoring instruction with `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md` and drop the "copy at wrap" cross-repo pattern; (c) fix concern 2's `--force --reason` framing, which misdescribes the start-side rule; (d) either scope `aiwf authorize --end` into the work or stop naming a flag that has never existed; (e) drop "no preflight" and the "sovereign-act declaration" bundle item. Then leave the gap open at `high` — concerns 1–3 are genuinely unimplemented and no epic covers them.
- **G-0121** — `edit-body`. Four sentences need correcting: the two present-tense "live and unfixed" citations (G-0567 and G-0558 are both `addressed`), the "*One read path* … cannot compare two surfaces" measurement (the walker already compares `aiwf list` against `tree.Load` after every step), the walker's operation list (`reallocate` is not in it; `retitle` is), and "three seed acceptance criteria" (two do). Also refresh the two "~" inventory counts in §"What's missing" and reconcile that section's flat "Each verb is tested in isolation" with §Notes, which contradicts it. Leave the gap open.
- **G-0161** — keep open and raise priority. `aiwf edit-body G-0161` to (a) correct the ANTI-0001, ANTI-0007, ANTI-0008 and ANTI-0012 sketches, (b) fix "18 cells" → 30, (c) drop "explicitly out of scope for M-0125", (d) delete the trailing `## Status` section. Separately, ANTI-0001, ANTI-0007, ANTI-0008 and ANTI-0012 in `internal/workflows/spec/antirules.go` each need a decision — amend the anti-rule or remove it — which is arguably its own entity.
- **G-0168** — `aiwf edit-body G-0168` to (a) delete the `milestone | tdd:` row from the "missing" table and the whole `## Re-discovery (2026-06-26)` section — the verb, its uniform-ordinary gating and its refuse-with-hint all landed exactly as written; (b) delete the entire `## Workaround (current)` section, which now recommends a commit the installed `commit-msg` hook refuses at exit 1; (c) delete `aiwf rewidth` from the comparison table (retired by ADR-0039, accepted); (d) replace the `## Downstream report` design-fork section with a one-line pointer to D-0048; (e) soften "an operator must hand-edit the markdown file and commit manually" — `aiwf import --on-collision update` is a trailered kernel route to the same three fields. What survives is a short gap: "three set-at-create reference fields have no per-kind subverb; shape fixed by D-0048, code deferred until demanded." Do **not** promote `addressed` — three fields remain.
- **G-0169** — `aiwf edit-body G-0169` to rewrite the body around the three surviving commands: delete the "Mutating, bespoke output path" bullet entirely (import shipped; `rewidth` no longer exists), fix "Why it matters" to name `render roadmap` alone as the unknown-flag example, and correct the Notes sentence about `formatExempt` (rewidth's entry *was* cleaned up; import's remains; `render roadmap`'s never named this gap). Then keep it open. Retitling to drop "non-FinishVerb verbs" would also be right — nothing in the surviving scope is a mutating verb.
- **G-0212** — do **not** promote `addressed` — the residual scope is real but undefined. `edit-body` the "Known classes" list so each of the six items records the scenario that now exercises it and what that scenario found (see the per-item table below), re-derive or delete the "26 reallocate commits" number, and delete the "Out of scope for E-0030" closing paragraph (E-0030 is `done` and archived). Then it becomes a one-line scoping question for a human rather than a stale catalogue a reader would re-execute.
- **G-0217** — keep open, `aiwf edit-body`, and retitle. Rewrite "What's missing" around the real defect ("the label says WRAP PENDING for a milestone that is wrapped, and cannot tell merged-to-epic from merged-nowhere"), delete case 1 and the table row marked "(current behavior)", fix the trunk-ref parenthetical (the code uses the local `main` branch, not `allocate.trunk`), repoint "step 11" at step 12, and drop or correct the G-0216 characterisation. Note for whoever implements: the renderer already resolves the parent epic (`WorktreeView.ParentEpicID` / `.ParentEpicStatus`, `internal/cli/status/worktrees.go:235-240`), so fix (a)'s stated precondition is already half-built.
- **G-0233** — Leave open. `aiwf edit-body G-0233` to narrow the first sentence — 4 of the 13 sites are already scoped to a named `<section>` / header block / `<tr>` and their comments cite the same CLAUDE.md lesson, so the worklist is 9, not ~13 — and to drop or re-measure the "~35 fault-shaped `//coverage:ignore` sites" figure, whose leading exemplar (ENOSPC) appears in no Go source at all. Item 2 (the AST policy) is untouched and is the load-bearing half.
- **G-0234** — keep the gap open; `aiwf edit-body G-0234` to (a) drop or correct the sentence "the typed `FSMTransitionError` already lists the set internally, so this is a serialization shape change, not a data change" — the three offending sites do not construct that type and `internal/entity` exports no AC/TDD allowed-set accessor, so the fix *is* a data change; (b) drop `cmd/aiwf/` from item 3's stated sweep scope (that directory is now `main.go`, 21 lines, with no flag-validation error in it); (c) replace "rough scope ~30 sites" with a re-derivable figure or drop the number; (d) cross-reference G-0483, which now owns the same seam with a decision attached. Consider whether item 2 survives as an independent item at all once G-0483 lands.
- **G-0235** — `aiwf edit-body G-0235` to reduce the body to its two live items and correct five wrong sentences (details per finding below); in particular replace the four Greek placeholders with the real ids `G-0227` / `G-0232` / `G-0230` / `G-0233`, delete "each trailered to this gap" (measurably false), and delete or requalify "it lives only implicitly in the `aiwfx-record-decision` skill body" (it lives explicitly, under its own heading). Do **not** promote `addressed` — `cited_entity_ids_resolve.go` and `cache_invalidation_documented.go` do not exist and nothing equivalent covers them. Consider retitling: the current title advertises a sweep that is done.
- **G-0246** — `aiwf edit-body G-0246` to repoint the three `internal/entity/entity.go` line citations (they were exact when written and the schema table has since moved down ~100 lines), and optionally record the sharper measured fact below: an ADR carrying `relates_to:` is *silently accepted* by the strict parser and produces zero `ForwardRef`s, so a dangling target is never reported. Keep it open at `medium`; the G-0168 pairing is still live.
- **G-0249** — leave open at `low`. One `edit-body` pass is worth it: drop the `ParentEpicStatus` sentence (that symbol is a CLI-render field the check engine may not import) and replace it with "the rule can read the parent epic's status with `t.ByID(m.Parent)`, exactly as `epicActiveNoDraftedMilestones` already does"; drop "or refusal severity" (the check engine has only `error` and `warning`); and retire the closing G-0248 comparison, whose subject is now `addressed` and archived.
- **G-0253** — leave the gap open. Two `edit-body` corrections are worth making: drop the attribution "G-0067's feasibility note flagged this as the hard part" (no such note exists in any version of G-0067), and sharpen "a changed conditional whose block runs at least once reads as 'covered'" to name the three arm shapes that actually have no block of their own — implicit else, a co-listed `case` value, and a short-circuited sub-condition — since the plain defensive arm the sentence evokes *is* caught.
- **G-0254** — `aiwf edit-body G-0254` on two things: (1) delete or correct "CI's `aiwf check` always runs" — no workflow has ever invoked `aiwf check`, which makes the recommended "always-on guarantee" not one, and cross-reference **G-0536**, which owns exactly that hole; (2) date the 474/4355 measurement or re-derive it (today: 708/10259, 6.90%). Also worth surfacing to whoever writes the `D-NNNN`: `docs/design/provenance-model.md:58` already says "the LLM is a *tool*, **not a co-author**", which is closer to the git trailer than the body's framing admits.
- **G-0282** — leave open; `edit-body` to replace the whole "Status (2026-07-21)" section and the `rewidth` bullet under "Adjacent cleanups". Specifically: (a) `rewidth` was retired by ADR-0039 (`db307fdc7`, 2026-08-03) so the "concrete seed fixture already on hand" no longer exists — the base registry now needs a different seed or none; (b) `aiwf milestone tdd` **shipped** on 2026-07-24 (`3e1e350ff`, M-0277/AC-1 under E-0071), so the gated-annotation extension is no longer blocked — but before reviving it, reconcile with G-0168, which settled the opposite conclusion one day after this gap was last edited.
- **G-0302** — keep the gap and its Direction, but `aiwf edit-body G-0302` the "Related stale documentation" paragraph: drop the `statusline.sh:10` item (fixed by `661d9f390`, 2026-08-19) and add the `--fast` flag's own Cobra help string, which is still stale *and* is operator-facing. The `runFast` doc-block item stands unchanged.
- **G-0307** — `aiwf edit-body G-0307` to (a) drop "only areas rejects unknown keys" from the title and body — the `docs:` block acquired the same guard in M-0289 — and (b) rewrite the "Coordinate with E-0057" section from future to present tense: E-0057 is `done`, `config.AcceptedKeys()` is exported, and E-0057's own wrap says "G-0307 is unblocked". Also fix the constraint sentence: a bare `yaml.Decoder.KnownFields(true)` flip would *not* reject the legacy keys (they are declared fields); it is the registry-derived key set that excludes them. Then `aiwf retitle`.
- **G-0311** — keep open at high priority; `aiwf edit-body G-0311` to (a) correct *"three separate epics wired by `depends_on`"* — no such wiring exists, which makes the gap's case stronger, not weaker; (b) drop or requalify the ADR-0021 half of the "E-0043 / ADR-0021" attribution — E-0043's spec carries the claim exactly, ADR-0021 does not; (c) replace *"prose buried under `docs/` explorations"* with the current state of `docs/initiatives/` (12 `status: captured`, 2 archived `status: realized`, declared as an authority tier in `CLAUDE.md`), which is now the workaround this gap describes; (d) mark the Liminara/FlowTime and 40–50% figures as external and unverified, or cite an artifact; (e) note ADR-0045 as prior art any seventh-kind ADR must answer.
- **G-0328** — `aiwf edit-body G-0328` to rewrite §"The gap" and §"Acceptance sketch" around the real absence — *a git-bearing golden fixture, and a named regeneration target* — citing `internal/cli/integration/check_summary_binary_test.go` as the existing apparatus to extend rather than duplicate. Do **not** promote addressed: the history-rule half is genuinely uncovered.
- **G-0333** — `aiwf edit-body G-0333` and rewrite it around what survives. Specifically: delete the sentence "The boundary … is stated in no AI-discoverable channel (CLAUDE.md, `docs/design/provenance-model.md`, or `--force --help`)" and the AI-discoverability-violation sentence that rests on it (both measured false); refresh all six `file:line` citations plus `hint.go:88`; extend the Tier-1 enumeration from three preconditions to six; add the third class the taxonomy omits (unconditional verb-time structural guards); replace "Every mutating verb ends with…" with the allowlisted-minority truth; and put the E-0079 sentence in the past tense (E-0079 is `done`, M-0293/AC-2 `met`). What remains is a narrow, genuine residue: two normative documents (`CLAUDE.md`, `docs/design/provenance-model.md`) say `--force` is human-only and say nothing about what it does and does not relax. Consider also filing the separate live defect recorded at the end of this section — a shipped skill and a shipped finding-hint both still offer an override the kernel refuses.
- **G-0366** — keep open, `aiwf edit-body` for two things. Drop the parenthetical "(inconsistently — see the sibling gap on `wf-patch`'s missing CHANGELOG step)" — G-0365 was addressed the same day this gap was filed and `wf-patch` now has a mandatory CHANGELOG step. And rework Direction item 3: wiring `render roadmap --write` into `wf-patch`'s wrap contradicts that ritual's own shipped design in two places, so the item needs either a decision to overturn that or to be replaced by "the section regenerates from the tree on any roadmap render, no `wf-patch` change needed". Fixing the renderer alone does not touch `wf-patch` and is the cleaner path.
- **G-0369** — leave the gap open and essentially as written. One sentence is worth an `aiwf edit-body`: the "generated hook text" half of the G-0538 sentence no longer describes the tree. Everything else measured true, including every count, path, identifier and behavioural claim.
- **G-0370** — Keep open, keep `priority: high`. `aiwf edit-body G-0370` to replace the whole final paragraph ("Needs a hand-written pinning test…" through "`deployer_card_release_triggers_test.go`") — the gate it names was deleted, one of the two exemplars it names was deleted, and D-0070 now rules out the class of test it prescribes over this exact surface. In the same edit, correct "the role-agent cards" to name `deployer.md` alone, and note that the fragment sits at 146/146 against the line budget in `internal/skills/guidance_test.go`, so the fix is two edits, not one. The question M-0312 deliberately left open — whether D-0070's trigger-phrase exemption reaches the always-on fragment as opposed to a skill's own `## When to use` — is a human decision that should be settled before anyone starts.
- **G-0372** — keep `open`. `aiwf edit-body` two supporting numbers: the "~1,400 commits landed in the first half of July alone" figure does not re-derive under any reading, even at the body's own measurement date (1,829 all / 1,684 no-merges / 539 first-parent); and the present-tense "multiplies every git object access ~7×" now measures 2.6–2.7× post-maintenance. Optionally re-date the performance paragraph with the numbers below — the dated 2026-07-18 figures are honest and were correct then, and the growth since is itself the gap's argument. Two blanks the prior audit pass left (`entity-truth-audit.md:1845-1846`) are closed below: the blob-read count and the native-fs figure.
- **G-0375** — leave open at high priority. `edit-body` the two sentences carrying counts: replace "Nineteen test files" and "Only 4 pre-existing test files" with a derived phrasing (the shipped guidance rule "keep the reasoning; derive the facts" applies — the argument survives without arithmetic), and either re-measure or de-quantify "221 failures in `internal/verb` and 62 in `internal/gitops`". Also soften the sentence claiming the blanket fix "was tried and reverted": no commit in history ever carried it, so a reader looking for the revert finds nothing.
- **G-0385** — Leave the gap open and untouched in substance. Two optional low-severity edits: date the reproduction paragraph (it reads as a dated observation but carries no date), and note in the first "Possible directions" bullet that `version.LatestTimeout` is 3 s while the gap's own datum has `/@latest` taking ~8 s in the reproduced window — the proposed fallback would have timed out there — and that `TestLatest_Happy` currently `t.Errorf`s on any `/@latest` request when the list is non-empty, so implementing the direction is also a test edit.
- **G-0396** — `aiwf edit-body G-0396` to fix four measured numbers/claims: drop "a `git merge main` into the branch" from the list of SHA-changing reconciliations (it demonstrably preserves the SHA — only history rewriting breaks it); replace "roughly a quarter" with the measured 45-of-421 (10.7%, and 15.6% when written); replace "roughly fifty … would fire the resolver warning" with the measured 324 (204 when written); and correct "read only as 'is it non-empty?'" — `sameCommits` resolves the stored SHA through git in the promote same-state guard. Optionally widen the enumeration to the three gaps whose stored SHAs resolve to no object at all. Keep at `low`, keep deferred.
- **G-0398** — Keep the gap open and keep the Direction/Scope sections as written — they are still the right work. `aiwf edit-body G-0398` should cut the entire "Discovered indirectly" paragraph (the stresstest reordering that made it true landed `e2efb07c1`, 2026-07-14, and the source now explicitly says that add "can no longer trip" this refusal), correct "a generic 'did not report ok' shape" (that phrase belongs to the stresstest harness, not the CLI — the real output is fully specific), soften the "misleading hint" argument (the hint has named the creation case and G-0398 since `54cc1de76`, three hours after this body's last edit), and narrow "every mutating verb runs a before/after projection-findings gate" (`set-priority` and `set-area` deliberately do not).
- **G-0400** — keep it open and re-derive it. Retitle to the measured figures (16 of 39 exercised — `aiwf retitle`), rewrite "What's missing" so the current numbers live in the body rather than being corrected by an appended Notes section, drop `rewidth` (retired by ADR-0039, accepted 2026-08-03), move `move`/`archive`/`rename`/`retitle`/`list` out of the unexercised lists, and add `set-priority` and `milestone tdd`, which the enumeration never contained. Fix TODO.md's line, which repeats the pre-M-0250 scenario count that the gap's own Notes already correct.
- **G-0412** — keep the gap open, `aiwf edit-body G-0412` and replace the whole second paragraph. The origin pointer (`archive.go:127`), the 12-name file list, and the "M-0254 through M-0256 still to land more" forecast are all dead. Replace with the live site list re-derived by grep (8 files with the verbatim string: `contract/recipes.go:39`, `contract/verify.go:60`, `history/history.go:64`, `list/list.go:209`, `show/show.go:94`, `status/status.go:325`, `update/update.go:109`, `whoami/whoami.go:42`; plus `cliutil/prelude.go:23` and `:42`, which keep the inaccurate "missing aiwf.yaml + a non-existent --root path" clause inside an otherwise-accurate sentence). Note that `96a796550` already corrected four sites to a good wording that the sweep can copy. The first paragraph (the substance — the rationale is wrong on both counts, the ignore itself is legitimate) measured true and should stay.
- **G-0414** — leave alone; the body needs no edit. When someone next opens this scenario, rename the test and rewrite its doc comment and failure message to describe prevention-plus-clean-baseline (which is what it verifies), or restructure it to force a genuine guard bypass. Nothing here is stale enough to justify an `edit-body` on its own.
- **G-0417** — keep the gap open. `aiwf edit-body G-0417` to (a) fix the proposed-fix step 4 path — `internal/policies/branch_cell_bijection_test.go` has never existed on any ref; the live meta-test is `internal/policies/m0162_ac4_bijection_test.go`; (b) drop or re-scope the ADR-0011 attribution, which the ADR's own §Scope contradicts; (c) widen step 2 — repointing `branch-cell-2`'s `ExpectedErrorCode` alone still leaves a false cell, because the cell's *outcome* is wrong too (measured below); and (d) drop the now-stale "nothing currently owns it". The enumeration of stale sites needs no change.
- **G-0434** — leave open at `medium`. `edit-body` to (a) correct the two line citations (`:368` → the function is now at `internal/check/provenance.go:391`; `:363-367` → the paragraph the caveat means is now `:370-377`, and was `:347-354` when the gap was written, never 363-367); (b) restate the M-0126 example as what the history actually shows — two parallel allocations of `M-0126` merged and then resolved by `reallocate`, with the E-0034 entity *keeping* the id — since that is the strongest form of the gap's own caveat; (c) fix the claim about `Tree.ResolveByCurrentOrPriorID`'s doc comment; (d) note that the helper has five call sites, not one. The reproduction recipe below is worth pasting in as the regression fixture the Direction asks for.
- **G-0436** — `aiwf edit-body G-0436` to drop the second bullet (id-allocation.md — fixed by `03127efea`), fix the section attribution in the first bullet ("What's enforced and where" → "CLI conventions"), and either widen the enumeration to the two further stale `cmd/aiwf/` claims measured below (`CLAUDE.md:260`, `docs/design/tree-discipline.md:92`) or restate the gap as "CLAUDE.md cites a relocated path". Do **not** promote `addressed` — the CLAUDE.md citation is live at line 283.
- **G-0439** — `aiwf edit-body G-0439` to (a) correct "nine consecutive runs" to the measured 39, (b) replace "Nothing caught this until a later release cut's pre-release link-check ran" with what actually happened — link-check reported it on the next push run and stayed red for three, unattended, inside an already-red workflow — and (c) name ADR-0033 and state the first sketch option as *revisiting* its second bullet rather than as an unnoticed blind spot. Leave the gap open; it is M-0317's input and M-0317 is `draft` with both ACs open. M-0317's Context repeats the wrong count and should be corrected in the same pass.
- **G-0442** — `aiwf edit-body G-0442` and restate the gap around what measurement leaves standing — no `--clear`, no unforced amend, and (the part worth keeping most) a forced re-point that leaves a stale reciprocal nothing reports. In the same pass: replace "Two frontmatter fields" with the four the tree actually has, drop the borrowed "G-0168's four fields … no mutation path at all" premise (`aiwf milestone tdd` shipped), fix the three-failures attribution, and replace "the FSM back-edge" with the mechanism that really carries the constraint (the verb's own flag validation — the FSM has no back-edge and the check layer does not police the state). Do not promote it addressed.
- **G-0444** — `aiwf edit-body G-0444` to correct the second bullet: `readHistoryChain` was not retired and did not become inline — it was extracted and exported as `entityview.ReadHistoryChain` (`internal/entityview/historyevent.go:117`) by `5d331e61d` (M-0272), three days *before* this gap was filed, and `internal/cli/history/history.go:100` calls it. The right repair to line 164 is `Run` + `entityview.ReadHistoryChain`, not "describe the inline chain handling". Also drop or refresh the `history.go:60` line number (now 57) and widen the test-comment sweep to `integration_g37_test.go:928`. Then it is a clean two-token doc fix.
- **G-0445** — keep the gap open, `aiwf edit-body G-0445` to fix three sentences. (1) Delete or rewrite *"But `docs/` is not an aiwf-managed path"* — `docs/adr/` **is** aiwf-managed, so the option *"scope the exclusion to the entity tree only (`work/`)"* must be struck or narrowed to *"scope to `work/` + `docs/adr/`"*. (2) Delete *"never a false-refuse"* — a false-refuse is measurable today. (3) Either drop *"red/green"* (the gate is red-only per D-0049) or note the same stale naming survives in `internal/config/schema.go:44` and `docs/design/design-decisions.md:238`, which is a second small fix worth folding in. Optionally note that the exclusion is already documented generically in the shipped `wf-tdd-cycle` skill, which partly satisfies option 3.
- **G-0448** — `aiwf edit-body G-0448` and `aiwf retitle G-0448`. The title and both bullets under *What's missing* need replacing; *Where to fix* omits the file holding seven of the rules. Restate the gap as "no chokepoint asserts every defined rule is wired into some surface" — that claim survives every measurement below, whereas "no single source" does not survive at finding-code granularity. Any registry design has to answer where it can live: `internal/check` imports no config package today, deliberately.
- **G-0453** — leave open, but `aiwf edit-body` three sentences before anyone acts on it. (1) Replace "one family of finding messages" with the measured blast radius: `shortHash` feeds **five** finding codes plus a hint, `short` feeds **fourteen**. (2) Drop or re-measure the parenthetical "(git's default short-SHA width)" — in this repo `git rev-parse --short` emits 9. (3) Say that the width split is repo-wide (five bespoke truncators, three of them 7-char, one parameterized at 8 and one delegating at 12), so a fix scoped to `internal/check` leaves an exported `entityview.ShortHash` doing the same job at the other width. Option (c) ("leave them if the widths are intentional") is now cheaply answerable: the code asserts *both* widths are the right one, in comments, and no record decides it.
- **G-0454** — leave open, but `aiwf edit-body` two things. (1) The headline says all three "`strconv.Atoi` the numeric tail"; measured, only two do — `IDGrepAlternation` uses `strings.TrimLeft(num, "0")`. Restate as "strip the kind prefix and interpret the numeric tail". (2) The "Why it matters" tax is misattributed: both grammar tables (`idPrefix`, `idPatterns`) are already single-source and all three sites read them, so a width- or prefix-level grammar change touches **one map**, not three call sites. Only a *structural* change to the id shape costs three edits. Add the measured asymmetry the body misses: `parseIDNumber` never consults `idPatterns`, so its acceptance space is strictly wider, and the prescribed "thin specialization" of a discovers-**and-validates** primitive is not behaviour-preserving.
- **G-0455** — leave open, but `aiwf edit-body` the "Why it matters" risk sentence: the specific hazard it names — differences in "what counts as a heading, how fenced code or nested headings are handled" — is measurably **absent**. All four walkers detect headings with byte-identical prefix tests and **none** of them is fence-aware, so there is nothing to flatten on that axis. The real differences are in return shape (map / ordered slice / line bounds) and heading level (`## ` vs `### `). That materially cheapens the "evaluate first" pass the gap gates on, and a reader acting on the sentence as written would go looking for a subtlety that is not there. Optionally note that the walk duplicates well outside `body.go` (a further nine non-test sites), so the gap's "Where to fix" is the narrow slice of a wider pattern.
- **G-0456** — leave the gap open; `aiwf edit-body` one clause. "The two envelope-arm verbs (archive, rewidth)" names a verb ADR-0039 retired — `aiwf rewidth` no longer exists. The membership today is **archive and import**, both measured emitting envelopes on a prelude failure, so the "two" survives with one name swapped. While editing, consider whether the enumerated 20-verb list should name `add ac` (the 21st call site) and whether "Surfaced by the design-lens review during the M-0279 wrap" earns its place in the body given `discovered_in: M-0279` already carries it.
- **G-0458** — `aiwf edit-body G-0458` to fix one sentence in *What's missing* — "while every other mutating verb now converges to a `Result.NoOp` at exit 0" is false and contradicts G-0459, filed the same day by the same milestone. Restate as something like "while every other verb with a same-state input either converges or silently appends — none refuses." Everything else in the body stands.
- **G-0459** — leave the body alone; every enumerated verb and every supporting claim measured true.
- **G-0460** — leave the body alone. Every claim, reference and quoted sentence measured true. It is the one gap in this batch whose resolution needs a design decision before code, exactly as it says.
- **G-0461** — leave the body alone. On the orchestrator's question — **the missing allowlist entry is correct, not an oversight** (see below).
- **G-0464** — leave the body alone; it is accurate end to end. Fixable as stated (three `== entity.StatusCancelled` → `entity.IsTerminalACStatus`). One optional edit: the `## Scope` paragraph narrates a superseded code state ("that route *used to* launder…"), which this repo's own rule discourages — see the low finding below. Worth telling the implementer that the follow-on sweep the body asks for is now done (result recorded below), so it need not be repeated.
- **G-0471** — Leave the body alone; no sentence in it needs editing. Keep at `high`. Two things a human should know that the body cannot: E-0076 — "Chokepoints for three documented rules that have no detector", which named G-0471 in its scope and whose success criteria included "G-0465, G-0471 and G-0474 are promoted to `addressed`" — is `cancelled`, so nothing is in flight; and G-0504's body currently asserts G-0471 was "addressed by E-0076", which the cancellation falsifies (that is a defect in G-0504, not in G-0471, and is not mine to fix).
- **G-0472** — Keep the gap for its verdicts; `aiwf edit-body` to (a) rewrite the *Why it matters* paragraph that says three installers destroy an unreadable hook — that is fixed; (b) drop "until G-0557 lands" from Options and the "a live data-loss path" Related line, keeping the sentence that already says landing G-0557 does not make the collapse advisable; (c) correct the `appendContracts`/`appendHooks` claim, which is measurably wrong in both its verb and its reason; (d) refresh five line citations shifted by the G-0557 fix. The four-family verdict, the shared-unit argument, and the legacy-key fix direction all re-derived true and should be left alone. Worth noting for the human: `internal/initrepo/hook_read_fault_test.go:28` already cites this gap's verdict in a shipped comment, which is exactly the "an open gap's lean is not something a permanent exemption can rest on" problem the body names.
- **G-0473** — Keep the gap; `aiwf edit-body` four sentences. Drop or rewrite "Nothing tracks them." / "not in any gap" (G-0472 now tracks six of the eight, and this body's own *Why it matters* says so). Fix the Related line for **G-0470**, which is `addressed` and archived, not "the live sibling concern". Recast the present-tense "the one clone … carries its own acknowledgement as a per-site `//nolint` rationale" as the option-4 end state it describes — there is no `//nolint:dupl` in the tree. Soften "the only inventory of acknowledged duplication" (D-0045 records one outside the list). Everything load-bearing — the count of eight, the two stale entries, the six live ones, the file-scope blindness, and option 4's threshold-250 measurement — re-derived true and should be left alone.
- **G-0477** — Leave the substance alone; take Option 1 when someone is next in these two files. Two small `edit-body` corrections are worth making first. (1) Option 3 ("Leave it and rely on the sibling gap's correction") has already happened — G-0472 was corrected on 2026-08-05, six days after G-0477 was filed, and now states the equivalence outright; as written, Option 3 reads as an available future choice when it is now just "do nothing". (2) The phrase `"mere prefix test"` is presented in quotation marks as the sibling analysis's wording; no version of G-0472, and nothing else in this repo's history, ever contained it. Worth noting for whoever takes Option 1: the twin `isTopLevelAiwfVersionLine` (`config.go:1064-1071`) carries the identical dead guard, without the misleading comment.
- **G-0478** — keep open, but `aiwf edit-body` the *Why it matters* section: the `59`, the `four`, the E-0073/M-0281 sentence and the `six consecutive runs` all re-derive to different numbers, and the E-0073/M-0281 pair was already repaired eight days before the revision that kept the sentence. Replace with the current, dated instance (G-0317 ×2 + G-0584, broken by `d82259f5a`, red for 10 runs). Fix "Every mover rewrites links" — `move` does not, on `main`. Most important: the *Detection* resolution shape proposes a pre-push check rule that accepted **ADR-0033** bullet 3 explicitly declines and that **G-0392** was already dispositioned into; the body cites neither, so a reader would build a thing an accepted ADR forbids. Cite ADR-0033 and `wf-doc-lint` check 5 and restate the ask as "reverse ADR-0033 bullet 3" or "make the advisory check fire earlier".
- **G-0483** — leave the gap open, but `aiwf edit-body` three sentences. (1) The "three sibling `emitErrorEnvelope(label, "", …)` sites … do exit 3" sentence is stale — `ab95ea1a9` (M-0291/AC-1, 2026-08-04, one day after this body's last edit) rewrote the apply-failure site to pass `codeStr` and to exit 1/2/3 by class; two literal-`""` sites remain. (2) The "A git subprocess that died, a tree that could not be written, a config that failed to parse mid-verb — each exits 2 today" sentence is two-thirds false by measurement: only the git-subprocess case reproduces at exit 2, and only on the pre-`Apply` claim-guard path; the other two measure exit 3. (3) "Exit 3 means 'aiwf broke' everywhere else" is contradicted by three measured cases, one of which is the sibling gap G-0561 in the same TODO cluster. Also worth recording that D-0044 (accepted) already ratified the three-class contract with `ExitUsage` as the residual, and that G-0483's preferred fix pulls in the *opposite* direction from G-0561's.
- **G-0486** — `aiwf edit-body G-0486` to rewrite the opening "What's missing" so it states only what is still true (the forced `100644`), and to fold the already-correct "Measured against E-0075's guard" section's verdict into it rather than leaving a contradiction 60 lines apart. In the same pass, widen the scope claim — the forced mode is not specific to directory moves — and correct "Nothing shipped by `aiwf init` is reachable this way". Retitle it (`aiwf retitle`) so the title no longer says "directory moves … dereference symlinks". Leave it open.
- **G-0493** — leave the substantive body alone; every claim measured true. **Do not batch this as a cheap fix** — the body's own `## Resolution options` section offers three mutually-exclusive routes with a stated trade, so it needs a decision before code. Before that decision is taken, add the one fact the body is missing (medium finding below): the *same* question is already answered field-based at the `verb.Apply` guard, for **both** modes, so bless mode's prelude is the only byte-based site of three — which strengthens the body's own option 1 and weakens option 2.
- **G-0497** — Keep the gap; `aiwf edit-body` to fix three things. (1) Re-derive the headline: **90 sites across 13 packages**, with `internal/policies` now at **5**, not 8 — the other twelve rows are unchanged. (2) Drop the sentence "The per-site `//nolint:gosec` disappears with it … so the sweep removes noise as well as risk": zero of the 90 sites carry one, and `.golangci.yml` excludes `gosec` from `_test.go` outright, so the sweep removes no suppressions at all. (3) Qualify the exposure claim — the 35-site `internal/initrepo` block, which the body singles out as "a plausible unit on its own", runs its hooks via `sh <path>` and so carries no ETXTBSY exposure on the file it writes; the genuinely exposed sites are the ones exec'd through `PATH` (the `go` stub, the `gh` stub, the `go` shim in `upgrade_cmd_test.go`, the fake validator in `contract_cmd_test.go`). That reordering also changes the sweep-ordering advice: `internal/policies` and `internal/cli/integration` earn priority on evidence; `internal/initrepo` is uniformity work, not risk work.
- **G-0498** — Leave the gap open and leave the argument alone. Two sentences want narrowing via `aiwf edit-body`: "On such a repo every mutating verb is unusable" (`aiwf add` still works — it is the verbs naming an *existing* entity that refuse), and the present-tense framing of the first consequence ("A verb silently rewrites content it was not asked to change"), which the ADR-0038 guard now precludes for guard-covered paths — the measured live symptom is the *inverse*: the verb's own commit leaves `git status` reporting the file it just wrote as modified. Everything else measured true.
- **G-0500** — Leave the gap open at `high`. One optional one-line `aiwf edit-body G-0500`: the References line for `internal/verb/editbody.go` credits it with "why neither takes the claim guard", which that file does not contain — the rationale lives in `internal/verb/claimguard.go` (the very next reference) and in `internal/policies/noop_claim_scope.go:106`. Retarget or trim the description.
- **G-0501** — leave the gap open and untouched except for one sentence. `aiwf edit-body G-0501` to correct "The existence probe uses `os.Stat`, which follows too, so the step sees a file and reports the import wired" — the writing step's existence test is the `fs.ErrNotExist` branch of its own `os.ReadFile`, and the `os.Stat` probe belongs to a different, init-only step. The rest of the body measured true verbatim and the decision it poses (refuse vs. write through) is still open.
- **G-0502** — leave the substantive body alone. Three cosmetic edits are worth one `aiwf edit-body` pass if the human is touching it anyway: G-0499 is now `addressed` and archived (the reference reads as a live sibling), the blob-only filter lives in `headEntries` rather than in `DivergentPaths` itself, and `C-0001` is an id-shaped token that resolves to nothing in this tree. Like G-0493 it is not quite design-free — its `## Scope` poses a genuine two-way choice — but the choice is small and one-line-answerable, so the cluster placement is defensible here in a way it is not for G-0493.
- **G-0504** — `aiwf edit-body G-0504`. One high-severity edit: delete "addressed by E-0076" — E-0076 is `cancelled` and G-0471 is still `open`, so the sentence tells a reader the neighbouring axis is handled when it is not. Then refresh `27` → `29`, widen "three artifact families" to four (hooks materialize to `.claude/hooks/` and are also content-blind), and either re-date or replace the `epic-spec.md` observation, which no longer reproduces in the form given (the file is still drifted, differently). Everything load-bearing about `doctor` measured true and needs no change.
- **G-0506** — leave the gap open and unchanged except for one sentence. `aiwf edit-body G-0506` the sentence beginning *"This is the window ADR-0038 opened the claim-side seam to close, and the ADR names the same shape in `promote`'s resolver re-point refusal"* — ADR-0038 does not contain that; `internal/verb/claimguard.go`'s `guardClaim` doc comment does. Repoint the citation there (it cites ADR-0038 itself, so nothing is lost). Everything else measured true.
- **G-0508** — Keep the gap and `aiwf edit-body` it upward, not downward. Replace "Four policies" and the four-item list with the measured eight (or seven, if the enumeration is scoped to files that use the prefix as a *filter*); replace "Three filter `fn.Recv != nil`; `noop_claim_scope.go` does not" with the measured four-vs-four split, since `verbs_validate_then_write.go` is a third un-filtered copy carrying no comment and no pin; add `verb_write_guard_coverage.go:143-144` alongside `noop_claim_scope.go:206` as the second redundant `_test.go` filter; and `aiwf retitle` — the title has said "Three" since the day the body said "Four", and TODO.md copied the title. The Scope proposal's field list also needs widening (see the last finding).
- **G-0510** — leave the gap alone as a defect record; it is accurate. Before doing the work, note two measured corrections a human may want folded in by `aiwf edit-body`: (a) the census turns up **zero** exceptions, so the "one call to take deliberately" about a transition period is moot and the sweep is behaviour-preserving; (b) `collectIgnoredLines` is shared by **two** policies (`PolicyEnumLiteralAdoption` and `PolicyFindingCodeAdoption`), and the "Where to fix" list omits the second — routing it through `hasDirectiveComment` changes `finding-code-adoption`'s allowlist too.
- **G-0512** — leave alone. Optionally tighten one sentence in *Why it matters* (see the low finding) at the next edit; nothing in the body misleads a reader who acts on it. Worth pairing with a note that `aiwf archive --help` already *promises* the coverage this gap says is missing — see the additional observation below.
- **G-0513** — leave alone. Every sentence measured true, including the quoted output string, the "that route is covered" contrast, the `aiwf check` load error, and the "nothing is written either way". The *Sketch* is implementable as written.
- **G-0514** — `aiwf edit-body G-0514`. Replace the "Measured on the shipped tree, these currently fire" framing with a dated observation naming M-0288 as the sweep that cleared the population, and re-anchor the argument on the one instance that survives in the shipped tree today (`internal/skills/embedded/aiwf-add/SKILL.md:149`, `aiwf milestone depends-on M-NNNN --on M-NNNN,M-NNNN`). Drop `classifySkillToken` from "Where to fix" (the symbol no longer exists), widen "Shared with `body-prose-id`" to name `doc-id-width` too, and cite D-0052 — which already pre-authorises the exemption route the gap lists as option 2 — and D-0051, whose reasoning option 1 works against. Do **not** promote addressed: the defect itself is unfixed.
- **G-0516** — leave the gap open. Two cheap `edit-body` fixes are worth making: replace the phrase example "`previously`" (which is *not* in the phrase set) with one that is, and add a clause noting the cited instance was repaired in `3e9a45f1f` so a reader following the citation is not confused by the now-correct comment. Separately, correct the `TODO.md` line's reframing (below) — it collapses three proposed resolution shapes into one and mis-ranks the gap.
- **G-0517** — keep open, `aiwf edit-body G-0517` to replace the `## Problem` second paragraph and the `## Resolution` wholesale. The corpus is the same genre as the corpus M-0289 already swept (worked-example fiction), so the fix is the canonical `<prefix>-NNNN` placeholder, not widening. **Blocker to know about first:** `internal/policies/m0289_residue_gap_test.go` requires the literals `Widen` and `canonical id` inside G-0517's `## Resolution` section, so the honest edit turns that test red. That test pins M-0289/AC-3 of a `done` milestone; changing it needs its own decision. Also fix the Resolution's config step: `docs/design/**` cannot be written into `docs.paths` (no glob support), and "leave genuinely-narrow archived references alone" cannot co-exist with `docs.strict: true`.
- **G-0518** — leave the body alone. The only edit worth making is optional: "a number of citations" is two, and the body deliberately declines to say so, which is consistent with the repo's own derive-don't-copy rule. If the gap is picked up, both instances are `work/gaps/G-0113-…md:13` (`E-19`) and `work/decisions/D-0016-…md:29` (`M-070`).
- **G-0519** — leave the body alone. Optionally one clause in `## Why this is larger than it looks` correcting the archived-narrow bullet (see the one finding below), since as written it asks a future rule to be silent where its already-shipped sibling blocks the push at error severity.
- **G-0523** — `aiwf edit-body G-0523`. Rewrite the "Fix shape" sentence "The events are already wired, so this needs no consent surface beyond what ADR-0015 already governs" — it is wrong on both halves and it understates the fix's cost (ADR-0032 makes a new hook default-declined, and silently refused in a headless `aiwf init`). Add the ADR-0018 Consequences sentence the gap's observation actually falsifies, instead of saying the gap "does not dispute that model". Drop or attribute the "three review passes" clause, which G-0520 does not carry. Leave everything else — it all measured true.
- **G-0524** — keep `open`, but `aiwf edit-body` three things. (1) Add the blocking chokepoint to *Fix shape*: `PolicyM0132DevcontainerShape` (`internal/policies/m0132_devcontainer_shape.go:72-77`) requires `workspaceMount` to contain `${localWorkspaceFolder}/..`, and the gap's own proposed replacement string fails it — measured. (2) Rewrite the "A session then has two candidate project-instruction files" sentence in the conditional: the exposure is a capability, and no parent-level `CLAUDE.md` is present now. (3) Fix the section attribution (§"Build", not §"Reopen in Container") and re-derive "30-plus". Also worth adding, because it strengthens the lean the gap records: the policy's stated justification for the widening ("cross-repo plugin testing per CLAUDE.md") no longer exists in CLAUDE.md.
- **G-0526** — Keep open — it is a genuine undecided design question and should not be closed by reflex. `aiwf edit-body G-0526` to (a) correct "There is no seam that would close it" and the "hooks run `aiwf check`" enumeration, which omit that aiwf's own materialized git hooks chain to a `<hook>.local` sibling and abort on its non-zero exit — the seam this repo's `comment-history-attrition` gate actually rides; (b) re-cost option 2, since the mandate/ban distinction and the owner-plus-retirement rule already ship in both the always-on fragment (H3) and `wf-codebase-health` (H3), leaving only the uniqueness/exactness pair and the classification method; (c) fix the D-0053 sentence, whose subject is now `proposed`-status, whose original retirement trigger its own body calls "void", and whose mandate has since been retired.
- **G-0527** — `aiwf edit-body G-0527` is needed before anyone acts on this. Cut the whole "The absence is silent rather than loud" block and the closing "the silent-success behavior should stop" sentence (both describe behaviour G-0528 already fixed), correct "three acceptable answers" to four, and replace "unmentioned by any surface" with the true, much narrower claim: unmentioned at the *worktree verb's own* surface (`aiwf worktree add --help` and the `aiwf-worktree` skill), while `wf-patch`, `aiwfx-wrap-milestone` and `aiwf status` already document the pair. What survives after that is a one-sentence documentation ask, not a design question — worth retitling too, since the current title asserts the dead claim.
- **G-0529** — Keep open, and `aiwf edit-body G-0529` to rewrite the "The failure is not hypothetical" paragraph and the second Direction bullet. Measured, G-0509 was filed **before** E-0075 wrapped, against a CHANGELOG that carried **nothing** for the epic; the wrap then wrote a 23-line entry that covers the refusal in detail. That is an absence caught by a human, not a thin entry that slipped past — so it is the gap's *first* (cheap, epic-citation) property that addresses it, not the second. Also fix "step 7" → step 6, and either drop "No other surface looks" or name `pr-conventions.yml`'s `changelog` job as the third presence-shaped check. Overlap noted for the human, not audited here: cluster-4's **G-0613** ("the wrap changelog's category set omits `Removed`") shares the wrap-CHANGELOG subject but is about the category vocabulary, not verification — not a duplicate.
- **G-0530** — `aiwf edit-body G-0530`. (1) Delete the final paragraph about `docs/design/growth.md` — the correction it requests, *including* the compose-into-a-mandate framing it proposes, landed in `2574341f7` on 2026-08-04 and growth.md now cites G-0530 as the tracker. (2) Restate the `## Work log` row: its 0-word median is an artefact of a subsection-excluding count, and by the template's own prescription its content lives in `### AC-N` subsections, where its median is 226.5 words — third-largest section in the spec. Drop the "empty in half" sentence; nothing re-derives it. (3) Fix "Cut the four from…the wrap ritual's step 4" — only `## Work log` is in step 4, and it also appears in wrap steps 2 and 12 and in `aiwfx-start-milestone`; `## Dependencies` and `## Surfaces touched` appear in no ritual at all. (4) Delete "Gap is the largest population and has no template at all" — a gap template shipped on 2026-08-22. (5) Refresh "47 files" to 51. ADR-0043/E-0084 do **not** block this gap: E-0084 names it out of scope explicitly, and none of the four sections is in `entity.RequiredSections`, so nothing enforced changes.
- **G-0533** — leave the body alone; the dated block is a dated observation and re-derives within ~1%. Fix the **TODO.md line**, which recommends the shape the body argues against taking first and prices it backwards. Optionally add today's re-derivation to the dated section, since the diff-scoped yield figure has moved to zero in a fresh window.
- **G-0535** — `aiwf edit-body G-0535` to repair three things: the Options-3 precedent (the sovereign policy was **deleted**, not given an anti-orphan test — cite `TestPolicyApplyCallersAcquireLock_ScopeIsNotOrphaned` instead, which is the only such test left); the G-0534 contrast (it is `addressed`, its redundancy question settled by D-0061, so "the redundancy question there is live" is dead); and the entry-point list under the trailer policy (the integration tests' real entry point is `cli.Execute(` — 627 sites across 88 files; `runVerb` is a local closure in one file and `run(` is not a call site at all). Add that R-AUDIT-0058 already exists for the trailer policy and states the dead `runBin` trigger, so that row needs correcting rather than creating.
- **G-0536** — `aiwf edit-body G-0536`. (1) Fix the hook enumeration — there are three positions, not two; `commit-msg` runs `aiwf check --commit-msg`. (2) Delete or rewrite "On the tree as it stands the step reports errors on day one. … G-0556 holds it" — measured false, and G-0556 is `addressed`/archived; this is the sentence a reader would act on. (3) Narrow the no-other-detector list: `body-prose-id`'s `unresolved` / `unresolved-milestone` subcodes now *do* have a CI detector (`TestPolicy_ThisRepoTreeHasNoDanglingReference`, landed after this body was written). (4) Point the "pattern to copy" at go.yml's `test` job, which already has both `fetch-depth: 0` and `go install ./cmd/aiwf`. The `docs/design/oracles.md` §"Position is per clone for the local rungs" paragraph carries the same two-positions error and the same absolute "no workflow runs it"; it should be edited in the same pass.
- **G-0537** — leave alone. Optionally correct the one sentence about the statusline (finding 1) with `aiwf edit-body G-0537`; the argument it supports is unaffected.
- **G-0538** — leave alone. Optionally one small `edit-body`: §"What's missing" enumerates "Three shapes" of printed output but §"What is already fixed" names a fourth (`aiwf doctor` output) as unfixed — make the enumeration four, or fold doctor into the `cliutil`-printed shape.
- **G-0539** — keep the gap; `aiwf edit-body G-0539` to rewrite the **Resolution shape** paragraph — its central premise ("a verb's own commit also stages entity files", so the hook needs an env-var marker to skip verbs) is false: aiwf verbs commit through git plumbing and fire no hooks at all, so no marker and no verb exemptions are needed, and the `aiwf archive` / `aiwf reallocate` sentence becomes moot. Also fix the title/lead "hides until push" — measured, it hides *through* the push and stays hidden afterwards.
- **G-0540** — Leave open and `aiwf edit-body G-0540`. Three edits: (a) thread 4 must repoint — `internal/policies/sovereign_test.go` was deleted on 2026-08-06 with the whole sovereign-dispatcher policy, so the surviving near-duplicate of `_UnreadableRootErrors` is `apply_callers_lock_test.go` + `coherence_guard_chokepoint_test.go`, and it is *not* identical "body and doc comment alike"; (b) the "two adjacent files" premise under Option 1 is now three files; (c) Option 3 ("Fold into E-0079") is dead — `E-0079` is `done` and archived. Threads 1-3 need no change.
- **G-0543** — leave the gap. One optional `aiwf edit-body`: the opening sentence says the harness *"proves each guarded golangci-lint config rule fires"*, but it has three rows against five project-specific forbidigo patterns — the `fmt.Println` / `fmt.Print` / `fmt.Printf` trio has no row. (They are separately backstopped by the `logging-chokepoint` AST policy, so this is a wording fix, not a new hole.) That phrasing is copied from the harness's own doc comment, so fixing it in one place without the other leaves a fork.
- **G-0544** — leave the body alone. Nothing in it needs editing. If the human wants one addition, it is that `aiwf --help`'s *Common flags* block advertises `--principal` as though it were universal ("required when `--actor` is non-human"), which is false for exactly these four verbs — a second, shipped surface making the same claim the gap says the verbs break.
- **G-0545** — leave the gap open; `aiwf edit-body G-0545` to fix two supports — "several ACs already cite" (measured: one) and the attribution "recorded when it was written" (the record carries a different hole). Add the fold's actual shape: the two scopes are not nested, so Option 1 is a union with a carve-out, not a widening.
- **G-0546** — `aiwf edit-body G-0546` on two sentences. In *Why it matters*, drop or restate "and it is why a refusal there speaks in trailer keys rather than in flags the operator typed" — the live refusal reads `aiwf-force requires a human/ actor (got actor="ai/claude"); only humans wield --force`. In *Options*, restate option 2's benefit: the refusal already names `--force` and the actor, so what option 2 buys is only *when* it is raised, not *how* it reads. In *Scope*, drop "rather than as a justification inside the ADR" — ADR-0040's Context carries exactly that justification and predates this gap.
- **G-0548** — keep open. `aiwf edit-body G-0548` to (a) say four clauses, not three, and name `inline lifecycle status` as a second unchecked one; (b) narrow "the path clause … [has] nothing" to "the general path shape has nothing", noting `skill-body-claude-md-section` (landed 2026-08-21, error severity, same corpus) as the sibling to model on; (c) drop "with the same masking of non-prose carriers" — the sibling that landed deliberately applies no mask, and its reasoning applies here; (d) correct or drop the 141-line/zero-slash observation and the "guidance fragment names its own materialized location" clause, both of which fail to re-derive. Add a `Related` line to G-0599 and G-0587.
- **G-0549** — leave alone. Optionally soften one phrase in "Why it matters" — see the single finding below. Fix it together with G-0553's `environment` half, as both bodies already say.
- **G-0550** — leave the defect alone; it reproduces exactly as described. Two body edits worth making: drop or relabel the `## Scope` paragraph (provenance narration, and the gap template says a gap carries two sections and "no options, no direction, no resolution shape" — this body carries an `## Options` section with a lean); and soften the "this epic's constraints forbid…" attribution, which E-0079's Constraints do not literally carry and which cites a `done` epic.
- **G-0551** — leave open. `aiwf edit-body G-0551` to narrow one sentence: "The audit returns no findings at all for a commit carrying no `aiwf-actor`" is false of the provenance audit and true only of its coherence group (finding 1). Worth adding while editing: the generated domain Option 2 needs already exists at `internal/verb/coherence_domain_test.go`, so Option 2 is cheaper than the body implies (finding 3).
- **G-0552** — leave open at Option 3. Optionally amend the one sentence that attributes the fill to the cache alone — measured today the cache is 15% of used space while the sibling leak in G-0555 is 32 GB — so a reader implementing only the cache bound knows it will not, by itself, keep the disk clear.
- **G-0553** — keep the gap; `aiwf edit-body G-0553` to (a) delete the sentence *"Two Fix cells prescribe a remedy for a case the rule cannot report, and one of those (`case-paths`) tells the operator to `git mv`, which the shipped guidance elsewhere forbids and which trips another finding."* — the Fix column was deleted by `ee29bc04c` hours after the gap was written; (b) restate or date the "102 rows" figure (107 now); (c) narrow *"the defects sit in rows nobody has touched since they were written"* to the six false cells, which is where it holds.
- **G-0554** — leave open, prefer the body's second direction. Two sentences are worth an `edit-body`: "materialized into the repo of every consumer" (the default scope is `~/.claude`, not the repo) and "It costs a `find` on every statusline render" (the write path is the cache-*miss* branch only). Neither changes the verdict.
- **G-0555** — keep the gap, but `aiwf edit-body` three things. (1) The paragraph beginning "**Periodic cleanup cannot substitute for the fix.**" — its generalisation is false today and it is actively telling a reader not to reclaim 24.4 GB on a filesystem with 8.6 GB free; restate it as the dated observation it is and drop the "only place the problem can actually be solved" conclusion. (2) "used across many test packages" — it is two. (3) The `--keep` sentence — no such flag exists, and the exemplar it holds up has 1243 leftovers on this machine. The third helper (`stresstest-shared-lockholder-`) should join the title's count or be named in the body.
- **G-0560** — keep the gap; `aiwf edit-body G-0560` to (a) delete the G-0559 gating sentence and reduce the sequence to "G-0517, then this gap's tables" — G-0559 is `addressed` and `aiwf schema` now prints canonical widths; (b) delete the ADR-0003 bullet — ADR-0003 is `rejected` and archived; (c) move the `rename`-updates-title citation from `architecture.md` to `design-decisions.md:100` and drop the "ADR-0037 split that" attribution; (d) either drop the age-clustering paragraph or restate it, since `workflows.md` and `provenance-model.md` falsify it.
- **G-0561** — leave the gap open and leave the argument alone; it is the best-evidenced of the three. One cheap `aiwf edit-body`: retarget `internal/cli/update/update.go:121` to `:130` (line 121 is the `config.Load` arm; the arm the measured error actually travels is `RefreshArtifacts`'s at 130-133), and consider softening "The whole classification is one arm" — that arm shape occurs at six sites across the two files, not two. Sequence this gap's exit-code question *with* G-0483's; they propose opposite moves on the same 2/3 boundary.
- **G-0562** — `aiwf promote G-0562 addressed --by-commit 793b1ad97b89e7830b1048c7ca311cd014953d99`, and the same for G-0578, which names the same single call site. Before closing, decide whether the surviving second question is worth its own entity or is already covered by G-0497 (the whole-tree sweep) — measured, `internal/policies` still holds **5** bare executable writes in test files, at `coverage_gate_wiring_test.go:127`, `m0155_statusline_scaffold_test.go:72`, `prepush_lint_hook_test.go:396`, `prepush_lint_hook_test.go:474`, `statusline_behavioral_test.go:121`.
- **G-0563** — keep the gap, `aiwf edit-body G-0563` on four sentences: (1) *"`aiwf add` is the exception"* — eight verbs load through `cliutil.LoadTreeWithTrunk`, and `add`'s own `--depends-on` path refuses anyway; (2) *"No supersession has ever been recorded in this repo"* — three decisions carry `status: superseded`; the true statement is that no `superseded_by` reference has ever been written; (3) *"`depends_on` edges run between milestones of one epic"* — 2 of 168 cross epics; (4) the Resolution shape should say that `milestone depends-on` and `add --depends-on` refuse at an argument-resolution guard (`t.ByID`) *ahead of* the projection, so switching the loader does not reach them. Worth adding: the measured transcript was taken in a repo with no remote, where ADR-0041's local-only escalation is off; the divergence needs the sibling branch **published**, which is measured below.
- **G-0564** — leave open at medium; `edit-body` the three sentences that treat E-0080 as live — the opening "are mechanized by E-0080", "which is why E-0080 excludes both rather than carrying them", and the two `Related` bullets for E-0080 and G-0121 — restating them in the past tense as scope E-0080 recorded before it was cancelled. Also narrow the E-0033 attribution (below) and drop the "nothing cross-links them" clause, which is measurably false.
- **G-0565** — leave the body alone. Two optional low-severity precision edits: (a) "Two rules do" undercounts — a third rule (`provenance-untrailered-scope-undefined`) also changes verdict with ref topology, though only at warning severity, so the argument survives; (b) "there is no `cross-branch-pending` finding in this repository" reads ambiguously — it is true of the *fixture* and false of the aiwf repo, where that subcode exists and fires.
- **G-0568** — leave the body alone — it is the most accurate of the four in this batch. The one thing worth doing is fixing the **TODO.md line**, which still carries the pre-revision framing (it promises severity, which the body declines). If anything is edited in the body, it would only be to note that the same field is missing from `htmlrender.StatusFinding` too, which the ten-line estimate does not cover.
- **G-0569** — leave it alone. One optional edit: the body says the cycle route "reaches an accepted ADR stating the FSM is one-directional" without naming it; the accepted-ADR sentences that exist scope that statement to the *per-kind* FSM, and the all-kinds version lives in a design-doc rule. Naming both (ADR-0036 §Context for the ratified half, `legal-workflows-audit.md` R-RULE-019 for the all-kinds half) would make the cost argument checkable.
- **G-0571** — Leave open; it is named as the closing condition of `E-0084` (`- Closing G-0571.` / `- [ ] G-0571 is \`addressed\`.`). `aiwf edit-body G-0571` to (a) re-measure the four counts in ¶"The consequence is already in the tree" and ¶"Closing it tree-wide" (35→31, 60→56, 119→111, 118→110), (b) replace "an active epic in this tree" with the fact as it now stands (the one epic missing `## Out of scope` is `E-0083`, status `proposed`), and (c) replace the closing "The narrower option is …" sentence, which `ADR-0043` (accepted) has since overtaken with a different and broader design. Note that `ADR-0043` and `E-0084` both restate the same stale 119/60/118 figures, so a re-measure that stops at the gap leaves two copies drifted.
- **G-0572** — `aiwf edit-body G-0572` to qualify two sentences (the opening "no unforced verb can repair" and "What the kernel does not offer is any unforced route out"), replacing them with the form G-0121 already uses for this very gap — *"whose only criterion-preserving repair is sovereign"* — and add the two missing rows to the exits table (`promote <m>/AC-1 deferred`, `milestone tdd <m> --policy none`), since the table is what carries the claim. Then fix the matching TODO.md line. Leave the gap open and at `high`.
- **G-0573** — promote `addressed --by-commit 29eb2a94c` — that commit's own message names this gap: "Addresses the structural half of G-0573 — the seven call sites now share one seam — and leaves it open for ADR-0042 to settle whether its remaining behavioural half exists at all." ADR-0042 has since been accepted and settles it: "G-0573's behavioural half becomes moot rather than fixed." If a human prefers to keep a tracker open for the surviving `aiwf add epic` symptom, it needs a **new** gap stated against the real cause (`entity-body-empty` reads body bytes off disk, so a file the verb has not written yet yields nothing to escalate), not this body — and it is retired outright by ADR-0042's `tdd.strict` retirement whenever that lands.
- **G-0574** — leave the gap open; it is the strongest-measuring body in this batch. Edit two things. (1) The closing paragraph's "that decision … is currently unwritten" must cite **D-0046** (accepted 2026-07-21), which decides this exact question for the sibling gate and whose Consequences section generalises to "any future consumer that diffs two … result sets". (2) The "naive repair is unsafe" paragraph is about a *set*-keyed diff; the repo already ships the *counted* form of the same message-free key, and measured, it does **not** lose the fourth identical finding. Restate the open question as "set vs multiset, and which fields" rather than "no decision exists". Optionally add the unforced escape route (`aiwf milestone tdd --policy none` → walk the ladder → `--policy required`), which the body's "unforced" framing currently implies does not exist.
- **G-0575** — leave the body substantially alone. Two optional one-word precision edits: the hint's first remedy offers `<met|deferred|cancelled>`, not `<met|cancelled>` (the omitted `deferred` is refused too, so nothing changes); and "the harness composes on one branch" is true of the *walker* but five stress scenarios do run `git merge`. Note for the reader that `discovered_in: M-0300` now points at a `cancelled` milestone under a `cancelled` epic (`E-0080`), so the provenance trail dead-ends.
- **G-0576** — leave the defect alone. Two body edits: the sentence "then walk a second cycle attaching nothing at all" understates the recipe — the phase FSM refuses `done → red|green|done`, so the second cycle needs its own `--force --reason` on the first phase promote (G-0569's exact subject); and "G-0569's fix (reset the phase on a demote)" presents one of two undecided routes in an `open` gap as settled.
- **G-0577** — keep the gap, but edit the last two paragraphs. The sentence "flipping a terminal silently changes what `aiwf cancel` does with no test to catch it" is wrong — two tests fail, measured. So is "no AC analogue exists": `internal/verb/ac_same_state_noop_test.go:44` declares the AC terminal set `{deferred, cancelled}` and asserts it against `IsTerminalACStatus`. Both edits *strengthen* the gap's own conclusion (don't exempt the sub-element FSMs) while removing a claim a reader would find false in one run. The six-row table should either be re-measured and dated, or restated as a dated observation.
- **G-0578** — `aiwf promote G-0578 addressed --by-commit 793b1ad97`. It is a duplicate of G-0562 (same file, same helper, same single write) filed five days later; whichever is closed second should say so. The one paragraph worth carrying forward before closing — the second half of "Why no rule reaches it" — is that `internal/policies` remains in neither half of the chokepoint, which is a live question G-0562 already asks and G-0497 already owns. Also remove or re-word TODO.md:230, which is word-for-word identical to TODO.md:226 (G-0562).
- **G-0579** — keep open; it is the sharpest, best-supported gap in this batch. Before repairing, widen its scope in one `aiwf edit-body`: drop or soften "this one ships in the binary" (measurably false of the comment), replace "Two further surfaces" with the four that actually carry the retired reason, and drop "Only the Consequences have gone stale" — D-0015's *Context* and a second *Consequences* bullet are stale too, in different ways. The open question it poses (correct in place vs. supersede) is genuine and still needs a human; `D-0075` (accepted, 2026-08-23) already records "G-0579 remains a defect under this rule, because D-0015 is live."
- **G-0580** — treat as superseded by **G-0618** ("Provenance backstop covers 19 of 50 shipped files", open, priority low), which names the same two classes plus the templates, the guidance fragment and the statusline, and describes the *current* predicate. Either close G-0580 as a duplicate, or `aiwf edit-body G-0580` rewriting the first paragraph (mechanism), the "It is not theoretical" paragraph (evidence gone), and the closing paragraph (cost premise dead). Do not act on the body as it stands.
- **G-0581** — keep the gap open and `aiwf edit-body G-0581` on two sentences. Replace "Its population is the verb skills and the ritual skills" with the measured population (19 verb skills + the 10 `aiwfx-*` ritual skills; the 9 `wf-*` ritual skills are excluded by the loader's own filter), and drop or rewrite "it is the one shipped surface whose verb citations nothing resolves" — 20 of the 49 shipped markdown surfaces are unwalked. Worth recording in the body while editing: the widening the gap contemplates would fire immediately on `wf-doc-lint/SKILL.md:150`, whose `aiwf frobnicate --legacy` is a deliberate illustration inside a sample report — that false positive is real evidence for the human's cost/benefit call.
- **G-0583** — leave open, and `edit-body` two things. Correct "Three of M-0306's five criteria were rewritten mid-flight" to two (the milestone's own decisions section records AC-2 and AC-3), and soften "The remaining bullets check the baseline builds and the tests pass" — seven bullets remain and only two are build/tests. Worth adding, since it is the single most useful fact for whoever picks this up: **E-0085 attempted exactly this and was cancelled.** `wf-measure-spec` shipped 2026-08-14 (`6728317a5`, M-0308/AC-1) and was withdrawn 2026-08-15 (`4a4019ec1`, "withdraw wf-measure-spec; it contradicts the specification it was built from"); M-0309, whose AC-1 read "The milestone preflight names the ritual, not a bare judgment call", is `cancelled`. The live design capture is `docs/initiatives/milestone-preflight-as-independent-review.md` (2026-08-14, `status: captured`) and D-0066 (`accepted`).
- **G-0585** — leave the gap open. `aiwf edit-body G-0585` to repair one sentence — the `wf-doc-lint` "two commands / two heuristics" split does not re-derive (finding 2) — and optionally to drop the `not "no findings ever"` quote from the always-on guidance's half of the disposition claim (finding 3). Separately, the *initiative's* appendix carries a drifted line number (finding 1) which a human may want to fix in `docs/initiatives/milestone-preflight-as-independent-review.md` rather than in the gap.
- **G-0586** — keep open but `aiwf edit-body G-0586`, cutting or restating three sentences: (1) the "no entity template carries a section in which an unsettled claim would appear" sentence is false — `epic-spec.md` ships `## Open questions` and `aiwf show --format=json` surfaces it as `open_questions`; (2) "spends its budget entirely on conclusions" is false — the block is 6 lines against a ~10-line cap, so four lines of headroom exist and the cap is not what forecloses the slot; (3) the six-line enumeration doesn't match the six lines (line 1 is missing; "what landed" and "the next action" are one line). Also name the clearance gap by id (G-0585). What survives after that edit is a real and smaller gap: the block's payload rule names a settled finding and not an open one, and the pointer it substitutes reaches a *milestone*, whose template carries `## Deferrals` / `## Reviewer notes` but no open-question section.
- **G-0587** — do not act on this gap as written — a reader would go looking for a workaround to a block that does not exist. Either `aiwf edit-body G-0587` to restate it around the collision that *is* real (CLAUDE.md's "markdown link to a design/ADR doc" carve-out versus `aiwfx-record-decision`'s ban on exactly that link) plus the measured answer to its own open question, or promote it `addressed` / cancel it and file the real collision fresh. The `skill-body-id` question the body leaves open is now settled (finding 4) and should come out either way.
- **G-0588** — leave open; it is a genuine kernel decision awaiting a human. One `aiwf edit-body G-0588` is worth it: "Nothing consumes it" is too flat — the shipped `aiwf-contract` skill routes a class of adopter to `aiwf import` and names a planned manifest extension, and the retirement blast-radius sentence omits four surfaces the `rewidth` retirement precedent (`internal/policies/m0290_retirement_surface_test.go`) shows are in scope.
- **G-0589** — keep open, `aiwf edit-body`. Drop "seven" (there are six tracked root markdown files; prefer naming them or saying "the tracked root markdown files"). Delete the closing paragraph "It will fork a third time at implementation…" — D-0069 rejected the pass. Restate the "load-bearing rather than advisory" sentence without leaning on the review pass. Replace the D-0054 citation with the single- source-of-truth force (code-health C1) or drop the attribution. Scope "There is no vocabulary for a derived document" to the hierarchy section. Add a pointer to **ADR-0044** (`proposed`, 2026-08-16), which decides exactly this question — the gap currently has no reference to it, and ADR-0044's own Context paragraph carries the same dead premise, so whoever edits one should look at the other.
- **G-0590** — keep open; `aiwf edit-body G-0590` to fix four supports — repoint the renderer citation from `history.go:135` to `history.go:138`, drop or correct "the HTML render path carries the same field" (it does not), re-derive the cancel-commit count and drop the wrong "entity-level" qualifier, re-derive "36 of 37", and soften "four-hundred-word" to the measured 162. The thesis and the fix both survive unchanged.
- **G-0591** — keep open; `aiwf edit-body G-0591` to fix the two date claims in "Why it matters" ("twelve days", "months") — neither re-derives — and to resolve the taxonomy tension around the `CLAUDE.md:51` instance, whose target is `addressed` (delivered) rather than declined. The measured-instances list and the no-oracle argument need no change.
- **G-0594** — `aiwf edit-body G-0594` on four sentences: (1) drop or restate "The specification's only gloss is …" — the specification carries an explicit definition of *ambiguous* 16 lines further on; (2) change "A later epic under the same initiative" to "a later milestone under the same epic" (both documents are E-0086's); (3) drop "and it is an open gap" from the G-0584 sentence (now `addressed`) — "a gap … records a defect and settles nothing" carries the argument without the status; (4) tighten the D-0054 contrast, which is under-inclusive. Consider naming M-0310 and M-0311 as the two breaching documents, so a later reader can reach the ledgers that hold the extracted rows — currently neither set of five is extractable from the body.
- **G-0595** — keep open at high; `aiwf edit-body G-0595` to fix three overstatements — the opening "every live planning record" (the inventory excludes epics and milestones), "most carry claims the code contradicts" (the measured figure is "≥1 finding of any of 24 classes"), and "six causes account for most of the findings" (the inventory's own tables attribute well under half). Also worth recording, in the follow-on bullet, that the `docs/adr/` exclusion is a reasoned design choice rather than an oversight, so the follow-on reads as a decision to take rather than a scope change to make.
- **G-0600** — `aiwf edit-body G-0600`. Rewrite the "Stamping is uneven" and "Comparing versions is not a plain ordering" paragraphs: both are already answered by `internal/skills/statusline.go` (`statuslineVersionRE:26`, `InstalledStatuslineVersion:41`, `decideStatuslineRefresh:135`), which stamps, reads the stamp back, refuses to downgrade, and handles the untagged/dirty case by declining to order. Re-point the resolution shape at "extend the statusline's upgrade-only refresh to guidance, skills and rituals" rather than "design a comparison". Keep everything else — the mechanism, the report wording, the doctor remedy, and the two Related pointers all measured true. Do **not** promote addressed.
- **G-0601** — keep the gap open and `aiwf edit-body G-0601` to fix three sentences: (1) the drop rule is `aiwf-verb` **or** `aiwf-actor`, not `aiwf-verb` alone; (2) "Every row that projection does render carries a verb" is false — replace with the measured fact that 19 commits on `main` render with a blank verb column; (3) the "Measured on the milestone's own implementation commit" paragraph names a commit the gate never sees — re-cite `91493c23f` / G-0620, which the gate does read. The Options paragraph then needs rewording: "how to label a row with no verb to name" is already answered in shipped code (blank verb column), so the open question is narrower than the body states.
- **G-0602** — leave the gap open, but edit the `## Options` paragraph. Two things in it are now inaccurate as guidance: the "say so where the premise is stated" alternative already landed in the policy's own doc comment at `667c8e0fc` (M-0312) — what is *not* said is CLAUDE.md §"Ritual content authoring" — and the flag survey omits `--diff-merges=combined`, which is strictly better than `first-parent` (it drops paths taken verbatim from the other side) while still not distinguishing a conflict resolution from a clean auto-merge, so the paragraph's conclusion survives but its evidence should name the better flag.
- **G-0603** — leave the body alone apart from one optional precision fix (finding below). Worth recording in the disposition that commit `370ad65d` (2026-08-21, *"fix(rituals): drop CLAUDE.md section citations that dangle in consumer repos"*) is an untrailered ten-file ritual edit sitting on `main` today — the gate landed after it in the DAG, so it was never blocked and, per the gap's own argument, never repaired.
- **G-0604** — `aiwf edit-body G-0604`. Rewrite the first paragraph of *What's missing* to name the five sites and what actually varies between them (canonicalized vs raw key; index-build vs single-id predicate; prior-ids arm present in exactly one). Retitle away from "four copies". The *Options* paragraph's closing caution ("worth checking whether the four callers genuinely want identical semantics") is the strongest sentence in the body and should be promoted to the premise rather than left as an aside — it is already measurably true.
- **G-0605** — `aiwf edit-body G-0605` to fix the opening sentence ("Every Go-source check … is a pattern match over a parsed syntax tree") and the "roughly sixty-six checks" figure; the same two corrections are needed in `docs/explorations/10-type-aware-static-analysis.md` (lines 18 and 99), plus its two off-by-one line citations. Leave the gap open.
- **G-0606** — `aiwf edit-body G-0606`. Replace "The live instance is `m0211-guidance-operating-anchors`" with the measured pair (`m0211-guidance-operating-anchors` **and** `m0210-trailer-commit-drift`), and delete or rewrite the closing paragraph — "one instance exists and it is documented … It grows the moment a second policy reaches for the shape" — since the second policy exists, is older, and is *not* documented as a deliberate exemption anywhere. Also soften the D-0070 attribution (see finding below).
- **G-0607** — `aiwf edit-body G-0607`. Drop or re-derive the "twelve sites" figure — it does not re-derive under any reading I could construct — and replace the `chore\(epic\): wrap` example, which names a site deleted by M-0313 *before* this gap was written. The live prose-ish instance to cite instead is `internal/policies/skill_table_severity_placement_test.go:875` /`:926`: `noEscalationClaim = regexp.MustCompile("no strictness knob|never escalates|does not escalate")` and `escalationClaim = regexp.MustCompile("(?i)escalated to error|escalates to \\*\\*error\\*\\*|promoted to \\*\\*error\\*\\*|severity escalates")`, both matched against rows read from the shipped `aiwf-check` skill, and both load-bearing (the test `t.Fatalf`s when the phrase list matches nothing).
- **G-0608** — keep the gap; `aiwf edit-body G-0608` to (a) replace "The instance is" with the two instances actually in the tree, adding `TestM0142_AC2_OldGapCodeFullyRenamed` — which matters because it is *invisible to the ban* and so would survive any decision taken only against the named one; (b) fix the "which is the failure D-0070 measured" attribution; (c) drop the two session-narration sentences. Separately worth an inline fix while someone is in the file: `TestNoReintroducedDeadVerbForms_ContractsAndSkill` still carries "AndSkill" in its name while watching no skill — a test whose name promises coverage it no longer has is this cluster's own thesis.
- **G-0613** — Keep the gap; `aiwf edit-body` three sentences. (1) The Direction section's closing sentence "The wrap ritual is now the only surface naming a category set … it settles in one place" is false — `wf-patch`'s SKILL.md step 4 has named the identical `Added` / `Changed` / `Fixed` set since 2026-07-05, so the fix lands in two shipped surfaces. (2) "the decision named three categories deliberately" overstates D-0031, whose own words are "already in use today" — a transcription, which is one of the two horns the gap poses as the open question, so the body pre-answers itself. (3) The second References paragraph is drafting-history narration and can go; what survives it is the one live fact (the release ritual now names no set). Optionally widen the evidence sentence: the tree carries `### Security` and `### Internal` too. No test pins the category set, so widening it is a two-file template edit.
- **G-0618** — keep the gap; `aiwf edit-body G-0618` to change "19 of 50" → "19 of 51" and add the shipped hook script (`internal/skills/embedded-hooks/worktree-rituals-check.sh`) to the "Outside it" enumeration (31 → 32), and `aiwf retitle` the title, which carries the same number. Separately, decide G-0618 against G-0580 — they are the same scope hole, and G-0580 states it against a predicate D-0071 retired.
- **G-0622** — Drop the cluster-4 entry, or move it to the in-flight header as work E-0088 already carries. It arrives on `main` already terminal.
