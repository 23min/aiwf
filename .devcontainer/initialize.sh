#!/usr/bin/env bash
# Runs on the HOST (not in the container) via devcontainer.json
# `initializeCommand`. Prepares stable mount sources under /tmp so
# devcontainer.json `mounts:` entries don't have to reference $HOME
# (which devcontainer.json can't expand portably).
#
# Plugin shadow-mount workaround for anthropics/claude-code#31388:
# Claude Code's plugin index stores absolute host paths. A
# macOS-pathed index (~/.claude/plugins/...) breaks inside a Linux
# container, and a Linux-pathed index breaks back on the host. We
# shadow the container's plugin-index dir with ~/.claude-linux/plugins
# so the host's macOS-pathed index stays untouched and the container
# has its own Linux-pathed parallel index.
#
# Remove the .claude-plugins-mount entries here AND the corresponding
# mount in devcontainer.json once claude-code#31388 ships a fix that
# resolves plugin paths relative to $HOME. Tracking issue:
#   https://github.com/anthropics/claude-code/issues/31388
#
# The mount points:
#   ~/.codex-linux          → /tmp/.codex-mount           (container-only Codex state)
#   ~/.claude               → /tmp/.claude-mount          (full state shared with host)
#   ~/.claude-linux/plugins → /tmp/.claude-plugins-mount  (container-only plugin index)
#   ~/.config/gh            → /tmp/.gh-mount              (gh auth shared with host)
#   ~/.guidance             → /tmp/.aiwf-guidance-mount   (personal assistant rules, read-write)

set -euo pipefail

mkdir -p "$HOME/.claude"
mkdir -p "$HOME/.claude-linux/plugins"
mkdir -p "$HOME/.config/gh"
mkdir -p "$HOME/.codex-linux"
chmod 700 "$HOME/.codex-linux"
mkdir -p "$HOME/.guidance"

ln -sfn "$HOME/.claude"                /tmp/.claude-mount
ln -sfn "$HOME/.claude-linux/plugins"  /tmp/.claude-plugins-mount
ln -sfn "$HOME/.config/gh"             /tmp/.gh-mount
ln -sfn "$HOME/.codex-linux"           /tmp/.codex-mount
ln -sfn "$HOME/.guidance"              /tmp/.aiwf-guidance-mount

# Personal guidance (23min/guidance). Claude reaches ~/.guidance through the
# shared ~/.claude, so only the container-only Codex home needs a link, and it
# is made only once a build has produced ~/.guidance/AGENTS.md, so the
# AGENTS.md there keeps working until then. A missing checkout or a failing
# build never blocks a start.
if [ -x "$HOME/.guidance/build" ]; then
  "$HOME/.guidance/build" \
    || echo "initialize: $HOME/.guidance/build failed; its previous output, if any, stays in use" >&2
else
  echo "initialize: no personal guidance in $HOME/.guidance. Set it up with:" \
    "git clone https://github.com/23min/guidance.git $HOME/.guidance && $HOME/.guidance/install" >&2
fi
codex_rules=$HOME/.codex-linux/AGENTS.md
if [ -e "$HOME/.guidance/AGENTS.md" ]; then
  if [ -L "$codex_rules" ] || [ ! -e "$codex_rules" ]; then
    ln -sfn ../.guidance/AGENTS.md "$codex_rules"
  else
    echo "initialize: $codex_rules exists and is not a link, so Codex in containers" \
      "doesn't read $HOME/.guidance; move it aside to use it" >&2
  fi
fi
