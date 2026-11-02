import sys; sys.path.insert(0, sys.argv[1])
from model import *
for d in ('S0',):
    r = explore(Cfg(d, 'mr', ents=(2, 1)), want_graph=False)
    m = r.model
    terms = [i for i in range(len(r.states)) if not m.successors(r.states[i])]
    dup_terms = [i for i in terms if dup_ids(tree(r.states[i].main))]
    by_budget = [renumber_count(r.states[i]) for i in dup_terms]
    print(d, 'terminal states', len(terms), 'with dup on main', len(dup_terms),
          'renumber counts of those', sorted(set(by_budget)))
