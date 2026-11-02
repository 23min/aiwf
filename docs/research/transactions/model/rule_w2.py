#!/usr/bin/env python3
"""Run the push rules for submit across the whole matrix.

  python3 -I rule_w2.py [--jobs N] [--time-limit SECS]

Variants, each a set of the model's what-if flags:

  none     no push rule (the matrix's own behaviour)
  W2       submit refuses when an operation it would fold already has its record
           on the transaction's remote branch, judged by operation identity
  W2+merge W2, and a transaction whose submit W2 refuses is merged as its
           per-verb commits once its local tip is pushed
  partial  submit folds only the operations not yet on the remote branch and
           leaves the pushed ones as they are; the operator merges once
           everything is pushed and nothing is left to fold

For both re-submit values of the design replay.*.record.opid.record, every
(host setting, condition) group runs under every variant. Each run also counts
the submits that turn a branch whose remote is an ancestor of its tip into one
whose remote is not: submits that make the next push need force on a branch
that did not need it. Writes rule_w2.json next to this file; make_tables.py
reads it.
"""
import argparse
import json
import multiprocessing as mp
import os
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import model as M  # noqa: E402
import run as R  # noqa: E402

VARIANTS = {
    'none': (),
    'W2': ('strict-unpushed-ops',),
    'W2+merge': ('strict-unpushed-ops', 'w2-merge-unsubmitted'),
    'partial': ('fold-unpushed',),
}
RESUBS = ('replace', 'append')

_fold = M.Model.fold
_ready = M.Model.ready


def counting_fold(self, i, st, nonatomic=False):
    out = _fold(self, i, st, nonatomic)
    if getattr(self, 'in_ready', False):
        return out          # the operator's readiness test calls fold too; it is not a submit
    tx = st.tx[i]
    if out and out[0] == 'ok' and tx.rtb is not None and tx.rtb in M.anc(tx.loc) and tx.rtb not in M.anc(out[1]):
        self.force_after_submit = getattr(self, 'force_after_submit', 0) + 1
    return out


def flagged_ready(self, i, st):
    self.in_ready = True
    try:
        return _ready(self, i, st)
    finally:
        self.in_ready = False


M.Model.fold = counting_fold
M.Model.ready = flagged_ready


def one(args):
    g, resub, variant, time_limit = args
    d = dict(R.DEFAULT, resub=resub)
    cfg = M.Cfg(mode=g['mode'], method=g['host'][0], sqmsg=g['host'][1], utd=g['host'][2], cond=g['cond'],
                design=tuple(sorted(d.items())), probes=frozenset(VARIANTS[variant]))
    r = M.explore(cfg, time_limit=time_limit)
    return dict(host=R.host_id(g['mode'], g['host']), cond=g['cond'], design=R.design_id(d), variant=variant,
                viol=sorted(r.viol), stuck=r.stuck is not None, refused=sorted(r.refused),
                refusals=sorted({why for _, reasons in r.refused.values() for _, whys, *_ in reasons for why in whys}),
                complete=not r.truncated, acyclic=r.acyclic, states=r.nstates,
                force_after_submit=getattr(r.model, 'force_after_submit', 0))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--jobs', type=int, default=os.cpu_count())
    ap.add_argument('--time-limit', type=float, default=600)
    a = ap.parse_args()
    jobs = [(g, rs, v, a.time_limit) for g in R.groups() for rs in RESUBS for v in VARIANTS]
    t0 = time.time()
    with mp.Pool(a.jobs) as pool:
        rows = list(pool.imap_unordered(one, jobs))
    out = dict(variants={k: list(v) for k, v in VARIANTS.items()}, rows=rows, wall=round(time.time() - t0, 1),
               when=time.strftime('%Y-%m-%d %H:%M UTC', time.gmtime()))
    with open(os.path.join(HERE, 'rule_w2.json'), 'w') as f:
        json.dump(out, f, indent=1)
    print(f"wrote rule_w2.json: {len(rows)} runs in {out['wall']}s; "
          f"incomplete {sum(not r['complete'] for r in rows)}, cyclic {sum(not r['acyclic'] for r in rows)}")


if __name__ == '__main__':
    main()
