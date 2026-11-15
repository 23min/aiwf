---
id: G-0768
title: Actor tests read the checkout's git config, so a local user.email fails them
status: open
---
## What's missing

Four tests isolate the global and system git config but not the checkout they run
in. `TestResolveActor_DerivedFromGitConfig`, `TestResolveActor_MalformedGitEmail`
and `TestResolveActor_NoConfigErrors` (`internal/cli/cliutil/actor_test.go`) point
`HOME` and `XDG_CONFIG_HOME` at a temp directory and set `GIT_CONFIG_NOSYSTEM=1`;
`TestBrokenGitIdentity_TriggersResolveActorFailure`
(`internal/cli/cliutil/testutil/fixtures_test.go`) relies on the
`testutil.BrokenGitIdentity` helper, which isolates only `HOME`. Each then calls
`ResolveActor("", "")`, which runs `git config user.email` in the working
directory. Under `go test` that is the package directory inside the developer's
checkout, so the checkout's own `.git/config` is read as well, and a `user.email`
set there wins. The helper's doc comment promises that `ResolveActor("", root)`
fails under it; with such an email it does not.

The other tests in `actor_test.go` are unaffected: one passes an explicit actor,
which returns before git runs, and the rest resolve against temp directories
outside the checkout.

Measured at `08feaacc9` (go1.25.12, git 2.51.1, Linux) in a fresh clone of this
repository, from its root:

```
$ go test -count=1 -run '^TestResolveActor|^TestBrokenGitIdentity' ./internal/cli/cliutil/...
ok  	github.com/23min/aiwf/internal/cli/cliutil	0.052s
ok  	github.com/23min/aiwf/internal/cli/cliutil/testutil	0.010s
$ git config --local user.email probe@example.com
$ go test -count=1 -run '^TestResolveActor|^TestBrokenGitIdentity' ./internal/cli/cliutil/...
--- FAIL: TestResolveActor_DerivedFromGitConfig (0.00s)
    actor_test.go:114: got "human/probe", want human/peter
--- FAIL: TestResolveActor_MalformedGitEmail (0.00s)
    actor_test.go:157: expected error from malformed git config user.email, got nil
--- FAIL: TestResolveActor_NoConfigErrors (0.00s)
    actor_test.go:172: expected error when neither --actor nor git config is set
FAIL
FAIL	github.com/23min/aiwf/internal/cli/cliutil	0.051s
--- FAIL: TestBrokenGitIdentity_TriggersResolveActorFailure (0.00s)
    fixtures_test.go:23: ResolveActor("", "") = nil error, want an error under BrokenGitIdentity
FAIL
FAIL	github.com/23min/aiwf/internal/cli/cliutil/testutil	0.008s
FAIL
```

## Why it matters

A contributor whose checkout carries its own `user.email`, set with
`git config user.email` inside the repository, gets failures in the test run that
point at actor resolution rather than at their environment. It has happened in
this repository: a probe wrote `user.email` into the checkout's `.git/config`,
and the `TestResolveActor_*` failures read as environmental.

## Related

- G-0375 — fixtures that inherit a global `commit.gpgsign`; closed with no sighting
  of that leak recorded.
- G-0720 — the init and update tests run against the real `HOME` and `PATH`.
