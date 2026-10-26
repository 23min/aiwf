#!/usr/bin/env python3
"""Run the design x condition matrix and the mutation probes.

  python3 -I run.py [--jobs N] [--max-states N] [--time-limit SECS] [--only SUBSTR]

Writes results.json and traces/*.txt next to this file.
"""
import argparse
import json
import multiprocessing as mp
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
from model import Cfg, DESIGNS, applicable, explore, render_trace  # noqa: E402

MODES = ('direct', 'mr', 'mrutd')
CONDS = ('base', 'offline', 'lost', 'xcite', 'abandon')

# bound sets: (label, ents per clone, cites per clone)
B2 = ('2c', (2, 1), (1, 1))          # base, offline, lost
B2X = ('2c', (1, 1), (1, 1))         # xcite, abandon: two clones
B3X = ('3c', (1, 0, 1), (0, 0, 0))   # xcite, abandon: B only cites; A and C allocate


def cond_kwargs(cond):
    return {'base': {}, 'offline': {'offline': True}, 'lost': {'lost': True},
            'xcite': {'xcite': True}, 'abandon': {'abandon': True, 'xcite': True}}[cond]


def matrix_jobs():
    jobs = []
    for d in DESIGNS:
        for mode in MODES:
            if applicable(d, mode):
                continue
            for cond in CONDS:
                if mode == 'direct' and cond in ('xcite', 'abandon'):
                    continue          # no unmerged branch is visible to anyone in direct mode
                bounds = [B2X, B3X] if cond in ('xcite', 'abandon') else [B2]
                for (bl, ents, cites) in bounds:
                    kw = dict(cond_kwargs(cond), ents=ents, cites=cites)
                    jobs.append(dict(kind='matrix', design=d, mode=mode, cond=cond, bound=bl, kw=kw))
    return jobs


PROBES = [
    # (id, what is broken, design, mode, kwargs, invariant expected to fail)
    ('P1', 'ledger claim pushed with force (no compare-and-swap)', 'Blazy', 'mr',
     dict(probes=frozenset({'nocas'})), 'I1'),
    ('P2', 'lost claim reply: clone assumes the claim landed', 'Blazy', 'mr',
     dict(lost=True, probes=frozenset({'assume_ok'})), 'I1'),
    ('P3', 'lost claim reply: reconcile checks the label, not the claimant', 'Blazy', 'mr',
     dict(lost=True, probes=frozenset({'bylabel'})), 'I1'),
    ('P4', 'lost claim reply: the clone never reconciles', 'Beager', 'mr',
     dict(lost=True, probes=frozenset({'noreconcile'})), 'I5'),
    ('P5', 'A with the unscoped rewrite (today\'s reallocate)', 'A', 'mrutd',
     dict(probes=frozenset({'unscoped'})), 'I2'),
    ('P6', 'A without the merge pipeline (host merges unchecked)', 'A', 'mrutd',
     dict(pipeline=False), 'I1'),
    ('P6b', 'A without the merge pipeline AND without the pre-push hook', 'A', 'mrutd',
     dict(pipeline=False, hook=False), 'I1'),
    ('P7', 'A in direct mode with the pre-push hook bypassed', 'A', 'direct',
     dict(hook=False), 'I1'),
    ('P8', 'E, two mint triggers each minting all pending entities, no compare-and-swap', 'Euniq', 'mrutd',
     dict(minters=2, ents=(1, 1), probes=frozenset({'nocas_mint'})), 'I1'),
    ('P8b', 'E, two mint triggers each minting only its own merge, no compare-and-swap', 'Euniq', 'mrutd',
     dict(minters=2, ents=(1, 1), probes=frozenset({'nocas_mint', 'mint_own'})), 'I1'),
    ('P8c', 'control for P8b: per-merge mint triggers WITH compare-and-swap', 'Euniq', 'mrutd',
     dict(minters=2, ents=(1, 1), probes=frozenset({'mint_own'})), None),
    ('P9', 'E with no mint trigger at all', 'Euniq', 'mrutd', dict(minters=0), 'I5'),
    ('P10', 'C per-clone namespace assigned twice (same namespace on two clones)', 'Cns', 'mr',
     dict(probes=frozenset({'samens'})), 'I1'),
    ('P11', 'C random ids from a 2-token space', 'Crand', 'mr',
     dict(probes=frozenset({'smalltok'})), 'I1'),
    ('P12', 'E without following a renamed handle (no late-reference fixup)', 'Euniq', 'mrutd',
     dict(xcite=True, ents=(1, 1), cites=(1, 1), probes=frozenset({'nofixref'})), 'I2'),
    ('P13', 'D without following a renamed handle', 'D', 'mrutd',
     dict(xcite=True, ents=(1, 1), cites=(1, 1), probes=frozenset({'nofixref'})), 'I2'),
]


def probe_jobs():
    jobs = []
    for pid, what, d, mode, kw, inv in PROBES:
        kw = dict(kw)
        kw.setdefault('ents', B2[1])
        kw.setdefault('cites', B2[2])
        jobs.append(dict(kind='probe', pid=pid, what=what, design=d, mode=mode, cond='probe',
                         bound='2c' if len(kw['ents']) == 2 else '3c', kw=kw, expect=inv))
    return jobs


