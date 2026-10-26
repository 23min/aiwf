#!/usr/bin/env bash
# Design B — three clones run the allocate-then-claim protocol at the same moment (background jobs),
# then a host merges all three branches with no check. Repeated <trials> times.
source "$(dirname "$0")/lib.sh"; source "$(dirname "$0")/b_lib.sh"
trials=${1:-5}
: > "$X/out/push-errors.log"
for t in $(seq 1 "$trials"); do
  setup "$X/runs/b-race"; for who in alpha beta gamma; do mkclone "$who" hooks; done
  ( cd "$W/alpha"; file_gap alpha G-0001 > "$W/res.alpha" ) &
  ( cd "$W/beta";  file_gap beta  M-0001 > "$W/res.beta" ) &
  ( cd "$W/gamma"; file_gap gamma E-0001 > "$W/res.gamma" ) &
  wait
  hostclone; ( cd "$W/hostwork"; for who in alpha beta gamma; do git merge -q --no-ff --no-edit "origin/mr/$who" || echo "merge conflict $who"; done
    git push -q --no-verify origin main )
  rm -rf "$W/after"; git clone -q "$W/host.git" "$W/after"; cd "$W/after"
  claims=$(git ls-tree --name-only origin/aiwf/ids work/gaps/ | xargs -n1 basename | tr '\n' ' ')
  errs=$(aiwf check </dev/null 2>&1 | grep -c 'error ids-unique')
  v=$(verify | tail -1)
  stab=""; for who in alpha beta gamma; do shared=$(grep -oE 'first-shared=G-[0-9]+' "$W/res.$who" | cut -d= -f2); onmain=$(basename "$(ls work/gaps/*-$who-gap.md)" | cut -c1-6); stab="$stab $who:$shared->$onmain"; done
  echo "trial $t: $(cat "$W"/res.alpha) | $(cat "$W"/res.beta) | $(cat "$W"/res.gamma)"
  echo "         ledger claims: $claims; main gaps: $(ls work/gaps | cut -c1-6 | tr '\n' ' '); ids-unique errors on main: $errs;$v; shared->main:$stab"
  cd "$X"
done
echo "push rejection reasons seen (counts):"
grep -oE '\((fetch first|non-fast-forward|stale info|incorrect old value provided|reference already exists)\)|cannot lock ref' "$X/out/push-errors.log" | sort | uniq -c
