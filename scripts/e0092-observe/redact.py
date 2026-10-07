#!/usr/bin/env python3
"""redact.py <M-0334 file> — prints M-0334's body with the rubric's
subsections and every section written about the observation withheld:
Decisions made during implementation, Release note, Validation and Reviewer
notes. Those are the sections that name the rubric's rules, its fixtures or
its results, so the tree the hosts work in carries none of them."""
import re, sys
s = open(sys.argv[1]).read()
body = s.split("\n---\n", 1)[1] if s.startswith("---\n") else s
a = body.index("\n### Harness\n")
b = body.index("\n## Surfaces touched\n")
body = body[:a] + "\n\nThe rubric is withheld from the trees observation runs work in.\n" + body[b:]
for heading in ("Release note", "Decisions made during implementation", "Validation", "Reviewer notes"):
    m = re.search(r"\n## " + re.escape(heading) + r"\n(.*?)(?=\n## |\Z)", body, re.S)
    if m:
        body = body[: m.start(1)] + "\nWithheld from observation runs.\n" + body[m.end(1):]
sys.stdout.write(body.lstrip("\n"))
