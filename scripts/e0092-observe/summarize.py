#!/usr/bin/env python3
"""summarize.py <observe-dir> — the baseline or after-run tables M-0334 and
M-0338 record: each rule's verdicts per host, and each task's loading
observations per host.

A read is a tool call naming the file: Claude's Read, Grep or Glob input, or a
shell command; Codex's main-thread commands. A skill counts as read when its
SKILL.md is read or Claude's Skill tool invokes it. A first edit is Claude's
Edit or Write, or a command codexrollout.is_write judges to write a file."""
import collections, glob, json, os, re, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from codexrollout import commands, is_write, records, threads

TARGETS = {
    "CLAUDE.md": r"(?<![\w./-])CLAUDE\.md",
    "router": r"\.guidance/project\.md",
    "index": r"\.guidance/index\.md",
    "code-health": r"packs/code-health/guide\.md",
    "code-health rubric": r"packs/code-health/rubric\.md",
    "go": r"packs/go/cobra/guide\.md",
    "python": r"packs/python/astral/guide\.md",
    "typescript": r"packs/typescript/npm-vitest/guide\.md",
    "aiwf-promote skill": r"aiwf-promote/SKILL\.md|^skill:aiwf-promote$",
    "aiwf-reallocate skill": r"aiwf-reallocate/SKILL\.md|^skill:aiwf-reallocate$",
}

def claude_events(path):
    for line in open(path):
        try:
            e = json.loads(line)
        except ValueError:
            continue
        if e.get("type") != "assistant":
            continue
        for c in e.get("message", {}).get("content", []):
            if c.get("type") != "tool_use":
                continue
            name, inp = c.get("name"), c.get("input") or {}
            if name in ("Edit", "Write", "NotebookEdit"):
                yield ("edit", inp.get("file_path", ""))
            elif name == "Skill":
                yield ("read", "skill:" + str(inp.get("skill") or inp.get("name") or ""))
            elif name == "Bash":
                cmd = inp.get("command", "")
                yield ("edit" if is_write(cmd) else "read", cmd)
            else:
                yield ("read", json.dumps(inp))

def codex_events(home):
    main, _ = threads(home)
    if not main:
        return
    for r in records(main):
        if r[0] != "call":
            continue
        _, _, name, script = r
        if name == "apply_patch" or "apply_patch" in script:
            yield ("edit", script)
            continue
        for cmd in (commands(script) if name == "exec" else [script]):
            yield ("edit" if is_write(cmd) else "read", cmd)

# Each on-demand document under .guidance/project/ is its own column, named
# by its file, so a round with documents the baseline lacked still shows them.
ON_DEMAND = re.compile(r"\.guidance/project/([\w-]+)\.md")

def loading(run, host):
    events = codex_events(os.path.join(run, "codex-home")) if host == "codex" else claude_events(os.path.join(run, "events.jsonl"))
    seen, edited = {}, False
    for kind, text in events:
        if kind == "edit":
            edited = True
            continue
        hits = [t for t, pat in TARGETS.items() if re.search(pat, text, re.M)]
        hits += [m + ".md" for m in ON_DEMAND.findall(text)]
        for t in hits:
            if t not in seen:
                seen[t] = "after" if edited else "before"
    return seen

def main():
    root = sys.argv[1]
    verdicts = collections.defaultdict(collections.Counter)
    loads = collections.defaultdict(collections.Counter)
    runs = collections.Counter()
    incomplete = []
    for run in sorted(glob.glob(f"{root}/*/*-[0-9]")):
        task = os.path.basename(os.path.dirname(run))
        host = os.path.basename(run).split("-")[0]
        if "INCOMPLETE" in open(os.path.join(run, "after.txt")).read():
            incomplete.append(f"{task}/{os.path.basename(run)}")
            continue
        runs[(task, host)] += 1
        v = os.path.join(run, "verdicts.json")
        if os.path.exists(v):
            for rule, x in json.load(open(v)).get("verdicts", {}).items():
                if isinstance(x, dict):
                    verdicts[(rule, host)][x.get("verdict")] += 1
        crc = os.path.join(run, "codex-read-claude.txt")
        if host == "codex" and os.path.exists(crc):
            verdicts[("codex-read-claude", "codex")][open(crc).read().split(":")[0]] += 1
        for t, when in loading(run, host).items():
            loads[(task, host)][f"{t} {when}"] += 1
    print("## Verdicts per rule and host (violated / complied / not exercised / split)\n")
    print("| rule | Claude | Codex |\n|---|---|---|")
    for rule in sorted({r for r, _ in verdicts}):
        cell = lambda h: "/".join(str(verdicts[(rule, h)][k]) for k in ("violated", "complied", "not exercised", "split")) if (rule, h) in verdicts else "—"
        print(f"| `{rule}` | {cell('claude')} | {cell('codex')} |")
    print("\n## Loading per task and host (runs reading each document before / after the first edit)\n")
    print("| task | host | runs | reads |\n|---|---|---|---|")
    for (task, host), n in sorted(runs.items()):
        c = loads[(task, host)]
        parts = []
        docs = sorted({k.rsplit(" ", 1)[0] for k in c} - set(TARGETS))
        for t in list(TARGETS) + docs:
            b, a = c[f"{t} before"], c[f"{t} after"]
            if b or a:
                parts.append(f"{t} {b}/{a}")
        print(f"| {task} | {host} | {n} | {', '.join(parts) or 'none'} |")
    if incomplete:
        print("\nIncomplete, not judged: " + ", ".join(incomplete))

main()
