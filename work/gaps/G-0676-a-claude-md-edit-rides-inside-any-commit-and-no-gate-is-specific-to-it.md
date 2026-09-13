---
id: G-0676
title: A CLAUDE.md edit rides inside any commit and no gate is specific to it
status: open
priority: high
discovered_in: E-0092
---
## What's missing

No gate is specific to an edit of `CLAUDE.md`. Four surfaces together let the file grow inside ordinary work, without the change ever being put as its own question.

**The commit seam.** Nothing under `internal/policies/` scopes `CLAUDE.md`. `skill_edit_provenance_backstop.go` holds a `SKILL.md` edit to a commit that names its owning entity; `CLAUDE.md` has no counterpart, so an edit to it rides inside any commit that also touches code, and passes as that commit. Measured 2026-09-13 in the devcontainer at the tree of E-0092's allocation commit:

```
git log --format=%h -- CLAUDE.md | wc -l                                    # 149 commits touch it
git log --format=%h -- CLAUDE.md | while read h; do git show --stat --format= "$h" | grep -c '|'; done | awk '$1>1' | wc -l   # 123 also touch other files
git log --format='%h %(trailers:key=aiwf-entity,valueonly,separator=%x2C)' -- CLAUDE.md | awk 'NF==1' | wc -l   # 135 carry no aiwf-entity trailer
```

**The discoverability channel list.** `internal/policies/discoverability.go` line 161 lists `CLAUDE.md` among the channels that make a finding code or config field discoverable, so one line there is the cheapest way to satisfy `finding-codes-are-discoverable` and `config-fields-are-discoverable` for anything new.

**The AC-evidence rule's scope.** `CLAUDE.md` §"AC promotion requires mechanical evidence" accepts, for a doc-shaped AC, a structural assertion scoped to a named markdown section, and §"Test design rules" (the substring-assertion bullet) states that D-0070's ban on prose-presence assertions stops at the shipped surfaces and leaves `CLAUDE.md` unchanged. Together they make "`CLAUDE.md` documents X" a writable criterion whose evidence is a sentence pinned in the file, and the pin then holds the sentence in place:

```
grep -rhE '^\s+title: .*CLAUDE\.md' work/epics --include='M-*.md' | sort -u | wc -l   # 23 AC titles name the file
git worktree add --detach /tmp/floor HEAD && rm /tmp/floor/CLAUDE.md \
  && (cd /tmp/floor && go test ./internal/policies/ -count=1 | grep -c '^--- FAIL')   # 21 tests fail with the file deleted
```

**The principle text.** `CLAUDE.md` §"Engineering principles" names "this file" as a channel through which kernel functionality may be made discoverable.

## Why it matters

The operating rule is that `CLAUDE.md` changes only as its own decision, gated in conversation. That rule has no chokepoint, so it holds by vigilance, and the record shows vigilance did not hold it:

```
for d in 2026-05-01 2026-07-01 2026-08-01; do git show $(git rev-list -1 --before=$d main -- CLAUDE.md):CLAUDE.md | wc -w; done
# 962, 12722, 8033; the working tree today: 9234
```

The split that moved operating rules into the shipped fragment cut the file by more than a third; it has regrown about a hundred words a week since. Each edit was approved, as the feature commit or the milestone plan it rode in, and none was decided. Every pinned sentence also raises the cost of the next cut, since the cut must first meet the test that holds the sentence.

E-0092 shrinks the file to a ceiling. Without a fence at the commit seam, the shrink regrows through the same hole at the same rate, and the ceiling policy alone would then refuse the regrowth without saying which edit should not have ridden where it did.
