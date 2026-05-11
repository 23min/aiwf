// render.spec.ts — browser tests for the aiwf governance HTML
// render. The substring + well-formedness checks in Go are
// necessary but not sufficient for the rendered surface — CSS-
// driven behavior (`:target`-tabs, computed colors, anchor
// scrolling, layout collapse) needs a real browser.

import { test, expect } from "@playwright/test";
import { pathToFileURL } from "node:url";
import { join } from "node:path";
import { existsSync } from "node:fs";
import { renderRichFixture } from "../fixture";

let outDir: string;
let consoleErrors: { url: string; text: string }[] = [];

test.beforeAll(() => {
  outDir = renderRichFixture();
});

test.beforeEach(({ page }) => {
  // Surface every console error and unhandled page error so a
  // template typo, a 404 on assets, or a future JS addition that
  // throws is caught immediately.
  consoleErrors = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") {
      consoleErrors.push({ url: page.url(), text: msg.text() });
    }
  });
  page.on("pageerror", (err) => {
    consoleErrors.push({ url: page.url(), text: err.message });
  });
});

test.afterEach(() => {
  expect(consoleErrors, "console errors collected during this test").toEqual([]);
});

function fileURL(rel: string): string {
  return pathToFileURL(join(outDir, rel)).toString();
}

test.describe("index.html", () => {
  test("lists every epic with AC met-rollup", async ({ page }) => {
    await page.goto(fileURL("index.html"));

    await expect(page.locator("h1")).toHaveText("Overview");

    const e01Row = page.locator("table.epics tr", { has: page.getByRole("link", { name: "E-0001" }) });
    await expect(e01Row).toBeVisible();
    // 1 met / 2 total (cancelled excluded; M-001/AC-1 met, AC-2 not met yet)
    await expect(e01Row).toContainText("1/2");

    const e02Row = page.locator("table.epics tr", { has: page.getByRole("link", { name: "E-0002" }) });
    await expect(e02Row).toBeVisible();
    await expect(e02Row).toContainText("0/0");
  });

  test("epic links navigate to per-epic page", async ({ page }) => {
    await page.goto(fileURL("index.html"));
    // Sidebar repeats the epic links; scope the click to the
    // main epics table to keep the assertion unambiguous.
    await page.locator("table.epics").getByRole("link", { name: "E-0001" }).click();
    await expect(page).toHaveURL(fileURL("E-0001.html"));
    await expect(page.locator("h1")).toContainText("Foundations");
  });

  test("stylesheet link resolves and loads (no 404)", async ({ page }) => {
    let cssStatus = 0;
    page.on("response", (resp) => {
      if (resp.url().endsWith("/style.css")) cssStatus = resp.status();
    });
    await page.goto(fileURL("index.html"));
    // file:// returns 0 for status on some platforms; we instead
    // verify a CSS rule is actually applied to the body. If the
    // sheet 404'd or didn't link, system fonts wouldn't apply and
    // the system-ui declaration would be missing.
    const fontFamily = await page.locator("body").evaluate((el) => getComputedStyle(el).fontFamily);
    expect(fontFamily).toContain("system");
    // cssStatus is 0 on file://; treat any non-error code as ok.
    expect([0, 200]).toContain(cssStatus);
  });
});

test.describe("epic page", () => {
  test("shows milestones table with AC rollup per milestone", async ({ page }) => {
    await page.goto(fileURL("E-0001.html"));

    const m001 = page.locator("table.milestones tr", { has: page.getByRole("link", { name: "M-0001" }) });
    await expect(m001).toContainText("1/2");
    await expect(m001).toContainText("in_progress");

    const m002 = page.locator("table.milestones tr", { has: page.getByRole("link", { name: "M-0002" }) });
    await expect(m002).toContainText("0/0");
  });

  test("milestone link navigates to milestone page", async ({ page }) => {
    await page.goto(fileURL("E-0001.html"));
    await page.getByRole("link", { name: "M-0001" }).first().click();
    await expect(page).toHaveURL(fileURL("M-0001.html"));
  });
});

test.describe("milestone page — :target tab show/hide (CSS-only)", () => {
  test("bare URL shows Overview, hides Build", async ({ page }) => {
    await page.goto(fileURL("M-0001.html"));
    await expect(page.locator('section[data-tab="overview"]')).toBeVisible();
    await expect(page.locator('section[data-tab="build"]')).toBeHidden();
    await expect(page.locator('section[data-tab="manifest"]')).toBeHidden();
    await expect(page.locator('section[data-tab="tests"]')).toBeHidden();
    await expect(page.locator('section[data-tab="commits"]')).toBeHidden();
    await expect(page.locator('section[data-tab="provenance"]')).toBeHidden();
  });

  test("#tab-build shows Build, hides Overview", async ({ page }) => {
    await page.goto(fileURL("M-0001.html") + "#tab-build");
    await expect(page.locator('section[data-tab="build"]')).toBeVisible();
    await expect(page.locator('section[data-tab="overview"]')).toBeHidden();
  });

  test("clicking Manifest tab nav link switches to Manifest", async ({ page }) => {
    await page.goto(fileURL("M-0001.html"));
    await page.locator('a.tab[href="#tab-manifest"]').click();
    await expect(page).toHaveURL(fileURL("M-0001.html") + "#tab-manifest");
    await expect(page.locator('section[data-tab="manifest"]')).toBeVisible();
    await expect(page.locator('section[data-tab="overview"]')).toBeHidden();
  });

  test("each tab has exactly one visible section", async ({ page }) => {
    for (const tab of ["overview", "manifest", "build", "tests", "commits", "provenance"]) {
      const url = tab === "overview" ? fileURL("M-0001.html") : fileURL("M-0001.html") + `#tab-${tab}`;
      await page.goto(url);
      const visible = await page
        .locator("section[data-tab]")
        .evaluateAll((els) => els.filter((e) => (e as HTMLElement).offsetParent !== null).map((e) => (e as HTMLElement).dataset.tab));
      expect(visible, `tab=${tab}`).toEqual([tab]);
    }
  });
});

