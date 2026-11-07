#!/usr/bin/env python3
"""Run the design x host x condition matrix, the mutation probes and the
fidelity checks.

  python3 -I run.py [--jobs N] [--max-states N] [--time-limit SECS] [--only SUBSTR]

Writes results.json, fidelity.json and traces/ next to this file.

  python3 -I run.py --amend [...]

also runs the amend designs (model.AMEND_DIMS) in every group, the amend
conditions (model.AMEND_CONDS) for the per-verb, amend and two fold designs
(AMEND_COMPARE), and the amend probes; make_amend.py renders the result.

Within one (mode, host, condition) group, a design whose values agree with an
already explored design on every dimension that run *read* is not explored
again: the model is deterministic and reads its design only through
Model.d(), so the two runs would execute identically. Each result row names
the run it was taken from.
"""
import argparse
import itertools
import json
import multiprocessing as mp
import os
import resource
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import model as M  # noqa: E402

HOSTS_MR = ([(m, '-', u) for m in ('merge', 'semi', 'ff') for u in (False, True)]
            + [('squash', q, u) for q in ('title', 'all', 'git') for u in (False, True)])
CONDS_DIRECT = ('base', 'after', 'mixed', 'crash', 'lost', 'revert', 'resolve', 'nocheck', 'auth', 'ack', 'abc',
                'adr')
DIM_ORDER = ('submit', 'fold', 'resub', 'txid', 'ptr', 'audit')


def host_id(mode, h):
    if mode == 'direct':
        return 'direct'
    m, q, u = h
    return m + (f"/{q}" if m == 'squash' else '') + ('+utd' if u else '')


def designs(amend=False):
    out = []
    for vals in itertools.product(*(M.DIMS[k] for k in DIM_ORDER)):
        d = dict(zip(DIM_ORDER, vals))
        # rewrite and refuse act only in a fold, so per-verb commits take leave or opid
        if d['submit'] == 'none' and d['ptr'] in ('rewrite', 'refuse'):
            continue
        out.append(d)
    if amend:
        # after every other design, so their runs and run ids are the ones the matrix without
        # amend has; an amend design carries only the dimensions an amend run reads
        for vals in itertools.product(*M.AMEND_DIMS.values()):
            out.append(dict(zip(M.AMEND_DIMS, vals)))
    return out


def design_id(d, used=None):
    if d['submit'] == 'none':
        return f"perverb.{'opid.' if d['ptr'] == 'opid' else ''}{d['audit']}"
    if d['submit'] == 'amend':
        return f"amend.{d['amend']}.{'opid.' if d['ptr'] == 'opid' else ''}{d['audit']}"
    return '.'.join(d[k] for k in DIM_ORDER[1:])


# the amend conditions run for these designs and the amend designs only
AMEND_COMPARE = ('perverb.parent', 'perverb.record', 'perverb.opid.parent', 'perverb.opid.record',
                 'replay.append.record.opid.record', 'replay.replace.record.opid.record')
CONDS_DIRECT_AMEND = tuple('sa-' + c for c in M.SA_BASE if c in CONDS_DIRECT) + (
    'fork', 'reuse4', 'untracked', 'scope', 'actor', 'byc', 'twostep', 'nr-fork', 'nr-scope', 'nr-byc')


def groups(amend=False):
    jobs = []
    for h in HOSTS_MR:
        for c in M.CONDS:
            jobs.append(dict(mode='mr', host=h, cond=c, amend=amend))
    for c in CONDS_DIRECT:
        jobs.append(dict(mode='direct', host=('merge', '-', False), cond=c, amend=amend))
    if amend:
        only = AMEND_COMPARE + tuple(design_id(d) for d in designs(True) if d['submit'] == 'amend')
        for h in HOSTS_MR:
            for c in M.AMEND_CONDS:
                jobs.append(dict(mode='mr', host=h, cond=c, amend=True, designs=only))
        for c in CONDS_DIRECT_AMEND:
            jobs.append(dict(mode='direct', host=('merge', '-', False), cond=c, amend=True, designs=only))
    return jobs


def safe(s):
    return ''.join(ch if ch.isalnum() or ch in '-_.@+[]=' else '_' for ch in s)


def write_trace(tdir, name, text):
    with open(os.path.join(tdir, name), 'w') as f:
        f.write(text + '\n')


