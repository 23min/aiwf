---
id: G-020
title: '`aiwf add ac` accepts prose titles, renders one giant `### AC-N — <title>` heading'
status: addressed
---

`aiwf add ac M-NNN --title "..."` writes the title both into the YAML frontmatter `acs[].title` field AND into a body heading `### AC-N — <title>`. When the title is a short label that's fine, but real-world ACs that the user passes verbatim from a planning conversation often arrive with markdown bold, multiple sentences, or paragraph-length prose — the result is one h3 heading containing 200+ characters of bold-rendered text in the milestone view, not a heading + prose body. Reproducer:

```
aiwf add ac M-NNN --title "**Full embedment inventory.** A machine-reviewable table in the milestone tracking doc enumerates every rule encoded in: (a) ModelValidator.cs, (b) ModelParser.cs, …"
```

Resolved in commit `<TBD>` (feat(aiwf): G20 — refuse prose-y AC titles, add acs-title-prose warning). Took the strict refusal + standing-check pair:

- `entity.IsProseyTitle(s string) bool` — pure detector. Triggers: length > 80 chars, newlines, markdown formatting (`**`, `__`, backticks), link brackets (`](`), or multiple sentences (sentence-ending punctuation followed by space + capital).
- `verb.AddAC` refuses prose-y titles up front with a usage-shaped error pointing the user at the workflow: pass a short label for `--title`, hand-edit the body section under the scaffolded heading to add detail prose, examples, references.
- New `acs-title-prose` (warning) finding in `check/acs.go`; runs on every `aiwf check` pass to catch titles that landed via hand-edits or pre-G20 tooling. Severity is warning, not error — the title is still usable as a label, the user just gets nudged to refactor.

Tests pin the load-bearing cases: the actual G20 reproducer string, single-sentence labels (no false positive), exact 80-char boundary (false), 81-char (true), markdown forms, multi-sentence detection. Verb-level tests confirm the refusal happens before any disk change (zero ACs added) and that the happy short-label path still works.

---