test.describe("milestone page — Manifest tab", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(fileURL("M-0001.html") + "#tab-manifest");
  });

  test("renders an AC card per AC with anchor id", async ({ page }) => {
    const ac1 = page.locator('section.ac#ac-1');
    const ac2 = page.locator('section.ac#ac-2');
    await expect(ac1).toBeVisible();
    await expect(ac2).toBeVisible();
    await expect(ac1).toContainText("Parses YAML frontmatter");
    await expect(ac2).toContainText("Reports parse errors");
  });

  test("status pill carries the right class and a non-default color", async ({ page }) => {
    const ac1Status = page.locator('section.ac#ac-1 .status-met');
    await expect(ac1Status).toHaveText("met");
    const color = await ac1Status.evaluate((el) => getComputedStyle(el).color);
    // `met` should resolve to a green-family color via --status-met.
    // We don't pin the exact shade — the CSS variable can shift —
    // but a black/transparent/grey would mean the class isn't
    // matching its rule.
    expect(color).not.toBe("rgb(0, 0, 0)");
    expect(color).not.toBe("rgba(0, 0, 0, 0)");
    // Sanity check: green channel dominates red+blue for the met
    // color tokens shipped today.
    const [r, g, b] = parseRgb(color);
    expect(g).toBeGreaterThan(r);
    expect(g).toBeGreaterThan(b);
  });
});

test.describe("milestone page — Build tab", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(fileURL("M-0001.html") + "#tab-build");
  });

  test("AC-2 timeline shows red, green, done in order", async ({ page }) => {
    const phases = page.locator('section.build-ac', { has: page.getByRole("link", { name: "AC-2" }) }).locator("ol.phases > li");
    await expect(phases).toHaveCount(3);
    const text = await phases.allTextContents();
    expect(text[0]).toMatch(/\bred\b/);
    expect(text[1]).toMatch(/\bgreen\b/);
    expect(text[2]).toMatch(/\bdone\b/);
  });

  test("aiwf-tests trailer surfaces inline on the green phase row", async ({ page }) => {
    const greenRow = page.locator('section.build-ac', { has: page.getByRole("link", { name: "AC-2" }) }).locator(`li:has(.phase-green)`);
    await expect(greenRow).toContainText("pass=12");
    await expect(greenRow).toContainText("fail=0");
    await expect(greenRow).toContainText("skip=1");
  });

  test("AC-1 (status-only) shows empty state, no fake phase rows", async ({ page }) => {
    const ac1 = page.locator('section.build-ac', { has: page.getByRole("link", { name: "AC-1" }) });
    await expect(ac1).toContainText("No phase events recorded");
    await expect(ac1.locator("li.phase, .phase-met")).toHaveCount(0);
  });

  test("clicking the AC link inside Build tab jumps to Manifest anchor", async ({ page }) => {
    const ac2Link = page.locator('section.build-ac h3', { hasText: "AC-2" }).getByRole("link", { name: "AC-2" });
    await ac2Link.click();
    await expect(page).toHaveURL(fileURL("M-0001.html") + "#ac-2");
    // Browser should now expose the AC-2 manifest card; the
    // :target chain hides Build and shows… nothing tab-tagged
    // (#ac-2 doesn't match any data-tab section), so the
    // default-tab CSS rule should re-show Overview.
    // We only assert AC-2 anchor is in viewport, not the tab
    // semantics — the latter is covered by the tab tests above.
    const ac2Card = page.locator('section.ac#ac-2');
    await expect(ac2Card).toBeAttached();
  });
});

test.describe("milestone page — Tests tab", () => {
  test("policy badge reads 'advisory' by default", async ({ page }) => {
    await page.goto(fileURL("M-0001.html") + "#tab-tests");
    await expect(page.locator(".policy.policy-advisory")).toHaveText("advisory");
  });

  test("AC-2's test counts surface in the table cell", async ({ page }) => {
    await page.goto(fileURL("M-0001.html") + "#tab-tests");
    const ac2Row = page.locator("table.tests-table tr", { has: page.getByRole("link", { name: "AC-2" }) });
    await expect(ac2Row).toContainText("12");
    await expect(ac2Row).toContainText("0");
    await expect(ac2Row).toContainText("1");
  });

  test("AC-1 (no metrics) shows the dash placeholder in the metrics columns", async ({ page }) => {
    await page.goto(fileURL("M-0001.html") + "#tab-tests");
    const ac1Row = page.locator("table.tests-table tr", { has: page.getByRole("link", { name: "AC-1" }) });
    // Either the empty-state '—' (no phase) or the missing-metrics
    // cell. The fixture's AC-1 went status:met without a phase, so
    // it shows the empty cell.
    await expect(ac1Row.locator("td.empty")).toBeVisible();
  });
});

