#!/usr/bin/env bash
# Design A edges.
#  E1: same title + aiwf's scaffold body on two branches (byte-identical epic.md) through the host rule.
#  E2: two host workers run the host rule at once on two up-to-date merge requests (10 trials):
#      does the non-force push of main let both land?
source "$(dirname "$0")/lib.sh"; source "$(dirname "$0")/a_host.sh"
echo "######## E1 identical same-path entity through the host rule"
setup "$X/runs/a-edge-identical"; mkclone alpha hooks; mkclone beta hooks
for who in alpha beta; do ( cd "$W/$who"; git checkout -q -b mr/$who
  aiwf add epic --title "Q3 planning" </dev/null >/dev/null; git push -q origin mr/$who 2>/dev/null ); done
echo "  MR mr/alpha -> $(host_mr mr/alpha)"
echo "  MR mr/beta  -> $(host_mr mr/beta)"
( cd "$W/beta"; git fetch -q origin; show git merge --no-edit origin/main | grep -E 'CONFLICT|Merge made|Already|exit'; git push -q origin mr/beta 2>&1 | grep -E ' error ' )
echo "  MR mr/beta  -> $(host_mr mr/beta)"
rm -rf "$W/after"; git clone -q "$W/host.git" "$W/after"; cd "$W/after"
echo "  epics on main: $(ls work/epics | tr '\n' ' ')"
show aiwf history E-0002

echo "######## E2 two host workers at once"
merged_total=0
for t in $(seq 1 10); do
  setup "$X/runs/a-edge-race"; mkclone alpha; mkclone beta
  ( cd "$W/alpha"; git checkout -q -b mr/alpha; addgap "alpha gap" >/dev/null; git push -q --no-verify origin mr/alpha )
  ( cd "$W/beta"; git checkout -q -b mr/beta; aiwf add decision --title "beta decision" --body $'## Question\n\nq.\n\n## Decision\n\nd.\n\n## Reasoning\n\nr.\n' </dev/null >/dev/null || echo "decision add failed"; git push -q --no-verify origin mr/beta )
  # prepare both workers' clones first so the race is only fetch/check/merge/push
  host_mr mr/none 1 >/dev/null 2>&1; host_mr mr/none 2 >/dev/null 2>&1
  host_mr mr/alpha 1 > "$W/r1" & host_mr mr/beta 2 > "$W/r2" & wait
  m=$(git -C "$W/host.git" log --oneline main | grep -c 'Merge')
  echo "  trial $t: worker1(alpha)=$(cat "$W/r1")  worker2(beta)=$(cat "$W/r2")  merge commits on host main=$m"
done
