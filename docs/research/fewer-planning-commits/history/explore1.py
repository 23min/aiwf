import json, sys, collections
d = json.load(open(sys.argv[1])); vs = json.load(open(sys.argv[2]))
C = d["commits"]; REG=set(vs["registered"]); RIT=set(vs["ritual"])
def tv(c,k): return [v for kk,v in c["trailers"] if kk.lower()==k]
cnt = collections.Counter(); unk = collections.Counter(); multi=0; actors=collections.Counter(); princ=collections.Counter()
nm_plan=0; m_plan=0; rit_nm=0; road=0; road_rit=0; road_unst=[]; subj_aiwf_nonverb=[]; verb_nonaiwf_subj=collections.Counter()
for h,c in C.items():
    verbs = tv(c,"aiwf-verb"); merge = len(c["parents"])>1
    if len(verbs)>1: multi+=1
    for v in verbs: cnt[v]+=1
    for v in verbs:
        if v not in REG and v not in RIT: unk[v]+=1
    for a in tv(c,"aiwf-actor"): actors[a]+=1
    for a in tv(c,"aiwf-principal"): princ[a]+=1
    if verbs and not merge: nm_plan+=1
    if verbs and merge: m_plan+=1
    if verbs and not merge and set(verbs)&RIT: rit_nm+=1
    paths = [p for s,p in c["files"]]
    if not merge and paths and all(p=="ROADMAP.md" for p in paths):
        road+=1
        if verbs and set(verbs)&RIT: road_rit+=1
        elif not verbs: road_unst.append((h[:9], c["subject"]))
    if not merge and c["subject"].startswith("aiwf ") and not verbs: subj_aiwf_nonverb.append((h[:9],c["subject"]))
    if verbs and not merge and not (set(verbs)&RIT) and not c["subject"].startswith("aiwf "): verb_nonaiwf_subj[tuple(verbs)]+=1
print("verbs", cnt.most_common()); print("unknown", unk); print("multi-verb commits", multi)
print("actors", actors); print("principals", princ)
print("non-merge planning", nm_plan, "merge with verb", m_plan, "ritual non-merge", rit_nm)
print("ROADMAP-only non-merge", road, "of which ritual-stamped", road_rit, "unstamped", len(road_unst))
for r in road_unst: print("  unstamped", r)
print("subject 'aiwf ' w/o verb", subj_aiwf_nonverb[:10], len(subj_aiwf_nonverb))
print("kernel-verb trailer but non-aiwf subject", verb_nonaiwf_subj)
ae = collections.Counter(c["ae"] for c in C.values()); print("authors", ae)
reb = [(h[:9],c["subject"][:60],c["ct"]-c["at"]) for h,c in C.items() if c["ct"]!=c["at"]]
print("author!=committer time", len(reb)); 
for r in reb[:40]: print("  ",r)