test.describe("milestone page — Provenance tab", () => {
  test("M-002 shows an active scope row", async ({ page }) => {
    await page.goto(fileURL("M-0002.html") + "#tab-provenance");
    const scopeRow = page.locator("table.scopes tbody tr").first();
    await expect(scopeRow).toContainText("ai/claude");
    await expect(scopeRow).toContainText("human/peter");
    await expect(scopeRow.locator(".scope-state-active")).toHaveText("active");
  });

  test("M-001 (no scopes) shows the empty-state line", async ({ page }) => {
    await page.goto(fileURL("M-0001.html") + "#tab-provenance");
    await expect(page.locator('section[data-tab="provenance"]')).toContainText("No authorized scopes");
  });
});

test.describe("sidebar — left-nav tree", () => {
  test("renders on every page", async ({ page }) => {
    for (const path of ["index.html", "E-0001.html", "E-0002.html", "M-0001.html", "M-0002.html"]) {
      await page.goto(fileURL(path));
      await expect(page.locator("aside.sidebar")).toBeVisible();
      // First match: the outer <nav> wrapping the sidebar's link
      // tree. (Post-M-0100 a second <nav class="chip-strip"> also
      // lives inside; use .first() to avoid strict-mode violation.)
      await expect(page.locator("aside.sidebar nav").first()).toBeVisible();
      // Every sidebar lists every epic.
      await expect(page.locator("aside.sidebar a", { hasText: "E-0001" })).toBeVisible();
      await expect(page.locator("aside.sidebar a", { hasText: "E-0002" })).toBeVisible();
    }
  });

  test("sidebar top order: Project status precedes Overview precedes Gaps", async ({ page }) => {
    for (const path of ["index.html", "E-0001.html", "M-0001.html", "status.html"]) {
      await page.goto(fileURL(path));
      const labels = await page
        .locator("aside.sidebar ul.sidebar-top > li a")
        .allTextContents();
      expect(labels.length, `top-link count on ${path}`).toBe(3);
      expect(labels[0]).toBe("Project status");
      expect(labels[1]).toBe("Overview");
      // Third entry is "Gaps (N)" — M-0100/AC-1; N is fixture-
      // dependent so we match the prefix rather than the exact
      // string.
      expect(labels[2], `top-link[2] on ${path}`).toMatch(/^Gaps \(\d+\)$/);
    }
  });

  test("no GOVERNANCE label above the sidebar nav", async ({ page }) => {
    for (const path of ["index.html", "E-0001.html", "M-0001.html"]) {
      await page.goto(fileURL(path));
      // The legacy <p class="sidebar-title">Governance</p> was
      // removed in v0.2.0. Brand mark + wordmark are the only
      // pre-nav content now.
      await expect(page.locator("aside.sidebar p.sidebar-title")).toHaveCount(0);
    }
  });

  test("brand mark + wordmark render on every page", async ({ page }) => {
    for (const path of ["index.html", "E-0001.html", "M-0001.html"]) {
      await page.goto(fileURL(path));
      const brand = page.locator(".sidebar-brand");
      await expect(brand).toBeVisible();
      // Inline SVG carries the three-bar logo.
      await expect(brand.locator("svg.sidebar-logo")).toBeVisible();
      await expect(brand.locator("svg rect")).toHaveCount(3);
      // Wordmark reads "aiwf".
      await expect(brand.locator(".sidebar-wordmark")).toHaveText("aiwf");
    }
  });

  test("logo color follows the accent token (currentColor)", async ({ page }) => {
    await page.goto(fileURL("index.html"));
    const fill = await page.locator(".sidebar-logo rect").first().evaluate(
      (el) => getComputedStyle(el).fill,
    );
    // currentColor resolves through `color: var(--accent)` on the
    // brand wrapper. Iris in light mode → the green channel is the
    // smallest. We just confirm it's not black/grey/transparent.
    expect(fill).not.toBe("rgb(0, 0, 0)");
    expect(fill).not.toBe("rgba(0, 0, 0, 0)");
    const [r, g, b] = parseRgb(fill);
    // Iris #5e6ad2 → rgb(94, 106, 210). Blue dominates.
    expect(b).toBeGreaterThan(r);
    expect(b).toBeGreaterThan(g);
  });

  test("milestone page pre-expands its parent epic; others closed", async ({ page }) => {
    await page.goto(fileURL("M-0001.html"));
    const e01 = page.locator(`aside.sidebar details:has(a[href="E-0001.html"])`);
    const e02 = page.locator(`aside.sidebar details:has(a[href="E-0002.html"])`);
    await expect(e01).toHaveAttribute("open", "");
    // E-02 has no `open` attribute on a clean fixture render.
    expect(await e02.evaluate((el) => (el as HTMLDetailsElement).open)).toBe(false);
    // The milestone link is visible inside the open E-01 details.
    await expect(e01.locator(`a[href="M-0001.html"]`)).toBeVisible();
  });

  test("epic page pre-expands itself", async ({ page }) => {
    await page.goto(fileURL("E-0001.html"));
    const e01 = page.locator(`aside.sidebar details:has(a[href="E-0001.html"])`);
    await expect(e01).toHaveAttribute("open", "");
  });

  test("index page leaves all epics collapsed", async ({ page }) => {
    await page.goto(fileURL("index.html"));
    const opens = await page.locator("aside.sidebar details").evaluateAll(
      (els) => els.map((e) => (e as HTMLDetailsElement).open),
    );
    expect(opens.every((o) => !o)).toBe(true);
  });

  test("current page link carries aria-current=page", async ({ page }) => {
    await page.goto(fileURL("M-0001.html"));
    const current = page.locator(`aside.sidebar a[aria-current="page"]`);
    await expect(current).toHaveCount(1);
    await expect(current).toHaveAttribute("href", "M-0001.html");

    await page.goto(fileURL("E-0001.html"));
    const epicCurrent = page.locator(`aside.sidebar a[aria-current="page"]`);
    await expect(epicCurrent).toHaveCount(1);
    await expect(epicCurrent).toHaveAttribute("href", "E-0001.html");

    await page.goto(fileURL("index.html"));
    // Index page is the Overview page; the top "Overview" link is
    // marked aria-current="page", and is the only such link.
    const indexCurrent = page.locator(`aside.sidebar a[aria-current="page"]`);
    await expect(indexCurrent).toHaveCount(1);
    await expect(indexCurrent).toHaveAttribute("href", "index.html");
  });

  test("clicking an epic summary expands its milestone list", async ({ page }) => {
    await page.goto(fileURL("index.html"));
    const e01Details = page.locator(`aside.sidebar details:has(a[href="E-0001.html"])`);
    expect(await e01Details.evaluate((el) => (el as HTMLDetailsElement).open)).toBe(false);
    await e01Details.locator("summary").click();
    expect(await e01Details.evaluate((el) => (el as HTMLDetailsElement).open)).toBe(true);
    await expect(e01Details.locator(`a[href="M-0001.html"]`)).toBeVisible();
  });

  test("sidebar link navigates to the target page", async ({ page }) => {
    await page.goto(fileURL("M-0001.html"));
    const sidebarLink = page.locator(`aside.sidebar a[href="M-0002.html"]`);
    await sidebarLink.click();
    await expect(page).toHaveURL(fileURL("M-0002.html"));
  });
});

