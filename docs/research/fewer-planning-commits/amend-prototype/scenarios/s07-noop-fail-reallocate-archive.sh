#!/usr/bin/env bash
# S07: NoOp verbs, a verb that fails validation mid-run, aiwf reallocate
# (inside a run, and after a merge collision), aiwf archive, the
# post-commit hook and STATUS.md.
. "$(dirname "$0")/lib.sh"
new_env s07; init_repo repo
GB=$'## What\'s missing\n\nx.\n\n## Why it matters\n\ny.'
DB=$'## Question\n\nq?\n\n## Decision\n\nd.\n\n## Reasoning\n\nr.'
gapt() { x aiwf add gap --title "$1" --body "$GB"; }

section "a: NoOp inside a run"
x aiwf add decision --title "decision a" --body "$DB"
x aiwf promote D-0001 accepted
H1=$(git rev-parse HEAD)
x aiwf promote D-0001 accepted
echo "HEAD unchanged by the NoOp: $([ "$H1" = "$(git rev-parse HEAD)" ] && echo yes || echo NO)"
x aiwf promote D-0001 accepted --format=json

section "b: a verb that fails validation mid-run, then the run continues"
x aiwf add epic --title "epic b"
x aiwf promote E-0001 done
x aiwf set-priority D-0001 high
H2=$(git rev-parse HEAD)
x git status --short
lg -1

section "c1: reallocate inside a run (add, reallocate, then a verb on the new id)"
x git push -q origin main
gapt "gap c1"
x aiwf reallocate G-0001
NEW=$(ls work/gaps | grep -i 'gap-c1' | cut -c1-6); echo "new id: $NEW"
x aiwf set-priority "$NEW" high
lg -1
git log -1 --format='%(trailers:only,unfold)'
x aiwf history "$NEW"
x aiwf history G-0001
x aiwf show "$NEW"

section "c2: reallocate after a merge collision"
x git push -q origin main
x git switch -q -c other
gapt "gap on other"
x git switch -q main
gapt "gap on main"
x git merge -q --no-ff -m "merge other" other
x aiwf check
P1=$(ls work/gaps | grep 'gap-on-other' )
x aiwf reallocate "work/gaps/$P1"
NEW2=$(ls work/gaps | grep 'gap-on-other' | cut -c1-6)
x aiwf set-priority "$NEW2" low
lg -3
x aiwf history "$NEW2"
x aiwf check

section "d: cancel then archive in one run"
x git push -q origin main
gapt "gap d"
GD=$(ls work/gaps | grep 'gap-d\.md' | cut -c1-6)
x git push -q origin main
x aiwf cancel "$GD" --reason "not needed"
x aiwf archive --apply
lg -1
git log -1 --format=%B | head -30
x aiwf history "$GD"
x aiwf show "$GD"
x aiwf check

section "e: post-commit hook and STATUS.md after amends"
x git push -q origin main
gapt "gap e1"
cp STATUS.md "$T/status-after-new-commit.md"
x aiwf set-priority "$(ls work/gaps | grep 'gap-e1' | cut -c1-6)" urgent
cp STATUS.md "$T/status-after-amend.md"
echo "--- STATUS.md diff (new commit -> after amend):"
diff "$T/status-after-new-commit.md" "$T/status-after-amend.md"
echo "--- STATUS.md recent activity after amend:"
sed -n '/[Rr]ecent/,/^$/p' STATUS.md | head -20
x git status --short
x aiwf status
