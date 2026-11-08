"""Shared model of the window: push times, lines (branches), commit classes.

Imported by the measurement scripts; reads data.json and verbsets.json from
this script's directory and calls git (read-only) on the repo for ranges.
"""
import collections
import json
import os
import subprocess

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = "/workspaces/aiwf"


def git(*a):
    return subprocess.run(["git", "-C", REPO, *a], capture_output=True, text=True, check=True).stdout


D = json.load(open(os.path.join(HERE, "data.json")))
VS = json.load(open(os.path.join(HERE, "verbsets.json")))
C = D["commits"]
ORDER = D["order"]                     # oldest first, topo
POS = {h: i for i, h in enumerate(ORDER)}
RITUAL = set(VS["ritual"])
REGISTERED = set(VS["registered"])


def trailers(h, key):
    return [v for k, v in C[h]["trailers"] if k.lower() == key]


def is_merge(h):
    return len(C[h]["parents"]) > 1


def verbs(h):
    return trailers(h, "aiwf-verb")


def klass(h):
    """'merge' | 'kernel' (aiwf verb op) | 'ritual' (plain commit stamped with a ritual verb) | 'code'."""
    if is_merge(h):
        return "merge"
    v = verbs(h)
    if not v:
        return "code"
    if set(v) & RITUAL:
        return "ritual"
    return "kernel"


def roadmap_only(h):
    paths = [p for s, p in C[h]["files"]]
    return (not is_merge(h)) and bool(paths) and all(p == "ROADMAP.md" for p in paths)


# ---- push boundaries: origin/main reflog, oldest first --------------------
PUSHES = D["pushes"]            # [sha, time, msg]
PUSHIDX = {}                    # commit -> index i of the first reflog entry containing it
PUSHTIME = {}                   # commit -> time of that entry
for i in range(1, len(PUSHES)):
    prev, cur = PUSHES[i - 1][0], PUSHES[i][0]
    # every transition must be a fast-forward, else "first entry containing" is ill-defined
    assert subprocess.run(["git", "-C", REPO, "merge-base", "--is-ancestor", prev, cur]).returncode == 0, (prev, cur)
    for h in git("rev-list", f"{prev}..{cur}").split():
        PUSHIDX[h] = i
        PUSHTIME[h] = PUSHES[i][1]
assert set(PUSHIDX) == set(C), "every window commit lies in exactly one push range"


def pushed_at(h, t):
    """True when h was reachable from origin/main at unix time t (per this clone's reflog)."""
    if h not in C:          # older than the window: pushed before the window began
        return True
    return PUSHTIME[h] <= t


# ---- main's own reflog: which first-parent commits were made on main -------
MAINLOG = D["mainlog"]
FP = git("rev-list", "--first-parent", f"{D['base']}..{D['tip']}").split()
FPSET = set(FP)
FF_ON_MAIN = set()               # first-parent commits that reached main by a fast-forward merge
FF_EVENTS = []
for o, n, t, m in MAINLOG:
    if m.startswith("merge ") and m.endswith("Fast-forward"):
        rng = git("rev-list", "--first-parent", f"{o}..{n}").split()
        FF_EVENTS.append((o, n, t, m, rng))
        FF_ON_MAIN.update(h for h in rng if h in C)
MADE_ON_MAIN = {}                # commit -> mainlog message of the entry that created it on main
for o, n, t, m in MAINLOG:
    if n in C and not m.startswith("reset") and not (m.startswith("merge ") and m.endswith("Fast-forward")):
        MADE_ON_MAIN.setdefault(n, m)

# ---- lines: main's first-parent line, then each merge's second-parent first-parent chain
LINE = {}                        # commit -> line id
LINES = collections.OrderedDict()  # line id -> list of commits oldest first
LINEMERGE = {}                   # line id -> the merge commit that brought it in (None for main)


def _assign(lid, shas, merge):
    shas = [h for h in shas if h in C and h not in LINE]
    if not shas:
        return
    LINES[lid] = shas
    LINEMERGE[lid] = merge
    for h in shas:
        LINE[h] = lid


_assign("main", list(reversed(FP)), None)
queue = collections.deque(["main"])
while queue:
    lid = queue.popleft()
    for h in reversed(LINES[lid]):          # newest merge first
        if not is_merge(h):
            continue
        p1 = C[h]["parents"][0]
        for k, p2 in enumerate(C[h]["parents"][1:], 1):
            chain = list(reversed(git("rev-list", "--first-parent", p2, f"^{p1}").split()))
            nid = f"{h[:9]}^{k + 1}"
            before = len(LINES)
            _assign(nid, chain, h)
            if len(LINES) > before:
                queue.append(nid)
UNASSIGNED = [h for h in C if h not in LINE]

# children within the window
CHILDREN = collections.defaultdict(list)
for h, c in C.items():
    for p in c["parents"]:
        CHILDREN[p].append(h)


def line_base(lid):
    """First parent of the line's oldest commit: where the branch was cut (or the previous merge tip)."""
    first = LINES[lid][0]
    return C[first]["parents"][0] if C[first]["parents"] else None


def line_of(h):
    return LINE.get(h, "pre-window")
