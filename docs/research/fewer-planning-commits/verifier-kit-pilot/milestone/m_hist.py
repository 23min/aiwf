import re, collections
ev=collections.defaultdict(list)
def norm(x):
    m=re.match(r'M-(\d+)(/AC-(\d+))?$', x.strip())
    if not m: return None
    return ('M-%04d'%int(m.group(1)), m.group(3))
for line in open('/tmp/claude-1000/-workspaces-aiwf/c25231db-35d6-4116-a924-225df6fdc15d/scratchpad/m_events.txt'):
    if not line.startswith('@@'): continue
    sha,date,verb,ents,to,force,reason=(line[2:].rstrip('\n').split('|')+['']*7)[:7]
    for e in ents.split(','):
        n=norm(e)
        if n: ev[n[0]].append((date,sha[:9],verb,n[1],to,force,reason))
zero_start=[];forced_start=[];dead_start=[];after_term=[]
for m,es in ev.items():
    acs={}  # ac -> status
    mstat='draft'
    for date,sha,verb,ac,to,force,reason in es:
        if ac:
            if verb=='add': acs[ac]='open'
            elif verb=='promote' and to in('open','met','deferred','cancelled'):
                if mstat in('done','cancelled'): after_term.append((m,ac,mstat,to,date,sha))
                acs[ac]=to
            elif verb=='cancel':
                if mstat in('done','cancelled'): after_term.append((m,ac,mstat,'cancelled',date,sha))
                acs[ac]='cancelled'
        else:
            if verb=='promote' and to=='in_progress':
                if force: forced_start.append((m,date,sha,force))
                if not acs: zero_start.append((m,date,sha,bool(force)))
                elif all(s in('deferred','cancelled') for s in acs.values()): dead_start.append((m,date,sha,dict(acs)))
            if verb=='promote' and to: mstat=to
            if verb=='cancel': mstat='cancelled'
print('milestones with events:',len(ev))
print('\nstarted (promote in_progress) with no AC added beforehand:',len(zero_start))
for z in zero_start: print(' ',z)
print('\nforced starts:',len(forced_start))
for z in forced_start: print(' ',z)
print('\nstarted while every AC was deferred/cancelled:',len(dead_start))
for z in dead_start: print(' ',z)
print('\nAC status changes after the milestone was done/cancelled:',len(after_term))
for z in after_term: print(' ',z)
