---
id: G-0664
title: shipped-prose-assertion misses an assertion whose shipped path is built inline
status: open
discovered_in: M-0328
---
## What's missing

`shipped-prose-assertion` judges a containment only once it knows the haystack
carries shipped-surface bytes, and it learns that from `shippedPathConsts`, which
collects names from a `//go:embed` directive or a declared `const`/`var`. A path
computed inside a function body — `data, err := os.ReadFile(filepath.Join(root,
"internal", "skills", "embedded-guidance", "aiwf-guidance.md"))` — is an
assignment statement, not a value spec, so the name it binds is never collected
and every containment over it goes unjudged.

`internal/policies/m0211_guidance_operating_anchors.go` is the live instance. It
reads the shipped guidance through exactly that inline join and asserts fourteen
curated substrings against the result. It appears in neither exemption map, and
both maps state that they carry no grandfather entries.

Measured: adding a bare `strings.Contains(lower, "each mutating action")` to that
file leaves the policy suite green. Every firing fixture the ban ships routes its
read through a named constant instead.

## Why it matters

D-0070 retired prose-content assertions over shipped surfaces because scoping a
phrase still pins a reading that rewording breaks, and nothing catches the break.
The ban is the mechanism that makes the retirement hold rather than merely being
declared. An escape reachable by spelling a path one way instead of another means
the retirement holds only where an author happened to declare a constant — and
the escape needs no intent, since an inline read is the more natural spelling.

The cost compounds through precedent. Anyone reaching for the anchor set as a
model for a new assertion over shipped prose writes the banned class, sees green,
and reasonably concludes the class is permitted here.

Whether the anchor set earns a recorded exemption or needs a different mechanism
is the open question. Neither is written down today, because nothing asked.
