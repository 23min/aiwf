## What's missing

`aiwf list --format=json` omits the `parent` field for milestones, so a script cannot group milestones by epic without reading each file.

## Why it matters

Reporting scripts re-read every milestone file to recover a value the loader already holds.
