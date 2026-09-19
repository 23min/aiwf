# Codex instructions for developing aiwf

Before changing this repository, read [CLAUDE.md](CLAUDE.md) in full. It is the
canonical source for aiwf's repository development rules, including engineering
conventions, validation, commit approvals, and release discipline. Keep those
rules in that file; do not maintain a second copy here.

For assistant-specific operations, follow the Codex-rendered aiwf guidance and
skills when installed in this checkout. Claude's `@file` imports and tool names
are not Codex instructions. Do not load `.claude/aiwf-guidance.md` as Codex's
workflow guidance; use the managed guidance in this file and `.agents/skills/`.
If those artifacts are absent, report that setup is needed before relying on a
workflow skill. Repository development rules still come from `CLAUDE.md`.

Reading the referenced development rules is an explicit step; a Markdown link
is not an automatic Codex import.
