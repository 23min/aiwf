#!/usr/bin/env bash
# Status line for Claude Code in the ai-workflow-v2 repo.
# Reads JSON from stdin (Claude Code's session context), prints one line.
#
# Layout: <ball> <model> · <repo> · <branch>[<dirty>][<sync>] · <epic?> · <milestone?> · ci:<state> · <tokens>
#
# All segments fail soft: anything that errors collapses to "?" or is dropped.
# Network calls are cached in /tmp with a TTL so the script stays sub-100ms.

set -u

# Read everything from stdin once (Claude Code passes session JSON).
input="$(cat 2>/dev/null || true)"

jq_get() {
  # $1 = jq filter; prints empty string on any failure.
  printf '%s' "$input" | jq -r "$1 // empty" 2>/dev/null
}

# --- Model + context window -------------------------------------------------

model="$(jq_get '.model.display_name')"
[ -z "$model" ] && model="$(jq_get '.model.id')"
[ -z "$model" ] && model="?"

# Detect 1M-context variant from the display name.
case "$model" in
  *"1M"*|*"1m"*) ctx_max=1000000 ;;
  *)             ctx_max=200000  ;;
esac

# --- Token usage from transcript -------------------------------------------

transcript="$(jq_get '.transcript_path')"
tokens=0
if [ -n "$transcript" ] && [ -r "$transcript" ]; then
  # Walk transcript bottom-up; first assistant message with usage wins.
  tokens="$(tac "$transcript" 2>/dev/null \
    | jq -r 'select(.message.usage != null)
             | .message.usage
             | (.input_tokens // 0)
             + (.cache_read_input_tokens // 0)
             + (.cache_creation_input_tokens // 0)' 2>/dev/null \
    | head -n 1)"
  [ -z "$tokens" ] && tokens=0
fi

# Format token count: 116k, 1.2M, 950 etc.
fmt_tokens() {
  local t="$1"
  if [ "$t" -ge 1000000 ]; then
    awk -v t="$t" 'BEGIN { printf "%.1fM", t/1000000 }'
  elif [ "$t" -ge 1000 ]; then
    awk -v t="$t" 'BEGIN { printf "%.0fk", t/1000 }'
  else
    printf '%s' "$t"
  fi
}
# Color thresholds — same scale for ball and token text.
# green <50%, yellow <80%, red >=80% (start a new session soon).
pct=$(( tokens * 100 / ctx_max ))
if   [ "$pct" -lt 50 ]; then color=$'\033[32m'   # green
elif [ "$pct" -lt 80 ]; then color=$'\033[33m'   # yellow
else                         color=$'\033[31m'   # red
fi
red=$'\033[31m'
blue=$'\033[34m'
reset=$'\033[0m'
ball="${color}●${reset}"
tokens_fmt="${color}$(fmt_tokens "$tokens") tokens${reset}"

# --- Repo name --------------------------------------------------------------

repo="$(git rev-parse --show-toplevel 2>/dev/null | xargs -I{} basename {} 2>/dev/null)"
[ -z "$repo" ] && repo="$(jq_get '.workspace.current_dir' | xargs -I{} basename {} 2>/dev/null)"
[ -z "$repo" ] && repo="?"

# --- Branch + dirty + sync --------------------------------------------------

branch_seg=""
cur_branch=""
if git rev-parse --git-dir >/dev/null 2>&1; then
  if br="$(git symbolic-ref --short HEAD 2>/dev/null)"; then
    branch_seg="$br"
    cur_branch="$br"
  else
    sha="$(git rev-parse --short HEAD 2>/dev/null)"
    branch_seg="@${sha:-?}"
  fi

  # Dirty marker.
  if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
    branch_seg="${branch_seg}*"
  fi

  # Sync state vs upstream.
  if up="$(git rev-parse --abbrev-ref --symbolic-full-name @{u} 2>/dev/null)" && [ -n "$up" ]; then
    counts="$(git rev-list --left-right --count HEAD..."$up" 2>/dev/null)"
    ahead="${counts%%	*}"
    behind="${counts##*	}"
    sync=""
    [ "${ahead:-0}" -gt 0 ] 2>/dev/null && sync="${sync}↑${ahead}"
    [ "${behind:-0}" -gt 0 ] 2>/dev/null && sync="${sync}↓${behind}"
    branch_seg="${branch_seg}${sync}"
  fi
fi

# --- Epic + milestone (derived from branch + file layout) ------------------
#
# When on a `milestone/M-NNN-<slug>` branch, show both the milestone id and
# its parent epic id. When on an `epic/E-NN-<slug>` branch, show only the
# epic id. On main and other branches, drop both segments — the kernel's
# branch policy ties in-flight work to ritual branches, so there's nothing
# meaningful to derive when we're not on one.

epic_seg=""
milestone_seg=""
case "$cur_branch" in
  milestone/M-*)
    m_id="$(printf '%s' "$cur_branch" | sed -E 's|^milestone/(M-[0-9]+).*|\1|')"
    if [ -n "$m_id" ]; then
      milestone_seg="$m_id"
      # Find parent epic by locating the milestone spec under work/epics/.
      m_file="$(git ls-files "work/epics/*/${m_id}-*.md" 2>/dev/null | head -1)"
      if [ -n "$m_file" ]; then
        e_id="$(printf '%s' "$m_file" | sed -E 's|^work/epics/(E-[0-9]+).*|\1|')"
        [ -n "$e_id" ] && epic_seg="$e_id"
      fi
    fi
    ;;
  epic/E-*)
    e_id="$(printf '%s' "$cur_branch" | sed -E 's|^epic/(E-[0-9]+).*|\1|')"
    [ -n "$e_id" ] && epic_seg="$e_id"
    ;;
esac

# --- CI status (cached) -----------------------------------------------------

ci_state="?"
ci_prefix=""
if command -v gh >/dev/null 2>&1 && [ -n "$branch_seg" ]; then
  ci_branch="${cur_branch:-HEAD}"
  cache_key="$(printf '%s/%s' "$(git rev-parse --show-toplevel 2>/dev/null)" "$ci_branch" | shasum | awk '{print $1}')"
  cache_file="/tmp/aiwf-statusline-ci-${cache_key}"
  ttl=45

  fetch_ci() {
    # Returns "<source>:<state>" where source is "" or "m" (main fallback).
    local b="$1" out conc status
    out="$(gh run list --branch "$b" --limit 1 --json conclusion,status 2>/dev/null)" || return 1
    [ -z "$out" ] || [ "$out" = "[]" ] && return 1
    conc="$(printf '%s' "$out" | jq -r '.[0].conclusion // empty' 2>/dev/null)"
    status="$(printf '%s' "$out" | jq -r '.[0].status // empty' 2>/dev/null)"
    if [ "$status" = "in_progress" ] || [ "$status" = "queued" ] || [ "$status" = "requested" ] || [ "$status" = "waiting" ]; then
      printf 'run'
    else
      case "$conc" in
        success)  printf 'ok'   ;;
        failure|cancelled|timed_out|action_required|startup_failure) printf 'fail' ;;
        *)        printf '?'    ;;
      esac
    fi
  }

  use_cache=false
  if [ -r "$cache_file" ]; then
    age=$(( $(date +%s) - $(stat -f %m "$cache_file" 2>/dev/null || stat -c %Y "$cache_file" 2>/dev/null || echo 0) ))
    [ "$age" -lt "$ttl" ] && use_cache=true
  fi

  if $use_cache; then
    cached="$(cat "$cache_file" 2>/dev/null)"
    ci_prefix="${cached%%|*}"
    ci_state="${cached##*|}"
  else
    s="$(fetch_ci "$ci_branch")"
    if [ -z "$s" ] || [ "$s" = "?" ]; then
      # Fall back to main when the current branch has no runs.
      if [ "$ci_branch" != "main" ]; then
        s="$(fetch_ci main)"
        [ -n "$s" ] && [ "$s" != "?" ] && ci_prefix="m:"
      fi
    fi
    [ -z "$s" ] && s="?"
    ci_state="$s"
    tmp="${cache_file}.tmp.$$"
    printf '%s|%s' "$ci_prefix" "$ci_state" >"$tmp" 2>/dev/null && mv -f "$tmp" "$cache_file" 2>/dev/null
  fi
fi

# Color the ci segment red when it's failed; leave other states neutral.
ci_text="ci:${ci_prefix}${ci_state}"
if [ "$ci_state" = "fail" ]; then
  ci_fmt="${red}${ci_text}${reset}"
else
  ci_fmt="$ci_text"
fi

# --- Compose ----------------------------------------------------------------

parts=("$ball $model" "$repo")
[ -n "$branch_seg" ]    && parts+=("$branch_seg")
[ -n "$epic_seg" ]      && parts+=("${blue}${epic_seg}${reset}")
[ -n "$milestone_seg" ] && parts+=("${blue}${milestone_seg}${reset}")
parts+=("$ci_fmt" "$tokens_fmt")

# Join with " · ".
out=""
for p in "${parts[@]}"; do
  if [ -z "$out" ]; then out="$p"; else out="$out · $p"; fi
done
printf '%s' "$out"
