# The two contributors' work, shared by the S0 and A scenarios. Sourced after lib.sh.
# Each allocates a gap and an epic from the same base and cites them with markers:
#   alpha: G(alpha-gap) cited from G-0001's body; E(alpha-epic) + a milestone whose parent is it
#   beta:  G(beta-gap) cited from M-0001's body;  E(beta-epic) + a gap discovered_in it
work_alpha() { # run in alpha's clone, on its branch
  local g e
  g=$(addgap "Alpha gap"); e=$(addepic "Alpha epic")
  addms "$e" "Alpha ms (parent means alpha-epic)" >/dev/null
  cite G-0001 "See $g (means:alpha-gap) for Alpha's follow-up."
  echo "  alpha allocated gap=$g epic=$e"
}
work_beta() {
  local g e
  g=$(addgap "Beta gap"); e=$(addepic "Beta epic")
  addgap "Beta finding (discovered_in means beta-epic)" --discovered-in "$e" >/dev/null
  cite M-0001 "Blocked on $g (means:beta-gap)."
  echo "  beta allocated gap=$g epic=$e"
}
