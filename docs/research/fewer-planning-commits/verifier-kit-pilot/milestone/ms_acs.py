import re, sys, glob, collections
rows=[]
for p in sorted(glob.glob('work/epics/**/M-*.md', recursive=True)):
    t=open(p).read()
    if not t.startswith('---'): continue
    fm=t.split('---\n')[1]
    mid=re.search(r'^id: (\S+)', fm, re.M).group(1)
    st=re.search(r'^status: (\S+)', fm, re.M).group(1)
    tdd=(re.search(r'^tdd: (\S+)', fm, re.M) or [None,'(absent)'])[1]
    title=re.search(r'^title: (.+)$', fm, re.M).group(1).strip("'\"")
    acs=[]
    if '\nacs:' in fm:
        blk=fm.split('\nacs:',1)[1]
        acs=re.findall(r'^\s+status: (\S+)', blk, re.M)
    rows.append((mid,st,tdd,acs,title,p))
print('milestones:',len(rows))
c=collections.Counter(st for _,st,_,_,_,_ in rows); print('by status:',dict(c))
zero=[r for r in rows if not r[3]]
print('\nzero-AC milestones:',len(zero))
for r in zero: print(' ',r[0],r[1],'tdd='+r[2],'|',r[4][:70])
dead=[r for r in rows if r[3] and all(s in('deferred','cancelled') for s in r[3])]
print('\nmilestones whose every AC is deferred/cancelled:',len(dead))
for r in dead: print(' ',r[0],r[1],r[3],'|',r[4][:70])
some=[r for r in rows if r[3] and any(s in('deferred','cancelled') for s in r[3])]
print('\nmilestones with at least one deferred/cancelled AC:',len(some))
open_done=[r for r in rows if r[1]=='done' and 'open' in r[3]]
print('done milestones with an open AC:',len(open_done), [r[0] for r in open_done])
canc=[r for r in rows if r[1]=='cancelled']
print('\ncancelled milestones:',len(canc))
for r in canc: print(' ',r[0],r[3],'|',r[4][:60])
