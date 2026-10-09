#!/usr/bin/env bash
# run.sh <checkout-sha> <host claude|codex> <task> <n> [cont]
#
# One observation run of M-0334's rubric: builds the run directory, starts the
# host on the task's prompt as the committed rubric states it, and writes the
# run's records under $OBSERVE_DIR/<task>[-cont]/<host>-<n>/ (OBSERVE_DIR
# defaults to ${TMPDIR:-/tmp}/e0092-observe). PROMPT_FILE replaces the
# rubric's prompt, for a smoke run of the harness itself.
set -uo pipefail
SHA=$1 HOST=$2 TASK=$3 N=$4 CONT=${5:-}
H=$(dirname "$(readlink -f "$0")")
SRC=${AIWF_SRC:-$(git -C "$H" rev-parse --show-toplevel)}
export OBSERVE_DIR=${OBSERVE_DIR:-${TMPDIR:-/tmp}/e0092-observe}
TIMEOUT=1800
SPEC=$(git -C "$SRC" ls-tree -r --name-only "$SHA" | grep '/M-0334-')
RUN=$OBSERVE_DIR/$TASK${CONT:+-cont}/$HOST-$N
rm -rf "$RUN"; mkdir -p "$RUN"
git -C "$SRC" show "$SHA:$SPEC" > "$RUN/spec.md"
if [ -n "${PROMPT_FILE:-}" ]; then cp "$PROMPT_FILE" "$RUN/prompt.txt"; else python3 "$H/prompts.py" "$RUN/spec.md" "$TASK" > "$RUN/prompt.txt" || exit 1; fi
"$H/setup.sh" "$RUN" "$SHA" "$TASK" > "$RUN/setup.log" 2>&1 || { echo "$RUN setup failed"; exit 1; }
START=$(cat "$RUN/start-head")
# The run's path as it ran, so its records still read right once moved.
echo "$RUN" > "$RUN/run-path.txt"
# Each run gets its own short temporary directory. gpg-agent puts sockets
# under it and refuses a socket path past about 107 characters, so a deep
# observation directory cannot hold it.
RUNTMP=$(mktemp -d /tmp/e0092.XXXXXX)
[ "${#RUNTMP}" -le 44 ] || { echo "$RUN temporary directory too long: $RUNTMP"; exit 1; }
echo "$RUNTMP" > "$RUN/tmpdir.txt"
# A host must not update itself mid-round, so every run of a round runs one version.
export PATH="$RUN/bin:$PATH" GOBIN="$RUN/gobin" DISABLE_AUTOUPDATER=1 TMPDIR="$RUNTMP"
cd "$RUN/repo"
# Runs of one prompt, a task and its continuation alike, write the same
# names outside their clones, so they never run side by side.
exec 9> "$OBSERVE_DIR/.lock-$TASK"
flock 9
start=$(date +%s)
if [ "$HOST" = claude ]; then
  extra=(); [ -n "$CONT" ] && extra=(--autocompact 120000)
  timeout "$TIMEOUT" claude --print --output-format stream-json --verbose --model claude-opus-5-5 --effort high \
    --permission-mode bypassPermissions --tools Read,Glob,Grep,Edit,Write,Bash,Skill --allowedTools Read,Glob,Grep,Edit,Write,Bash,Skill \
    --strict-mcp-config --mcp-config '{"mcpServers":{}}' "${extra[@]}" < "$RUN/prompt.txt" > "$RUN/events.jsonl" 2> "$RUN/stderr.txt"
else
  extra=(); [ -n "$CONT" ] && extra=(-c model_auto_compact_token_limit=120000)
  CODEX_HOME="$RUN/codex-home" timeout "$TIMEOUT" codex exec --cd "$RUN/repo" --sandbox danger-full-access -c allow_login_shell=false "${extra[@]}" --json \
    -m gpt-6-astra -c model_reasoning_effort="medium" - < "$RUN/prompt.txt" > "$RUN/events.jsonl" 2> "$RUN/stderr.txt"
