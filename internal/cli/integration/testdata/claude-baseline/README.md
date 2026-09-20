# Claude artifact compatibility baseline

These inventories pin the complete regular-file output exercised by
`TestClaudeArtifacts_MatchBaseline`: project adapters and manifests, root
instructions, configuration, Git hooks, user settings, and optional statusline.
Each line records an octal permission mode, SHA-256 of the complete file bytes,
and a relative path. `user/` identifies the isolated fixture HOME; `git-hooks/`
identifies the repository's shared hook directory. Lines are sorted lexically.

Six scenarios cover undecided, explicitly enabled, and declined lifecycle hooks;
stored hook consent with agent tiers and guidance opt-out; and statusline
installation with and without settings-wiring consent. The test names, seed
files, configuration, and flags define each input. Personal skills, agents,
templates, settings, Git ignore entries, and root instructions are included so
the comparison also detects damage to user content.

The same inventory must match initial setup, update, and an aiwf-created Git
worktree. Only the user-owned project files are explicitly added to Git before
worktree creation; aiwf must regenerate its ignored artifacts there. User-scope
artifacts and Git hooks are shared by those checkouts.

`.claude/health.aiwf.json` is excluded: its timestamp and diagnostic messages
depend on runtime state and installed tools. No adapter bytes are normalized.
The CLI is stamped `v0.36.0` at build time to make generated version fields
deterministic and exercise tagged-version statusline consent.

## Provenance and maintenance

Captured from the production sources at commit
`488d09bfd44d2ee100f8abd45aa62872925868ad`, before renderer implementation,
using a separate filesystem capture with Node's SHA-256 and file-stat APIs.
The binary was built with:

```sh
go build -ldflags='-X github.com/23min/aiwf/internal/version.Stamp=v0.36.0' -o /tmp/claude-baseline/aiwf ./cmd/aiwf
```

To reproduce a baseline, build that revision in a disposable checkout, apply
the inputs in `TestClaudeArtifacts_MatchBaseline` to isolated consumer repos,
and inventory the files with the format above. The test does not regenerate
expected output and does not derive expectations from current embedded sources.
Review any intentional baseline replacement alongside the output change it
accepts; do not regenerate fixtures just to make a failed compatibility check
pass.

## Approved frontmatter exceptions

M-0340 permits two output changes to satisfy AC-3's valid-frontmatter contract:
the descriptions in `aiwf-area/SKILL.md` and `wf-codebase-health/SKILL.md` use
YAML folded block scalars (`description: >-` followed by the original text,
indented two spaces). Their original unquoted colon-space sequences are invalid
YAML. Description text, body bytes, paths, and modes are unchanged.

Each inventory includes these two corrected hashes. They were computed
independently of the renderer: read each source from the capture revision above,
replace only its single-line description with that block-scalar form, and hash
the resulting complete bytes with Node's SHA-256 API. Replace only the matching
two inventory lines and sort again; all other records remain the captured
baseline. Do not normalize the generated files during comparison.

| Skill | Original SHA-256 | Corrected SHA-256 |
| --- | --- | --- |
| `aiwf-area` | `512a8a10734fb6128c80aceec612a03b6d9f8764e96330ffbf735d3863231054` | `d6b141d62c26926e02f66ae841a0fedd581811b0de723538e32e88ab06043d0f` |
| `wf-codebase-health` | `c3a88b2ef4cac7f2716bb1a5997e0ddb6fe5849f6b358d575e1e1f18e47807c3` | `59ed01ac5e0dfef0d9736500319c749aa448a1ae0a045da8f36727ebc48c0b5d` |

## Codex guidance configuration exception

M-0341 adds `guidance.wire_agentsmd` to the shared config schema. The generated
config example changes only the `guidance` comment to describe both instruction
files and adds the commented `wire_agentsmd: true` field with its description.
Newly scaffolded `aiwf.yaml` files include the same two comment changes. Existing
consumer config, Claude adapters, instructions, settings, hooks, paths and modes
remain unchanged.

The six inventories change only `aiwf.example.yaml` and, in the four scenarios
that scaffold config, `aiwf.yaml`. Expectations were derived independently of the
new schema generator: capture those files using the pre-change `c2e1bf6ec` binary,
verify their hashes against the historical inventory, apply only the two textual
comment edits above, and compute SHA-256 of the complete result. The enabled-hook
scenario includes its existing hook-consent mapping. All other inventory records
remain frozen; sort the records after substituting these hashes.

