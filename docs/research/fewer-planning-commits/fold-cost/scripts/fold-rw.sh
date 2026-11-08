#!/usr/bin/env bash
# Minimal fold, plain git plumbing.
#   fold.sh <base> [sign|nosign|inherit]
# Walks base..HEAD on the first-parent line. A planning commit is a non-merge commit
# carrying an aiwf-verb trailer. Commits before the first planning commit keep their
# SHAs. From there on, every other commit is re-created in order by a three-way
# merge of its own change (parent -> commit) onto the new line; a merge is re-made
# with its original second parent and its change taken relative to its first parent.
# Then the planning commits' changes are applied in order, three-way, onto the new
# tip's tree and committed once, carrying every planning commit's trailers in order.
# The branch moves by compare-and-swap. Old->new SHAs are written to .git/fold-map.
# VARIANT: also rewrites every full 40-hex old SHA of a re-created commit to its new SHA
# in re-created commit messages, in the folded trailers, and in the planning files the
# fold commits (short SHAs are not rewritten).
# sign: pass -S to commit-tree; nosign: --no-gpg-sign; inherit (default): neither.
set -euo pipefail
base=$1; mode=${2:-inherit}
case $mode in sign) SF=(-S);; nosign) SF=(--no-gpg-sign);; inherit) SF=();; *) echo "bad mode" >&2; exit 2;; esac
ref=$(git symbolic-ref HEAD)
old=$(git rev-parse HEAD)
gd=$(git rev-parse --git-dir)
mapfile -t commits < <(git rev-list --reverse --first-parent "$base..$old")

is_merge() { [ "$(git rev-list --no-walk --parents "$1" | wc -w)" -gt 2 ]; }
is_planning() { ! is_merge "$1" && [ -n "$(git log -1 --format='%(trailers:key=aiwf-verb,valueonly)' "$1")" ]; }
rawmsg() { git cat-file commit "$1" | sed '1,/^$/d'; }

i0=-1
for i in "${!commits[@]}"; do if is_planning "${commits[$i]}"; then i0=$i; break; fi; done
if [ $i0 -lt 0 ]; then echo "fold: no planning commits in $base..HEAD"; exit 0; fi

tip=$(git rev-parse "${commits[$i0]}^")
planning=()
: > "$gd/fold-map"; : > "$gd/fold-map.sed"
for c in "${commits[@]:$i0}"; do
  if is_planning "$c"; then planning+=("$c"); continue; fi
  if is_merge "$c"; then
    p1=$(git rev-parse "$c^1"); p2=$(git rev-parse "$c^2")
    parents=(-p "$tip" -p "$p2")
  else
    p1=$(git rev-parse "$c^"); parents=(-p "$tip")
  fi
  if ! out=$(git merge-tree --write-tree --merge-base="$p1" "$tip" "$c"); then
    echo "fold: conflict re-creating $(git log -1 --format='%h %s' "$c"); nothing changed" >&2
    echo "$out" | sed 's/^/  /' >&2; exit 1
  fi
  tree=$(echo "$out" | head -1)
  new=$(rawmsg "$c" | sed -f "$gd/fold-map.sed" | env \
      GIT_AUTHOR_NAME="$(git log -1 --format=%an "$c")" \
      GIT_AUTHOR_EMAIL="$(git log -1 --format=%ae "$c")" \
      GIT_AUTHOR_DATE="$(git log -1 --format=%ad --date=raw "$c")" \
      git commit-tree "${SF[@]}" "$tree" "${parents[@]}") || { echo "fold: commit-tree failed re-creating $(git log -1 --format=%h "$c"); nothing changed" >&2; exit 1; }
  echo "$c $new" >> "$gd/fold-map"; echo "s/$c/$new/g" >> "$gd/fold-map.sed"
  tip=$new
done

tree=$(git rev-parse "$tip^{tree}")
subjects=""; trailers=""
# git 2.51 merge-tree takes commits, not trees, as <branch1>: carry the accumulated
# tree in an unreferenced, unsigned scratch commit.
carrier=$tip
for p in "${planning[@]}"; do
  if ! out=$(git merge-tree --write-tree --merge-base="$p^" "$carrier" "$p"); then
    echo "fold: conflict applying $(git log -1 --format='%h %s' "$p"); nothing changed" >&2
    echo "$out" | sed 's/^/  /' >&2; exit 1
  fi
  tree=$(echo "$out" | head -1)
  carrier=$(echo carrier | git commit-tree --no-gpg-sign "$tree")
  subjects+="- $(git log -1 --format=%s "$p")"$'\n'
  trailers+="$(git log -1 --format='%(trailers:only=true,unfold=true)' "$p" | sed '/^$/d' | sed -f "$gd/fold-map.sed")"$'\n'
done
# rewrite old SHAs inside the planning files this commit carries
export GIT_INDEX_FILE="$gd/fold-index"; git read-tree "$tree"
for f in $(git diff --name-only "$tip" "$tree"); do
  b=$(git rev-parse "$tree:$f" 2>/dev/null) || continue
  nb=$(git cat-file blob "$b" | sed -f "$gd/fold-map.sed" | git hash-object -w --stdin)
  [ "$nb" != "$b" ] && { git update-index --cacheinfo "$(git ls-tree "$tree" -- "$f" | cut -d' ' -f1),$nb,$f"; echo "fold: rewrote SHAs in $f"; }
done
tree=$(git write-tree); unset GIT_INDEX_FILE; rm -f "$gd/fold-index"
msg="chore(plan): fold ${#planning[@]} planning operations"$'\n\n'"$subjects"$'\n'"$trailers"
new=$(printf '%s' "$msg" | git commit-tree "${SF[@]}" "$tree" -p "$tip") || { echo "fold: commit-tree failed on the planning commit; nothing changed" >&2; exit 1; }
for p in "${planning[@]}"; do echo "$p $new" >> "$gd/fold-map"; done

git update-ref -m "fold onto $(git rev-parse --short "$base")" "$ref" "$new" "$old"
git update-index -q --refresh >/dev/null; git read-tree -m -u "$old" "$new"
echo "fold: ${#planning[@]} planning commits folded, $(( ${#commits[@]} - i0 - ${#planning[@]} )) commits re-created"
if git diff --quiet "$old" "$new"; then echo "fold: final tree identical"; else echo "fold: FINAL TREE DIFFERS"; fi
