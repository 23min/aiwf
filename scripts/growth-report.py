#!/usr/bin/env python3
"""Snapshot the growth metrics that docs/design/growth.md tracks.

Advisory and read-only: it measures, it never gates. Every metric is derived
from git history and a tree snapshot, so any past commit can be measured as
readily as HEAD -- pass --at <rev> to reconstruct an earlier baseline, and
--baseline <rev> to print the delta beside it.

    scripts/growth-report.py                      # HEAD
    scripts/growth-report.py --baseline v0.20.0   # HEAD vs a tag
    scripts/growth-report.py --at <sha> --tsv     # a past commit, machine-readable
"""

from __future__ import annotations

import argparse
import collections
import datetime as dt
import re
import subprocess
import sys
from dataclasses import dataclass, field

# A policy names itself either inline at the Violation construction site or via
# a local `policyID` constant it then references; both spellings are in use, so
# counting only the first silently under-reports the corpus.
POLICY_ID = re.compile(r'Policy:\s*"([^"]+)"|policyID\s*=\s*"([^"]+)"')
# Placeholder ids used inside the policy corpus' own test fixtures; they name
# no real chokepoint.
FIXTURE_POLICY_IDS = {"<id>", "<kebab-id>", "alpha-dark", "beta-lit"}
FRONTMATTER = re.compile(r"^---\n(.*?)\n---\n(.*)$", re.S)
TRAILER_VERB = re.compile(r"^aiwf-verb:\s*(\S+)", re.M)
TRAILER_ENTITY = re.compile(r"^aiwf-entity:\s*(\S+)", re.M)
CLOSING_TRANSITION = re.compile(r"->\s*(addressed|wontfix)")


def git(*args: str) -> str:
    return subprocess.run(
        ["git", *args], capture_output=True, text=True, check=True
    ).stdout


def read_blobs(rev: str, paths: list[str]) -> dict[str, str]:
    """Read many blobs in one `git cat-file --batch` pass.

    One subprocess for the whole tree rather than one per file: a snapshot
    touches ~1,500 blobs, and the per-process cost dominates everything else.
    """
    if not paths:
        return {}
    stdin = "".join(f"{rev}:{p}\n" for p in paths)
    proc = subprocess.run(
        ["git", "cat-file", "--batch"],
        input=stdin.encode(),
        capture_output=True,
        check=True,
    )
    out, cursor, blobs = proc.stdout, 0, {}
    for path in paths:
        nl = out.index(b"\n", cursor)
        header = out[cursor:nl].decode()
        cursor = nl + 1
        if header.endswith(("missing", "ambiguous")):
            continue
        size = int(header.rsplit(" ", 1)[1])
        blobs[path] = out[cursor : cursor + size].decode("utf-8", "replace")
        cursor += size + 1  # trailing newline git appends after each blob
    return blobs


@dataclass
class Snapshot:
    rev: str
    date: str
    prod_lines: int = 0
    test_lines: int = 0
    policy_files: int = 0
    policy_lines: int = 0
    policy_ids: int = 0
    entity_files: int = 0
    entity_words: int = 0
    gaps_total: int = 0
    gap_status: dict[str, int] = field(default_factory=dict)
    shipped_words: int = 0
    docs_words: int = 0

    @property
    def test_ratio(self) -> float:
        return self.test_lines / max(self.prod_lines, 1)

    @property
    def policy_share(self) -> float:
        return 100 * self.policy_lines / max(self.prod_lines, 1)


def snapshot(rev: str) -> Snapshot:
    date = git("show", "-s", "--format=%ad", "--date=short", rev).strip()
    tree = git("ls-tree", "-r", "--name-only", rev).splitlines()

    def want(pred) -> list[str]:
        return [p for p in tree if pred(p)]

    go_prod = want(lambda p: p.startswith("internal/") and p.endswith(".go") and not p.endswith("_test.go"))
    go_test = want(lambda p: p.startswith("internal/") and p.endswith("_test.go"))
    policies = want(lambda p: p.startswith("internal/policies/") and p.endswith(".go"))
    entities = want(lambda p: p.startswith("work/") and p.endswith(".md"))
    shipped = want(lambda p: p.startswith("internal/skills/embedded") and p.endswith(".md"))
    docs = want(lambda p: p.startswith("docs/") and p.endswith(".md"))

    blobs = read_blobs(rev, go_prod + go_test + policies + entities + shipped + docs)
    lines = lambda paths: sum(blobs.get(p, "").count("\n") for p in paths)
    words = lambda paths: sum(len(blobs.get(p, "").split()) for p in paths)

    snap = Snapshot(rev=rev[:9], date=date)
    snap.prod_lines = lines(go_prod)
    snap.test_lines = lines(go_test)
    snap.policy_files = len(policies)
    snap.policy_lines = lines(policies)
    ids = set()
    for path in policies:
        for inline, viaconst in POLICY_ID.findall(blobs.get(path, "")):
            ids.add(inline or viaconst)
    snap.policy_ids = len(ids - FIXTURE_POLICY_IDS)
    snap.entity_files = len(entities)
    snap.entity_words = words(entities)
    snap.shipped_words = words(shipped)
    snap.docs_words = words(docs)

    status = collections.Counter()
    for path in entities:
        if "/gaps/" not in path:
            continue
        match = FRONTMATTER.match(blobs.get(path, ""))
        if not match:
            continue
        found = re.search(r"^status:\s*(\S+)", match.group(1), re.M)
        status[found.group(1) if found else "?"] += 1
    snap.gaps_total = sum(status.values())
    snap.gap_status = dict(status)
    return snap


