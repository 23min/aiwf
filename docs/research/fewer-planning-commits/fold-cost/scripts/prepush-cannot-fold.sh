#!/bin/sh
# A pre-push hook that rewrites the branch being pushed does not change what is
# pushed: git has already chosen the commits. Usage: prepush-cannot-fold.sh <scratch-dir>
set -eu
D=$1; rm -rf "$D"; mkdir -p "$D"; cd "$D"
git init -q --bare remote.git
git init -q -b main work; cd work
git -c user.name=t -c user.email=t@example.com commit -q --allow-empty -m one
git -c user.name=t -c user.email=t@example.com commit -q --allow-empty -m "plan 1"
git -c user.name=t -c user.email=t@example.com commit -q --allow-empty -m "plan 2"
git remote add origin ../remote.git
cat > .git/hooks/pre-push <<'HOOK'
#!/bin/sh
# fold the last two commits into one before the push
git reset -q --soft HEAD~2 && git -c user.name=t -c user.email=t@example.com commit -q --allow-empty -m folded
exit 0
HOOK
chmod +x .git/hooks/pre-push
git push -q origin main
echo "git $(git --version | cut -d' ' -f3)"
echo "local branch after the push: $(git log --format=%s main | tr '\n' ' ')"
echo "remote branch:               $(git --git-dir=../remote.git log --format=%s main | tr '\n' ' ')"
