#!/usr/bin/env python3
"""Explicit-state model of aiwf id-allocation designs under concurrent work.

Git abstraction
---------------
A branch's content is a frozenset of *ops*. A commit adds ops; a merge is a
set union; "A is an ancestor of B" is A <= B. Every integration is a merge (no
rebase), so set inclusion is fast-forward-ability and a non-force push is the
compare-and-swap  remote <= pushed.

  ('N', ent, label, origin)        entity file created with an id or handle
  ('R', ent, old, new, why)        entity relabelled (renumber, claim, mint ...)
  ('F', rid, meant, label, origin) reference written; `meant` is the hidden
                                   identity its author meant
  ('W', rid, old, new)             reference rewritten

A tree is derived from an op set by following each entity's relabel chain and
each reference's rewrite chain. Two relabels of one entity (or two rewrites of
one reference) from the same label to different labels, or two entities on the
same provisional path, are a git merge conflict.

Every entity has a hidden identity (`ent`, e.g. 'a1' = clone A's first
entity), independent of its id, and every reference records the identity its
author meant, so a misdirected reference is a checkable violation.
"""
from collections import deque
from functools import lru_cache
from typing import NamedTuple
import time

CL = 'ABC'


# --------------------------------------------------------------------- labels
def is_handle(l):
    return isinstance(l, str) and l[:1] == '~'


def is_id(l):
    return not is_handle(l)


def fmt(l):
    return f"G-{l}" if isinstance(l, int) else str(l)


# ---------------------------------------------------------------------- trees
class Tree:
    __slots__ = ('label', 'prior', 'origin', 'refs', 'conflict', 'by_label')


@lru_cache(maxsize=None)
def tree(ops):
    first, origin, rel, rfirst, rw = {}, {}, {}, {}, {}
    for op in ops:
        k = op[0]
        if k == 'N':
            first[op[1]] = op[2]
            origin[op[1]] = op[3]
        elif k == 'R':
            rel.setdefault(op[1], []).append((op[2], op[3]))
        elif k == 'F':
            rfirst[op[1]] = (op[2], op[3], op[4])
        else:
            rw.setdefault(op[1], []).append((op[2], op[3]))
    conflict = None
    label, prior = {}, {}
    for e, l in first.items():
        chain, cur, steps = [l], l, rel.get(e, ())
        for _ in range(8):
            nxt = {n for (o, n) in steps if o == cur}
            if not nxt:
                break
            if len(nxt) > 1:
                conflict = ('relabel', e)
                break
            cur = next(iter(nxt))
            chain.append(cur)
        label[e] = cur
        prior[e] = tuple(chain[:-1])
    refs = {}
    for r, (meant, l, o) in rfirst.items():
        cur, steps = l, rw.get(r, ())
        for _ in range(8):
            nxt = {n for (a, n) in steps if a == cur}
            if not nxt:
                break
            if len(nxt) > 1:
                conflict = ('rewrite', r)
                break
            cur = next(iter(nxt))
        refs[r] = (meant, cur, o)
    by = {}
    for e, l in label.items():
        by.setdefault(l, []).append(e)
    for l, es in by.items():
        if is_handle(l) and len(es) > 1:
            conflict = conflict or ('path', l)   # same inbox/<slug>.md path
    t = Tree()
    t.label, t.prior, t.origin, t.refs, t.conflict = label, prior, origin, refs, conflict
    t.by_label = {l: tuple(sorted(es)) for l, es in by.items()}
    return t


def resolve(T, l):
    """aiwf refs-resolve: current id only (prior_ids are not consulted)."""
    return T.by_label.get(l, ())


def dup_ids(T):
    return {l: es for l, es in T.by_label.items() if is_id(l) and len(es) > 1}


def max_num(*trees_):
    m = 0
    for T in trees_:
        for l in T.label.values():
            if isinstance(l, int) and l > m:
                m = l
    return m


# ---------------------------------------------------------------------- state
class Clone(NamedTuple):
    wt: frozenset      # the branch the clone works on (feature branch, or local main in direct mode)
    rt: tuple          # remote-tracking snapshot: (main, branches-tuple, ledger)
    n: int             # entities created
    c: int             # citations made
    online: bool
    active: bool       # False once the clone abandoned its branch
    mrnum: object      # D: MR number the clone knows for its open MR
    busy: object       # B-eager: entity whose claim must complete first
    unc: object        # B: (ent, label) claim whose push reply was lost


class St(NamedTuple):
    main: frozenset    # remote default branch
    br: tuple          # remote feature branch per clone (None = absent/deleted)
    mr: tuple          # open MR per clone: None | (num, pipeline_ok, pipeline_head)
    mrcount: int
    ledger: tuple      # B: ((label, ent), ...) on the ledger ref
    cl: tuple
    mint: tuple        # E: per minter, the main snapshot it fetched (None = none)
    pub: frozenset     # ghost: (label, ent) bindings ever published on the remote
    xc: int            # cross-branch citations used
    flt: int           # faults used
    ab: int            # abandons used


