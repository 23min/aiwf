import subprocess, sys
from collections import defaultdict
S=sys.argv[1]; repo=sys.argv[2]
def git(*a): return subprocess.run(["git","-C",repo,*a],capture_output=True,text=True,check=True).stdout
meta={}
for line in open(f"{S}/meta.tsv"):
    h,p,ct,at,verb,actor=line.rstrip("\n").split("\t"); meta[h]=(p.split(),verb)
children=defaultdict(int)
for h,(ps,v) in meta.items():
    for p in ps: children[p]+=1
def segs(rng, break_on_fork):
    rows=git("log","--first-parent","--reverse","--format=%H",rng).split()
    n=0; open_=False
    for h in rows:
        ps,v=meta.get(h,([],""))
        if len(ps)>1: open_=False; continue        # merge on the line: must flush before
        if v and not open_: n+=1; open_=True
        if break_on_fork and children[h]>1: open_=False   # a branch is cut here: flush before
    return n
pt=[l.split() for l in open(f"{S}/pushtimes.txt")]
fp=set(open(f"{S}/fpall.txt").read().split())
for bf in (False,True):
    m=sum(segs(f"{pt[i-1][0]}..{pt[i][0]}",bf) for i in range(1,len(pt)))
    b=0
    for l in open(f"{S}/merges.txt"):
        mm,p1,p2=l.split()
        if p2 in fp: continue
        b+=segs(f"{p1}..{p2}",bf)
    print(f"flush at push+merge{'+fork' if bf else ''}: main={m} branches={b} total={m+b}")
def segs2(rng):
    rows=git("log","--first-parent","--reverse","--format=%H",rng).split()
    n=0; open_=False
    for h in rows:
        ps,v=meta.get(h,([],""))
        if len(ps)>1:
            # a merge whose 2nd parent descends from an earlier commit of this segment pins that commit
            continue
        if v and not open_: n+=1; open_=True
        if children[h]>1: open_=False
    return n
m=sum(segs2(f"{pt[i-1][0]}..{pt[i][0]}") for i in range(1,len(pt)))
b=0
for l in open(f"{S}/merges.txt"):
    mm,p1,p2=l.split()
    if p2 in fp: continue
    b+=segs2(f"{p1}..{p2}")
print(f"fold bounded by push+fork (merges re-made): main={m} branches={b} total={m+b}")
ritual={"plan-epic","plan-milestones","wrap-epic","wrap-milestone"}
def isrit(v): return bool(set(v.split(","))&ritual) if v else False
def segs3(rng, break_merge):
    rows=git("log","--first-parent","--reverse","--format=%H",rng).split()
    n=0; r=0; open_=False
    for h in rows:
        ps,v=meta.get(h,([],""))
        if len(ps)>1:
            if break_merge: open_=False
            continue
        if isrit(v): r+=1; continue
        if v and not open_: n+=1; open_=True
        if children[h]>1: open_=False
    return n,r
for bm,label in ((False,"c1 fold (push+fork bounded)"),(True,"c2 batch (push+merge+fork bounded)")):
    tot=[0,0]
    for i in range(1,len(pt)):
        a,b=segs3(f"{pt[i-1][0]}..{pt[i][0]}",bm); tot[0]+=a; tot[1]+=b
    for l in open(f"{S}/merges.txt"):
        mm,p1,p2=l.split()
        if p2 in fp: continue
        a,b=segs3(f"{p1}..{p2}",bm); tot[0]+=a; tot[1]+=b
    print(f"{label}: kernel-verb planning commits={tot[0]} + ritual commits kept={tot[1]} => {sum(tot)}")
