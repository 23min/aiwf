#!/usr/bin/env bash
# Design B — lost replies.
#  L1: the push is applied but the client is killed before it hears so (host post-receive sleeps
#      once; the client's push runs under `timeout 1`). Recovery a = re-read the ledger and look for
#      my nonce; b = naive (any holder means "someone else", so reallocate).
#  L2: the push is rejected but the client believes it succeeded (it ignores the exit code).
#      Recovery a = naive (use the id); b = read back the ledger and check the holder's nonce.
source "$(dirname "$0")/lib.sh"; source "$(dirname "$0")/b_lib.sh"
summary() { # after both branches merge: what main and the ledger hold
  hostclone; ( cd "$W/hostwork"; for b in $(git branch -r | grep -oE 'origin/mr/[a-z]+'); do git merge -q --no-ff --no-edit "$b"; done; git push -q --no-verify origin main )
  rm -rf "$W/after"; git clone -q "$W/host.git" "$W/after"
  ( cd "$W/after"
    echo "  ledger claims: $(git ls-tree -r origin/aiwf/ids --name-only | xargs -n1 basename | tr '\n' ' ')"
    for f in $(git ls-tree -r origin/aiwf/ids --name-only); do echo "    $(basename "$f"): $(git cat-file -p "origin/aiwf/ids:$f" | tr '\n' ' ')"; done
    echo "  main gaps: $(ls work/gaps | tr '\n' ' ')"
    echo "  ids-unique errors on main: $(aiwf check </dev/null 2>&1 | grep -c 'error ids-unique')"
    verify | tail -3 )
}
for rec in nonce naive; do
  echo "######## L1-$rec: push applied, reply lost"
  setup "$X/runs/b-lost1-$rec"; mkclone alpha
  cat > "$W/host.git/hooks/post-receive" <<EOF
#!/bin/sh
if [ -f "$W/sleep-once" ]; then rm -f "$W/sleep-once"; sleep 3; fi
EOF
  chmod +x "$W/host.git/hooks/post-receive"; touch "$W/sleep-once"
  ( cd "$W/alpha"; git checkout -q -b mr/alpha; git fetch -q origin
    id=$(addgap "alpha gap"); nonce=alpha-1
    PUSH_PREFIX="timeout 1" claim_once "$id" "$nonce"; rc=$?
    echo "  claim of $id under timeout: client rc=$rc; host ledger now: $(git -C "$W/host.git" rev-parse -q --verify --short refs/heads/aiwf/ids || echo absent)"
    sleep 3   # let the killed receive-pack's hook finish
    [ $rec = naive ] && export CLAIM_NO_NONCE_CHECK=1
    tries=1; while :; do tries=$((tries+1)); claim_once "$id" "$nonce"; rc=$?; [ $rc = 0 ] && break
      [ $rc = 2 ] && { aiwf reallocate "$(entity_file "$id")" </dev/null >/dev/null; id=$(basename "$(entity_file_glob alpha-gap)" | cut -c1-6); }; [ $tries -gt 6 ] && break; done
    unset CLAIM_NO_NONCE_CHECK
    echo "  recovery ($rec): final id $id after $tries claim attempts"
    cite G-0001 "See $id (means:alpha-gap)."; git push -q --no-verify origin mr/alpha )
  summary
done
for rec in naive readback; do
  echo "######## L2-$rec: push rejected, client believes it succeeded"
  setup "$X/runs/b-lost2-$rec"; mkclone alpha; mkclone beta
  ( cd "$W/beta"; git checkout -q -b mr/beta; git fetch -q origin; addgap "beta gap" >/dev/null )   # beta's view: ledger absent
  ( cd "$W/alpha"; git checkout -q -b mr/alpha; git fetch -q origin; id=$(addgap "alpha gap")
    claim_once "$id" alpha-1; echo "  alpha claims $id: rc=$?"; cite G-0001 "See $id (means:alpha-gap)."; git push -q --no-verify origin mr/alpha )
  ( cd "$W/beta"; id=G-0002
    c=$(ledger_commit "" "work/gaps/$id.md" "claimed-by: beta
nonce: beta-1")
    git push -q --no-verify origin "$c:refs/heads/aiwf/ids" 2>/dev/null; echo "  beta claims $id from its stale view: push rc=$? (beta treats it as success)"
    if [ $rec = readback ]; then
      git fetch -q origin; h=$(ledger_holder "$id")
      if ! grep -q 'nonce: beta-1' <<<"$h"; then
        echo "  read-back: ledger holds $id for '$(head -1 <<<"$h")' -> reallocate and claim again"
        aiwf reallocate "$(entity_file "$id")" </dev/null >/dev/null; id=$(basename "$(entity_file_glob beta-gap)" | cut -c1-6)
        claim_once "$id" beta-2; echo "  beta claims $id: rc=$?"
      fi
    fi
    cite M-0001 "Blocked on $id (means:beta-gap)."; git push -q --no-verify origin mr/beta )
  summary
done