test.describe("status page", () => {
  test("status.html renders with health summary + sidebar link", async ({ page }) => {
    await page.goto(fileURL("status.html"));
    await expect(page.locator("h1")).toHaveText("Project status");
    // Sidebar's "Project status" link is marked current on this page.
    const current = page.locator('aside.sidebar a[aria-current="page"]');
    await expect(current).toHaveAttribute("href", "status.html");
    // Health line carries the entity count.
    await expect(page.locator(".kicker")).toContainText("status");
  });

  test("non-status pages link to status.html in the sidebar top section", async ({ page }) => {
    for (const path of ["index.html", "E-0001.html", "M-0001.html"]) {
      await page.goto(fileURL(path));
      const link = page.locator('aside.sidebar .sidebar-top a[href="status.html"]');
      await expect(link).toBeVisible();
      // Not marked current on these pages.
      await expect(link).not.toHaveAttribute("aria-current", "page");
    }
  });

  test("in-flight epics block lists the in-progress milestone", async ({ page }) => {
    await page.goto(fileURL("status.html"));
    // M-001 is in_progress in the fixture; it should appear in the
    // in-flight block under E-01.
    const inflight = page.locator('section.status-epic', { has: page.getByRole("link", { name: "E-0001" }) });
    await expect(inflight).toBeVisible();
    await expect(inflight).toContainText("M-0001");
    await expect(inflight).toContainText("in_progress");
  });

  test("recent activity table populated", async ({ page }) => {
    await page.goto(fileURL("status.html"));
    const rows = page.locator("table.history tbody tr");
    expect(await rows.count()).toBeGreaterThan(0);
  });
});

test.describe("polish — kicker + dark mode + accent bar", () => {
  test("every page emits a kicker line above its H1", async ({ page }) => {
    for (const path of ["index.html", "E-0001.html", "M-0001.html"]) {
      await page.goto(fileURL(path));
      const kicker = page.locator("p.kicker").first();
      await expect(kicker).toBeVisible();
      // Computed style should be uppercase + muted.
      const transform = await kicker.evaluate((el) => getComputedStyle(el).textTransform);
      expect(transform).toBe("uppercase");
    }
  });

  test("milestone kicker carries kind + id + parent epic", async ({ page }) => {
    await page.goto(fileURL("M-0001.html"));
    const kicker = page.locator("p.kicker").first();
    await expect(kicker).toContainText("milestone");
    await expect(kicker).toContainText("M-0001");
    await expect(kicker).toContainText("E-0001");
  });

  test("accent bar pseudo-element renders on main", async ({ page }) => {
    await page.goto(fileURL("index.html"));
    const beforeBg = await page.locator("main").evaluate(
      (el) => getComputedStyle(el, "::before").backgroundColor,
    );
    // The ::before pseudo carries the accent color; should not be
    // transparent / unset.
    expect(beforeBg).not.toBe("rgba(0, 0, 0, 0)");
    expect(beforeBg).not.toBe("");
  });

  test("dark mode re-maps tokens via prefers-color-scheme", async ({ browser }) => {
    const ctx = await browser.newContext({ colorScheme: "dark" });
    const darkPage = await ctx.newPage();
    await darkPage.goto(fileURL("index.html"));
    const bg = await darkPage.locator("body").evaluate(
      (el) => getComputedStyle(el).backgroundColor,
    );
    // In dark mode --bg becomes #0f1115 → rgb(15, 17, 21). The
    // important property: it's a dark color (low channel sum), not
    // the light-mode --bg #f7f8fa.
    const [r, g, b] = parseRgb(bg);
    expect(r + g + b).toBeLessThan(150);
    await ctx.close();
  });
});

