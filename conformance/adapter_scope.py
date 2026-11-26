"""Connects spec/scope/spine.yaml to the aiwf on PATH, in throwaway git repositories.

It only translates: aiwf decides whether a step is allowed, and the adapter reports what aiwf did.

What aiwf must offer it:
  - the verbs below, exiting 0 when a step succeeds or converges and non-zero when it refuses;
  - `aiwf list --kind milestone --archived --format json`, whose `result` lists the milestone
    in this checkout, done or not, with its `id`; a row for an id found on another branch carries
    `cross_branch_ref` or `cross_branch_collision`, and the adapter skips it, since after a
    reallocate the old id stays on every other branch the milestone is on;
  - `aiwf show <id> --format json`, whose `result` gives the milestone's `status`, its `acs`
    with each `id` and `status`, and its `scopes` with each `state`.

The world new() starts from, built with the aiwf on PATH by the first new() of the adapter's
process, which spine conform starts once per run: one repository where `aiwf init` ran with
`hosts: []`, epic E-0001 active, milestone M-0001 in progress with two ACs whose bodies are
written, and the branch milestone/M-0001-first-milestone checked out. git's identity is
t@example.com, so the human is human/t. git reads no global or system config.

Events (chapter.md, §1), each on the milestone's current id <m>:
  authorize   aiwf authorize <m> --to ai/claude --reason go
  act         aiwf retitle <m> "Retitled <n>" --actor ai/claude --principal human/t
  pause       aiwf authorize <m> --pause hold
  resume      aiwf authorize <m> --resume go
  end         aiwf authorize <m> --end --reason over
  close       aiwf promote <m>/<ac> met for each AC that is open, then aiwf promote <m> done,
              stopping at the first that is refused
  rebase      main gains a commit that touches nothing of the milestone; the branch is rebased
              onto main with --rebase-merges
  copy        the branch is kept as side/copy-<n>; main gains a commit; the branch is rebased
              onto main with --rebase-merges; side/copy-<n> is merged back in
  reallocate  aiwf reallocate <m>
  check       aiwf check
Each first reads the milestone's id with `aiwf list`, and close reads its ACs with `aiwf show`;
a read that fails raises, and conform reports the step as crashed. Otherwise each says "ok" when
every aiwf verb it ran exited 0, and "rejected" otherwise. rebase and copy run git commands, whose
hooks run aiwf; a git command that fails, refused by a hook or not, raises the same way.
--rebase-merges keeps an earlier copy's merge (spec/scope/audit/decisions.md, Q13).

observe() takes the milestone's id from `aiwf list` and its status and scopes from `aiwf show`,
and names:
  none      the milestone is in progress and has no scope
  active    in progress, with exactly one scope that is not ended, and it is active
  paused    in progress, with exactly one scope that is not ended, and it is paused
  ended     in progress, with scopes that are all ended
  closed    the milestone is done, and no scope is active or paused
Anything else is named as aiwf shows it, such as "in_progress; scopes not ended: active, active",
which no spine state matches, so conform reports it.

SCOPE_RUNS names the folder the repositories go in, a new temporary folder when it is unset.
Building the world replaces template/ there, and each new() deletes every run-* folder there.
"""
import json
import os
import re
import shutil
import subprocess
import tempfile
from pathlib import Path

BRANCH = "milestone/M-0001-first-milestone"
AGENT = "ai/claude"
RUNS = Path(os.environ.get("SCOPE_RUNS") or tempfile.mkdtemp(prefix="scope-conform-"))
TEMPLATE = RUNS / "template"
_runs = [0]


def _env():
    RUNS.mkdir(parents=True, exist_ok=True)
    empty = RUNS / "gitconfig"
    empty.touch()
    env = dict(os.environ)
    env.update(GIT_CONFIG_GLOBAL=str(empty), GIT_CONFIG_NOSYSTEM="1")
    return env


ENV = _env()


def run(cwd, *args):
    return subprocess.run(args, cwd=cwd, capture_output=True, text=True, env=ENV)


def must(cwd, *args):
    p = run(cwd, *args)
    if p.returncode != 0:
        raise RuntimeError(f"{' '.join(args)} exited {p.returncode}: {(p.stderr or p.stdout).strip()}")
    return p


