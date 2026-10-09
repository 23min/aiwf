"""Shared reading of a Codex session rollout: what the model was shown."""
import json, os, re

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
    concatenated = re.search(r"cmd:\s*[\"'`](?:[^\"'`\\]|\\.)*[\"'`]\s*\+", script)
    if script.count("exec_command(") > len(found) or concatenated:
        found += [_unescape(c) for c in re.findall(r'"((?:[^"\\]|\\.)*)"', script)]
        found += [_unescape(c) for c in re.findall(r"'((?:[^'\\]|\\.)*)'", script)]
        found += re.findall(r"`((?:[^`\\]|\\.)*)`", script)
    return found

def run_path(run):
    """The absolute path a run ran at, as run.sh recorded it, so records read
    the same after they move; where none is recorded, where they are now."""
    p = os.path.join(run, "run-path.txt")
    return open(p).read().strip() if os.path.exists(p) else os.path.abspath(run)

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

# A command writes a file when it redirects into one, edits one in place,
# copies, moves or installs onto one, formats one in place, applies a patch,
# or runs a writing aiwf verb; or when a heredoc it feeds to an interpreter
# writes one. The shell text is split into words the way the shell would, so
# a quoted pattern or one-liner is a word rather than syntax, and a quoted
# target is judged by its text. A write counts only inside the run's
# repository, at absolute path repo when given; a relative path is resolved
# against a `cd` to an absolute directory earlier in the same command, and
# is otherwise inside it; a `cd` inside a subshell holds only until its `)`.
# A variable the command itself sets to an absolute path, or to a `mktemp`
# result under the temporary directory, stands for that path. Any other
# variable, such as $PWD, a loop variable or a command substitution, is read
# as a relative path. A help request runs no aiwf verb.
import shlex

_HEREDOC = re.compile(r"<<-?\s*(['\"]?)(\w+)\1")
_CODE_WRITE = re.compile(
    r"\.write_(?:text|bytes)\(|\bopen\([^)]*['\"](?:w|a|wb|ab|w\+|r\+|x)['\"]|\bos\.replace\(|\bshutil\.(?:copy\w*|move)\(|\bfs\.writeFile")
_AIWF_WRITERS = {"add", "promote", "edit-body", "retitle", "rename", "move", "cancel", "reallocate", "archive", "update", "init"}
_SEPARATORS = {";", "&&", "||", "|", "&", "|&", "(", ")"}
_REDIRECTS = {">", ">>", ">|", "&>", "&>>"}

_ASSIGN = re.compile(r"(?:^|[;&|\s(])(?:export\s+)?([A-Za-z_]\w*)=(\$\(mktemp[^)]*\)|\"[^\"]*\"|'[^']*'|[^\s;&|)]+)")
_VAR = re.compile(r"^\$(?:\{(\w+)\}|(\w+))")

def _assignments(shell):
    """The absolute paths the command's own variable assignments name."""
    env = {}
    for name, value in _ASSIGN.findall(shell):
        value = value.strip("'\"")
        if value.startswith("$(mktemp"):
            if not re.search(r"\s(?:-p|--tmpdir)\b", value):
                env[name] = "/tmp/mktemp"
            continue
        value = _expand(value, env)
        if value.startswith("/"):
            env[name] = value
    return env

def _expand(target, env):
    """target with a leading variable the command sets replaced by its value."""
    m = _VAR.match(target)
    name = m and (m.group(1) or m.group(2))
    return env[name] + target[m.end():] if name in env else target

def _inside(target, repo, cwd=None, env=None):
    target = _expand(target.strip("'\""), env or {})
    if not target or target.startswith(("/dev/", "&")) or target.isdigit():
        return False
    if target.startswith(("~", "$HOME", "${HOME}", "$TMPDIR", "${TMPDIR}")):
        return False
    if not target.startswith("/"):
        if not cwd:
            return True
        target = os.path.normpath(os.path.join(cwd, target))
    return bool(repo) and (target + "/").startswith(repo.rstrip("/") + "/")

def _split_heredocs(cmd):
    shell, bodies, rest = [], [], cmd
    while True:
        m = _HEREDOC.search(rest)
        if not m:
            shell.append(rest)
            return "\n".join(shell), bodies
        line_end = rest.find("\n", m.end())
        line_end = len(rest) if line_end < 0 else line_end
        shell.append(rest[: m.start()] + rest[m.end():line_end])
        end = re.search(r"^\s*" + re.escape(m.group(2)) + r"\s*$", rest[line_end:], re.M)
        body_end = line_end + (end.start() if end else len(rest) - line_end)
        bodies.append(rest[line_end:body_end])
        rest = rest[line_end + (end.end() if end else len(rest) - line_end):]

def _commands(shell):
    """The simple commands in shell text, each as (words, redirect targets)."""
    lx = shlex.shlex(shell.replace("\n", " ; "), posix=True, punctuation_chars=True)
    lx.whitespace_split = True
    tokens = list(lx)
    out, words, targets, i = [], [], [], 0
    while i < len(tokens):
        t = tokens[i]
        if t in _SEPARATORS:
            out.append((words, targets))
            words, targets = [], []
            if t in ("(", ")"):
                out.append(([t], []))
        elif t in _REDIRECTS:
            if i + 1 < len(tokens):
                targets.append(tokens[i + 1])
            i += 1
        elif t in ("<", "<<<", ">&", "<&"):
            i += 1
        else:
            words.append(t)
        i += 1
    out.append((words, targets))
    return [c for c in out if c[0] or c[1]]

