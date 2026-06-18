import { expect, test, type Page } from "@playwright/test";

// Hermetic E2E: the /api/v1 backend is stubbed with route mocks so these run
// without a live server or database. Mock payloads mirror the common envelope
// from docs/specifications/v1-front.md.

const FEED_A = "00000000-0000-7000-8000-0000000000a1";
const FEED_B = "00000000-0000-7000-8000-0000000000b2";

function feed(id: string, title: string, host: string) {
  return {
    id,
    title,
    feed_url: `https://${host}/feed.xml`,
    website_url: `https://${host}`,
    description: `${title} description`,
    registered_at: "2026-06-15T00:00:00Z",
    updated_at: "2026-06-15T00:00:00Z",
    language: "en",
  };
}

function article(id: string, feedId: string, title: string, host: string) {
  return {
    id,
    title,
    description: "Article description for the timeline row.",
    content: "Full content body.",
    website_url: `https://${host}/post/${id}`,
    published_at: "2026-06-16T00:00:00Z",
    feed_id: feedId,
    external_id: `ext-${id}`,
  };
}

const FEEDS = [
  feed(FEED_A, "Mock Feed A", "a.example.com"),
  feed(FEED_B, "Mock Feed B", "b.example.com"),
];
const ARTICLES = [
  article(
    "00000000-0000-7000-8000-0000000000c1",
    FEED_A,
    "Alpha article from A",
    "a.example.com",
  ),
  article(
    "00000000-0000-7000-8000-0000000000c2",
    FEED_B,
    "Bravo article from B",
    "b.example.com",
  ),
  article(
    "00000000-0000-7000-8000-0000000000c3",
    FEED_A,
    "Gamma article from A",
    "a.example.com",
  ),
];

const CORS = {
  "access-control-allow-origin": "*",
  "access-control-allow-methods": "GET,POST,DELETE,OPTIONS",
  "access-control-allow-headers": "content-type",
};

const meta = (pagination = false) => ({
  request_id: "test-req",
  ...(pagination ? { pagination: { next_cursor: null, has_more: false } } : {}),
});

/** Install stateful mocks; deleting a feed removes it from later list responses. */
async function installMocks(page: Page) {
  const deleted = new Set<string>();

  await page.route("**/api/v1/**", async (route) => {
    const req = route.request();
    const method = req.method();
    const path = new URL(req.url()).pathname;
    const json = (data: unknown, status = 200) =>
      route.fulfill({
        status,
        headers: CORS,
        contentType: "application/json",
        body: JSON.stringify(data),
      });

    if (method === "OPTIONS")
      return route.fulfill({ status: 204, headers: CORS });

    const deleteMatch = path.match(/\/api\/v1\/feeds\/([^/]+)$/);
    if (method === "DELETE" && deleteMatch) {
      deleted.add(deleteMatch[1]);
      return route.fulfill({ status: 204, headers: CORS, body: "" });
    }

    if (path.endsWith("/feeds")) {
      const feeds = FEEDS.filter((f) => !deleted.has(f.id));
      return json({ data: { feeds }, meta: meta(true) });
    }

    const feedArticles = path.match(/\/api\/v1\/feeds\/([^/]+)\/articles$/);
    if (feedArticles) {
      const items = ARTICLES.filter((a) => a.feed_id === feedArticles[1]);
      return json({ data: { articles: items }, meta: meta(true) });
    }

    if (path.endsWith("/articles")) {
      return json({ data: { articles: ARTICLES }, meta: meta(true) });
    }

    return json({ data: { feeds: FEEDS }, meta: meta(true) });
  });
}

test.beforeEach(async ({ page }) => {
  await installMocks(page);
});

test("renders feeds and articles from the API", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByText("FeedGo", { exact: true })).toBeVisible();
  await expect(page.locator("aside").getByText("Mock Feed A")).toBeVisible();
  await expect(page.getByText("Alpha article from A")).toBeVisible();
  await expect(page.locator('article[role="link"]')).toHaveCount(3);
});

test("toggles dark mode", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: /モードに切替/ }).click();
  await expect
    .poll(() =>
      page.evaluate(() => document.documentElement.classList.contains("dark")),
    )
    .toBe(true);
});

test("opens the add-feed dialog and switches modes", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "フィードを追加" }).first().click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  await expect(dialog.locator('input[type="url"]')).toBeVisible();
  await dialog.getByRole("button", { name: /サイト URL から検出/ }).click();
  await expect(dialog.locator('input[placeholder*="blog"]')).toBeVisible();
});

test("filters the timeline to a single feed", async ({ page }) => {
  await page.goto("/");
  await page.getByText("Alpha article from A").waitFor();
  await page.locator('[data-testid="feed-select"]').first().click();
  await expect(page.locator("h1")).toContainText("Mock Feed A");
  await expect(page.getByText("Bravo article from B")).toHaveCount(0);
  await expect(page.locator('article[role="link"]')).toHaveCount(2);
});

test("deletes a feed", async ({ page }) => {
  page.on("dialog", (d) => d.accept());
  await page.goto("/");
  const sidebar = page.locator("aside");
  await expect(sidebar.getByText("Mock Feed B")).toBeVisible();
  await page.getByRole("button", { name: "Mock Feed B を削除" }).click();
  await expect(sidebar.getByText("Mock Feed B")).toHaveCount(0);
  await expect(sidebar.getByText("Mock Feed A")).toBeVisible();
});

// Japanese typography: the article body (serif stack) and the UI chrome (sans
// stack) must both resolve their CJK fallback to Noto Sans JP (gothic), and
// Japanese-bearing headings/titles must not carry the Latin-tuned negative
// letter-spacing. These assert the *computed* CSS in a real browser, since
// next/font hashes the family name and Tailwind's tracking only resolves there.
test.describe("Japanese typography", () => {
  const computed = (page: Page, selector: string, prop: "fontFamily" | "letterSpacing") =>
    page.locator(selector).first().evaluate(
      (node, p) => getComputedStyle(node as HTMLElement)[p as "fontFamily" | "letterSpacing"],
      prop,
    );

  test("article titles (serif stack) fall back to Noto Sans JP, not Noto Serif JP", async ({
    page,
  }) => {
    await page.goto("/");
    await page.getByText("Alpha article from A").waitFor();
    const family = await computed(page, 'article[role="link"] h3', "fontFamily");
    expect(family).toContain("Noto Sans JP");
    expect(family).not.toContain("Noto Serif JP");
  });

  test("UI headings (sans stack) include the Noto Sans JP webfont", async ({ page }) => {
    await page.goto("/");
    await page.locator("h1").first().waitFor();
    const family = await computed(page, "h1", "fontFamily");
    expect(family).toContain("Noto Sans JP");
  });

  test("Japanese titles and headings do not tighten letter-spacing", async ({ page }) => {
    await page.goto("/");
    await page.getByText("Alpha article from A").waitFor();
    expect(await computed(page, 'article[role="link"] h3', "letterSpacing")).toBe("normal");
    expect(await computed(page, "h1", "letterSpacing")).toBe("normal");
  });
});
