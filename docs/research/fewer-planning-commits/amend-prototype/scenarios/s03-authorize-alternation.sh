#!/usr/bin/env bash
# S03: human and agent alternating under aiwf authorize. Runs the same
# body twice: control (AIWF_EXPERIMENT_AMEND unset) and amend (corrected
# rule). Parts 3a/3b also run a variant under AIWF_EXPERIMENT_AMEND_RULE=naive
# to show what the names-HEAD check prevents.
#  part 1: start-milestone ritual (promote + authorize on the epic branch,
#          cut the milestone branch, agent acts).
#  part 2: authorize alone; agent, agent, human pause, agent (refused),
#          human resume, agent, human --end.
#  part 3a: authorize then --end in the same run.
#  part 3b: authorize then a terminal cancel in the same run.
#  part 4: authorize, then another human op amends after it, then agent.
. "$(dirname "$0")/lib.sh"
new_env s03
AG=(--actor ai/claude --principal human/peter)
ACB=$T/acbody.md; printf 'The criterion holds when a test says so.\n' > "$ACB"

codecommit() { echo "$1" >> notes.txt; git add notes.txt; git commit -qm "chore: $1" >/dev/null 2>&1; echo "\$ git commit (code): chore: $1"; }

setup() {
  init_repo "$1"
  aiwf add epic --title "Epic one" >/dev/null 2>&1
  for n in 1 2 3 4 5 6 7; do
    aiwf add milestone --epic E-0001 --tdd none --title "Milestone $n" >/dev/null 2>&1
    aiwf add ac M-000$n --title "crit $n" --body-file "$ACB" >/dev/null 2>&1
  done
  aiwf promote M-0005/AC-1 cancelled >/dev/null 2>&1   # so M-0005/M-0006 can be cancelled in part 3b
  aiwf promote M-0006/AC-1 cancelled >/dev/null 2>&1
  git push -q origin main >/dev/null 2>&1 || echo "setup: push failed"
  aiwf promote E-0001 active >/dev/null 2>&1
  git push -q origin main >/dev/null 2>&1 || echo "setup: push failed"
  git switch -q -c epic/E-0001-epic-one
  git push -q -u origin epic/E-0001-epic-one >/dev/null 2>&1 || echo "setup: push failed"
  echo "setup done: $(lg -1)"
}

scopes_of() { echo "--- aiwf show $1 (scopes section):"; aiwf show "$1" | sed -n '/Scopes/,/^$/p'; echo "--- aiwf history $1 (last 2 events):"; aiwf history "$1" | tail -6; }

body() {
  local mode=$1
  section "[$mode] part 1: start-milestone ritual"
  x aiwf promote M-0001 in_progress
  x aiwf authorize M-0001 --to ai/claude --branch milestone/M-0001-milestone-1 --reason "delegate M-0001"
  lg -3
  x git switch -q -c milestone/M-0001-milestone-1
  x aiwf promote M-0001/AC-1 met "${AG[@]}"
  scopes_of M-0001
  x git switch -q epic/E-0001-epic-one

  section "[$mode] part 2: authorize alone, then alternation"
  codecommit part2
  x aiwf authorize M-0002 --to ai/claude --branch milestone/M-0002-milestone-2 --reason "delegate M-0002"
  x git switch -q -c milestone/M-0002-milestone-2
  x aiwf add ac M-0002 --title "agent crit a" "${AG[@]}"
  x aiwf add ac M-0002 --title "agent crit b" "${AG[@]}"
  x aiwf authorize M-0002 --pause "waiting on review"
  x aiwf add ac M-0002 --title "agent crit while paused" "${AG[@]}"
  x aiwf authorize M-0002 --resume "review done"
  echo "--- HEAD message after resume:"; git log -1 --format=%B | sed -n '1,40p'
  x aiwf add ac M-0002 --title "agent crit after resume" "${AG[@]}"
  x aiwf authorize M-0002 --end --reason "taking it back"
  x aiwf add ac M-0002 --title "agent crit after end" "${AG[@]}"
  lg -6
  scopes_of M-0002
  x git switch -q epic/E-0001-epic-one

  section "[$mode] part 3a: authorize then --end in the same run"
  codecommit part3a
  x aiwf authorize M-0003 --to ai/claude --branch milestone/M-0003-milestone-3 --reason "delegate M-0003"
  x aiwf authorize M-0003 --end --reason "changed my mind"
  lg -3
  scopes_of M-0003
  x git switch -q -c milestone/M-0003-milestone-3
  x aiwf promote M-0003/AC-1 met "${AG[@]}"
  x git switch -q epic/E-0001-epic-one

  section "[$mode] part 3b: authorize then terminal cancel in the same run"
  codecommit part3b
  x aiwf authorize M-0005 --to ai/claude --branch milestone/M-0005-milestone-5 --reason "delegate M-0005"
  x aiwf cancel M-0005 --reason "dropped"
  lg -3
  scopes_of M-0005

  if [ "$mode" = amend ]; then
    section "[naive rule] part 3a: authorize then --end in the same run"
    codecommit part3a-naive
    AIWF_EXPERIMENT_AMEND_RULE=naive x aiwf authorize M-0004 --to ai/claude --branch milestone/M-0004-milestone-4 --reason "delegate M-0004"
    AIWF_EXPERIMENT_AMEND_RULE=naive x aiwf authorize M-0004 --end --reason "changed my mind"
    lg -2
    git log -1 --format='%(trailers:only,unfold)'
    echo "HEAD is $(git rev-parse HEAD)"
    scopes_of M-0004
    x git switch -q -c milestone/M-0004-milestone-4
    x aiwf promote M-0004/AC-1 met "${AG[@]}"
    x git switch -q epic/E-0001-epic-one

    section "[naive rule] part 3b: authorize then terminal cancel in the same run"
    codecommit part3b-naive
    AIWF_EXPERIMENT_AMEND_RULE=naive x aiwf authorize M-0006 --to ai/claude --branch milestone/M-0006-milestone-6 --reason "delegate M-0006"
    AIWF_EXPERIMENT_AMEND_RULE=naive x aiwf cancel M-0006 --reason "dropped"
    lg -2
    scopes_of M-0006
  fi

  section "[$mode] part 4: authorize, then another human op after it"
  codecommit part4
  x aiwf authorize M-0007 --to ai/claude --branch milestone/M-0007-milestone-7 --reason "delegate M-0007"
  x aiwf retitle M-0007 "Milestone 7, retitled"
  lg -2
  x git switch -q -c milestone/M-0007-milestone-7
  x aiwf promote M-0007/AC-1 met "${AG[@]}"
  scopes_of M-0007
  x git switch -q epic/E-0001-epic-one

  section "[$mode] aiwf check, push of the epic branch"
  x aiwf check
  aiwf check --verbose 2>&1 | grep -v 'entity-body-empty\|milestone-draft-incomplete\|epic-active-no-drafted' | grep -E 'warning|error'
  x git push origin epic/E-0001-epic-one
  ops HEAD
}

section "CONTROL: AIWF_EXPERIMENT_AMEND unset"
( unset AIWF_EXPERIMENT_AMEND; setup control; body control )
section "AMEND: AIWF_EXPERIMENT_AMEND=1, corrected rule"
( setup repo; body amend )
