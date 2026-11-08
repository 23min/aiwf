#!/usr/bin/env python3
"""Explicit-state model of aiwf transactions: folding a branch's planning
commits into one planning commit at submit, then landing it through a host.

Git abstraction
---------------
Commits are content-addressed: a commit is (parents, tree, message, kind,
hid, author, salt) and its id is its SHA. Re-creating a commit with the same
parents and content yields the same SHA, as `git rebase` does when a commit's
parent is unchanged (G12); a commit re-created on a new parent gets a new SHA.
Commit dates are abstracted away; `salt` stands in for a new committer date
where a probe re-creates a commit on its old parent.

A tree maps *paths* to values. A path is a merge unit: one frontmatter line or
one body of one entity file (G3: a status line and a body paragraph of one file
merge cleanly; two different edits of one line conflict). A value is
(label, writer, prevs): `writer` is the hidden identity of the operation or
commit that wrote it and `prevs` the value(s) it replaced, so the set of
writers in a value's lineage says which operations "landed". Three-way merges
compare values for equality only, as git compares content.

A message is a tuple of paragraphs; git parses trailers only from the last
paragraph (G1). An aiwf operation's record is a trailer *group* that carries
the operation's hidden identity, so "this operation's record survived" and
"this record names the commit its author meant" are checkable.
"""
from collections import deque, Counter
from typing import NamedTuple
import time

# ----------------------------------------------------------------- the project
PLAN = ('ADR', 'G.body', 'G.st', 'M.ac', 'M.body', 'M.st', 'NA', 'NB')   # every path a verb writes
WORK = tuple(p for p in PLAN if p != 'ADR')                       # the ones under work/
CODE = ('cA', 'cB')
ENTITY = {'M.st': 'M', 'M.body': 'M', 'M.ac': 'M', 'G.st': 'G', 'G.body': 'G', 'NA': 'NA', 'NB': 'NB', 'ADR': 'ADR'}
STATUS = {'M.st': 'milestone', 'M.ac': 'ac', 'G.st': 'gap', 'NB': 'gap', 'NA': 'decision', 'ADR': 'adr'}
STEPENT = {'M.ac': 'M/AC-1'}          # the aiwf-entity a promote of this path names
FSM = {
    'milestone': {'draft': ('in_progress', 'cancelled'), 'in_progress': ('done', 'cancelled'),
                  'done': (), 'cancelled': ()},
    'gap': {'open': ('addressed', 'wontfix'), 'addressed': (), 'wontfix': ()},
    'decision': {'proposed': ('accepted', 'rejected'), 'accepted': ('superseded',), 'superseded': (), 'rejected': ()},
    'adr': {'proposed': ('accepted', 'rejected'), 'accepted': ('superseded',), 'superseded': (), 'rejected': ()},
    'ac': {'open': ('met', 'deferred', 'cancelled'), 'met': (), 'deferred': (), 'cancelled': ()},
}
TXS = ('A', 'B')
OPER = {'A': 'peter', 'B': 'carol'}
CODEPATH = {'A': 'cA', 'B': 'cB'}


def sname(label):
    return label if isinstance(label, str) else label[0]


def legal(kind, a, b):
    return b in FSM[kind].get(a, ())


class Op(NamedTuple):
    opid: str
    tx: str
    verb: str
    ent: str
    actor: str
    writes: tuple          # ((path, label), ...); empty for an empty commit (authorize, acknowledge)
    ptrs: tuple            # ((trailer key, hidden identity of the commit it names), ...)
    pre: object            # (path, status the verb requires) or None
    to: object             # aiwf-to of a promote
    treeptr: object        # hidden identity named by addressed_by_commit (written into G.st), or None


OPS = {o.opid: o for o in [
    Op('A_ip', 'A', 'promote', 'M', 'human/peter', (('M.st', 'in_progress'),), (), ('M.st', 'draft'), 'in_progress', None),
    Op('A_done', 'A', 'promote', 'M', 'human/peter', (('M.st', 'done'),), (), ('M.st', 'in_progress'), 'done', None),
    Op('A_gb', 'A', 'edit-body', 'G', 'human/bob', (('G.body', 'A_gb'),), (), None, None, None),
    Op('A_adr', 'A', 'add', 'ADR', 'human/peter', (('ADR', 'proposed'),), (), None, None, None),
    Op('A_auth', 'A', 'authorize', 'M', 'human/peter', (), (), None, None, None),
    Op('A_ipa', 'A', 'promote', 'M', 'ai/agent', (('M.st', 'in_progress'),), (('authorized-by', 'A_auth'),),
       ('M.st', 'draft'), 'in_progress', None),
    Op('A_ack', 'A', 'acknowledge', 'G', 'human/peter', (), (('force-for', 'h1'),), None, None, None),
    Op('A_gaddr', 'A', 'promote', 'G', 'human/peter', (('G.st', 'addressed'),), (), ('G.st', 'open'), 'addressed', 'a1'),
    Op('B_add', 'B', 'add', 'NB', 'human/carol', (('NB', 'open'),), (), None, None, None),
    Op('B_gb', 'B', 'edit-body', 'G', 'human/carol', (('G.body', 'B_gb'),), (), None, None, None),
    Op('X_mb', 'A', 'edit-body', 'M', 'human/bob', (('M.body', 'X_mb'),), (), None, None, None),
    # the spike's E4 fixture (fidelity only): add an AC, start M, bob edits G, mark the AC met
    Op('E_ac', 'A', 'add', 'M/AC-1', 'human/peter', (('M.ac', 'open'),), (), None, None, None),
    Op('E_met', 'A', 'promote', 'M/AC-1', 'human/peter', (('M.ac', 'met'),), (), ('M.ac', 'open'), 'met', None),
    # amend conditions only (AMEND_CONDS): A's edit-body G by A's own actor, so that it and the
    # promote after it are consecutive verbs by one actor
    Op('A_gbp', 'A', 'edit-body', 'G', 'human/peter', (('G.body', 'A_gbp'),), (), None, None, None),
    # `aiwf authorize M --end`: an empty commit whose aiwf-scope-ends names the authorize operation
    Op('A_end', 'A', 'authorize', 'M', 'human/peter', (), (('scope-ends', 'A_auth'),), None, None, None),
    # the human's promote of M to done, which ends the scope on M (aiwf-scope-ends)
    Op('A_donee', 'A', 'promote', 'M', 'human/peter', (('M.st', 'done'),), (('scope-ends', 'A_auth'),),
       ('M.st', 'in_progress'), 'done', None),
    # `promote G addressed --by-commit <the commit of promote M in_progress>`: addressed_by_commit
    # naming a verb's commit rather than a code commit
    Op('A_gaddrv', 'A', 'promote', 'G', 'human/peter', (('G.st', 'addressed'),), (), ('G.st', 'open'), 'addressed',
       'A_ip'),
]}
RECORD_ONLY = {o for o, v in OPS.items() if not v.writes}
OWNER = {o: v.tx for o, v in OPS.items()}
SEQ = {}            # filled per scenario: opid -> position in its transaction


# ----------------------------------------------------------------- the store
class Commit(NamedTuple):
    parents: tuple
    tree: int
    msg: tuple
    kind: str        # base code verb plan hand mixed merge squash revert
    hid: object      # hidden identity: op id, code commit name, or None
    author: str
    salt: object


class Grp(NamedTuple):
    op: str          # hidden operation identity
    verb: str
    ent: str
    actor: str
    to: object
    ptrs: tuple      # ((key, target, meant), ...): target is a SHA (int) or ('op', opid)
    tx: str          # transaction identity; readers consult it only under txid=record
    seq: int


class Store:
    def __init__(self):
        self.tidx, self.tlist, self.tdict = {}, [], []
        self.cidx, self.cm, self.anc = {}, [], []
        self.cache = {}


S = Store()


def reset():
    global S
    S = Store()


def tid(d):
    key = tuple(sorted((k, v) for k, v in d.items() if v is not None))
    i = S.tidx.get(key)
    if i is None:
        i = len(S.tlist)
        S.tidx[key] = i
        S.tlist.append(key)
        S.tdict.append(dict(key))
    return i


def td(t):
    return S.tdict[t]


def mk(parents, tree, msg, kind, hid, author, salt=None):
    c = Commit(tuple(parents), tree, msg, kind, hid, author, salt)
    i = S.cidx.get(c)
    if i is None:
        i = len(S.cm)
        S.cidx[c] = i
        S.cm.append(c)
        a = {i}
        for p in parents:
            a |= S.anc[p]
        S.anc.append(frozenset(a))
    return i


def C(i):
    return S.cm[i]


def anc(i):
    return S.anc[i]


def tree_of(i):
    return td(S.cm[i].tree)


def parsed(i):
    m = S.cm[i].msg
    return m[-1][1] if m and m[-1][0] == 't' else ()


def delta(i):
    k = ('delta', i)
    r = S.cache.get(k)
    if r is None:
        c = S.cm[i]
        a = td(c.tree)
        b = td(S.cm[c.parents[0]].tree) if c.parents else {}
        r = {p: (b.get(p), a.get(p)) for p in set(a) | set(b) if b.get(p) != a.get(p)}
        S.cache[k] = r
    return r


