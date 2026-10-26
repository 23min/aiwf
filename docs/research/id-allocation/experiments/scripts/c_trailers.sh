#!/usr/bin/env bash
# Design C — a commit whose aiwf trailers name a namespaced id, through aiwf's commit-msg hook;
# then aiwf history on that id; then aiwf's pre-push on a branch carrying the hand-made file.
source "$(dirname "$0")/lib.sh"
setup "$X/runs/c-trailers"; mkclone alpha hooks; cd "$W/alpha"; git checkout -q -b feature/ns
printf -- '---\nid: G-pb-0012\ntitle: Namespaced\nstatus: open\n---\n%s' "$GB" > work/gaps/G-pb-0012-namespaced.md
git add -A
printf 'aiwf add gap G-pb-0012 "Namespaced"\n\naiwf-verb: add\naiwf-entity: G-pb-0012\naiwf-actor: human/alpha\n' > "$W/msg"
show git commit -q -F "$W/msg"
show git log -1 --format=%B
show aiwf history G-pb-0012
show git push -q -u origin feature/ns
show aiwf check --verbose | grep -E 'G-pb|provenance|trailer|findings|exit' | cut -c1-220
