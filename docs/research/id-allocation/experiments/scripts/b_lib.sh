# Design B ledger client, sourced after lib.sh. The ledger is an ordinary branch aiwf/ids on the host
# (so a default fetch brings it to refs/remotes/origin/aiwf/ids, where aiwf's allocator scans it).
# Each claim is one commit adding work/gaps/<id>.md whose content names the claimer and a nonce.
# Protocol (no aiwf code change): allocate with `aiwf add`, then claim the id aiwf gave; if someone
# else holds it, `aiwf reallocate` (the allocator now sees their claim) and claim again — all before
# the id is cited or the branch is pushed.

ledger_commit() { # ledger_commit <parent-or-empty> <path> <content>   (cwd = clone)
  local p=$1 idx blob t
  idx=$(mktemp "$X/out/idx.XXXXXX"); rm -f "$idx"
  [ -n "$p" ] && GIT_INDEX_FILE=$idx git read-tree "$p"
  blob=$(printf '%s\n' "$3" | git hash-object -w --stdin)
  GIT_INDEX_FILE=$idx git update-index --add --cacheinfo "100644,$blob,$2"
  t=$(GIT_INDEX_FILE=$idx git write-tree); rm -f "$idx"
  if [ -n "$p" ]; then git commit-tree "$t" -p "$p" -m "claim $2"; else git commit-tree "$t" -m "claim $2"; fi
}

# ledger_holder <id>: print the content of the ledger's claim for <id> (empty if unclaimed).
ledger_holder() {
  git cat-file -p "refs/remotes/origin/aiwf/ids:work/gaps/$1.md" 2>/dev/null || true
}

# claim_once <id> <nonce> [push-cmd-prefix]: 0 = mine, 1 = ledger moved / push failed, 2 = held by another.
# With CLAIM_TRUST_PUSH=1 the client believes a push it was told failed (or vice versa, see callers).
claim_once() {
  local id=$1 nonce=$2 parent holder c rc
  git fetch -q origin 2>/dev/null
  holder=$(ledger_holder "$id")
  if [ -n "$holder" ]; then
    if [ "${CLAIM_NO_NONCE_CHECK:-0}" = 0 ] && grep -q "nonce: $nonce" <<<"$holder"; then return 0; fi
    return 2
  fi
  parent=$(git rev-parse -q --verify refs/remotes/origin/aiwf/ids || true)
  c=$(ledger_commit "$parent" "work/gaps/$id.md" "claimed-by: $(git config user.name)
nonce: $nonce")
  ${PUSH_PREFIX:-} git push -q --no-verify origin "$c:refs/heads/aiwf/ids" 2>>"$X/out/push-errors.log"; rc=$?
  [ $rc = 0 ] && return 0 || return 1
}

# file_gap <who> <cite-from>: the whole protocol on a fresh branch mr/<who>; prints a summary line.
file_gap() {
  local who=$1 cite_from=$2 id f nonce rc tries=0 reallocs=0
  git checkout -q -b "mr/$who"
  git fetch -q origin
  id=$(addgap "$who gap"); nonce="$who-$RANDOM$RANDOM"
  while [ $tries -lt 20 ]; do
    tries=$((tries+1))
    claim_once "$id" "$nonce"; rc=$?
    [ $rc = 0 ] && break
    if [ $rc = 2 ]; then
      f=$(entity_file "$id"); aiwf reallocate "$f" </dev/null >/dev/null || die "reallocate $f"
      reallocs=$((reallocs+1)); id=$(basename "$(entity_file_glob "$who-gap")" | cut -c1-6)
    fi
  done
  cite "$cite_from" "See $id (means:$who-gap)."
  local hp=ok; git push -q origin "mr/$who" >/dev/null 2>&1 || { hp=refused; git push -q --no-verify origin "mr/$who" >/dev/null 2>&1; }
  echo "$who claimed=$id tries=$tries reallocates=$reallocs first-shared=$id hooked-push=$hp"
}
entity_file_glob() { ls work/gaps/*-"$1".md 2>/dev/null | head -1; }
