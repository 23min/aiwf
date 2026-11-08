# Evidence for "Fewer planning commits"

The scripts, outputs and reports behind [../fewer-planning-commits.md](../fewer-planning-commits.md).
Everything ran on 2026-11-07, `reflog-rule/` on 2026-11-08 (git 2.51.1, Go 1.25.12, Python 3.12.3, Linux
devcontainer) against this repository at `4343f2bd1`. Scripts hold the absolute
paths they ran with (`/workspaces/aiwf` and a scratch directory); adjust them to
re-run.

| Path | What it holds |
|---|---|
| `reports/00-options-brief.md` | The candidates and first measurements, as handed to the reviewers |
| `reports/01-independent-review.md` | The first independent review: corrected counts, the corrected amend rule, measured and argued risks; its scripts are in `review/`, its experiments are command sequences recorded in the report |
| `reports/02-history-measurements.md` | Replay of the window under amend, fold and batch; SHA references; branch cuts |
| `reports/03-amend-prototype.md` | The amend prototype, ten git scenarios, the risk register |
| `reports/04-fold-cost.md` | What re-creating code commits breaks; the side-ref variant's hazards |
| `reports/05-model-amend.md` | The transactions model with amend designs (results in `../transactions/model/RESULTS-amend.md`) |
| `history/` | Replay scripts: `extract.py` builds `data.json` from the repository (not kept; about 2 MB); `base.py` is the shared model; `amend.py`, `foldbatch.py`, `refs_*.py`, `cuts.py`, `syncmerges.py` replay; `ordering_variants.py` and `ordering_analysis.py` give the ritual-order variants and the run breakdowns; `render.py` prints the history views. `out/` holds their outputs |
| `history/zz_histmeasure_dump_test.go.txt`, `history/overlay.json` | A scratch Go test, used through `go test -overlay`, that dumps the registered and ritual verb sets into `verbsets.json` (kept). It is stored as `.txt` so the module does not compile it; `overlay.json` names the path it ran from and has to point at a `.go` copy to re-run |
| `amend-prototype/amend-experiment.patch` | The prototype, behind `AIWF_EXPERIMENT_AMEND=1`; apply to `4343f2bd1` to rebuild it |
| `amend-prototype/scenarios/`, `amend-prototype/runs/` | The ten scenario scripts (`lib.sh`, `ops.py`, `s01`–`s10`) and each one's output |
| `amend-prototype/gotest-*.log` | `go test ./internal/gitops/... ./internal/verb/...` with the experiment off and on |
| `fold-cost/scripts/`, `fold-cost/out/` | The plumbing fold, the fork, signing and intermediate-state probes, the side-ref simulation, the pre-push test (`prepush-cannot-fold.sh`), and their transcripts (`signing.txt` re-reads the signing runs' signature status) |
| `review/` | The first review's replay scripts |
| `reflog-rule/scripts/`, `reflog-rule/out/` | The reflog clause, run 2026-11-08: which git commands write HEAD's reflog and the rebase hole in a scratch repository (`reflog-probe.sh`), a patch-id test's cost against the reflog read (`patchid-cost.sh`), and the runs the clause would split in this clone's main checkout (`reflog-splits.sh`, `checkouts.sh`); each takes the repository or scratch directory as its argument |

Re-running, from a copy of `history/`:

```
python3 -I extract.py /workspaces/aiwf .                         # data.json
python3 -I refs_extract.py                                       # refs.json; amend.py reads it
python3 -I amend.py --dump amend_a.json                          # amend, history as is
python3 -I amend.py --drop-roadmap --dump amend_b.json           # amend, without ROADMAP-only commits
python3 -I foldbatch.py --dump fold_a.json                       # fold (F1, F2) and batch
python3 -I foldbatch.py --drop-roadmap --dump fold_b.json
python3 -I refs_analysis.py fold_a.json amend_a.json             # SHA references a fold breaks
python3 -I ordering_variants.py                                  # ritual-order variants; writes variant_*.json
python3 -I ordering_analysis.py                                  # run breakdowns, batch exposure, duplicates
python3 -I render.py                                             # history views; needs amend_b.json
python3 -I explore1.py data.json verbsets.json                   # census
```
