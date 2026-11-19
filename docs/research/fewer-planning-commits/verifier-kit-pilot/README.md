# verifier-kit pilot

Two aiwf state machines written as spines for [verifier-kit](https://github.com/23min/verifier-kit)'s
`spine` tool, and what `spine conform` reported when it played each one against the `aiwf` binary.
The pilot asks whether a spine finds defects in aiwf before ADR-0056 is built, and what writing one
costs. `scope/` is a machine ADR-0056 changes. `milestone/` tries the method on a machine that
planning commits do not touch, so its findings are about milestones and ACs, not about planning
commits.

| Path | What it holds |
|---|---|
| `scope/spec/scope/spine.yaml` | One agent's delegation on one milestone: the states none, active and ended, and the events `authorize`, `act`, `end`, `rebase` and `check` |
| `scope/conformance/adapter_scope.py`, `scope/template.sh` | The adapter that plays each event with `aiwf` in disposable repositories, and the script that builds their starting world |
| `scope/conform-d4.txt`, `scope/conform-d5.txt` | `spine conform` to depth 4 and 5, run 2026-10-28 with verifier-kit at `7aef1bf` and `aiwf` built from `c06ffdaf9` |
| `milestone/spec/milestone/chapter.md` | The rules for a milestone with one AC, in prose, citing the sources each rule comes from |
| `milestone/spec/milestone/readings/` | The prompt (`prompt.md`, and `filled-prompt.md` as the readers received it), two independent readings of the chapter into a spine (`reading-a.yaml`, `reading-b.yaml`), `spine check` of each, and `spine diff` of one against the other |
| `milestone/spec/milestone/answers.md` | The four questions the readings left open, as decided |
| `milestone/spec/milestone/spine.yaml` | The spine with the answers applied |
| `milestone/conformance/adapter_milestone.py`, `milestone/template.sh` | The adapter and the starting world |
| `milestone/conform-k1.log` | `spine conform` at margin 1, run 2026-11-16 with verifier-kit at `1ac00ef` |
| `milestone/repro.sh`, `milestone/repro2.sh`, `milestone/repro2.log` | The divergences reproduced with plain `aiwf` commands, and `repro2.sh`'s output |
| `milestone/sources-ac-less.md`, `milestone/ms_acs.py`, `milestone/m_hist.py` | The sources and history collected for questions 3 and 4, and the scripts behind its history counts |

Each adapter runs the `aiwf` on `PATH`. `repro2.log` records the version of the build it ran;
`conform-k1.log` records none.

Re-running, with verifier-kit checked out at `1ac00ef` in `$VK` and a scratch directory `$T`.
verifier-kit asks for a newer Go than this repository's, which `GOTOOLCHAIN=auto` fetches:

```
(cd "$VK/tools" && GOTOOLCHAIN=auto go build -o "$T/spine" ./cmd/spine)

bash scope/template.sh "$T/scope-template"
(cd scope && SCOPE_TEMPLATE="$T/scope-template" SCOPE_RUNS="$T/scope-runs" \
  "$T/spine" conform spec/scope/spine.yaml --margin 2 \
  --adapter-cmd "python3 -I $VK/tools/adapter_shim.py conformance/adapter_scope.py")

bash milestone/template.sh "$T/ms-template"
(cd milestone && MILESTONE_TEMPLATE="$T/ms-template" MILESTONE_RUNS="$T/ms-runs" \
  "$T/spine" conform spec/milestone/spine.yaml --margin 1 \
  --adapter-cmd "python3 -I $VK/tools/adapter_shim.py conformance/adapter_milestone.py")

bash milestone/repro2.sh "$T/ms-template"
```

`ms_acs.py` runs from the repository root. `m_hist.py` reads the output of the command below
from the absolute path it holds:

```
git log --reverse --date=short --all-match --grep='aiwf-entity: M-' \
  --format='@@%H|%ad|%(trailers:key=aiwf-verb,valueonly,separator=%x2C)|%(trailers:key=aiwf-entity,valueonly,separator=%x2C)|%(trailers:key=aiwf-to,valueonly,separator=%x2C)|%(trailers:key=aiwf-force,valueonly,separator=%x2C)|%(trailers:key=aiwf-reason,valueonly,separator=%x2C)'
```
