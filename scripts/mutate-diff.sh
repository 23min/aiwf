#!/usr/bin/env bash
# mutate-diff — diff-scoped mutation testing (advisory, G-0267).
#
# Runs gremlins on just the internal/ Go lines changed since the
# merge-base with origin/main — committed, staged, unstaged, or in an
# untracked new file — instead of the whole kernel. It is the
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
# unit-mutation-tested. A package whose changed lines are all in its
# tests has no code line to mutate; it is named, with the whole-package
# command that asks whether the new tests kill anything the old ones
# missed.
#
# How gremlins is scoped. `gremlins unleash --diff <ref>` filters by
# the output of `git diff --merge-base <ref>`, run where gremlins was
# started, and names the files it walks relative to the directory it is
# pointed at; it reads each hunk's added lines as one run starting at
# the hunk's first change. So each run starts inside its directory, and
# a git shim ahead of gremlins on PATH answers that one diff call with
# paths relative to it, zero context lines (every change its own hunk),
# plain output, and — when untracked files exist — a copy of the index
# holding them as intent-to-add entries. Every other git call, the
# test suites' included, passes through untouched, and the operator's
# index is never written. Where gremlins' filter and git's diff still
# disagree, each mutant is named (SKIPPED-CHANGED: a changed line left
# untested; MUTATED-UNCHANGED: an unchanged line mutated), and a run
# gremlins exits non-zero from, or leaves no readable report, is named
# FAILED; either one withholds the advisory pass. A run whose directory
# holds nothing gremlins can mutate ends with no report at exit 0 and
# is named NO MUTANTS.
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

root="$(git rev-parse --show-toplevel)"
cd "$root"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

index=""
untracked="$(git ls-files --others --exclude-standard -- 'internal/*.go')"
if [ -n "$untracked" ]; then
	index="$tmpdir/index"
	# -p keeps the index's timestamp: git re-reads a same-size file
	# edited in the index's own second only while the index is no newer
	# than that edit, and a fresh timestamp would hide it.
	cp -p "$(git rev-parse --git-path index)" "$index"
	# Written whole: a split copy would leave a new shared-index file in
	# the operator's git directory.
	printf '%s\n' "$untracked" | GIT_INDEX_FILE="$index" git -c core.splitIndex=false add --intent-to-add --pathspec-from-file=-
fi

# The diff as both this script and gremlins read it: fixed output
# options whatever the caller's git config or GIT_DIFF_OPTS says (both
# readers unset it), and no index refresh,
# so reading it never writes the operator's index. Renames are off
# because gremlins' view is relative to one directory: a file moved in
# from outside it can only read as new there, so it reads as new here
# too.
diff_opts="-c core.quotePath=false -c diff.autoRefreshIndex=false diff -U0 --inter-hunk-context=0 --no-renames --no-color --no-ext-diff --src-prefix=a/ --dst-prefix=b/"

# gitread runs git over the index copy when there is one.
gitread() {
	(
		unset GIT_DIFF_OPTS
		if [ -n "$index" ]; then
			GIT_INDEX_FILE="$index" git "$@"
		else
			git "$@"
		fi
	)
}

# path:line for every added line in internal/ Go files, repo-relative.
# shellcheck disable=SC2086 # diff_opts is a fixed word list
gitread $diff_opts --no-relative "$base" -- 'internal/*.go' | awk '
	/^\+\+\+ b\// { file = substr($0, 7); sub(/\t$/, "", file); next }
	/^@@ / {
		split($3, span, ",")
		start = substr(span[1], 2) + 0
		count = (2 in span) ? span[2] + 0 : 1
		for (i = 0; i < count; i++) print file ":" (start + i)
	}' >"$tmpdir/changed-lines"

# dirs_of code|test lists the directories holding changed lines of
# non-test or test Go files.
dirs_of() {
	awk -v want="$1" '{
		sub(/:[0-9]+$/, "")
		if ((want == "test") != ($0 ~ /_test\.go$/)) next
		sub(/\/[^\/]*$/, "")
		print
	}' "$tmpdir/changed-lines" | LC_ALL=C sort -u
}
code_dirs="$(dirs_of code)"
test_only="$(dirs_of test | grep -vxF -f <(printf '%s\n' "$code_dirs") | grep . || true)"

# One gremlins run per top-most changed directory: a run walks its
# subdirectories too, so a changed child rides its changed parent's run
# and coverage pass rather than repeating both.
runs="$(printf '%s\n' "$code_dirs" | awk 'NF {
	for (i = 1; i <= n; i++) if (index($0, kept[i] "/") == 1) next
	kept[++n] = $0
	print
}')"

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

mkdir "$tmpdir/bin"
index_env=""
if [ -n "$index" ]; then
	index_env="GIT_INDEX_FILE=$(printf '%q' "$index")"