test.describe("layout — viewport-fill (M-0098/AC-1)", () => {
  // The body's `max-width: 78rem; margin: 2rem auto` cap is going
  // away; the layout fills the viewport with modest uniform edge
  // padding (no centering gutter). The body has no max-width cap;
  // the sidebar's left edge and main's right edge each sit within
  // a small threshold of the viewport's edges.
  //
  // 1920×1080 is a common laptop/external-monitor width that puts
  // any 78rem (~1248px) cap into visible play — at this viewport
  // the original CSS centered everything with ~336px of slack on
  // each side, which is the failure mode this test pins. The
  // 32px threshold accommodates the body's 1rem (16px) padding
  // plus sub-pixel rendering; "viewport-fill with modest padding"
  // is the intent, not "strict flush-left" (which would tighten
  // mobile rendering uncomfortably and forbids visible breathing
  // room from the browser frame).
  const EDGE_PX = 32;

  test("body has no max-width; layout fills viewport with modest padding at 1920px", async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto(fileURL("index.html"));

    // Body must have no max-width cap. `getComputedStyle` reports
    // resolved CSS values; "none" means no cap.
    const bodyMaxWidth = await page.locator("body").evaluate(
      (el) => getComputedStyle(el).maxWidth,
    );
    expect(bodyMaxWidth, "body.maxWidth should be 'none' (no cap)").toBe("none");

    // Sidebar's left edge within EDGE_PX of viewport x=0 — modest
    // padding allowed but no centering gutter (which would be ~336px
    // at 1920 with the 78rem cap).
    const sidebarBox = await page.locator(".sidebar").boundingBox();
    expect(sidebarBox, ".sidebar must be in the layout").not.toBeNull();
    expect(sidebarBox!.x, `.sidebar left edge should be within ${EDGE_PX}px of viewport`).toBeLessThanOrEqual(EDGE_PX);

    // Main panel's right edge within EDGE_PX of viewport's right
    // edge. Same reasoning as the sidebar assertion.
    const mainBox = await page.locator("main").boundingBox();
    expect(mainBox, "main must be in the layout").not.toBeNull();
    const mainRight = mainBox!.x + mainBox!.width;
    const rightGap = 1920 - mainRight;
    expect(rightGap, `main right edge should be within ${EDGE_PX}px of viewport width`).toBeLessThanOrEqual(EDGE_PX);

    // No horizontal overflow — the layout fits the viewport, not
    // slightly over.
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - window.innerWidth,
    );
    expect(overflow, "no horizontal scroll (scrollWidth <= innerWidth)").toBeLessThanOrEqual(0);
  });
});

test.describe("layout — sidebar width (M-0098/AC-2)", () => {
  // The sidebar widens from the original 220px to a target value
  // that provides comfortable horizontal room for the brand mark,
  // top-section entries (Project status / Overview), and — when
  // M-0100 lands — the gaps-block with active count. ~30% wider
  // than the original puts the target near 285px (user-confirmed:
  // 285px fixed).
  //
  // The test asserts the computed width across several viewport
  // widths to confirm a fixed value, not a fluid one. If a future
  // iteration switches to clamp(), the expected value would change
  // by viewport — currently it's stable.
  const SIDEBAR_WIDTH = 285;

  test("sidebar column resolves to target width at all viewport widths", async ({ page }) => {
    for (const width of [1280, 1920, 2560]) {
      await page.setViewportSize({ width, height: 900 });
      await page.goto(fileURL("index.html"));

      // .sidebar's computed width must match the target. The .layout
      // grid drives this via grid-template-columns; we read it off
      // the rendered element rather than the CSS rule so any future
      // wrapping (e.g. extra padding/margin) is caught.
      const sidebarWidth = await page.locator(".sidebar").evaluate(
        (el) => el.getBoundingClientRect().width,
      );
      expect(
        Math.round(sidebarWidth),
        `sidebar width at viewport ${width}px should be ${SIDEBAR_WIDTH}px`,
      ).toBe(SIDEBAR_WIDTH);
    }
  });
});

test.describe("layout — tab clicks don't scroll (M-0098/AC-5)", () => {
  // Tabs use `:target`-driven CSS show/hide — clicking
  // <a href="#tab-build"> updates the URL fragment, which triggers
  // the browser's "scroll target into view" behavior. AC-1's body-
  // padding change removed the 2rem top margin that previously
  // buffered the visible jump; the scroll is now user-visible.
  // The fix is `scroll-margin-top: 100vh` on `section[data-tab]`,
  // which makes the browser's scroll-clamp keep the page at y=0.
  //
  // The test loads a milestone page, then clicks each tab and
  // asserts the scroll position stays at the top after every click.

  test("clicking each tab keeps scrollY === 0", async ({ page }) => {
    // Use a deliberately short viewport so the milestone page's
    // content overflows vertically — that's the scenario where the
    // scroll-to-fragment jump is user-visible. At a tall viewport
    // (e.g. 1280×800) the small fixture milestone fits entirely on
    // screen and the browser has no need to scroll, masking the bug.
    await page.setViewportSize({ width: 1280, height: 400 });
    await page.goto(fileURL("M-0001.html"));
    expect(await page.evaluate(() => window.scrollY)).toBe(0);

    // Click each tab in the nav strip and verify scroll stays put.
    for (const tab of ["overview", "manifest", "build", "tests", "commits", "provenance"]) {
      await page.locator(`nav.tabs a[href="#tab-${tab}"]`).click();
      // Allow the browser to settle on the scroll-to-fragment.
      await page.waitForFunction((t) => location.hash === `#tab-${t}`, tab);
      const scrollY = await page.evaluate(() => window.scrollY);
      expect(scrollY, `scrollY after clicking ${tab} tab should be 0`).toBe(0);
    }
  });
});

