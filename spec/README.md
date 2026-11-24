# spec/

Models of aiwf's state machines, made with [verifier-kit](https://github.com/23min/verifier-kit)
(E-0100). Each machine has a folder, `spec/<machine>/`, holding its audit, its chapter, the two
readings and the answers to their questions, the ratified `spine.yaml` with its drawing, and
`conform.md`, the record of the last comparison with the `aiwf` binary. The adapter that makes
that comparison is `conformance/adapter_<machine>.py` at the repository root, started by the line
in `conformance/adapter_<machine>.cmd`.

## Running verifier-kit

verifier-kit is not installed in this repository. Its tools run from a separate checkout at the
version the models were made with, v0.3.3 (E-0100, §"Open questions"). With `$VK` the absolute
path of a folder outside this repository, since the draw line below runs from `spec/<m>`:

```sh
git clone -q https://github.com/23min/verifier-kit.git "$VK"
git -C "$VK" checkout -q v0.3.3
(cd "$VK/tools" && GOTOOLCHAIN=auto go build -o bin/spine ./cmd/spine)
```

Then, from this repository's root, for a machine `<m>`:

```sh
"$VK/tools/bin/spine" check spec/<m>/spine.yaml
(cd spec/<m> && "$VK/tools/bin/spine" draw spine.yaml drawing.yaml > drawing.svg && git diff --exit-code drawing.svg)
PATH="$VK/tools:<folder holding the aiwf to compare>:$PATH" "$VK/tools/bin/spine" conform \
  spec/<m>/spine.yaml --margin 2 --adapter-cmd "$(cat conformance/adapter_<m>.cmd)"
python3 spec/check.py <m> "$VK"
```

`check` passes with no findings, and `draw` writes the committed drawing again. `conform` plays
the spine against the `aiwf` it finds on `PATH`; with the `aiwf` that `spec/<m>/conform.md`
names, it reports the divergences that file classifies. The adapter's header says what it needs
from `aiwf`, and where it puts its throwaway repositories. `check.py` checks the machine's records
against what E-0100's milestones ask of them, as its header lists, and exits 1 when one fails.
It reads the records in the shapes these files have, so a change to one of those shapes changes
it too. E-0100 owns `check.py`, the adapters and those shapes; when it closes they pass to the
first build epic that makes `spine conform` its done-check, which takes up installing
verifier-kit again (E-0100, §"Open questions").
