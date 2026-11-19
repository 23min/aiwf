"""Connects the milestone spine to the real aiwf binary, in throwaway repositories.

It only translates. It never decides whether an action is allowed.

  new()      -> a copy of the starting world template.sh built: clone a on the epic branch, E-0001
                active, M-0001 draft (tdd: none) and M-0001/AC-1 open, pushed to a bare remote
  apply()    -> runs the aiwf command for one spine event in clone a; "ok" on exit 0, else
                "rejected"
  outputs()  -> ["commit"] for each commit the step added on top of HEAD, [] for none
  observe()  -> reads M-0001's file as raw text and names the state <milestone>_<AC>

Events:
  start   aiwf promote M-0001 in_progress
  finish  aiwf promote M-0001 done
  back    aiwf promote M-0001 draft
  cancel  aiwf cancel M-0001
  meet    aiwf promote M-0001/AC-1 met
  defer   aiwf promote M-0001/AC-1 deferred
  drop    aiwf cancel M-0001/AC-1
  reopen  aiwf promote M-0001/AC-1 open
  check   aiwf check
"""
import os
import re
import shutil
import subprocess
from pathlib import Path

TEMPLATE = Path(os.environ["MILESTONE_TEMPLATE"])
RUNS = Path(os.environ["MILESTONE_RUNS"])
LOG = os.environ.get("MILESTONE_LOG")
_count = [0]

COMMANDS = {
    "start": ["promote", "M-0001", "in_progress"],
    "finish": ["promote", "M-0001", "done"],
    "back": ["promote", "M-0001", "draft"],
    "cancel": ["cancel", "M-0001"],
    "meet": ["promote", "M-0001/AC-1", "met"],
    "defer": ["promote", "M-0001/AC-1", "deferred"],
    "drop": ["cancel", "M-0001/AC-1"],
    "reopen": ["promote", "M-0001/AC-1", "open"],
    "check": ["check"],
}


def run(cwd, *args):
    return subprocess.run(args, cwd=cwd, capture_output=True, text=True)


def head(a):
    return run(a, "git", "rev-parse", "HEAD").stdout.strip()


def new():
    RUNS.mkdir(parents=True, exist_ok=True)
    for old in RUNS.iterdir():
        shutil.rmtree(old)
    _count[0] += 1
    w = RUNS / str(_count[0])
    shutil.copytree(TEMPLATE, w, symlinks=True)
    run(w / "a", "git", "remote", "set-url", "origin", str(w / "remote.git"))
    return {"a": w / "a", "before": None, "after": None, "steps": []}


def apply(h, event, env):
    a = h["a"]
    h["before"] = head(a)
    p = run(a, "aiwf", *COMMANDS[event])
    h["after"] = head(a)
    result = "ok" if p.returncode == 0 else "rejected"
    h["steps"].append(event)
    if LOG:
        with open(LOG, "a") as f:
            f.write(f"{' '.join(h['steps'])}\t{result}\texit={p.returncode}\t"
                    f"{(p.stdout + p.stderr).strip().splitlines()[-1:] }\n")
    return result


def outputs(h):
    if h["before"] == h["after"]:
        return []
    n = int(run(h["a"], "git", "rev-list", "--count", f"{h['before']}..{h['after']}").stdout.strip())
    return ["commit"] * n


def observe(h):
    files = list(h["a"].glob("work/**/M-0001-*.md"))
    if len(files) != 1:
        raise RuntimeError(f"expected one M-0001 file, found {files}")
    text = files[0].read_text()
    front = text.split("---\n")[1]
    milestone = re.search(r"^status: (\S+)$", front, re.M).group(1)
    acs = front.split("\nacs:\n", 1)[1]
    ac = re.search(r"^\s+status: (\S+)$", acs, re.M).group(1)
    return f"{milestone}_{ac}"
