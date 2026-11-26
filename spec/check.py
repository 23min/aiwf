"""Checks a machine's records under spec/<machine>/ and conformance/ against what E-0100's milestones
ask of them.

Run from the repository root, with the verifier-kit checkout spec/README.md builds:

    python3 spec/check.py <machine> "$VK"

It prints one line per check and exits 1 when any fails.

  audit     Every answer in audit/decisions.md fills in its date, answer, reason, the sources it
            makes wrong and the claims it settles. audit/docs.md exists, and so does
            audit/history.md where an answer settles one of its H claims; each of the two that
            exists has a Conflicts, a Not covered and a Names section. Every claim id a Conflicts
            or Not covered section cites is among the claims settled. Each Names section has at
            least one part, a ### heading or a bold line, and each part is named in one answer's
            claims settled as its file, section and part, followed by `,`, `.`, `;`, `)`,
            " and `" or the end of that field ("`docs.md` §4 States,"). Each (state, event) pair a
            Not covered section lists on a line starting "- " or "<n>. " is named, as
            "`state, event`", in an answer's claims settled or in the section on cases no source
            covers. An id, part or pair named anywhere in a claims settled field counts as
            settled by that answer.
  readings  The two readings name different models. Each header gives the hash of a version of
            chapter.md in this repository's history, of readings/prompt.md, and of the checkout's
            docs/reference/spine-format.md. Every case that `spine diff` of the two readings or
            `spine check` of either reports is named, as state/event, by an answer in answers.md
            that the chapter cites as "(answers.md, Q<n>".
  spine     `spine check` of spine.yaml passes; `spine diff` of it against each reading reports
            only those cases; `spine draw` reports no error or warning and writes drawing.svg
            byte for byte; the header names readings/ and answers.md and says it is ratified by the
            reviewed merge of the file.
  conform   adapter_<machine>.cmd is one line that names conformance/adapter_<machine>.py, and it
            is the --adapter-cmd conform.md records. conform.md records the checkout's
            verifier-kit version and commit, the aiwf version, the run time, a margin of 2 or
            more, the same in the command and in the run's coverage line, and the git hash-object
            of the spine and adapter in the checkout. The cases of the run's error lines are
            exactly the cases of the classification rows, and each row cites a gap.

`spine diff` reports each case as case-disagrees, and each behavior-differs line it prints is the
shortest witness of a case reported so; `spine check` reports a case as undecided-cell or
missing-case. Any other finding code is reported as one the check does not know.
"""
import re
import subprocess
import sys
from pathlib import Path

CASE_CODES = {"case-disagrees", "undecided-cell", "missing-case"}
WITNESS_CODES = {"behavior-differs"}
failures = []


def report(ok, what, detail=""):
    print(("ok: " if ok else "FAIL: ") + what + ("" if ok or not detail else f": {detail}"))
    if not ok:
        failures.append(what)


def git(*args):
    return subprocess.run(["git", *args], capture_output=True, text=True, check=True).stdout.strip()


def sections(path):
    """A markdown file's level-2 sections, by heading."""
    parts = re.split(r"(?m)^## ", path.read_text(encoding="utf-8"))[1:]
    return {p.split("\n", 1)[0].strip(): p for p in parts}


def numbered(path, title):
    """The number and text of a section headed "<n>. <title>", or (None, "") when there is none."""
    for heading, text in sections(path).items():
        m = re.fullmatch(rf"(\d+)\. {re.escape(title)}", heading)
        if m:
            return m.group(1), text
    return None, ""


def claim_ids(text):
    out = set()
    for m in re.finditer(r"\b([CH])(\d+)(?:\s*[–-]\s*\1?(\d+))?\b", text):
        kind, first, last = m.group(1), int(m.group(2)), m.group(3)
        out.update(f"{kind}{n}" for n in range(first, int(last or first) + 1))
    return out


def names(spine_text):
    states = re.findall(r"(?m)^  ([a-z_]+):", spine_text.split("\nevents:")[0].split("\nstates:")[1])
    events = re.search(r"(?m)^events: \[(.*)\]", spine_text).group(1).replace(" ", "").split(",")
    return states, events


def spine_cases(spine, *args):
    """The cases a `spine diff` or `spine check` reports, and any finding code not known here."""
    out = subprocess.run([spine, *args], capture_output=True, text=True).stdout
    cases, unknown = set(), []
    for line in out.splitlines():
        fields = line.split()
        if len(fields) < 3 or fields[0] not in ("error", "warning"):
            continue
        if fields[1] in CASE_CODES:
            cases.add(fields[2].rstrip(":"))
        elif fields[1] not in WITNESS_CODES:
            unknown.append(line.strip())
    return cases, unknown


