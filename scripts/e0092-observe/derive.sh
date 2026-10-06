#!/usr/bin/env bash
# derive.sh <run-dir> — rebuilds a run's transcript, and for Codex its
# codex-read-claude judgment, from the raw records run.sh kept.
set -euo pipefail
RUN=$1
H=$(dirname "$(readlink -f "$0")")
if head -1 "$RUN/after.txt" | grep -q '^host=codex '; then
  python3 "$H/codex_read_claude.py" "$RUN/codex-home" "$RUN/CLAUDE.at-start.md" > "$RUN/codex-read-claude.txt"
  python3 "$H/reduce.py" "$RUN/codex-home" "$RUN" > "$RUN/transcript.txt"
else
  python3 "$H/reduce.py" "$RUN/events.jsonl" "$RUN" > "$RUN/transcript.txt"
fi