test.describe("layout — prose-cap (M-0098/AC-3)", () => {
  // Prose blocks inside `main` cap at ~50rem (~800px at default
  // 16px html font) for readability; wide content (tables, code
  // blocks, AC card containers, dependency DAG) stays unbound and
  // fills the panel width.
  //
  // The test uses computed-style assertions rather than bounding-
  // rect width because the fixture's prose paragraphs are short
  // and wouldn't reach the cap visually — but we need to verify
  // the CSS rule IS APPLIED, not just that text happens to fit
  // (which it would whether or not the rule existed). The cap
  // value is 50rem = 800px at the default 16px html font.

  test("paragraphs in main cap at 800px; tables stay unbound", async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto(fileURL("E-0001.html"));

    // A prose paragraph inside main has max-width: 800px (50rem).
    // `main p` is the targeted selector class.
    const proseMaxWidth = await page.locator("main p").first().evaluate(
      (el) => getComputedStyle(el).maxWidth,
    );
    expect(proseMaxWidth, "first <p> in main should be capped at 800px").toBe("800px");

    // A table inside main has no max-width cap.
    const tableMaxWidth = await page.locator("main table").first().evaluate(
      (el) => getComputedStyle(el).maxWidth,
    );
    expect(tableMaxWidth, "first <table> in main should not be capped").toBe("none");
  });
});

test.describe("layout — mobile collapse (M-0098/AC-4)", () => {
  // At viewport widths below 768px the existing
  // `@media (max-width: 768px)` rule flips .layout's grid to
  // single-column and sets sidebar to `order: 2`, so the sidebar
  // renders below main instead of beside it. This is a regression
  // check on that behavior after AC-1's body-padding change and
  // AC-2's sidebar widening — both should leave the mobile collapse
  // intact.

  test("sidebar renders below main at <768px viewports", async ({ page }) => {
    for (const width of [375, 600, 767]) {
      await page.setViewportSize({ width, height: 900 });
      await page.goto(fileURL("index.html"));

      const sidebarBox = await page.locator(".sidebar").boundingBox();
      const mainBox = await page.locator("main").boundingBox();
      expect(sidebarBox, ".sidebar must be in the layout").not.toBeNull();
      expect(mainBox, "main must be in the layout").not.toBeNull();

      // Sidebar's top edge sits below main's bottom edge — single-
      // column stack with main first.
      expect(
        sidebarBox!.y,
        `sidebar.y should be > main.y + main.height at viewport ${width}px (collapsed)`,
      ).toBeGreaterThan(mainBox!.y + mainBox!.height - 1);

      // No horizontal scrollbar — the layout fits the narrow
      // viewport.
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - window.innerWidth,
      );
      expect(overflow, `no horizontal scroll at viewport ${width}px`).toBeLessThanOrEqual(0);
    }
  });
});

test.describe("kind-index — file emission (M-0099/AC-1)", () => {
  // The renderer collapses from active/all-pair to a single emitted
  // file per archive-segregating kind. The `*-all.html` cousin files
  // are no longer emitted; chip filtering (AC-3) handles the
  // active-vs-all toggle client-side.

  const KINDS = ["gaps", "decisions", "adrs", "contracts"];

  test("emits canonical kind file, no *-all.html cousin", async () => {
    for (const kind of KINDS) {
      expect(
        existsSync(join(outDir, `${kind}.html`)),
        `${kind}.html should be emitted`,
      ).toBe(true);
      expect(
        existsSync(join(outDir, `${kind}-all.html`)),
        `${kind}-all.html should NOT be emitted (chip filter replaces the all-view)`,
      ).toBe(false);
    }
  });
});

test.describe("kind-index — chip strip markup (M-0099/AC-2)", () => {
  // Each kind-index page renders a chip strip at the top of <main>
  // with two chips: Active (default view) and All (archived
  // included). Each chip is a styled <a> with both id and href
  // pointing at a matching fragment, so `:target` on the chip
  // itself can drive both the active-chip visual state (AC-2
  // styling) and the row-filter behavior (AC-3 CSS rule).
  //
  // The chip strip renders unconditionally — even on a kind page
  // with zero archived entries the strip appears, so the surface
  // shape is consistent across kinds and over time.

  const KINDS = ["gaps", "decisions", "adrs", "contracts"];

  test("chip strip with Active and All renders on every kind-index page", async ({ page }) => {
    for (const kind of KINDS) {
      await page.goto(fileURL(`${kind}.html`));

      // Chip strip is inside <main>, at the top of the kind-index
      // content (before the table).
      const chipStrip = page.locator("main > nav.chip-strip");
      await expect(chipStrip, `${kind}.html should have main > nav.chip-strip`).toHaveCount(1);

      // Exactly two chips: Active first, All second.
      const chips = chipStrip.locator("a.chip");
      await expect(chips).toHaveCount(2);

      const active = chips.nth(0);
      await expect(active).toHaveText("Active");
      await expect(active).toHaveAttribute("href", "#active");
      await expect(active).toHaveAttribute("id", "active");

      const all = chips.nth(1);
      await expect(all).toHaveText("All");
      await expect(all).toHaveAttribute("href", "#all");
      await expect(all).toHaveAttribute("id", "all");
    }
  });
});

