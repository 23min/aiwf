#!/usr/bin/env bash
# mutate-diff — diff-scoped mutation testing (advisory, G-0267).
#
# Runs gremlins on just the internal/ Go packages changed since the
# merge-base with origin/main, instead of the whole kernel. It is the
# diff-scoped companion to wf-vacuity's manual probe and the
# whole-package `mutate-hunt` workflow: where the coverage gate
# (G-0067) proves a changed line *ran*, this probes whether the
# assertions on that line actually *kill* a mutant.
#
# Advisory by design: it prints surviving mutants for human triage and
# always exits 0. Mutation is slow and equivalent-mutant / unreachable
# noise makes "0 survivors" un-gateable without judgment — this is a
# signal, not a blocking gate (the same reason `mutate-hunt` is
# workflow_dispatch-only).
#
# Scope is internal/ only — the kernel-logic surface mutate-hunt
# targets by default; cmd/main.go is integration-tested, not
# unit-mutation-tested. A test-only diff still changes its package, so
# gremlins re-mutates that package's production code against the new
# assertions (the "did this new test strengthen anything?" question).
#
# Overrides (env):
#   MUTATE_DIFF_BASE         base ref to diff against (default origin/main)
#   MUTATE_DIFF_COEFFICIENT  gremlins --timeout-coefficient (default 15)
#
# Requires gremlins and jq; absence of either is reported and treated
# as a non-failure (advisory).
set -euo pipefail

BASE_REF="${MUTATE_DIFF_BASE:-origin/main}"
COEFF="${MUTATE_DIFF_COEFFICIENT:-15}"

if ! command -v gremlins >/dev/null 2>&1; then
	echo "mutate-diff: gremlins not installed — advisory tool unavailable. Install with:" >&2
	echo "  go install github.com/go-gremlins/gremlins/cmd/gremlins@latest" >&2
	exit 0
fi
if ! command -v jq >/dev/null 2>&1; then
	echo "mutate-diff: jq not installed — required to summarize gremlins' JSON report." >&2
	exit 0
fi

# Resolve the diff base. merge-base mirrors the coverage gate (G-0067):
# "changed since we forked from trunk", not "changed since trunk's tip".
base="$(git merge-base "$BASE_REF" HEAD 2>/dev/null || true)"
if [ -z "$base" ]; then
	echo "mutate-diff: cannot resolve base ref '$BASE_REF' — nothing to mutate."
	exit 0
fi

# Changed internal/ packages: committed-since-base + working-tree
# (tracked) via `git diff`, plus untracked new files via `git ls-files
# --others`. Map each .go file to its package dir, restrict to
# internal/, dedup.
pkgs="$(
	{
		git diff --name-only "$base" -- '*.go'
		git ls-files --others --exclude-standard -- '*.go'
	} | grep -E '^internal/' | xargs -r -n1 dirname | sort -u | sed 's#^#./#' || true
)"

if [ -z "$pkgs" ]; then
	echo "mutate-diff: no changed internal/ Go packages vs ${BASE_REF} — nothing to mutate."
	exit 0
fi

npkgs="$(printf '%s\n' "$pkgs" | grep -c .)"
echo "mutate-diff: ${npkgs} changed internal/ package(s) vs ${BASE_REF} (base ${base:0:12}):"
printf '%s\n' "$pkgs" | sed 's/^/  /'
echo

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

total_lived=0
for pkg in $pkgs; do
	# A package whose only change since base is a file deletion derives a
	# dir that no longer exists; skip it rather than emit gremlins'
	# confusing "no such file or directory" FAIL block.
	[ -d "${pkg#./}" ] || continue
	echo "── mutate-diff: gremlins ${pkg} ──"
	report="${tmpdir}/$(printf '%s' "$pkg" | tr '/.' '__').json"
	# --workers 1 / --timeout-coefficient: see the mutate-hunt workflow
	# header for the rationale (default workers time out on this repo).
	gremlins unleash --workers 1 --timeout-coefficient "$COEFF" --output "$report" "$pkg" || true

	[ -f "$report" ] || continue
	# Self-authored, stable survivor markers from the JSON — decoupled
	# from gremlins' stdout wording. LIVED = covered-but-survived (the
	# vacuity signal this tool exists to surface). NOT_COVERED overlaps
	# the diff-scoped coverage gate (G-0067), so it is left to gremlins'
	# own output above and not re-tallied here.
	lived="$(jq -r --arg pkg "$pkg" '
		.files[]? as $f
		| $f.mutations[]?
		| select(.status == "LIVED")
		| "  SURVIVOR LIVED \($f.file_name):\(.line):\(.column) (\(.type)) in \($pkg)"
	' "$report")"
	if [ -n "$lived" ]; then
		printf '%s\n' "$lived"
		total_lived=$((total_lived + $(printf '%s\n' "$lived" | grep -c '^  SURVIVOR ')))
	fi
done

echo
if [ "$total_lived" -gt 0 ]; then
	echo "mutate-diff: ${total_lived} surviving mutant(s) (LIVED) in changed internal/ packages — ADVISORY."
	echo "  Triage per wf-vacuity §1: strengthen the assertion that should kill each, or"
	echo "  confirm it is an equivalent/unreachable mutant (expected noise — do not chase)."
else
	echo "mutate-diff: no surviving mutants (LIVED) in changed internal/ packages — advisory pass."
fi
exit 0
