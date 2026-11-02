#!/usr/bin/env bash
# Design C — two gaps whose ids differ only in width (G-0012 and G-00012), each made by hand.
source "$(dirname "$0")/lib.sh"
setup "$X/runs/c-widths"; cd "$W/seed"
for id in G-0012 G-00012; do printf -- '---\nid: %s\ntitle: width %s\nstatus: open\n---\n%s' "$id" "$id" "$GB" > "work/gaps/$id-width.md"; done
git add -A; git commit -q -m "chore: two widths" 2>/dev/null
show aiwf check | grep -E 'error|G-00?012|findings|exit' | cut -c1-200
show aiwf show G-0012 | head -2
show aiwf show G-00012 | head -2
show aiwf show G-12 | head -2
echo "  next aiwf add gap: $(addgap "after widths")"