class Cfg(NamedTuple):
    design: str
    mode: str              # 'direct' | 'mr' | 'mrutd'
    ents: tuple = (2, 1)   # entity budget per clone (len = number of clones)
    cites: tuple = (1, 1)  # citation budget per clone
    xcite: bool = False
    offline: bool = False
    lost: bool = False
    abandon: bool = False
    pipeline: bool = True
    hook: bool = True
    minters: int = 1
    max_renums: int = 3    # renumbers allowed per trace; hitting it is reported as a livelock
    probes: frozenset = frozenset()
    max_states: int = 2_000_000


DESIGNS = ('S0', 'S0b', 'A', 'A+', 'Beager', 'Blazy', 'Cns', 'Crand', 'D', 'Euniq', 'Eslug')


def applicable(design, mode):
    if design in ('A', 'A+') and mode == 'mr':
        return "A is defined by the up-to-date rule; without it A is S0b"
    if design == 'D' and mode == 'direct':
        return 'no merge requests in direct mode'
    return None


HUMAN_REMEDY = {'unres', 'conflict', 'dup'}   # 'dup' counts only where no repair verb exists (C)


# ---------------------------------------------------------------------- model
class Model:
    def __init__(self, cfg):
        self.cfg = cfg
        d, p = cfg.design, cfg.probes
        self.n = len(cfg.ents)
        self.mrmode = cfg.mode in ('mr', 'mrutd')
        self.utd = cfg.mode == 'mrutd'
        if d in ('S0', 'S0b', 'A', 'A+'):
            self.fam = 'S0'
        elif d in ('Euniq', 'Eslug'):
            # ADR-0001: on trunk, `aiwf add` mints immediately "same as today",
            # and reallocate remains the fix-up tool.
            self.fam = 'E' if self.mrmode else 'S0'
        else:
            self.fam = d[0]
        unscoped = d in ('S0', 'Euniq', 'Eslug') or 'unscoped' in p
        self.scope = 'all' if unscoped else 'own'
        self.widened = d == 'A+'
        self.fixref = d in ('D', 'Euniq', 'Eslug') and 'nofixref' not in p
        self._fc = {}

    # ------------------------------------------------------------ helpers
    def initial(self):
        n = self.n
        rt = (frozenset(), (None,) * n, ())
        cls = tuple(Clone(frozenset(), rt, 0, 0,
                          not (self.cfg.offline and i == n - 1), True, None, None, None)
                    for i in range(n))
        return St(frozenset(), (None,) * n, (None,) * n, 0, (), cls,
                  (None,) * self.cfg.minters, frozenset(), 0, 0, 0)

    @staticmethod
    def view_trees(rt):
        return [tree(rt[0])] + [tree(b) for b in rt[1] if b is not None]

    def live(self, st, l, x):
        for ops in (st.main,) + tuple(b for b in st.br if b is not None):
            if tree(ops).label.get(x) == l:
                return True
        return (l, x) in st.ledger



    def publish(self, st, opsets, pairs, evs):
        """Add the bindings visible in the written refs to the ghost `pub`
        and judge I3 (an id published for two identities) and I4 (an entity
        published under a new id after its old id was published)."""
        new = set(pairs)
        for ops in opsets:
            if ops is None:
                continue
            for e, l in tree(ops).label.items():
                new.add((l, e))
        pub = st.pub
        added = new - pub
        for (l, e) in sorted(added, key=str):
            others = sorted({x for (ll, x) in pub if ll == l and x != e})
            if others and is_id(l):
                live = any(self.live(st, l, x) for x in others)
                evs.append(('I3' if live else 'I3gone', f"{fmt(l)} published for {e} after it had been published for {others[0]} "
                                  f"({'that binding still on a ref' if live else 'that binding on no ref any more'})"))
            olds = sorted({ll for (ll, x) in pub if x == e and ll != l}, key=str)
            for ol in olds:
                if is_handle(l):
                    continue
                evs.append(('I4h' if is_handle(ol) else 'I4',
                            f"{e} published as {fmt(l)} after it had been published as {fmt(ol)}"))
        return pub | added

    def rewrite_ops(self, T, target, old, new):
        ops = []
        tor = T.origin[target]
        for rid, (meant, l, o) in sorted(T.refs.items()):
            if l != old:
                continue
            if self.scope == 'all' or o == tor:
                ops.append(('W', rid, old, new))
        return ops

    def idle(self, c, cl, st):
        return (cl.n >= self.cfg.ents[c] and cl.c >= self.cfg.cites[c] and cl.busy is None
                and cl.unc is None and cl.wt <= cl.rt[0]
                and not (self.cfg.xcite and self.mrmode and st.xc == 0))   # a citation still to write

    # ------------------------------------------------------------ checks
    def findings(self, T_ops, trunk_ops, rt_br, mrnum, where, own=None):
        key = (T_ops, trunk_ops, rt_br, mrnum, where, own)
        r = self._fc.get(key)
        if r is None:
            r = self._fc[key] = self._findings(T_ops, trunk_ops, rt_br, mrnum, where, own)
        return r

    def _findings(self, T_ops, trunk_ops, rt_br, mrnum, where, own):
        """Blocking aiwf-check findings on tree T against the trunk tree, with
        the remote branches as the cross-branch view for reference resolution."""
        T = tree(T_ops)
        views = [tree(trunk_ops)] + [tree(b) for b in rt_br if b is not None]
        F = []
        if T.conflict:
            F.append(('conflict', T.conflict))
        for l, es in dup_ids(T).items():
            F.append(('dup', l, es))
        M = tree(trunk_ops)
        for e, l in T.label.items():
            if not is_id(l):
                continue
            for x in M.by_label.get(l, ()):
                if x != e and x not in T.label:
                    F.append(('tcoll', l, e, x))
        for rid, (meant, l, o) in sorted(T.refs.items()):
            if resolve(T, l):
                continue
            if any(resolve(V, l) for V in views):
                continue           # cross-branch-pending: a warning only
            F.append(('unres', rid, l))
        if self.fam == 'B':
            for e, l in T.label.items():
                if is_handle(l):
                    F.append(('unclaimed', e))
        if self.fam == 'D' and (where == 'pipeline' or mrnum is not None):
            for e, l in T.label.items():
                if is_handle(l):
                    F.append(('handle', e))
        if self.widened and where == 'prepush' and own is not None:
            ci, pushed = own
            P = tree(pushed) if pushed is not None else None
            for e, l in T.label.items():
                # only this clone's own entities, not on trunk, not yet published under this id
                if (not is_id(l) or T.origin[e] != ci or e in M.label
                        or (P is not None and P.label.get(e) == l)):
                    continue
                for V in views:
                    for x in V.by_label.get(l, ()):
                        if x != e and x not in T.label:
                            F.append(('xcoll', l, e, x))
        return F

    def clone_findings(self, c, cl, st=None):
        own = (c, st.br[c]) if (self.widened and st is not None) else None
        return self.findings(cl.wt, cl.rt[0], cl.rt[1], cl.mrnum, 'prepush', own)

    def pipeline_ok(self, br_ops, main_ops, branches):
        if not self.cfg.pipeline:
            return True
        # CI checks the merge result and (assumption) fetches every branch
        return not self.findings(br_ops | main_ops, main_ops, branches, None, 'pipeline')

    # ------------------------------------------------------------ allocation
    def alloc(self, c, cl, e):
        d, p = self.cfg.design, self.cfg.probes
        if self.fam == 'S0':
            return [max_num(tree(cl.wt), *self.view_trees(cl.rt)) + 1]
        if self.fam == 'B':
            return ['~' + e]
        if self.fam == 'C':
            if d == 'Cns':
                ns = 'A' if 'samens' in p else CL[c]
                return [f"{ns}.{cl.n + 1}"]
            if 'smalltok' in p:
                return ['r0', 'r1']          # a draw from a 2-token space
            return ['r' + e]                 # a draw from a large space: unique
        if self.fam == 'D':
            if cl.mrnum is not None:
                k = 1 + sum(1 for l in tree(cl.wt).label.values()
                            if isinstance(l, str) and l.startswith(f"!{cl.mrnum}."))
                return [f"!{cl.mrnum}.{k}"]
            return ['~' + e]
        if self.fam == 'E':
            if d == 'Eslug' and f"~t{cl.n + 1}" not in tree(cl.wt).by_label:
                return [f"~t{cl.n + 1}"]       # same title-derived slug on every clone
            return ['~' + e]
        raise ValueError(d)

    # ------------------------------------------------------------ E minting
    @staticmethod
    def mint_ops(snap, only=None):
        T = tree(snap)
        pend = sorted(e for e, l in T.label.items() if is_handle(l) and (only is None or T.origin[e] == only))
        base = max_num(T)
        ops, newl = [], {}
        for i, e in enumerate(pend):
            newl[e] = base + 1 + i
            ops.append(('R', e, T.label[e], newl[e], 'mint'))
        for rid, (meant, l, o) in sorted(T.refs.items()):
            if not is_handle(l):
                continue
            cur = T.by_label.get(l, ())
            if len(cur) == 1 and cur[0] in newl:
                ops.append(('W', rid, l, newl[cur[0]]))
            elif not cur:
                # a late reference: follow the aiwf-mint record of the handle
                hits = [e for e, pr in T.prior.items() if l in pr]
                if len(hits) == 1 and is_id(T.label[hits[0]]):
                    ops.append(('W', rid, l, T.label[hits[0]]))
        return ops, newl

    # ------------------------------------------------------------ successors
    def successors(self, st):
        out = []
        main_dup = bool(dup_ids(tree(st.main)))
        # one designated clone (the lowest-index idle one) repairs a duplicate on main
        self.repairer = next((c for c in range(self.n) if st.cl[c].active and self.idle(c, st.cl[c], st)), None)
        for c in range(self.n):
            cl = st.cl[c]
            if cl.active:
                br = list(st.br); br[c] = None      # a clone's view of the others' branches
                R = (st.main, tuple(br), st.ledger)
                self.clone_actions(st, c, cl, R, main_dup, out)
        if self.mrmode:
            self.host_actions(st, out)
        if self.fam == 'E':
            self.minter_actions(st, out)
        return out

    @staticmethod
    def setcl(st, ci, cl, **kw):
        cls = list(st.cl)
        cls[ci] = cl._replace(**kw) if kw else cl
        return st._replace(cl=tuple(cls))

    def clone_actions(self, st, c, cl, R, main_dup, out):
        cfg, C = self.cfg, CL[c]
        T = tree(cl.wt)
        blocked = cl.busy is not None or cl.unc is not None
        idle = self.idle(c, cl, st)

        if not cl.online:
            out.append((f"{C}: reconnect", self.setcl(st, c, cl, online=True), []))

        # create -- optionally with one citation written in the same commit:
        # the new entity is cited from an existing one (as in G-0745), or the
        # new entity cites an entity already in this clone's tree
        if cl.n < cfg.ents[c] and not blocked:
            e = f"{C.lower()}{cl.n + 1}"
            for l in self.alloc(c, cl, e):
                kw = dict(wt=cl.wt | {('N', e, l, c)}, n=cl.n + 1)
                if cfg.design == 'Beager' and cl.online:
                    kw['busy'] = e
                ns = self.setcl(st, c, cl, **kw)
                out.append((f"{C}: create {e} as {fmt(l)}", ns, []))
                if cl.c < cfg.cites[c]:
                    rid = f"r{C}{cl.c + 1}"
                    T2 = tree(kw['wt'])
                    views = self.view_trees(cl.rt)
                    for (t, tl) in [(e, l)] + sorted(T.label.items()):
                        if self.widened and (len(resolve(T2, tl)) > 1
                                             or any(x != t for W in views for x in resolve(W, tl))):
                            continue      # A+: refuse a citation whose id is ambiguous in the view
                        kw2 = dict(kw, wt=kw['wt'] | {('F', rid, t, tl, c)}, c=cl.c + 1)
                        ns = self.setcl(st, c, cl, **kw2)
                        out.append((f"{C}: create {e} as {fmt(l)} + cite {t} as {fmt(tl)} ({rid})", ns, []))

        # cross-branch citation of an entity on another clone's unmerged branch
        if cfg.xcite and st.xc == 0 and self.mrmode and not blocked:
            views = self.view_trees(cl.rt)
            Mt = tree(cl.rt[0])
            for d in range(self.n):
                b = cl.rt[1][d]
                if d == c or b is None:
                    continue
                V = tree(b)
                for e in sorted(V.label):
                    if e in T.label or e in Mt.label:
                        continue
                    l = V.label[e]
                    if self.widened and (resolve(T, l) or any(x != e for W in views for x in resolve(W, l))):
                        continue        # A+: refuse a citation whose id is ambiguous in the view
                    ns = self.setcl(st, c, cl, wt=cl.wt | {('F', 'x', e, l, c)})._replace(xc=1)
                    out.append((f"{C}: cross-cite {e} (on {CL[d]}'s branch) as {fmt(l)} (x)", ns, []))

        # fetch (prunes deleted branches); an idle clone does not fetch
        mrn = (st.mr[c][0] if st.mr[c] else None) if self.fam == 'D' else cl.mrnum
        if cl.online and not idle and (cl.rt != R or mrn != cl.mrnum):
            out.append((f"{C}: fetch", self.setcl(st, c, cl, rt=R, mrnum=mrn), []))

        # pull = fetch + merge origin/main into the working branch; an idle
        # clone pulls only when main carries a duplicate id someone must repair
        if cl.online and not st.main <= cl.wt and (not idle or (main_dup and c == self.repairer)):
            u = cl.wt | st.main
            ucl = cl._replace(rt=R, mrnum=mrn)
            if not tree(u).conflict:
                out.append((f"{C}: pull (fetch + merge origin/main)", self.setcl(st, c, ucl, wt=u), []))
            elif cfg.design == 'Eslug' and self.mrmode:
                self.slug_rename(st, c, ucl, u, out)

        # push
        pushed = st.br[c] if self.mrmode else cl.rt[0]
        if cl.online and not blocked and not cl.wt <= cl.rt[0] and cl.wt != pushed:
            if not cfg.hook or not self.clone_findings(c, cl, st):
                self.push(st, c, cl, out)

        # a clone whose branch is fully merged does not act on it, except the
        # designated repairer, which has pulled main
        if self.fam == 'S0' and (not idle or c == self.repairer):
            self.realloc(st, c, cl, out)
        if self.fam == 'B' and cl.online:
            self.claim(st, c, cl, R, out)
        if self.fam == 'D':
            self.dactions(st, c, cl, out)
        if self.fixref:
            self.fix_refs(st, c, cl, out)

        if cfg.abandon and st.ab == 0 and st.br[c] is not None and not cl.wt <= st.main:
            br = list(st.br); br[c] = None
            mr = list(st.mr); mr[c] = None
            ns = self.setcl(st, c, cl, active=False)._replace(br=tuple(br), mr=tuple(mr), ab=1)
            out.append((f"{C}: abandon branch (remote branch deleted, MR closed)", ns, []))

    # ------------------------------------------------------------
    def outcomes(self, st):
        o = [('ok', True, True)]
        if self.cfg.lost and st.flt == 0:
            o += [('lost-applied', True, False), ('lost-unapplied', False, False)]
        return o

    def push(self, st, c, cl, out):
        C = CL[c]
        outs = self.outcomes(st) if not self.mrmode else [('ok', True, True)]
        for tag, applied, reply in outs:
            evs = []
            ns = st
            if not self.mrmode:
                if not st.main <= cl.wt:
                    continue                       # non-fast-forward: rejected, nothing changes
                if applied:
                    ns = ns._replace(main=cl.wt, pub=self.publish(st, [cl.wt], (), evs))
                rt = (cl.wt, cl.rt[1], cl.rt[2]) if reply else cl.rt
            else:
                br, mr, mrc = list(st.br), list(st.mr), st.mrcount
                if applied:
                    br[c] = cl.wt
                    pub = self.publish(st, [cl.wt], (), evs)
                    if self.fam != 'D' and mr[c] is None:
                        mr[c] = (0, None, None)      # MR numbers matter only to D
                    if mr[c] is not None:
                        mr[c] = (mr[c][0], self.pipeline_ok(cl.wt, st.main, tuple(br)), cl.wt)
                    ns = ns._replace(br=tuple(br), mr=tuple(mr), mrcount=mrc, pub=pub)
                rt = cl.rt
            ns = self.setcl(ns, c, cl, rt=rt)
            if tag != 'ok':
                ns = ns._replace(flt=1)
            label = f"{C}: push {'branch' if self.mrmode else 'main'}"
            if tag != 'ok':
                label += f" [FAULT: reply lost, push {'applied' if applied else 'not applied'}]"
            out.append((label, ns, evs))

    def realloc(self, st, c, cl, out):
        C = CL[c]
        T = tree(cl.wt)
        views = self.view_trees(cl.rt)
        trunk = tree(cl.rt[0])
        targets = []
        for f in self.clone_findings(c, cl, st):
            if f[0] in ('tcoll', 'xcoll'):
                targets.append((f[1], f[2], 'branch side of a trunk collision' if f[0] == 'tcoll'
                                else 'collides with an id on another branch'))
            elif f[0] == 'dup':
                l, es = f[1], f[2]
                intrunk = [e for e in es if e in trunk.label]   # add commit is an ancestor of trunk
                if len(intrunk) == len(es) - 1:
                    targets += [(l, e, 'tiebreaker: the other side is in trunk') for e in es if e not in intrunk]
                else:
                    targets += [(l, e, 'human picks: ancestry cannot decide') for e in es]
        if targets and renumber_count(st) >= self.cfg.max_renums:
            self.budget_hit = True
            return
        seen = set()
        for (l, e, why) in targets:
            if (l, e) in seen:
                continue
            seen.add((l, e))
            new = max_num(T, *views) + 1
            evs = []
            ops = [('R', e, l, new, 'renum')] + self.rewrite_ops(T, e, l, new)
            u = cl.wt | frozenset(ops)
            if tree(u).conflict:
                continue
            rw = [o[1] for o in ops[1:]]
            ns = self.setcl(st, c, cl, wt=u)
            out.append((f"{C}: reallocate {e} {fmt(l)} -> {fmt(new)} ({why}; rewrites {rw or 'no refs'})",
                        ns, evs))

    def slug_rename(self, st, c, cl, u, out):
        """E-slug: the pull hits an add/add conflict on inbox/<slug>.md; the
        clone renames its own slug (rewriting its refs) and completes the merge."""
        C = CL[c]
        T, U = tree(cl.wt), tree(u)
        if not (U.conflict and U.conflict[0] == 'path'):
            return
        l = U.conflict[1]
        for e in T.by_label.get(l, ()):
            if T.origin[e] != c:
                continue
            new = '~' + e
            evs = []
            ops = frozenset([('R', e, l, new, 'slug')] + self.rewrite_ops(T, e, l, new))
            u2 = cl.wt | ops | st.main
            if tree(u2).conflict:
                continue
            ns = self.setcl(st, c, cl, wt=u2)
            out.append((f"{C}: pull hits add/add on {l}; rename slug of {e} {l} -> {new} and merge", ns, evs))

    def claim(self, st, c, cl, R, out):
        cfg, C, p = self.cfg, CL[c], self.cfg.probes
        T = tree(cl.wt)
        if cl.unc is not None:
            e, L = cl.unc
            if 'assume_ok' in p or 'noreconcile' in p:
                return
            ok = (L, e) in st.ledger if 'bylabel' not in p else any(l == L for (l, _) in st.ledger)
            ns = self.setcl(st, c, cl, rt=R, unc=None)
            evs = []
            if ok:
                ns = self.apply_claim(ns, c, ns.cl[c], e, L, evs)
            out.append((f"{C}: reconcile lost claim of {fmt(L)} for {e} (fetch ledger): "
                        f"{'claimed' if ok else 'not claimed, will retry'}", ns, evs))
            return
        for e, l in sorted(T.label.items()):
            if not is_handle(l):
                continue
            if cfg.design == 'Beager' and cl.busy is not None and cl.busy != e:
                continue
            L = max([x for (x, _) in cl.rt[2]] + [0]) + 1
            if st.ledger != cl.rt[2] and 'nocas' not in p:
                continue                                   # CAS fails: rejected, must fetch
            newledger = (cl.rt[2] if 'nocas' in p else st.ledger) + ((L, e),)
            for tag, applied, reply in self.outcomes(st):
                evs = []
                ns = st
                if applied:
                    ns = ns._replace(ledger=newledger, pub=self.publish(st, [], [(L, e)], evs))
                if reply or 'assume_ok' in p:
                    ncl = ns.cl[c]._replace(rt=(cl.rt[0], cl.rt[1], newledger) if reply else cl.rt)
                    ns = self.setcl(ns, c, ncl)
                    ns = self.apply_claim(ns, c, ns.cl[c], e, L, evs)
                else:
                    ns = self.setcl(ns, c, ns.cl[c], unc=(e, L))
                if tag != 'ok':
                    ns = ns._replace(flt=1)
                lab = f"{C}: claim {fmt(L)} for {e} on the ledger{' (force push)' if 'nocas' in p else ''}"
                if tag != 'ok':
                    lab += f" [FAULT: reply lost, {'applied' if applied else 'not applied'}]"
                out.append((lab, ns, evs))

    def apply_claim(self, st, c, cl, e, L, evs):
        T = tree(cl.wt)
        old = T.label[e]
        ops = [('R', e, old, L, 'claim')] + self.rewrite_ops(T, e, old, L)
        busy = None if cl.busy == e else cl.busy
        return self.setcl(st, c, cl, wt=cl.wt | frozenset(ops), busy=busy)

    def dactions(self, st, c, cl, out):
        cfg, C = self.cfg, CL[c]
        if cl.online and cl.mrnum is None and not self.idle(c, cl, st):
            if st.mr[c] is None:
                num = st.mrcount + 1
                head = st.br[c]
                ok = self.pipeline_ok(head, st.main, st.br) if head is not None else None
                mr = list(st.mr); mr[c] = (num, ok, head)
                ns = st._replace(mr=tuple(mr), mrcount=num)
                out.append((f"{C}: open MR !{num}", self.setcl(ns, c, cl, mrnum=num), []))
                if cfg.lost and st.flt == 0:
                    out.append((f"{C}: open MR !{num} [FAULT: reply lost, MR created]",
                                ns._replace(flt=1), []))
            else:
                out.append((f"{C}: open MR -> host reports the existing MR !{st.mr[c][0]}",
                            self.setcl(st, c, cl, mrnum=st.mr[c][0]), []))
        if cl.mrnum is not None:
            T = tree(cl.wt)
            hs = sorted(e for e, l in T.label.items() if is_handle(l) and T.origin[e] == c)
            if hs:
                k = sum(1 for l in T.label.values() if isinstance(l, str) and l.startswith(f"!{cl.mrnum}."))
                ops, evs = [], []
                for e in hs:
                    k += 1
                    new = f"!{cl.mrnum}.{k}"
                    ops += [('R', e, T.label[e], new, 'mr')] + self.rewrite_ops(T, e, T.label[e], new)
                ns = self.setcl(st, c, cl, wt=cl.wt | frozenset(ops))
                out.append((f"{C}: assign MR ids to {hs}", ns, evs))

    def fix_refs(self, st, c, cl, out):
        C = CL[c]
        T = tree(cl.wt)
        views = [T] + self.view_trees(cl.rt)
        for rid, (meant, l, o) in sorted(T.refs.items()):
            if not is_handle(l) or resolve(T, l):
                continue
            hits = {e for V in views for e, pr in V.prior.items() if l in pr}
            if len(hits) != 1:
                continue
            e = next(iter(hits))
            cur = next((V.label[e] for V in views if e in V.label and is_id(V.label[e])), None)
            if cur is None:
                continue
            ns = self.setcl(st, c, cl, wt=cl.wt | {('W', rid, l, cur)})
            out.append((f"{C}: follow renamed handle in {rid}: {l} -> {fmt(cur)}", ns, []))

    # ------------------------------------------------------------
    def host_actions(self, st, out):
        cfg = self.cfg
        for c in range(self.n):
            mr, b = st.mr[c], st.br[c]
            if mr is None or b is None or b <= st.main:
                continue
            if cfg.pipeline and not (mr[1] and mr[2] == b):
                continue
            if self.utd and not st.main <= b:
                continue
            u = st.main | b
            if tree(u).conflict:
                continue
            evs = []
            pub = self.publish(st, [u], (), evs)
            mrs = list(st.mr); mrs[c] = None
            out.append((f"host: merge {'MR !%d' % mr[0] if mr[0] else 'MR'} ({CL[c]}'s branch)",
                        st._replace(main=u, mr=tuple(mrs), pub=pub), evs))

    def minter_actions(self, st, out):
        p = self.cfg.probes
        if self.cfg.minters == 1:
            # one mint trigger: fetch-then-CAS-push is equivalent to an atomic
            # mint, since a rejected push only retries against the new main
            ops, newl = self.mint_ops(st.main)
            if ops:
                newmain = st.main | frozenset(ops)
                if not tree(newmain).conflict:
                    evs = []
                    ns = st._replace(main=newmain, pub=self.publish(st, [newmain], (), evs))
                    desc = ', '.join(f"{e}->{fmt(v)}" for e, v in sorted(newl.items())) or 'refs only'
                    out.append((f"minter: mint commit on main ({desc})", ns, evs))
            return
        for m in range(self.cfg.minters):
            snap = st.mint[m]
            mi = list(st.mint)
            only = m if 'mint_own' in p else None
            if snap is None:
                if self.mint_ops(st.main, only)[0]:
                    mi[m] = st.main
                    out.append((f"minter{m}: fetch main (provisional entities pending)",
                                st._replace(mint=tuple(mi)), []))
                continue
            mi[m] = None
            ops, newl = self.mint_ops(snap, only)
            if not ops:
                out.append((f"minter{m}: nothing left to mint", st._replace(mint=tuple(mi)), []))
                continue
            if st.main == snap or 'nocas_mint' in p:
                outs = [('ok', True)]
                if self.cfg.lost and st.flt == 0:
                    outs.append(('lost-unapplied', False))
            else:
                outs = [('rejected', False)]
            for tag, applied in outs:
                evs = []
                ns = st._replace(mint=tuple(mi))
                if applied:
                    newmain = st.main | frozenset(ops)     # == snap | ops when the CAS held
                    if tree(newmain).conflict:
                        continue
                    ns = ns._replace(main=newmain, pub=self.publish(st, [newmain], (), evs))
                if tag == 'lost-unapplied':
                    ns = ns._replace(flt=1)
                desc = ', '.join(f"{e}->{fmt(v)}" for e, v in sorted(newl.items())) or 'refs only'
                lab = {'ok': f"minter{m}: push mint commit ({desc}){' without CAS' if 'nocas_mint' in p else ''}",
                       'rejected': f"minter{m}: push mint commit rejected (main moved)",
                       'lost-unapplied': f"minter{m}: push mint commit [FAULT: reply lost, not applied]"}[tag]
                out.append((lab, ns, evs))

    # ------------------------------------------------------------ invariants
    @staticmethod
    def state_events(st):
        M = tree(st.main)
        evs = []
        d = dup_ids(M)
        if d:
            evs.append(('I1', 'duplicate id on main: ' + ', '.join(f"{fmt(l)}={list(es)}" for l, es in d.items())))
        for rid, (meant, l, o) in sorted(M.refs.items()):
            res = resolve(M, l)
            if res and meant not in res:
                if any(op[0] == 'W' and op[1] == rid for op in st.main):
                    cause = 'rewrite'          # a renumber/rename rewrote it onto the wrong entity
                elif meant in M.label:
                    cause = 'stale'            # its target was renumbered after it was written
                else:
                    cause = 'taken'            # its target is not on main; another entity holds the id
                d = f"main: {rid}, written by {CL[o]} meaning {meant}, reads {fmt(l)} = {list(res)} ({cause})"
                evs.append(('I2', d))
                evs.append(('I2:' + cause, d))
        return evs

    def terminal(self, st):
        M = tree(st.main)
        probs = []
        self.budget_hit = False
        self.successors(st)
        if self.budget_hit:
            probs.append('renumber-budget')
        if dup_ids(M):
            probs.append('dup-on-main')
        if any(is_handle(l) for l in M.label.values()):
            probs.append('provisional-on-main')
        for rid, (meant, l, o) in sorted(M.refs.items()):
            res = resolve(M, l)
            if not res:
                probs.append(f'dangling:{rid}')
            elif tuple(res) != (meant,):
                probs.append(f'misdirected:{rid}')
        reasons = []
        for c, cl in enumerate(st.cl):
            if not cl.active:
                continue
            T = tree(cl.wt)
            unl = [e for e in T.label if e not in M.label] + [r for r in T.refs if r not in M.refs]
            if unl:
                kinds = {f[0] for f in self.clone_findings(c, cl, st)}
                if tree(cl.wt | st.main).conflict:
                    kinds.add('conflict')        # git cannot merge main in: a human resolves it
                reasons.append((CL[c], unl, sorted(kinds)))
        return probs, reasons


