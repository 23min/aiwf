"""Check that every marked citation names the entity its author meant.

Markers (written by the experiment scripts):
  body prose:   "<ID> (means:<slug>)"         -- the author meant the entity whose slug is <slug>
  title:        "... (<field> means <slug>)"  -- frontmatter <field> was meant to name <slug>

Ids are resolved with aiwf's own loader (`aiwf list --kind K --archived --format=json`),
so a duplicate id shows up as AMBIGUOUS rather than being guessed.
"""
import json, os, re, subprocess, sys

root = os.path.abspath(sys.argv[1])
KINDS = ["epic", "milestone", "gap", "decision", "adr", "contract"]
byid, ents = {}, []
for k in KINDS:
    out = subprocess.run(["aiwf", "list", "--kind", k, "--archived", "--format=json", "--root", root],
                         stdin=subprocess.DEVNULL, capture_output=True, text=True).stdout
    for e in json.loads(out).get("result") or []:
        if "cross_branch_ref" in e or "cross_branch_collision" in e:
            continue  # visible only on another ref, not in this working tree
        byid.setdefault(e["id"], []).append(e)
        ents.append(e)

def slug_of(path):
    parts = path.split("/")
    base = parts[-2] if parts[-1] in ("epic.md", "contract.md") else parts[-1][:-3]
    return re.sub(r"^(ADR|[EMGDC])-\d+-?", "", base)

def resolve(i):
    hits = byid.get(i, [])
    if not hits:
        return "DANGLING", None
    if len(hits) > 1:
        return "AMBIGUOUS", ",".join(slug_of(h["path"]) for h in hits)
    return "ONE", slug_of(hits[0]["path"])

ok = bad = 0
for e in ents:
    text = open(os.path.join(root, e["path"]), encoding="utf-8").read()
    fm, body = text.split("\n---\n", 1) if "\n---\n" in text else (text, "")
    marks = [("body", m.group(1), m.group(2)) for m in re.finditer(r"\b((?:ADR|[EMGDC])-\d+) \(means:([a-z0-9-]+)\)", body)]
    for m in re.finditer(r"\((\w+) means ([a-z0-9-]+)\)", e["title"]):
        field, want = m.group(1), m.group(2)
        fv = re.search(r"^%s:\s*(.*)$" % field, fm, re.M)
        val = fv.group(1).strip() if fv else ""
        ids = re.findall(r"(?:ADR|[EMGDC])-\d+", val)
        if not ids:  # list form on following lines
            blk = re.search(r"^%s:\s*\n((?:\s+- .*\n?)+)" % field, fm + "\n", re.M)
            ids = re.findall(r"(?:ADR|[EMGDC])-\d+", blk.group(1)) if blk else []
        for i in ids:
            marks.append((field, i, want))
    for where, i, want in marks:
        kind, got = resolve(i)
        if kind == "ONE" and got == want:
            ok += 1; verdict = "OK"
        else:
            bad += 1; verdict = "WRONG" if kind == "ONE" else kind
        print(f"  {verdict:9} {e['id']} {where}: {i} meant {want}; resolves to {got}")
print(f"  verify: {ok} correct, {bad} not")
