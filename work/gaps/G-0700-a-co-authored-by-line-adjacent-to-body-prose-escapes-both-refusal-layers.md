---
id: G-0700
title: A Co-Authored-By line adjacent to body prose escapes both refusal layers
status: open
---
## What's missing

Both layers of the `Co-Authored-By` refusal locate the trailer through git's own
trailer-block heuristic, so a co-author line git does not recognize as a trailer
is invisible to both. A line sitting directly under body prose is such a line.

Reproduced on git 2.54.0, in a repo whose `aiwf.yaml` lists
`noreply@anthropic.com` under `provenance.refuse_coauthors`:

```
$ printf 'feat: a thing\n\nSome explanation of the change.\nCo-Authored-By: Claude <noreply@anthropic.com>\n' > msg
$ aiwf check --commit-msg msg --root .
$ echo $?
0
$ git interpret-trailers --parse < msg | wc -c
0
```

Expected: refusal, as for the same line in a trailer block. Observed: exit 0 and
an empty parse — git takes trailers from the last paragraph only, and that
paragraph opens with prose, so the block is not a block. `%(trailers)` in
`internal/policies/coauthor_trailer_ban.go` reads the same heuristic, so the
range check reports the commit clean too. The two layers agree; they are both
blind.

The consequence depends on who else parses the message, and that part is not
settled here. GitHub is reported to accept a co-author line more leniently than
git does, which would mean such a commit still renders an AI co-author on the
pull request while both aiwf layers pass it. Not measured — settle it by pushing
a commit of exactly the shape above to a scratch GitHub repository and reading
the rendered commit page.

## Why it matters

The refusal's worth is the claim that a co-author line cannot land. Where git's
heuristic is the oracle, that claim holds only for lines git would have read as
trailers — which is the right scope if the trailer block is what matters, and
the wrong one if the rendered attribution is.

The decision this needs is which oracle the rule is written against, and it is a
decision rather than a defect: widening past git's heuristic means matching a
`Co-Authored-By:`-shaped line anywhere in the message, which no longer has git's
parser behind it and would refuse the line quoted inside a message discussing
this very rule.

D-0096 records the current design and does not address this case.
