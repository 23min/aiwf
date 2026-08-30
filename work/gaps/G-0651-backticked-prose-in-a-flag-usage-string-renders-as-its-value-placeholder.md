---
id: G-0651
title: Backticked prose in a flag usage string renders as its value placeholder
status: open
discovered_in: M-0325
---
## What's missing

Cobra reads a backtick-quoted span inside a flag's usage string as the name of that
flag's *value placeholder*. Several `aiwf` flags quote a command or an example value
that way, so `--help` prints the quoted phrase where the value's type should be.

Measured against `aiwf` built from `acde2cd7c`:

```
$ aiwf authorize --help
      --branch main     ritual branch the scope is bound to ...
$ aiwf add --help
      --fetch git fetch --all   before allocating the id, ...
      --title aiwf add ac       entity title (required; ...
$ aiwf promote --help
      --reason aiwf history    free-form prose explaining why; ...
```

Expected `--branch string`, `--title string`, `--reason string` — the rendering every
other string flag gets. Observed the phrase from the usage text instead. `aiwf add`'s
`--validator` is affected the same way. The registrations are in
`internal/cli/authorize/authorize.go`, `internal/cli/add/`, and `internal/cli/promote/`;
each offender is a usage string containing a backticked span.

## Why it matters

`--help` is one of the four channels the kernel's AI-discoverability rule names, and
here it states something false about the interface it documents: a reader — human or
assistant — sees `--branch main` and has no way to tell whether the flag takes an
arbitrary ref, requires the literal word `main`, or takes a two-word argument.
`--title aiwf add ac` reads as though the title flag accepts a command. Nothing
detects the class: the help-banner drift test checks that a flag is *mentioned* in
the hand-written banner, and the completion drift test checks that a completion
function is *registered*; neither reads the rendered flag table.
