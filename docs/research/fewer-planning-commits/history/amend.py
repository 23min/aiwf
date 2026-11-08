"""Candidate 3 (coalesce into the unpushed tip) replayed over the window.

Usage: python3 -I amend.py [--drop-roadmap] [--drop-roadmap-unstamped] [--cut-unknown-blocks]
                           [--no-ref-check] [--no-sha-check] [--dump FILE]
(--no-ref-check / --no-sha-check drop those two conditions, to show what each one costs)

Eligibility (review's corrected rule), evaluated for every kernel verb commit V
against its HEAD P (V's first parent, skipping commits dropped by the variant):
  - P is a non-merge commit whose operations are all aiwf verb ops (kernel
    class: carries aiwf-verb, not a ritual verb), same aiwf-actor/aiwf-principal;
  - no remote-tracking ref contains P at V's commit time (origin/main reflog);
  - no ref other than V's branch contains P: main (exact, from main's reflog),
    a branch line cut from P / a merge taking P as second parent (child commit
    on another line; 'verified' when that child predates V, 'unknown' when it
    does not -- counted as no-block unless --cut-unknown-blocks), the parent
    branch when V is the first commit of its branch;
  - nothing V writes (trailers, message, added file lines) names the SHA of a
    commit already in P's run (all of which would be HEAD after the amends).
A git cherry-pick is not a verb run and always makes its own commit.
"""
import collections
import json
import os
import subprocess
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from base import (C, CHILDREN, FF_EVENTS, FF_ON_MAIN, HERE, LINE, LINEMERGE, LINES, MADE_ON_MAIN,  # noqa: E402
                  MAINLOG, ORDER, PUSHTIME, REPO, is_merge, klass, line_base, pushed_at, roadmap_only, trailers)

args = sys.argv[1:]
DROP_RM = "--drop-roadmap" in args
DROP_RM_UNSTAMPED = "--drop-roadmap-unstamped" in args
CUT_UNKNOWN_BLOCKS = "--cut-unknown-blocks" in args
DUMP = args[args.index("--dump") + 1] if "--dump" in args else None
NO_REF = "--no-ref-check" in args
NO_SHA = "--no-sha-check" in args
REFCHECK = ("first commit on a branch", "main contains HEAD", "HEAD came by fast-forward", "branch cut from HEAD")


def dropped(h):
    if not roadmap_only(h):
        return False
    k = klass(h)
    return (DROP_RM and k == "ritual") or (DROP_RM_UNSTAMPED and k == "code")


def eff_parent(h):
    p = C[h]["parents"][0] if C[h]["parents"] else None
    while p in C and dropped(p):
        p = C[p]["parents"][0]
    return p


_anc = {}


def is_anc(a, b):
    k = (a, b)
    if k not in _anc:
        _anc[k] = subprocess.run(["git", "-C", REPO, "merge-base", "--is-ancestor", a, b]).returncode == 0
    return _anc[k]


ML = sorted(MAINLOG, key=lambda r: r[2])


def main_at(t):
    cur = None
    for o, n, tt, m in ML:
        if tt <= t:
            cur = n
        else:
            break
    return cur


# branch identity: a line whose base is the merged tip of an earlier line continues that branch
GROUP = {"main": "main"}
for lid in LINES:
    if lid == "main":
        continue
    b = line_base(lid)
    bl = LINE.get(b)
    if bl and bl != "main" and LINES[bl][-1] == b:
        GROUP[lid] = GROUP.get(bl, bl)
    else:
        GROUP[lid] = lid
FFGROUP = {}
for i, (o, n, t, m, rng) in enumerate(FF_EVENTS):
    name = m[len("merge "):].split(":")[0]
    for h in rng:
        FFGROUP[h] = "ff:" + name


def branch(h):
    if h in FFGROUP:
        return FFGROUP[h]
    if h in MADE_ON_MAIN:
        return "main"
    return GROUP[LINE[h]]


def actor(h):
    return (tuple(trailers(h, "aiwf-actor")), tuple(trailers(h, "aiwf-principal")))


REFS = collections.defaultdict(set)
for r in json.load(open(os.path.join(HERE, "refs.json"))):
    REFS[r["rec"]].add(r["sha"])


def other_children(p):
    """Children of p that put p inside a ref other than p's own branch line."""
    out = []
    for ch in CHILDREN.get(p, []):
        if dropped(ch):
            continue
        ps = C[ch]["parents"]
        if ps[0] == p and LINE[ch] == LINE[p]:
            continue                       # p's own successor on its line
        kind = "merge-2nd-parent" if (is_merge(ch) and ps[0] != p) else "branch-cut"
        out.append((ch, kind))
    return out


