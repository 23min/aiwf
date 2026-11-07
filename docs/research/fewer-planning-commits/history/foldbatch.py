"""Candidates 1 (fold before push/merge) and 2 (batch at write time) replayed over the window.

Usage: python3 -I foldbatch.py [--drop-roadmap] [--verified-forks] [--dump FILE]

Segments: main's first-parent line cut at every origin/main reflog entry (push
or fetch) and at every fast-forward merge onto main (those commits were made on
a branch whose fold runs before it merges); every branch line (one merge's
second-parent first-parent chain) is one segment ending at that merge.

A *fork* is a commit with a child on another line: a branch cut from it, or a
merge that takes it as a second parent (e.g. main merged into a branch). The
merge that ends the commit's own line is not a fork.

fold F1 (the review's model): a fork closes the current group (the operator
  folds before cutting); merges on the line are re-created, not barriers.
fold F2: fold runs only at push/merge; a fork freezes it and everything before
  it in the segment, so planning commits up to the last fork stay one each.
batch (the review's model): the pending batch commits at every push, at every
  merge on the line, after every fork, and at the line's end.
Ritual commits are plain commits: kept one each, never folded/batched, not barriers.
"""
import collections
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from base import (C, CHILDREN, FF_EVENTS, FF_ON_MAIN, LINE, LINEMERGE, LINES, PUSHES, PUSHIDX,  # noqa: E402
                  is_merge, klass, roadmap_only)

args = sys.argv[1:]
DROP = "--drop-roadmap" in args
DUMP = args[args.index("--dump") + 1] if "--dump" in args else None
VERIFIED_FORKS = "--verified-forks" in args   # count a fork only if a child on another line predates the fold


def dropped(h):
    return DROP and roadmap_only(h) and klass(h) == "ritual"


def eff(h):
    while h in C and dropped(h):
        h = C[h]["parents"][0]
    return h


FORK_TIME = collections.defaultdict(list)   # commit -> creation times of children on other lines
for p in C:
    for ch in CHILDREN.get(p, []):
        if dropped(ch):
            continue
        ps = C[ch]["parents"]
        if ps[0] == p and LINE[ch] == LINE[p]:
            continue
        if is_merge(ch) and ch == LINEMERGE.get(LINE[p]) and LINES[LINE[p]][-1] == p:
            continue                         # the merge that ends p's own line
        FORK_TIME[eff(p)].append(C[ch]["ct"])
ALL_FORKS = set(FORK_TIME)

FFIDX = {}
for i, (o, n, t, m, rng) in enumerate(FF_EVENTS):
    for h in rng:
        FFIDX[h] = (i, t)

segments = []        # (label, [commits oldest first], fold_time)
cur_key, cur = None, []
for h in LINES["main"]:
    key = ("ff", FFIDX[h][0]) if h in FFIDX else ("push", PUSHIDX[h])
    if key != cur_key and cur:
        segments.append((cur_key, cur))
        cur = []
    cur_key = key
    cur.append(h)
if cur:
    segments.append((cur_key, cur))
segs = []
for key, shas in segments:
    t = FF_EVENTS[key[1]][2] if key[0] == "ff" else PUSHES[key[1]][1]
    segs.append(("main:" + key[0], shas, t))
for lid, shas in LINES.items():
    if lid != "main":
        segs.append(("branch", shas, C[LINEMERGE[lid]]["ct"]))
assert sum(len(s) for _, s, _ in segs) == len(C)


def groups_f1(shas, FORK):
    out, g = [], []
    for h in shas:
        g.append(h)
        if h in FORK:
            out.append(g)
            g = []
    if g:
        out.append(g)
    return out


def rewritten(group):
    """Commits a fold of this group re-creates: from its first kernel commit on,
    unless that is the group's only kernel commit and already last."""
    ks = [i for i, h in enumerate(group) if klass(h) == "kernel"]
    if not ks or (len(ks) == 1 and ks[0] == len(group) - 1):
        return []
    return group[ks[0]:]


res = collections.Counter()
FORKSTAT = collections.Counter()
REWRITE = {"F1": {}, "F2": {}}    # commit -> fold time
KEPT_RITUAL = 0
for label, shas, t in segs:
    shas = [h for h in shas if not dropped(h)]
    FORK = {h for h in shas if h in ALL_FORKS and (not VERIFIED_FORKS or min(FORK_TIME[h]) < t)}
    for h in shas:
        if h in ALL_FORKS:
            FORKSTAT["verified (a child on another line predates the fold)" if min(FORK_TIME[h]) < t else "unknown (every such child postdates the fold)"] += 1
    where = "main" if label.startswith("main") else "branch"
    KEPT_RITUAL_SEG = sum(1 for h in shas if klass(h) == "ritual")
    KEPT_RITUAL += KEPT_RITUAL_SEG
    # F1
    for g in groups_f1(shas, FORK):
        if any(klass(h) == "kernel" for h in g):
            res[("F1", where)] += 1
        ft = min(FORK_TIME[g[-1]]) if g[-1] in FORK else t
        for h in rewritten(g):
            REWRITE["F1"][h] = ft
    # F2
    last = max([i for i, h in enumerate(shas) if h in FORK], default=-1)
    frozen, tail = shas[:last + 1], shas[last + 1:]
    res[("F2", where)] += sum(1 for h in frozen if klass(h) == "kernel")
    if any(klass(h) == "kernel" for h in tail):
        res[("F2", where)] += 1
    for h in rewritten(tail):
        REWRITE["F2"][h] = t
    # batch
    open_ = False
    for h in shas:
        if is_merge(h):
            open_ = False
            continue
        if klass(h) == "kernel" and not open_:
            res[("batch", where)] += 1
            open_ = True
        if h in FORK:
            open_ = False
    # ritual-folds variants (rituals treated like kernel ops)
    open_ = False
    for h in shas:
        if klass(h) in ("kernel", "ritual") and not open_:
            res[("F1+rituals", where)] += 1
            open_ = True
        if h in FORK:
            open_ = False

kernel = sum(1 for h in C if klass(h) == "kernel")
print("variant:", "drop ROADMAP-only ritual commits" if DROP else "as-is", "| forks: verified only" if VERIFIED_FORKS else "| forks: any time (review)")
print("fork timing:", dict(FORKSTAT))
print("segments:", collections.Counter(l for l, _, _ in segs), " forks:", len(ALL_FORKS))
print("kernel verb commits:", kernel, " ritual commits kept:", KEPT_RITUAL)
for m in ("F1", "F2", "batch"):
    mm, bb = res[(m, "main")], res[(m, "branch")]
    print(f"{m:6s}: kernel groups main={mm} branch={bb}  + ritual {KEPT_RITUAL}  => PLANNING COMMITS REMAINING {mm + bb + KEPT_RITUAL}")
print(f"F1 if ritual commits could fold too: {res[('F1+rituals', 'main')] + res[('F1+rituals', 'branch')]}")
print("commits re-created by the fold: F1", collections.Counter(klass(h) for h in REWRITE["F1"]),
      " F2", collections.Counter(klass(h) for h in REWRITE["F2"]))
if DUMP:
    json.dump(dict(rewrite=REWRITE, fork={h: v for h, v in FORK_TIME.items()}, res={"/".join(k): v for k, v in res.items()},
                   segs=[(l, s, t) for l, s, t in segs]), open(DUMP, "w"))
