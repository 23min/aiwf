"""Q4/Q5: merges that pull a commit from another line into a branch (e.g. main merged
into a patch branch) while that commit was not yet pushed -- the same fork shape as a cut.

Usage: python3 -I syncmerges.py
"""
import collections
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from base import C, LINE, LINEMERGE, PUSHTIME, is_merge, klass  # noqa: E402

c = collections.Counter()
for h in C:
    if not is_merge(h) or LINE[h] == "main":
        continue
    p2 = C[h]["parents"][1]
    if LINEMERGE.get(LINE.get(p2)) == h:
        continue              # an ordinary branch merge (milestone into epic): not a sync
    src = "main" if LINE.get(p2) == "main" else ("pre-window" if p2 not in C else "branch line")
    unp = p2 in C and PUSHTIME[p2] > C[h]["ct"]
    c[(src, "unpushed at merge" if unp else "pushed at merge", klass(p2) if p2 in C else "pre-window")] += 1
for k, v in sorted(c.items()):
    print(f"{v:4d}  {' | '.join(k)}")
print("total sync merges:", sum(c.values()))
