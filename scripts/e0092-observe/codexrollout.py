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
    return [_unescape(c) for c in re.findall(r'cmd:\s*"((?:[^"\\]|\\.)*)"', script)]

def records(path):
    """Yields ('assistant', text), ('call', call_id, name, input),
    ('output', call_id, text, truncated) and ('compacted',) in order."""
    for line in open(path):
        try:
            e = json.loads(line)
        except ValueError:
            continue
        if e.get("type") == "compacted":
            yield ("compacted",)
            continue
        p = e.get("payload")
        if e.get("type") != "response_item" or not isinstance(p, dict):
            continue
        t = p.get("type")
        if t == "message" and p.get("role") == "assistant":
            yield ("assistant", "".join(c.get("text", "") for c in p.get("content", [])))
        elif t in ("custom_tool_call", "function_call"):
            yield ("call", p.get("call_id"), p.get("name"), p.get("input") or p.get("arguments") or "")
        elif t in ("custom_tool_call_output", "function_call_output"):
            text, truncated = visible(p.get("output"))
            yield ("output", p.get("call_id"), text, truncated)