def lineage(v):
    if v is None:
        return frozenset()
    k = ('lin', v)
    r = S.cache.get(k)
    if r is None:
        s = {v[1]}
        for p in v[2]:
            s |= lineage(p)
        r = S.cache[k] = frozenset(s)
    return r


def tree_ops(i):
    """Operations whose writes are in the lineage of commit i's tree."""
    k = ('tops', i)
    r = S.cache.get(k)
    if r is None:
        s = set()
        for p, v in tree_of(i).items():
            if p in PLAN:
                s |= lineage(v)
        r = S.cache[k] = frozenset(x for x in s if x in OPS)
    return r


def ever_ops(i):
    """Operations present in the tree of i or of any of its ancestors."""
    k = ('ever', i)
    r = S.cache.get(k)
    if r is None:
        r = set(tree_ops(i))
        for p in S.cm[i].parents:
            r |= ever_ops(p)
        r = S.cache[k] = frozenset(r)
    return r


def recs(i):
    """opid -> [commits whose git-parsed trailers carry its group], over anc(i)."""
    k = ('recs', i)
    r = S.cache.get(k)
    if r is None:
        r = {}
        for c in anc(i):
            for g in parsed(c):
                r.setdefault(g.op, []).append(c)
        S.cache[k] = r
    return r


def merge3(b, o, t):
    out, bad = {}, []
    for p in set(b) | set(o) | set(t):
        bv, ov, tv = b.get(p), o.get(p), t.get(p)
        if ov == tv:
            v = ov
        elif ov == bv:
            v = tv
        elif tv == bv:
            v = ov
        else:
            bad.append(p)
            continue
        if v is not None:
            out[p] = v
    return out, sorted(bad)


def merge_base(a, b):
    common = anc(a) & anc(b)
    best = max(common, key=lambda c: (len(anc(c)), c))
    return best


def first_parent_chain(tip, base):
    """First-parent commits of tip that are not reachable from base, oldest first."""
    A, out, c = anc(base), [], tip
    while c not in A:
        out.append(c)
        ps = C(c).parents
        if not ps:
            break
        c = ps[0]
    out.reverse()
    return out


def classify(c):
    cc = C(c)
    if len(cc.parents) > 1:
        return 'merge'
    d = delta(c)
    pl = any(p in PLAN for p in d)
    co = any(p in CODE for p in d)
    if cc.kind == 'plan' or parsed(c):
        return 'mixed' if co else 'plan'
    if pl and co:
        return 'mixed'
    return 'hand' if pl else 'code'


def fmt_c(c):
    cc = C(c)
    if isinstance(cc.hid, tuple) and cc.hid[0] == 'amend':
        return f"{cc.kind}:{'+'.join(cc.hid[1:])}#{c}"
    h = cc.hid if not isinstance(cc.hid, tuple) else ''.join(map(str, cc.hid))
    return f"{cc.kind}:{h or ''}#{c}"


# ----------------------------------------------------------------- state
class Tx(NamedTuple):
    loc: int          # the operator's local branch tip (local main in direct mode)
    rtm: int          # last fetched default branch
    rtb: object       # last pushed or fetched remote branch: the lease expectation (None before the first push)
    pc: int           # next script step
    subs: int         # submits run
    unk: bool         # a push reply was lost and not yet reconciled
    side: object = None  # condition 'fork' only: the local branch the checked-out branch was cut from
    verbhead: bool = False  # amend rule 'reflog' only: an aiwf verb moved loc last (the top line of HEAD's reflog)


class St(NamedTuple):
    main: int
    rb: tuple         # remote transaction branch per transaction (None)
    mr: tuple         # open merge request per transaction: None | (pipeline ok, pipeline head)
    tx: tuple
    nmerge: tuple     # merges per transaction
    last: tuple       # per transaction: None | (main before, main after) of its latest merge
    rev: tuple        # per transaction: its latest merge was reverted
    flt: int          # faults used
    bob: int          # second operator has pushed
    ok: frozenset     # ghost: (code hid, SHA) pairs a code commit may legitimately have
    landed: frozenset  # ghost: operations that ever landed on main
    mrec: frozenset   # ghost: operations present in a merged head


class Cfg(NamedTuple):
    mode: str                  # 'mr' | 'direct'
    method: str = 'merge'      # merge | semi | ff | squash
    sqmsg: str = 'all'         # title | all | git   (squash only)
    utd: bool = False          # host requires the branch to be up to date
    cond: str = 'base'
    design: tuple = ()         # (('submit', 'fold'), ('fold', 'replay'), ...)
    probes: frozenset = frozenset()
    max_states: int = 3_000_000


DIMS = {
    'submit': ('none', 'fold'),
    'fold': ('replay', 'copy'),
    'resub': ('replace', 'append'),
    'txid': ('branch', 'record'),
    'ptr': ('leave', 'rewrite', 'opid', 'refuse'),
    'audit': ('parent', 'record'),
}
# submit=amend: a verb amends HEAD when HEAD is eligible, else commits per-verb; no submit step.
# Its eligibility test is the 'amend' dimension, read only when submit=amend:
#   naive  HEAD is a non-merge verb commit, by the same author and aiwf-actor as the new
#          operation, and no remote-tracking ref (origin/main, origin/<branch>) contains it
#   safe   naive, and no other local branch contains HEAD, and the new operation names no SHA of
#          HEAD in a trailer or in the bytes it writes
#   reflog safe, and an aiwf verb was the last to move HEAD: the top line of HEAD's reflog is the
#          verb's, so no rebase, pull, merge, cherry-pick, checkout or git commit has moved it since
AMEND_DIMS = {'submit': ('amend',), 'amend': ('naive', 'safe', 'reflog'), 'ptr': ('leave', 'opid'),
              'audit': ('parent', 'record')}

CONDS = ('base', 'after', 'bob', 'mixed', 'crash', 'lost', 'revert', 'cherry', 'resolve', 'nocheck',
         'reuse', 'auth', 'ack', 'abc', 'adr')
# Conditions that exercise amend. 'sa-<c>' is condition c with A's edit-body G run by A's own actor
# (A_gbp for A_gb), so that it and A's last promote are consecutive verbs by one actor.
SA_BASE = ('base', 'after', 'bob', 'mixed', 'crash', 'lost', 'revert', 'cherry', 'resolve', 'nocheck', 'reuse')
AMEND_CONDS = (tuple('sa-' + c for c in SA_BASE)
               + ('fork', 'reuse4', 'untracked', 'scope', 'actor', 'byc', 'twostep')
               + ('nr-fork', 'nr-scope', 'nr-byc'))
EARLY = {'after', 'bob'}          # A may submit and push before its script is complete
STAGES = {'reuse': (2,), 'reuse4': (4,), 'untracked': (4,)}   # A may also stop, submit, push and merge after this many steps


def scripts(cond):
    if cond.startswith('nr-'):
        return scripts(cond[3:])
    if cond.startswith('sa-'):
        s = scripts(cond[3:])
        s['A'] = [('op', 'A_gbp') if stp == ('op', 'A_gb') else stp for stp in s['A']]
        return s
    if cond in ('reuse4', 'untracked'):
        return {'A': [('code', 'a1'), ('op', 'A_ip'), ('code', 'a2'), ('op', 'A_gbp'), ('op', 'A_done')],
                'B': [('code', 'b1'), ('op', 'B_add')]}
    if cond in ('fork', 'scope', 'actor', 'byc', 'twostep'):
        A = {'fork': [('op', 'A_ip'), ('branch', 'm'), ('op', 'A_gbp'), ('code', 'a1'), ('merge', 'm')],
             'scope': [('code', 'a1'), ('op', 'A_auth'), ('op', 'A_end')],
             'actor': [('op', 'A_auth'), ('op', 'A_ipa'), ('op', 'A_donee')],
             'byc': [('code', 'a1'), ('op', 'A_ip'), ('op', 'A_gaddrv')],
             'twostep': [('code', 'a1'), ('op', 'A_ip'), ('op', 'A_done')]}[cond]
        return {'A': A, 'B': [('code', 'b1'), ('op', 'B_add')]}
    A = [('code', 'a1'), ('op', 'A_ip'), ('code', 'a2'), ('op', 'A_gb'), ('op', 'A_done')]
    B = [('code', 'b1'), ('op', 'B_add')]
    if cond == 'mixed':
        A[2] = ('mixed', 'a2')                      # git add -A: code plus a hand edit of M's body
    elif cond == 'resolve':
        B = [('code', 'b1'), ('op', 'B_gb')]        # B edits the same body as A: a conflict
    elif cond == 'auth':
        A = [('op', 'A_auth'), ('code', 'a1'), ('op', 'A_ipa')]
    elif cond == 'ack':
        A = [('op', 'A_ip'), ('hand', 'h1'), ('op', 'A_ack'), ('code', 'a1')]
    elif cond == 'abc':
        A = [('op', 'A_ip'), ('code', 'a1'), ('op', 'A_gaddr')]
    elif cond == 'adr':
        A = [('code', 'a1'), ('op', 'A_adr'), ('op', 'A_ip')]
    elif cond == 'e4':
        A = [('code', 'a1'), ('op', 'E_ac'), ('code', 'a2'), ('op', 'A_ip'), ('code', 'a3'), ('op', 'A_gb'),
             ('op', 'E_met')]
    return {'A': A, 'B': B}


