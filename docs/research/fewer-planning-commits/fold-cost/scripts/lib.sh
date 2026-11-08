# Shared helpers. Source with:  . lib.sh <run dir>
# Creates an isolated HOME under the run dir and puts the scratch aiwf build first on PATH.
S=/tmp/claude-1000/-workspaces-aiwf/eb2423dd-a702-4bd8-a308-1404db3d3991/scratchpad/fold-cost
W=${1:?run dir}
export HOME=$W/home
export PATH=$S/bin:$S/scripts:/usr/local/go/bin:/usr/bin:/bin
export GIT_CONFIG_NOSYSTEM=1
unset GIT_DIR GIT_WORK_TREE SSH_AUTH_SOCK
export GIT_EDITOR=true

fresh_home() {
  rm -rf "$W"; mkdir -p "$HOME"
  git config --global user.email peter@example.com
  git config --global user.name Peter
  git config --global init.defaultBranch main
  git config --global advice.detachedHead false
}

# Enable SSH commit signing with a key generated in scratch.
enable_signing() {
  mkdir -p "$HOME/.ssh"
  ssh-keygen -q -t ed25519 -N '' -C peter@example.com -f "$HOME/.ssh/id_ed25519"
  git config --global gpg.format ssh
  git config --global user.signingkey "$HOME/.ssh/id_ed25519"
  git config --global commit.gpgsign true
  echo "peter@example.com $(cat "$HOME/.ssh/id_ed25519.pub")" > "$HOME/.ssh/allowed_signers"
  git config --global gpg.ssh.allowedSignersFile "$HOME/.ssh/allowed_signers"
}

# One line per commit on the first-parent line base..HEAD: sha, signature status, subject.
show_line() { git log --first-parent --format='%h %G? %s' "$1..${2:-HEAD}"; }
