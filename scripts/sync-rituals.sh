#!/usr/bin/env bash
# sync-rituals.sh — vendor the ai-workflow-rituals plugins/ subtree at the
# pinned ref recorded in rituals.lock into internal/skills/embedded-rituals/.
#
# Build-time embed of a vendored snapshot is the distribution mechanism for
# the rituals (ADR-0014). This script is the "vendor" half: it fetches the
# upstream commit named in rituals.lock and replaces the vendored copy with
# upstream's plugins/ tree, verbatim. It does NOT commit — review the diff
# and commit yourself.
#
# The drift test (internal/policies/rituals_drift_test.go) fails if the
# vendored copy diverges from upstream@ref, so a forgotten re-sync is caught
# mechanically.
#
# Usage: make sync-rituals   (or: scripts/sync-rituals.sh)
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
lock="$repo_root/rituals.lock"
dest="$repo_root/internal/skills/embedded-rituals/plugins"

[ -f "$lock" ] || { echo "ERROR: $lock not found" >&2; exit 2; }

url="$(grep -E '^url=' "$lock" | head -n1 | cut -d= -f2-)"
ref="$(grep -E '^ref=' "$lock" | head -n1 | cut -d= -f2-)"
[ -n "$url" ] || { echo "ERROR: rituals.lock has no url=" >&2; exit 2; }
[ -n "$ref" ] || { echo "ERROR: rituals.lock has no ref=" >&2; exit 2; }

echo "Vendoring $url @ $ref"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

git -C "$tmp" init -q
git -C "$tmp" remote add origin "$url"
git -C "$tmp" fetch -q --depth 1 origin "$ref"
git -C "$tmp" checkout -q FETCH_HEAD

[ -d "$tmp/plugins" ] || { echo "ERROR: upstream@$ref has no plugins/ dir" >&2; exit 1; }

rm -rf "$dest"
mkdir -p "$(dirname "$dest")"
cp -R "$tmp/plugins" "$dest"

count="$(find "$dest" -type f | wc -l | tr -d ' ')"
echo "Vendored $count files into ${dest#"$repo_root"/}"
echo "Next: review the diff and commit (the snapshot is not auto-committed)."