def check_audit(root, states, events):
    audit = root / "audit"
    dec = sections(audit / "decisions.md")
    answers = {h.split(".", 1)[0]: t for h, t in dec.items() if re.match(r"Q\d+\. ", h)}
    cases_text = next((t for h, t in dec.items() if h.startswith("Cases no source covers")), "")
    fields = (("date", r"\*\*Date:\*\*"), ("answer", r"\*\*Answer:\*\*"), ("reason", r"\*\*Reason:\*\*"),
              ("sources", r"\*\*Sources this makes wrong[^*]*:\*\*"), ("claims settled", r"\*\*Claims settled:\*\*"))
    missing = [f"{q} has no {name}" for q, t in answers.items() for name, pat in fields
               if not re.search(pat + r"(?:[ \t]*\S|[ \t]*\n[ \t]+\S)", t)]
    report(answers and not missing, f"audit: each of {len(answers)} answers gives every field", "; ".join(missing))

    settled_text = {q: (re.search(r"\*\*Claims settled:\*\*(.*?)(?=\n- \*\*|\Z)", t, re.S) or [None, ""])[1]
                    for q, t in answers.items()}
    settled = set().union(set(), *(claim_ids(t) for t in settled_text.values()))
    history = audit / "history.md"
    files = [audit / "docs.md"] + ([history] if history.exists() or any(i[0] == "H" for i in settled) else [])
    absent = [f"{p.name} {t}" for p in files for t in ("Conflicts", "Not covered", "Names that don't fit")
              if not p.exists() or numbered(p, t)[0] is None]
    report(not absent, "audit: docs.md, and history.md where it exists or an answer settles an H claim, have "
           "their Conflicts, Not covered and Names sections", ", ".join(absent))
    files = [p for p in files if p.exists()]
    raised = set().union(set(), *(claim_ids(numbered(p, t)[1]) for p in files for t in ("Conflicts", "Not covered")))
    unsettled = sorted(raised - settled, key=lambda i: (i[0], int(i[1:])))
    report(not unsettled, f"audit: each of {len(raised)} claim ids raised is settled", ", ".join(unsettled))

    claimed = [" ".join(t.split()) for t in settled_text.values()]
    for p in files:
        n, text = numbered(p, "Names that don't fit")
        parts = [a or b for a, b in re.findall(r"(?m)^(?:### (.+?)|\*\*(.+?)\*\*)\s*$", text)]
        open_parts = [s for s in parts
                      if not any(re.search(rf"`{re.escape(p.name)}` §{n} {re.escape(s)}(?=[,.;)]| and `|$)", f)
                                 for f in claimed)]
        report(n and parts and not open_parts,
               f"audit: each of {len(parts)} parts of {p.name} §{n} (names) is named in an answer's claims settled",
               ", ".join(open_parts))

    pair = re.compile(rf"`?\b({'|'.join(states)})`?,\s*`?({'|'.join(events)})\b`?")
    listed = {m.groups() for p in files for line in numbered(p, "Not covered")[1].splitlines()
              if re.match(r"(?:- |\d+\. )", line) for m in pair.finditer(line)}
    answered = "\n".join(settled_text.values()) + cases_text
    open_pairs = sorted(f"{s}, {e}" for s, e in listed if f"`{s}, {e}`" not in answered)
    report(listed and not open_pairs, f"audit: each of {len(listed)} not-covered pairs is answered", "; ".join(open_pairs))


def check_readings(root, spine, vk):
    r = root / "readings"
    heads = {x: "".join(l for l in (r / f"reading-{x}.yaml").read_text().splitlines(True) if l.startswith("#"))
             for x in "ab"}
    models = {x: (re.search(r"by ([\w.-]+), \d{4}-\d{2}-\d{2}", h) or [None, None])[1] for x, h in heads.items()}
    report(all(models.values()) and models["a"] != models["b"],
           f"readings: by two different models, {models['a']} and {models['b']}")

    chapter = f"{root}/chapter.md"
    versions = {git("rev-parse", f"{c}:{chapter}"): c[:9]
                for c in git("log", "--format=%H", "--", chapter).split()}
    prompt = git("hash-object", str(r / "prompt.md"))
    fmt = git("-C", str(vk), "rev-parse", "HEAD:docs/reference/spine-format.md")
    for x, h in heads.items():
        cited = (re.search(r"chapter\.md ([0-9a-f]{40})", h) or [None, None])[1]
        report(cited in versions, f"readings: reading-{x} cites chapter.md as of {versions.get(cited, 'no commit')}")
        report(f"readings/prompt.md {prompt}" in h, f"readings: reading-{x} cites readings/prompt.md in the checkout")
        report(f"spine-format.md {fmt}" in h, f"readings: reading-{x} cites the checkout's spine-format.md")

    found, unknown = set(), []
    for args in (("diff", r / "reading-a.yaml", r / "reading-b.yaml"),
                 ("check", r / "reading-a.yaml"), ("check", r / "reading-b.yaml")):
        cases, odd = spine_cases(spine, *map(str, args))
        found |= cases
        unknown += odd
    report(not unknown, "readings: spine reports no finding the check does not know", "; ".join(unknown))

    text = (root / "answers.md").read_text(encoding="utf-8")
    entries = dict(re.findall(r"(?ms)^- \*\*(Q\d+)\.(.*?)(?=^- \*\*Q\d+\.|\Z)", text))
    cited = set(re.findall(r"\(answers\.md,\s+(Q\d+)", (root / "chapter.md").read_text(encoding="utf-8")))
    unanswered = sorted(c for c in found
                        if not any(re.search(rf"(?<![\w/]){re.escape(c)}(?![\w/])", t) and q in cited
                                   for q, t in entries.items()))
    report(found and not unanswered,
           f"readings: each of {len(found)} cases they split or leave open is answered and cited by the chapter",
           ", ".join(unanswered))
    return found


