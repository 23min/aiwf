#!/usr/bin/env python3
"""summarize.py <observe-dir> — the tables M-0334 and M-0338 record: each
rule's verdicts per host, the continuation runs' verdicts, and each task's
loading observations and task-loaded words per host.

A guidance document counts as read when a tool result showed the model one of
its signature lines: lines at least 40 characters long that occur once in the
document as the run's start commit holds it, three taken from its start,
middle and end. Claude invoking a skill through its Skill tool counts as
reading that skill. A first edit is Claude's Edit or Write on a file in the
run's repository, or a command codexrollout.is_write judges to write one.
Reads are counted until the first edit, and after it."""
import collections, glob, json, os, re, subprocess, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from codexrollout import _inside, commands, is_write, records, run_path, threads

# The documents the loading observations follow. CLAUDE.md is Codex's only:
# Claude has it injected, so no tool call of Claude's reads it.
FIXED = {
    "CLAUDE.md": "CLAUDE.md",
    "router": ".guidance/project.md",
    "index": ".guidance/index.md",
    "code-health": ".guidance/packs/code-health/guide.md",
    "code-health rubric": ".guidance/packs/code-health/rubric.md",
    "go": ".guidance/packs/go/cobra/guide.md",
    "python": ".guidance/packs/python/astral/guide.md",
    "typescript": ".guidance/packs/typescript/npm-vitest/guide.md",
    "aiwf-promote skill": ".claude/skills/aiwf-promote/SKILL.md",
    "aiwf-reallocate skill": ".claude/skills/aiwf-reallocate/SKILL.md",
}
# A document no tool reads at the start, such as one a reduction creates,
# joins the columns by its file name under .guidance/project/.
PRIMED = {"CLAUDE.md", "router", "index"}

def text_at(repo, start, path):
    """A document as the run started with it: from the clone at the start
    commit, or, once the clone is gone, from the run's start-tree/ copy."""
    if os.path.isdir(os.path.join(repo, ".git")):
        out = subprocess.run(["git", "-C", repo, "show", f"{start}:{path}"], capture_output=True, text=True)
        if out.returncode == 0:
            return out.stdout
        p = os.path.join(repo, path)
    else:
        p = os.path.join(os.path.dirname(repo), "start-tree", path)
    return open(p).read() if os.path.exists(p) else None

