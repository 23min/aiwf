---
id: G-0794
title: trailer-verb-unknown stays a warning in every repository but aiwf's own
status: open
discovered_in: M-0365
---
## What's missing

`trailer-verb-unknown` reports an `aiwf-verb` outside the verb tree and the ritual allowlist as
an error only on a commit descending from `HookInstallSHA`, aiwf's own commit `0baed90b`
(`internal/check/trailer_verb_unknown.go`). In any other repository `check.WalkPostCutoffSHAs`
(`internal/check/post_cutoff.go`) fails to resolve `HookInstallSHA..HEAD` and returns nil, so
every such finding is a warning and the push goes through. G-0218 asked for an error on "commits
authored after the hook lands"; `post_cutoff.go`'s comment lists a "repo unrelated to aiwf trunk"
among the cases that stay silent, so the code documents as a fallback what falls short of that.

A fix lands in `HookInstallSHA` and the severity choice in `RunTrailerVerbUnknown`;
`WalkPostCutoffSHAs`; the `postCutoffSHAs` parameter that `internal/cli/check/check.go` passes
through `RunProvenanceCheck` (`internal/cli/check/provenance.go`) into `RunTrailerVerbUnknown`;
`post_cutoff_test.go`, `trailer_verb_unknown_test.go` and `provenance_test.go`; and the
`aiwf-check` skill's `trailer-verb-unknown` row, which states the cutoff.

Measured 2026-11-28 with aiwf built by `go build ./cmd/aiwf` from `main` at `35356d4a4`, git
2.51.1, Linux devcontainer, `GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null`, the binary
first on `PATH`. The setup commands' output is left out, and of the hooked commit's output only
the commit-msg refusal's first line is shown:

```
$ git init -q --bare -b main remote.git && git init -q -b main w && cd w
$ git config user.email t@example.com && git config user.name t
$ printf 'hosts: []\nguidance:\n  enabled: false\n' > aiwf.yaml && aiwf init
$ git add -A && git commit -q -m "chore: adopt aiwf"
$ aiwf add gap --title "A gap" --body-file ../body.md
$ git remote add origin ../remote.git && git push -q -u origin main
$ echo notes > notes.txt && git add notes.txt
$ git commit -q -m "feat: notes" --trailer "aiwf-verb: implement" --trailer "aiwf-entity: G-0001" --trailer "aiwf-actor: human/t"
aiwf check: commit-msg refuses aiwf-verb trailer value(s): ["implement"]
$ git commit -q --no-verify -m "feat: notes" --trailer "aiwf-verb: implement" --trailer "aiwf-entity: G-0001" --trailer "aiwf-actor: human/t"
$ aiwf history G-0001
2026-11-28T09:03:09Z  human/t           add         -             aiwf add gap G-0001 "A gap"  c5242cf
2026-11-28T09:03:10Z  human/t           implement   -             feat: notes  ece58bf
$ git push origin main
trailer-verb-unknown (warning) × 1 — commit ece58bf5 carries aiwf-verb: "implement" which is not a registered top-level verb or subverb (closed set sourced from the running binary's Cobra command tree)

1 findings (0 errors, 1 warnings)
run `aiwf check --verbose` for each warning's location and remediation hint
To ../remote.git
   c5242cf..ece58bf  main -> main
```

Expected: an error, and the push refused.

## Why it matters

In any repository but aiwf's, a commit that gets a made-up `aiwf-verb` past the commit-msg hook,
with `--no-verify` or git plumbing, reaches the remote, and `aiwf history` lists it as a verb
event on the entity it names, as `implement` above. The hook is then the only refusal.

## Related

- The behaviour to restore is answer Q16 of M-0365's audit (`spec/operation/audit/decisions.md`
  on `milestone/M-0365-model-operation-identity-in-a-ratified-spine`, reaching `main` with
  E-0100): the rule reads only the push range, which keeps history on trunk out, so the cutoff
  goes and every unknown verb in that range is an error; `aiwf acknowledge illegal <sha>` silences
  one on a commit that cannot be rewritten.
- As of 2026-11-28 nothing orders it after other work: E-0100 changes no aiwf code, and no
  unmerged branch touches these files.
- G-0765: the rule misses the commits a `--no-ff` merge brings in. G-0679: the rule does not run
  at a branch's first push from a local ref.
