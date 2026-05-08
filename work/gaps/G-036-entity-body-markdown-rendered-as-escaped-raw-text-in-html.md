---
id: G-036
title: Entity body markdown rendered as escaped raw text in HTML
status: addressed
addressed_by_commit:
  - d1bf1e1
---

Resolved in commit `(this commit)` (fix(aiwf): G35/G36 — render gap/ADR/decision/contract pages with HTML markdown bodies). New `markdownToHTML` helper in `internal/htmlrender/markdown.go` runs each body section through `goldmark` and returns `template.HTML` so the rendered HTML isn't double-escaped. Goldmark configured with Tables/Strikethrough/Linkify/TaskList extensions but raw-HTML pass-through OFF — bodies are committed to git but the static-site step refuses to upgrade that trust into "browser-executable" (XSS guard pinned by `TestMarkdownToHTML_RawHTMLEscaped`). `epic.tmpl`, `milestone.tmpl`, and the new `entity.tmpl` route every body section through the helper. New dep `github.com/yuin/goldmark v1.8.2` — pure-Go CommonMark renderer, no CGO, single transitive tree (justified per `CLAUDE.md` Dependencies). Tests: `TestMarkdownToHTML_RenderingShapes` covers paragraphs, fenced code, inline code, ordered/unordered lists, links, subheadings, emphasis; `TestRender_BodyMarkdownRendersAsHTML` is the verb-seam test that drives a real page render and asserts `<ul>/<li>`, `<code>aiwf check</code>`, link `href`, `<pre>` all appear and that no raw-markdown source leaks through. Smoke: rendered a fixture body with lists, links, fenced code blocks — output is correct HTML.

---

<a id="g37"></a>
