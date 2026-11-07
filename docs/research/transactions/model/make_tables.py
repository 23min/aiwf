#!/usr/bin/env python3
"""Render results.json and fidelity.json into tables.md and RESULTS.md
(RESULTS.template.md with its <!-- MARKERS --> filled in)."""
import itertools
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import model as M  # noqa: E402
from run import CONDS_DIRECT, DIM_ORDER, HOSTS_MR, PROBES, host_id  # noqa: E402

INVS = ('T1', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7', 'L')
HOSTS = [host_id('mr', h) for h in HOSTS_MR] + ['direct']
DEFAULT_ID = 'replay.replace.record.opid.record'
BEST_ID = 'replay.append.record.opid.record'


def marks(r):
    """One mark per invariant for one result row."""
    v = r['viol']

    def det(k):
        e = v[k]
        return '!' if e['detected'] and not e['silent'] else '✗'
    m = {}
    m['T1'] = '✗' if 'T1' in v else '✓'
    m['T2'] = '✗' if 'T2' in v else ('m' if 'T2m' in v else '✓')
    m['T3'] = det('T3') if 'T3' in v else ('x' if 'T3x' in v else '✓')
    t4 = ('✗' if 'T4' in v else '') + ('d' if 'T4d' in v else '') + ('a' if 'T4a' in v else '')
    m['T4'] = t4 or '✓'
    m['T5'] = '✗' if 'T5' in v else '✓'
    m['T6'] = det('T6') if 'T6' in v else '✓'
    m['T7'] = det('T7') if 'T7' in v else ('x' if 'T7x' in v else '✓')
    if r['stuck']:
        m['L'] = '✗'
    elif 'M' in r['refused']:
        m['L'] = 'M'
    elif 'R' in r['refused']:
        m['L'] = 'R'
    else:
        m['L'] = '✓'
    return m


def holds(inv, mark):
    """Whether a mark counts as the invariant holding. 'x' (a merge commit's own
    content, excluded from the audits today) and the L refusals R and M (each
    with an actionable remedy) count as holding; 'a' (records not attributable
    to their transaction) is reported but does not fail T4."""
    if inv == 'L':
        return mark in ('✓', 'R', 'M')
    if inv == 'T4':
        return mark in ('✓', 'a')
    return mark in ('✓', 'x', 'm')


def holds_live(inv, m):
    """The invariant holds and the run is live: a design that never lands keeps
    every safety invariant vacuously, so a safety result counts only alongside L."""
    return holds(inv, m[inv]) and (inv == 'L' or holds('L', m['L']))


def cell_short(m, complete=True):
    bad = []
    for inv in INVS:
        mk = m[inv]
        if mk == '✓':
            continue
        if inv == 'L':
            if mk == '✗':
                bad.append('L✗')
            continue
        if inv == 'T4':
            # 'a' alone holds and shows as T4a; a failure shows without it, so that
            # T4a never stands for a lost record
            if '✗' in mk:
                bad.append('T4')
            if 'd' in mk:
                bad.append('T4d')
            if mk == 'a':
                bad.append('T4a')
            continue
        bad.append(inv + ('' if mk == '✗' else mk))
    return (' '.join(bad) or '—') + ('' if complete else '*')


def load():
    res = json.load(open(os.path.join(HERE, 'results.json')))
    # only what this document reports: a results file from `run.py --amend` also holds the amend
    # designs, conditions and probes, which make_amend.py renders
    ids = set(design_ids())
    rows = [r for r in res['rows'] if r['kind'] == 'matrix' and r['design_id'] in ids
            and r['cond'] in conds_for(r['host'])]
    pids = {p[0] for p in PROBES}
    probes = [r for r in res['rows'] if r['kind'] == 'probe' and r['pid'] in pids]
    by = {}
    for r in rows:
        by[(r['host'], r['cond'], r['design_id'])] = r
    return res, rows, probes, by


def design_ids():
    out = ['perverb.parent', 'perverb.record', 'perverb.opid.parent', 'perverb.opid.record']
    for vals in itertools.product(*(M.DIMS[k] for k in DIM_ORDER[1:])):
        out.append('.'.join(vals))
    return out


def conds_for(host):
    return CONDS_DIRECT if host == 'direct' else M.CONDS


def parse(did):
    if did.startswith('perverb'):
        return dict(submit='none', audit=did.split('.')[-1])
    return dict(zip(DIM_ORDER[1:], did.split('.')), submit='fold')


def formula(ids):
    """A compact description of a set of fold design ids: the fewest
    dimensions whose values decide membership, and the satisfying values."""
    allf = [d for d in design_ids() if not d.startswith('perverb')]
    S = set(i for i in ids if not i.startswith('perverb'))
    if not S:
        return 'none'
    if S == set(allf):
        return 'all'
    dims = DIM_ORDER[1:]
    for n in range(1, len(dims) + 1):
        for D in itertools.combinations(dims, n):
            ok, sat = True, set()
            groups = {}
            for d in allf:
                key = tuple(parse(d)[k] for k in D)
                groups.setdefault(key, set()).add(d in S)
            if all(len(v) == 1 for v in groups.values()):
                sat = sorted(k for k, v in groups.items() if True in v)
                parts = []
                for k in sat:
                    parts.append(' & '.join(f"{a}={b}" for a, b in zip(D, k)))
                return ' or '.join(f"({p})" if ' & ' in p and len(sat) > 1 else p for p in parts)
    return f"{len(S)} of {len(allf)}"


def main():
    res, rows, probes, by = load()
    out = []
    ids = design_ids()
    # -------------------------------------------------- per (host, cond, inv): which designs hold
    holdset, liveset = {}, {}
    for h in HOSTS:
        for c in conds_for(h):
            for inv in INVS:
                S, SL = set(), set()
                for d in ids:
                    r = by.get((h, c, d))
                    if r and holds(inv, marks(r)[inv]):
                        S.add(d)
                    if r and holds_live(inv, marks(r)):
                        SL.add(d)
                holdset[(h, c, inv)] = S
                liveset[(h, c, inv)] = SL
    # -------------------------------------------------- the matrix: default design and today, per condition x host
    def matrix(did, title):
        out.append(f"\n#### {title}\n")
        hs = [h for h in HOSTS]
        out.append("| condition | " + " | ".join(f"`{h}`" for h in hs) + " |")
        out.append("|---|" + "---|" * len(hs))
        for c in M.CONDS:
            cells = []
            for h in hs:
                r = by.get((h, c, did))
                cells.append('n/a' if r is None else cell_short(marks(r), r['complete']))
            out.append(f"| {c} | " + " | ".join(cells) + " |")
    matrix(DEFAULT_ID, f"The proposal's default design (`{DEFAULT_ID}`)")
    matrix('perverb.parent', "Today: per-verb commits, parent-diff audit (`perverb.parent`)")
    # -------------------------------------------------- per-verb commits with operation-id pointers, against today
    out.append("\n#### Today's per-verb commits with operation-id pointers (`perverb.opid.parent`)\n")
    out.append("`perverb.parent` with `aiwf-authorized-by` written as the `authorize` operation's identity instead "
               "of its SHA. Only the cells that differ from `perverb.parent` are listed.\n")
    per = {}
    for c in M.CONDS:
        for h in HOSTS:
            if c not in conds_for(h):
                continue
            a, b = by[(h, c, 'perverb.parent')], by[(h, c, 'perverb.opid.parent')]
            sa, sb = cell_short(marks(a), a['complete']), cell_short(marks(b), b['complete'])
            if sa != sb:
                per.setdefault((c, sa, sb), []).append(h)
    if per:
        out.append("| condition | host settings | `perverb.parent` | `perverb.opid.parent` |")
        out.append("|---|---|---|---|")
        for (c, sa, sb), hs in per.items():
            out.append(f"| {c} | {', '.join(f'`{h}`' for h in hs)} | {sa} | {sb} |")
    else:
        out.append("No cell differs.")
    # the fold design with the fewest failing cells over every host and condition
    allf = [d for d in ids if not d.startswith('perverb')]
    def nfail(d):
        return sum(1 for h in HOSTS for c in conds_for(h) for inv in INVS
                   if not holds_live(inv, marks(by[(h, c, d)])))
    tot = {d: nfail(d) for d in allf}
    lo = min(tot.values())
    tops = sorted(d for d in allf if tot[d] == lo)
    for d in tops[:2]:
        if d != DEFAULT_ID:
            matrix(d, f"The fold design failing the fewest cells overall (`{d}`, {lo} cells; "
                      + ("no other design ties)" if len(tops) == 1 else f"{len(tops)} designs tie)"))
    out.append(f"\nFailing cells over all host settings and conditions, counting a safety cell as failing "
               f"also where the run is not live: default design {tot[DEFAULT_ID]}, best {lo} "
               f"({', '.join(f'`{t}`' for t in tops)}), per-verb today {nfail('perverb.parent')}, "
               f"per-verb with operation-id pointers {nfail('perverb.opid.parent')}.\n")
    # -------------------------------------------------- which designs keep each invariant, per condition x host
    out.append("\n#### Which fold designs keep each invariant\n")
    out.append("Each row names the fold designs under which the invariant holds in every host setting listed; "
               "`all` means every fold design, `none` means none does. The next column narrows that to the "
               "designs that are also live there (a design that never lands keeps every safety invariant "
               "vacuously); `(same)` means no narrowing. The last column is today's per-verb shape. A row appears "
               "only where some answer is not `all`.\n")
    out.append("| condition | invariant | host settings | fold designs that keep it | ... and stay live (L) | "
               "today (per-verb) |")
    out.append("|---|---|---|---|---|---|")
    for c in M.CONDS:
        for inv in INVS:
            per = {}
            for h in HOSTS:
                if c not in conds_for(h):
                    continue
                f = formula(holdset[(h, c, inv)])
                fl = formula(liveset[(h, c, inv)]) if inv != 'L' else '='
                pv = 'holds' if 'perverb.parent' in holdset[(h, c, inv)] else 'fails'
                per.setdefault((f, fl, pv), []).append(h)
            for (f, fl, pv), hs in sorted(per.items(), key=lambda kv: HOSTS.index(kv[1][0])):
                if f == 'all' and fl in ('all', '=') and pv == 'holds':
                    continue
                out.append(f"| {c} | {inv} | {', '.join(f'`{h}`' for h in hs)} | {f} | "
                           f"{'(same)' if fl in ('=', f) else fl} | {pv} |")
    # -------------------------------------------------- designs that keep everything any design keeps
    out.append("\n#### Ranking the fold designs, per host setting\n")
    out.append("A cell is one (condition, invariant); a safety cell counts as kept only where the run is also "
               "live. Cells that no fold design keeps are listed separately: no choice fixes them. For the rest, "
               "the fold designs that fail the fewest are listed with the cells they still fail; today's "
               "per-verb shape is scored on the same cells.\n")
    out.append("| host setting | cells no fold design keeps | best fold designs | cells they fail | "
               "per-verb today fails |")
    out.append("|---|---|---|---|---|")
    best = {}
    allf = [d for d in ids if not d.startswith('perverb')]
    for h in HOSTS:
        nokeep, cells = [], []
        for c in conds_for(h):
            for inv in INVS:
                if not (liveset[(h, c, inv)] & set(allf)):
                    nokeep.append(f"{c}:{inv}")
                else:
                    cells.append((c, inv))
        score = {d: [f"{c}:{inv}" for (c, inv) in cells if d not in liveset[(h, c, inv)]] for d in allf}
        lo = min(len(v) for v in score.values())
        top = sorted(d for d in allf if len(score[d]) == lo)
        best[h] = top
        failsets = sorted({tuple(score[d]) for d in top})
        pv = [f"{c}:{inv}" for (c, inv) in cells if 'perverb.parent' not in liveset[(h, c, inv)]]
        out.append(f"| `{h}` | {group_cells(nokeep, h)} | {formula(top)} | "
                   f"{' / '.join(group_cells(list(fs), h) for fs in failsets)} | {group_cells(pv, h)} |")
    # -------------------------------------------------- per dimension, one at a time around a reference design
    for ref, refname in ((BEST_ID, 'the best-ranked design'), (DEFAULT_ID, "the proposal's default design")):
        out.append(f"\n#### One choice at a time, around {refname} (`{ref}`)\n")
        out.append("Each row changes one dimension of the reference design. `worse`: the reference keeps the "
                   "invariant and the variant does not; `better`: the reverse. `vacuously`: the side that keeps a "
                   "safety invariant is not live in those runs, so what would break it never lands. The trace is "
                   "the failing side's shortest violation.\n")
        out.append("| dimension | variant | invariant | worse or better | conditions | host settings | trace |")
        out.append("|---|---|---|---|---|---|---|")
        pr = parse(ref)
        for k in DIM_ORDER[1:]:
            for v in M.DIMS[k]:
                if v == pr[k]:
                    continue
                var = '.'.join(dict(pr, **{k: v})[x] for x in DIM_ORDER[1:])
                found = {}
                for h in HOSTS:
                    for c in conds_for(h):
                        m1, m2 = marks(by[(h, c, ref)]), marks(by[(h, c, var)])
                        for inv in INVS:
                            a, b = holds(inv, m1[inv]), holds(inv, m2[inv])
                            if a == b:
                                continue
                            if a:
                                way = 'worse' if (inv == 'L' or holds('L', m1['L'])) else \
                                    'worse, but the reference keeps it only vacuously'
                                r = by[(h, c, var)]
                            else:
                                way = 'better' if (inv == 'L' or holds('L', m2['L'])) else 'better, vacuously'
                                r = by[(h, c, ref)]
                            found.setdefault((inv, way, c), [[], trace_file(r, inv)])[0].append(h)
                if not found:
                    out.append(f"| {k} | {v} | — | no difference | — | — | — |")
                    continue
                merged = {}
                for (inv, way, c), (hs, tr) in found.items():
                    allh = [h for h in HOSTS if c in conds_for(h)]
                    hstr = 'all' if len(hs) == len(allh) else ', '.join(f"`{h}`" for h in hs)
                    e = merged.setdefault((inv, way, hstr), [[], tr])
                    e[0].append(c)
                for (inv, way, hstr), (cs, tr) in sorted(merged.items(), key=lambda kv: (INVS.index(kv[0][0]),
                                                                                         kv[0][1])):
                    cs = [c for c in M.CONDS if c in cs]
                    out.append(f"| {k} | {v} | {inv} | {way} | {', '.join(cs)} | {hstr} | `{tr}` |")
    # -------------------------------------------------- probes
    out.append("\n#### Mutation probes\n")
    out.append("| probe | invariant | what is broken | run | probe run fails | control run fails | trace |")
    out.append("|---|---|---|---|---|---|---|")
    for p in sorted(probes, key=lambda p: (p['pid'][0] == 'W', int(p['pid'][1:]))):
        def fl(x):
            s = sorted(set(x['viol']) | ({'L'} if x['stuck'] else set()))
            return ', '.join(s) or 'none'
        inv = p['inv']
        tr = trace_file(p['probe'], inv)
        out.append(f"| {p['pid']} | {inv} | {p['what']} | `{p['host']}@{p['cond']}` | {fl(p['probe'])} | "
                   f"{fl(p['control'])} | {f'`{tr}`' if tr else '—'} |")
    # -------------------------------------------------- statistics
    runs = {}
    for r in rows:
        runs.setdefault(r['run'], r)
    for p in probes:
        for t in ('probe', 'control'):
            runs.setdefault(p[t]['run'], p[t])
    out.append("\n#### Run statistics\n")
    tot = sum(r['states'] for r in runs.values())
    trunc = [r for r in runs.values() if not r['complete']]
    nonacyc = [r for r in runs.values() if r.get('acyclic') is False]
    big = sorted(runs.values(), key=lambda r: -r['states'])[:12]
    nd = len({(r['host'], r['cond'], r['design_id']) for r in rows})
    out.append(f"{len(runs)} distinct explorations (matrix and probes) produced the {nd:,} (host setting, condition, "
               f"design) results; a result whose design agrees with an explored one on every dimension that run "
               f"read is taken from it. {tot:,} states in total; {len(trunc)} runs hit the state cap; "
               f"{len(nonacyc)} state graphs had a cycle. Largest runs:\n")
    out.append("| run | states | BFS depth | complete | secs |")
    out.append("|---|---|---|---|---|")
    for r in big:
        out.append(f"| `{r['run']}` | {r['states']:,} | {r['max_depth']} | {'yes' if r['complete'] else 'NO'} | "
                   f"{r['secs']} |")
    peak = max(s['peak_mb'] for s in res['stats'])
    out.append(f"\nPeak resident memory of one worker: {peak:,} MB; wall time {res['wall']} s with the worker count "
               f"given on the command line; Python {res['python']}, {res['cpus']} CPUs, run {res['when']}.\n")
    for rr in res.get('rerun', []):
        out.append(f"Re-run into the same results on {rr['when']} ({rr['secs']} s): groups "
                   f"{', '.join(f'`{g}`' for g in rr['groups'])}{' and the probes' if rr['probes'] else ''}; "
                   f"their numbers above are from that run.\n")
    with open(os.path.join(HERE, 'tables.md'), 'w') as f:
        f.write('\n'.join(out) + '\n')
    fill(res, rows, probes, by, best, runs, trunc, tot, peak, out)
    print('\n'.join(out[:60]))


def group_cells(cells, h):
    """'cond:inv' cells grouped per condition; a condition whose every cell is
    listed is shown as 'cond (all)'."""
    if not cells:
        return '—'
    by = {}
    for x in cells:
        c, inv = x.split(':')
        by.setdefault(c, []).append(inv)
    parts = []
    for c in [c for c in M.CONDS if c in by]:
        invs = by[c]
        parts.append(f"{c} (all)" if len(invs) == len(INVS) else f"{c} {' '.join(invs)}")
    return '; '.join(parts)


def trace_file(r, inv):
    if inv == 'L':
        return r['stuck']['trace'] if r.get('stuck') else ''
    v = r['viol']
    for k in (inv, inv + 'd', inv + 'x', inv + 'a', inv + 'm'):
        if k in v:
            e = v[k]
            return e.get('silent_trace') or e['trace']
    if inv == 'T4d' and 'T4d' in v:
        return v['T4d']['trace']
    return ''


def fill(res, rows, probes, by, best, runs, trunc, tot, peak, out):
    tpl = os.path.join(HERE, 'RESULTS.template.md')
    if not os.path.exists(tpl):
        return
    text = open(tpl).read()
    text = text.replace('<!-- TABLES -->', '\n'.join(out[:out.index("\n#### Run statistics\n")]).strip())
    text = text.replace('<!-- STATS -->', '\n'.join(out[out.index("\n#### Run statistics\n"):]).strip())
    text = text.replace('<!-- NRUNS -->', str(len(runs)))
    text = text.replace('<!-- NROWS -->', f"{len({(r['host'], r['cond'], r['design_id']) for r in rows}):,}")
    text = text.replace('<!-- NSTATES -->', f"{tot:,}")
    text = text.replace('<!-- NTRUNC -->', str(len(trunc)))
    text = text.replace('<!-- NCYC -->', str(sum(1 for r in runs.values() if r.get('acyclic') is False)))
    text = text.replace('<!-- PEAK -->', f"{peak:,}")
    text = text.replace('<!-- WALL -->', str(res['wall']))
    text = text.replace('<!-- WHEN -->', res['when'])
    fid = json.load(open(os.path.join(HERE, 'fidelity.json')))
    t = ["| shape | host merge | commits added (model / spike) | first-parent | aiwf-verb values | c1 SHA kept | match |",
         "|---|---|---|---|---|---|---|"]
    for e in fid['e4']:
        mo, sp = e['model'], e['spike']
        t.append(f"| {e['shape']} | {e['host']} | {mo[0]} / {sp[0]} | {mo[1]} / {sp[1]} | {mo[2]} / {sp[2]} | "
                 f"{mo[3]} / {sp[3]} | {'yes' if e['match'] else '**no**'} |")
    text = text.replace('<!-- E4 -->', '\n'.join(t))
    e1 = fid['e1']
    text = text.replace('<!-- E1 -->', (
        f"final tree identical: {e1['same_tree']}; code patches identical: {e1['patch_same']}; "
        f"code SHAs kept: {e1['sha_same']}; the planning commit's record, in order: "
        f"{', '.join(f'{o} ({a})' for o, a in e1['record'])}"))
    text = text.replace('<!-- FIDELITY -->', fidelity_block(by, probes))
    text = text.replace('<!-- RULE -->', rule_block(rows))
    with open(os.path.join(HERE, 'RESULTS.md'), 'w') as f:
        f.write(text)


def rule_block(rows):
    """The push rules for submit across the matrix, from rule_w2.json."""
    p = os.path.join(HERE, 'rule_w2.json')
    if not os.path.exists(p):
        return '`rule_w2.json` is absent; run `python3 -I rule_w2.py`.'
    w = json.load(open(p))

    def fails(viol, stuck):
        # the matrices' tokens without their reporting marks and without T4's 'a',
        # which holds; x and m marks kept, L✗ when stuck
        return (set(viol) - {'T4a'}) | ({'L✗'} if stuck else set())

    def show(f):
        return ' '.join(sorted(f)) or '—'

    hosts = list(dict.fromkeys(r['host'] for r in rows if r['kind'] == 'matrix'))
    got = {(r['design'], r['host'], r['cond'], r['variant']): r for r in w['rows']}
    designs = list(dict.fromkeys(r['design'] for r in w['rows']))
    variants = [v for v in w['variants'] if v != 'none']
    today = {(r['host'], r['cond']): fails(r['viol'], r.get('stuck'))
             for r in rows if r['kind'] == 'matrix' and r['design_id'] == 'perverb.parent'}

    def cells(d, v):
        for c in M.CONDS:
            for h in hosts:
                if (d, h, c, v) in got:
                    yield h, c, got[(d, h, c, v)]

    out = [f"Run {w['when']}: {len(w['rows'])} runs, {sum(not r['complete'] for r in w['rows'])} hit the state cap, "
           f"{sum(not r['acyclic'] for r in w['rows'])} state graphs have a cycle.", '',
           'Cells each rule changes against no rule. Each lists what fails, with the matrices\' tokens but without '
           'their `!`/`✗` reporting marks and without T4\'s `a`, which holds; a cell no rule changes is not listed.', '',
           '| rule | design | condition | without a rule | with it | host settings |', '|---|---|---|---|---|---|']
    for v in variants:
        for d in designs:
            groups = {}
            for h, c, b in cells(d, v):
                a = got[(d, h, c, 'none')]
                fa, fb = fails(a['viol'], a['stuck']), fails(b['viol'], b['stuck'])
                if fa != fb:
                    groups.setdefault((c, show(fa), show(fb)), []).append(h)
            for (c, fa, fb), hs in groups.items():
                out.append(f"| {v} | `{d}` | {c} | {fa} | {fb} | {', '.join(f'`{h}`' for h in hs)} |")
    out.append('')
    if len(designs) == 2:
        for v in variants:
            diff = []
            for h, c, a in cells(designs[0], v):
                b = got[(designs[1], h, c, v)]
                fa, fb = fails(a['viol'], a['stuck']), fails(b['viol'], b['stuck'])
                if fa != fb:
                    diff.append(f"`{h}` {c}: `{designs[0]}` fails {show(fa)}, `{designs[1]}` fails {show(fb)}")
            out.append(f"Under {v}, the two designs differ in: " + ('; '.join(diff) if diff else 'no cell') + '.')
            out.append('')
    same = []
    for i, v in enumerate(variants):
        for u in variants[i + 1:]:
            if all((sorted(got[(d, h, c, v)]['viol']), got[(d, h, c, v)]['stuck']) ==
                   (sorted(got[(d, h, c, u)]['viol']), got[(d, h, c, u)]['stuck'])
                   for d in designs for h, c, _ in cells(d, v)):
                same.append(f"{v} and {u}")
    if same:
        out.append('Rules that agree on every invariant and on liveness in every cell, for both designs: '
                   + '; '.join(same) + '.')
        out.append('')
    out.append('Refusals each rule adds against no rule (the model counts each as actionable, so no table above '
               'shows them): the number of cells where the refusal ends some run under the rule, and how many of '
               'those also have it without a rule.')
    out.append('')
    out.append('| rule | design | refusal | cells with the rule | of them also without it |')
    out.append('|---|---|---|---|---|')
    for v in variants:
        for d in designs:
            cnt = {}
            for h, c, r in cells(d, v):
                before = set(got[(d, h, c, 'none')]['refusals'])
                for why in r['refusals']:
                    n = cnt.setdefault(why, [0, 0])
                    n[0] += 1
                    n[1] += why in before
            for why, (n, m) in sorted(cnt.items()):
                if n != m:
                    out.append(f"| {v} | `{d}` | {why} | {n} | {m} |")
    out.append('')
    force = {}
    for r in w['rows']:
        force[(r['design'], r['variant'])] = force.get((r['design'], r['variant']), 0) + r['force_after_submit']
    out.append('Explored submits that make the next push need force on a branch that did not need it (the remote '
               'branch was an ancestor of the tip before the submit and is not after it), summed over every run: ' +
               '; '.join(f"`{d}` " + ', '.join(f"{force[(d, v)]:,} under {v}" for v in w['variants']) for d in designs)
               + '.')
    out.append('')
    for v in variants:
        worse = []
        for d in designs:
            for h, c, b in cells(d, v):
                if (h, c) in today:
                    extra = fails(b['viol'], b['stuck']) - today[(h, c)]
                    if extra:
                        worse.append(f"`{d}`, `{h}` {c}: {show(extra)}")
        out.append(f"Under {v}, cells where a design fails something today's per-verb commits (`perverb.parent`) "
                   f"keep: " + ('; '.join(worse) if worse else 'none') + '.')
        out.append('')
    return '\n'.join(out).rstrip()


def fidelity_block(by, probes):
    def row(h, c, d):
        return by[(h, c, d)]

    def ent(r, inv):
        e = r['viol'].get(inv)
        if not e:
            return 'not found'
        return (f"{e['detail']} ({e['steps']} steps; `aiwf check` on main afterwards "
                f"{'reports it' if e['detected'] and not e['silent'] else 'reports nothing'}; "
                f"`traces/{e.get('silent_trace') or e['trace']}`)")
    L = []
    r = row('merge', 'base', 'copy.replace.record.opid.record')
    L.append("| check | §8 measured | the model, from its own rules |")
    L.append("|---|---|---|")
    L.append(f"| R1, copy reverts other planning | the planning commit deleted the other transaction's gap; "
             f"`aiwf show` said \"not found\"; `aiwf check --since base` reported nothing | `fold=copy`, `merge`, "
             f"`base`: T2: {ent(r, 'T2')}; T3: {ent(r, 'T3')} |")
    r = row('merge', 'auth', 'replay.replace.record.leave.record')
    st = r['stuck']
    L.append(f"| R2, authorization pointer to a commit that never lands | the check on the transaction's branch "
             f"failed with `provenance-authorization-missing` | `ptr=leave`, `merge`, `auth`: stuck with "
             f"{st['detail'] if st else 'no stuck state'} (`traces/{st['trace'] if st else ''}`) |")
    r = row('merge', 'resolve', DEFAULT_ID)
    L.append(f"| R4, resolution in a merge commit | a conflict resolution that also moved a milestone draft → done "
             f"passed with no errors | `merge`, `resolve`: T7x: {ent(r, 'T7x')}; T3x: {ent(r, 'T3x')} |")
    r = row('merge', 'revert', DEFAULT_ID)
    L.append(f"| R4, host revert | untrailered entity changes and errors | `merge`, `revert`: T3: {ent(r, 'T3')}; "
             f"T7: {ent(r, 'T7')} |")
    r = row('merge', 'mixed', 'perverb.parent')
    ref = r['refused'].get('R')
    L.append(f"| E3, `git add -A` sweeps planning into a code commit | hooks accepted the commit; a first push "
             f"with no upstream went through; `aiwf check --since base` raised "
             f"`provenance-untrailered-entity-commit` | per-verb, `merge`, `mixed`: refused, "
             f"{ref['detail'] if ref else 'not refused'} (`traces/{ref['trace'] if ref else ''}`) |")
    p5 = [p for p in probes if p['pid'] == 'P5'][0]
    L.append(f"| E5 probe-drop: a record naming only the last operation | `aiwf check` reported nothing | "
             f"P5: T3: {ent(p5['probe'], 'T3')} |")
    return '\n'.join(L)


if __name__ == '__main__':
    main()
