#!/usr/bin/env bash
# S02: the review's E2 shape under the corrected rule.
#  (a) start-epic: promote + authorize on main (unpushed), cut the epic branch,
#      verbs on both sides, merge back.
#  (b) start-milestone: promote + authorize on the epic branch, cut the
#      milestone branch, verbs on both, merge back.
#  (c) hole: a second clone made from the working repo (not the remote)
#      holds the unpushed planning commit; no ref here shows it.
. "$(dirname "$0")/lib.sh"
new_env s02; init_repo repo
AG=(--actor ai/claude --principal human/peter)

section "plan on main, push"
x aiwf add epic --title "Epic one"
x aiwf add milestone --epic E-0001 --tdd none --title "First milestone"
x aiwf add milestone --epic E-0001 --tdd none --title "Second milestone"
x git push -q origin main
lg

section "(a) start-epic shape: promote + authorize on main, unpushed"
x aiwf promote E-0001 active
x aiwf authorize E-0001 --to ai/claude --branch epic/E-0001-epic-one --reason "delegate the epic"
lg -2
x git switch -c epic/E-0001-epic-one
section "(a) on the epic branch: first verb (human) — HEAD is also on main"
x aiwf retitle M-0002 "Second milestone, retitled"
section "(a) agent verb on the epic branch"
x aiwf retitle M-0002 "Second milestone, retitled by agent" "${AG[@]}"
lg -4
section "(a) back on main: a verb there while the epic branch contains main's HEAD"
x git switch main
x aiwf add gap --title "Main-side gap" --body $'## What\'s missing\n\nx.\n\n## Why it matters\n\ny.'
lg -3

section "(b) start-milestone shape on the epic branch"
x git switch epic/E-0001-epic-one
x aiwf promote M-0001 in_progress
x aiwf authorize M-0001 --to ai/claude --branch milestone/M-0001-first-milestone --reason "delegate the milestone"
lg -3
x git switch -c milestone/M-0001-first-milestone
x aiwf add ac M-0001 --title "first criterion" "${AG[@]}"
x aiwf add ac M-0001 --title "second criterion" "${AG[@]}"
x aiwf promote M-0001/AC-1 met "${AG[@]}"
lg -4
section "(b) human verb on the epic branch while the milestone branch contains its HEAD"
x git switch epic/E-0001-epic-one
x aiwf retitle M-0002 "Second milestone, retitled on epic"
lg -3

section "merge milestone -> epic, epic -> main"
x git merge --no-ff -q -m "merge milestone/M-0001" milestone/M-0001-first-milestone
x git switch main
x git merge --no-ff -q -m "merge epic/E-0001" epic/E-0001-epic-one
lg --graph -14

section "every operation recorded once? (ops.py over main)"
ops main
section "aiwf check on main"
x aiwf check
section "push (pre-push hook)"
x git push origin main

section "(c) hole: a clone of the working repo holds the unpushed commit"
x aiwf add gap --title "Gap before clone" --body $'## What\'s missing\n\nx.\n\n## Why it matters\n\ny.'
x git clone -q "$T/repo" "$T/peer"
echo "refs in repo that contain HEAD: $(git for-each-ref --contains HEAD --format='%(refname)' | tr '\n' ' ')"
section "(c1) next op touches a different file (silent case)"
x aiwf add decision --title "Decision after clone" --body $'## Question\n\nq?\n\n## Decision\n\nd.\n\n## Reasoning\n\nr.'
lg -2
section "(c1) the peer pushes its copy of the pre-amend commit; repo pulls (merge)"
( cd "$T/peer" && git remote set-url origin "$REMOTE" && git push -q origin main 2>&1 | tail -1; echo "peer pushed $(git rev-parse --short HEAD)" )
x git pull -q --no-rebase origin main
lg --graph -5
ops main
x aiwf history G-0002
x aiwf check
x git push origin main
