---
id: G-0279
title: Stale devcontainer banner instructs retired rituals plugin install
status: open
---
## What's missing

The devcontainer post-install banner in `.devcontainer/init.sh` (the `cat <<'BANNER' … BANNER` heredoc, ~lines 161–187) still tells the operator that "one manual step remaining" is to install the rituals as marketplace plugins at PROJECT scope:

```
  /plugin marketplace add 23min/ai-workflow-rituals
  /plugin   # install aiwf-extensions + wf-rituals at PROJECT scope
  /reload-plugins
```

That flow was retired. ADR-0014 / E-0038 moved ritual materialization into the engine binary — `aiwf init` / `aiwf update` materializes the `aiwfx-*` / `wf-*` skills, role agents, and templates into `.claude/` directly. ADR-0016 / G-0193 then retired the upstream `23min/ai-workflow-rituals` marketplace channel entirely (that repo is archived/read-only). The script *already* runs `aiwf init` at line 142, so the banner instructs an obsolete manual step on top of the real, working one. The `aiwf doctor` `recommended-plugin-not-installed` warning the banner references no longer exists either.

A matching stale reference lives in `.devcontainer/README.md` (~line 112: `ls /workspaces/ai-workflow-rituals/plugins/ … aiwf-extensions + wf-rituals`).

## Why it matters

Every container (re)open prints authoritative-looking instructions to perform a retired workflow against an archived upstream repo. A contributor — human or AI — who follows them either fails (the marketplace add points at a read-only repo) or is left believing rituals require a manual plugin install that `aiwf init` has in fact already done. This contradicts the current onboarding contract in CLAUDE.md ("no separate install step"; verify with `aiwf doctor`'s `rituals:` line), erodes trust in the devcontainer's own output, and wastes onboarding time. The fix is a doc/script correction: drop the manual-step block from the banner (confirm `aiwf init` ran, point at `aiwf doctor`'s `rituals:` line for verification) and correct the README reference.
