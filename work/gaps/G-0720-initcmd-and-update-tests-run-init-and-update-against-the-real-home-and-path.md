---
id: G-0720
title: initcmd and update tests run init and update against the real HOME and PATH
status: open
discovered_in: M-0352
---
## What's missing

The in-process tests in `internal/cli/initcmd` and `internal/cli/update` run `initcmd.Run` and `update.Run` with the test process's own HOME and PATH. Neither package's `TestMain` (`setup_test.go`) isolates them: it only prepends a `claude` stand-in to PATH through `testsupport.RunWithClaudeOnPATH`, and only the serial tests listed in `internal/cli/update/setup_test.go` isolate personal-guidance variables. Most tests leave guidance maintenance enabled, so each run reaches `projectguidance.CheckCompatibility` (`internal/projectguidance/compatibility.go`), which executes the first `dotfiles-guidance-check` on PATH, or, if there is none, reads the personal Claude and Codex instruction files under HOME (honouring `CLAUDE_CONFIG_DIR` and `CODEX_HOME`) and looks for `dotfiles-sync` launchers. `update.Run` also auto-refreshes the user-scope statusline under HOME (`refreshStatuslineInPlace` in `internal/cli/update/update.go`).

Measured at `f5a2c53f8`, running each package with HOME set to a scratch directory and a logging stand-in for `dotfiles-guidance-check` first on PATH:

```
$ HOME=<scratch> PATH=<stand-in dir>:$PATH go test -count=1 ./internal/cli/initcmd/   # stand-in invoked 11 times
$ HOME=<scratch> PATH=<stand-in dir>:$PATH go test -count=1 ./internal/cli/update/    # stand-in invoked 11 times
HOME=<scratch> PWD=…/internal/cli/initcmd      (a line from the stand-in's log)
```

Each invocation received the test process's HOME, which in a developer's shell is their real home. A failing `internal/cli/update` test's captured output in the devcontainer, with the real HOME, showed `skipped    statusline (user scope)  (cannot order versions (installed "v0.38.0", binary "(devel)") — not auto-refreshing)`, the version of the statusline installed in the real `~/.claude`. The reads of the personal instruction files follow from the code named above rather than from a measurement.

## Why it matters

The suites run the developer's installed `dotfiles-guidance-check` and reach their personal instruction files and statusline, so test outcomes depend on the machine: a failing or absent checker, or ai-dotfiles instructions in the real home, turns guidance refusals on or off under the tests. Any side effect of that checker, or a future write path in the statusline refresh or guidance handover, lands in the developer's real home.
