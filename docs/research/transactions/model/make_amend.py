#!/usr/bin/env python3
"""Render the amend results: results.json from `run.py --amend` (and traces/)
into RESULTS-amend.md (RESULTS-amend.template.md with its <!-- MARKERS -->
filled in). Cells are marked exactly as make_tables.py marks them."""
import json
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import model as M  # noqa: E402
from run import (AMEND_COMPARE, AMEND_PROBES, CONDS_DIRECT, CONDS_DIRECT_AMEND, designs,  # noqa: E402
                 design_id)
from make_tables import BEST_ID, DEFAULT_ID, HOSTS, INVS, cell_short, holds, holds_live, marks  # noqa: E402

AMEND_IDS = [design_id(d) for d in designs(True) if d['submit'] == 'amend']
PERVERB_IDS = ['perverb.parent', 'perverb.record', 'perverb.opid.parent', 'perverb.opid.record']
REF = 'amend.reflog.opid.record'
NAIVE_TODAY = 'amend.naive.parent'


def orig_conds(h):
    return CONDS_DIRECT if h == 'direct' else M.CONDS


def amend_conds(h):
    return CONDS_DIRECT_AMEND if h == 'direct' else M.AMEND_CONDS


def all_conds(h):
    return tuple(orig_conds(h)) + tuple(amend_conds(h))


ROW_CONDS = list(M.CONDS) + list(M.AMEND_CONDS)


def counterpart(did):
    """The per-verb design with the same pointers and audit as an amend design."""
    parts = did.split('.')
    return 'perverb.' + '.'.join(parts[2:])


def parse_amend(did):
    parts = did.split('.')
    return dict(amend=parts[1], ptr='opid' if parts[2] == 'opid' else 'leave', audit=parts[-1])


def amend_id(amend, ptr, audit):
    return f"amend.{amend}.{'opid.' if ptr == 'opid' else ''}{audit}"


def load():
    res = json.load(open(os.path.join(HERE, 'results.json')))
    by = {}
    for r in res['rows']:
        if r['kind'] == 'matrix':
            by[(r['host'], r['cond'], r['design_id'])] = r
    probes = [r for r in res['rows'] if r['kind'] == 'probe']
    return res, by, probes


def cell(by, h, c, d):
    r = by.get((h, c, d))
    if r is None:
        return 'n/a'
    return cell_short(marks(r), r['complete'])


def matrix(by, did, title, conds=ROW_CONDS):
    out = [f"\n#### {title}\n", "| condition | " + " | ".join(f"`{h}`" for h in HOSTS) + " |",
           "|---|" + "---|" * len(HOSTS)]
    for c in conds:
        out.append(f"| {c} | " + " | ".join(cell(by, h, c, did) for h in HOSTS) + " |")
    return out


def nfail(by, did, condsf):
    return sum(1 for h in HOSTS for c in condsf(h) for inv in INVS
               if (h, c, did) in by and not holds_live(inv, marks(by[(h, c, did)])))


def trace_of(r, inv):
    if inv == 'L':
        return (r['stuck']['trace'], r['stuck']['steps']) if r.get('stuck') else (None, None)
    for k in (inv, inv + 'd', inv + 'x', inv + 'm'):
        if k in r['viol']:
            e = r['viol'][k]
            return e['trace'], e['steps']
    return None, None


def failing_tokens(r):
    """The failure tokens of a cell, as cell_short prints them, without marks."""
    m = marks(r)
    out = set()
    for inv in INVS:
        if inv == 'L':
            if m['L'] == '✗':
                out.add('L')
        elif inv == 'T4':
            if '✗' in m['T4']:
                out.add('T4')
            if 'd' in m['T4']:
                out.add('T4d')
        elif not holds(inv, m[inv]):
            out.add(inv)
    return out


def digest(fn):
    """The numbered action lines of a trace file and its violation line."""
    p = os.path.join(HERE, 'traces', fn)
    if not os.path.exists(p):
        return [f"(trace `{fn}` not found)"]
    lines = open(p).read().splitlines()
    out = [x for x in lines if x.startswith('violation:')]
    out += [x for x in lines if re.match(r'^\d+\. ', x)]
    return out


