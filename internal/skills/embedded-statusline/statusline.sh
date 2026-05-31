#!/usr/bin/env bash
# Status line for Claude Code in the aiwf repo.
# Reads JSON from stdin (Claude Code's session context), prints one line.
#
# Layout: <ball> <model> · <epic?> · <milestone?> · <repo> · <branch>[<dirty>][<sync>] · ci:<state> · <tokens>
#
# All segments fail soft: anything that errors collapses to "?" or is dropped.
# Network calls are cached in /tmp with a TTL so the script stays sub-100ms.

set -u

# Skip the opportunistic `.git/index.lock` write that `git status` (and
# other "read-only" callers) take to refresh the stat-cache on the way
# through. The render only reads, so the write is pure cost — and a
# SIGKILLed render dying mid-rename would orphan the lock, blocking the
# next real `git commit` in the same repo until someone cleans it up.
# Equivalent to prefixing every git call with `--no-optional-locks`,
# but cheaper: one export, every child inherits. Available since git 2.15.
export GIT_OPTIONAL_LOCKS=0

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

# Compress "Opus 4.7 (1M context)" → "Opus 4.7". The "(1M context)" suffix
# is implicit in the high ctx_max threshold + colored ball; redundant now
# that the token count sits inline next to the model name.
model_short="${model% (*}"

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
  # tail -r is BSD/macOS; tac is GNU. Brace-group so both branches
  # feed jq — `|` binds tighter than `||`, so without the group a
  # successful `tail -r` would route its raw output to the command
  # substitution and bypass the jq filter entirely.
  tokens="$({ tail -r "$transcript" 2>/dev/null || tac "$transcript" 2>/dev/null; } \
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
yellow=$'\033[33m'
green=$'\033[32m'
reset=$'\033[0m'
ball="${color}●${reset}"
tokens_fmt="${color}$(fmt_tokens "$tokens")${reset}"

# entity_color maps an aiwf entity status to the four-color status palette:
#   blue   = proposed
#   yellow = in-flight / not-yet-terminal (draft/active/in_progress/open)
#   green  = terminal-success (done/addressed/accepted/met/retired)
#   red    = terminal-failure (cancelled/wontfix/rejected/deprecated/superseded)
# Unrecognized statuses return empty (id renders in the default terminal color).
entity_color() {
  case "$1" in
    proposed) printf '%s' "$blue" ;;
    draft|active|in_progress|open) printf '%s' "$yellow" ;;
    done|addressed|accepted|met|retired) printf '%s' "$green" ;;
    cancelled|wontfix|rejected|deprecated|superseded) printf '%s' "$red" ;;
  esac
}

# read_status pulls the `status:` value from an entity file's frontmatter.
# Unquoted values only (the aiwf writer emits unquoted statuses).
read_status() {
  awk '/^status:/{
    sub(/^status:[[:space:]]*/, "")
    sub(/[[:space:]]+$/, "")
    print
    exit
  }' "$1" 2>/dev/null
}

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
    # Default-IFS read splits on space or tab — survives editor / patch-tool
    # reflow that retabs the source. Empty $counts (no upstream output)
    # leaves both vars unset; the ${ahead:-0} / ${behind:-0} guards below
    # cover that case.
    read -r ahead behind <<<"$counts"
    sync=""
    [ "${ahead:-0}" -gt 0 ] 2>/dev/null && sync="${sync}↑${ahead}"
    [ "${behind:-0}" -gt 0 ] 2>/dev/null && sync="${sync}↓${behind}"
    branch_seg="${branch_seg}${sync}"
  fi
fi

# --- Epic + milestone + gap (derived from branch + file layout) ------------
#
# When on a `milestone/M-NNN-<slug>` branch, show both the milestone id and
# its parent epic id. When on an `epic/E-NN-<slug>` branch, show only the
# epic id. When on a `patch/g-NNN-<slug>` branch (wf-patch flow), show the
# gap id. On main and other branches, drop the entity segments — the kernel's
# branch policy ties in-flight work to ritual branches, so there's nothing
# meaningful to derive when we're not on one.
#
# Each id is color-coded by its current `status:` (entity_color above).
# Frontmatter is read from the worktree's own checked-out files via
# `git ls-files` — so the badge reflects the branch's view, matching the
# `aiwf status --worktrees` per-worktree-tree resolution.

