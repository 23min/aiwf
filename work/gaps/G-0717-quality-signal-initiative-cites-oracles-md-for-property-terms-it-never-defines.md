---
id: G-0717
title: quality-signal initiative cites oracles.md for property terms it never defines
status: open
---
## What's missing

`docs/initiatives/quality-signal-and-cadence.md` classifies the gaps it cites by
which oracle property each loses, and names `docs/design/oracles.md` as where that
vocabulary lives. Three of the four terms it uses appear nowhere in that file.
Measured in the main checkout at `573948282`:

```
$ sed -n 247,252p docs/initiatives/quality-signal-and-cadence.md

The vocabulary for classifying these lives in
[`../design/oracles.md`](../design/oracles.md) — what makes a check able to
decide anything, and which property each of the above loses. G-0253 and G-0328
lose *depth*; G-0110 loses *reach*; G-0317 loses *specificity*; the audit above
loses *independence*, which is the property whose absence is hardest to see,

$ for w in depth reach specificity independence; do printf "%s: " $w; grep -c -i -w "$w" docs/design/oracles.md; done
depth: 0
reach: 0
specificity: 0
independence: 4
```

`oracles.md` §"The properties that matter" names six properties — Independence;
A verdict, not a vibe; Cost proportional to the frequency you need; Diagnostic
locality; Stability under irrelevant change; A deliberately chosen failure
asymmetry — and none of them is depth, reach, or specificity. No line of the file
has ever contained the three strings, in any case:

```
$ for w in depth reach specificity; do printf "%s: " $w; git log --oneline -i --pickaxe-regex -S"$w" -- docs/design/oracles.md | wc -l; done
depth: 0
reach: 0
specificity: 0
```

Two near-echoes in `oracles.md` do not recover the mapping. "Free, broad, shallow"
(lines 63–64) describes the Implicit class of oracle, not a property, and G-0253
and G-0328 are not in that class. "How specifically" (line 23) is how precisely a
failure is reported — the Diagnostic locality property — while G-0317 concerns a
check that passes on a mere reference to the file, which is not a reporting
problem.

The defect shows at the initiative's lines 250–252. The definitions the sentence
points to would sit in `oracles.md` §"The properties that matter".

## Why it matters

A reader who follows the link to learn what *depth*, *reach* and *specificity* mean
finds a different set of properties, so three of the four classifications cannot be
checked against the source the sentence names, and the attribution tells that
reader the definitions exist.

*Depth* is also the term the initiative organizes its Q2 stance around — "known
depth gaps" (line 214), "depth over breadth, per Q2" (line 445). Work promoted from
that stance carries a term with no definition in the source it cites.

## Relates to

G-0716 records a different defect in the same passage: the "G-0110 loses *reach*"
clause rests on the premise about G-0110 that G-0716 records as false.
