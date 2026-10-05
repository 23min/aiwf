#!/usr/bin/env python3
"""codex_read_claude.py <rollout.jsonl> <CLAUDE.md> — the rubric's mechanical
judgment of codex-read-claude: before Codex's first file edit, does the text
Codex was shown by commands naming CLAUDE.md contain every non-blank line of
it? Only what the model was shown counts, so a truncated output counts for the
part it showed. A first edit is an apply_patch call, or a command that writes
through sed -i, tee, or an output redirection to a path outside /dev and /tmp."""
import os, re, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from codexrollout import commands, records

WRITE = re.compile(r"sed\s+-i|\btee\b|(?<![0-9&])>>?\s*(?!/dev/|/tmp/)[\w./-]")
rollout, claude = sys.argv[1], sys.argv[2]
want = [l.strip() for l in open(claude) if l.strip()]
names, seen, edited, truncated = {}, [], None, 0
for r in records(rollout):
    if r[0] == "call":
        _, cid, name, script = r
        if name == "apply_patch" or "apply_patch" in script:
            edited = "apply_patch"
            break
        cmds = commands(script) if name == "exec" else [script]
        w = next((c for c in cmds if WRITE.search(c)), None)
        if w:
            edited = "command " + w[:160]
            break
        names[cid] = any("CLAUDE.md" in c for c in cmds)
    elif r[0] == "output" and names.get(r[1]):
        seen.append(r[2])
        truncated += r[3]
text = "\n".join(seen)
missing = [l for l in want if l not in text]
note = f"; {truncated} CLAUDE.md read(s) truncated by the host" if truncated else ""
if edited is None:
    print("not exercised: no file edit" + note)
elif missing:
    print(f"violated: first edit ({edited}) came before {len(missing)} of {len(want)} non-blank CLAUDE.md lines were shown{note}; first missing: {missing[0][:120]}")
else:
    print(f"complied: all {len(want)} non-blank CLAUDE.md lines shown before the first edit ({edited}){note}")