epic_seg=""
milestone_seg=""
gap_seg=""
e_color=""
m_color=""
g_color=""
case "$cur_branch" in
  milestone/M-*)
    m_id="$(printf '%s' "$cur_branch" | sed -E 's|^milestone/(M-[0-9]+).*|\1|')"
    if [ -n "$m_id" ]; then
      milestone_seg="$m_id"
      m_file="$(git ls-files "work/epics/*/${m_id}-*.md" "work/epics/archive/*/${m_id}-*.md" 2>/dev/null | head -1)"
      if [ -n "$m_file" ]; then
        e_id="$(printf '%s' "$m_file" | sed -E 's|^work/epics/(archive/)?(E-[0-9]+).*|\2|')"
        if [ -n "$e_id" ]; then
          epic_seg="$e_id"
          e_file="$(git ls-files "work/epics/${e_id}-*/epic.md" "work/epics/archive/${e_id}-*/epic.md" 2>/dev/null | head -1)"
          [ -n "$e_file" ] && e_color="$(entity_color "$(read_status "$e_file")")"
        fi
        m_color="$(entity_color "$(read_status "$m_file")")"
      fi
    fi
    ;;
  epic/E-*)
    e_id="$(printf '%s' "$cur_branch" | sed -E 's|^epic/(E-[0-9]+).*|\1|')"
    if [ -n "$e_id" ]; then
      epic_seg="$e_id"
      e_file="$(git ls-files "work/epics/${e_id}-*/epic.md" "work/epics/archive/${e_id}-*/epic.md" 2>/dev/null | head -1)"
      [ -n "$e_file" ] && e_color="$(entity_color "$(read_status "$e_file")")"
    fi
    ;;
  patch/[Gg]-*)
    g_id="$(printf '%s' "$cur_branch" | sed -E 's|^patch/[Gg]-([0-9]+).*|G-\1|')"
    if [ -n "$g_id" ]; then
      gap_seg="$g_id"
      g_file="$(git ls-files "work/gaps/${g_id}-*.md" "work/gaps/archive/${g_id}-*.md" 2>/dev/null | head -1)"
      [ -n "$g_file" ] && g_color="$(entity_color "$(read_status "$g_file")")"
    fi
    ;;
esac

# --- Other in-flight ritual worktrees count (+N⎇) --------------------------
#
# Counts other worktrees on ritual branches (epic/E-*, milestone/M-*,
# patch/[Gg]-*). The current session's worktree is excluded — the segment
# answers "is parallel work happening?" not "what's in flight overall".
# Omitted entirely when zero so single-session repos see no chrome.

other_wt_count=0
cur_wt="$(git rev-parse --show-toplevel 2>/dev/null)"
if [ -n "$cur_wt" ]; then
  while IFS= read -r line; do
    case "$line" in
      "worktree "*) wt_path="${line#worktree }" ;;
      "branch refs/heads/"*)
        wt_branch="${line#branch refs/heads/}"
        case "$wt_branch" in
          epic/E-*|milestone/M-*|patch/[Gg]-*)
            [ "$wt_path" != "$cur_wt" ] && other_wt_count=$((other_wt_count + 1))
            ;;
        esac
        ;;
    esac
  done < <(git worktree list --porcelain 2>/dev/null)
fi

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
    age=$(( $(date +%s) - $(stat -c %Y "$cache_file" 2>/dev/null || stat -f %m "$cache_file" 2>/dev/null || echo 0) ))
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

parts=("$ball $model_short ▸ $tokens_fmt")
[ -n "$epic_seg" ]      && parts+=("${e_color}${epic_seg}${reset}")
[ -n "$milestone_seg" ] && parts+=("${m_color}${milestone_seg}${reset}")
[ -n "$gap_seg" ]       && parts+=("${g_color}${gap_seg}${reset}")
parts+=("$repo")
[ -n "$branch_seg" ]    && parts+=("$branch_seg")
[ "$other_wt_count" -gt 0 ] && parts+=("+${other_wt_count}⎇")
parts+=("$ci_fmt")

# Join with " · ".
out=""
for p in "${parts[@]}"; do
  if [ -z "$out" ]; then out="$p"; else out="$out · $p"; fi
done
printf '%s' "$out"
