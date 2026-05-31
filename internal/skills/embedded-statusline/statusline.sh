#!/usr/bin/env bash
# Status line for Claude Code in the aiwf repo.
# Reads JSON from stdin (Claude Code's session context), prints one line.
#
# Layout: <ball> <model> <effort?> ▸ <tokens> · <epic HUD> · <repo> · <branch…>[<dirty>][<sync>] · <ci-glyph> ci
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
gray=$'\033[90m'
bold=$'\033[1m'
reset=$'\033[0m'
ball="${color}●${reset}"
tokens_fmt="${color}$(fmt_tokens "$tokens")${reset}"

effort="$(jq_get '.effort.level')"

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

  # Truncate long branch names: keep first 30 chars, append …
  if [ "${#branch_seg}" -gt 30 ]; then
    branch_seg="${branch_seg:0:30}…"
  fi

  # Dirty marker: ✎ (pencil) for uncommitted changes, prefixed before branch.
  if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
    branch_seg="✎ ${branch_seg}"
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

# --- In-flight epic HUD (G-0188) --------------------------------------------
#
# Scans work/epics/*/epic.md for non-terminal epics and renders each with a
# canonical status glyph (→ active, ○ proposed/draft) and color. Shows on
# every branch — not just ritual branches.
#
# On ritual branches, the current epic is accentuated (bold + ▸ pointer)
# with its milestone shown inline. Other in-flight epics render alongside
# but visually secondary.
#
# Capped at 3 epics shown; +N for overflow.

status_glyph() {
  case "$1" in
    active|in_progress) printf '→' ;;
    proposed|draft|open) printf '○' ;;
    done|addressed|accepted|met|retired) printf '✓' ;;
    cancelled|wontfix|rejected|superseded) printf '✗' ;;
    *) printf '?' ;;
  esac
}

is_terminal() {
  case "$1" in
    done|addressed|accepted|met|retired|cancelled|wontfix|rejected|superseded) return 0 ;;
    *) return 1 ;;
  esac
}

# Derive the current-branch context (epic id, milestone id).
ctx_epic_id=""
ctx_milestone_id=""
case "$cur_branch" in
  milestone/M-*)
    ctx_milestone_id="$(printf '%s' "$cur_branch" | sed -E 's|^milestone/(M-[0-9]+).*|\1|')"
    m_file="$(git ls-files "work/epics/*/${ctx_milestone_id}-*.md" "work/epics/archive/*/${ctx_milestone_id}-*.md" 2>/dev/null | head -1)"
    if [ -n "$m_file" ]; then
      ctx_epic_id="$(printf '%s' "$m_file" | sed -E 's|^work/epics/(archive/)?(E-[0-9]+).*|\2|')"
    fi
    ;;
  epic/E-*)
    ctx_epic_id="$(printf '%s' "$cur_branch" | sed -E 's|^epic/(E-[0-9]+).*|\1|')"
    ;;
esac

# Collect all non-terminal epics.
epic_hud_parts=()
epic_hud_count=0
epic_hud_cap=3
epic_hud_total=0
for epic_file in work/epics/E-*/epic.md; do
  [ -f "$epic_file" ] || continue
  e_status="$(read_status "$epic_file")"
  is_terminal "$e_status" && continue
  e_id="$(printf '%s' "$epic_file" | sed -E 's|^work/epics/(E-[0-9]+).*|\1|')"
  [ -z "$e_id" ] && continue
  epic_hud_total=$((epic_hud_total + 1))
  [ "$epic_hud_count" -ge "$epic_hud_cap" ] && continue

  e_clr="$(entity_color "$e_status")"
  e_glyph="$(status_glyph "$e_status")"

  if [ "$e_id" = "$ctx_epic_id" ]; then
    # Current epic: bold + ▸ pointer. Append milestone inline.
    entry="${bold}${e_clr}▸ ${e_glyph} ${e_id}${reset}"
    if [ -n "$ctx_milestone_id" ]; then
      m_file="$(git ls-files "work/epics/*/${ctx_milestone_id}-*.md" "work/epics/archive/*/${ctx_milestone_id}-*.md" 2>/dev/null | head -1)"
      m_status=""
      [ -n "$m_file" ] && m_status="$(read_status "$m_file")"
      m_clr="$(entity_color "$m_status")"
      m_glyph="$(status_glyph "$m_status")"
      entry="${entry}${gray}/${reset}${m_clr}${m_glyph} ${ctx_milestone_id}${reset}"
    fi
    epic_hud_parts=("$entry" "${epic_hud_parts[@]}")
  else
    entry="${e_clr}${e_glyph} ${e_id}${reset}"
    epic_hud_parts+=("$entry")
  fi
  epic_hud_count=$((epic_hud_count + 1))
done

# Build the epic HUD segment: join parts with space, add overflow.
epic_hud=""
for p in "${epic_hud_parts[@]+"${epic_hud_parts[@]}"}"; do
  if [ -z "$epic_hud" ]; then epic_hud="$p"; else epic_hud="$epic_hud $p"; fi
done
overflow=$((epic_hud_total - epic_hud_count))
[ "$overflow" -gt 0 ] && epic_hud="${epic_hud} ${gray}+${overflow}${reset}"

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
    local b="$1" out conc status
    out="$(gh run list --branch "$b" --limit 1 --json conclusion,status 2>/dev/null)" || return 1
    [ -z "$out" ] || [ "$out" = "[]" ] && return 1
    conc="$(printf '%s' "$out" | jq -r '.[0].conclusion // empty' 2>/dev/null)"
    status="$(printf '%s' "$out" | jq -r '.[0].status // empty' 2>/dev/null)"
    if [ "$status" = "in_progress" ] || [ "$status" = "queued" ] || [ "$status" = "requested" ] || [ "$status" = "waiting" ]; then
      printf '→'
    else
      case "$conc" in
        success)  printf '✓'    ;;
        failure|cancelled|timed_out|action_required|startup_failure) printf '✗' ;;
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

# CI glyph precedes the label: ✓ ci / ✗ ci / → ci / ? ci.
# Color matches the glyph's meaning: green ✓, red ✗, yellow →.
# Main-fallback prefix surfaces as ✓ m:ci.
ci_label="ci"
[ -n "$ci_prefix" ] && ci_label="${ci_prefix}${ci_label}"
case "$ci_state" in
  '✓') ci_fmt="${green}${ci_state} ${ci_label}${reset}" ;;
  '✗') ci_fmt="${red}${ci_state} ${ci_label}${reset}" ;;
  '→') ci_fmt="${yellow}${ci_state} ${ci_label}${reset}" ;;
  *)   ci_fmt="${ci_state} ${ci_label}" ;;
esac

# --- Compose ----------------------------------------------------------------

head_seg="$ball $model_short"
[ -n "$effort" ] && head_seg="$head_seg ${gray}${effort}${reset}"
head_seg="$head_seg ▸ $tokens_fmt"
parts=("$head_seg")
[ -n "$epic_hud" ]      && parts+=("$epic_hud")
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
