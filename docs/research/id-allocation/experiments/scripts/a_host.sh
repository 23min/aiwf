# Design A host rule, sourced after lib.sh. Simulates a host that merges a merge request only if
#   (1) the branch contains the default branch's tip, and
#   (2) `aiwf check` passes on the branch tip,
# then merges with --no-ff and pushes main WITHOUT force (so a concurrent merge makes it fail).
# host_mr <branch> [worker]: prints MERGED / REFUSED-<reason>; returns 0 only on MERGED.
host_mr() {
  local b=$1 hw="$W/host-${2:-1}" out
  if [ ! -d "$hw" ]; then
    git clone -q "$W/host.git" "$hw"; git -C "$hw" config user.name host; git -C "$hw" config user.email host@example.com
  fi
  ( cd "$hw"
    git fetch -q --prune origin
    git checkout -q -B main origin/main
    if ! git merge-base --is-ancestor origin/main "origin/$b"; then echo "REFUSED-not-up-to-date"; exit 1; fi
    git checkout -q -B mr "origin/$b"
    if ! out=$(aiwf check </dev/null 2>&1); then
      echo "REFUSED-check-failed: $(echo "$out" | grep -oE 'error [a-z/-]+: id "[A-Z]+-[0-9]+"' | sort -u | tr '\n' ';')"; exit 1; fi
    git checkout -q main
    git merge -q --no-ff --no-edit "origin/$b" || { git merge --abort; echo "REFUSED-conflict"; exit 1; }
    if ! git push -q --no-verify origin main 2>/dev/null; then echo "REFUSED-main-moved"; exit 1; fi
    echo "MERGED" )
}
