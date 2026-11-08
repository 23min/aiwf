#!/usr/bin/env bash
# Probes for Experiment 1, run in <run dir>/a.   m1.sh <run dir>
set -uo pipefail
. "$(dirname "$0")/lib.sh" "$1"
cd "$W/a"; . "$W/shas.env"
echo "### first-parent line over base (sha, %G?, subject)"; show_line "$BASE"
echo "### aiwf check (full)"; aiwf check 2>&1 | tail -n 25
echo "### aiwf check --since $(git rev-parse --short "$BASE")"; aiwf check --since "$BASE" 2>&1 | tail -n 25
echo "### addressed_by_commit in G-0001 and whether HEAD contains it"
abc=$(grep -h -A2 'addressed_by_commit' work/gaps/G-0001-*.md | grep -oE '[0-9a-f]{40}' | head -1)
echo "addressed_by_commit: $abc"
git merge-base --is-ancestor "$abc" HEAD && echo "  contained in HEAD: yes" || echo "  contained in HEAD: NO"
echo "### aiwf-force-for target of the acknowledgment and whether HEAD contains it"
for t in $(git log --format='%(trailers:key=aiwf-force-for,valueonly)' "$BASE..HEAD" | grep -oE '[0-9a-f]{7,40}'); do
  printf '%s ' "$t"; git merge-base --is-ancestor "$t" HEAD 2>/dev/null && echo "contained: yes" || echo "contained: NO"; done
echo "### aiwf-authorized-by targets and whether HEAD contains them"
for t in $(git log --format='%(trailers:key=aiwf-authorized-by,valueonly)' "$BASE..HEAD" | grep -oE '[0-9a-f]{7,40}'); do
  printf '%s ' "$t"; git merge-base --is-ancestor "$t" HEAD 2>/dev/null && echo "contained: yes" || echo "contained: NO"; done
echo "### SHAs named in commit subjects/bodies on the line, and whether HEAD contains them"
git log --format=%B "$BASE..HEAD" | grep -oE '\b[0-9a-f]{40}\b' | sort -u | while read -r t; do
  printf '%s ' "${t:0:12}"; git merge-base --is-ancestor "$t" HEAD 2>/dev/null && echo "contained: yes" || echo "contained: NO"; done
for id in G-0001 M-0001 E-0001; do echo "### aiwf history $id"; aiwf history "$id" 2>&1; done
