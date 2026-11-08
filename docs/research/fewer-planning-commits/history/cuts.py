"""Q4: branches cut from a commit that was not yet pushed, and whether that commit was a planning commit at the tip.

Usage: python3 -I cuts.py [--list]

A cut is a branch line (a merge's second-parent first-parent chain, or a
fast-forwarded branch on main's line) whose base X (the first parent of the
line's oldest commit F) is not the merged tip of an earlier line of the same
branch. The cut happened at an unknown time in [X became available, ct(F)].
  unpushed at cut, verified:  origin/main first reached X after ct(F);
  unpushed at cut, possible:  origin/main reached X between ct(X) and ct(F);
  pushed:                     X predates the window (pushed before it began).
  X at the tip, verified:     X's line got no successor before ct(F) (main: from main's reflog;
                              a branch line: the next commit on that line);
  X at the tip, possible:     X's line moved on before ct(F), so the cut may have been earlier.
"""
import collections
import datetime
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from base import C, D, FF_EVENTS, LINE, LINEMERGE, LINES, MAINLOG, PUSHTIME, klass, line_base  # noqa: E402

LIST = "--list" in sys.argv


def ts(t):
    return datetime.datetime.fromtimestamp(t, datetime.UTC).strftime("%Y-%m-%d %H:%M")


def moved_off(x):
    """Time x's own line got its next commit (main: main's reflog leaving x)."""
    if LINE.get(x) == "main":
        ts_ = [t for o, n, t, m in MAINLOG if o == x and n != x]
        return min(ts_) if ts_ else None
    if x not in C:
        return None
    ln = LINES[LINE[x]]
    i = ln.index(x)
    return C[ln[i + 1]]["ct"] if i + 1 < len(ln) else None


cuts = []
for lid, shas in LINES.items():
    if lid == "main":
        continue
    x = line_base(lid)
    xl = LINE.get(x)
    if xl and xl != "main" and LINES[xl][-1] == x and LINEMERGE.get(xl):
        continue      # continuation of a branch already merged once
    cuts.append((lid, x, shas[0]))
for o, n, t, m, rng in FF_EVENTS:
    if m.startswith("merge a100c7c64"):
        first = rng[-1]
        cuts.append(("ff:" + m.split(":")[0][6:], C[first]["parents"][0], first))

res = collections.Counter()
rows = []
for lid, x, f in cuts:
    tf = C[f]["ct"]
    if x not in C:
        push = "pushed (X predates the window)"
    elif PUSHTIME[x] > tf:
        push = "unpushed at cut, verified"
    else:
        push = "unpushed at cut, possible"
    kx = klass(x) if x in C else "pre-window"
    mo = moved_off(x)
    tip = "tip, verified" if (mo is None or mo >= tf) else "tip, possible"
    parent = "main" if LINE.get(x) == "main" else ("pre-window" if x not in C else "branch line")
    res[("all", push)] += 1
    res[(push, kx)] += 1
    res[(push, kx, tip)] += 1
    res[(push, "parent " + parent)] += 1
    rows.append((lid, parent, x[:9], kx, push, tip, C[x]["subject"][:60] if x in C else "", ts(tf)))

print("cuts:", len(cuts))
for k in sorted(res, key=str):
    print(f"  {res[k]:4d}  {' | '.join(k)}")
if LIST:
    for r in rows:
        print(*r, sep=" | ")
