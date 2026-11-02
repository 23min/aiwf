#!/usr/bin/env bash
# S0.4b — same as s0_samepath.sh "identical", with epics created from aiwf's own scaffold body
# (no --body): two people independently `aiwf add epic --title "Q3 planning"`.
source "$(dirname "$0")/lib.sh"
setup "$X/runs/s0same-epic"; mkclone alpha hooks; mkclone beta hooks
for who in alpha beta; do ( cd "$W/$who"; git checkout -q -b feature/$who
  aiwf add epic --title "Q3 planning" </dev/null; git push -q origin feature/$who 2>&1 | grep -E 'error|findings' ); done
echo "  blob alpha: $(git -C $W/beta rev-parse origin/feature/alpha:work/epics/E-0002-q3-planning/epic.md 2>/dev/null || git -C $W/alpha rev-parse HEAD:work/epics/E-0002-q3-planning/epic.md)"
echo "  blob beta:  $(git -C $W/beta rev-parse HEAD:work/epics/E-0002-q3-planning/epic.md)"
hostclone; cd "$W/hostwork"
for who in alpha beta; do show git merge --no-ff --no-edit origin/feature/$who | grep -E 'CONFLICT|Merge made|exit'; done
git push -q origin main                     # the host's push; hostclone installs no hooks
echo "== a fresh clone of main"
rm -rf "$W/carol"; mkclone carol; cd "$W/carol"
echo "  epics: $(ls work/epics | tr '\n' ' ')"
show aiwf check | grep -E 'error|findings|exit'
show aiwf history E-0002
