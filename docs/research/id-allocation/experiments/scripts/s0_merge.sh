#!/usr/bin/env bash
# S0.1 — today's behaviour. Two clones allocate from the same base without fetching each
# other's work, push their branches, and a host merges both merge requests by <method>:
#   noff        git merge --no-ff of each branch in turn
#   ff-rebase   rebase each branch onto the current main, then fast-forward
#   semilinear  rebase each branch onto the current main, then git merge --no-ff
# <hooks>: "nohooks" = the host pushes main with no aiwf hooks (a hosted merge button);
#          "hooks"   = a maintainer merges locally with aiwf's hooks installed and pushes main.
# Then the collision is looked for on main and repaired there by aiwf reallocate.
source "$(dirname "$0")/lib.sh"; source "$(dirname "$0")/actors.sh"
method=${1:-noff}; hooks=${2:-nohooks}
setup "$X/runs/s0-$method-$hooks"
mkclone alpha; mkclone beta
( cd "$W/alpha"; git checkout -q -b feature/A; work_alpha; git push -q --no-verify origin feature/A )
( cd "$W/beta";  git checkout -q -b feature/B; work_beta;  git push -q --no-verify origin feature/B )

echo "== host merges feature/A then feature/B by $method ($hooks)"
hostclone; cd "$W/hostwork"
[ "$hooks" = hooks ] && for h in pre-push pre-commit commit-msg post-commit; do cp "$W/seed/.git/hooks/$h" .git/hooks/; done
for b in A B; do
  case $method in
    noff)       show git merge --no-ff --no-edit origin/feature/$b | tail -3 ;;
    ff-rebase)  git checkout -q -b tmp$b origin/feature/$b
                show git rebase main | tail -2
                git checkout -q main; show git merge --ff-only tmp$b | tail -2 ;;
    semilinear) git checkout -q -b tmp$b origin/feature/$b
                show git rebase main | tail -2
                git checkout -q main; show git merge --no-ff --no-edit tmp$b | tail -2 ;;
  esac
done
if [ "$hooks" = hooks ]; then show git push origin main; else show git push --no-verify origin main; fi
pushed=$?
echo "== state of host main (pushed exit=$pushed)"
rm -rf "$W/after"; git clone -q "$W/host.git" "$W/after"; cd "$W/after"; git config user.name carol; git config user.email carol@example.com
echo "  gaps:  $(ls work/gaps | tr '\n' ' ')"
echo "  epics: $(ls work/epics | tr '\n' ' ')"
show aiwf check | grep -E 'ids-unique|findings|exit'
verify
[ "$pushed" = 0 ] || exit 0
echo "== repair on main: carol reallocates beta's gap and beta's epic (by path)"
show aiwf reallocate "$(ls work/gaps/G-0002-beta-gap.md)"
show aiwf reallocate "$(ls work/epics/E-0002-beta-epic/epic.md)"
echo "  gaps:  $(ls work/gaps | tr '\n' ' ')"
echo "  epics: $(ls -R work/epics | grep -E '^work|\.md$' | tr '\n' ' ')"
show aiwf check | grep -E 'error|findings|exit'
verify