def job_id(j):
    if j['kind'] == 'probe':
        return f"{j['pid']}_{j['design']}@{j['mode']}"
    return f"{j['design']}@{j['mode']}+{j['cond']}[{j['bound']}]"


def safe(s):
    return ''.join(ch if ch.isalnum() or ch in '-_.@+[]' else '_' for ch in s)


def run_job(args):
    j, max_states, time_limit, graph = args
    kw = dict(j['kw'])
    cfg = Cfg(j['design'], j['mode'], max_states=max_states, **kw)
    r = explore(cfg, want_graph=graph, time_limit=time_limit)
    jid = job_id(j)
    tdir = os.path.join(HERE, 'traces')
    os.makedirs(tdir, exist_ok=True)
    res = dict(j, kw={k: (sorted(v) if isinstance(v, frozenset) else v) for k, v in kw.items()},
               id=jid, states=r.nstates, edges=r.nedges, terminals=r.nterm, complete=not r.truncated,
               depth_complete=r.depth_complete, max_depth=r.max_depth,
               secs=round(r.secs, 1), acyclic=r.acyclic, done=r.term_done, viol={}, eventual={},
               refused={}, stuck=None)
    for inv, (i, act, detail) in sorted(r.viol.items()):
        fn = safe(f"{jid}__{inv}.txt")
        steps = len(path_len(r, i)) + (1 if act else 0)
        with open(os.path.join(tdir, fn), 'w') as f:
            f.write(render_trace(r, i, act, detail, f"{jid}: {inv}") + '\n')
        res['viol'][inv] = dict(trace=fn, detail=detail, steps=steps)
    for key, (i, pb) in sorted(r.term_eventual.items()):
        fn = safe(f"{jid}__{key}-terminal.txt")
        with open(os.path.join(tdir, fn), 'w') as f:
            f.write(render_trace(r, i, None, f"terminal state: {pb}", f"{jid}: {key}") + '\n')
        res['eventual'][key] = dict(trace=fn, detail=pb, steps=len(path_len(r, i)))
    for key, (i, human) in sorted(r.term_refused.items()):
        fn = safe(f"{jid}__refused-{key}.txt")
        with open(os.path.join(tdir, fn), 'w') as f:
            f.write(render_trace(r, i, None, f"terminal, refused with a human remedy: {human}",
                                 f"{jid}: refused ({key})") + '\n')
        res['refused'][key] = dict(trace=fn, detail=str(human))
    if r.term_stuck:
        i, stuck = r.term_stuck
        fn = safe(f"{jid}__I5-stuck.txt")
        with open(os.path.join(tdir, fn), 'w') as f:
            f.write(render_trace(r, i, None, f"terminal with work that can never land: {stuck}",
                                 f"{jid}: I5 stuck") + '\n')
        res['stuck'] = dict(trace=fn, detail=str(stuck), steps=len(path_len(r, i)))
    rn = r.renums or [0]
    res['renumbers'] = dict(max=max(rn), min=min(rn),
                            share_with_any=round(sum(1 for x in rn if x) / len(rn), 3))
    return res


def path_len(r, i):
    from model import path
    return path(r, i)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--jobs', type=int, default=max(1, (os.cpu_count() or 2) - 2))
    ap.add_argument('--max-states', type=int, default=2_500_000)
    ap.add_argument('--time-limit', type=int, default=900)
    ap.add_argument('--only', default='', help='comma-separated substrings of run ids')
    ap.add_argument('--no-graph', action='store_true', help='skip the cycle check (saves memory)')
    ap.add_argument('--merge', action='store_true', help='replace these runs inside an existing results file')
    ap.add_argument('--out', default=os.path.join(HERE, 'results.json'))
    a = ap.parse_args()
    pats = a.only.split(',')
    jobs = [j for j in matrix_jobs() + probe_jobs() if any(p in job_id(j) for p in pats)]
    print(f"{len(jobs)} runs, {a.jobs} workers", flush=True)
    results = []
    with mp.Pool(a.jobs, maxtasksperchild=1) as pool:
        for res in pool.imap_unordered(run_job, [(j, a.max_states, a.time_limit, not a.no_graph) for j in jobs]):
            results.append(res)
            flag = '' if res['complete'] else ' TRUNCATED'
            print(f"{res['id']:40s} {res['states']:>9} states {res['secs']:>6}s{flag} "
                  f"viol={sorted(res['viol'])} ev={sorted(res['eventual'])} "
                  f"stuck={'yes' if res['stuck'] else 'no'} refused={sorted(res['refused'])}", flush=True)
    if a.merge and os.path.exists(a.out):
        new = {r['id'] for r in results}
        results += [r for r in json.load(open(a.out)) if r['id'] not in new]
    results.sort(key=lambda r: r['id'])
    with open(a.out, 'w') as f:
        json.dump(results, f, indent=1)
    print('wrote', a.out)


if __name__ == '__main__':
    main()
