#!/usr/bin/env bash
# mutate-diff — diff-scoped mutation testing (advisory, G-0267).
#
# Runs gremlins on just the internal/ Go lines changed since the
# merge-base with origin/main — committed, staged or unstaged — instead
# of the whole kernel. It is the diff-scoped companion to wf-vacuity's
# manual probe and the whole-package `mutate-hunt` workflow: where the
# coverage gate (G-0067) proves a changed line *ran*, this probes
# whether the assertions on that line actually *kill* a mutant.
#
# Advisory by design: it prints surviving mutants for human triage and
# always exits 0. Mutation is slow and equivalent-mutant / unreachable
# noise makes "0 survivors" un-gateable without judgment — this is a
# signal, not a blocking gate (the same reason `mutate-hunt` is
# workflow_dispatch-only).
#
# Scope is internal/ only — the kernel-logic surface mutate-hunt
# targets by default; cmd/main.go is integration-tested, not
# unit-mutation-tested. An untracked file is not in `git diff`, so it is
# named, not mutated; `git add -N <path>` brings it in. A package whose
# changed lines are all in its tests has no code line to mutate; it is
# named, with the whole-package command that asks whether the new tests
# kill anything the old ones missed.
#
# How gremlins is scoped. `gremlins unleash --diff <ref>` keeps the
# lines `git diff --merge-base <ref>` reports added, running that diff
# where gremlins was started and matching its paths against the files
# it walks, which it names relative to its directory. It reads each
# hunk's added lines as one run from the hunk's first change. So each
# run starts inside the top-most changed directory — a changed child
# rides its parent's run — and a git shim ahead of gremlins on PATH
# answers that one diff call relative to the directory, with zero
# context lines so every change is its own hunk. Every other git call,
# the test suites' included, passes straight through. The shim also
# holds that diff's shape against the caller's git config and
# environment, each of which misleads gremlins' parser: color or an
# external diff tool reads as "everything changed", dropped path
# prefixes leave a subdirectory's files unmatched, and wider context
# (diff.interHunkContext, GIT_DIFF_OPTS) merges changes back into one
# hunk. A moved file reads as wholly new. A run gremlins exits non-zero
# from, or whose report jq cannot read, is named FAILED and withholds
# the advisory pass; a directory holding nothing gremlins can mutate
# ends at exit 0 with no report and is named NO MUTANTS.
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

cd "$(git rev-parse --show-toplevel)"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

# Files with added lines, then the directories holding them. Renames
# are off here as in the shim's diff, so a moved file names its new
# directory rather than an `{old => new}` path.
git diff --no-renames --numstat "$base" -- 'internal/*.go' | awk -F '\t' '$1 > 0 { print $3 }' >"$tmpdir/changed"
code_dirs="$(awk '!/_test\.go$/ { sub(/\/[^\/]*$/, ""); print }' "$tmpdir/changed" | LC_ALL=C sort -u)"
test_dirs="$(awk '/_test\.go$/ { sub(/\/[^\/]*$/, ""); print }' "$tmpdir/changed" | LC_ALL=C sort -u)"
test_only="$(LC_ALL=C comm -23 <(printf '%s\n' "$test_dirs") <(printf '%s\n' "$code_dirs") | grep . || true)"
runs="$(printf '%s\n' "$code_dirs" | awk 'NF {
	for (i = 1; i <= n; i++) if (index($0, kept[i] "/") == 1) next
	kept[++n] = $0
	print
}')"

untracked="$(git ls-files --others --exclude-standard -- 'internal/*.go')"
if [ -n "$untracked" ]; then
	echo "mutate-diff: untracked files are not mutated — stage each to include it (git add -N <path>):"
	printf '%s\n' "$untracked" | sed 's/^/  UNTRACKED /'
	echo
fi

