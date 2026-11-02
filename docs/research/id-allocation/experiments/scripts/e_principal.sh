#!/usr/bin/env bash
# Design E — a mint-style commit with a non-human aiwf-actor, with and without aiwf-principal.
source "$(dirname "$0")/lib.sh"
for variant in no-principal with-principal; do
  echo "######## $variant"
  setup "$X/runs/e-principal-$variant"; mkclone bot; cd "$W/bot"
  mkdir -p work/gaps; printf -- '---\nid: G-0002\ntitle: minted\nstatus: open\n---\n%s' "$GB" > work/gaps/G-0002-minted.md
  p=""; [ $variant = with-principal ] && p=$'\naiwf-principal: human/merger'
  printf 'aiwf assign-pending: minted\n\naiwf-verb: assign-pending\naiwf-entity: G-0002\naiwf-actor: framework/aiwf-mint%s\n' "$p" > "$W/msg"
  git add -A; git commit -q --no-verify -F "$W/msg"; git push -q --no-verify origin main
  rm -rf "$W/after"; git clone -q "$W/host.git" "$W/after"; cd "$W/after"
  show aiwf check | grep -E 'error|findings|exit' | cut -c1-200
done