# ---------------------------------------------------------------------- search
class Result:
    pass


def explore(cfg, want_graph=True, time_limit=None):
    m = Model(cfg)
    t0 = time.time()
    init = m.initial()
    idx = {init: 0}
    states, parent, depth = [init], [(-1, None)], [0]
    edges = [] if want_graph else None
    q = deque([0])
    viol, terms = {}, []
    truncated = False
    nedges = 0

    def note(inv, detail, i, act):
        if inv not in viol:
            viol[inv] = (i, act, detail)

    for ev in m.state_events(init):
        note(ev[0], ev[1], 0, None)
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
                for ev in m.state_events(ns):
                    note(ev[0], ev[1], j, None)
            if want_graph:
                edges[i].append(j)
            for ev in evs:
                note(ev[0], ev[1], i, act)
        if len(states) > cfg.max_states or (time_limit and time.time() - t0 > time_limit):
            truncated = True
            break

    r = Result()
    r.cfg, r.model, r.states, r.parent, r.viol = cfg, m, states, parent, viol
    r.nstates, r.nedges, r.truncated = len(states), nedges, truncated
    # with BFS, every state shallower than the first unexpanded one was expanded
    r.depth_complete = depth[q[0]] if (truncated and q) else max(depth)
    r.max_depth = max(depth)
    r.secs = time.time() - t0
    r.term_done, r.term_refused, r.term_stuck, r.term_eventual = 0, {}, None, {}
    r.renums = []
    for i in terms:
        st = states[i]
        probs, reasons = m.terminal(st)
        r.renums.append(renumber_count(st))
        for pb in probs:
            key = ('I1ev' if pb == 'dup-on-main' else
                   'I2ev' if pb.startswith(('dangling', 'misdirected')) else
                   'I5livelock' if pb == 'renumber-budget' else 'I5')
            r.term_eventual.setdefault(key, (i, pb))
        if not reasons and not probs:
            r.term_done += 1
            continue
        human = [x for x in reasons if x[2] and set(x[2]) <= HUMAN_REMEDY]
        stuck = [x for x in reasons if not x[2] or not set(x[2]) <= HUMAN_REMEDY]
        if ('provisional-on-main' in probs or 'renumber-budget' in probs
                or ('dup-on-main' in probs and m.fam != 'C')):
            stuck.append(('main', probs, []))
        if stuck:
            if r.term_stuck is None:
                r.term_stuck = (i, stuck)
        elif human:
            key = ','.join(sorted({k for x in human for k in x[2]}))
            r.term_refused.setdefault(key, (i, human))
        else:
            r.term_done += 1
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
    return r