# ----------------------------------------------------------------- model
class Model:
    def __init__(self, cfg):
        reset()
        self.cfg = cfg
        self.dv = dict(cfg.design)
        self.used = set()
        self.p = cfg.probes
        # an 'sa-' condition behaves as the condition it names; only A's script differs. In an
        # 'nr-' condition the operators never run git rebase: they take in main by merging it
        cond = cfg.cond
        if cond.startswith('nr-'):
            cond, self.p = cond[3:], self.p | {'norebase'}
        cond = cond[3:] if cond.startswith('sa-') else cond
        self.cond = cond
        self.direct = cfg.mode == 'direct'
        self.sc = scripts(cfg.cond)
        SEQ.clear()
        for X, s in self.sc.items():
            for k, st in enumerate(s):
                if st[0] == 'op':
                    SEQ[st[1]] = k
        SEQ['X_mb'] = 99
        self.early = cond in EARLY
        # A may merge twice where its work can outlive one merge: 'reuse' by design, 'after' because
        # the host may merge an early push while A's later verbs are still to come; 'reuse4' and
        # 'untracked' merge at the stage and again at the end
        self.maxmerge = {'A': 2 if cond in ('reuse', 'after', 'reuse4', 'untracked') else 1, 'B': 1}
        # before its first merge, A takes in main only where it must (as B does); in direct mode a
        # push needs main, so there A may always take it in
        self.holdoff = cond == 'reuse' or (cond in ('reuse4', 'untracked') and not self.direct)
        # liveness is conditional on legal transactions: the one whose branch carries the fault is not
        # (in 'resolve', whichever transaction resolved the conflict by hand; see is_legal)
        self.legal = {'A': cond != 'mixed', 'B': cond != 'cherry'}
        base_tree = tid({'M.st': ('draft', 'init', ()), 'M.body': ('m0', 'init', ()),
                         'G.st': ('open', 'init', ()), 'G.body': ('g0', 'init', ())})
        self.base = mk((), base_tree, (('s', 'initial'),), 'base', None, 'init')
        self.chk = {}
        self.refusals = {}
        # read here, so that every amend run reads the rule: a reflog run's states differ from a
        # naive or safe run's even where no verb reaches the eligibility test
        self.reflog = self.amend_design() and self.d('amend') == 'reflog'

    def d(self, key):
        self.used.add(key)
        return self.dv[key]

    def fold_design(self):
        return self.d('submit') == 'fold'

    def amend_design(self):
        return self.d('submit') == 'amend'

    def head_moved(self, tx, by_verb, **kw):
        """tx with HEAD moved as kw says; under the reflog rule it records whether an aiwf
        verb moved it, as the top line of HEAD's reflog would."""
        return tx._replace(verbhead=by_verb and self.reflog, **kw)

    def amendable(self, tx, op, author, ptrs, treeptr):
        """The amend eligibility test (AMEND_DIMS) for running op at tx.loc. ptrs are the
        new operation's pointers as it would write them; treeptr the commit its file bytes name."""
        h = tx.loc
        hc = C(h)
        if hc.kind != 'verb' or len(hc.parents) != 1 or hc.author != author:
            return False
        if 'amend-any-actor' not in self.p and any(g.actor != op.actor for g in parsed(h)):
            return False
        # remote-tracking refs: origin/main always; origin/<branch> unless the clone tracks only main
        tracked = [tx.rtm] + ([tx.rtb] if tx.rtb is not None and self.cond != 'untracked' else [])
        if any(h in anc(r) for r in tracked):
            return False
        rule = self.d('amend')
        if rule in ('safe', 'reflog'):
            if tx.side is not None and h in anc(tx.side):
                return False
            if any(t == h for (_, t, _) in ptrs) or treeptr == h:
                return False
        if rule == 'reflog' and not tx.verbhead:
            return False
        return True

    # ------------------------------------------------------------ setup
    def initial(self):
        b = self.base
        txs = tuple(Tx(b, b, None, 0, 0, False) for _ in TXS)
        n = len(TXS)
        return St(b, (None,) * n, (None,) * n, txs, (0,) * n, (None,) * n, (False,) * n, 0, 0,
                  frozenset(), frozenset(), frozenset())

    @staticmethod
    def settx(st, i, tx):
        t = list(st.tx)
        t[i] = tx
        return st._replace(tx=tuple(t))

    @staticmethod
    def setv(tup, i, v):
        t = list(tup)
        t[i] = v
        return tuple(t)

    def refuse(self, X, why):
        self.refusals.setdefault(X, set()).add(why)

    # ------------------------------------------------------------ aiwf check
    @staticmethod
    def where(F, main):
        return ', '.join(sorted({f"{f[0]} on {'main' if f[1] in anc(main) else 'the branch'}" for f in F}))

    def acks(self, head):
        k = ('acks', head)
        r = S.cache.get(k)
        if r is None:
            r = set()
            for c in anc(head):
                for g in parsed(c):
                    for (key, target, meant) in g.ptrs:
                        if key == 'force-for' and isinstance(target, int):
                            r.add(target)
            r = S.cache[k] = frozenset(r)
        return r

    def fsm_bad(self, c):
        """Illegal status steps in non-merge commit c, read the way the chosen audit reads them."""
        k = ('fsm', c)
        if k in S.cache:
            return S.cache[k]
        cc = C(c)
        bad = []
        if len(cc.parents) == 1:
            for p, (b, a) in sorted(delta(c).items()):
                if p not in STATUS or b is None or a is None:
                    continue       # a new file, or a deleted one: no transition to compare
                sb, sa = sname(b[0]), sname(a[0])
                if sb == sa:
                    continue
                kind = STATUS[p]
                par_ok = legal(kind, sb, sa)
                steps = [g.to for g in parsed(c) if g.verb == 'promote' and g.ent == STEPENT.get(p, ENTITY[p])
                         and g.to]
                if steps:
                    cur, rec_ok = sb, True
                    for s in steps:
                        if not legal(kind, cur, s):
                            rec_ok = False
                        cur = s
                    rec_ok = rec_ok and cur == sa
                else:
                    rec_ok = par_ok
                ok = par_ok if par_ok == rec_ok else (rec_ok if self.d('audit') == 'record' else par_ok)
                if not ok:
                    bad.append((p, sb, sa))
        S.cache[k] = tuple(bad)
        return S.cache[k]

    @staticmethod
    def untrailered(c):
        cc = C(c)
        return len(cc.parents) == 1 and not parsed(c) and any(p in PLAN for p in delta(c))

    def auth_missing(self, g, head):
        out = []
        for (key, target, meant) in g.ptrs:
            if key != 'authorized-by':
                continue
            if isinstance(target, int):
                ok = target in anc(head) and any(x.verb == 'authorize' for x in parsed(target))
            else:
                ok = any(any(x.op == target[1] and x.verb == 'authorize' for x in parsed(c2))
                         for c2 in recs(head).get(target[1], ()))
            if not ok:
                out.append(target)
        return out

    def check(self, head, prov_base):
        """Blocking findings of `aiwf check` at head. Provenance rules cover the
        commits after prov_base, and are skipped when prov_base is None (a
        first push with no upstream, G-0539/G-0679); the FSM walk covers all
        of head's history."""
        k = (head, prov_base)
        r = self.chk.get(k)
        if r is not None:
            return r
        F = []
        ak = self.acks(head)
        if prov_base is not None:
            R = recs(head)
            for c in sorted(anc(head) - anc(prov_base)):
                if self.untrailered(c) and c not in ak:
                    F.append(('provenance-untrailered-entity-commit', c))
                for g in parsed(c):
                    if self.auth_missing(g, head):
                        F.append(('provenance-authorization-missing', c))
                # with the transaction identity in the record, an operation recorded
                # in two planning commits is detectable
                if (C(c).kind == 'plan' and 'nodupcheck' not in self.p
                        and any(len([x for x in R.get(g.op, ()) if C(x).kind == 'plan']) > 1 for g in parsed(c))
                        and self.d('txid') == 'record'):
                    F.append(('duplicate-operation-record', c))
        for c in sorted(anc(head) - {self.base}):
            if self.fsm_bad(c) and c not in ak:
                F.append(('fsm-history-consistent', c))
        r = self.chk[k] = tuple(F)
        return r

    def pipeline(self, main, head):
        """The merge request's pipeline: `aiwf check` on the merge result as of now.
        Returns the blocking finding codes (empty: passed)."""
        if main in anc(head):
            h = head
        else:
            t, bad = merge3(tree_of(merge_base(main, head)), tree_of(main), tree_of(head))
            if bad:
                return ('conflict',)
            h = mk((main, head), tid(t), (('m', 'pipeline'),), 'merge', None, 'host')
        return tuple(sorted({self.where([f], main) for f in self.check(h, main)}))

    # ------------------------------------------------------------ verbs and commits
    def find_hid(self, tip, hid):
        for c in sorted(anc(tip), reverse=True):
            h = C(c).hid
            if h == hid or (isinstance(h, tuple) and h[0] == 'amend' and hid in h[1:]):
                return c
        return None

    def run_step(self, X, tx, stp):
        """Returns (new loc, refusal) for one script step."""
        kind, name = stp
        loc = tx.loc
        T = dict(tree_of(loc))
        me = OPER[X]
        if kind == 'code':
            p = CODEPATH[X]
            T[p] = (name, name, (T.get(p),))
            return mk((loc,), tid(T), (('s', f'feat: {name}'),), 'code', name, me), None
        if kind == 'hand':
            T['G.body'] = (name, name, (T.get('G.body'),))
            return mk((loc,), tid(T), (('s', 'docs: hand edit of the gap body'),), 'hand', name, me), None
        if kind == 'mixed':
            p = CODEPATH[X]
            T[p] = (name, name, (T.get(p),))
            T['M.body'] = (name + 'w', name, (T.get('M.body'),))
            return mk((loc,), tid(T), (('s', f'feat: {name} (git add -A)'),), 'mixed', name, me), None
        op = OPS[name]
        if op.pre is not None:
            pv = T.get(op.pre[0])
            if pv is None or sname(pv[0]) != op.pre[1]:
                return None, 'precondition'
        ptrs = []
        for key, target in op.ptrs:
            if target in OPS and self.d('ptr') == 'opid':
                ptrs.append((key, ('op', target), target))
            else:
                c = self.find_hid(loc, target)
                ptrs.append((key, c, target))
        tp = None
        for p, lab in op.writes:
            if op.treeptr:
                tp = self.find_hid(loc, op.treeptr)
                lab = (lab, ('abc', tp, op.treeptr))
            T[p] = (lab, name, (T.get(p),))
        g = Grp(name, op.verb, op.ent, op.actor, op.to, tuple(ptrs), op.tx, SEQ[name])
        author = 'bob' if name == 'X_mb' else me
        if self.amend_design() and self.amendable(tx, op, author, ptrs, tp):
            # HEAD re-created on its parent with the new operation's change, and the new group
            # appended to HEAD's: one trailer paragraph, one group per operation, in operation
            # order, as the fold's planning commit records them (A12)
            h = C(loc)
            groups = parsed(loc) + (g,)
            return mk(h.parents, tid(T), (h.msg[0], ('t', groups)), 'verb',
                      ('amend',) + tuple(x.op for x in groups), author), None
        msg = (('s', f'aiwf {op.verb} {op.ent}'), ('t', (g,)))
        return mk((loc,), tid(T), msg, 'verb', name, author), None

    def rebase(self, tip, onto, salt=None, exclude=frozenset()):
        """git rebase: replay the non-merge commits of tip that onto lacks, in
        topological order; a commit that becomes empty is dropped (G13), one
        that started empty is kept. Returns (new tip, {old: new}) or None on a
        conflict."""
        todo = sorted((c for c in anc(tip) - anc(onto) - exclude if len(C(c).parents) == 1),
                      key=lambda c: (len(anc(c)), c))
        # git skips a commit whose patch is already upstream (patch-id), as G13 printed
        upstream = {frozenset(delta(u).items()) for u in anc(onto) - anc(tip) if len(C(u).parents) == 1
                    and delta(u)}
        new, m = onto, {}
        for c in todo:
            cc = C(c)
            if delta(c) and frozenset(delta(c).items()) in upstream:
                continue
            t, bad = merge3(tree_of(cc.parents[0]), tree_of(new), td(cc.tree))
            if bad:
                return None
            if delta(c) and tid(t) == C(new).tree:
                continue
            nc = mk((new,), tid(t), cc.msg, cc.kind, cc.hid, cc.author, salt)
            m[c] = nc
            new = nc
        return new, m

    def okadd(self, ok, m):
        add = {(C(o).hid, n) for o, n in m.items() if C(o).kind in ('code', 'hand', 'mixed')}
        return ok | add if add else ok

    # ------------------------------------------------------------ the fold
    def fold(self, i, st, nonatomic=False):
        """Submit. Returns None (nothing to fold), ('refused', why), or
        ('ok', new loc, new ok-ghost, events)."""
        X = TXS[i]
        me = OPER[X]
        tx = st.tx[i]._replace(rtm=st.main)        # submit fetches the default branch first
        chain = first_parent_chain(tx.loc, tx.rtm)
        if not chain:
            return None
        cls = [classify(c) for c in chain]
        planning = [k for k, c in enumerate(chain) if cls[k] == 'plan']
        if not planning:
            return None
        qpos = [k for k in planning if C(chain[k]).kind == 'plan']
        if qpos and self.d('resub') == 'append':
            fold_idx = [k for k in planning if k > max(qpos)]
        else:
            fold_idx = planning
        if not fold_idx:
            return None
        if 'fold-unpushed' in self.p and tx.rtb is not None:
            # fold only the operations not yet on the remote branch; the pushed ones
            # stay as they are, so they must all come before the first one folded
            pushed = recs(tx.rtb)
            held = [k for k in fold_idx if any(g.op in pushed for g in parsed(chain[k]))]
            fold_idx = [k for k in fold_idx if k not in held]
            if not fold_idx:
                return None
            if held and max(held) > fold_idx[0]:
                return ('refused', 'pushed')
        if len(fold_idx) == 1 and C(chain[fold_idx[0]]).kind == 'plan' and fold_idx[0] == len(chain) - 1:
            return None                                  # the planning commit is already at the tip
        if 'mixed' in cls:
            return ('refused', 'mixed')
        fset = set(fold_idx)
        first = fold_idx[0]
        if 'strict-unpushed' in self.p and tx.rtb is not None and chain[first] in anc(tx.rtb):
            return ('refused', 'pushed')
        if 'strict-unpushed-ops' in self.p and tx.rtb is not None and any(
                g.op in recs(tx.rtb) for k in fold_idx for g in parsed(chain[k])):
            return ('refused', 'pushed')
        # refuse when the commits it would fold or rewrite include anyone else's
        if any(C(c).author != me for c in chain[first:] if classify(c) != 'merge'):
            return ('refused', 'foreign')
        groups = []
        for k in fold_idx:
            groups += list(parsed(chain[k]))
        net = {}
        for k in fold_idx:
            for p, (b, a) in delta(chain[k]).items():
                if p not in PLAN or ('workonly' in self.p and p not in WORK):
                    continue
                if p in net:
                    net[p][1] = a
                else:
                    net[p] = [b, a]
        has_ptr = any(g.ptrs for g in groups) or any(
            a is not None and not isinstance(a[0], str) for (b, a) in net.values())
        mode = self.d('ptr') if has_ptr else 'leave'
        if mode == 'refuse' and ('refuse-trailers-only' not in self.p or any(g.ptrs for g in groups)):
            return ('refused', 'ptr')
        # rebuild every other commit on top of the commits it kept
        new, rmap, dropped, ok = chain[0], {}, False, st.ok
        new = C(chain[0]).parents[0]
        for k, c in enumerate(chain):
            if k in fset:
                dropped = True
                continue
            cc = C(c)
            if cc.parents[0] == new and 'rebuildall' not in self.p:
                new = c
                continue
            if len(cc.parents) > 1:
                # a merge is re-made as git rebase --rebase-merges does: a fresh merge
                # of the rebuilt tip with the same second parent
                t, bad = merge3(tree_of(merge_base(new, cc.parents[1])), tree_of(new), tree_of(cc.parents[1]))
            else:
                t, bad = merge3(tree_of(cc.parents[0]), tree_of(new), td(cc.tree))
            if bad:
                return ('refused', 'conflict')
            salt = ('rebuild', X, tx.subs) if 'rebuildall' in self.p else None
            nc = mk((new,) + cc.parents[1:], tid(t), cc.msg, cc.kind, cc.hid, cc.author, salt)
            rmap[c] = nc
            if dropped and cc.kind in ('code', 'hand', 'mixed'):
                ok = ok | {(cc.hid, nc)}          # it follows a planning commit: a permitted rewrite
            new = nc

        def rw_val(v):
            if mode != 'rewrite' or v is None or isinstance(v[0], str):
                return v
            lab, (tag, sha, meant) = v[0]
            return ((lab, (tag, rmap.get(sha, sha), meant)), v[1], v[2])

        def rw_grp(g):
            if mode != 'rewrite' or not g.ptrs:
                return g
            return g._replace(ptrs=tuple((k, rmap.get(t, t) if isinstance(t, int) else t, m)
                                         for (k, t, m) in g.ptrs))
        base_T = tree_of(new)
        # replay: the net change, three-way onto the branch as it stands
        rT, rbad = dict(base_T), []
        for p, (b, a) in sorted(net.items()):
            a = rw_val(a)
            cur = rT.get(p)
            if cur == b or 'overwrite' in self.p:
                if a is None:
                    rT.pop(p, None)
                else:
                    rT[p] = a
            elif cur != a:
                rbad.append(p)
        # copy: the planning tree of the last folded commit
        snap = tree_of(chain[fold_idx[-1]])
        cT = dict(base_T)
        for p in (WORK if 'workonly' in self.p else PLAN):
            v = rw_val(snap.get(p))
            if v is None:
                cT.pop(p, None)
            else:
                cT[p] = v
        if (rbad, rT) == ([], cT):
            T, bad = rT, []
        elif self.d('fold') == 'replay':
            T, bad = rT, rbad
        else:
            T, bad = cT, []
        if bad:
            return ('refused', 'conflict')
        groups = [rw_grp(g) for g in groups]
        known = recs(st.main)
        if any(g.op in known for g in groups) and self.d('txid') == 'record':
            # The default branch already records some of these operations under this
            # transaction's identity, so the branch holds copies of merged commits.
            # Once the branch contains main their changes are main's, and their
            # records are dropped; before that, submit asks for the update first.
            if st.main not in anc(tx.loc):
                return ('refused', 'merged-before')
            groups = [g for g in groups if g.op not in known]
        if 'droprec' in self.p:
            groups = groups[-1:]
        if 'unordered' in self.p:
            groups = sorted(groups, key=lambda g: (g.ent, str(g.to)))
        msg = (('s', 'chore(aiwf): planning commit'), ('t', tuple(groups)))
        if 'footer' in self.p:
            msg = msg + (('s', 'footer'),)
        if nonatomic:
            q = new                                 # crashed after the rebuild, before the planning commit
        else:
            q = mk((new,), tid(T), msg, 'plan', (X, 'Q'), me)
        evs = []
        if not nonatomic:
            expect = {p: rw_val(a) for p, (b, a) in net.items() if base_T.get(p) != rw_val(a)}
            got = {p: a for p, (b, a) in delta(q).items() if p in PLAN}
            if got != expect:
                diff = sorted(set(got) ^ set(expect) | {p for p in got if p in expect and got[p] != expect[p]})
                evs.append(('T2', f"{X}'s planning commit {fmt_c(q)} changes {diff} beyond the folded "
                                  f"commits' net change", False))
        evs += self.drop_events(X, tx.loc, q, 'submit', st.main)
        return ('ok', q, ok, evs)

    def drop_events(self, X, old, new, how, main=None):
        """T1: one of X's own operations was on the branch and is gone, and not
        on main either. T2: another transaction's operation is gone although the
        new tip descends from a commit that had it, so merging the branch will
        delete it from main."""
        evs = []
        on_main = tree_ops(main) if main is not None else frozenset()
        for o in sorted(tree_ops(old) - tree_ops(new)):
            if OWNER[o] == X:
                if o not in on_main:
                    evs.append(('T1', f"{how} on {X}'s branch drops {X}'s operation {o}", False))
            elif o in ever_ops(new):
                evs.append(('T2', f"{how} on {X}'s branch reverts {OWNER[o]}'s operation {o}", False))
        return evs

    # ------------------------------------------------------------ successors
    def finished(self, i, st):
        """Every operation the transaction ever had has landed on main at some
        point (a later removal by someone else is T2's failure, not this one's),
        and its record-only operations are recorded there."""
        X = TXS[i]
        tx = st.tx[i]
        if tx.pc < len(self.sc[X]):
            return False
        if st.rev[i]:
            return True
        own = {o for o in ever_ops(tx.loc) if OWNER[o] == X}
        if st.nmerge[i] == 0 and not own:
            return False
        if not own <= tree_ops(st.main) | st.landed:
            return False
        mine = {o for o in recs(tx.loc) if OWNER.get(o) == X and o in RECORD_ONLY}
        return mine <= st.mrec | set(recs(st.main))

    def ready(self, i, st):
        """The operator asks the host to merge: the script is complete and, under
        the fold design, the pushed branch is submitted (no verb commit after its
        last planning commit). Condition 'after' drops the second requirement:
        verbs run after submit reach the host as they are."""
        X = TXS[i]
        if st.nmerge[i] >= self.maxmerge[X]:
            return False
        pc = st.tx[i].pc
        if pc < len(self.sc[X]) and not (X == 'A' and pc in STAGES.get(self.cond, ()) and st.nmerge[i] == 0):
            return False
        if (self.fold_design() and self.cond != 'after' and self.unfolded_at(st.rb[i], st.main)
                and not self.fold_refused_ptr(i, st) and not self.lands_unfolded(i, st)):
            return False
        return True

    def lands_unfolded(self, i, st):
        """What-ifs for W2's cost. With 'w2-merge-unsubmitted', a transaction whose
        submit W2 refuses is merged as its per-verb commits. With 'fold-unpushed',
        pushed verbs stay per-verb and the operator merges once nothing is left to
        fold. Both first push the local tip."""
        tx = st.tx[i]
        if st.rb[i] != tx.loc:
            return False
        if 'w2-merge-unsubmitted' in self.p:
            r = self.fold(i, st)
            return r is not None and r[0] == 'refused' and r[1] == 'pushed'
        if 'fold-unpushed' in self.p:
            return self.fold(i, st) is None
        return False

    def fold_refused_ptr(self, i, st):
        """Under ptr=refuse a transaction whose fold is refused lands per-verb."""
        r = self.fold(i, st)
        return r is not None and r[0] == 'refused' and r[1] == 'ptr'

    @staticmethod
    def unfolded_at(tip, base):
        ch = first_parent_chain(tip, base)
        qs = [k for k, c in enumerate(ch) if C(c).kind == 'plan']
        last = max(qs) if qs else -1
        return any(classify(c) == 'plan' and C(c).kind != 'plan' for c in ch[last + 1:])

    def unfolded(self, tx):
        return self.unfolded_at(tx.loc, tx.rtm)

    def successors(self, st):
        self.refusals = {}
        out = []
        for i, X in enumerate(TXS):
            if not self.finished(i, st):
                self.operator(i, st, out)
        if not self.direct:
            self.host(st, out)
        self.environment(st, out)
        return out

    def operator(self, i, st, out):
        X = TXS[i]
        tx = st.tx[i]
        scr = self.sc[X]
        allpc = tx.pc >= len(scr) or (X == 'A' and tx.pc in STAGES.get(self.cond, ()))
        early = self.early and X == 'A'
        # 1. the next script step
        if tx.pc < len(scr) and scr[tx.pc][0] in ('branch', 'merge'):
            self.fork_step(i, st, out)
        elif tx.pc < len(scr):
            nl, why = self.run_step(X, tx, scr[tx.pc])
            if why:
                self.refuse(X, why)
            else:
                ok = st.ok | {(scr[tx.pc][1], nl)} if scr[tx.pc][0] != 'op' else st.ok
                lab, evs = f"{X}: {scr[tx.pc][0]} {scr[tx.pc][1]}", []
                if scr[tx.pc][0] == 'op' and C(nl).parents != (tx.loc,):
                    lab += ' [amends HEAD]'
                    evs = self.drop_events(X, tx.loc, nl, 'amend', st.main)
                ns = self.settx(st, i, self.head_moved(tx, scr[tx.pc][0] == 'op', loc=nl, pc=tx.pc + 1))
                out.append((lab, ns._replace(ok=ok), evs))
        if self.direct and tx.side is not None:
            self.push_side(i, st, out)
        # 2. submit
        if self.fold_design() and tx.subs >= 2 and (allpc or early) and self.unfolded(tx._replace(rtm=st.main)):
            self.refuse(X, 'bound: submits')
        if self.fold_design() and tx.subs < 2 and (allpc or early):
            r = self.fold(i, st)
            if r is not None:
                if r[0] == 'refused':
                    self.refuse(X, 'submit refused: ' + r[1])
                else:
                    _, q, ok, evs = r
                    ns = self.settx(st, i, tx._replace(loc=q, subs=tx.subs + 1, rtm=st.main))._replace(ok=ok)
                    out.append((f"{X}: submit (fold into {fmt_c(q)})", ns, evs))
                    if self.cond == 'crash' and st.flt == 0:
                        if 'nonatomic' in self.p:
                            _, q2, ok2, evs2 = self.fold(i, st, nonatomic=True)
                            ns2 = self.settx(st, i, tx._replace(loc=q2, subs=tx.subs + 1, rtm=st.main))._replace(
                                ok=ok2, flt=1)
                            out.append((f"{X}: submit [FAULT: crash after rewriting the branch, before the "
                                        f"planning commit]", ns2, evs2))
                        else:
                            out.append((f"{X}: submit [FAULT: crash; the branch ref is updated once, so it "
                                        f"still names the old tip]", st._replace(flt=1), []))
        # 3. push
        if allpc or early:
            (self.push_direct if self.direct else self.push_mr)(i, st, out)
        # 4. fetch, to learn where the remote branch is (after a lost reply, or another operator's push)
        if not self.direct and st.rb[i] is not None and tx.rtb != st.rb[i]:
            out.append((f"{X}: fetch", self.settx(st, i, tx._replace(rtm=st.main, rtb=st.rb[i], unk=False)), []))
        # 5. integrate others' pushes to the transaction's own branch
        rb = st.rb[i]
        if (not self.direct and rb is not None and tx.rtb == rb and rb not in anc(tx.loc)
                and self.others_new_work(X, rb, tx.loc, st.main)):
            if tx.loc in anc(rb):
                nl, m = rb, {}
            else:
                r = self.rebase(tx.loc, rb, exclude=anc(st.main))
                if r is None:
                    self.refuse(X, 'conflict')
                    nl = None
                else:
                    nl, m = r
            if nl is not None:
                ns = self.settx(st, i, self.head_moved(tx, False, loc=nl))._replace(ok=self.okadd(st.ok, m))
                out.append((f"{X}: pull --rebase its own remote branch", ns, []))
        # 6. update from the default branch: A at any time (an environment move);
        #    B only where it must (direct mode, the up-to-date rule, or a conflict)
        if not (self.holdoff and X == 'A' and st.nmerge[i] == 0
                and not (st.rb[i] is not None and self.cfg.utd)):
            if st.main not in anc(tx.loc) and 'noupdate' not in self.p:
                if X == 'A' or self.direct or (st.rb[i] is not None and (self.cfg.utd or self.conflicted(i, st))):
                    self.update(i, st, out)
            elif X == 'A' and 'norebase' not in self.p and any(len(C(c).parents) > 1
                                                                for c in anc(tx.loc) - anc(st.main)):
                # git rebase main still linearises a branch that merged main, dropping
                # commits whose change is already upstream (G13)
                r = self.rebase(tx.loc, st.main)
                if r is not None and r[0] != tx.loc:
                    nl, m = r
                    ns = self.settx(st, i, self.head_moved(tx, False, loc=nl, rtm=st.main))._replace(
                        ok=self.okadd(st.ok, m))
                    out.append((f"{X}: rebase onto main", ns, []))

    def fork_step(self, i, st, out):
        """Condition 'fork': cut a branch from the checked-out tip (the tip stays on the
        branch it was cut from, side), or merge that branch back into side with --no-ff,
        as the wrap rituals merge, and check side out again."""
        X = TXS[i]
        tx = st.tx[i]
        kind, name = self.sc[X][tx.pc]
        if kind == 'branch':
            out.append((f"{X}: cut branch {name} from its unpushed tip and check it out",
                        self.settx(st, i, self.head_moved(tx, False, side=tx.loc, pc=tx.pc + 1)), []))
            return
        t, bad = merge3(tree_of(merge_base(tx.side, tx.loc)), tree_of(tx.side), tree_of(tx.loc))
        if bad:
            self.refuse(X, 'conflict')
            return
        m = mk((tx.side, tx.loc), tid(t), (('s', f"Merge branch '{name}'"),), 'merge', None, OPER[X])
        out.append((f"{X}: merge branch {name} into the branch it was cut from (--no-ff)",
                    self.settx(st, i, self.head_moved(tx, False, loc=m, side=None, pc=tx.pc + 1)), []))

    def push_side(self, i, st, out):
        """Direct mode, condition 'fork': while the cut branch is checked out, the operator may
        push local main (side), with the pre-push hook."""
        X = TXS[i]
        tx = st.tx[i]
        side = tx.side
        if side in anc(st.main) or st.main not in anc(side):
            return
        F = self.check(side, tx.rtm)
        if F:
            self.refuse(X, 'pre-push hook: ' + self.where(F, st.main))
            return
        ns, evs = self.land(st, i, side, f"{X}'s push of main")
        out.append((f"{X}: push main (the branch it cut from)", self.settx(ns, i, tx._replace(rtm=side)), evs))

    def conflicted(self, i, st):
        rb = st.rb[i]
        return bool(merge3(tree_of(merge_base(rb, st.main)), tree_of(st.main), tree_of(rb))[1])

    @staticmethod
    def others_new_work(X, rb, loc, main):
        """rb carries a commit by someone else whose change is not already in the
        operator's branch or on main: neither the same patch is there (git's
        patch-id) nor are its results already in that tree (a rebase would find
        it empty and drop it, G13)."""
        have = {frozenset(delta(c).items()) for c in anc(loc) | anc(main) if len(C(c).parents) == 1}
        trees = (tree_of(loc), tree_of(main))

        def applied(c):
            return any(all(T.get(p) == a for p, (b, a) in delta(c).items()) for T in trees)
        return any(C(c).author != OPER[X] and len(C(c).parents) == 1 and delta(c)
                   and frozenset(delta(c).items()) not in have and not applied(c)
                   for c in anc(rb) - anc(loc) - anc(main))

    def update(self, i, st, out):
        X = TXS[i]
        tx = st.tx[i]
        me = OPER[X]
        t, bad = merge3(tree_of(merge_base(tx.loc, st.main)), tree_of(tx.loc), tree_of(st.main))
        msg = (('s', f"Merge branch 'main' into {X}"),)
        if not bad:
            m = mk((tx.loc, st.main), tid(t), msg, 'merge', None, me)
            out.append((f"{X}: merge main into the branch",
                        self.settx(st, i, self.head_moved(tx, False, loc=m, rtm=st.main)), []))
        elif self.cond == 'resolve' and st.flt == 0:
            for how in (('both',) if 'resolve-both' in self.p else ('ours', 'theirs', 'both', 'both+M.st=done')):
                T = dict(t)
                for p in bad:
                    o, th = tree_of(tx.loc).get(p), tree_of(st.main).get(p)
                    T[p] = {'ours': o, 'theirs': th}.get(how, ('res', 'res', (o, th)))
                if how.endswith('done'):
                    T['M.st'] = ('done', 'res', (T.get('M.st'),))
                m = mk((tx.loc, st.main), tid(T), msg, 'merge', None, me)
                evs = self.drop_events(X, tx.loc, m, 'conflict resolution', st.main)
                evs = [(k + 'm' if k == 'T2' else k, d, det) for (k, d, det) in evs]
                out.append((f"{X}: merge main, conflict on {bad} [FAULT: resolved by hand: {how}]",
                            self.settx(st, i, self.head_moved(tx, False, loc=m, rtm=st.main))._replace(flt=2 + i),
                            evs))
        else:
            self.refuse(X, 'conflict')
        if 'norebase' in self.p:
            return
        r = self.rebase(tx.loc, st.main)
        if r is None:
            self.refuse(X, 'conflict')
        else:
            nl, m = r
            ns = self.settx(st, i, self.head_moved(tx, False, loc=nl, rtm=st.main))._replace(ok=self.okadd(st.ok, m))
            out.append((f"{X}: rebase onto main", ns, []))

    def push_mr(self, i, st, out):
        X = TXS[i]
        tx = st.tx[i]
        me = OPER[X]
        rb = st.rb[i]
        if tx.loc == rb or (rb is not None and tx.loc in anc(rb)):
            return
        force = rb is not None and rb not in anc(tx.loc)
        if force and 'nolease' not in self.p:
            if rb != tx.rtb:
                self.refuse(X, 'push refused: stale lease')
                return
            if self.others_new_work(X, tx.rtb, tx.loc, st.main):
                self.refuse(X, 'push refused: the remote branch carries others\' commits')
                return
        F = self.check(tx.loc, tx.rtb)
        if F:
            self.refuse(X, 'pre-push hook: ' + self.where(F, st.main))
            return
        outs = [('ok', True, True)]
        if force and self.cond == 'lost' and st.flt == 0:
            outs += [('lost-applied', True, False), ('lost-unapplied', False, False)]
        for tag, applied, reply in outs:
            ns, evs = st, []
            if applied:
                okp = self.pipeline(st.main, tx.loc)
                ns = ns._replace(rb=self.setv(st.rb, i, tx.loc), mr=self.setv(st.mr, i, (okp, tx.loc)))
                if force:
                    evs = self.drop_events(X, rb, tx.loc, 'force push', st.main)
            ntx = tx._replace(rtb=tx.loc if reply else tx.rtb, unk=not reply)
            ns = self.settx(ns, i, ntx)
            lab = f"{X}: {'force-with-lease ' if force else ''}push{' (first; MR opened)' if rb is None else ''}"
            if tag != 'ok':
                ns = ns._replace(flt=1)
                lab += f" [FAULT: reply lost, push {'applied' if applied else 'not applied'}]"
            out.append((lab, ns, evs))

    def push_direct(self, i, st, out):
        X = TXS[i]
        tx = st.tx[i]
        if tx.loc == st.main or tx.loc in anc(st.main):
            return
        if st.main not in anc(tx.loc):
            self.refuse(X, 'push rejected: non-fast-forward (pull first)')
            return
        if self.fold_design() and self.unfolded(tx) and not self.fold_refused_ptr(i, st):
            return                                   # fold-before-push: submit first
        if self.cond != 'nocheck':
            F = self.check(tx.loc, tx.rtm)
            if F:
                self.refuse(X, 'pre-push hook: ' + self.where(F, st.main))
                return
        outs = [('ok', True, True)]
        if self.cond == 'lost' and st.flt == 0:
            outs += [('lost-applied', True, False), ('lost-unapplied', False, False)]
        for tag, applied, reply in outs:
            ns, evs = st, []
            if applied:
                ns, evs = self.land(st, i, tx.loc, f"{X}'s push")
            # 'untracked': the push names the remote by URL, so origin/main moves only on a fetch
            ntx = tx._replace(rtm=tx.loc if reply and self.cond != 'untracked' else tx.rtm, unk=not reply)
            ns = self.settx(ns, i, ntx)
            lab = f"{X}: push main"
            if tag != 'ok':
                ns = ns._replace(flt=1)
                lab += f" [FAULT: reply lost, push {'applied' if applied else 'not applied'}]"
            out.append((lab, ns, evs))
        if self.cond == 'lost' and tx.unk:
            pass

    # ------------------------------------------------------------ host
    def host(self, st, out):
        cfg = self.cfg
        for i, X in enumerate(TXS):
            mr, rb = st.mr[i], st.rb[i]
            if mr is None or rb is None:
                continue
            if st.nmerge[i] >= self.maxmerge[X]:
                self.refuse(X, 'bound: merges')
                continue
            if not self.ready(i, st):
                continue
            if self.cond != 'nocheck' and (mr[0] or mr[1] != rb):
                self.refuse(X, 'pipeline: ' + (', '.join(mr[0]) if mr[1] == rb else 'not run on the head'))
                continue
            if cfg.utd and st.main not in anc(rb):
                self.refuse(X, 'host: not up to date')
                continue
            base = merge_base(st.main, rb)
            t, bad = merge3(tree_of(base), tree_of(st.main), tree_of(rb))
            if bad:
                self.refuse(X, 'conflict')
                continue
            ok, nrb = st.ok, rb
            if cfg.method in ('semi', 'ff') and st.main not in anc(rb):
                r = self.rebase(rb, st.main)
                if r is None:
                    self.refuse(X, 'conflict')
                    continue
                nrb, m = r
                ok = self.okadd(ok, m)
            if cfg.method == 'merge' or cfg.method == 'semi':
                head = mk((st.main, nrb), C(nrb).tree if cfg.method == 'semi' else tid(t),
                          (('m', X),), 'merge', None, 'host')
            elif cfg.method == 'ff':
                head = nrb
            else:
                head = mk((st.main,), tid(t), self.squash_msg(st.main, rb), 'squash', None, 'host')
            ns = st._replace(ok=ok, rb=self.setv(st.rb, i, nrb), mr=self.setv(st.mr, i, None))
            ns, evs = self.land(ns, i, head, f"host merge of {X}", merged_head=rb)
            lab = f"host: merge {X}'s MR by {cfg.method}" + (f" ({cfg.sqmsg})" if cfg.method == 'squash' else '')
            if nrb != rb:
                lab += ' after rebasing it'
            out.append((lab, ns, evs))

    def squash_msg(self, main, rb):
        how = self.cfg.sqmsg
        title = ('s', 'Resolve JIRA-1 (#1)')
        if how == 'title':
            return (title,)
        commits = sorted(anc(rb) - anc(main), key=lambda c: (len(anc(c)), c))
        if how == 'git':
            return (title, ('s', 'Squashed commit of the following:')) + tuple(
                ('i', parsed(c)) for c in commits)
        msg = [title]
        for c in commits:
            msg += list(C(c).msg)
        return tuple(msg)

    def land(self, st, i, head, how, merged_head=None, merging=True):
        """Main moves to head; evaluate every invariant that reads the default branch."""
        old = st.main
        evs = []
        new_commits = sorted(anc(head) - anc(old))
        post = self.check(head, old)
        flagged = {f[1] for f in post}
        ak = self.acks(head)
        for c in new_commits:
            cc = C(c)
            if len(cc.parents) > 1:
                p0, p1 = tree_of(cc.parents[0]), tree_of(cc.parents[1])
                own = [p for p in PLAN if tree_of(c).get(p) not in (p0.get(p), p1.get(p))]
                if own:
                    evs.append(('T3x', f"merge commit {fmt_c(c)} carries its own change to {own}, "
                                       f"which no trailer attests (merge commits are not audited)", False))
                for p in own:
                    if p in STATUS and p0.get(p) and tree_of(c).get(p):
                        a, b = sname(p0[p][0]), sname(tree_of(c)[p][0])
                        if a != b and not legal(STATUS[p], a, b):
                            evs.append(('T7x', f"merge commit {fmt_c(c)} moves {p} {a} -> {b}; the audit "
                                               f"skips merge commits", False))
                continue
            if c in ak:
                continue
            ents = {ENTITY[p] for p in delta(c) if p in PLAN}
            named = {g.ent.split('/')[0] for g in parsed(c)}
            miss = sorted(ents - named)
            if miss:
                evs.append(('T3', f"{fmt_c(c)} on main changes {miss} with no trailer naming them", c in flagged))
            for (p, a, b) in self.fsm_bad(c):
                evs.append(('T7', f"{fmt_c(c)} on main reads as {p} {a} -> {b}", c in flagged))
        ns = st._replace(main=head)
        if merging and (merged_head is not None or self.direct):
            mh = merged_head if merged_head is not None else head
            X = TXS[i]
            mine = {o for o in tree_ops(mh) | set(recs(mh)) if OWNER.get(o) == X}
            for o in sorted(o for o in mine if o not in RECORD_ONLY):
                if o not in tree_ops(head):
                    evs.append(('T1', f"{how} leaves out {X}'s operation {o}", False))
            ns = ns._replace(nmerge=self.setv(st.nmerge, i, min(st.nmerge[i] + 1, 2)),
                             last=self.setv(st.last, i, (old, head)), mrec=st.mrec | mine)
            # T5: code commits keep their SHAs unless a permitted rewrite produced the new one
            for c in new_commits:
                cc = C(c)
                if cc.kind == 'code' and cc.hid and (cc.hid, c) not in ns.ok:
                    evs.append(('T5', f"code commit {cc.hid} reaches main as {fmt_c(c)}, a SHA no permitted "
                                      f"rewrite produced", False))
        evs += self.main_events(ns, post)
        landed_now = tree_ops(head)
        ns = ns._replace(landed=ns.landed | landed_now)
        return ns, evs

    def main_events(self, st, post):
        M = st.main
        evs = []
        now = tree_ops(M)
        R = recs(M)
        merged_tx = {TXS[k] for k in range(len(TXS)) if st.nmerge[k] and not st.rev[k]}
        reverted = {TXS[k] for k in range(len(TXS)) if st.rev[k]}
        # T1: an unmerged transaction's operations are not on main
        for o in sorted(now):
            if st.nmerge[TXS.index(OWNER[o])] == 0:
                evs.append(('T1', f"{o} is on main although {OWNER[o]} has not merged", False))
        # T2: nothing that landed is reverted except by reverting its own transaction
        for o in sorted(st.landed - now):
            if OWNER[o] not in reverted:
                evs.append(('T2', f"{OWNER[o]}'s operation {o} landed on main and is no longer there", False))
        # T4: every merged operation has exactly one record on main, attributable to its transaction
        for o in sorted(st.mrec):
            if OWNER[o] in reverted:
                continue
            n = len(R.get(o, ()))
            if n == 0:
                evs.append(('T4', f"{o} ({OWNER[o]}) landed with no git-parsed record on main", False))
            elif n > 1:
                evs.append(('T4d', f"{o} has {n} records on main", False))
        for X in sorted(merged_tx):
            ops = [o for o in sorted(st.mrec) if OWNER[o] == X and R.get(o)]
            homes = {R[o][0] for o in ops}
            if len(homes) > 1 and not self.attributable(X, ops, M):
                evs.append(('T4a', f"{X}'s records sit in {len(homes)} commits on main and nothing on main "
                                   f"ties them to one transaction", False))
        # T6: every record that names a commit names a reachable one, the one it meant
        auth_flag = any(f[0] == 'provenance-authorization-missing' for f in post)
        for c in anc(M):
            for g in parsed(c):
                for (key, target, meant) in g.ptrs:
                    if not self.resolves(M, target, meant):
                        evs.append(('T6', f"{key} in {g.op}'s record names {self.fmt_t(target)}, not "
                                          f"reachable as {meant} from main", key == 'authorized-by' and auth_flag))
        v = tree_of(M).get('G.st')
        if v is not None and not isinstance(v[0], str):
            _, sha, meant = v[0][1]
            if not self.resolves(M, sha, meant):
                evs.append(('T6', f"addressed_by_commit names {self.fmt_t(sha)}, not reachable as {meant} "
                                  f"from main", False))
        return evs

    def fmt_t(self, t):
        return f"op {t[1]}" if isinstance(t, tuple) else (fmt_c(t) if t is not None else 'nothing')

    def resolves(self, M, target, meant):
        if target is None:
            return False
        if isinstance(target, tuple):
            return bool(recs(M).get(target[1]))
        if target not in anc(M):
            return False
        return C(target).hid == meant or any(g.op == meant for g in parsed(target))

    def attributable(self, X, ops, M):
        R = recs(M)
        for o in ops:
            c = R[o][0]
            if C(c).kind == 'plan' and self.d('txid') == 'record':
                continue
            if any(C(m).msg and C(m).msg[0] == ('m', X) and c in anc(C(m).parents[1]) - anc(C(m).parents[0])
                   for m in anc(M) if len(C(m).parents) > 1):
                continue
            return False
        return True

    # ------------------------------------------------------------ environment
    def environment(self, st, out):
        cond = self.cond
        if st.flt == 0 and cond == 'revert':
            for i, X in enumerate(TXS):
                if st.last[i] is None or st.rev[i]:
                    continue
                pre, post = st.last[i]
                t, bad = merge3(tree_of(post), tree_of(st.main), tree_of(pre))
                if bad:
                    continue
                c = mk((st.main,), tid(t), (('s', f'Revert "{X}"'), ('s', f'This reverts commit {post}.')),
                       'revert', None, 'host')
                ns = st._replace(rev=self.setv(st.rev, i, True), flt=1)
                ns, evs = self.land(ns, i, c, f"revert of {X}", merging=False)
                out.append((f"host: revert {X}'s merge [FAULT]", ns, evs))
        if self.direct:
            return
        if cond == 'bob' and st.bob == 0 and st.rb[0] is not None and st.mr[0] is not None:
            rb = st.rb[0]
            nl, _ = self.run_step('A', Tx(rb, st.main, rb, 0, 0, False), ('op', 'X_mb'))
            okp = self.pipeline(st.main, nl)
            ns = st._replace(rb=self.setv(st.rb, 0, nl), mr=self.setv(st.mr, 0, (okp, nl)), bob=1)
            out.append(("bob: fetch A's branch, edit-body M, push", ns, []))
        if cond == 'cherry' and st.flt == 0 and st.rb[0] is not None:
            tb = st.tx[1]
            if tb.pc >= 1:
                for c in first_parent_chain(st.rb[0], st.main):
                    if classify(c) != 'plan':
                        continue
                    cc = C(c)
                    t, bad = merge3(tree_of(cc.parents[0]), tree_of(tb.loc), td(cc.tree))
                    if bad:
                        continue
                    nc = mk((tb.loc,), tid(t), cc.msg, cc.kind, cc.hid, cc.author, ('cherry', 'carol'))
                    ns = self.settx(st, 1, self.head_moved(tb, False, loc=nc))._replace(flt=1)
                    out.append((f"B: cherry-pick {fmt_c(c)} from A's branch [FAULT]", ns, []))

    # ------------------------------------------------------------ terminal states
    def is_legal(self, X, st):
        return self.legal[X] and st.flt != 2 + TXS.index(X)

    def terminal(self, st):
        self.successors(st)
        reasons = []
        for i, X in enumerate(TXS):
            if self.finished(i, st):
                continue
            reasons.append((X, sorted(set(self.refusals.get(X, ()))), self.is_legal(X, st)))
        return reasons


