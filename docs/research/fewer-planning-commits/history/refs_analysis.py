"""Q3/Q5: which SHA-naming records would a fold (or an amend) have broken?

Usage: python3 -I refs_analysis.py <fold_dump.json> <amend_dump.json> [--list CATEGORY]

A record is (R, S): commit R wrote a name for commit S. File records are
deduplicated to the earliest commit that added the token to that file
(archive moves and body rewrites re-add lines); a message token that repeats
the same commit's trailer value is folded into the trailer record.
For each record:
  written-while-unpushed: S is a window commit and origin/main reached S after R's commit time;
  fold breaks (F1/F2): the fold re-creates S, and the fold runs after R was written;
  amend breaks: S is a kernel commit whose amend-run gained a later operation
    after R was written (the SHA R names is replaced by that amend).
"""
import collections
import json
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from base import C, D, HERE, LINE, PUSHTIME, REPO, klass, verbs  # noqa: E402
import subprocess  # noqa: E402

fold = json.load(open(sys.argv[1]))
amend = json.load(open(sys.argv[2]))
LIST = sys.argv[sys.argv.index("--list") + 1] if "--list" in sys.argv else None
RUN = amend["run"]
RUNMEM = collections.defaultdict(list)
for h, r in RUN.items():
    RUNMEM[r].append(h)

refs = json.load(open(os.path.join(HERE, "refs.json")))


def category(r):
    p = r["key_or_path"] or ""
    if r["where"] == "trailer":
        return {"aiwf-force-for": "acknowledge illegal (aiwf-force-for)"}.get(p, "trailer " + p)
    if r["where"] == "message":
        return "commit message text"
    if p.startswith("work/gaps/") and re.fullmatch(r"-\s*[0-9a-f]{7,40}", r["line"]) and "promote" in verbs(r["rec"]):
        return "gap addressed_by_commit (--by-commit)"
    if p.startswith("work/"):
        return "entity body under work/"
    if p == ".gitleaksignore":
        return ".gitleaksignore fingerprint"
    if p.startswith("internal/") or p == "Makefile":
        return "code / test source"
    return "docs and other files"


def ident(r):
    if r["where"] != "file":
        return ("msg", r["rec"])
    return ("file", re.sub(r"(^|/)archive/", r"\1", r["key_or_path"]))


best = {}
for r in refs:
    cat = category(r)
    key = (ident(r), r["sha"], cat if r["where"] != "message" else "msg")
    if r["where"] == "message" and any(x["rec"] == r["rec"] and x["sha"] == r["sha"] and x["where"] == "trailer" for x in refs):
        continue
    if key not in best or C[r["rec"]]["ct"] < C[best[key]["rec"]]["ct"]:
        best[key] = dict(r, cat=cat)


def anc(a, b):
    return subprocess.run(["git", "-C", REPO, "merge-base", "--is-ancestor", a, b]).returncode == 0


rows = collections.defaultdict(collections.Counter)
listing = []
for (idn, s, _), r in sorted(best.items(), key=lambda kv: C[kv[1]["rec"]]["ct"]):
    cat, R = r["cat"], r["rec"]
    t = C[R]["ct"]
    if s == R:
        continue
    if s in C:
        status = "window"
    elif anc(s, D["base"]):
        status = "pre-window (pushed)"
    else:
        status = "not reachable from HEAD"
    c = rows[cat]
    c["records"] += 1
    c["S " + status] += 1
    unpushed = s in C and PUSHTIME[s] > t
    if not unpushed:
        continue
    c["written while S unpushed"] += 1
    c["  S class " + klass(s)] += 1
    c["  S same line as R" if LINE.get(s) == LINE.get(R) else "  S on another line than R"] += 1
    f1 = s in fold["rewrite"]["F1"] and fold["rewrite"]["F1"][s] > t
    f2 = s in fold["rewrite"]["F2"] and fold["rewrite"]["F2"][s] > t
    am = s in RUN and any(C[m]["ct"] > t and m != R for m in RUNMEM[RUN[s]])
    c["  fold F1 breaks"] += f1
    c["  fold F2 breaks"] += f2
    c["  amend breaks"] += am
    if LIST and cat.startswith(LIST):
        listing.append((R[:9], klass(R), s[:9], klass(s), "F1" if f1 else "", "F2" if f2 else "", "AM" if am else "", r["line"][:110]))

for cat, c in rows.items():
    print(cat)
    for k in ["records", "S window", "S pre-window (pushed)", "S not reachable from HEAD", "written while S unpushed",
              "  S class kernel", "  S class code", "  S class ritual", "  S class merge",
              "  S same line as R", "  S on another line than R",
              "  fold F1 breaks", "  fold F2 breaks", "  amend breaks"]:
        if c.get(k):
            print(f"   {c[k]:5d}  {k}")
for row in listing:
    print(*row, sep=" | ")
