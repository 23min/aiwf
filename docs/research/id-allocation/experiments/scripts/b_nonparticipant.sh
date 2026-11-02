#!/usr/bin/env bash
# Design B — a clone that does not follow the protocol (no fetch since cloning, plain aiwf add)
# next to one that does. Does the ledger protect main?
source "$(dirname "$0")/lib.sh"; source "$(dirname "$0")/b_lib.sh"
setup "$X/runs/b-nonpart"; mkclone alpha hooks; mkclone dave hooks
( cd "$W/alpha"; file_gap alpha G-0001 )
( cd "$W/dave"; git checkout -q -b mr/dave; id=$(addgap "dave gap"); cite M-0001 "Blocked on $id (means:dave-gap)."
  echo "dave (no fetch, no claim) allocated $id"
  show git push -q origin mr/dave | grep -E 'error|exit' )
hostclone; ( cd "$W/hostwork"; for b in alpha dave; do git merge -q --no-ff --no-edit origin/mr/$b; done; git push -q --no-verify origin main )
rm -rf "$W/after"; git clone -q "$W/host.git" "$W/after"; cd "$W/after"
echo "  main gaps: $(ls work/gaps | tr '\n' ' ')"; echo "  ids-unique errors on main: $(aiwf check </dev/null 2>&1 | grep -c 'error ids-unique')"
