// fixture.ts — shared helpers for the aiwf HTML-render e2e suite.
//
// renderFixture() builds the cmd binary (once per test process),
// scaffolds a populated planning tree in a tmp dir, and renders to
// a separate tmp out dir. Returns the out dir so each test can
// point its baseURL at file://<out>/.
//
// Fixture content is deliberately rich: 2 epics, 2 milestones, 2
// ACs, a status promotion, a full red→green→done phase cycle with
// an aiwf-tests trailer, and an open authorize scope. That covers
// every populated-branch the templates carry; pages with empty
// states are exercised in separate tests by truncating which
// commands run.

import { execFileSync, type ExecFileSyncOptions } from "node:child_process";
import { mkdtempSync, mkdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

/** Repo root — three levels up from this file (tools/e2e/playwright/). */
export const repoRoot = resolve(__dirname, "..", "..", "..");

/** Path the test process re-uses across tests for the built binary. */
let cachedBin: string | null = null;

/** Build (once) the aiwf binary into a fresh tmp dir; return its path. */
export function buildAiwf(): string {
  if (cachedBin) return cachedBin;
  const dir = mkdtempSync(join(tmpdir(), "aiwf-e2e-bin-"));
  const bin = join(dir, "aiwf");
  execFileSync("go", ["build", "-o", bin, "./tools/cmd/aiwf"], {
    cwd: repoRoot,
    stdio: "pipe",
  });
  cachedBin = bin;
  return bin;
}

/**
 * Run a verb against a fixture repo. Each invocation is its own
 * subprocess so the binary path is exercised end-to-end (the
 * production target, not the in-process `run()` shape).
 */
function runAiwf(bin: string, repoDir: string, ...args: string[]): string {
  const opts: ExecFileSyncOptions = {
    stdio: "pipe",
    env: {
      ...process.env,
      GIT_AUTHOR_NAME: "aiwf-e2e",
      GIT_AUTHOR_EMAIL: "e2e@example.com",
      GIT_COMMITTER_NAME: "aiwf-e2e",
      GIT_COMMITTER_EMAIL: "e2e@example.com",
    },
  };
  const out = execFileSync(bin, args, { ...opts, cwd: repoDir });
  return out.toString();
}

function runGit(repoDir: string, ...args: string[]): void {
  execFileSync("git", args, {
    cwd: repoDir,
    stdio: "pipe",
    env: {
      ...process.env,
      GIT_AUTHOR_NAME: "aiwf-e2e",
      GIT_AUTHOR_EMAIL: "e2e@example.com",
      GIT_COMMITTER_NAME: "aiwf-e2e",
      GIT_COMMITTER_EMAIL: "e2e@example.com",
    },
  });
}

/**
 * Build a populated fixture and render it to HTML. Returns the
 * absolute path of the output directory.
 *
 * The shape: 2 epics, 1st with 2 milestones; M-001 has 2 ACs (one
 * promoted to met, one walked through red→green→done with
 * aiwf-tests). M-002 has an open authorize scope. Enough surface
 * to populate every page-template branch except `--force` and
 * `--audit-only` (those are tested by their own narrower fixtures).
 */
export function renderRichFixture(): string {
  const bin = buildAiwf();
  const repoDir = mkdtempSync(join(tmpdir(), "aiwf-e2e-repo-"));
  mkdirSync(repoDir, { recursive: true });
  runGit(repoDir, "init", "-q");
  runGit(repoDir, "config", "user.email", "e2e@example.com");
  runGit(repoDir, "config", "user.name", "e2e");
  // Disable hooks: aiwf init writes pre-push and pre-commit; the
  // pre-commit hook re-execs the aiwf binary on every commit which
  // slows things and risks reentrancy under e2e.
  runGit(repoDir, "config", "core.hooksPath", "/var/empty");

  runAiwf(bin, repoDir, "init", "--actor", "human/peter");

  const verbs: string[][] = [
    ["add", "epic", "--title", "Foundations", "--actor", "human/peter"],
    ["add", "epic", "--title", "Adoption", "--actor", "human/peter"],
    ["add", "milestone", "--epic", "E-01", "--title", "Schema parser", "--actor", "human/peter"],
    ["add", "milestone", "--epic", "E-01", "--title", "Tree loader", "--actor", "human/peter"],
    ["add", "ac", "M-001", "--title", "Parses YAML frontmatter", "--actor", "human/peter"],
    ["add", "ac", "M-001", "--title", "Reports parse errors", "--actor", "human/peter"],
    ["promote", "M-001/AC-1", "met", "--actor", "human/peter"],
    ["promote", "M-001/AC-2", "--phase", "red", "--actor", "human/peter"],
    ["promote", "M-001/AC-2", "--phase", "green", "--tests", "pass=12 fail=0 skip=1", "--actor", "human/peter"],
    ["promote", "M-001/AC-2", "--phase", "done", "--actor", "human/peter"],
    ["promote", "M-001", "in_progress", "--actor", "human/peter"],
    // Promote E-01 to active so the status page's in-flight block
    // has something to render. buildStatus only lists active epics.
    ["promote", "E-01", "active", "--actor", "human/peter"],
    ["authorize", "M-002", "--to", "ai/claude", "--actor", "human/peter"],
  ];
  for (const args of verbs) {
    runAiwf(bin, repoDir, ...args);
  }

  const outDir = mkdtempSync(join(tmpdir(), "aiwf-e2e-site-"));
  runAiwf(bin, repoDir, "render", "--format", "html", "--out", outDir);
  return outDir;
}
