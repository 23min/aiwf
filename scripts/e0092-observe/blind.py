#!/usr/bin/env python3
"""blind.py <run-dir> — writes the judge's copies of a run's after-state and
diff, judge-after.txt and judge-diff.patch, with every mark of the host
removed and no evidence dropped: the run's path in any spelling becomes
<run>, product and model names are masked, the header, version and plugin
lines go, and the commit list is recomputed to leave out commits a task's
setup put on trunk."""
import os, re, subprocess, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from codexrollout import run_path
from reduce import mask

run = os.path.abspath(sys.argv[1])
recorded = run_path(run)
start = open(os.path.join(run, "start-head")).read().strip()
origin = open(os.path.join(run, "start-origin")).read().strip()
repo = os.path.join(run, "repo")

# Claude Code's own housekeeping files appear only for Claude runs, so they
# would name the host without bearing on any rule.
HOUSEKEEPING = re.compile(r"/\.claude/(?:\.last-cleanup|\.last-update-result\.json|skills/synced/)")
# The plugins section is Claude's alone. The binary's after-run hash and the
# /tmp listing are records for a human reader, kept out of the judge's copy so
# every round's judge reads the same sections the baseline's did.
DROP_SECTIONS = {"--- plugins", "--- binary under test after the run", "--- files written under /tmp"}

sections, current = [("", [])], None
for line in open(os.path.join(run, "after.txt")).read().rstrip("\n").split("\n"):
    if line.startswith("--- "):
        sections.append((line, []))
    else:
        sections[-1][1].append(line)
out = []
for header, body in sections:
    if any(header.startswith(d) for d in DROP_SECTIONS):
        continue
    if header.startswith("--- commits since start"):
        if os.path.isdir(os.path.join(repo, ".git")):
            out_ = subprocess.run(["git", "-C", repo, "log", "--all", "--format=%h %D | %s%n%b", f"^{start}", f"^{origin}"],
                                  capture_output=True, text=True, check=True).stdout.rstrip("\n")
            body = out_.split("\n") if out_ else []
        else:
            # Without the clone, the recorded list stands, less the commit a
            # task's setup put on trunk, which is the start of origin itself.
            kept, drop = [], False
            for l in body:
                first = l.split(" ", 1)[0]
                if re.fullmatch(r"[0-9a-f]{7,}", first):
                    drop = origin.startswith(first)
                if not drop:
                    kept.append(l)
            body = kept
        while body and not body[-1].strip():
            body.pop()
    body = [l for l in body if not l.startswith(("host=", "versions:", "rollout=")) and not HOUSEKEEPING.search(l)]
    out += ([header] if header else []) + body
open(os.path.join(run, "judge-after.txt"), "w").write(mask("\n".join(out) + "\n", recorded))
diff = open(os.path.join(run, "diff.patch"), errors="replace").read()
open(os.path.join(run, "judge-diff.patch"), "w").write(mask(diff, recorded))
