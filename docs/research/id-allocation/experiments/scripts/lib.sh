# Shared helpers for the id-allocation experiments. Sourced, not executed.
# Adapted from spike/scripts/lib.sh (same seed shape: E-0001, M-0001, G-0001 on main).
set -uo pipefail
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
X=${IDEXP_WORK:-${TMPDIR:-/tmp}/aiwf-idexp}   # out/ and runs/ live here, outside the repository
mkdir -p "$X/out" "$X/runs"
command -v aiwf >/dev/null || { echo "lib.sh: put the aiwf binary under test on PATH (the hooks call it too)" >&2; exit 1; }
export GIT_CONFIG_NOSYSTEM=1
export GIT_TERMINAL_PROMPT=0

die()  { echo "FAILED: $*" >&2; exit 1; }
must() { "$@" </dev/null || die "$*"; }
# show <cmd...>: print the command, run it with stdin closed, indent its output, keep its exit code.
show() { echo "\$ $*"; "$@" </dev/null 2>&1 | sed 's/^/    /'; local rc=${PIPESTATUS[0]}; echo "    [exit $rc]"; return $rc; }

GB=$'## What\'s missing\n\nx.\n\n## Why it matters\n\ny.\n'
EB=$'## Goal\n\ng.\n\n## Scope\n\ns.\n\n## Out of scope\n\no.\n'
MB=$'## Goal\n\ng.\n\n## Acceptance criteria\n'

# setup <W>: seed repo W/seed (aiwf init + E-0001, M-0001, G-0001, hooks installed),
# bare host W/host.git with main pushed.
setup() {
  W=$1; rm -rf "$W"; mkdir -p "$W/seed"
  ( cd "$W/seed"
    git init -q -b main .
    git config user.name seed; git config user.email seed@example.com
    printf 'hosts: []\nguidance:\n  enabled: false\n' > aiwf.yaml
    aiwf init </dev/null >/dev/null 2>&1 || die "aiwf init"
    git add -A; git commit -q -m "chore: initial project with aiwf"
    must aiwf add epic --title "Seed epic" --body "$EB" >/dev/null
    must aiwf add milestone --epic E-0001 --tdd none --title "Seed ms" --body "$MB" >/dev/null
    must aiwf add gap --title "Seed gap" --body "$GB" >/dev/null )
  git init -q --bare -b main "$W/host.git"
  git -C "$W/seed" push -q --no-verify "$W/host.git" main
}

# mkclone <name> [hooks]: clone the host into W/<name> with that git identity;
# with "hooks", copy the seed's aiwf hooks in (clones carry none).
mkclone() {
  local n=$1
  git clone -q "$W/host.git" "$W/$n"
  git -C "$W/$n" config user.name "$n"; git -C "$W/$n" config user.email "$n@example.com"
  if [ "${2:-}" = hooks ]; then
    for h in pre-push pre-commit commit-msg post-commit; do cp "$W/seed/.git/hooks/$h" "$W/$n/.git/hooks/$h"; done
  fi
}

# addgap <title> [extra aiwf flags...]: add a gap in the cwd, print the allocated id.
addgap() { local t=$1; shift; aiwf add gap --title "$t" --body "$GB" "$@" --format=json </dev/null | jq -r .metadata.entity_id; }
addepic() { aiwf add epic --title "$1" --body "$EB" --format=json </dev/null | jq -r .metadata.entity_id; }
addms() { aiwf add milestone --epic "$1" --tdd none --title "$2" --body "$MB" --format=json </dev/null | jq -r .metadata.entity_id; }

# entity_file <id>: path of the (first) entity file carrying <id> in the cwd.
entity_file() {
  { ls work/gaps/$1-*.md work/decisions/$1-*.md work/epics/*/$1-*.md work/epics/$1-*/epic.md docs/adr/$1-*.md 2>/dev/null || true; } | head -1
}

# cite <id> <sentence>: append a sentence to <id>'s body via aiwf edit-body.
cite() {
  local f tmp; f=$(entity_file "$1"); [ -n "$f" ] || die "cite: no file for $1"
  tmp=$(mktemp "$X/out/cite.XXXXXX")
  awk '/^---$/{n++; next} n>=2' "$f" > "$tmp"; printf '\n%s\n' "$2" >> "$tmp"
  aiwf edit-body "$1" --body-file "$tmp" </dev/null >/dev/null || die "cite $1"
  rm -f "$tmp"
}

# verify [dir]: resolve every "(means:<slug>)" citation and "(<field> means <slug>)" title marker.
verify() { python3 -I "$HERE/verify.py" "${1:-.}"; }

# hostwork: a host-side clone used to perform merges; recreated per call.
hostclone() {
  rm -rf "$W/hostwork"; git clone -q "$W/host.git" "$W/hostwork"
  git -C "$W/hostwork" config user.name host; git -C "$W/hostwork" config user.email host@example.com
}