HUMAN_REMEDY = ('conflict', 'submit refused: pushed', 'submit refused: mixed', 'submit refused: conflict', 'submit refused: foreign',
                "push refused: the remote branch carries others' commits", 'submit refused: merged-before',
                'host: not up to date', 'push refused: stale lease')


def classify_refusal(why, legal):
    """Classify the reasons a transaction is left unmerged in a terminal state.

    'bound' when any reason is one of the model's bounds (more submits or merges
    would let it continue). Otherwise each reason, and each finding inside a
    hook or pipeline reason, must be actionable: 'M' when every finding sits on
    a commit already on main (someone else's landing; the remedy is
    `aiwf acknowledge illegal` there), 'R' for a git conflict, a mixed commit,
    another author's commit, an out-of-date branch, a duplicate record (rebase
    onto main), or, for a transaction that is not legal, a finding on its own
    branch or a verb's precondition. None: stuck, a liveness failure."""
    if not why:
        return None
    if any(w.startswith('bound') for w in why):
        return 'bound'
    kinds = set()
    for w in why:
        if w in HUMAN_REMEDY:
            kinds.add('R')
            continue
        if w == 'precondition' and not legal:
            kinds.add('R')
            continue
        if ':' not in w:
            return None
        for part in (x.strip() for x in w.split(':', 1)[1].split(',')):
            if part.endswith('on main'):
                kinds.add('M')
            elif part.startswith('duplicate-operation-record') or part == 'conflict':
                kinds.add('R')
            elif not legal and part.endswith('on the branch'):
                kinds.add('R')
            else:
                return None
    return 'M' if 'M' in kinds else 'R'


