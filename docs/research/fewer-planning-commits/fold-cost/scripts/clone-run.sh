#!/usr/bin/env bash
# clone-run.sh <src run> <dst run>: copy a run dir and repoint its clones' origin at the copy's bare remote.
set -euo pipefail
rm -rf "$2"; cp -a "$1" "$2"
for c in "$2"/a "$2"/b; do [ -d "$c" ] && git -C "$c" remote set-url origin "$2/remote.git"; done
sed -i "s#$1/home#$2/home#g" "$2/home/.gitconfig"
if [ -f "$2/home/.ssh/allowed_signers" ]; then :; fi
