"""Breakdowns behind section 4.5 of fewer-planning-commits.md.

Reads the --dump files ordering_variants.py writes and data.json. Run after it.
"""
import ast, collections, contextlib, io, json, re, subprocess, sys
sys.path.insert(0, ".")
with contextlib.redirect_stdout(io.StringIO()):
    import base as B

def load(name):
    d = json.load(open(name)); return d["run"], dict((h, r) for h, r in d["starts"])
def runs_of(RUN):
    r = collections.defaultdict(list)
    for h in B.ORDER:
        if h in RUN: r[RUN[h]].append(h)
    return r
def parent(h):
    p = B.C[h]["parents"]; p = ast.literal_eval(p) if isinstance(p, str) else p; return p[0]
def verb(h): return (B.verbs(h) or ["?"])[0]
def is_phase(h): return verb(h) == "promote" and " --phase " in B.C[h]["subject"]

print("== 1. Why runs start (baseline, without ROADMAP), first reason the replay records")
RUN, ST = load("variant_baseline.json"); RUNS = runs_of(RUN)
c = collections.Counter(r[0] if r else "?" for r in ST.values())
for k, n in c.most_common(): print(f"  {n:4d}  {k}")
print(f"  {sum(c.values()):4d}  total")

print("\n== 2. Runs started because HEAD is a code commit (baseline)")
code_runs = [st for st, r in ST.items() if r and r[0] == "code commit"]
edit_only = [st for st in code_runs if all(verb(h) == "edit-body" for h in RUNS[st])]
with_phase = [st for st in code_runs if any(is_phase(h) for h in RUNS[st])]
print(f"  {len(code_runs)} runs; only edit-body: {len(edit_only)}; containing a TDD --phase step: {len(with_phase)}")

print("\n== 3. Runs started because HEAD is a merge (baseline): first operation")
merge_runs = [st for st, r in ST.items() if r and r[0] == "merge"]
first = collections.Counter(re.sub(r"[A-Z]+-\d{4}(/AC-\d+)?", "X", B.C[st]["subject"])[:44] for st in merge_runs)
for k, n in first.most_common(5): print(f"  {n:4d}  {k}")
gap = [st for st in merge_runs if re.match(r"aiwf promote G-\d{4} open -> addressed", B.C[st]["subject"])]
after = collections.Counter("patch merge" if B.C[parent(st)]["subject"].startswith("Merge patch/") else
                            "milestone wrap merge" if B.C[parent(st)]["subject"].startswith("chore(milestone)") else "other"
                            for st in gap)
print(f"  gap closures right after a merge: {len(gap)} {dict(after)}")
done = [st for st in merge_runs if re.match(r"aiwf promote M-\d{4} in_progress -> done", B.C[st]["subject"])]
nxt = sum(1 for st in done if any("draft -> in_progress" in B.C[h]["subject"] for h in RUNS[st]))
print(f"  runs opening with a milestone's promote to done: {len(done)}; of which also start the next milestone: {nxt}")

print("\n== 4. Spec bookkeeping deferred (variant: both ritual changes)")
RUN2, ST2 = load("variant_both.json"); RUNS2 = runs_of(RUN2)
def deferrable(st):
    lid = B.LINE[st]; line = B.LINES[lid]; i = line.index(RUNS2[st][-1])
    for h in line[i + 1:]:
        if B.is_merge(h): return False
        if h in ST2:
            return not (lid == "main" and B.PUSHIDX.get(h) != B.PUSHIDX.get(st))
    return False
eo2 = [st for st, r in ST2.items() if r and r[0] == "code commit" and all(verb(h) == "edit-body" for h in RUNS2[st])]
mov = [st for st in eo2 if deferrable(st)]
left = len(set(RUN2.values())) 
print(f"  edit-body-only runs after a code commit: {len(eo2)}; could join the next run on their line: {len(mov)}")
print(f"  planning commits left if every one of those joins: 528 - {len(mov)} = {528 - len(mov)} (fewest possible under this variant)")

print("\n== 5. Code commits made while a verb's write would be pending under batch (pending clears at a merge)")
def exposure(lines):
    n = 0
    for seq in lines:
        pending = False
        for h in seq:
            k = B.klass(h)
            if k == "kernel": pending = True  # a verb's write that batch would leave uncommitted
            elif k == "code" and pending: n += 1
            elif k == "merge": pending = False  # git refuses a merge over pending changes, so batch commits first
    return n
main_push = collections.defaultdict(list)
for h in B.LINES["main"]: main_push[B.PUSHIDX[h]].append(h)
print(f"  branches: {exposure([v for k, v in B.LINES.items() if k != 'main'])}; main, per push: {exposure(main_push.values())}")

print("\n== 6. Operations recorded twice: verb commits with the same patch-id")
kern = [h for h in B.ORDER if B.klass(h) == "kernel"]
p = subprocess.run(["git", "-C", B.REPO, "patch-id", "--stable"], input=subprocess.run(
    ["git", "-C", B.REPO, "show", "--format=commit %H", *kern], capture_output=True, text=True).stdout,
    capture_output=True, text=True).stdout.split("\n")
pid = collections.defaultdict(list)
for line in p:
    if line.strip(): a, h = line.split(); pid[(a, B.C[h]["subject"])].append(h)
dups = [v for v in pid.values() if len(v) > 1]
print(f"  pairs of verb commits with the same patch-id and subject: {len(dups)}")
for v in dups: print("   ", " = ".join(x[:9] for x in v), "|", B.C[v[0]]["subject"][:60])
