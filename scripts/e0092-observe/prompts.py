#!/usr/bin/env python3
"""prompts.py <M-0334 file> <task> — prints the task's prompt verbatim from the
committed rubric, so the harness never carries a second copy of it."""
import re, sys
body = open(sys.argv[1]).read()
task = sys.argv[2]
m = re.search(r"\*\*" + re.escape(task) + r" — [^*]*\*\*.*?Prompt:\n\n((?:> [^\n]*\n?)+)", body, re.S)
if not m:
    sys.exit(f"no prompt for {task}")
print("\n".join(l[2:] for l in m.group(1).rstrip("\n").split("\n")))
