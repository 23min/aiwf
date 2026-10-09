#!/usr/bin/env python3
"""pilot-go-conventions.py — the fixture for M-0334's pilot. Run in a run's
repo by setup.sh when FIXTURE_HOOK names it: moves CLAUDE.md's Go conventions
subsections verbatim into the on-demand documents M-0349's inventory names,
keeps Commit conventions primed, gives the router M-0349's draft entries for
the documents made, and retires the pins on the moved passages."""
import os, re
c = open("CLAUDE.md").read()
a = c.index("## Go conventions\n")
b = c.index("<!-- aiwf:guidance:START")
section, rest = c[a:b], c[b:]
parts = re.split(r"\n(?=### )", section)
intro = parts[0]
subs = {p.split("\n", 1)[0][4:].strip(): p for p in parts[1:]}
home = {
    "go-conventions": ["Formatting and linting", "Error handling", "Concurrency", "CLI conventions", "Dependencies", "Naming", "Type design"],
    "testing": ["Testing", "Test design rules", "Test discipline", "Coverage"],
    "verb-design": ["Designing a new verb"],
    "release": ["Release process"],
    "shipped-content": ["Skills policy"],
    "enforcement": ["What's enforced and where"],
}
titles = {"go-conventions": "Go conventions", "testing": "Testing", "verb-design": "Verb design", "release": "Release", "shipped-content": "Shipped content", "enforcement": "Enforcement"}
assert sorted(sum(home.values(), []) + ["Commit conventions"]) == sorted(subs), sorted(subs)
def relink(text):
    return re.sub(r"\]\((?!https?:|#|/)([^)]+)\)", r"](../../\1)", text)
os.makedirs(".guidance/project", exist_ok=True)
for doc, names in home.items():
    body = "\n".join(subs[n].strip() + "\n" for n in names)
    if doc == "go-conventions":
        body = intro.split("\n", 1)[1].strip() + "\n\n" + body
    body = re.sub(r"^(#+) ", lambda m: m.group(1)[1:] + " ", body, flags=re.M)
    open(f".guidance/project/{doc}.md", "w").write(f"# {titles[doc]}\n\n" + relink(body).rstrip() + "\n")
commit = subs["Commit conventions"].replace("### Commit conventions", "## Commit conventions", 1)
open("CLAUDE.md", "w").write(c[:a] + commit.rstrip() + "\n\n" + rest)
open(".guidance/project.md", "w").write("""# Project guidance

`CLAUDE.md` holds the rules for every task in this repository, and they take
precedence over the packs. Before the first edit a task governs, read the
document that matches it:

- [Testing](project/testing.md) — writing or changing Go tests, the coverage
  gates, running tests on macOS.
- [Verb design](project/verb-design.md) — adding or changing a verb, a check
  rule or the loader.
- [Go conventions](project/go-conventions.md) — Go code in this repository;
  where it differs from the Go pack, logging included, it overrides the pack.
- [Shipped content](project/shipped-content.md) — editing embedded skills,
  rituals, templates or the guidance fragment.
- [Release](project/release.md) — cutting a release.
- [Enforcement](project/enforcement.md) — where a rule is enforced, from
  pre-commit to the release tag.
""")

# The pins on the moved passages are retired, as M-0335 retires them, so the
# host meets the suite the reduced tree would carry.
import glob
retired = ["TestPolicy_CLAUDEMDCLIConventionsLogging", "TestPolicy_M0228SkillsPolicyBroadenedPrinciple",
           "TestPolicy_M0134ClaudeMdTestRunningSections", "TestSkillEditProvenance_DocumentedInClaudeMd",
           "TestM0195_AC5_SkillBodyDisciplineInClaudeMd", "TestPolicy_ClaudeMdTestDisciplineSection"]
for f in glob.glob("internal/policies/*_test.go"):
    src = open(f).read()
    for name in list(retired):
        sig = "func " + name + "(t *testing.T) {" + chr(10)
        if sig in src:
            src = src.replace(sig, sig + chr(9) + 't.Skip("retired with the guidance passage it pinned")' + chr(10), 1)
            retired.remove(name)
    open(f, "w").write(src)
assert not retired, retired