fi
rc=$?
end=$(date +%s)
flock -u 9
ROLLOUT=""
[ "$HOST" = codex ] && ROLLOUT=$(python3 -c "import sys; sys.path.insert(0, '$H'); from codexrollout import threads; print(threads('$RUN/codex-home')[0] or '')")
breaker=$(python3 -c "
import json
for l in open('$RUN/events.jsonl'):
    try: e=json.loads(l)
    except ValueError: continue
    if e.get('type')=='result' and e.get('terminal_reason')=='rapid_refill_breaker': print('yes')
")
{
  echo "host=$HOST task=$TASK run=$N continuation=${CONT:-no} exit=$rc seconds=$((end-start)) timeout=$TIMEOUT date=$(date -u +%FT%TZ)"
  [ "$rc" = 124 ] && echo "INCOMPLETE: timeout"
  [ -n "$breaker" ] && echo "INCOMPLETE: autocompact thrash breaker"
  echo "versions: $(claude --version 2>&1 | head -1) | $(codex --version 2>&1 | head -1)"
  echo "binary: $(head -1 "$RUN/binary.txt") $(tail -1 "$RUN/binary.txt" | cut -c1-16)"
  echo "checkout=$SHA start-head=$START start-origin=$(cat "$RUN/start-origin")"; cat "$RUN/index-commit.txt"
  echo "--- artifacts"; cat "$RUN/artifacts.sha256"
  echo "--- overlay: ~/.guidance HEAD $(git -C ~/.guidance rev-parse HEAD 2>/dev/null), dirty files: $(git --no-optional-locks -C ~/.guidance status --porcelain 2>/dev/null | wc -l)"
  for f in ~/.claude/rules/guidance/* ~/.codex/AGENTS.md ~/.claude/settings.json ~/.codex/config.toml; do [ -e "$f" ] && echo "$(sha256sum < "$f" | cut -c1-16) $f"; done
  for d in ~/.claude/skills ~/.codex/skills; do echo "$(find -L "$d" -type f -print0 2>/dev/null | sort -z | xargs -0 cat 2>/dev/null | sha256sum | cut -c1-16) $d"; done
  if [ "$HOST" = claude ]; then echo "--- plugins"; python3 -c "
import json
for l in open('$RUN/events.jsonl'):
    e=json.loads(l)
    if e.get('type')=='system' and e.get('subtype')=='init': print([p.get('name') for p in e.get('plugins',[])], 'skills:', e.get('skills')); break
"; fi
  if [ -n "$CONT" ]; then
    echo "--- compaction events"
    if [ "$HOST" = claude ]; then grep -c '"subtype":"compact_boundary"' "$RUN/events.jsonl"
    else echo "rollout=$ROLLOUT"; [ -n "$ROLLOUT" ] && grep -c '"type":"compacted"' "$ROLLOUT"; fi
  fi
  echo "--- branch at end: $(git branch --show-current)"
  echo "--- commits since start (all refs)"; git log --all --format='%h %D | %s%n%b' "^$START" "^$(cat "$RUN/start-origin")" 2>/dev/null
  echo "--- binary under test after the run"; sha256sum "$RUN/bin/aiwf-real" | cut -c1-64
  echo "--- files written under /tmp during the run, outside run directories"; find /tmp -xdev -newer "$RUN/.start" -type f 2>/dev/null | grep -v -F -e "$OBSERVE_DIR" -e "$RUNTMP" | grep -v -E '^/tmp/claude-[0-9]+/' | head -50
  echo "--- status"; git status --porcelain
  echo "--- worktrees"; git worktree list
  echo "--- origin refs"; git -C "$RUN/origin.git" for-each-ref --format='%(refname) %(objectname:short)'
  echo "--- gitignored settings"; for f in .claude/settings.local.json; do [ -e "$f" ] && { echo "== $f"; cat "$f"; }; done
  echo "--- run home and GOBIN"; find "$RUN/home" "$RUN/gobin" -type f
  echo "--- files changed outside the clone during run (host session and log stores excluded)"
  find ~/.claude ~/.codex ~/go/bin ~/.guidance -newer "$RUN/.start" -type f 2>/dev/null | grep -v -E '/projects/|/sessions/|/shell-snapshots/|/todos/|/statsig/|/logs?/|\.log$|\.sqlite(-wal|-shm)?$|history\.jsonl|/file-history/|/session-env/|models_cache\.json|/tmp/|/plugins/|\.claude\.json|/backups/|/\.git/'
} > "$RUN/after.txt" 2>&1
{
  git diff "$START"
  git ls-files --others --exclude-standard | while read -r f; do echo "=== untracked $f (first 20,000 bytes)"; head -c 20000 "$f"; done
  git worktree list --porcelain | awk '/^worktree /{print $2}' | tail -n +2 | while read -r w; do echo "=== worktree $w"; git -C "$w" diff "$START"; git -C "$w" ls-files --others --exclude-standard | while read -r f; do echo "=== untracked $w/$f (first 20,000 bytes)"; head -c 20000 "$w/$f"; done; done
} > "$RUN/diff.patch" 2>&1
git -C "$RUN/repo" show "$START:CLAUDE.md" > "$RUN/CLAUDE.at-start.md"
"$H/derive.sh" "$RUN"
echo "$RUN exit=$rc $((end-start))s"
