#!/usr/bin/env python3
"""Render results.json as markdown tables (written to tables.md)."""
import json
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
from model import DESIGNS, applicable  # noqa: E402
from run import CONDS, MODES  # noqa: E402

NAMES = {'S0': 'S0', 'S0b': 'S0b', 'A': 'A', 'A+': 'A+ (refined)', 'Beager': 'B eager', 'Blazy': 'B lazy',
         'Cns': 'C namespace', 'Crand': 'C random', 'D': 'D', 'Euniq': 'E unique handle', 'Eslug': 'E slug handle'}


def cell(runs):
    """Combine the runs of one (design, mode, condition) cell."""
    viol = set().union(*[set(r['viol']) for r in runs])
    ev = set().union(*[set(r['eventual']) for r in runs])
    stuck = any(r['stuck'] for r in runs)
    refused = set().union(*[set(r['refused']) for r in runs])
    complete = all(r['complete'] for r in runs)
    acyc = all(r['acyclic'] is not False for r in runs)
    gone = any('I3gone' in r['viol'] for r in runs)
    i1 = '✗' if 'I1ev' in ev else ('◐' if 'I1' in viol else '✓')
    i2 = '✗' if 'I2' in viol else ('◐' if 'I2ev' in ev else '✓')
    i3 = ('✗' if 'I3' in viol else '') + ('g' if gone else '') or '✓'
    i4 = '✗' if 'I4' in viol else ('h' if 'I4h' in viol else '✓')
    if stuck or not acyc:
        i5 = '✗'
    elif refused:
        i5 = 'R'
    else:
        i5 = '✓'
    mark = '' if complete else '*'
    return f"{i1} {i2} {i3} {i4} {i5}{mark}"


def main():
    res = json.load(open(os.path.join(HERE, 'results.json')))
    by = {}
    for r in res:
        if r['kind'] == 'matrix':
            by.setdefault((r['design'], r['mode'], r['cond']), []).append(r)
    out = []
    for mode in MODES:
        title = {'direct': 'direct-to-default-branch (non-force push to main, pre-push hook)',
                 'mr': 'protected default branch, merge requests, pipeline, NO up-to-date rule',
                 'mrutd': 'protected default branch, merge requests, pipeline, up-to-date rule'}[mode]
        out.append(f"\n#### Mode `{mode}`: {title}\n")
        out.append("| design | " + " | ".join(CONDS) + " |")
        out.append("|---|" + "---|" * len(CONDS))
        for d in DESIGNS:
            why = applicable(d, mode)
            if why:
                out.append(f"| {NAMES[d]} | " + " | ".join(['n/a'] * len(CONDS)) + " |")
                continue
            cells = []
            for c in CONDS:
                runs = by.get((d, mode, c))
                cells.append('n/a' if not runs else cell(runs))
            out.append(f"| {NAMES[d]} | " + " | ".join(cells) + " |")
    out.append("\n#### Mutation probes\n")
    out.append("| probe | what is broken | run | expected to fail | failed | trace |")
    out.append("|---|---|---|---|---|---|")
    probes = sorted((r for r in res if r['kind'] == 'probe'),
                    key=lambda r: (int(re.sub(r'\D', '', r['pid'])), r['pid']))
    for r in probes:
        failed = sorted(set(r['viol']) | {k for k in r['eventual']} | ({'I5'} if r['stuck'] else set()))
        exp = r['expect'] or '(control: none)'
        key = r['expect']
        tr = ''
        if key and key in r['viol']:
            tr = r['viol'][key]['trace']
        elif key == 'I5' and r['stuck']:
            tr = r['stuck']['trace']
        elif key and (key + 'ev') in r['eventual']:
            tr = r['eventual'][key + 'ev']['trace']
        out.append(f"| {r['pid']} | {r['what']} | {r['design']}@{r['mode']} | {exp} | "
                   f"{', '.join(failed) or 'none'} | {tr} |")
    out.append("\n#### Run statistics\n")
    out.append("| run | states | complete | BFS depth | secs | terminal states | done | renumbers max | terminals with a renumber |")
    out.append("|---|---|---|---|---|---|---|---|---|")
    for r in res:
        out.append(f"| {r['id']} | {r['states']} | {'yes' if r['complete'] else 'NO, to depth ' + str(r['depth_complete'])} | "
                   f"{r['max_depth']} | {r['secs']} | {r['terminals']} | {r['done']} | {r['renumbers']['max']} | "
                   f"{r['renumbers']['share_with_any']} |")
    with open(os.path.join(HERE, 'tables.md'), 'w') as f:
        f.write('\n'.join(out) + '\n')
    # RESULTS.md = template + the matrix and probe tables + the truncation list
    tpl = os.path.join(HERE, 'RESULTS.template.md')
    if os.path.exists(tpl):
        text = open(tpl).read()
        cut = out.index("\n#### Run statistics\n")
        trunc = [r for r in res if not r['complete']]
        text = text.replace('<!-- TABLES -->', '\n'.join(out[:cut]).strip())
        text = text.replace('<!-- TRUNC -->', str(len(trunc)))
        text = text.replace('<!-- TRUNCLIST -->', '; '.join(
            f"`{r['id']}` (depth {r['depth_complete']})" for r in trunc))
        text = text.replace('<!-- NRUNS -->', str(len(res)))
        text = text.replace('<!-- NCOMPLETE -->', str(len(res) - len(trunc)))
        text = text.replace('<!-- NSTATES -->', f"{sum(r['states'] for r in res) / 1e6:.0f}")
        with open(os.path.join(HERE, 'RESULTS.md'), 'w') as f:
            f.write(text)
    print('\n'.join(out[:80]))


if __name__ == '__main__':
    main()