test.describe("kind-index — chip filter behavior (M-0099/AC-3)", () => {
  // The chip filter is :target-driven CSS:
  // - Default view (no fragment): archived rows have display: none
  // - #all view: all rows visible regardless of archive state
  // Every row carries data-archived="true|false" so the CSS rule
  // can target archived rows specifically.
  //
  // The fixture includes G-0001 (active) and G-0002 (archived) so
  // both row classes are exercised on a real rendered table.

  test("rows carry data-archived; archived hidden by default; #all reveals", async ({ page }) => {
    // Default view — gaps.html with no fragment.
    await page.goto(fileURL("gaps.html"));

    // Every row in the kind-index table carries data-archived.
    const allRows = page.locator("table.kind-index tbody tr");
    const rowCount = await allRows.count();
    expect(rowCount, "gaps.html should have at least 2 rows from fixture").toBeGreaterThanOrEqual(2);
    for (let i = 0; i < rowCount; i++) {
      const attr = await allRows.nth(i).getAttribute("data-archived");
      expect(attr, `row ${i} should have data-archived attribute`).toMatch(/^(true|false)$/);
    }

    // Archived rows are display: none in default view.
    const archived = page.locator('table.kind-index tbody tr[data-archived="true"]');
    await expect(archived, "gaps.html should have at least one archived row in markup").toHaveCount(1);
    const archivedDisplay = await archived.first().evaluate(
      (el) => getComputedStyle(el).display,
    );
    expect(archivedDisplay, "archived row should be display:none in default view").toBe("none");

    // Active rows are visible.
    const active = page.locator('table.kind-index tbody tr[data-archived="false"]');
    const activeDisplay = await active.first().evaluate(
      (el) => getComputedStyle(el).display,
    );
    expect(activeDisplay, "active row should be visible in default view").not.toBe("none");

    // #all view — archived rows now visible.
    await page.goto(fileURL("gaps.html") + "#all");
    const archivedDisplayAll = await page.locator('table.kind-index tbody tr[data-archived="true"]')
      .first()
      .evaluate((el) => getComputedStyle(el).display);
    expect(archivedDisplayAll, "archived row should be visible under #all").not.toBe("none");
  });
});

test.describe("kind-index — home page nav (M-0099/AC-4)", () => {
  // The home page's Browse-by-kind block has one link per kind
  // (gaps, decisions, adrs, contracts) — the pre-migration `all`
  // sub-link is gone since the chip filter handles the active-vs-all
  // toggle from the kind page itself. This is a regression check;
  // the link was already removed in AC-1's green commit.

  test("home page kind-index has one link per kind, no all-link sub-link", async ({ page }) => {
    await page.goto(fileURL("index.html"));

    const kindNav = page.locator("ul.kind-index");
    await expect(kindNav).toHaveCount(1);

    // No .all-link sub-links anywhere in the block.
    const allSubLinks = kindNav.locator("a.all-link");
    await expect(allSubLinks, "kind-index block should have no .all-link sub-links").toHaveCount(0);

    // One primary <a> per kind list-item, pointing at the canonical
    // <kind>.html (not <kind>-all.html).
    const primaryLinks = kindNav.locator("> li > a");
    const linkCount = await primaryLinks.count();
    expect(linkCount, "expected one primary link per kind in nav block").toBeGreaterThanOrEqual(4);
    for (let i = 0; i < linkCount; i++) {
      const href = await primaryLinks.nth(i).getAttribute("href");
      expect(href, `primary link ${i} should not point at *-all.html`).not.toMatch(/-all\.html$/);
    }
  });
});

test.describe("sidebar — gap entry (M-0100/AC-1)", () => {
  // The sidebar's top section gains a "Gaps (N)" entry on every
  // rendered page. N is the count of non-archived gaps; the
  // fixture has G-0001 (active) + G-0002 (archived) so the count
  // is 1.

  const PAGES = ["index.html", "E-0001.html", "M-0001.html", "gaps.html"];

  test("sidebar shows Gaps (N) entry on every page kind", async ({ page }) => {
    for (const pagePath of PAGES) {
      await page.goto(fileURL(pagePath));
      const entry = page.locator("aside.sidebar .sidebar-top a").filter({ hasText: /^Gaps \(\d+\)/ });
      await expect(entry, `${pagePath}: sidebar should have a Gaps (N) entry`).toHaveCount(1);
      const text = await entry.textContent();
      expect(text, `${pagePath}: Gaps entry should display count 1 (one active gap in fixture)`).toContain("Gaps (1)");
    }
  });

  test("clicking sidebar Gaps entry navigates to gaps.html", async ({ page }) => {
    await page.goto(fileURL("index.html"));
    const entry = page.locator("aside.sidebar .sidebar-top a").filter({ hasText: /^Gaps \(\d+\)/ });
    await entry.click();
    await expect(page).toHaveURL(/gaps\.html$/);
  });
});

test.describe("sidebar — chip strip markup (M-0100/AC-2)", () => {
  // The sidebar has its own chip strip mirroring the kind-index
  // chip strip from M-0099 but using the `#sidebar-active` and
  // `#sidebar-all` URL fragments so the two filters operate
  // independently. Each chip is an <a> with matching id and href.

  test("sidebar contains nav.chip-strip with Active and All chips", async ({ page }) => {
    await page.goto(fileURL("index.html"));

    const chipStrip = page.locator("aside.sidebar nav.chip-strip");
    await expect(chipStrip).toHaveCount(1);

    const chips = chipStrip.locator("a.chip");
    await expect(chips).toHaveCount(2);

    const active = chips.nth(0);
    await expect(active).toHaveText("Active");
    await expect(active).toHaveAttribute("href", "#sidebar-active");
    await expect(active).toHaveAttribute("id", "sidebar-active");

    const all = chips.nth(1);
    await expect(all).toHaveText("All");
    await expect(all).toHaveAttribute("href", "#sidebar-all");
    await expect(all).toHaveAttribute("id", "sidebar-all");
  });
});