| Config artifact | Original SHA-256 | Updated SHA-256 |
| --- | --- | --- |
| `aiwf.example.yaml` | `5bf701304f5b83410eee6ad1c28e4bd771bb9d762e5aab817fece1f11b78657a` | `046f0df0cf0390a3fabf39a15006172732caeba47eb7ae964dddc8d4ec8a3619` |
| Scaffolded `aiwf.yaml` | `57a8cd51e36dcd55d39be610d80c393cd77dfdc2bf32b956d71daae49b72c33b` | `73748c6a5091a363cbabb8d05b93e902cab9e2299adb889301d917b38e73e00c` |
| Scaffolded `aiwf.yaml` with hook consent | `d32092a6a30030d7c0801050657463e986e45f294c2a0d44a73b34d7a8a7f62e` | `c9f7f10ac9af60a447fe926a48ff5079c87acaf77da42f6d1ca602e2f65670a7` |

## Artifact recovery ignore exception

M-0341 adds temporary recovery receipts to the shared artifact writer. Successful
materialization removes them. The only inventory change is `.gitignore`: add
`.claude/skills/.aiwf-pending`, `.claude/agents/.aiwf-pending`, and
`.claude/templates/.aiwf-pending`, each immediately after its family's existing
`.aiwf-owned` entry. Generated Claude skills, agents, templates, manifests,
provenance, instructions, settings, hooks and their modes remain frozen.

The expected hash was derived independently of the new writer: capture
`.gitignore` with the pre-change `a195023ca` binary in a fresh consumer seeded
with `user-cache/` plus a newline, verify its historical SHA-256, insert only the
three lines above, and hash the complete bytes using Node's SHA-256 API. Replace
only the `.gitignore` record in all six inventories and sort them again.

| Artifact | Original SHA-256 | Updated SHA-256 |
| --- | --- | --- |
| `.gitignore` | `1154158ecbaa05116b8e8b5ff9b3f3f9d0a1ade84bbd4c5bba68d51ef4198457` | `8d01c4e9083f5ad84582127a76e688b0eed4b2dd674a8f8b3a30f0f1bff0e210` |

## Host selection configuration exception

M-0342 documents automatic detection, explicit host selection and the empty-list
override in the generated `hosts` comment. Only that comment changes in
`aiwf.example.yaml` and in newly scaffolded `aiwf.yaml`; existing consumer
configuration and all Claude artifact bytes remain frozen.

Expectations were derived from a `v0.36.0`-stamped binary built from production
sources at `a6e54b8e3`. Capture the undecided and enabled-hook configurations,
verify their complete hashes against the existing inventories, replace the
single host-description comment with the new description, and hash the result.
This textual transformation is independent of the changed schema generator.
Substitute only these records in the six inventories and sort them again; do
not recapture output from the new implementation.

| Config artifact | Original SHA-256 | Updated SHA-256 |
| --- | --- | --- |
| `aiwf.example.yaml` | `046f0df0cf0390a3fabf39a15006172732caeba47eb7ae964dddc8d4ec8a3619` | `6977f2ad7ff8191f8e32ee351119bfb303fae2fdc49f4c9a8136100c72ece789` |
| Scaffolded `aiwf.yaml` | `73748c6a5091a363cbabb8d05b93e902cab9e2299adb889301d917b38e73e00c` | `855415b8a161438817c332e8b07535f2fe29779140268f35a429bbf576c9c5ed` |
| Scaffolded `aiwf.yaml` with hook consent | `c9f7f10ac9af60a447fe926a48ff5079c87acaf77da42f6d1ca602e2f65670a7` | `daea2561b4f7955b6d676b0ded324e6d915fe7495c8438b23c2c3b0c814ef4b9` |

## Shared worktree instruction correction

The E-0093 worktree-guidance clarification intentionally changes only
`.claude/skills/aiwf-worktree/SKILL.md` in the six Claude inventories. It
qualifies ignored-artifact availability and selected-host setup, distinguishes
shell entry from assistant session entry, and guards shell composition against
creation failure. Claude-specific entry fragments remain unchanged; Codex's
managed-worktree boundary and session-loading instructions stay in its own
fragment.

Compute the expected bytes independently of the renderer by substituting the
shared skill's sole `host_label` token with `Claude Code`, then hash the complete
file. Verify the old source hash against each inventory before replacing only
that skill's record. All other paths, modes and hashes remain frozen.

| Artifact | Original SHA-256 | Updated SHA-256 |
| --- | --- | --- |
| `aiwf-worktree/SKILL.md` | `57c31fce28de1b12753c9f026eea3eedd503f131c1e3bab6c0be82cb666dd58f` | `194891ebbedce46737533f303c2f22f989b0d4acfa035090afce6ff9f2602aa1` |

## Patch review instruction exception

G-0694 changes `wf-patch` review dispatch and mutation-harness instructions.
Each inventory replaces only `.claude/skills/wf-patch/SKILL.md` and is sorted
again. The expected hash was derived by verifying the pre-change Claude adapter
against the stored hash, applying only the reviewed step 6 edit to those bytes,
and hashing the result. All other records remain unchanged.
