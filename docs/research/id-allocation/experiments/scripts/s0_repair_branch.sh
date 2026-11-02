#!/usr/bin/env bash
# S0.5 — a duplicate has landed on main (hosted merge, no check). Which hooked pushes can
# carry a repair, and what does the duplicate do to unrelated pushes meanwhile?
#   fix/dup-g0002  renumber the second clone's copies only (the hint's repair)
#   fix/both       renumber both copies of each id
#   main           the hint's repair made on main itself, pushed with hooks
#   chore/other    an unrelated change on a branch cut from the duplicated main
source "$(dirname "$0")/lib.sh"
bash "$(dirname "$0")/s0_merge.sh" noff nohooks >/dev/null 2>&1
W=$X/runs/s0-noff-nohooks
hooked() { rm -rf "$W/$1"; mkclone "$1" hooks; cd "$W/$1"; }
DB=$'## Question\n\nq.\n\n## Decision\n\nd.\n\n## Reasoning\n\nr.\n'

hooked dave; echo "  main gaps: $(ls work/gaps | tr '\n' ' ')"
echo "== fix/dup-g0002: renumber the second clone's copies"
git checkout -q -b fix/dup-g0002
show aiwf reallocate work/gaps/G-0002-beta-gap.md
show aiwf reallocate work/epics/E-0002-beta-epic/epic.md
echo "  branch gaps: $(ls work/gaps | tr '\n' ' ')"
show git push origin fix/dup-g0002

hooked erin
echo "== fix/both, first-clone copy first: the verb refuses it"
git checkout -q -b fix/both
show aiwf reallocate work/gaps/G-0002-alpha-gap.md | grep -E 'error|findings|exit'
echo "== fix/both: second clone's copies first, then the first clone's"
for p in work/gaps/G-0002-beta-gap.md work/epics/E-0002-beta-epic/epic.md \
         "$(ls work/gaps/G-0002-alpha-gap.md)" work/epics/E-0002-alpha-epic/epic.md; do
  show aiwf reallocate "$p"; done
echo "  branch gaps: $(ls work/gaps | tr '\n' ' ')"
show git push origin fix/both | grep -E 'error|findings|new branch|exit'

hooked frank
echo "== main: the second clone's copies renumbered on main, pushed with hooks"
show aiwf reallocate work/gaps/G-0002-beta-gap.md
show aiwf reallocate work/epics/E-0002-beta-epic/epic.md
show git push origin main | grep -E 'error|findings|exit'

hooked grace
echo "== chore/other: an unrelated decision on a branch cut from the duplicated main"
git checkout -q -b chore/other
show aiwf add decision --title "Unrelated" --body "$DB"
show git push origin chore/other | grep -E 'error|findings|exit'
