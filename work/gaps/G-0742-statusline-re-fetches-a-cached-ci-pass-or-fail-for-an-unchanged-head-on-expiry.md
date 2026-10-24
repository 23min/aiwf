---
id: G-0742
title: Statusline re-fetches a cached CI pass or fail for an unchanged HEAD on expiry
status: open
---
## What's missing

The statusline's CI probe treats a cached `✓` or `✗` for the commit still checked
out the same way it treats a cached `→` (a run in progress). Once the entry is 45
seconds old, the next render calls GitHub again. A `→` needs that call, because its
verdict is expected to change. A `✓` or `✗` for an unchanged HEAD changes only when
a workflow is re-run or a new run starts on that commit. This repository has
triggers that start new runs: `fuzz.yml` runs on a schedule and can be started by
hand, and `flake-hunt.yml` and `mutate-hunt.yml` can be started by hand.

This is in the CI block of `internal/skills/embedded-statusline/statusline.sh`, the
shipped script that every `aiwf init --statusline` consumer materializes. The cache
key includes HEAD (line 300), but the only test for a cache hit is age: `ttl=45`
(line 305) and `[ "$age" -lt "$ttl" ] && use_cache=true` (line 333). On a miss,
`gh run list` runs (line 317). When that fetch fails, the result becomes `?`
(line 351) and is written over the cached verdict (line 354).

Measured on 2026-10-24 in the aiwf devcontainer at `main` `b33f745a1`, against the
source script, with a stub `gh` that logs each call:

```bash
S=<scratch dir>; mkdir -p $S/bin $S/cache
HEAD_SHA=$(git rev-parse HEAD)
cat > $S/bin/gh <<EOF
#!/usr/bin/env bash
echo "\$(date +%T) gh \$*" >> $S/gh.log
[ -f $S/offline ] && exit 1
printf '[{"conclusion":"success","status":"completed","headSha":"$HEAD_SHA"}]'
EOF
chmod +x $S/bin/gh
render() { PATH=$S/bin:$PATH AIWF_STATUSLINE_CACHE_DIR=$S/cache \
  bash internal/skills/embedded-statusline/statusline.sh < stdin.json \
  | sed 's/\x1b\[[0-9;]*m//g' | grep -o '[^·]*ci$'; }
render; render                                              # cold, then fresh cache
touch -d '-50 seconds' $S/cache/aiwf-statusline-ci-*; render   # aged, gh answering
touch -d '-50 seconds' $S/cache/aiwf-statusline-ci-*; touch $S/offline
render; render                                              # aged, then fresh; gh failing
cat $S/gh.log $S/cache/aiwf-statusline-ci-*
```

`stdin.json` is a minimal Claude Code session object (model, context_window,
workspace.current_dir). The run echoed each label before its render. Output:

```
render 1 (cold):                                          ✓ ci
render 2 (fresh cache):                                   ✓ ci
render 3 (cache aged 50s, same HEAD, verdict already ✓):  ✓ ci
render 4 (cache aged 50s, gh failing):                    ? ci
render 5 (fresh cache after failure, gh still failing):   ? ci
--- gh calls:
10:55:46 gh run list --branch main --limit 30 --json conclusion,status,headSha
10:55:47 gh run list --branch main --limit 30 --json conclusion,status,headSha
10:55:47 gh run list --branch main --limit 30 --json conclusion,status,headSha
--- cache:
|?
```

Renders 1, 3 and 4 each called GitHub. Renders 3 and 4 asked about a commit whose
verdict was already cached as `✓`.

## Why it matters

While a session is active, the statusline calls GitHub on the first render after
each 45-second expiry, about once every 45 seconds per checkout, even when the
cached verdict is a `✓` or `✗` for the commit still checked out. Parallel worktrees
each make their own calls. Because of this, a statusline `refreshInterval` timer
cannot be enabled without polling GitHub from a session sitting idle on a finished
commit.

A failed fetch replaces a cached `✓` or `✗` with `?`. If the machine is offline, or
the `gh` call fails for any other reason, the verdict the cache held for HEAD turns
into `? ci` once the entry expires. It stays `?` until a fetch succeeds.

## Related

G-0554 concerns the same cache key and write block in `statusline.sh`.