fi
realgit="$(printf '%q' "$(command -v git)")"
# Absolute paths throughout: tests run git under a PATH of their own,
# which may hold neither bash nor env.
cat >"$tmpdir/bin/git" <<EOF
#!$(command -v bash)
if [ "\$#" -eq 3 ] && [ "\$1" = diff ] && [ "\$2" = --merge-base ]; then
	unset GIT_DIFF_OPTS
	${index_env} exec ${realgit} ${diff_opts} --relative --merge-base "\$3"
fi
exec ${realgit} "\$@"
EOF
chmod +x "$tmpdir/bin/git"

nruns="$(printf '%s\n' "$runs" | grep -c .)"
echo "mutate-diff: changed internal/ Go lines vs ${BASE_REF} (base ${base:0:12}), in ${nruns} run(s):"
printf '%s\n' "$runs" | sed 's#^#  ./#'
echo

total_lived=0
total_disagree=0
failed=0
nrun=0
for dir in $runs; do
	echo "── mutate-diff: gremlins ./${dir} ──"
	nrun=$((nrun + 1))
	report="${tmpdir}/report-${nrun}.json"
	echo 0 >"$tmpdir/rc"
	# --workers 1 / --timeout-coefficient: see the mutate-hunt workflow
	# header for the rationale (default workers time out on this repo).
	# SKIPPED lines are the unchanged code gremlins walks past; the
	# agreement check below reads them from the report instead.
	(
		cd "$dir"
		PATH="$tmpdir/bin:$PATH" gremlins unleash --diff "$base" --workers 1 --timeout-coefficient "$COEFF" \
			--output "$report" "${root}/${dir}" || echo "$?" >"$tmpdir/rc"
	) | grep -v '^ *SKIPPED ' || true

	rc="$(cat "$tmpdir/rc")"
	if ! jq -e 'type == "object"' "$report" >/dev/null 2>&1; then
		if [ "$rc" = 0 ] && [ ! -e "$report" ]; then
			echo "  NO MUTANTS ./${dir} — gremlins found nothing to mutate."
		else
			echo "  FAILED ./${dir} — gremlins exited ${rc} and left no readable report; nothing in this run was tested."
			failed=$((failed + 1))
		fi
		continue
	fi
	if [ "$rc" != 0 ]; then
		echo "  FAILED ./${dir} — gremlins exited ${rc}; its report is read below."
		failed=$((failed + 1))
	fi

	# Self-authored, stable survivor markers from the JSON — decoupled
	# from gremlins' stdout wording. LIVED = covered-but-survived (the
	# vacuity signal this tool exists to surface). NOT_COVERED overlaps
	# the diff-scoped coverage gate (G-0067), so it is left to gremlins'
	# own output above and not re-tallied here.
	lived="$(jq -r --arg pkg "./${dir}" '
		.files[]? as $f
		| $f.mutations[]?
		| select(.status == "LIVED")
		| "  SURVIVOR LIVED \($f.file_name):\(.line):\(.column) (\(.type)) in \($pkg)"
	' "$report")"
	if [ -n "$lived" ]; then
		printf '%s\n' "$lived"
		total_lived=$((total_lived + $(printf '%s\n' "$lived" | grep -c '^  SURVIVOR ')))
	fi

	disagree="$(jq -r --arg dir "$dir" '
		.files[]? as $f
		| $f.mutations[]?
		| "\($dir)/\($f.file_name):\(.line)\t\(.status)\t\($f.file_name):\(.line):\(.column) (\(.type)) in ./\($dir)"
	' "$report" | awk -F '\t' -v changed="$tmpdir/changed-lines" '
		BEGIN { while ((getline line < changed) > 0) set[line] }
		$2 == "SKIPPED" && ($1 in set) { print "  SKIPPED-CHANGED " $3 }
		$2 != "SKIPPED" && !($1 in set) { print "  MUTATED-UNCHANGED " $3 }
	')"
	if [ -n "$disagree" ]; then
		printf '%s\n' "$disagree"
		total_disagree=$((total_disagree + $(printf '%s\n' "$disagree" | grep -c .)))
	fi
done

report_test_only

echo
if [ "$failed" -gt 0 ]; then
	echo "mutate-diff: WARNING ${failed} gremlins run(s) FAILED (above) — those packages were not tested."
fi
if [ "$total_disagree" -gt 0 ]; then
	echo "mutate-diff: WARNING gremlins' diff filter disagreed with git's on ${total_disagree} mutant(s) (SKIPPED-CHANGED / MUTATED-UNCHANGED above)."
fi
if [ "$total_lived" -gt 0 ]; then
	echo "mutate-diff: ${total_lived} surviving mutant(s) (LIVED) — ADVISORY."
	echo "  Triage per wf-vacuity §1: strengthen the assertion that should kill each, or"
	echo "  confirm it is an equivalent/unreachable mutant (expected noise — do not chase)."
elif [ "$failed" -gt 0 ] || [ "$total_disagree" -gt 0 ]; then
	echo "mutate-diff: no surviving mutants (LIVED) reported, but the run is incomplete — see the WARNING above."
else
	echo "mutate-diff: no surviving mutants (LIVED) on changed internal/ lines — advisory pass."
fi
exit 0