RUN = {}
starts = []          # (V, reasons list)
amended = []
for v in ORDER:
    if dropped(v):
        continue
    if klass(v) != "kernel":
        continue
    t = C[v]["ct"]
    p = eff_parent(v)
    rs = []
    if MADE_ON_MAIN.get(v, "").startswith("cherry-pick"):
        rs.append("cherry-pick (git, not a verb run)")
    if p not in C:
        rs.append("window start")
    elif klass(p) == "merge":
        rs.append("merge")
    elif klass(p) == "code":
        rs.append("code commit")
    elif klass(p) == "ritual":
        rs.append("ritual commit")
    else:
        if pushed_at(p, t):
            rs.append("push boundary")
        if branch(p) != branch(v):
            rs.append("first commit on a branch (HEAD also on the parent branch)")
        elif branch(v) != "main":
            m = main_at(t)
            if m and is_anc(p, m):
                rs.append("main contains HEAD")
        if branch(v) == "main" and p in FF_ON_MAIN:
            rs.append("HEAD came by fast-forward (source branch contains it)")
        for ch, kind in other_children(p):
            if ch == v:
                continue
            if C[ch]["ct"] < t:
                rs.append(f"branch cut from HEAD ({kind}, before V)")
            else:
                rs.append(f"branch cut from HEAD ({kind}, time unknown)")
        if actor(p) != actor(v):
            rs.append("actor switch")
        run = RUN[p]
        members = {h for h, r in RUN.items() if r == run}
        if REFS.get(v, set()) & members:
            rs.append("operation names HEAD's SHA")
    rs = sorted(set(rs))
    blocking = [r for r in rs if (CUT_UNKNOWN_BLOCKS or "time unknown" not in r)
                and not (NO_REF and r.startswith(REFCHECK)) and not (NO_SHA and r == "operation names HEAD's SHA")]
    if blocking:
        RUN[v] = v
        starts.append((v, rs))
    else:
        RUN[v] = RUN[p]
        amended.append((v, rs))

PRIORITY = ["window start", "merge", "code commit", "ritual commit", "cherry-pick (git, not a verb run)",
            "push boundary", "first commit on a branch (HEAD also on the parent branch)",
            "main contains HEAD", "HEAD came by fast-forward (source branch contains it)",
            "branch cut from HEAD (branch-cut, before V)", "branch cut from HEAD (merge-2nd-parent, before V)",
            "actor switch", "operation names HEAD's SHA",
            "branch cut from HEAD (branch-cut, time unknown)", "branch cut from HEAD (merge-2nd-parent, time unknown)"]


def primary(rs):
    for p in PRIORITY:
        if p in rs and (CUT_UNKNOWN_BLOCKS or "time unknown" not in p) \
                and not (NO_REF and p.startswith(REFCHECK)) and not (NO_SHA and p == "operation names HEAD's SHA"):
            return p
    raise AssertionError(rs)


ritual_kept = [h for h in ORDER if klass(h) == "ritual" and not dropped(h)]
kernel_total = sum(1 for h in ORDER if klass(h) == "kernel")
where = lambda h: "main-line" if LINE[h] == "main" else "branch-lines"  # noqa: E731
print("variant:", "drop-roadmap(ritual)" if DROP_RM else "as-is",
      "+unstamped" if DROP_RM_UNSTAMPED else "", "| cut-unknown blocks" if CUT_UNKNOWN_BLOCKS else "| cut-unknown does not block",
      "| NO ref check" if NO_REF else "", "| NO sha check" if NO_SHA else "")
print("dropped commits:", sum(1 for h in ORDER if dropped(h)))
print("kernel verb commits:", kernel_total, " amended:", len(amended), " new commits (run starts):", len(starts))
print("ritual commits kept:", len(ritual_kept))
print("PLANNING COMMITS REMAINING:", len(starts) + len(ritual_kept),
      dict(collections.Counter(where(h) for h, _ in starts) + collections.Counter(where(h) for h in ritual_kept)))
print("primary reason a verb could not amend its HEAD (= what ended the run before it):")
pc = collections.Counter(primary(rs) for v, rs in starts)
for k in PRIORITY:
    if pc.get(k):
        sub = collections.Counter(where(v) for v, rs in starts if primary(rs) == k)
        print(f"  {pc[k]:5d}  {k}   {dict(sub)}")
print("any-reason counts among run starts (a start can have several):")
ac = collections.Counter(r for v, rs in starts for r in rs)
sole = collections.Counter(rs[0] for v, rs in starts if len([r for r in rs if CUT_UNKNOWN_BLOCKS or 'time unknown' not in r]) == 1 and len(rs) == 1)
for k in PRIORITY:
    if ac.get(k):
        print(f"  {ac[k]:5d}  {k}  (sole reason: {sole.get(k, 0)})")
print("amends that carried a non-blocking 'time unknown' cut flag:", sum(1 for v, rs in amended if rs))
if DUMP:
    json.dump(dict(starts=starts, amended=amended, run={k: v for k, v in RUN.items()}), open(DUMP, "w"))
