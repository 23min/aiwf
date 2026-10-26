#!/usr/bin/env bash
# Design A edge E3: a branch that contains main's tip AND a colliding id (merged main in, no
# reallocate, pushed without hooks) reaches the host rule.
source "$(dirname "$0")/lib.sh"; source "$(dirname "$0")/a_host.sh"
setup "$X/runs/a-edge-check"; mkclone alpha; mkclone beta
for who in alpha beta; do ( cd "$W/$who"; git checkout -q -b mr/$who; addgap "$who gap" >/dev/null; git push -q --no-verify origin mr/$who ); done
echo "  MR mr/alpha -> $(host_mr mr/alpha)"
( cd "$W/beta"; git fetch -q origin; git merge -q --no-edit origin/main; git push -q --no-verify origin mr/beta )
echo "  MR mr/beta (main merged in, no reallocate) -> $(host_mr mr/beta)"
