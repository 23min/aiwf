#!/usr/bin/env bash
# Design E (ADR-0001 mint at integration) and D (merge-request numbers) — the parts measurable locally.
# There is no `aiwf assign-pending`; the mint step below is a shell stand-in (git mv + sed), so what is
# measured is git, the simulated host and today's aiwf around it, not a mint implementation.
source "$(dirname "$0")/lib.sh"
setup "$X/runs/e-mint"
# Simulated protected main: only pushes carrying push option actor=merge-bot or actor=mint-bot may update it.
git -C "$W/host.git" config receive.advertisePushOptions true
cat > "$W/host.git/hooks/pre-receive" <<'EOF'
#!/bin/sh
actor=""; i=0
while [ $i -lt "${GIT_PUSH_OPTION_COUNT:-0}" ]; do eval "o=\$GIT_PUSH_OPTION_$i"; case $o in actor=*) actor=${o#actor=};; esac; i=$((i+1)); done
while read old new ref; do
  if [ "$ref" = refs/heads/main ] && [ "$actor" != merge-bot ] && [ "$actor" != mint-bot ]; then
    echo "protected: main accepts only merge-bot or mint-bot (got '${actor:-none}')"; exit 1; fi
done
EOF
chmod +x "$W/host.git/hooks/pre-receive"
echo "== P1: a developer's direct push to protected main"
mkclone alpha hooks; ( cd "$W/alpha"; aiwf add gap --title "direct" --body "$GB" </dev/null >/dev/null; show git push -q origin main | grep -E 'protected|exit' )
git -C "$W/alpha" reset -q --hard origin/main
echo "== P2: two branches each file a pre-mint gap in the inbox shape ADR-0001 describes"
for who in alpha beta; do
  [ -d "$W/$who" ] || mkclone "$who" hooks
  ( cd "$W/$who"; git checkout -q -b "mr/$who"; mkdir -p work/gaps/inbox
    printf -- '---\nid: %s-login-loop\ntitle: %s login loop\nstatus: open\n---\n%s' "$who" "$who" "$GB" > "work/gaps/inbox/$who-login-loop.md"
    git add -A; git commit -q -m "chore: file $who-login-loop in the inbox" 2>/dev/null
    show git push -q origin "mr/$who" | grep -E 'error|unexpected|exit' | cut -c1-160 )
done
echo "== P3: merge-bot merges mr/alpha; a fresh clone in the window before the mint"
hostclone; ( cd "$W/hostwork"; git merge -q --no-ff --no-edit origin/mr/alpha; show git push -q --no-verify -o actor=merge-bot origin main | grep -E 'protected|exit' )
rm -rf "$W/window"; git clone -q "$W/host.git" "$W/window"
( cd "$W/window"; echo "  window: gap ids aiwf sees: $(aiwf list --kind gap --format=json </dev/null | jq -r '[.result[].id]|join(" ")')"
  show aiwf check | grep -E 'inbox|findings|exit' | cut -c1-170 )
mint() { # mint in the cwd (a host clone on main): every inbox gap -> next G id, one commit
  local f slug n id
  for f in work/gaps/inbox/*.md; do
    [ -e "$f" ] || continue
    slug=$(basename "$f" .md); n=$(ls work/gaps | grep -oE '^G-[0-9]+' | sed 's/G-//' | sort -n | tail -1); id=$(printf 'G-%04d' $((10#$n + 1)))
    git mv "$f" "work/gaps/$id-$slug.md"; sed -i "s/^id: $slug\$/id: $id/" "work/gaps/$id-$slug.md"
    printf 'aiwf-entity: %s\naiwf-mint: %s = gap:%s\n' "$id" "$id" "$slug" >> "$W/mint-trailers"
  done
  rmdir work/gaps/inbox 2>/dev/null
}
echo "== P4: mint-bot mints on its clone while merge-bot merges mr/beta first (race)"
rm -rf "$W/mintwork"; git clone -q "$W/host.git" "$W/mintwork"; git -C "$W/mintwork" config user.name mint-bot; git -C "$W/mintwork" config user.email mint@example.com
( cd "$W/mintwork"; : > "$W/mint-trailers"; mint
  printf 'aiwf assign-pending: minted\n\naiwf-verb: assign-pending\n%s\naiwf-actor: framework/aiwf-mint\naiwf-principal: human/merger\n' "$(cat "$W/mint-trailers")" > "$W/mint-msg"
  echo "-- commit through aiwf's commit-msg hook (copied from the seed):"
  for h in commit-msg pre-commit; do cp "$W/seed/.git/hooks/$h" .git/hooks/; done
  git add -A; show git commit -q -F "$W/mint-msg" | cut -c1-260
  git commit -q --no-verify -F "$W/mint-msg"; echo "  minted: $(ls work/gaps | tr '\n' ' ')" )
( cd "$W/hostwork"; git fetch -q origin; git merge -q --no-ff --no-edit origin/mr/beta; git push -q --no-verify -o actor=merge-bot origin main )
( cd "$W/mintwork"; show git push -q --no-verify -o actor=mint-bot origin main | grep -E 'rejected|protected|exit'
  git fetch -q origin; git reset -q --hard origin/main; : > "$W/mint-trailers"; mint
  printf 'aiwf assign-pending: minted\n\naiwf-verb: assign-pending\n%s\naiwf-actor: framework/aiwf-mint\naiwf-principal: human/merger\n' "$(cat "$W/mint-trailers")" > "$W/mint-msg"
  git add -A; git commit -q --no-verify -F "$W/mint-msg"
  show git push -q --no-verify -o actor=mint-bot origin main | grep -E 'rejected|protected|exit'
  echo "  after re-mint: $(ls work/gaps | tr '\n' ' ')" )
rm -rf "$W/after"; git clone -q "$W/host.git" "$W/after"
( cd "$W/after"; show aiwf check | grep -E 'error|trailer|findings|exit' | cut -c1-200; show aiwf history G-0002 )
