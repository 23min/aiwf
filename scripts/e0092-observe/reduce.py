#!/usr/bin/env python3
"""reduce.py <events.jsonl | codex-home> <run-dir> — prints one plain
transcript, for Codex the main thread followed by each sub-agent thread: assistant text, tool calls with their arguments, tool results as
the model saw them truncated at 1,500 characters, and compaction events.
Product and model names are replaced; the run directory becomes <run>."""
import json, os, re, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from codexrollout import records, threads

LIMIT = 1500
# Case-sensitive product and model names only: CLAUDE.md, AGENTS.md, .claude/
# and .agents/ paths are evidence the judge needs, so they stay.
NAMES = re.compile(r"claude-opus-5-5|gpt-6-astra|\b(?:Claude Code|Claude|Codex|Anthropic|OpenAI|GPT|Opus|Sonnet|Haiku)\b")

def mask(text, run):
    """Removes every mark of the host from text a judge reads: the run's path
    in any spelling, the hosts' own home directories and file names, and
    product and model names."""
    run = run.rstrip("/")
    observe = os.path.dirname(os.path.dirname(run))
    text = text.replace(run, "<run>")
    text = text.replace(run.replace("/", "-"), "<run>")
    # A path into another run, or anywhere under the observation directory,
    # names a task and host the same way, as does Claude Code's own
    # temporary directory.
    text = re.sub(r"\S*/(?:T\d+|L\d+)(?:-cont)?/(?:claude|codex)-\d+", "<run>", text)
    text = text.replace(observe, "<observe>").replace(observe.replace("/", "-"), "<observe>")
    text = re.sub(r"/tmp/claude-[\w-]+", "/tmp/<tmp>", text)
    text = text.replace("<run>/codex-home", "<run>/agent-home")
    text = re.sub(r"\bcodex\d+-", "tool-", text)
    return NAMES.sub("<assistant>", text)

# The run whose transcript is being reduced; set before any reduction, so a
# result is masked before it is cut and no half path escapes the mask.
RUN = ""

def trunc(s):
    s = mask(s, RUN) if RUN else s
    return s if len(s) <= LIMIT else s[:LIMIT] + f"\n…[{len(s) - LIMIT} more chars]"

def claude(path):
    out = []
    for line in open(path):
        try:
            e = json.loads(line)
        except ValueError:
            continue
        t = e.get("type")
        if t == "assistant":
            for c in e.get("message", {}).get("content", []):
                if c.get("type") == "text":
                    out.append("ASSISTANT: " + c["text"])
                elif c.get("type") == "tool_use":
                    out.append(f"TOOL CALL {c.get('name')}: " + json.dumps(c.get("input"), ensure_ascii=False))
        elif t == "user":
            content = e.get("message", {}).get("content", [])
            if isinstance(content, list):
                for c in content:
                    if c.get("type") == "tool_result":
                        r = c.get("content")
                        if isinstance(r, list):
                            r = "\n".join(x.get("text", "") for x in r if isinstance(x, dict))
                        out.append("TOOL RESULT: " + trunc(str(r)))
        elif t == "system" and e.get("subtype") == "compact_boundary":
            out.append("EVENT: context compacted")
        elif t == "result":
            out.append(f"END: {e.get('subtype')} {e.get('terminal_reason') or ''}".rstrip())
    return out

def codex(path):
    out = []
    for r in records(path):
        if r[0] == "assistant":
            out.append("ASSISTANT: " + r[1])
        elif r[0] == "call":
            out.append(f"TOOL CALL {r[2]}: {r[3]}")
        elif r[0] == "output":
            out.append(("TOOL RESULT (truncated by host): " if r[3] else "TOOL RESULT: ") + trunc(r[2]))
        elif r[0] == "compacted":
            out.append("EVENT: context compacted")
    return out

if __name__ == "__main__":
    path, run = sys.argv[1], sys.argv[2]
    RUN = os.path.abspath(run)
    if os.path.isdir(path):
        main, subs = threads(path)
        out = codex(main) if main else ["EVENT: no main rollout found"]
        for agent, f in subs:
            out.append(f"=== SUB-AGENT THREAD {agent} ===")
            out.extend(codex(f))
    else:
        out = claude(path)
    print(mask("\n\n".join(out), os.path.abspath(run)))
