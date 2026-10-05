#!/usr/bin/env python3
"""redact.py <M-0334 file> — prints M-0334's body with the rubric's
subsections and the Validation section withheld, for the fixture commit
that keeps the rubric out of the tree the hosts work in."""
import sys
s = open(sys.argv[1]).read()
body = s.split("\n---\n", 1)[1] if s.startswith("---\n") else s
a = body.index("\n### Harness\n")
b = body.index("\n## Surfaces touched\n")
body = body[:a] + "\n\nThe rubric is withheld from the trees observation runs work in.\n" + body[b:]
v = body.index("\n## Validation\n")
n = body.index("\n## ", v + 1)
body = body[: v + len("\n## Validation\n")] + "\nWithheld from observation runs.\n" + body[n:]
sys.stdout.write(body.lstrip("\n"))
