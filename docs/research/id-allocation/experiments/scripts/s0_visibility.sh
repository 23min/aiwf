#!/usr/bin/env bash
# S0.2 — what beta's `aiwf add gap` allocates, given what alpha has done with its G-0002
# and what beta has fetched. Each row is a fresh world.
source "$(dirname "$0")/lib.sh"
row() { # row <label> <alpha-action> <beta-action>
  setup "$X/runs/s0vis-$1"; mkclone alpha; mkclone beta
  ( cd "$W/alpha"; git checkout -q -b feature/A; addgap "Alpha gap" >/dev/null
    case $2 in push|push-then-delete) git push -q --no-verify origin feature/A;; esac
    [ "$2" = push-then-delete ] || true )
  ( cd "$W/beta"; git checkout -q -b feature/B
    case $3 in
      none) ;;
      fetch) git fetch -q origin ;;
      fetch-prune-after-delete) git fetch -q origin; git -C "$W/alpha" push -q --no-verify origin --delete feature/A; git fetch -q --prune origin ;;
    esac
    if [ "$3" = add-fetch ]; then id=$(addgap "Beta gap" --fetch); else id=$(addgap "Beta gap"); fi
    printf '  %-28s alpha:%-17s beta:%-26s -> beta got %s\n' "$1" "$2" "$3" "$id" )
}
row unpushed          commit-only       none
row pushed-nofetch    push              none
row pushed-fetch      push              fetch
row pushed-addfetch   push              add-fetch
row pushed-pruned     push-then-delete  fetch-prune-after-delete
# Same clone, sibling worktree: alpha's branch is a local ref of beta's repository.
setup "$X/runs/s0vis-worktree"; mkclone alpha
( cd "$W/alpha"; git checkout -q -b feature/A; addgap "Alpha gap" >/dev/null
  git worktree add -q -b feature/B "$W/alpha-wt" main
  cd "$W/alpha-wt"; id=$(addgap "Beta gap")
  printf '  %-28s alpha:%-17s beta:%-26s -> beta got %s\n' worktree commit-only same-clone-worktree "$id" )
