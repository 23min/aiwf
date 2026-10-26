#!/usr/bin/env bash
# S0.3 — where a cross-machine collision is detected with aiwf's hooks installed, and what
# each repair order does to the references. <mode>:
#   before-merge  beta reallocates on its branch, then merges origin/main in
#   after-merge   beta merges origin/main in, then reallocates (the hint's route)
#   after-rebase  beta rebases onto origin/main, then reallocates
source "$(dirname "$0")/lib.sh"; source "$(dirname "$0")/actors.sh"
mode=${1:-before-merge}
setup "$X/runs/s0rep-$mode"; mkclone alpha hooks; mkclone beta hooks
( cd "$W/alpha"; git checkout -q -b feature/A; work_alpha; show git push -q origin feature/A | tail -1 )
cd "$W/beta"; git checkout -q -b feature/B; work_beta
echo "== D1: beta fetches (sees origin/feature/A) and pushes its branch; A is not merged yet"
git fetch -q origin; show git push -q origin feature/B | grep -E 'error|exit'
echo "== host merges feature/A into main (no check)"
hostclone; ( cd "$W/hostwork"; git merge -q --no-ff --no-edit origin/feature/A; git push -q --no-verify origin main )
cd "$W/beta"
echo "== D2: beta commits again and pushes WITHOUT fetching (origin/main is stale)"
cite G-0003 "Beta finding note 1."; show git push -q origin feature/B | grep -E 'error|exit'
echo "== D3: beta fetches and pushes again"
cite G-0003 "Beta finding note 2."; git fetch -q origin
show git push -q origin feature/B | grep -E 'ids-unique|exit' | cut -c1-170
echo "== repair: $mode"
case $mode in
  before-merge)
    show aiwf reallocate "$(ls work/gaps/G-0002-beta-gap.md)"
    show aiwf reallocate "$(ls work/epics/E-0002-beta-epic/epic.md)"
    show git merge -q --no-edit origin/main ;;
  after-merge)
    show git merge -q --no-edit origin/main
    show aiwf reallocate G-0002
    show aiwf reallocate "$(ls work/epics/E-0002-beta-epic/epic.md)" ;;
  after-rebase)
    show git rebase -q origin/main | tail -1
    show aiwf reallocate G-0002
    show aiwf reallocate "$(ls work/epics/E-0002-beta-epic/epic.md)" ;;
esac
verify
push_flag=; [ "$mode" = after-rebase ] && push_flag=--force-with-lease
show git push -q $push_flag origin feature/B | grep -E 'error|findings|exit' | cut -c1-170
echo "== host merges feature/B into main; fresh clone of main"
hostclone; ( cd "$W/hostwork"; git merge -q --no-ff --no-edit origin/feature/B; git push -q --no-verify origin main )
rm -rf "$W/after"; git clone -q "$W/host.git" "$W/after"; cd "$W/after"
echo "  gaps:  $(ls work/gaps | tr '\n' ' ')"
show aiwf check | grep -E 'error|findings|exit' | cut -c1-170
verify
echo "== history as aiwf reports it"
show aiwf history G-0002
show aiwf history "$(ls work/gaps | grep beta-gap | cut -c1-6)"
