"""List every aiwf operation reachable from a rev and flag duplicates.

Usage: git log --format='%H%x1f%aI%x1f%(trailers:only,unfold)%x1e' REV | python3 -I ops.py

An operation is one trailer group (split at each aiwf-verb line). Its
time is its aiwf-op-time, or the commit's author date when the commit
carries a single operation. The same operation reached twice (e.g. once
as the original commit and once inside an amended copy) has the same
(verb, entity, to, scope, actor, time) key.
"""
import collections
import sys

ops = []
for rec in sys.stdin.read().split("\x1e"):
    rec = rec.strip()
    if not rec:
        continue
    sha, adate, trailers = (rec.split("\x1f") + ["", ""])[:3]
    groups = []
    for line in trailers.splitlines():
        if ":" not in line:
            continue
        k, v = [s.strip() for s in line.split(":", 1)]
        if k == "aiwf-verb":
            groups.append({"verb": v})
        elif groups:
            groups[-1].setdefault(k, v)
    for g in groups:
        key = (g.get("verb"), g.get("aiwf-entity", "-"), g.get("aiwf-to", "-"),
               g.get("aiwf-scope", "-"), g.get("aiwf-actor"), g.get("aiwf-op-time", adate))
        ops.append((key, sha[:7]))

count = collections.defaultdict(list)
for key, sha in ops:
    count[key].append(sha)
print(f"operations reachable: {len(ops)}  distinct: {len(count)}")
dups = {k: v for k, v in count.items() if len(v) > 1}
for k, shas in dups.items():
    print("DUPLICATE", k, "in commits", shas)
if not dups:
    print("no duplicate operations")
