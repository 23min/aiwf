#!/usr/bin/env bash
# Design D — ids taken from merge-request numbers, approximated locally: an entity is filed with
# aiwf add, and once "MR !57" exists its id is changed to G-0057 by hand (aiwf has no verb that
# renumbers to a chosen id). Then: what does aiwf say, and how does the sequential allocator interact?
source "$(dirname "$0")/lib.sh"
setup "$X/runs/d-mr"; mkclone alpha hooks; mkclone bob hooks
cd "$W/alpha"; git checkout -q -b mr/57
g=$(addgap "alpha gap"); cite G-0001 "See $g (means:alpha-gap)."; git push -q -u origin mr/57 >/dev/null 2>&1
echo "  alpha filed $g, opened MR !57; renumbering by hand to G-0057"
show aiwf reallocate --help | grep -E 'Usage|reallocate <'
git mv "work/gaps/$g-alpha-gap.md" work/gaps/G-0057-alpha-gap.md
sed -i "s/^id: $g\$/id: G-0057/" work/gaps/G-0057-alpha-gap.md
sed -i "s/$g (means/G-0057 (means/" work/gaps/G-0001-seed-gap.md
git add -A; show git commit -q -m "chore: renumber $g to MR number G-0057" | grep -E 'error|exit'
show git push -q origin mr/57 | grep -E 'error|provenance|exit' | cut -c1-200
verify
echo "  (pushing the renumber with --no-verify, standing in for a verb that would carry trailers)"
git push -q --no-verify origin mr/57
hostclone; ( cd "$W/hostwork"; git merge -q --no-ff --no-edit origin/mr/57; git push -q --no-verify origin main )
echo "== bob allocates sequentially after MR !57 landed"
cd "$W/bob"; git fetch -q origin; git checkout -q -b feature/bob origin/main
echo "  bob's aiwf add gap allocates: $(addgap "bob gap")"
show aiwf history G-0057
