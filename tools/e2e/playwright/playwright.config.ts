import { defineConfig, devices } from "@playwright/test";

// Playwright config for the aiwf HTML-render e2e suite.
//
// The suite renders a fixture tree via the aiwf binary into a temp
// directory, then drives a headless Chromium against `file://` URLs.
// No web server is needed — the static-site render is exactly what
// the consumer publishes.
//
// Chromium-only by intent: aiwf's renderer is hand-written HTML +
// CSS with no framework, no JS. Cross-browser smoke is the next
// step if real-use friction surfaces a Firefox/WebKit-specific
// rendering issue.
export default defineConfig({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? "github" : "list",
  use: {
    // The fixture sets baseURL per-test via test.use({ baseURL })
    // because each test gets its own freshly-rendered out dir.
    headless: true,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