def build_template():
    if TEMPLATE.exists():
        shutil.rmtree(TEMPLATE)
    TEMPLATE.mkdir(parents=True)
    t = TEMPLATE
    must(t, "git", "init", "-q", "-b", "main")
    must(t, "git", "config", "user.email", "t@example.com")
    must(t, "git", "config", "user.name", "t")
    (t / "aiwf.yaml").write_text("hosts: []\n")
    must(t, "aiwf", "init")
    must(t, "git", "add", "-A")
    must(t, "git", "commit", "-qm", "init", "--no-verify")
    must(t, "aiwf", "add", "epic", "--title", "Engine")
    must(t, "aiwf", "add", "milestone", "--epic", "E-0001", "--title", "First milestone", "--tdd", "none")
    must(t, "aiwf", "add", "ac", "M-0001", "--title", "one")
    must(t, "aiwf", "add", "ac", "M-0001", "--title", "two")
    must(t, "aiwf", "promote", "E-0001", "active")
    must(t, "git", "checkout", "-qb", "epic/E-0001-engine")
    spec = next(t.glob("work/epics/E-0001-*/M-0001-*.md"))
    text = re.sub(r"(### AC-(\d)[^\n]*\n)", lambda m: f"{m.group(1)}\nBody of AC-{m.group(2)}.\n",
                  spec.read_text())
    spec.write_text(text)
    must(t, "aiwf", "edit-body", "M-0001")
    must(t, "aiwf", "promote", "M-0001", "in_progress")
    must(t, "git", "checkout", "-qb", BRANCH)


def new():
    if _runs[0] == 0:
        build_template()
    for old in RUNS.glob("run-*"):
        shutil.rmtree(old)
    _runs[0] += 1
    w = RUNS / f"run-{_runs[0]}"
    shutil.copytree(TEMPLATE, w, symlinks=True)
    return {"w": w, "n": 0}


def milestone_id(h):
    p = must(h["w"], "aiwf", "list", "--kind", "milestone", "--archived", "--format", "json")
    rows = json.loads(p.stdout)["result"]
    ids = [m["id"] for m in rows if not (m.get("cross_branch_ref") or m.get("cross_branch_collision"))]
    if len(ids) != 1:
        raise RuntimeError(f"expected one milestone, aiwf lists {ids}")
    return ids[0]


def show(h, mid):
    p = must(h["w"], "aiwf", "show", mid, "--format", "json")
    return json.loads(p.stdout)["result"]


def aiwf(h, *args):
    return run(h["w"], "aiwf", *args).returncode == 0


def move_main(h):
    w = h["w"]
    must(w, "git", "checkout", "-q", "main")
    (w / f"other-{h['n']}.txt").write_text(f"main moves {h['n']}\n")
    must(w, "git", "add", f"other-{h['n']}.txt")
    must(w, "git", "commit", "-qm", f"chore: main moves {h['n']}", "--no-verify")
    must(w, "git", "checkout", "-q", BRANCH)


def apply(h, event, env):
    w = h["w"]
    h["n"] += 1
    m = milestone_id(h)
    if event == "authorize":
        ok = aiwf(h, "authorize", m, "--to", AGENT, "--reason", "go")
    elif event == "act":
        ok = aiwf(h, "retitle", m, f"Retitled {h['n']}", "--actor", AGENT, "--principal", "human/t")
    elif event == "pause":
        ok = aiwf(h, "authorize", m, "--pause", "hold")
    elif event == "resume":
        ok = aiwf(h, "authorize", m, "--resume", "go")
    elif event == "end":
        ok = aiwf(h, "authorize", m, "--end", "--reason", "over")
    elif event == "close":
        open_acs = [ac["id"] for ac in show(h, m).get("acs") or [] if ac["status"] == "open"]
        ok = all(aiwf(h, "promote", f"{m}/{ac}", "met") for ac in open_acs)
        ok = ok and aiwf(h, "promote", m, "done")
    elif event == "rebase":
        move_main(h)
        must(w, "git", "rebase", "-q", "--rebase-merges", "main")
        ok = True
    elif event == "copy":
        side = f"side/copy-{h['n']}"
        must(w, "git", "branch", "-f", side)
        move_main(h)
        must(w, "git", "rebase", "-q", "--rebase-merges", "main")
        must(w, "git", "merge", "-q", "--no-ff", "--no-edit", side)
        ok = True
    elif event == "reallocate":
        ok = aiwf(h, "reallocate", m)
    elif event == "check":
        ok = aiwf(h, "check")
    return "ok" if ok else "rejected"


def observe(h):
    d = show(h, milestone_id(h))
    status = d["status"]
    scopes = [s["state"] for s in d.get("scopes") or []]
    live = [s for s in scopes if s != "ended"]
    if status == "done" and not live:
        return "closed"
    if status == "in_progress":
        if not scopes:
            return "none"
        if not live:
            return "ended"
        if len(live) == 1:
            return live[0]
    return f"{status}; scopes not ended: {', '.join(live) or 'none'}"