def summarise_run(r, rid, tdir):
    """Result fields of one explored run, writing its traces."""
    res = dict(run=rid, states=r.nstates, edges=r.nedges, terminals=r.nterm, complete=not r.truncated,
               depth_complete=r.depth_complete, max_depth=r.max_depth, secs=round(r.secs, 2),
               acyclic=r.acyclic, done=r.done, nbound=r.nbound, used=sorted(r.used), viol={}, refused={}, stuck=None)
    for inv, (i, act, detail, det) in sorted(r.viol.items()):
        fn = safe(f"{rid}__{inv}.txt")
        write_trace(tdir, fn, M.render_trace(r, i, act, detail, f"{rid}: {inv}"))
        sil = r.silent.get(inv)
        entry = dict(trace=fn, detail=detail, detected=det, steps=len(M.path(r, i)) + (1 if act else 0),
                     silent=sil is not None)
        if sil is not None and sil[:2] != (i, act):
            fn2 = safe(f"{rid}__{inv}-silent.txt")
            write_trace(tdir, fn2, M.render_trace(r, sil[0], sil[1], sil[2], f"{rid}: {inv} (not detected)"))
            entry['silent_trace'] = fn2
        res['viol'][inv] = entry
    for key, (i, reasons) in sorted(r.refused.items()):
        fn = safe(f"{rid}__refused-{key}.txt")
        write_trace(tdir, fn, M.render_trace(r, i, None, f"terminal, refused: {reasons}", f"{rid}: refused ({key})"))
        res['refused'][key] = dict(trace=fn, detail=str(reasons))
    if r.stuck:
        i, why = r.stuck
        fn = safe(f"{rid}__L-stuck.txt")
        write_trace(tdir, fn, M.render_trace(r, i, None, f"terminal, stuck: {why}", f"{rid}: stuck"))
        res['stuck'] = dict(trace=fn, detail=str(why), steps=len(M.path(r, i)))
    return res


def run_group(args):
    g, max_states, time_limit, only_designs = args
    tdir = os.path.join(HERE, 'traces')
    os.makedirs(tdir, exist_ok=True)
    mode, h, cond = g['mode'], g['host'], g['cond']
    hid = host_id(mode, h)
    explored = []          # (used keys, design, result)
    rows = []
    t0 = time.time()
    for d in designs(g.get('amend', False)):
        if only_designs and design_id(d) not in only_designs:
            continue
        if g.get('designs') and design_id(d) not in g['designs']:
            continue
        hit = None
        for used, d0, res0 in explored:
            if all(d.get(k) == d0.get(k) for k in used):
                hit = res0
                break
        if hit is None:
            cfg = M.Cfg(mode=mode, method=h[0], sqmsg=h[1], utd=h[2], cond=cond,
                        design=tuple(sorted(d.items())), max_states=max_states)
            r = M.explore(cfg, time_limit=time_limit)
            rid = f"{hid}@{cond}@{design_id(d)}"
            hit = summarise_run(r, rid, tdir)
            explored.append((sorted(r.used), d, hit))
            del r
        rows.append(dict(kind='matrix', mode=mode, host=hid, cond=cond, design=d, design_id=design_id(d), **hit))
    peak = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss / 1024
    return rows, dict(group=f"{hid}@{cond}", runs=len(explored), secs=round(time.time() - t0, 1),
                      peak_mb=round(peak))


DEFAULT = dict(submit='fold', fold='replay', resub='replace', txid='record', ptr='opid', audit='record')

