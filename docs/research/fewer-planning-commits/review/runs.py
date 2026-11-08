import subprocess, sys
S = sys.argv[1]; repo = sys.argv[2]
def git(*a): return subprocess.run(["git","-C",repo,*a],capture_output=True,text=True,check=True).stdout
meta = {}
for line in open(f"{S}/meta.tsv"):
    h,p,ct,at,verb,actor = line.rstrip("\n").split("\t")
    meta[h] = dict(parents=p.split(), ct=int(ct), verb=verb, actor=actor)
pt = [l.split() for l in open(f"{S}/pushtimes.txt")]
pushtime = {}; pushidx = {}
for i in range(1,len(pt)):
    prev, cur, t = pt[i-1][0], pt[i][0], int(pt[i][1])
    for h in git("rev-list", f"{prev}..{cur}").split():
        pushtime[h] = t; pushidx[h] = i
fp = set(git("rev-list","--first-parent","HEAD").split())
def planning(h): m = meta.get(h); return m is not None and len(m["parents"])==1 and m["verb"]!=""
starts = {"main":0,"branch":0}; joins={"main":0,"branch":0}
cross_ref_amends = 0
for h,m in meta.items():
    if not planning(h): continue
    where = "main" if h in fp else "branch"
    p = m["parents"][0]
    cont = planning(p) and p in pushtime and pushtime[p] > m["ct"] and meta[p]["actor"]==m["actor"]
    # variant used by brief: same push window on main
    if cont:
        joins[where]+=1
        if (p in fp) != (h in fp): cross_ref_amends += 1
    else: starts[where]+=1
print("runs(starts):",starts,"total",sum(starts.values()))
print("amends:",joins)
print("amend whose parent is on a different line (main<->branch):",cross_ref_amends)
# variant: join iff parent planning and same push window (no time test)
st={"main":0,"branch":0}; odd=[]
for h,m in meta.items():
    if not planning(h): continue
    where = "main" if h in fp else "branch"
    p = m["parents"][0]
    cont = planning(p) and pushidx.get(p)==pushidx.get(h)
    if not cont: st[where]+=1
    tcont = planning(p) and p in pushtime and pushtime[p] > m["ct"]
    if cont != tcont: odd.append((h[:9],where,p[:9],pushidx.get(p),pushidx.get(h),m["ct"],pushtime.get(p)))
print("push-window variant starts:",st,sum(st.values()))
print("disagreements:",len(odd)); [print(o) for o in odd[:12]]
st={"main":0,"branch":0}
for h,m in meta.items():
    if not planning(h): continue
    where = "main" if h in fp else "branch"
    p = m["parents"][0]
    while p in meta and len(meta[p]["parents"])>1: p = meta[p]["parents"][0]
    cont = planning(p) and pushidx.get(p)==pushidx.get(h)
    if not cont: st[where]+=1
print("merges-transparent, push-window:",st,sum(st.values()))
from collections import defaultdict
children=defaultdict(list)
for h,m in meta.items():
    for p in m["parents"]: children[p].append(h)
fork_amends=0; fork_parents=set()
for h,m in meta.items():
    if not planning(h): continue
    p=m["parents"][0]
    if planning(p) and p in pushtime and pushtime[p]>m["ct"] and len(children[p])>1:
        fork_amends+=1; fork_parents.add(p)
print("time-based amends whose HEAD^ is a fork point (another child exists):",fork_amends,"distinct parents:",len(fork_parents))
simple={"edit-body","add","retitle","set-priority"}
st=0; st_any=0
for h,m in meta.items():
    if not planning(h): continue
    p=m["parents"][0]
    eligible = planning(p) and p in pushtime and pushtime[p]>m["ct"] and meta[p]["actor"]==m["actor"]
    same = eligible and m["verb"]==meta[p]["verb"] and m["verb"] in simple
    if not same: st+=1
print("same-verb coalescing (edit-body/add/retitle/set-priority only), time-based:",st)
ritual={"plan-epic","plan-milestones","wrap-epic","wrap-milestone"}
def kverb(h): return planning(h) and not (set(meta[h]["verb"].split(","))&ritual)
res={"main":0,"branch":0}; fork_extra=0
for h,m in meta.items():
    if not planning(h): continue
    where="main" if h in fp else "branch"
    p=m["parents"][0]
    elig = kverb(h) and kverb(p) and p in pushtime and pushtime[p]>m["ct"] and meta[p]["actor"]==m["actor"]
    if not elig: res[where]+=1
    elif len(children[p])>1: fork_extra+=1
print("c3 excluding ritual commits:",res,sum(res.values()),"+ fork-point amends (upper bound):",fork_extra)
import re, collections
subj=dict(l.rstrip("\n").split("\t",1) for l in open(f"{S}/subj.tsv"))
# group c3 runs (ritual-excluded, time-based); classify each run by whether it has a phase promote / code neighbour
runstart={}
order=[h for h in meta if planning(h)]
def elig(h):
    m=meta[h]; p=m["parents"][0]
    return kverb(h) and kverb(p) and p in pushtime and pushtime[p]>m["ct"] and meta[p]["actor"]==m["actor"]
starts=[h for h in order if not elig(h)]
kinds=collections.Counter()
for h in starts:
    s=subj[h]
    if "--phase" in s: k="AC phase promote"
    elif re.search(r"AC-\d+ open -> met",s): k="AC met"
    elif set(meta[h]["verb"].split(","))&ritual: k="ritual commit"
    else: k=meta[h]["verb"].split(",")[0]
    kinds[k]+=1
print("c3 residue by first commit of each run:",dict(kinds.most_common()))
