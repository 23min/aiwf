#!/usr/bin/env python3
"""Replay the spike's E1/E4 fixture through the model's own actions and compare
with what the spike measured (the proposal's §8, E1 and E4). The spike's scripts and
output are not committed, so its E4 counts are typed in below from that output.

  python3 -I fidelity.py      # writes fidelity.json next to this file

The fixture: three code commits interleaved with four operations (add an AC,
start the milestone, bob edits the gap's body, mark the AC met); the default
branch moves meanwhile (another transaction adds a gap). Each merge method is
then applied by the model's host, and the commits it adds are counted.
"""
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import model as M  # noqa: E402

# spike E4: commits added, first-parent, aiwf-verb values, c1 SHA kept
E4 = {
    ('perverb', 'merge'): (8, 1, 4, True), ('perverb', 'semi'): (8, 1, 4, False),
    ('perverb', 'ff'): (7, 7, 4, False), ('perverb', 'squash/title'): (1, 1, 0, None),
    ('perverb', 'squash/all'): (1, 1, 1, None),
    ('planning', 'merge'): (5, 1, 4, True), ('planning', 'semi'): (5, 1, 4, False),
    ('planning', 'ff'): (4, 4, 4, False), ('planning', 'squash/title'): (1, 1, 0, None),
    ('planning', 'squash/all'): (1, 1, 4, None),
}
FOLD = dict(submit='fold', fold='replay', resub='replace', txid='record', ptr='opid', audit='record')
NONE = dict(FOLD, submit='none')


def drive(cfg, labels):
    m = M.Model(cfg)
    st = m.initial()
    hist = [st]
    for want in labels:
        succ = m.successors(st)
        hits = [(a, ns) for (a, ns, evs) in succ if a.startswith(want) and 'FAULT' not in a]
        if not hits:
            raise SystemExit(f"no successor starting {want!r}; have {[a for a, _, _ in succ]}")
        st = hits[0][1]
        hist.append(st)
    return m, hist


def first_parent_count(new, old):
    n, c = 0, new
    while c not in M.anc(old):
        n += 1
        c = M.C(c).parents[0]
    return n


def main():
    out = {'e4': [], 'e1': None}
    A_steps = ['A: code a1', 'A: op E_ac', 'A: code a2', 'A: op A_ip', 'A: code a3', 'A: op A_gb', 'A: op E_met']
    for shape, design in (('perverb', NONE), ('planning', FOLD)):
        for host in ('merge', 'semi', 'ff', 'squash/title', 'squash/all'):
            method, sq = (host.split('/') + ['all'])[:2]
            cfg = M.Cfg(mode='mr', method=method, sqmsg=sq, cond='e4', design=tuple(sorted(design.items())))
            labels = list(A_steps)
            if shape == 'planning':
                labels.append('A: submit')
            labels += ['B: code b1', 'B: op B_add'] + (['B: submit'] if shape == 'planning' else [])
            labels += ['B: push', "host: merge B's MR", 'A: push', "host: merge A's MR"]
            m, hist = drive(cfg, labels)
            before, after = hist[-2].main, hist[-1].main
            added = sorted(M.anc(after) - M.anc(before))
            values = sum(1 for c in added for g in M.parsed(c))
            a1_born = M.C(hist[1].tx[0].loc)
            a1_on_main = [c for c in added if M.C(c).hid == 'a1']
            kept = (a1_on_main[0] == hist[1].tx[0].loc) if a1_on_main else None
            got = (len(added), first_parent_count(after, before), values, kept)
            exp = E4[(shape, host)]
            out['e4'].append(dict(shape=shape, host=host, model=got, spike=exp, match=got == exp))
            if shape == 'planning' and host == 'merge':
                # E1: the fold keeps the final tree; code commits before the first
                # planning commit keep their SHA, the ones after it get new ones
                tip_before = hist[len(A_steps)].tx[0].loc
                tip_after = hist[len(A_steps) + 1].tx[0].loc
                same_tree = M.C(tip_before).tree == M.C(tip_after).tree
                code_before = {M.C(c).hid: c for c in M.anc(tip_before) if M.C(c).kind == 'code'}
                code_after = {M.C(c).hid: c for c in M.anc(tip_after) if M.C(c).kind == 'code'}
                patch_same = {h: M.delta(code_before[h]) == M.delta(code_after[h]) for h in code_before}
                sha_same = {h: code_before[h] == code_after[h] for h in code_before}
                q = M.C(tip_after)
                out['e1'] = dict(same_tree=same_tree, patch_same=patch_same, sha_same=sha_same,
                                 record=[(g.op, g.actor) for g in M.parsed(tip_after)], kind=q.kind)
    with open(os.path.join(HERE, 'fidelity.json'), 'w') as f:
        json.dump(out, f, indent=1)
    for r in out['e4']:
        print(f"{r['shape']:9s} {r['host']:13s} model {r['model']} spike {r['spike']} {'match' if r['match'] else 'DIFFERS'}")
    print('E1', out['e1'])


if __name__ == '__main__':
    main()