# ----------------------------------------------------------------- search
class Result:
    pass


def explore(cfg, time_limit=None, want_graph=True):
    m = Model(cfg)
    t0 = time.time()
    init = m.initial()
    idx = {init: 0}
    states, parent, depth = [init], [(-1, None)], [0]
    edges = [] if want_graph else None
    q = deque([0])
    viol, silent, terms = {}, {}, []
    truncated = False
    nedges = 0
    while q:
        i = q.popleft()
        succ = m.successors(states[i])
        if want_graph:
            edges.append([])
        if not succ:
            terms.append(i)
            continue
        for act, ns, evs in succ:
            nedges += 1
            j = idx.get(ns)
            if j is None:
                j = len(states)
                idx[ns] = j
                states.append(ns)
                parent.append((i, act))
                depth.append(depth[i] + 1)
                q.append(j)
            if want_graph:
                edges[i].append(j)
            for (inv, detail, det) in evs:
                if inv not in viol:
                    viol[inv] = (i, act, detail, det)
                if not det and inv not in silent:
                    silent[inv] = (i, act, detail, det)
        if len(states) > cfg.max_states or (time_limit and time.time() - t0 > time_limit):
            truncated = True
            break
    r = Result()
    r.cfg, r.model, r.states, r.parent = cfg, m, states, parent
    r.viol, r.silent = viol, silent
    r.nstates, r.nedges, r.truncated = len(states), nedges, truncated
    r.depth_complete = depth[q[0]] if (truncated and q) else max(depth)
    r.max_depth = max(depth)
    r.done, r.refused, r.stuck, r.nbound = 0, {}, None, 0
    for i in terms:
        reasons = m.terminal(states[i])
        if not reasons:
            r.done += 1
            continue
        kinds = [(X, why, 'R' if not legal else classify_refusal(why, legal)) for (X, why, legal) in reasons]
        bad = [(X, why) for (X, why, k) in kinds if k is None]
        if bad:
            if r.stuck is None:
                r.stuck = (i, bad)
        else:
            ks = {k for (_, _, k) in kinds}
            key = 'M' if 'M' in ks else ('R' if 'R' in ks else 'bound')
            r.refused.setdefault(key, (i, reasons))
            r.nbound = getattr(r, 'nbound', 0) + (key == 'bound')
    r.nterm = len(terms)
    r.acyclic = None
    if want_graph and not truncated:
        indeg = [0] * len(states)
        for es in edges:
            for j in es:
                indeg[j] += 1
        stack = [k for k in range(len(states)) if indeg[k] == 0]
        seen = 0
        while stack:
            k = stack.pop()
            seen += 1
            for j in edges[k]:
                indeg[j] -= 1
                if indeg[j] == 0:
                    stack.append(j)
        r.acyclic = seen == len(states)
    r.used = set(m.used)
    r.secs = time.time() - t0
    return r


