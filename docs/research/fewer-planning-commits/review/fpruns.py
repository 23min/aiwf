import subprocess, sys
S=sys.argv[1]; repo=sys.argv[2]
def git(*a): return subprocess.run(["git","-C",repo,*a],capture_output=True,text=True,check=True).stdout
pt=[l.split() for l in open(f"{S}/pushtimes.txt")]
res={"merge_breaks":0,"merge_verb_is_planning":0,"merges_ignored":0}
for i in range(1,len(pt)):
    rows=[l.split("\t") for l in git("log","--first-parent","--reverse","--format=%H%x09%P%x09%(trailers:key=aiwf-verb,valueonly,separator=%x2C)",f"{pt[i-1][0]}..{pt[i][0]}").splitlines()]
    seq=[(len(p.split())>1, v!="") for h,p,v in rows]
    for mode in res:
        prev=False
        for ismerge,isplan in seq:
            if mode=="merge_breaks": plan = isplan and not ismerge
            elif mode=="merge_verb_is_planning": plan = isplan
            else:
                if ismerge: continue
                plan=isplan
            if plan and not prev: res[mode]+=1
            prev=plan
print(res)
