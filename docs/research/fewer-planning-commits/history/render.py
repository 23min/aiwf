import sys, io, contextlib, json
sys.path.insert(0, ".")
sys.argv = ["foldbatch.py", "--drop-roadmap"]
with contextlib.redirect_stdout(io.StringIO()):
    import base as B
    import foldbatch as FB
RUN = json.load(open("amend_b.json"))["run"]
def s7(h): return h[:9]
def subj(h, w=74):
    t = B.C[h]["subject"]; return t if len(t) <= w else t[:w-1] + "…"
def op(h):
    v = B.verbs(h); e = B.trailers(h, "aiwf-entity")
    return f"{v[0] if v else '?'} {e[0] if e else ''}".strip()
def ops_line(g):
    ops = [op(h) for h in g]
    return f"aiwf: {len(g)} operations ({', '.join(ops[:4])}{', …' if len(ops) > 4 else ''})"
def tag(h):
    k = B.klass(h)
    return {"kernel": "plan ", "ritual": "rit  ", "code": "code ", "merge": "merge"}[k]

def actual(shas):
    for h in shas:
        r = " [ROADMAP-only, dropped in the other views]" if FB.dropped(h) else ""
        print(f"  {s7(h)} {tag(h)} {subj(h)}{r}")
def amend(shas):
    shas = [h for h in shas if not FB.dropped(h)]
    i = 0
    while i < len(shas):
        h = shas[i]
        if B.klass(h) == "kernel":
            r = RUN.get(h, h); g = [h]
            while i + 1 < len(shas) and B.klass(shas[i+1]) == "kernel" and RUN.get(shas[i+1]) == r:
                i += 1; g.append(shas[i])
            print(f"  {s7(g[0])} plan  {ops_line(g) if len(g) > 1 else subj(h)}")
        else:
            print(f"  {s7(h)} {tag(h)} {subj(h)}")
        i += 1
def fold(shas, label):
    shas = [h for h in shas if not FB.dropped(h)]
    FORK = {h for h in shas if h in FB.ALL_FORKS}
    for g in FB.groups_f1(shas, FORK):
        rw = set(FB.rewritten(g)); ks = [h for h in g if B.klass(h) == "kernel"]
        for h in g:
            if B.klass(h) == "kernel" and len(ks) > 1: continue
            mark = "*" if h in rw and B.klass(h) != "kernel" else " "
            print(f" {mark}{s7(h)} {tag(h)} {subj(h)}")
        if len(ks) > 1:
            print(f"  {'(new)':9} plan  {ops_line(ks)}")
        if g[-1] in FORK: print("   ---- (fold point: a branch is cut here) ----")


if __name__ == "__main__":
    # The last whole pushes on main covering at least 27 commits, and milestone
    # M-0349's branch, as today, under amend and under the fold.
    segs = sorted((s for s in FB.segs if s[0].startswith("main")), key=lambda s: B.POS[s[1][0]])
    pick, n = [], 0
    for s in reversed(segs):
        pick.insert(0, s); n += len(s[1])
        if n >= 27:
            break
    lid = "1c0f85113^2"
    for title in ("ACTUAL", "AMEND", "FOLD"):
        print(f"\n===== {title}: main (newest pushes, oldest first) =====")
        for label, shas, t in pick:
            print("  -- push --")
            {"ACTUAL": actual, "AMEND": amend}.get(title, lambda x: fold(x, label))(shas)
        print(f"\n===== {title}: milestone branch M-0349 (merged by 1c0f85113) =====")
        shas = B.LINES[lid]
        {"ACTUAL": actual, "AMEND": amend}.get(title, lambda x: fold(x, "branch"))(shas)
