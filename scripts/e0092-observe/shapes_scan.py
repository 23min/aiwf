#!/usr/bin/env python3
"""shapes_scan.py <observe-root>: every recorded command, Claude's
Bash and Codex's in any thread, holding a shape codexrollout.is_write is known
to misjudge, where its answer goes the way that shape errs: a missed write
(answer False) or a write that is not one (answer True). Each line is for a
reader to check by hand."""
import sys, glob, json, os, re, collections
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from codexrollout import commands, is_write, records, run_path, threads
# shape: (pattern, the answer the shape can make wrong)
SHAPES = {
    "cd in a subshell or substitution": (r"\(\s*cd\s", False),
    "variable reassigned": (None, False),
    "variable set from $HOME, $TMPDIR or ~": (r"\b\w+=(?:\$HOME|\$\{HOME\}|~|\$TMPDIR|\$\{TMPDIR\})", True),
    "default expansion": (r"\$\{\w+:-", True),
    "redirect into a substitution": (r">\s*\"?\$\(", True),
    "mktemp placed by -p or --tmpdir": (r"mktemp[^)\n;]*(?:\s-\w*p\b|--tmpdir)", True),
    "mktemp in backticks": (r"`\s*mktemp", True),
    "cd not followed": (r"(?:^|[;&|\s(])(?:cd\s+(?:~|-|\$HOME|\$\{HOME\})(?:\s|$|/)|pushd\s)", True),
    "loop over absolute paths": (r"\bfor\s+\w+\s+in\s+['\"]?/", True),
}
root = sys.argv[1]; hits = collections.Counter()
for run in sorted(glob.glob(root + "/*/*-[0-9]")):
    repo = os.path.join(run_path(run), "repo"); cmds = []
    if "/codex-" in run:
        main, subs = threads(run + "/codex-home")
        for f in [main] + [x for _, x in subs]:
            for r in records(f):
                if r[0] == "call": cmds += commands(r[3]) if r[2] == "exec" else [r[3]]
    elif os.path.exists(run + "/events.jsonl"):
        for l in open(run + "/events.jsonl"):
            try: e = json.loads(l)
            except ValueError: continue
            if e.get("type") == "assistant":
                for c in e["message"].get("content", []):
                    if c.get("type") == "tool_use" and c.get("name") == "Bash": cmds.append(c["input"].get("command", ""))
    for c in cmds:
        w = is_write(c, repo)
        assigned = re.findall(r"(?:^|[;&|\s(])(?:export\s+)?([A-Za-z_]\w*)=", c)
        for n, (p, errs) in SHAPES.items():
            found = len(assigned) != len(set(assigned)) if p is None else re.search(p, c)
            if found and w == errs:
                hits[n] += 1
                print(f"{os.path.relpath(run, root)}\t{n}\twrite={w}\t{' '.join(c.split())[:220]}")
print(dict(hits))
