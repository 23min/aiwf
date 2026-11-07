#!/usr/bin/env bash
# S08: what today's readers make of amended multi-operation commits.
#  a: two promotes of one entity in one run (proposed -> active -> done):
#     fsm-history-consistent compares each commit with its parent.
#  b: the same, plus a forced op on another entity in the same run.
#  c: a TDD phase ladder (red, green, done, met) in one run, then the code commit.
#  d: aiwf history / show / status / check over the result; JSON history.
# Runs a/b/c twice: control (amend off) and amend.
. "$(dirname "$0")/lib.sh"
new_env s08
ACB=$T/acbody.md; printf 'The criterion holds when a test says so.\n' > "$ACB"
GB=$'## What\'s missing\n\nx.\n\n## Why it matters\n\ny.'

setup() {
  init_repo "$1"
  for e in "Epic one" "Epic two" "Epic three"; do aiwf add epic --title "$e" >/dev/null 2>&1; done
  aiwf add milestone --epic E-0001 --tdd required --title "TDD milestone" >/dev/null 2>&1
  aiwf add ac M-0001 --title "crit one" --body-file "$ACB" >/dev/null 2>&1
  aiwf add gap --title "gap one" --body "$GB" >/dev/null 2>&1
  git push -q origin main >/dev/null 2>&1 || echo "setup push failed"
  aiwf promote E-0001 active >/dev/null 2>&1
  git push -q origin main >/dev/null 2>&1 || echo "setup push failed"
  echo "setup: $(lg -1)"
}

body() {
  local mode=$1
  section "[$mode] a: promote E-0002 active then done in one run"
  x aiwf promote E-0002 active
  x aiwf promote E-0002 done
  lg -2
  x aiwf check
  section "[$mode] b: promote E-0003 active then done, plus a forced op on G-0001"
  x git push -q origin main
  x aiwf promote E-0003 active
  x aiwf promote E-0003 done
  x aiwf promote G-0001 addressed --force --reason "closed out of band"
  lg -1
  x aiwf check
  aiwf check --verbose 2>&1 | grep fsm-history
  x git push origin main

  section "[$mode] c: TDD ladder in one run, then the code commit"
  x git switch -q -c epic/E-0001-epic-one
  x aiwf promote M-0001 in_progress
  x git switch -q -c milestone/M-0001-tdd-milestone
  echo 'package x // test' > x_test.go
  x aiwf promote M-0001/AC-1 --phase red
  echo 'package x // impl' > x.go
  x aiwf promote M-0001/AC-1 --phase green --tests "pass=1 fail=0 skip=0"
  x aiwf promote M-0001/AC-1 --phase done
  x aiwf promote M-0001/AC-1 met
  git add x.go x_test.go; git commit -qm "feat: implement AC-1" >/dev/null 2>&1; echo "\$ git commit (code)"
  lg -3
  x aiwf history M-0001/AC-1
  x aiwf history M-0001/AC-1 --format=json --pretty
  section "[$mode] d: show / status / check"
  x aiwf show M-0001
  x aiwf status
  x aiwf check
}

section "CONTROL"
( unset AIWF_EXPERIMENT_AMEND; setup control; body control )
section "AMEND"
( setup repo; body amend )