def _command_writes(words, targets, repo, cwd, env=None):
    inside = lambda t: _inside(t, repo, cwd, env)
    if any(inside(t) for t in targets):
        return True
    while words and re.match(r"^\w+=", words[0]):
        words = words[1:]
    if not words:
        return False
    name, args = os.path.basename(words[0]), words[1:]
    operands = [a for a in args if not a.startswith("-")]
    if name == "aiwf":
        if "--help" in args or "-h" in args or not args:
            return False
        if args[0] in _AIWF_WRITERS:
            return True
        return args[:2] == ["render", "roadmap"] and "--write" in args
    if name in ("sed", "perl"):
        inplace = any(a == "--in-place" or a.startswith("--in-place=") or (a.startswith("-") and not a.startswith("--") and "i" in a[1:]) for a in args)
        return inplace and any(inside(o) for o in operands[1:] or operands)
    if name == "tee":
        return any(inside(o) for o in operands)
    if name in ("cp", "mv", "install"):
        return len(operands) >= 2 and inside(operands[-1])
    if name in ("gofmt", "gofumpt") and "-w" in args:
        return any(inside(o) for o in operands) or not operands
    if name == "go" and args[:2] == ["mod", "tidy"]:
        return True
    if name == "git" and args[:1] == ["apply"]:
        return True
    return name == "apply_patch"

def is_write(cmd, repo=None):
    """Whether a shell command writes a file in the run's repository."""
    shell, bodies = _split_heredocs(cmd)
    try:
        commands_ = _commands(shell)
    except ValueError:
        # Text the shell would not parse, such as an unbalanced quote, is
        # judged by its plain words.
        commands_ = [(shell.split(), [])]
    cwd, env, outer = None, _assignments(shell), []
    for words, targets in commands_:
        if words == ["("]:
            outer.append(cwd)
        elif words == [")"] and outer:
            cwd = outer.pop()
        if words[:1] == ["cd"] and len(words) == 2 and _expand(words[1], env).startswith("/"):
            cwd = _expand(words[1], env)
        if _command_writes(words, targets, repo, cwd, env):
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
        "tee -a notes.md": True, 'echo x > "/tmp/x"': False, 'echo x > "$TMPDIR/x"': False, 'cp a "$TMPDIR/b"': False,
        "du -h internal > sizes.txt": True, "cmd &> out.log": True, "echo x >| f": True, "sed --in-place s/a/b/ f": True,
        "sed -i s/a/b/ /tmp/f": False, "gofmt -w /tmp/x.go": False, "cd /tmp/x && printf y > ../ac.md": False,
        "J=/tmp/tmp.tjKoBLrYNB; A=$J/aiwf; rm -rf $J/r3 && mkdir -p $J/r3/main && cd $J/r3/main && $A add milestone --epic E-0001 --body-file $J/ms.md >/dev/null && printf 'probe body\\n' > ../ac.md": False,
        "cat scripts/guidance-audit.py && D=$(mktemp -d) && make check-fast > $D/cf.log 2>&1; echo \"exit=$?\"": False,
        "echo x > $R/aiwf.yaml": True, "echo x > $PWD/f.go": True, 'echo x > "$(pwd)/f.go"': True,
        "W=$(pwd); echo x > $W/f.go": True, "W=$(git rev-parse --show-toplevel); echo x > $W/f.go": True,
        "for f in a.go; do echo x > $f; done": True, 'while read f; do printf y > "$f"; done < list.txt': True,
        "T=$(mktemp -d); (cd $T && git init -q) > init.log": True, "echo > ${W:-x}/f": True, "T=$(mktemp -d -p .); echo > $T/f": True, "W=" + repo + "/internal; echo y > $W/a.go": True, 'echo y > "${HOME}/x"': False,
        "cat <<EOF > /tmp/x\nhi\nEOF": False, "cd " + repo + "/internal && echo y > a.go": True, '"$(go env GOBIN)/aiwf" update --wire-settings </dev/null': True,
        '"$(go env GOBIN)/aiwf" update --help': False, "/opt/bin/aiwf promote M-0001 done": True, "printf 'probe\\n' >> README.md": True, "x=$(cat a) && echo $x": False,
    }
    bad = [c for c, want in writes.items() if is_write(c, repo) != want]
    cmds = {
        'text(await tools.exec_command({cmd:"cat CLAUDE.md"}))': ["cat CLAUDE.md"],
        "text(await tools.exec_command({cmd:'sed -n 1,9p CLAUDE.md'}))": ["sed -n 1,9p CLAUDE.md"],
        'for (const cmd of ["cat a.md","cat b.md"]) out += (await tools.exec_command({cmd})).output;': ["cat a.md", "cat b.md"],
        'const f="CLAUDE.md"; text(await tools.exec_command({cmd:"sed -n 1,400p "+f}))': ["CLAUDE.md"],
    }
    bad += [s for s, want in cmds.items() if not all(w in commands(s) for w in want)]
    if bad:
        raise SystemExit("codexrollout selftest failed: " + repr(bad))
    print("codexrollout selftest: ok")

if __name__ == "__main__":
    selftest()
