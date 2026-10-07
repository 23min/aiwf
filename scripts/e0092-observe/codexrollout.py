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
    """The shell commands an exec script runs: each literal `cmd:` value, in
    any of JavaScript's three quote forms. A script that runs a command whose
    text is not a literal there, such as one built in a loop or by
    concatenation, also yields every string literal it holds, so a file the
    loop names still counts."""
    found = [_unescape(c) for c in re.findall(r'cmd:\s*"((?:[^"\\]|\\.)*)"', script)]
    found += [_unescape(c) for c in re.findall(r"cmd:\s*'((?:[^'\\]|\\.)*)'", script)]
    found += re.findall(r"cmd:\s*`((?:[^`\\]|\\.)*)`", script)
    if script.count("exec_command(") > len(found):
        found += [_unescape(c) for c in re.findall(r'"((?:[^"\\]|\\.)*)"', script)]
        found += [_unescape(c) for c in re.findall(r"'((?:[^'\\]|\\.)*)'", script)]
        found += re.findall(r"`((?:[^`\\]|\\.)*)`", script)
    return found

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
# aiwf verb; or when a heredoc it feeds to an interpreter writes one. Each
# shell segment, split at `;`, `&&`, `||`, `|` and newlines, is judged on its
# own with quoted text masked, so a pattern a grep searches for or code a
# one-liner runs is not shell syntax; a segment asking for help writes
# nothing. A heredoc's body is judged as the program it feeds. A path
# outside the run's repository is not a write the rubric counts, except that
# a relative path is always inside it.
_HEREDOC = re.compile(r"<<-?\s*(['\"]?)(\w+)\1")
_QUOTED = re.compile(r"'[^']*'|\"(?:[^\"\\]|\\.)*\"")
_SEGMENT = re.compile(r"&&|\|\||;|\||\n")
_TARGET = r"(?:Q|[^\s;&|<>()]+)"
_REDIRECT = re.compile(r"(?<![<=&-])(?:\d)?>>?\s*(?!&)(" + _TARGET + r")")
_INPLACE = re.compile(r"\b(?:sed|perl)\s+(?:-\S+\s+)*-\w*i|\btee\s+(?:-\S+\s+)*(" + _TARGET + r")|\bgo(?:fmt|fumpt)\s+(?:-\w+\s+)*-w\b|\bgo\s+mod\s+tidy\b|\bgit\s+apply\b|\bapply_patch\b")
_COPY = re.compile(r"(?:^|\s)(?:cp|mv|install)\s")
_AIWF = re.compile(r"\baiwf\s+(?:add|promote|edit-body|retitle|rename|move|cancel|reallocate|archive|update|init|render\s+roadmap\s+--write)\b")
_HELP = re.compile(r"(?:^|\s)(?:--help|-h)(?:\s|$)")
_CODE_WRITE = re.compile(
    r"\.write_(?:text|bytes)\(|\bopen\([^)]*['\"](?:w|a|wb|ab|w\+|r\+|x)['\"]|\bos\.replace\(|\bshutil\.(?:copy\w*|move)\(|\bfs\.writeFile")

def _inside(target, repo):
    target = target.strip("'\"")
    if target in ("", "Q"):
        return True
    if target.startswith(("/dev/", "&")):
        return False
    if target.startswith("/"):
        return bool(repo) and target.startswith(repo.rstrip("/") + "/")
    return not target.startswith(("~", "$HOME", "$TMPDIR"))

def _segment_writes(seg, repo):
    if _HELP.search(seg):
        return False
    for m in _REDIRECT.finditer(seg):
        if _inside(m.group(1), repo):
            return True
    m = _INPLACE.search(seg)
    if m and (m.group(1) is None or _inside(m.group(1), repo)):
        return True
    if _COPY.search(" " + seg.strip()):
        words = seg.split()
        if len(words) >= 3 and _inside(words[-1], repo):
            return True
    return bool(_AIWF.search(seg))