def check_spine(root, spine, decided):
    p = subprocess.run([spine, "check", str(root / "spine.yaml")], capture_output=True, text=True)
    report(p.returncode == 0, "spine: spine check passes", p.stdout.strip())
    for x in "ab":
        cases, unknown = spine_cases(spine, "diff", str(root / "spine.yaml"), str(root / f"readings/reading-{x}.yaml"))
        extra = sorted(cases - decided) + unknown
        report(not extra, f"spine: against reading-{x}, only cases the readings split or leave open", ", ".join(extra))
    d = subprocess.run([spine, "draw", "spine.yaml", "drawing.yaml"], capture_output=True, cwd=root)
    report(d.returncode == 0 and not d.stderr, "spine: spine draw reports no error or warning",
           (d.stderr or (d.stdout if d.returncode else b"")).decode().strip())
    report(d.stdout == (root / "drawing.svg").read_bytes(), "spine: drawing.svg is what spine draw writes")
    head = " ".join(l for l in (root / "spine.yaml").read_text().splitlines() if l.startswith("#"))
    report(re.search(r"readings/.*answers\.md.*Ratified by the reviewed merge of this file\.", head) is not None,
           "spine: the header names readings/ and answers.md and the ratifying merge")


def check_conform(root, machine, vk):
    adapter, cmd = Path(f"conformance/adapter_{machine}.py"), Path(f"conformance/adapter_{machine}.cmd")
    text = (root / "conform.md").read_text(encoding="utf-8")
    lines = cmd.read_text().splitlines() if cmd.exists() else []
    recorded = (re.search(r'--adapter-cmd "([^"]+)"', text) or [None, None])[1]
    report(adapter.exists() and len(lines) == 1 and str(adapter) in lines[0].split() and lines[0] == recorded,
           f"conform: {cmd} is one line that names {adapter}, the one conform.md records", f"{lines} vs {recorded!r}")
    vk_run = re.search(r"\*\*verifier-kit:\*\* (v\d+\.\d+\.\d+) at `([0-9a-f]{7,40})`", text)
    tag = subprocess.run(["git", "-C", str(vk), "describe", "--tags", "--exact-match"], capture_output=True, text=True)
    report(vk_run is not None and vk_run[1] == tag.stdout.strip()
           and git("-C", str(vk), "rev-parse", "HEAD").startswith(vk_run[2]),
           "conform: conform.md records the checkout's verifier-kit version and commit")
    for what, pat in (("the aiwf version", r"\*\*aiwf:\*\* `[^`]+`"), ("the run time", r"run time \d+ seconds")):
        report(re.search(pat, text) is not None, f"conform: conform.md records {what}")
    block = re.search(r"(?ms)^```\n(.*?)^```", text)
    margin = re.search(r"--margin (\d+)", text)
    covered = re.search(r"(?m)^info +coverage +.*margin (\d+):", block.group(1) if block else "")
    report(margin is not None and covered is not None and margin[1] == covered[1] and int(margin[1]) >= 2,
           f"conform: margin {margin and margin[1]} in the command and {covered and covered[1]} in the run, 2 or more")
    for path in (root / "spine.yaml", adapter):
        report(f"`{path}` `{git('hash-object', str(path))}`" in text, f"conform: conform.md records the hash of the {path} in the checkout")
    run = sorted(l.split()[2].rstrip(":") for l in (block.group(1) if block else "").splitlines()
                 if l.startswith("error"))
    rows = [l.split("|") for l in text.splitlines() if re.match(r"\| [a-z_]+/[a-z_]+", l)]
    uncited = [r[1].strip() for r in rows if not re.search(r"G-\d{4}", r[-2])]
    table = sorted(c.strip() for r in rows for c in r[1].split(","))
    report(run and run == table and not uncited,
           f"conform: each of {len(run)} errors in the run is classified, each row citing a gap",
           f"run {run}, table {table}, rows citing no gap {uncited}")


def main(machine, vk):
    root, vk = Path("spec") / machine, Path(vk)
    spine = str(vk / "tools" / "bin" / "spine")
    states, events = names((root / "spine.yaml").read_text())
    check_audit(root, states, events)
    decided = check_readings(root, spine, vk)
    check_spine(root, spine, decided)
    check_conform(root, machine, vk)
    print(f"{len(failures)} failed" if failures else "all passed")
    return 1 if failures else 0


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    sys.exit(main(*sys.argv[1:]))
