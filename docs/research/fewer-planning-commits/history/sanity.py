import sys, os, collections
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from base import *
print("window commits", len(C), "fp", len(FP), "lines", len(LINES), "unassigned", len(UNASSIGNED))
print("FF events on main:", [(m[:70], len(r)) for o,n,t,m,r in FF_EVENTS])
fp_not_made = [h for h in FP if h not in MADE_ON_MAIN and h not in FF_ON_MAIN]
print("fp commits neither made on main nor FF'd:", len(fp_not_made), [ (h[:9], C[h]['subject'][:50]) for h in fp_not_made[:10]])
made_not_fp = [h for h in MADE_ON_MAIN if h not in FPSET]
print("made on main but not on fp:", len(made_not_fp), [(h[:9], MADE_ON_MAIN[h][:30], C[h]['subject'][:40]) for h in made_not_fp[:10]])
# line bases
kinds = collections.Counter()
for lid, shas in LINES.items():
    if lid == "main": continue
    b = line_base(lid)
    bl = line_of(b)
    # continuation: base is the second parent of an earlier merge
    cont = any(len(C[m]["parents"])>1 and b in C[m]["parents"][1:] for m in C if is_merge(m) and m != LINEMERGE[lid])
    kinds[("main" if bl=="main" else ("pre-window" if bl=="pre-window" else "branch"), cont)] += 1
print("line bases (base line kind, continuation of an earlier merged tip):", kinds)
print("klass counts", collections.Counter(klass(h) for h in C))
print("non-merge planning", sum(1 for h in C if klass(h) in ("kernel","ritual")))
print("by line kind:", collections.Counter((("main" if LINE[h]=="main" else "branch"), klass(h)) for h in C))
print("pushes", len(PUSHES), "first", PUSHES[0][2], PUSHES[1][2])
