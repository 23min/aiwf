import subprocess, sys
S=sys.argv[1]; repo=sys.argv[2]
def git(*a): return subprocess.run(["git","-C",repo,*a],capture_output=True,text=True,check=True).stdout
def exposed(rng):
    rows=[l.split("\t") for l in git("log","--first-parent","--reverse","--format=%H%x09%P%x09%(trailers:key=aiwf-verb,valueonly,separator=%x2C)",rng).splitlines()]
    seen=False; n=0; nm=0
    for h,p,v in rows:
        ism=len(p.split())>1
        if v and not ism: seen=True
        elif seen and not ism and not v: n+=1
        elif seen and ism: nm+=1
    return n,nm
pt=[l.split() for l in open(f"{S}/pushtimes.txt")]
main=[exposed(f"{pt[i-1][0]}..{pt[i][0]}") for i in range(1,len(pt))]
print("main: code commits after a planning verb in same push:",sum(a for a,b in main),"merges after:",sum(b for a,b in main))
fp=set(open(f"{S}/fpall.txt").read().split())
br=[];
for l in open(f"{S}/merges.txt"):
    m,p1,p2=l.split()
    if p2 in fp: continue
    br.append(exposed(f"{p1}..{p2}"))
print("branches: code commits after a planning verb:",sum(a for a,b in br),"merges after:",sum(b for a,b in br), "branches with any:", sum(1 for a,b in br if a))