@dataclass
class GapFlow:
    opened: int
    closed: int
    same_day: int
    open_now: int
    median_age_days: float

    @property
    def same_day_share(self) -> float:
        return 100 * self.same_day / max(self.closed, 1)


def gap_flow(rev: str) -> GapFlow:
    """Gap open/close flow read from commit trailers, the audit trail itself."""
    log = git("log", "--reverse", "--format=%H%x01%aI%x01%s%x01%b%x02", rev)
    opened: dict[str, dt.datetime] = {}
    closed: dict[str, dt.datetime] = {}
    latest = None
    for chunk in log.split("\x02"):
        parts = chunk.strip("\n").split("\x01")
        if len(parts) < 4:
            continue
        _, iso, subject, body = parts[:4]
        entity = TRAILER_ENTITY.search(body)
        if not entity or not entity.group(1).startswith("G-"):
            continue
        gid = "G-%04d" % int(re.sub(r"\D", "", entity.group(1)))
        when = dt.datetime.fromisoformat(iso)
        latest = when if latest is None else max(latest, when)
        verb = TRAILER_VERB.search(body)
        if verb and verb.group(1) == "add":
            opened.setdefault(gid, when)
        if CLOSING_TRANSITION.search(subject):
            closed.setdefault(gid, when)

    lifetimes = [
        (closed[g] - opened[g]).total_seconds() / 86400 for g in opened if g in closed
    ]
    still = sorted(
        (latest - opened[g]).total_seconds() / 86400 for g in opened if g not in closed
    )
    return GapFlow(
        opened=len(opened),
        closed=len(lifetimes),
        same_day=sum(1 for d in lifetimes if d < 1),
        open_now=len(still),
        median_age_days=still[len(still) // 2] if still else 0.0,
    )


def rows(snap: Snapshot, flow: GapFlow) -> list[tuple[str, str, float]]:
    """(label, formatted value, raw value) — raw drives the delta column."""
    return [
        ("production lines (internal/)", f"{snap.prod_lines:,}", snap.prod_lines),
        ("test lines (internal/)", f"{snap.test_lines:,}", snap.test_lines),
        ("test : production ratio", f"{snap.test_ratio:.2f}", snap.test_ratio),
        ("policy files", f"{snap.policy_files:,}", snap.policy_files),
        ("policy chokepoints (distinct ids)", f"{snap.policy_ids:,}", snap.policy_ids),
        ("policy lines", f"{snap.policy_lines:,}", snap.policy_lines),
        ("policy lines as % of production", f"{snap.policy_share:.1f}%", snap.policy_share),
        ("entity files (work/)", f"{snap.entity_files:,}", snap.entity_files),
        ("entity body words", f"{snap.entity_words:,}", snap.entity_words),
        ("words per entity", f"{snap.entity_words / max(snap.entity_files, 1):.0f}", snap.entity_words / max(snap.entity_files, 1)),
        ("gap entities", f"{snap.gaps_total:,}", snap.gaps_total),
        ("gaps open", f"{snap.gap_status.get('open', 0):,}", snap.gap_status.get("open", 0)),
        ("gaps opened (all history)", f"{flow.opened:,}", flow.opened),
        ("gaps closed (all history)", f"{flow.closed:,}", flow.closed),
        ("same-day gap closures", f"{flow.same_day:,}", flow.same_day),
        ("same-day share of closures", f"{flow.same_day_share:.0f}%", flow.same_day_share),
        ("median age of an open gap", f"{flow.median_age_days:.1f}d", flow.median_age_days),
        ("shipped skill + guidance words", f"{snap.shipped_words:,}", snap.shipped_words),
        ("docs/ narrative words", f"{snap.docs_words:,}", snap.docs_words),
    ]


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--at", default="HEAD", help="revision to measure (default: HEAD)")
    ap.add_argument("--baseline", help="revision to compare against; prints a delta column")
    ap.add_argument("--tsv", action="store_true", help="tab-separated output")
    args = ap.parse_args()

    snap = snapshot(args.at)
    current = rows(snap, gap_flow(args.at))
    base = None
    if args.baseline:
        base_snap = snapshot(args.baseline)
        base = (base_snap, rows(base_snap, gap_flow(args.baseline)))

    if args.tsv:
        for i, (label, value, _) in enumerate(current):
            cells = [label, value]
            if base:
                cells.append(base[1][i][1])
            print("\t".join(cells))
        return 0

    print(f"aiwf growth report — {snap.rev} ({snap.date})")
    if base:
        print(f"baseline               — {base[0].rev} ({base[0].date})")
    print()
    width = max(len(label) for label, _, _ in current)
    for i, (label, value, raw) in enumerate(current):
        line = f"  {label:<{width}}  {value:>12}"
        if base:
            was, was_raw = base[1][i][1], base[1][i][2]
            factor = f"{raw / was_raw:.2f}x" if was_raw else "--"
            line += f"  {was:>12}  {factor:>7}"
        print(line)
    print()
    print("  Duplication in the test corpus is measured separately (slow, needs golangci-lint):")
    print("    golangci-lint run --enable-only dupl ./...   # with the _test.go exclusions removed")
    print()
    print("  Interpretation and the levers: docs/design/growth.md")
    return 0


if __name__ == "__main__":
    sys.exit(main())
