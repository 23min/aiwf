"""Connects the scope spine to the real aiwf binary, in throwaway repositories.

It only translates. It never decides whether an action is allowed.

  new()      -> a copy of the starting world template.sh built: clone a on the epic branch
                with M-0001 in progress, clone b on main, one bare remote
  apply()    -> runs the commands for one spine event in clone a; "ok" on exit 0, else "rejected"
  observe()  -> reads `aiwf show M-0001 --format json` and names the spine state

Events:
  authorize  aiwf authorize M-0001 --to ai/claude
  act        aiwf retitle M-0001 <a new title> --actor ai/claude --principal human/peter
  end        aiwf authorize M-0001 --end --scope <sha>, for every active scope; "rejected" when
             there is none or any end is refused
  rebase     clone b pushes a commit to main; clone a fetches and rebases onto origin/main
  check      aiwf check; "rejected" when it exits non-zero
"""
import json
import os
import shutil
import subprocess
from pathlib import Path

TEMPLATE = Path(os.environ["SCOPE_TEMPLATE"])
RUNS = Path(os.environ["SCOPE_RUNS"])
_count = [0]


def run(cwd, *args):
    return subprocess.run(args, cwd=cwd, capture_output=True, text=True)


def new():
    RUNS.mkdir(parents=True, exist_ok=True)
    for old in RUNS.iterdir():
        shutil.rmtree(old)
    _count[0] += 1
    w = RUNS / str(_count[0])
    shutil.copytree(TEMPLATE, w, symlinks=True)
    for clone in ("a", "b"):
        run(w / clone, "git", "remote", "set-url", "origin", str(w / "remote.git"))
    return {"w": w, "a": w / "a", "b": w / "b", "n": 0}


def scopes(h):
    p = run(h["a"], "aiwf", "show", "M-0001", "--format", "json")
    d = json.loads(p.stdout)
    return [s for s in d.get("result", d).get("scopes") or [] if s["agent"] == "ai/claude"]


def apply(h, event, env):
    a = h["a"]
    if event == "authorize":
        p = run(a, "aiwf", "authorize", "M-0001", "--to", "ai/claude", "--reason", "spike")
        return "ok" if p.returncode == 0 else "rejected"
    if event == "act":
        h["n"] += 1
        p = run(a, "aiwf", "retitle", "M-0001", f"Spike milestone {h['n']}",
                "--actor", "ai/claude", "--principal", "human/peter")
        return "ok" if p.returncode == 0 else "rejected"
    if event == "end":
        live = [s["auth_sha"] for s in scopes(h) if s["state"] == "active"]
        if not live:
            return "rejected"
        for sha in live:
            p = run(a, "aiwf", "authorize", "M-0001", "--end", "--scope", sha[:7], "--reason", "over")
            if p.returncode != 0:
                return "rejected"
        return "ok"
    if event == "rebase":
        b = h["b"]
        h["n"] += 1
        run(b, "git", "pull", "-q", "--rebase", "origin", "main")
        (b / f"other{h['n']}.txt").write_text("y\n")
        run(b, "git", "add", "-A")
        run(b, "git", "commit", "-qm", f"feat: other {h['n']}")
        if run(b, "git", "push", "-q", "origin", "main").returncode != 0:
            raise RuntimeError("clone b could not move main")
        run(a, "git", "fetch", "-q", "origin")
        p = run(a, "git", "rebase", "-q", "origin/main")
        return "ok" if p.returncode == 0 else "rejected"
    if event == "check":
        p = run(a, "aiwf", "check")
        return "ok" if p.returncode == 0 else "rejected"
    raise ValueError(f"unknown event {event!r}")


def observe(h):
    s = scopes(h)
    if not s:
        return "none"
    if any(x["state"] == "active" for x in s):
        return "active"
    return "ended"