PROBES = [
    # (id, invariant it targets, what is broken, mode, host, cond, design overrides, probes)
    ('P1', 'T1', 'a crash during submit leaves the branch rewritten without its planning commit (ref updated in two steps)',
     'mr', ('merge', '-', False), 'crash', {}, {'nonatomic'}),
    ('P2', 'T1', 'the re-submit push is a plain --force (no lease, no check for others\' commits)',
     'mr', ('merge', '-', False), 'bob', {}, {'nolease'}),
    ('P3', 'T1', 'the fold carries only work/, not every path the verbs wrote (docs/adr/ is dropped)',
     'mr', ('merge', '-', False), 'adr', {}, {'workonly'}),
    ('P4', 'T2', 'replay overwrites a path with the planning value instead of merging three ways',
     'mr', ('merge', '-', False), 'resolve', {}, {'overwrite', 'resolve-both'}),
    ('P5', 'T3', 'the planning commit\'s record names only its last operation (E5\'s probe-drop)',
     'mr', ('merge', '-', False), 'adr', {}, {'droprec'}),
    ('P6', 'T4', 'the planning commit has a paragraph after its trailer block',
     'mr', ('merge', '-', False), 'nocheck', {}, {'footer'}),
    ('P7', 'T5', 'submit re-creates every commit since the fork point (as git rebase --force-rebase would)',
     'mr', ('merge', '-', False), 'base', {}, {'rebuildall'}),
    ('P8', 'T6', 'ptr=refuse checks trailer pointers only, missing addressed_by_commit in frontmatter '
                 '(both runs: operators merge main, never rebase)',
     'mr', ('merge', '-', False), 'abc', {'ptr': 'refuse'}, {'refuse-trailers-only', 'norebase'}),
    ('P9', 'T7', 'the fold writes its groups out of operation order; no hook runs (direct mode, hook absent)',
     'direct', ('merge', '-', False), 'nocheck', {}, {'unordered'}),
    ('P10', 'T4d', 'txid=record without the duplicate-record check in aiwf check',
     'mr', ('ff', '-', False), 'reuse', {}, {'nodupcheck'}),
    ('W1', 'T4d', 'what-if, not a mutation: submit refuses to fold a commit already pushed '
                  '(\u00a74.2 "runs before the first push, or refuses" read strictly)',
     'mr', ('merge', '-', False), 'bob', {}, {'strict-unpushed'}),
    ('W2', 'T4d', 'what-if, not a mutation: submit refuses to fold an operation whose record is already on '
                  'the remote branch (judged by operation identity, not SHA)',
     'mr', ('merge', '-', False), 'bob', {}, {'strict-unpushed-ops'}),
    ('W3', 'T4d', 'what-if, not a mutation: W2\'s rule (refuse to fold an operation already recorded on the '
                  'remote branch), under `after`',
     'mr', ('merge', '-', False), 'after', {}, {'strict-unpushed-ops'}),
    ('P11', 'L', 'the fold writes its groups out of operation order, so the record audit reads '
                 'done before in_progress',
     'mr', ('merge', '-', False), 'base', {}, {'unordered'}),
]


AMEND_DEFAULT = dict(submit='amend', amend='naive', ptr='leave', audit='record')

AMEND_PROBES = [
    ('AP1', 'L', 'amend.naive without the actor test: the agent\'s verb may amend the human\'s authorize commit, '
                 'naming it by the SHA the amend replaces (both runs: operators merge main, never rebase)',
     'mr', ('merge', '-', False), 'actor', {}, {'amend-any-actor', 'norebase'}),
    ('AW1', 'L', 'what-if, not a mutation: amend.safe without the actor test, so only its SHA clause stands '
                 'between the two (both runs: operators merge main, never rebase)',
     'mr', ('merge', '-', False), 'actor', {'amend': 'safe'}, {'amend-any-actor', 'norebase'}),
]