# ----------------------------------------------------------------- traces
def path(r, i):
    steps = []
    while i > 0:
        pi, a = r.parent[i]
        steps.append((a, i))
        i = pi
    steps.reverse()
    return steps


def show_tree(T):
    parts = []
    for p in PLAN:
        v = T.get(p)
        if v is None:
            continue
        lab = v[0]
        if not isinstance(lab, str):
            lab = f"{lab[0]}(abc->#{lab[1][1]})"
        parts.append(f"{p}={lab}<{v[1]}>")
    return ' '.join(parts)


def show_log(tip, stop):
    out = []
    for c in sorted(anc(tip) - anc(stop), key=lambda c: (len(anc(c)), c)):
        cc = C(c)
        g = parsed(c)
        rec = ('[' + ','.join(x.op for x in g) + ']') if g else '[]'
        par = ','.join(f"#{p}" for p in cc.parents)
        out.append(f"#{c} {cc.kind}:{cc.hid if not isinstance(cc.hid, tuple) else ''} by {cc.author} "
                   f"parents {par} trailers {rec}")
    return out


def summarize(r, st):
    m = r.model
    L = [f"       main #{st.main}: {show_tree(tree_of(st.main))}"]
    for k in show_log(st.main, m.base):
        L.append(f"         {k}")
    for i, X in enumerate(TXS):
        tx = st.tx[i]
        rb = st.rb[i]
        extra = f" | remote branch #{rb}" if rb is not None else ''
        mr = st.mr[i]
        if mr is not None:
            extra += f" | MR open, pipeline {'failed: ' + ','.join(mr[0]) if mr[0] else 'passed'} on #{mr[1]}"
        L.append(f"       {X} local #{tx.loc} (step {tx.pc}, submits {tx.subs}){extra}; merged {st.nmerge[i]}x"
                 f"{' (reverted)' if st.rev[i] else ''}")
        for k in show_log(tx.loc, st.main):
            L.append(f"         {k}")
        if tx.side is not None:
            L.append(f"       {X} branch it was cut from #{tx.side}")
    return '\n'.join(L)


def render_trace(r, i, act, detail, title):
    out = [title, f"violation: {detail}", "", "0. initial state: main holds M (draft) and G (open)"]
    steps = path(r, i)
    for k, (a, j) in enumerate(steps, 1):
        out.append(f"{k}. {a}")
        out.append(summarize(r, r.states[j]))
    if act is not None:
        out.append(f"{len(steps) + 1}. {act}   <-- the violating step")
    return '\n'.join(out)
