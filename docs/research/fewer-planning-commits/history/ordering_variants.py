"""Replay amend under ritual-order variants, without ROADMAP-only ritual commits.

`rituals-as-verbs` reports every ritual commit except the ROADMAP-only ones as a
verb commit, so it can be amended; the ROADMAP-only ones stay ritual commits so
that --drop-roadmap still drops them. Writes one --dump file per variant.
"""
import io, contextlib, runpy, sys
sys.path.insert(0, ".")
import base
orig = base.klass
VARIANTS = [
    ("baseline", False, []),
    ("evidence cites the code commit", False, ["--no-sha-check"]),
    ("rituals commit through verbs", True, []),
    ("both", True, ["--no-sha-check"]),
]
for name, rituals_as_verbs, extra in VARIANTS:
    base.klass = (lambda h: "kernel" if orig(h) == "ritual" and not base.roadmap_only(h) else orig(h)) if rituals_as_verbs else orig
    dump = "variant_" + name.split()[0] + ".json"
    sys.argv = ["amend.py", "--drop-roadmap", "--dump", dump] + extra
    out = io.StringIO()
    with contextlib.redirect_stdout(out):
        runpy.run_path("amend.py", run_name="__main__")
    rem = [l for l in out.getvalue().splitlines() if "PLANNING COMMITS REMAINING" in l or "dropped" in l.lower()]
    print(f"{name:34s} {' | '.join(rem)}  (dump {dump})")
base.klass = orig
