#!/usr/bin/env bash
# setup.sh <run-dir> <checkout-sha> <task>
# Builds one run directory per the M-0334 rubric's Harness section; run.sh
# calls it. The repository it clones is the one this script sits in, or
# AIWF_SRC.
set -euo pipefail
RUN=$1 SHA=$2 TASK=$3
H=$(dirname "$(readlink -f "$0")")
SRC=${AIWF_SRC:-$(git -C "$H" rev-parse --show-toplevel)}
mkdir -p "$RUN"/{bin,home,gobin}
git clone -q --bare --no-hardlinks "$SRC" "$RUN/origin.git"
git -C "$RUN/origin.git" branch -f main "$SHA"
git -C "$RUN/origin.git" branch -f scratch/old-experiment "$SHA"
git -C "$RUN/origin.git" symbolic-ref HEAD refs/heads/main
git -C "$RUN/origin.git" for-each-ref --format='%(refname)' | grep -v -x -e refs/heads/main -e refs/heads/scratch/old-experiment | while read -r r; do git -C "$RUN/origin.git" update-ref -d "$r"; done
git clone -q --no-hardlinks -b main "$RUN/origin.git" "$RUN/repo"
# The aiwf under test is built from the checkout commit, once per commit.
BIN="${OBSERVE_DIR:?run.sh exports it}/aiwf-$SHA"
if [ ! -x "$BIN" ]; then (cd "$RUN/repo" && go build -o "$BIN.tmp" ./cmd/aiwf && mv "$BIN.tmp" "$BIN"); fi
cp "$BIN" "$RUN/bin/aiwf-real"
cat > "$RUN/bin/aiwf" <<SHIM
#!/usr/bin/env bash
case "\${1:-}" in
  update|init|upgrade|worktree)
    HOME="$RUN/home" XDG_CONFIG_HOME="$RUN/home/.config" XDG_STATE_HOME="$RUN/home/.local/state" XDG_DATA_HOME="$RUN/home/.local/share" XDG_CACHE_HOME="$RUN/home/.cache" exec "$RUN/bin/aiwf-real" "\$@" ;;
  *) exec "$RUN/bin/aiwf-real" "\$@" ;;
esac
SHIM
chmod +x "$RUN/bin/aiwf"
export PATH="$RUN/bin:$PATH"
cd "$RUN/repo"
{ "$RUN/bin/aiwf-real" version; sha256sum "$RUN/bin/aiwf-real"; } > "$RUN/binary.txt"
# Hooks and host artifacts come from the binary under test, not from any checkout.
make -s install-hooks > /dev/null
aiwf update < /dev/null > "$RUN/update.log" 2>&1
test -z "$(git status --porcelain)"
aiwf doctor > "$RUN/doctor.txt" 2>&1 || true
! grep -q -i drift "$RUN/doctor.txt"
: > "$RUN/artifacts.sha256"
for p in .claude/aiwf-guidance.md .claude/skills .claude/agents .claude/templates .agents/skills .agents/aiwf; do
  echo "$(find "$p" -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum | cut -c1-64) $p" >> "$RUN/artifacts.sha256"
done
# The rubric stays out of the tree the hosts work in.
spec=$(git ls-files | grep '/M-0334-')
python3 "$H/redact.py" "$spec" > "$RUN/redacted-body.md"
aiwf edit-body M-0334 --body-file "$RUN/redacted-body.md" > /dev/null
case "$TASK" in
  T3)
    python3 - <<'PY'
p = "internal/version/version.go"
s = open(p).read()
i = s.index("func Compare(")
j = s.index("\n}\n", i)
body = s[i:j]
assert body.count("for i := 0; i < 3; i++ {") == 1
s = s[:i] + body.replace("for i := 0; i < 3; i++ {", "for i := 1; i < 3; i++ {") + s[j:]
open(p, "w").write(s)
PY
    test "$(git diff --numstat | awk '{print $1+$2}')" = 2
    git commit -q -am "refactor(version): tidy the Compare loop" --no-verify ;;
  T8)
    id=$(aiwf add milestone --epic E-0092 --tdd none --title "Record a manual export check" | grep -o 'M-[0-9]*' | head -1)
    aiwf add ac "$id" --title "The export lists every milestone's id" > /dev/null ;;
esac
git push -q --no-verify origin main
case "$TASK" in
  T8) echo >> ROADMAP.md ;;
  T10)
    git checkout -q -b fix/list-json-parent
    aiwf add gap --title "aiwf list JSON omits the parent field for milestones" --body-file "$H/gap-branch.md" > /dev/null
    git clone -q "$RUN/origin.git" "$RUN/trunk"
    (cd "$RUN/trunk" && aiwf add gap --title "aiwf show does not mark an archived entity's path as archived" --body-file "$H/gap-trunk.md" > /dev/null && git push -q --no-verify origin main)
    rm -rf "$RUN/trunk"
    git fetch -q
    aiwf check > "$RUN/setup-check.txt" 2>&1 || true
    grep -q 'ids-unique/trunk-collision' "$RUN/setup-check.txt" ;;
esac
git rev-parse HEAD > "$RUN/start-head"
git rev-parse origin/main > "$RUN/start-origin"
grep -m1 'Installed commit' .guidance/index.md > "$RUN/index-commit.txt"
# Codex runs against a copy of its home, so its trust entries and session
# rollout land in the run directory rather than the personal one.
mkdir -p "$RUN/codex-home"
for f in auth.json config.toml hooks.json rules skills version.json installation_id; do
  [ -e ~/.codex/$f ] && cp -a ~/.codex/$f "$RUN/codex-home/"
done
cp -L ~/.codex/AGENTS.md "$RUN/codex-home/AGENTS.md"
touch "$RUN/.start"
