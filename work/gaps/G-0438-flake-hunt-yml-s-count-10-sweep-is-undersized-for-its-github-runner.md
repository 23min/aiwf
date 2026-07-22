---
id: G-0438
title: flake-hunt.yml's -count=10 sweep is undersized for its GitHub runner
status: open
---
## What's missing

`flake-hunt.yml` runs `go test -race -count=10 -parallel 8 ./...` — the entire module, repeated 10 times, race-instrumented — on a stock GitHub-hosted `ubuntu-latest` runner (2 vCPU, 7GB RAM). At that scale the runner cannot service the combined load: package-level parallelism, per-package `-parallel 8`, 10x repetition, and race-detector overhead compound into severe CPU/scheduling starvation. There's no configuration matching the workflow's actual resource ceiling — no reduced `-count`, no package subset, no bigger runner.

## Why it matters

The workflow's own header states its purpose: "run before tagging a release; if it stays green ... the tag is safe to push." A v0.28.0 release cut hit exactly this: flake-hunt failed across 4 packages (cmd/stresstest, internal/stresstest, internal/cli/doctor, internal/cli/integration) with PATH-resolution and timing failures. All 4 were confirmed clean at the identical flags on a 20-core/63GB machine, both individually and combined — the runner's resource ceiling produced the red signal, not a code defect. Every future release cut now has to redo this same manual local-repro investigation to distinguish "real regression" from "runner too small," which is exactly the judgment call this workflow exists to make automatically. Left as-is, operators either learn to reflexively distrust flake-hunt's result (defeating its purpose) or burn an investigation cycle per release.