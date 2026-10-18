package policies

// m0209GuidanceFixturePath is the canonical authoring location for the
// per-turn LLM guidance fragment (ADR-0018). `.claude/aiwf-guidance.md` in a
// consumer repo is materialized from these embedded bytes by `aiwf init` /
// `aiwf update`, so tests read the source, never the gitignored rendered
// artifact.
const m0209GuidanceFixturePath = "internal/skills/embedded-guidance/aiwf-guidance.md"
