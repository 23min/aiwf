---
id: G-0724
title: aiwf's managed .gitignore omits .claude/health.aiwf.json, which aiwf writes
status: open
---
## What's missing

`aiwf init` and `aiwf update` write a managed block into the consumer repo's
`.gitignore` (`ensureGitignore`, `internal/initrepo/initrepo.go`, with patterns
from `GitignorePatternsFor` in `internal/skills/skills.go`) covering the
artifacts aiwf generates there. The block leaves out `.claude/health.aiwf.json`,
which `aiwf doctor --write-health` and `aiwf update` write through `WriteHealth`
(`internal/cli/doctor/health.go`). In any repo that selects Claude Code the file
is therefore untracked and not ignored, unless something outside aiwf's block
ignores it. The aiwf repo has such a hand-written line (`.gitignore`,
`.claude/health.*.json`), so the defect does not show here.

Reproduced with aiwf v0.40.0 on Linux x86_64, each writer in a fresh temporary
repo:

```sh
for writer in "aiwf update --no-prompt" "aiwf doctor --write-health"; do
  tmp=$(mktemp -d) && cd "$tmp" && git init -q && git -c user.email=t@example.com -c user.name=t commit -q --allow-empty -m init
  printf 'hosts: [claude-code]\n' > aiwf.yaml
  aiwf init --no-prompt </dev/null >/dev/null 2>&1
  echo "== after aiwf init, then $writer:"
  echo "managed block present: $(grep -c 'aiwf: marker-managed framework artifacts' .gitignore), health lines: $(grep -c health .gitignore)"
  $writer </dev/null >/dev/null 2>&1
  git status --short -uall | grep health
  git check-ignore -v .claude/health.aiwf.json || echo "not ignored"
done
```

Expected `.claude/health.aiwf.json` to be ignored by aiwf's own block. Observed:

```
== after aiwf init, then aiwf update --no-prompt:
managed block present: 1, health lines: 0
?? .claude/health.aiwf.json
not ignored
== after aiwf init, then aiwf doctor --write-health:
managed block present: 1, health lines: 0
?? .claude/health.aiwf.json
not ignored
```

## Why it matters

Every consumer repo that selects Claude Code shows the regenerated health file
as untracked after `aiwf update` or `aiwf doctor --write-health`. Committing it
does not settle it: each later run rewrites its `generated_at`, so the tracked
file shows as modified again.

Copier (9.18.2 measured) refuses `copier update` with "Destination repository is
dirty; cannot continue" (exit 1) while the file is untracked or modified, so a
Copier-managed repo that also runs aiwf cannot take template updates until the
file is ignored by hand or deleted.

## Related

G-0305 introduced the file and specified it as gitignored via
`.claude/health.*.json`.
