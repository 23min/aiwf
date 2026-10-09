#!/usr/bin/env python3
"""trunc_final.py <observe-root>: Codex's whole-file reads of
CLAUDE.md -- main-thread outputs of a call one of whose commands is `cat
CLAUDE.md` -- with the characters shown, how many carry a truncation marker,
how many of the file's leading non-blank lines each shows, and in how many the
file's last non-blank line was shown."""
import os, sys, glob, re
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from codexrollout import commands, records, threads
root = sys.argv[1]
claude = open(root + '/T1/codex-1/CLAUDE.at-start.md').read()
lines = claude.split('\n'); last = [l for l in lines if l.strip()][-1].strip()
n = 0; lens = []; heads = []; ends = 0; markers = 0
for d in sorted(glob.glob(root + '/*/codex-[0-9]')):
    main, _ = threads(d + '/codex-home'); whole = {}
    for r in records(main):
        if r[0] == 'call':
            cmds = commands(r[3]) if r[2] == 'exec' else [r[3]]
            whole[r[1]] = any(re.search(r'(^|[;&|]\s*)cat\s+(\./)?CLAUDE\.md(\s|$|;|&)', c) for c in cmds)
        elif r[0] == 'output' and whole.get(r[1]):
            t = r[2]; n += 1; lens.append(len(t))
            markers += bool(re.search(r'tokens truncated', t))
            k = 0
            for l in lines:
                if l.strip() and l.strip() not in t: break
                k += 1
            heads.append(k)
            ends += last in t
print('whole-file reads', n, 'output chars', min(lens), max(lens), 'with marker', markers, 'leading lines shown', min(heads), max(heads), 'file end shown', ends, 'file chars', len(claude), 'lines', len(lines))