def run_probe(args):
    p, max_states, time_limit = args
    pid, inv, what, mode, h, cond, over, probes = p
    base = AMEND_DEFAULT if pid.startswith('A') else DEFAULT
    tdir = os.path.join(HERE, 'traces')
    os.makedirs(tdir, exist_ok=True)
    out = {}
    keep = {'resolve-both', 'norebase'}          # scenario restrictions, kept in the control run
    for tag, pr in (('probe', frozenset(probes)), ('control', frozenset(x for x in probes if x in keep))):
        d = dict(base, **over)
        cfg = M.Cfg(mode=mode, method=h[0], sqmsg=h[1], utd=h[2], cond=cond, design=tuple(sorted(d.items())),
                    probes=pr, max_states=max_states)
        r = M.explore(cfg, time_limit=time_limit)
        rid = f"{pid}-{tag}_{host_id(mode, h)}@{cond}@{design_id(d)}"
        out[tag] = summarise_run(r, rid, tdir)
    return dict(kind='probe', pid=pid, inv=inv, what=what, mode=mode, host=host_id(mode, h), cond=cond,
                design=dict(base, **over), probes=sorted(probes), **out)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--jobs', type=int, default=12)
    ap.add_argument('--max-states', type=int, default=3_000_000)
    ap.add_argument('--time-limit', type=int, default=1200)
    ap.add_argument('--only', default='', help='comma-separated substrings of group ids (host@cond)')
    ap.add_argument('--designs', default='', help='comma-separated design ids to restrict to')
    ap.add_argument('--no-probes', action='store_true')
    ap.add_argument('--probes-only', action='store_true', help='rerun the probes into an existing results file')
    ap.add_argument('--merge', action='store_true', help='replace the rerun groups (and probes) in an existing results file')
    ap.add_argument('--out', default=os.path.join(HERE, 'results.json'))
    ap.add_argument('--amend', action='store_true', help='also the amend designs, conditions and probes')
    a = ap.parse_args()
    pats = [x for x in a.only.split(',')]
    gs = [g for g in groups(a.amend) if any(p in f"{host_id(g['mode'], g['host'])}@{g['cond']}" for p in pats)]
    if a.probes_only:
        gs = []
    only_designs = set(x for x in a.designs.split(',') if x)
    # biggest groups first, so the long ones do not start last
    weight = {'bob': 0, 'after': 1, 'reuse': 2}
    gs.sort(key=lambda g: weight.get(g['cond'], 9))
    print(f"{len(gs)} groups, {a.jobs} workers", flush=True)
    rows, stats = [], []
    t0 = time.time()
    with mp.Pool(a.jobs, maxtasksperchild=1) as pool:
        work = [(g, a.max_states, a.time_limit, only_designs) for g in gs]
        for rr, st in pool.imap_unordered(run_group, work):
            rows += rr
            stats.append(st)
            print(f"{st['group']:28s} {st['runs']:3d} runs {st['secs']:7.1f}s peak {st['peak_mb']:6d} MB "
                  f"[{time.time() - t0:6.0f}s]", flush=True)
        if not a.no_probes:
            probes = PROBES + (AMEND_PROBES if a.amend else [])
            for res in pool.imap_unordered(run_probe, [(p, a.max_states, a.time_limit) for p in probes]):
                rows.append(res)
                inv = res['inv']
                failed = sorted(set(res['probe']['viol']) | ({'L'} if res['probe']['stuck'] else set()))
                ctl = sorted(set(res['control']['viol']) | ({'L'} if res['control']['stuck'] else set()))
                print(f"{res['pid']:4s} {inv:4s} probe fails {failed} control fails {ctl}", flush=True)
    rows.sort(key=lambda r: (r['kind'], r.get('host', ''), r.get('cond', ''), r.get('design_id', r.get('pid', ''))))
    meta = dict(stats=stats, wall=round(time.time() - t0, 1),
                when=time.strftime('%Y-%m-%d %H:%M UTC', time.gmtime()), python=sys.version.split()[0],
                cpus=os.cpu_count(), jobs=a.jobs)
    if a.probes_only or a.merge:
        old = json.load(open(a.out))
        redone = {(r['host'], r['cond']) for r in rows if r['kind'] == 'matrix'}
        keep = [r for r in old['rows'] if (r['kind'] == 'matrix' and (r['host'], r['cond']) not in redone)
                or (r['kind'] == 'probe' and a.no_probes)]
        rows = keep + rows
        rows.sort(key=lambda r: (r['kind'], r.get('host', ''), r.get('cond', ''), r.get('design_id', r.get('pid', ''))))
        gid = {s_['group'] for s_ in stats}
        meta = {k: old[k] for k in ('wall', 'when', 'python', 'cpus') if k in old}
        meta['stats'] = [s_ for s_ in old.get('stats', []) if s_['group'] not in gid] + stats
        meta['rerun'] = old.get('rerun', []) + [dict(when=time.strftime('%Y-%m-%d %H:%M UTC', time.gmtime()),
                                                     groups=sorted(gid), probes=not a.no_probes,
                                                     secs=round(time.time() - t0, 1))]
    with open(a.out, 'w') as f:
        json.dump(dict(rows=rows, **meta), f, indent=1)
    print('wrote', a.out, f"in {time.time() - t0:.0f}s")


if __name__ == '__main__':
    main()
