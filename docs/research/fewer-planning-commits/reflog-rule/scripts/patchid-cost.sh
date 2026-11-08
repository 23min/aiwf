#!/usr/bin/env bash
# The cost of a patch-id test (is HEAD's change already on a remote-tracking ref HEAD lacks?)
# against the cost of reading HEAD's reflog. A patch-id test computes one patch-id per commit
# the remote has and HEAD lacks; the two largest branch merges into main in the window stand
# in for a branch pushed and then rebased.
# Usage: patchid-cost.sh <repo>
set -eu
R=${1:?repo}; cd "$R"; TIMEFORMAT='%R s'
git --version; echo "repo HEAD: $(git rev-parse --short HEAD)"
echo "remote-tracking refs: $(git for-each-ref refs/remotes --format=x | wc -l)"
echo "commits on remote-tracking refs that HEAD lacks: $(git rev-list --count --no-merges --remotes --not HEAD)"
echo "== the reflog read, 3 runs"; for i in 1 2 3; do time git reflog -1 HEAD --format=%gs >/dev/null; done
echo "== patch-ids of every remote commit HEAD lacks, 3 runs"
for i in 1 2 3; do time (git log -p --no-merges --format='commit %H' --remotes --not HEAD | git patch-id --stable >/dev/null); done
for m in 14185cd09daa304109be19bca767d6c78d948947 760ef9401c50d3e4b58813c1688cc74d7e0df11d; do
  echo "== merge ${m:0:9}: patch-ids of the $(git rev-list --count --no-merges $m^2 --not $m^1) branch commits main lacked, 3 runs"
  for i in 1 2 3; do time (git log -p --no-merges --format='commit %H' $m^2 --not $m^1 | git patch-id --stable >/dev/null); done
done