def is_write(cmd, repo=None):
    """Whether a shell command writes a file in the run's repository, at
    absolute path repo when given."""
    shell, bodies, rest = [], [], cmd
    while True:
        m = _HEREDOC.search(rest)
        if not m:
            shell.append(rest)
            break
        line_end = rest.find("\n", m.end())
        line_end = len(rest) if line_end < 0 else line_end
        shell.append(rest[: m.start()] + rest[m.end():line_end])
        end = re.search(r"^\s*" + re.escape(m.group(2)) + r"\s*$", rest[line_end:], re.M)
        body_end = line_end + (end.start() if end else len(rest) - line_end)
        bodies.append(rest[line_end:body_end])
        rest = rest[line_end + (end.end() if end else len(rest) - line_end):]
    # A quoted path to the aiwf binary is still aiwf, so its verb is judged.
    masked = _QUOTED.sub(lambda m: "aiwf" if re.search(r"(?:^|/)aiwf$", m.group(0)[1:-1]) else "Q", "\n".join(shell))
    if any(_segment_writes(seg, repo) for seg in _SEGMENT.split(masked)):
        return True
    return any(_CODE_WRITE.search(b) for b in bodies)


def selftest():
    """The detection cases the rubric's definitions turn on. Run before a round."""
    repo = "/tmp/r/T1/x-1/repo"
    writes = {
        "echo x >> README.md": True, "git diff --cached > .git-x": True, "go test ./... 2>&1 | tail": False,
        "cat CLAUDE.md": False, "sed -n 1,20p CLAUDE.md": False, "sed -i s/a/b/ f": True, "perl -pi -e 's/a/b/' f": True,
        "cat > a.go <<EOF\npackage a\nEOF": True, "cat <<'EOF' > f.go\npackage a\nEOF": True,
        "python3 - <<'PY'\nif a > b: print(1)\nPY": False, "python3 - <<'PY'\nPath('a').write_text(x)\nPY": True,
        "gofumpt -w x.go": True, "aiwf promote M-0001/AC-1 met": True, "aiwf show M-0001": False,
        "aiwf reallocate --help; git branch --show-current": False, "aiwf update -h": False,
        "grep -n \"aiwf update\" README.md": False, "node -e \"x => y\"": False, "echo x > \"out.txt\"": True,
        "go test ./x 2> err.log": True, "ls > /tmp/x": False, "cmd 2>&1 >/dev/null": False, "cp a /tmp/b": False,
        "cp a b": True, "install -m 755 bin/x " + repo + "/bin/x": True, "go test -coverprofile=x " + repo + "/c.out > " + repo + "/c.out": True,
        "make ci > /tmp/ci.log 2>&1": False, "aiwf render roadmap": False, "aiwf render roadmap --write": True,
        "tee -a notes.md": True, '"$(go env GOBIN)/aiwf" update --wire-settings </dev/null': True,
        '"$(go env GOBIN)/aiwf" update --help': False, "/opt/bin/aiwf promote M-0001 done": True, "printf 'probe\\n' >> README.md": True, "x=$(cat a) && echo $x": False,
    }
    bad = [c for c, want in writes.items() if is_write(c, repo) != want]
    cmds = {
        'text(await tools.exec_command({cmd:"cat CLAUDE.md"}))': ["cat CLAUDE.md"],
        "text(await tools.exec_command({cmd:'sed -n 1,9p CLAUDE.md'}))": ["sed -n 1,9p CLAUDE.md"],
        'for (const cmd of ["cat a.md","cat b.md"]) out += (await tools.exec_command({cmd})).output;': ["cat a.md", "cat b.md"],
    }
    bad += [s for s, want in cmds.items() if not all(w in commands(s) for w in want)]
    if bad:
        raise SystemExit("codexrollout selftest failed: " + repr(bad))
    print("codexrollout selftest: ok")

if __name__ == "__main__":
    selftest()
