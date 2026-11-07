#!/usr/bin/env bash
# S10: the review's E2 shape run under both rules with the prototype.
# Unpushed planning commit on main, branch cut from it, then a verb on
# main; merge the branch back.
. "$(dirname "$0")/lib.sh"
new_env s10
GB=$'## What\'s missing\n\nx.\n\n## Why it matters\n\ny.'
run() {
  local rule=$1
  init_repo "$rule"
  export AIWF_EXPERIMENT_AMEND_RULE=$rule
  x aiwf add epic --title "Epic one"
  x git push -q origin main
  x aiwf promote E-0001 active
  x git switch -q -c epic/E-0001-epic-one
  x aiwf add gap --title "on epic branch" --body "$GB"
  x git switch -q main
  x aiwf add gap --title "on main after the cut" --body "$GB"
  lg --all -4
  x git merge -q --no-ff -m "merge epic" epic/E-0001-epic-one
  ops HEAD
  x aiwf history E-0001
  x aiwf check
  x git push origin main
}
section "naive rule"; ( run naive )
section "corrected rule"; ( run corrected )