def renumber_count(st):
    ops = set(st.main)
    for b in st.br:
        if b is not None:
            ops |= b
    for cl in st.cl:
        ops |= cl.wt
    return sum(1 for o in ops if o[0] == 'R' and o[4] in ('renum', 'slug'))


# ---------------------------------------------------------------------- traces
def path(r, i):
    steps = []
    while i > 0:
        pi, a = r.parent[i]
        steps.append((a, i))
        i = pi
    steps.reverse()
    return steps


def summarize(st, n):
    def ents(T):
        return ' '.join(f"{e}={fmt(l)}" for e, l in sorted(T.label.items())) or '-'

    def refs(T):
        return ' '.join(f"{r}:{fmt(l)}(meant {m})" for r, (m, l, o) in sorted(T.refs.items())) or '-'
    M = tree(st.main)
    lines = [f"       main: {ents(M)} | refs {refs(M)}"]
    for c in range(n):
        b, cl = st.br[c], st.cl[c]
        W = tree(cl.wt)
        extra = ('' if cl.active else ' (abandoned)') + ('' if cl.online else ' (offline)')
        rb = '' if b is None else f" | pushed: {ents(tree(b))}"
        lines.append(f"       {CL[c]} working: {ents(W)} | refs {refs(W)}{rb}{extra}")
    if st.ledger:
        lines.append("       ledger: " + ' '.join(f"{fmt(l)}:{e}" for l, e in st.ledger))
    return '\n'.join(lines)


def render_trace(r, i, act, detail, title):
    n = len(r.cfg.ents)
    out = [title, f"violation: {detail}", "", "0. initial state: empty default branch"]
    steps = path(r, i)
    for k, (a, j) in enumerate(steps, 1):
        out.append(f"{k}. {a}")
        out.append(summarize(r.states[j], n))
    if act is not None:
        out.append(f"{len(steps) + 1}. {act}   <-- the violating step")
    return '\n'.join(out)
