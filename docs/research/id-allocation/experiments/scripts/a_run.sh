#!/usr/bin/env bash
# Design A — the default branch as the ledger. Three clones allocate concurrently from the same
# base (gap + epic each, cited with markers) and open merge requests; the host rule in a_host.sh
# merges one at a time. A refused developer brings the branch up to date by <policy>:
#   right  fetch; reallocate every own entity aiwf check reports as a trunk-collision (the tree
#          holds only the developer's own copy); then merge origin/main in
#   wrong  fetch; merge origin/main in; reallocate every id aiwf check reports as ids-unique
#          (the tree holds both copies; the trunk-ancestry tiebreaker picks the target)
# <schedule>:
#   seq    a refused developer updates and pushes before the next merge request is tried
#   simul  every refused developer of a round fetches first, then all update and push
# <fetch>: all = `git fetch origin` (sees other branches); main = `git fetch origin main` only
source "$(dirname "$0")/lib.sh"; source "$(dirname "$0")/a_host.sh"
policy=${1:-right}; schedule=${2:-seq}; scope=${3:-all}
setup "$X/runs/a-$policy-$schedule-$scope"
ATTEMPTS=0
mkclone alpha hooks; mkclone beta hooks; mkclone gamma hooks
work() { # work <who> <cite-from>
  cd "$W/$1"; git checkout -q -b "mr/$1"
  local g e; g=$(addgap "$1 gap"); e=$(addepic "$1 epic")
  addms "$e" "$1 ms (parent means $1-epic)" >/dev/null
  cite "$2" "See $g (means:$1-gap)."
  echo "  $1 allocated gap=$g epic=$e"
  git push -q origin "mr/$1" 2>&1 | grep -E ' error ' ; : > "$W/count.$1"
}
work alpha G-0001; work beta M-0001; work gamma E-0001
fetch_() { if [ "$scope" = all ]; then git fetch -q origin; else git fetch -q origin main; fi; }
update() { # update <who>, cwd = the clone; assumes fetch already done
  local p i n=0 guard=0
  case $policy in
    right)
      while p=$(aiwf check </dev/null 2>&1 | grep -oE 'trunk-collision: id "[A-Z]+-[0-9]+" is allocated on this branch \([^)]*\)' | head -1 | sed -E 's/.*\((.*)\)/\1/'); [ -n "$p" ] && [ $guard -lt 8 ]; do guard=$((guard+1))
        aiwf reallocate "$p" </dev/null >/dev/null || die "reallocate $p"; n=$((n+1)); done
      git merge -q --no-edit origin/main || die "merge" ;;
    wrong)
      git merge -q --no-edit origin/main || die "merge"
      while i=$(aiwf check </dev/null 2>&1 | grep -oE 'error ids-unique: id "[A-Z]+-[0-9]+"' | head -1 | grep -oE '[A-Z]+-[0-9]+'); [ -n "$i" ] && [ $guard -lt 8 ]; do guard=$((guard+1))
        aiwf reallocate "$i" </dev/null >/dev/null || die "reallocate $i"; n=$((n+1)); done ;;
  esac
  echo $n >> "$W/count.$1"
  echo "    $1 updated: $n reallocate(s); now $(ls work/gaps | grep "$1-gap" | cut -c1-6) / $(ls work/epics | grep "$1-epic" | cut -c1-6)"
}
pushb() { git push -q origin "mr/$1" 2>&1 | grep -E ' error |rejected' ; }
queue=(alpha beta gamma); round=0
while [ ${#queue[@]} -gt 0 ] && [ $round -lt 8 ]; do
  round=$((round+1)); refused=()
  for who in "${queue[@]}"; do
    ATTEMPTS=$((ATTEMPTS+1)); r=$(host_mr "mr/$who"); echo "  round $round: MR mr/$who -> $r"
    if [ "$r" != MERGED ]; then
      if [ "$schedule" = seq ]; then (cd "$W/$who"; fetch_; update "$who"; pushb "$who"); fi
      refused+=("$who")
    fi
  done
  if [ "$schedule" = simul ]; then
    for who in "${refused[@]}"; do (cd "$W/$who"; fetch_); done
    for who in "${refused[@]}"; do (cd "$W/$who"; update "$who"; pushb "$who"); done
  fi
  queue=("${refused[@]}")
done
total=$(cat "$W"/count.* | awk '{s+=$1} END{print s+0}')
echo "== summary: policy=$policy schedule=$schedule fetch=$scope merge-attempts=$ATTEMPTS reallocates=$total rounds=$round"
rm -rf "$W/after"; git clone -q "$W/host.git" "$W/after"; cd "$W/after"
echo "  gaps:  $(ls work/gaps | tr '\n' ' ')"
echo "  epics: $(ls work/epics | tr '\n' ' ')"
show aiwf check | grep -E 'error|findings|exit' | cut -c1-170
verify