def signature(text, others=""):
    """Up to three lines that identify a document: at least 40 characters,
    once in it, and in no other text the run could be shown in their place."""
    lines = [l.strip() for l in text.split("\n")]
    long_once = [l for l in lines if len(l) >= 40 and lines.count(l) == 1 and l not in others]
    if not long_once:
        best = max(lines, key=len, default="")
        return [best] if best else []
    picks = {long_once[0], long_once[len(long_once) // 2], long_once[-1]}
    return sorted(picks)

def documents(run):
    repo = os.path.join(run, "repo")
    recorded = os.path.join(run_path(run), "repo")
    start = open(os.path.join(run, "start-head")).read().strip()
    docs = {}
    for name, path in FIXED.items():
        t = text_at(repo, start, path)
        if t:
            docs[name] = (path, t)
    if os.path.isdir(os.path.join(repo, ".git")):
        listed = subprocess.run(["git", "-C", repo, "ls-tree", "--name-only", start, ".guidance/project/"], capture_output=True, text=True).stdout.split()
    else:
        tree = os.path.join(os.path.dirname(repo), "start-tree", ".guidance", "project")
        listed = [f".guidance/project/{f}" for f in sorted(os.listdir(tree))] if os.path.isdir(tree) else []
    for path in listed:
        t = text_at(repo, start, path)
        if t:
            docs[os.path.basename(path)] = (path, t)
    agents = text_at(repo, start, "AGENTS.md") or ""
    return docs, recorded, agents

def claude_events(path, repo):
    pending = {}
    for line in open(path):
        try:
            e = json.loads(line)
        except ValueError:
            continue
        if e.get("type") == "assistant":
            for c in e.get("message", {}).get("content", []):
                if c.get("type") != "tool_use":
                    continue
                name, inp = c.get("name"), c.get("input") or {}
                if name in ("Edit", "Write", "NotebookEdit") and _inside(inp.get("file_path", "") or inp.get("notebook_path", ""), repo):
                    yield ("edit", None)
                elif name == "Bash" and is_write(inp.get("command", ""), repo):
                    yield ("edit", None)
                elif name == "Skill":
                    yield ("skill", str(inp.get("skill") or inp.get("name") or ""))
        elif e.get("type") == "user":
            content = e.get("message", {}).get("content", [])
            if isinstance(content, list):
                for c in content:
                    if c.get("type") == "tool_result":
                        r = c.get("content")
                        if isinstance(r, list):
                            r = "\n".join(x.get("text", "") for x in r if isinstance(x, dict))
                        yield ("shown", str(r))

def codex_events(home, repo):
    """What Codex's main thread was shown, and edits from any thread, in time
    order: a sub-agent's edit is the run's edit, its reads are not the main
    model's."""
    main, subs = threads(home)
    if not main:
        return
    stream = [(ts, True, r) for ts, r in records(main, True)] + [(ts, False, r) for _, f in subs for ts, r in records(f, True)]
    stream.sort(key=lambda x: x[0])
    for _, in_main, r in stream:
        if r[0] == "call":
            _, _, name, script = r
            if name == "apply_patch" or "apply_patch" in script or any(is_write(c, repo) for c in (commands(script) if name == "exec" else [script])):
                yield ("edit", None)
        elif r[0] == "output" and in_main:
            yield ("shown", r[2])

def loading(run, host):
    docs, repo, agents = documents(run)
    if host == "claude":
        docs.pop("CLAUDE.md", None)
    sigs = {name: signature(t, agents + "".join(o for n2, (_, o) in docs.items() if n2 != name)) for name, (_, t) in docs.items()}
    events = codex_events(os.path.join(run, "codex-home"), repo) if host == "codex" else claude_events(os.path.join(run, "events.jsonl"), repo)
    seen, edited = {}, False
    for kind, payload in events:
        if kind == "edit":
            edited = True
            continue
        hits = []
        if kind == "skill":
            hits = [n for n in docs if n.endswith(" skill") and n.split(" ")[0] == payload.split(":")[-1]]
        else:
            hits = [n for n, lines in sigs.items() if lines and any(l in payload for l in lines)]
        for n in hits:
            seen.setdefault(n, "after" if edited else "before")
    words = sum(len(docs[n][1].split()) for n in seen if n not in PRIMED)
    return seen, words

def incomplete(run):
    return re.search(r"^INCOMPLETE", open(os.path.join(run, "after.txt")).read(), re.M) is not None

def main():
    root = sys.argv[1]
    verdicts = {"main": collections.defaultdict(collections.Counter), "cont": collections.defaultdict(collections.Counter)}
    loads = collections.defaultdict(collections.Counter)
    words = collections.defaultdict(list)
    runs = collections.Counter()
    skipped = []
    for run in sorted(glob.glob(f"{root}/*/*-[0-9]")):
        task = os.path.basename(os.path.dirname(run))
        host = os.path.basename(run).split("-")[0]
        if incomplete(run):
            skipped.append(f"{task}/{os.path.basename(run)}")
            continue
        runs[(task, host)] += 1
        table = verdicts["cont" if task.endswith("-cont") else "main"]
        v = os.path.join(run, "verdicts.json")
        if os.path.exists(v):
            for rule, x in json.load(open(v)).get("verdicts", {}).items():
                assert x.get("verdict") in ("violated", "complied", "not exercised", "split"), (run, rule, x)
                table[(rule, host)][x["verdict"]] += 1
        crc = os.path.join(run, "codex-read-claude.txt")
        if host == "codex" and os.path.exists(crc):
            verdict = open(crc).read().split(":")[0]
            assert verdict in ("violated", "complied", "not exercised"), (run, verdict)
            table[("codex-read-claude", "codex")][verdict] += 1
        seen, w = loading(run, host)
        for t, when in seen.items():
            loads[(task, host)][f"{t} {when}"] += 1
        words[(task, host)].append(w)
    for key, title in (("main", "Verdicts per rule and host"), ("cont", "Continuation runs' verdicts per rule and host")):
        if not verdicts[key]:
            continue
        print(f"## {title} (violated / complied / not exercised / split)\n")
        print("| rule | Claude | Codex |\n|---|---|---|")
        for rule in sorted({r for r, _ in verdicts[key]}):
            cell = lambda h: "/".join(str(verdicts[key][(rule, h)][k]) for k in ("violated", "complied", "not exercised", "split")) if (rule, h) in verdicts[key] else "—"
            print(f"| `{rule}` | {cell('claude')} | {cell('codex')} |")
        print()
    print("## Loading per task and host (runs shown each document before / after their first edit; mean task-loaded words)\n")
    print("| task | host | runs | reads | task-loaded words |\n|---|---|---|---|---|")
    for (task, host), n in sorted(runs.items()):
        c = loads[(task, host)]
        names = list(FIXED) + sorted({k.rsplit(" ", 1)[0] for k in c} - set(FIXED))
        parts = [f"{t} {c[f'{t} before']}/{c[f'{t} after']}" for t in names if c[f"{t} before"] or c[f"{t} after"]]
        mean = round(sum(words[(task, host)]) / n)
        print(f"| {task} | {host} | {n} | {', '.join(parts) or 'none'} | {mean} |")
    if skipped:
        print("\nIncomplete, not judged: " + ", ".join(skipped))

main()
