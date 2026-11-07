"""Find every place a window commit names a commit SHA, read-only.

Usage: python3 -I refs_extract.py
Scans (1) each window commit's full message (trailers classified by key) and
(2) each non-merge window commit's added diff lines, for hex tokens of 7-40
chars that resolve uniquely to a commit object in the repository's object
store (reachable or not). Writes refs.json: list of records
  {rec, sha, token, where, key_or_path, line}
where = 'trailer' | 'message' | 'file'.
"""
import bisect
import json
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from base import C, D, HERE, git  # noqa: E402

objs = sorted(l.split()[0] for l in git("cat-file", "--batch-all-objects", "--batch-check=%(objectname) %(objecttype)").splitlines()
              if l.endswith(" commit"))


def resolve(tok):
    i = bisect.bisect_left(objs, tok)
    hits = []
    while i < len(objs) and objs[i].startswith(tok):
        hits.append(objs[i])
        i += 1
        if len(hits) > 1:
            return None
    return hits[0] if hits else None


TOK = re.compile(r"(?<![0-9a-fA-F])[0-9a-f]{7,40}(?![0-9a-fA-F])")
out = []
for h, c in C.items():
    tkeys = {}
    for k, v in c["trailers"]:
        tkeys.setdefault(v, k)
    for line in c["body"].split("\n"):
        for m in TOK.finditer(line):
            s = resolve(m.group(0))
            if not s:
                continue
            key = None
            if ":" in line:
                k, v = line.split(":", 1)
                if k.strip().lower().startswith("aiwf-") and v.strip() == m.group(0):
                    key = k.strip().lower()
            out.append(dict(rec=h, sha=s, token=m.group(0), where="trailer" if key else "message",
                            key_or_path=key, line=line.strip()[:200]))

RS = "\x1e"
cur = path = None
raw = git("log", "-p", "--no-merges", "-U0", "--no-renames", "--no-color", "--no-ext-diff",
          "--format=" + RS + "%H", f"{D['base']}..{D['tip']}")
for line in raw.split("\n"):
    if line.startswith(RS):
        cur, path = line[1:], None
        continue
    if line.startswith("+++ "):
        path = line[6:] if line.startswith("+++ b/") else None
        continue
    if not line.startswith("+") or path is None:
        continue
    for m in TOK.finditer(line[1:]):
        s = resolve(m.group(0))
        if s:
            out.append(dict(rec=cur, sha=s, token=m.group(0), where="file", key_or_path=path, line=line[1:].strip()[:200]))

json.dump(out, open(os.path.join(HERE, "refs.json"), "w"), indent=0)
print("records", len(out))