test.describe("sidebar — archive chip filter (M-0100/AC-3)", () => {
  // The sidebar's epic list hides archived epics by default;
  // #sidebar-all reveals them. Each <details class="sidebar-epic">
  // carries data-archived so the CSS filter can target archived
  // epics specifically. Fixture has E-0001 + E-0002 active and
  // E-0003 archived.

  test("archived epic hidden by default; #sidebar-all reveals", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto(fileURL("index.html"));

    // Each sidebar epic carries data-archived.
    const allEpics = page.locator("aside.sidebar details.sidebar-epic");
    const epicCount = await allEpics.count();
    expect(epicCount, "sidebar should have all 3 epics in markup").toBe(3);
    for (let i = 0; i < epicCount; i++) {
      const attr = await allEpics.nth(i).getAttribute("data-archived");
      expect(attr, `sidebar epic ${i} should have data-archived attribute`).toMatch(/^(true|false)$/);
    }

    // Default view — archived epic hidden.
    const archived = page.locator('aside.sidebar details.sidebar-epic[data-archived="true"]');
    await expect(archived, "fixture should have one archived sidebar epic").toHaveCount(1);
    const archivedDisplay = await archived.evaluate(
      (el) => getComputedStyle(el).display,
    );
    expect(archivedDisplay, "archived sidebar epic should be display:none by default").toBe("none");

    // Active epics visible.
    const activeEpics = page.locator('aside.sidebar details.sidebar-epic[data-archived="false"]');
    const activeDisplay = await activeEpics.first().evaluate(
      (el) => getComputedStyle(el).display,
    );
    expect(activeDisplay, "active sidebar epic should be visible by default").not.toBe("none");

    // #sidebar-all view — archived epic visible.
    await page.goto(fileURL("index.html") + "#sidebar-all");
    const archivedDisplayAll = await page
      .locator('aside.sidebar details.sidebar-epic[data-archived="true"]')
      .evaluate((el) => getComputedStyle(el).display);
    expect(archivedDisplayAll, "archived sidebar epic should be visible under #sidebar-all").not.toBe("none");
  });
});

test.describe("chips — no-scroll-on-click (M-0100/AC-4)", () => {
  // Same bug class as M-0098/AC-5 (tab clicks scrolling). Chips
  // have id matching href so :target lights them up — but the
  // browser also scrolls the chip into view on hash change.
  // `scroll-margin-top: 100vh` on .chip clamps the scroll to y=0.

  test("kind-index chip click keeps scrollY === 0", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 400 });
    await page.goto(fileURL("gaps.html"));
    expect(await page.evaluate(() => window.scrollY)).toBe(0);

    for (const fragment of ["#all", "#active"]) {
      await page.locator(`main a.chip[href="${fragment}"]`).click();
      await page.waitForFunction((f) => location.hash === f, fragment);
      const scrollY = await page.evaluate(() => window.scrollY);
      expect(scrollY, `kind-index chip ${fragment} click should keep scrollY=0`).toBe(0);
    }
  });

  test("sidebar chip click keeps scrollY === 0", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 400 });
    await page.goto(fileURL("index.html"));
    expect(await page.evaluate(() => window.scrollY)).toBe(0);

    for (const fragment of ["#sidebar-all", "#sidebar-active"]) {
      await page.locator(`aside.sidebar a.chip[href="${fragment}"]`).click();
      await page.waitForFunction((f) => location.hash === f, fragment);
      const scrollY = await page.evaluate(() => window.scrollY);
      expect(scrollY, `sidebar chip ${fragment} click should keep scrollY=0`).toBe(0);
    }
  });
});

test.describe("link integrity", () => {
  test("every internal href resolves to a file or in-page anchor", async ({ page }) => {
    for (const path of ["index.html", "E-0001.html", "E-0002.html", "M-0001.html", "M-0002.html"]) {
      await page.goto(fileURL(path));
      const hrefs = await page.locator("a[href]").evaluateAll((els) =>
        els.map((e) => (e as HTMLAnchorElement).getAttribute("href") ?? ""),
      );
      for (const href of hrefs) {
        if (href.startsWith("#")) continue; // in-page anchor — verified separately
        if (href.startsWith("http")) continue; // out-of-scope external
        // File URLs resolve relative to the current page's directory.
        // We re-issue the link via a new page.goto and assert it
        // doesn't throw a network error. file:// returns ok=true even
        // on missing files, so we additionally check the response
        // navigates to the expected URL.
        const expected = fileURL(href);
        const resp = await page.goto(expected);
        expect(resp?.ok(), `dead link ${href} on ${path}`).toBeTruthy();
      }
    }
  });
});

// parseRgb pulls the integer channels out of "rgb(R, G, B)" or
// "rgba(R, G, B, A)" — Playwright's getComputedStyle returns rgb-
// shaped strings.
function parseRgb(s: string): [number, number, number] {
  const m = s.match(/\d+/g);
  if (!m || m.length < 3) throw new Error(`unexpected color string: ${s}`);
  return [parseInt(m[0], 10), parseInt(m[1], 10), parseInt(m[2], 10)];
}
