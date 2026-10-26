#!/usr/bin/env bash
# S0.4 — the two colliding gaps get the SAME path (same id, same title).
#   identical  same title and same body: the two files are byte-identical
#   differ     same title, different body
source "$(dirname "$0")/lib.sh"
for variant in identical differ; do
  echo "######## $variant"
  setup "$X/runs/s0same-$variant"; mkclone alpha hooks; mkclone beta hooks
  ( cd "$W/alpha"; git checkout -q -b feature/A; id=$(addgap "Login fails")
    cite G-0001 "See $id (means:alpha-login)."; echo "  alpha: $id"; git push -q origin feature/A 2>/dev/null )
  ( cd "$W/beta"; git checkout -q -b feature/B
    if [ $variant = identical ]; then id=$(addgap "Login fails")
    else id=$(aiwf add gap --title "Login fails" --body $'## What\'s missing\n\nbeta.\n\n## Why it matters\n\nbeta.\n' --format=json </dev/null | jq -r .metadata.entity_id); fi
    cite M-0001 "Blocked on $id (means:beta-login)."; echo "  beta:  $id" )
  hostclone; ( cd "$W/hostwork"; git merge -q --no-ff --no-edit origin/feature/A; git push -q --no-verify origin main )
  cd "$W/beta"; git fetch -q origin
  echo "== beta's pre-push after A has landed on main and beta fetched"
  show git push -q origin feature/B | grep -E 'error|findings|exit' | cut -c1-170
  echo "  blob A: $(git rev-parse origin/feature/A:work/gaps/G-0002-login-fails.md)  blob B: $(git rev-parse HEAD:work/gaps/G-0002-login-fails.md)"
  echo "== host merges feature/B"
  cd "$W/hostwork"; git fetch -q origin
  show git merge --no-ff --no-edit origin/feature/B | grep -Ei 'conflict|automatic|exit|Merge made'
  if git diff --quiet --cached && [ -z "$(git ls-files -u)" ]; then
    git push -q --no-verify origin main
    rm -rf "$W/after"; git clone -q "$W/host.git" "$W/after"; cd "$W/after"
    echo "  gaps on main: $(ls work/gaps | tr '\n' ' ')"
    show aiwf check | grep -E 'error|findings|exit' | cut -c1-170
    verify
    show aiwf history G-0002
  fi
done
