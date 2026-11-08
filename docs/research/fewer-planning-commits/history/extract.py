"""Extract the window's commit data from /workspaces/aiwf, read-only.

Usage: python3 -I extract.py <repo> <outdir>
Writes <outdir>/data.json with:
  commits: {sha: {parents, ae, at, ct, subject, body, trailers:[[k,v]...], files:[[status,path]...]}}
  order:   window shas, oldest first (topo order, reversed)
  pushes:  origin/main reflog entries oldest first: [sha, unix_time, message]
  mainlog: refs/heads/main reflog oldest first: [old, new, unix_time, message]
  headlog: HEAD reflog (main checkout) oldest first: [old, new, unix_time, message]
"""
import json
import subprocess
import sys

REPO, OUT = sys.argv[1], sys.argv[2]
WINDOW_BASE = "6b87e0f240eadf10d632f89cd3006e11d33cc3b6"


def git(*a):
    return subprocess.run(["git", "-C", REPO, *a], capture_output=True, text=True, check=True).stdout


tip = git("rev-parse", "HEAD").strip()
rng = f"{WINDOW_BASE}..{tip}"

RS, US = "\x1e", "\x1f"
fmt = RS + US.join(["%H", "%P", "%ae", "%at", "%ct", "%s", "%(trailers:only,unfold)", "%B"])
commits = {}
for rec in git("log", "--format=" + fmt, rng).split(RS)[1:]:
    h, p, ae, at, ct, s, tr, body = rec.split(US)
    trailers = []
    for line in tr.splitlines():
        if ":" in line:
            k, v = line.split(":", 1)
            trailers.append([k.strip(), v.strip()])
    commits[h] = dict(parents=p.split(), ae=ae, at=int(at), ct=int(ct), subject=s,
                      body=body, trailers=trailers, files=[])

# files touched by each non-merge commit (vs its only parent), renames off
cur = None
for line in git("log", "--no-merges", "--no-renames", "--name-status", "--format=" + RS + "%H", rng).split("\n"):
    if line.startswith(RS):
        cur = line[1:]
        continue
    if line.strip() and cur:
        st, path = line.split("\t", 1)
        commits[cur]["files"].append([st, path])

order = git("rev-list", "--topo-order", "--reverse", rng).split()
assert set(order) == set(commits)


def reflog(ref):
    # %gs = reflog subject; %ct of the reflog entry is not exposed, so read the raw file
    pass


def rawlog(path):
    rows = []
    with open(path) as f:
        for line in f:
            head, _, msg = line.rstrip("\n").partition("\t")
            parts = head.split(" ")
            old, new = parts[0], parts[1]
            t = int(parts[-2])
            rows.append([old, new, t, msg])
    return rows


gitdir = git("rev-parse", "--git-common-dir").strip()
if not gitdir.startswith("/"):
    gitdir = REPO + "/" + gitdir
pushes = [[new, t, msg] for old, new, t, msg in rawlog(gitdir + "/logs/refs/remotes/origin/main")]
mainlog = rawlog(gitdir + "/logs/refs/heads/main")
headlog = rawlog(gitdir + "/logs/HEAD")

json.dump(dict(tip=tip, base=WINDOW_BASE, commits=commits, order=order, pushes=pushes,
               mainlog=mainlog, headlog=headlog), open(OUT + "/data.json", "w"))
print("commits", len(commits), "pushes", len(pushes), "mainlog", len(mainlog), "headlog", len(headlog))