def main():
    res, by, allprobes = load()
    probes = [p for p in allprobes if p['pid'] in {x[0] for x in AMEND_PROBES}]
    T = {}

    # ------------------------------------------------ failing-cell counts
    def counts(did):
        return nfail(by, did, orig_conds), nfail(by, did, amend_conds)
    rows = ["| design | failing cells, original conditions | failing cells, amend conditions |", "|---|---|---|"]
    for did in AMEND_IDS + PERVERB_IDS + [BEST_ID, DEFAULT_ID]:
        a, b = counts(did)
        rows.append(f"| `{did}` | {a} | {b} |")
    T['COUNTS'] = '\n'.join(rows)

    # ------------------------------------------------ matrices
    out = []
    out += matrix(by, NAIVE_TODAY, f"The review's naive rule with today's SHA pointers and parent-diff audit (`{NAIVE_TODAY}`)")
    out += matrix(by, REF, f"The corrected rule with operation-id pointers and the record audit (`{REF}`)")
    out += matrix(by, 'perverb.parent', "Today: per-verb commits, parent-diff audit (`perverb.parent`), amend conditions",
                  list(M.AMEND_CONDS))
    out += matrix(by, 'perverb.opid.record', "Per-verb commits with operation-id pointers and the record audit "
                  "(`perverb.opid.record`), amend conditions", list(M.AMEND_CONDS))
    out += matrix(by, BEST_ID, f"The best fold design (`{BEST_ID}`), amend conditions", list(M.AMEND_CONDS))
    T['MATRICES'] = '\n'.join(out).strip()

    # ------------------------------------------------ each amend design against its per-verb counterpart
    out = ["| amend design | condition | amend fails | per-verb fails | host settings |", "|---|---|---|---|---|"]
    for did in AMEND_IDS:
        pv = counterpart(did)
        groups = {}
        for c in ROW_CONDS:
            for h in HOSTS:
                if c not in all_conds(h) or (h, c, did) not in by:
                    continue
                a, b = cell(by, h, c, did), cell(by, h, c, pv)
                if a != b:
                    groups.setdefault((c, a, b), []).append(h)
        if not groups:
            out.append(f"| `{did}` | — | no cell differs from `{pv}` | | |")
        for (c, a, b), hs in groups.items():
            out.append(f"| `{did}` | {c} | {a} | {b} (`{pv}`) | {', '.join(f'`{h}`' for h in hs)} |")
    T['VSPERVERB'] = '\n'.join(out)

    # ------------------------------------------------ the amend design closest to the fold, against the best fold
    out = ["| condition | `" + REF + "` fails | `" + BEST_ID + "` fails | host settings |", "|---|---|---|---|"]
    groups = {}
    for c in ROW_CONDS:
        for h in HOSTS:
            if c not in all_conds(h) or (h, c, REF) not in by or (h, c, BEST_ID) not in by:
                continue
            a, b = cell(by, h, c, REF), cell(by, h, c, BEST_ID)
            if a != b:
                groups.setdefault((c, a, b), []).append(h)
    for (c, a, b), hs in groups.items():
        out.append(f"| {c} | {a} | {b} | {', '.join(f'`{h}`' for h in hs)} |")
    T['VSFOLD'] = '\n'.join(out)

    # ------------------------------------------------ one choice at a time around REF
    out = ["| dimension | variant | invariant | worse or better | conditions | host settings | trace |",
           "|---|---|---|---|---|---|---|"]
    pr = parse_amend(REF)
    for k, vals in (('amend', M.AMEND_DIMS['amend']), ('ptr', M.AMEND_DIMS['ptr']), ('audit', M.AMEND_DIMS['audit'])):
        for v in vals:
            if v == pr[k]:
                continue
            var = amend_id(**dict(pr, **{k: v}))
            found = {}
            for h in HOSTS:
                for c in all_conds(h):
                    m1, m2 = marks(by[(h, c, REF)]), marks(by[(h, c, var)])
                    for inv in INVS:
                        a, b = holds(inv, m1[inv]), holds(inv, m2[inv])
                        if a == b:
                            continue
                        if a:
                            way = 'worse' if (inv == 'L' or holds('L', m1['L'])) else 'worse, vacuously'
                            r = by[(h, c, var)]
                        else:
                            way = 'better' if (inv == 'L' or holds('L', m2['L'])) else 'better, vacuously'
                            r = by[(h, c, REF)]
                        found.setdefault((inv, way, c), [[], trace_of(r, inv)[0]])[0].append(h)
            if not found:
                out.append(f"| {k} | {v} | — | no difference | — | — | — |")
                continue
            merged = {}
            for (inv, way, c), (hs, tr) in found.items():
                allh = [h for h in HOSTS if c in all_conds(h)]
                hstr = 'all' if len(hs) == len(allh) else ', '.join(f"`{h}`" for h in hs)
                merged.setdefault((inv, way, hstr), [[], tr])[0].append(c)
            for (inv, way, hstr), (cs, tr) in sorted(merged.items(), key=lambda kv: (INVS.index(kv[0][0]), kv[0][1])):
                cs = [c for c in ROW_CONDS if c in cs]
                out.append(f"| {k} | {v} | {inv} | {way} | {', '.join(cs)} | {hstr} | `{tr}` |")
    T['ONEATATIME'] = '\n'.join(out)

    # ------------------------------------------------ failure classes: amend fails, its per-verb counterpart keeps
    classes = {}
    for did in AMEND_IDS:
        pv = counterpart(did)
        for h in HOSTS:
            for c in all_conds(h):
                if (h, c, did) not in by:
                    continue
                ra, rp = by[(h, c, did)], by[(h, c, pv)]
                extra = failing_tokens(ra) - failing_tokens(rp)
                for tok in extra:
                    inv = {'T4d': 'T4'}.get(tok, tok)
                    fn, steps = trace_of(ra, inv) if tok != 'T4d' else (ra['viol']['T4d']['trace'],
                                                                       ra['viol']['T4d']['steps'])
                    key = (tok, c)
                    e = classes.setdefault(key, dict(designs=set(), hosts=set(), best=None))
                    e['designs'].add(did)
                    e['hosts'].add(h)
                    if fn and (e['best'] is None or steps < e['best'][0]):
                        e['best'] = (steps, fn, h, did)
    out = []
    for (tok, c), e in sorted(classes.items(), key=lambda kv: (kv[0][1] not in M.AMEND_CONDS, kv[0])):
        ds = sorted(e['designs'], key=AMEND_IDS.index)
        hs = [h for h in HOSTS if h in e['hosts']]
        out.append(f"\n**{tok} in `{c}`**, where the per-verb design with the same pointers and audit keeps it. "
                   f"Amend designs: {', '.join(f'`{d}`' for d in ds)}. Host settings: "
                   f"{', '.join(f'`{h}`' for h in hs)}.")
        if e['best']:
            steps, fn, h, did = e['best']
            out.append(f"Shortest: `{fn}` ({steps} steps; `{h}`, `{did}`):\n")
            out.append('```')
            out += digest(fn)
            out.append('```')
    T['CLASSES'] = '\n'.join(out).strip() if out else 'No cell where an amend design fails what its per-verb counterpart keeps.'

    # ------------------------------------------------ the 'sa-' conditions leave non-amend designs unchanged
    diffs = []
    n = 0
    for did in AMEND_COMPARE:
        for h in HOSTS:
            for c in M.SA_BASE:
                if c not in orig_conds(h):
                    continue
                a, b = by.get((h, c, did)), by.get((h, 'sa-' + c, did))
                if a is None or b is None:
                    continue
                n += 1
                if (cell_short(marks(a), a['complete']), a['states']) != (cell_short(marks(b), b['complete']),
                                                                          b['states']):
                    diffs.append(f"`{h}` {c} `{did}`")
    T['SACHECK'] = (f"For the per-verb and fold designs the amend conditions run, every `sa-` cell equals its "
                    f"original condition's cell, state count included, in all {n} cells compared."
                    if not diffs else f"`sa-` cells that differ from their original condition: {'; '.join(diffs)}.")

    # ------------------------------------------------ probes
    out = ["| probe | invariant | what is changed | run | probe run fails | control run fails | trace |",
           "|---|---|---|---|---|---|---|"]
    for p in sorted(probes, key=lambda p: p['pid']):
        def fl(x):
            s = sorted(set(x['viol']) | ({'L'} if x['stuck'] else set()))
            return ', '.join(s) or 'none'
        tr = trace_of(p['probe'], p['inv'])[0] or trace_of(p['control'], p['inv'])[0]
        out.append(f"| {p['pid']} | {p['inv']} | {p['what']} | `{p['host']}@{p['cond']}` | {fl(p['probe'])} | "
                   f"{fl(p['control'])} | {f'`{tr}`' if tr else '—'} |")
    T['PROBES'] = '\n'.join(out)

    # ------------------------------------------------ statistics of the runs this document reads
    runs = {}
    for (h, c, d), r in by.items():
        if d in AMEND_IDS or c in M.AMEND_CONDS or c in CONDS_DIRECT_AMEND:
            runs.setdefault(r['run'], r)
    for p in probes:
        for t in ('probe', 'control'):
            runs.setdefault(p[t]['run'], p[t])
    tot = sum(r['states'] for r in runs.values())
    trunc = [r for r in runs.values() if not r['complete']]
    cyc = [r for r in runs.values() if r.get('acyclic') is False]
    orig = {r['run'] for (h, c, d), r in by.items() if d not in AMEND_IDS and c in orig_conds(h)}
    orig |= {p[t]['run'] for p in allprobes if p not in probes for t in ('probe', 'control')}
    nrows = len({k for k in by if k[2] in AMEND_IDS or k[1] in M.AMEND_CONDS or k[1] in CONDS_DIRECT_AMEND})
    big = sorted(runs.values(), key=lambda r: -r['states'])[:8]
    st = [f"{len(runs)} distinct explorations (amend designs in every group, every design in the amend conditions, "
          f"and the amend probes) produced the {nrows:,} (host setting, condition, design) results this document "
          f"adds; {tot:,} states in total; {len(trunc)} runs hit the state cap; {len(cyc)} state graphs had a cycle. "
          f"The same run re-ran the original matrix and probes ({len(orig)} explorations), whose results are "
          f"RESULTS.md's. Largest runs:", '',
          "| run | states | BFS depth | complete | secs |", "|---|---|---|---|---|"]
    for r in big:
        st.append(f"| `{r['run']}` | {r['states']:,} | {r['max_depth']} | {'yes' if r['complete'] else 'NO'} | "
                  f"{r['secs']} |")
    T['STATS'] = '\n'.join(st)
    T['WHEN'] = res['when']
    T['WALL'] = str(res['wall'])
    T['PEAK'] = f"{max(s['peak_mb'] for s in res['stats']):,}"
    T['CPUS'] = str(res['cpus'])
    T['JOBS'] = str(res.get('jobs', ''))
    T['PYTHON'] = res['python']
    T['NRUNS'] = str(len(runs))
    T['NTRUNC'] = str(len(trunc))
    T['NCYC'] = str(len(cyc))

    tpl = open(os.path.join(HERE, 'RESULTS-amend.template.md')).read()
    for k, v in T.items():
        tpl = tpl.replace(f'<!-- {k} -->', v)
    left = re.findall(r'<!-- [A-Z]+ -->', tpl)
    if left:
        raise SystemExit(f"unfilled markers: {left}")
    with open(os.path.join(HERE, 'RESULTS-amend.md'), 'w') as f:
        f.write(tpl)
    print('wrote RESULTS-amend.md')


if __name__ == '__main__':
    main()