report_test_only() {
	[ -n "$test_only" ] || return 0
	echo
	echo "mutate-diff: only test lines changed in these packages — no code line to mutate."
	echo "  To ask whether the new tests kill anything the old ones missed, mutate the whole package:"
	printf '%s\n' "$test_only" | while IFS= read -r pkg; do
		echo "  TEST-ONLY ./${pkg}"
		echo "    gremlins unleash --workers 1 --timeout-coefficient ${COEFF} ./${pkg}"
	done
}

if [ -z "$runs" ]; then
	echo "mutate-diff: no changed internal/ Go code lines vs ${BASE_REF} — nothing to mutate."
	report_test_only
	exit 0
fi

# Absolute paths only: tests run git under a PATH of their own, which
# may hold neither bash nor env.
mkdir "$tmpdir/bin"
cat >"$tmpdir/bin/git" <<EOF
#!/bin/sh
if [ "\$#" -eq 3 ] && [ "\$1" = diff ] && [ "\$2" = --merge-base ]; then
	unset GIT_DIFF_OPTS
	exec '$(command -v git)' diff -U0 --inter-hunk-context=0 --no-renames --no-color --no-ext-diff --src-prefix=a/ --dst-prefix=b/ --relative --merge-base "\$3"
fi
exec '$(command -v git)' "\$@"
EOF
chmod +x "$tmpdir/bin/git"

echo "mutate-diff: changed internal/ Go lines vs ${BASE_REF} (base ${base:0:12}), in $(printf '%s\n' "$runs" | grep -c .) run(s):"
printf '%s\n' "$runs" | sed 's#^#  ./#'
echo

report="$tmpdir/report.json"
total_lived=0
failed=0
for dir in $runs; do
	echo "── mutate-diff: gremlins ./${dir} ──"
	rm -f "$report"
	rc=0
	# --workers 1 / --timeout-coefficient: see the mutate-hunt workflow
	# header for the rationale (default workers time out on this repo).
	# --output-statuses leaves out the SKIPPED unchanged lines.
	(cd "$dir" && PATH="$tmpdir/bin:$PATH" gremlins unleash --diff "$base" --workers 1 --timeout-coefficient "$COEFF" \
		--output-statuses lctkv --output "$report") || rc=$?
	if [ "$rc" -ne 0 ]; then
		echo "  FAILED ./${dir} — gremlins exited ${rc}."
		failed=$((failed + 1))
		continue
	fi
	if [ ! -e "$report" ]; then
		echo "  NO MUTANTS ./${dir} — gremlins found nothing to mutate."
		continue
	fi
	# Self-authored, stable survivor markers from the JSON — decoupled
	# from gremlins' stdout wording. LIVED = covered-but-survived (the
	# vacuity signal this tool exists to surface).
	if ! lived="$(jq -r --arg pkg "./${dir}" '
		.files[] as $f
		| $f.mutations[]
		| select(.status == "LIVED")
		| "  SURVIVOR LIVED \($f.file_name):\(.line):\(.column) (\(.type)) in \($pkg)"
	' "$report")"; then
		echo "  FAILED ./${dir} — jq cannot read gremlins' report."
		failed=$((failed + 1))
		continue
	fi
	if [ -n "$lived" ]; then
		printf '%s\n' "$lived"
		total_lived=$((total_lived + $(printf '%s\n' "$lived" | grep -c .)))
	fi
done

report_test_only

echo
if [ "$failed" -gt 0 ]; then
	echo "mutate-diff: WARNING ${failed} gremlins run(s) FAILED (above) — those packages were not tested."
fi
if [ "$total_lived" -gt 0 ]; then
	echo "mutate-diff: ${total_lived} surviving mutant(s) (LIVED) — ADVISORY."
	echo "  Triage per wf-vacuity §1: strengthen the assertion that should kill each, or"
	echo "  confirm it is an equivalent/unreachable mutant (expected noise — do not chase)."
elif [ "$failed" -gt 0 ]; then
	echo "mutate-diff: no surviving mutants (LIVED) reported, but the run is incomplete — see the WARNING above."
else
	echo "mutate-diff: no surviving mutants (LIVED) on changed internal/ lines — advisory pass."
fi
exit 0
