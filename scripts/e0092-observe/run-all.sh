#!/usr/bin/env bash
# run-all.sh <checkout-sha> [parallel]
#
# The rubric's full matrix: every task three times per host, the T1
# continuation three times per host, then the judge over every complete run.
# Runs `parallel` at a time (default 4); a run already holding events is
# skipped, so an interrupted matrix resumes where it stopped.
set -uo pipefail
SHA=$1 P=${2:-4}
H=$(dirname "$(readlink -f "$0")")
export OBSERVE_DIR=${OBSERVE_DIR:-${TMPDIR:-/tmp}/e0092-observe}
mkdir -p "$OBSERVE_DIR"
jobs=()
for task in T1 T2 T3 T4 T5 T6 T7 T8 T9 T10 T11 L1 L2; do
  for host in claude codex; do for n in 1 2 3; do jobs+=("$host $task $n"); done; done
done
for host in claude codex; do for n in 1 2 3; do jobs+=("$host T1 $n cont"); done; done
printf '%s\n' "${jobs[@]}" | while read -r host task n cont; do
  dir="$OBSERVE_DIR/$task${cont:+-cont}/$host-$n"
  [ -s "$dir/events.jsonl" ] && [ -s "$dir/transcript.txt" ] && continue
  echo "$host $task $n${cont:+ $cont}"
done | xargs -P "$P" -L 1 bash -c '"$0/run.sh" "$1" "$2" "$3" "$4" ${5:-}' "$H" "$SHA" >> "$OBSERVE_DIR/run-all.log" 2>&1
for dir in "$OBSERVE_DIR"/*/*-[0-9]; do
  [ -s "$dir/verdicts.json" ] && continue
  grep -q '^INCOMPLETE' "$dir/after.txt" 2>/dev/null && continue
  echo "$dir"
done | xargs -P "$P" -I{} "$H/judge.py" {} >> "$OBSERVE_DIR/judge.log" 2>&1
echo "done: $(ls -d "$OBSERVE_DIR"/*/*-[0-9] | wc -l) runs, $(ls "$OBSERVE_DIR"/*/*-[0-9]/verdicts.json 2>/dev/null | wc -l) judged"
