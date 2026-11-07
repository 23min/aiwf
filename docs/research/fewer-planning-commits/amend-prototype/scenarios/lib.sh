# Shared helpers for the amend-prototype scenarios. Source, don't run.
# Each scenario gets a fresh temp dir with its own HOME, a bare remote
# and a repo initialised with `aiwf init`.
P=/tmp/claude-1000/-workspaces-aiwf/eb2423dd-a702-4bd8-a308-1404db3d3991/scratchpad/amend-proto
WT=/workspaces/aiwf/.claude/worktrees/patch/research-commit-noise
export PATH=$P/bin:$PATH          # hooks resolve `aiwf` via PATH: the prototype
export AIWF_EXPERIMENT_AMEND=1
unset AIWF_EXPERIMENT_AMEND_FAIL AIWF_EXPERIMENT_AMEND_DELAY

new_env() {
  T=$(mktemp -d "$P/runs/$1.XXXX")
  export HOME=$T/home; mkdir -p "$HOME"
  export GIT_CONFIG_NOSYSTEM=1
  git config --global user.email peter@example.com
  git config --global user.name Peter
  git config --global init.defaultBranch main
  git config --global advice.detachedHead false
  git config --global pull.rebase false
  echo "## environment"
  echo "date: $(date -u +%FT%TZ)"
  echo "git: $(git --version)"
  echo "aiwf binary: $P/bin/aiwf sha256=$(sha256sum "$P/bin/aiwf" | cut -c1-16)"
  echo "built from: $WT at $(git -C "$WT" rev-parse --short HEAD) + uncommitted diff sha256=$( (git -C "$WT" diff -- internal/gitops; cat "$WT/internal/gitops/amend_experiment.go") | sha256sum | cut -c1-16)"
  echo "AIWF_EXPERIMENT_AMEND=$AIWF_EXPERIMENT_AMEND HOME=$HOME"
  echo
}

# init_repo <dir> : bare remote at $T/<dir>-remote.git ($REMOTE), repo at $T/<dir>, one
# pushed code commit holding the aiwf init output.
init_repo() {
  REMOTE="$T/$1-remote.git"
  [ -e "$REMOTE" ] && { echo "init_repo: $REMOTE exists"; exit 1; }
  git init -q --bare "$REMOTE"
  git init -q "$T/$1"
  cd "$T/$1" || exit 1
  git remote add origin "$REMOTE"
  aiwf init --no-prompt >/dev/null 2>&1 || { echo "aiwf init failed"; exit 1; }
  git add -A && git commit -qm "chore: init aiwf"
  git push -q -u origin main >/dev/null 2>&1 || { echo "init_repo: initial push failed"; exit 1; }
}

# x <cmd...> : echo the command, run it, show a non-zero exit code.
x() {
  echo "\$ $*"
  "$@" 2>&1
  local rc=$?
  [ $rc -ne 0 ] && echo "[exit $rc]"
  return 0
}

# lg [args] : one line per commit with its verbs.
lg() {
  git log --format='%h %<(10,trunc)%p %s  {verbs: %(trailers:key=aiwf-verb,valueonly,separator=%x2C)}' "$@"
}

section() { echo; echo "## $*"; }

# count_ops <rev-range> : number of aiwf-verb trailers in range.
count_ops() { git log --format='%(trailers:key=aiwf-verb,valueonly)' "$@" | grep -c .; }

# ops [rev] : operation accounting (ops.py) over everything reachable from rev.
ops() { git log --format='%H%x1f%aI%x1f%(trailers:only,unfold)%x1e' "${1:-HEAD}" | python3 -I "$P/scenarios/ops.py"; }
