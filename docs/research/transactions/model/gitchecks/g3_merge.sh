#!/usr/bin/env bash
# Merge granularity for an aiwf-shaped entity file.
set -euo pipefail
D=$(mktemp -d "${TMPDIR:-/tmp}/aiwf-gitcheck-merge.XXXX"); cd "$D"
git init -q -b main .; git config user.name peter; git config user.email peter@example.com; git config commit.gpgsign false
mk() { cat > m.md <<M
---
id: M-0001
title: Spike milestone
status: $1
parent: E-0001
tdd: none
---

## Goal

$2

## Acceptance criteria
M
}
mk draft "Show what a planning commit keeps."; git add m.md; git commit -qm base
git checkout -q -b a; mk in_progress "Show what a planning commit keeps."; git commit -qam "a: status"
git checkout -q -b b main; mk draft "Show what a planning commit keeps, rewritten."; git commit -qam "b: body"
git checkout -q a
if git merge -q --no-edit b >/dev/null 2>&1; then echo "G3a status line vs body paragraph of one file: clean merge; result status=$(grep ^status m.md) body=$(sed -n 11p m.md)"; else echo "G3a: CONFLICT"; git merge --abort; fi
git checkout -q -b c main; mk done "Show what a planning commit keeps."; git commit -qam "c: status"
git checkout -q a
if git merge -q --no-edit c >/dev/null 2>&1; then echo "G3b two different status values: clean"; else echo "G3b two different status values on one line: CONFLICT"; git merge --abort; fi
git checkout -q -b d main; mk draft "Body by d."; git commit -qam d
git checkout -q -b e main; mk draft "Body by e."; git commit -qam e
if git merge -q --no-edit d >/dev/null 2>&1; then echo "G3c two body edits: clean"; else echo "G3c two different edits of one body line: CONFLICT"; git merge --abort; fi
