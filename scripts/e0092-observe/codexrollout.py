"""Shared reading of a Codex session rollout: what the model was shown."""
import json, re

def _unescape(s):
    s = re.sub(r"\\u([0-9a-fA-F]{4})", lambda m: chr(int(m.group(1), 16)), s)
    return s.replace('\\"', '"').replace("\\n", "\n").replace("\\t", "\t").replace("\\\\", "\\")

def visible(output):
    """The text a tool output put before the model, the command output
    unwrapped from its JSON envelope, and whether Codex truncated it."""
    texts = [o.get("text", "") for o in output] if isinstance(output, list) else [str(output)]
    raw = "\n".join(texts)
    truncated = "Warning: truncated output" in raw
    parts = []
    for t in texts:
        i = t.find('{"chunk_id"')
        if i < 0:
            parts.append(t)
            continue
        try:
            parts.append(json.loads(t[i:]).get("output", ""))
        except ValueError:
            m = re.search(r'"output":"(.*)', t[i:], re.S)
            parts.append(_unescape(m.group(1)) if m else t)
    return "\n".join(parts), truncated

def commands(script):
    """The shell commands an exec script runs."""
    quoted = [_unescape(c) for c in re.findall(r'cmd:\s*"((?:[^"\\]|\\.)*)"', script)]
    templated = re.findall(r"cmd:\s*`((?:[^`\\]|\\.)*)`", script)
    return quoted + templated

def threads(codex_home):
    """The run's session rollouts: the main thread's path, then each
    sub-agent's as (agent path, rollout path), told apart by session_meta."""
    import glob
    main, subs = None, []
    for f in sorted(glob.glob(f"{codex_home}/sessions/*/*/*/rollout-*.jsonl")):
        try:
            meta = json.loads(open(f).readline()).get("payload", {})
        except ValueError:
            continue
        src = meta.get("source")
        if isinstance(src, dict) and "subagent" in src:
            spawn = src["subagent"].get("thread_spawn", {})
            subs.append((spawn.get("agent_path") or meta.get("id"), f))
        elif main is None:
            main = f
    return main, subs

def records(path, with_time=False):
    """Yields ('assistant', text), ('call', call_id, name, input),
    ('output', call_id, text, truncated) and ('compacted',) in order; with
    with_time, each record is preceded by its timestamp as (ts, record)."""
    for line in open(path):
        try:
            e = json.loads(line)
        except ValueError:
            continue
        for r in _record(e):
            yield (e.get("timestamp", ""), r) if with_time else r

def _record(e):
    """The transcript records one rollout line carries."""
    if e.get("type") == "compacted":
        yield ("compacted",)
        return
    p = e.get("payload")
    if e.get("type") != "response_item" or not isinstance(p, dict):
        return
    t = p.get("type")
    if t == "message" and p.get("role") == "assistant":
        yield ("assistant", "".join(c.get("text", "") for c in p.get("content", [])))
    elif t in ("custom_tool_call", "function_call"):
        yield ("call", p.get("call_id"), p.get("name"), p.get("input") or p.get("arguments") or "")
    elif t in ("custom_tool_call_output", "function_call_output"):
        text, truncated = visible(p.get("output"))
        yield ("output", p.get("call_id"), text, truncated)

# A command writes a file when its shell text redirects into one, edits in
# place, copies or moves, formats in place, applies a patch, or runs a writing
# aiwf verb; or when a heredoc it feeds to an interpreter writes one. Shell
# syntax is judged only outside heredoc bodies, so a `>` inside Python code is
# not a redirect.
_HEREDOC = re.compile(r"<<-?\s*['\"]?(\w+)['\"]?")
_SHELL_WRITE = re.compile(
    r"(?<![0-9&<-])>>?\s*(?!&|/dev/|/tmp/)[\w./$~-]"
    r"|\bsed\s+(?:-\w+\s+)*-i|\bperl\s+(?:-\w+\s+)*-i|\btee\b"
    r"|\b(?:cp|mv|install)\s+(?:-\S+\s+)*\S+\s+(?!/tmp/|/dev/)\S"
    r"|\bgo(?:fmt|fumpt)\s+(?:-\w+\s+)*-w\b|\bgo\s+mod\s+tidy\b|\bgit\s+apply\b|\bapply_patch\b"
    r"|\baiwf\s+(?:add|promote|edit-body|retitle|rename|move|cancel|reallocate|archive|update|init|render\s+roadmap\s+--write)\b")
_CODE_WRITE = re.compile(
    r"\.write_(?:text|bytes)\(|\bopen\([^)]*['\"](?:w|a|wb|ab|w\+|r\+)['\"]|\bos\.replace\(|\bshutil\.(?:copy\w*|move)\(|\bfs\.writeFile")

def is_write(cmd):
    """Whether a shell command, as Codex issued it, writes a file."""
    m = _HEREDOC.search(cmd)
    shell, body = (cmd[: m.start()], cmd[m.end():]) if m else (cmd, "")
    return bool(_SHELL_WRITE.search(shell) or _CODE_WRITE.search(body))
