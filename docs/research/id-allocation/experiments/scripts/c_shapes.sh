#!/usr/bin/env bash
# Design C (and the id shapes D and E would produce) — what today's aiwf does with a hand-made gap
# whose id is not today's G-NNNN. Each shape gets a fresh repo with aiwf's hooks installed.
#   <label> <id> <file basename under work/gaps/ (or a subpath)>
source "$(dirname "$0")/lib.sh"
shapes=(
  "ns-prefix      G-pb-0012   G-pb-0012-namespaced.md"
  "random-b36     G-7f3k      G-7f3k-random.md"
  "wide-5         G-00012     G-00012-wide.md"
  "long-numeric   G-123456789 G-123456789-long.md"
  "ns-in-slug     G-0012      G-0012-pb-namespaced-in-slug.md"
  "letter-suffix  G-0012a     G-0012a-suffix.md"
  "mr-subindex    G-1234.2    G-1234.2-second-in-mr.md"
  "mr-dash-sub    G-1234-2    G-1234-2-second-in-mr.md"
  "inbox-slug     login-loop  inbox/login-loop.md"
)
for row in "${shapes[@]}"; do
  read -r label id file <<<"$row"
  echo "######## $label: id=$id file=work/gaps/$file"
  setup "$X/runs/c-$label"; cd "$W/seed"
  mkdir -p "$(dirname "work/gaps/$file")"
  printf -- '---\nid: %s\ntitle: Hand-made %s\nstatus: open\n---\n%s' "$id" "$label" "$GB" > "work/gaps/$file"
  git add -A
  show git commit -q -m "chore: hand-made $label gap" | grep -E 'error|warning|exit' | cut -c1-200
  show aiwf check | grep -E "error|$id|findings|exit" | grep -v 'milestone-draft\|entity-body-empty' | cut -c1-220
  show aiwf show "$id" | head -2 | cut -c1-160
  echo "  aiwf list --kind gap ids: $(aiwf list --kind gap --format=json </dev/null | jq -r '[.result[].id]|join(" ")')"
  echo "== a citation of it from G-0001's body"
  f=$(entity_file G-0001); tmp=$(mktemp "$X/out/cite.XXXXXX"); awk '/^---$/{n++; next} n>=2' "$f" > "$tmp"; printf '\nSee %s for the hand-made gap.\n' "$id" >> "$tmp"
  show aiwf edit-body G-0001 --body-file "$tmp" | cut -c1-200
  show aiwf check | grep -E "body-prose|refs-resolve|$id|findings|exit" | grep -v 'milestone-draft' | cut -c1-220
  echo "== a frontmatter reference to it through a verb: aiwf add decision --relates-to $id"
  show aiwf add decision --title "Relates to $label" --relates-to "$id" --body $'## Question\n\nq.\n\n## Decision\n\nd.\n\n## Reasoning\n\nr.\n' | cut -c1-200
  echo "== verbs on it"
  show aiwf retitle "$id" "Retitled $label" | cut -c1-200
  show aiwf promote "$id" wontfix | cut -c1-200
  echo "== the next aiwf add gap allocates: $(addgap "next after $label")"
done
