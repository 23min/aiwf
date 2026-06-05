#!/usr/bin/env bash
# stamp-cellid.sh <file> <prefix>
# Walks <file> line-by-line; for each line matching `^[\t ]*Name:`,
# inserts a `CellID: "branch-cell-<prefix>-c<N>"` line above with the
# same indentation. N increments per scenario.

set -e
file="$1"
prefix="$2"
[ -z "$file" ] || [ -z "$prefix" ] && { echo "usage: $0 <file> <prefix>"; exit 1; }

python3 - <<PY
import re
with open("$file", "r", encoding="utf-8") as f:
    src = f.read()
lines = src.splitlines(keepends=True)
out = []
count = 0
name_re = re.compile(r'^(\s*)Name:\s+(.*)\n$')
for line in lines:
    m = name_re.match(line)
    if m:
        count += 1
        indent = m.group(1)
        rest = m.group(2)
        out.append(f'{indent}CellID: "branch-cell-$prefix-c{count}",\n')
        out.append(f'{indent}Name:   {rest}\n')
    else:
        out.append(line)
with open("$file", "w", encoding="utf-8") as f:
    f.write(''.join(out))
print(f"stamped {count} CellIDs in {'$file'}")
PY
