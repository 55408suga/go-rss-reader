import { describe, expect, it } from "vitest";
import {
  filterArticles,
  isNewArticle,
  newCount,
  unreadCount,
  unreadCountByFeed,
  type ReadingContext,
} from "./article-status";
import type { Article } from "./api/schemas";

function makeArticle(
  p: Partial<Article> & { id: string; published_at: string },
): Article {
  return {
    id: p.id,
    title: p.title ?? "title",
    description: p.description ?? "",
    content: p.content ?? "",
    website_url: p.website_url ?? "https://example.com/post",
    published_at: p.published_at,
    feed_id: p.feed_id ?? "feed-1",
    external_id: p.external_id ?? "ext-1",
  };
}

const LAST_VISIT = Date.parse("2026-06-15T00:00:00Z");

function ctx(over: Partial<ReadingContext> = {}): ReadingContext {
  return {
    readIds: new Set(),
    starredIds: new Set(),
    lastVisitedAt: LAST_VISIT,
    ...over,
  };
}

describe("isNewArticle", () => {
  it("flags an unread article published after the last visit", () => {
    const a = makeArticle({ id: "a", published_at: "2026-06-15T10:00:00Z" });
    expect(isNewArticle(a, ctx())).toBe(true);
  });

  it("does not flag an article that has already been read", () => {
    const a = makeArticle({ id: "a", published_at: "2026-06-15T10:00:00Z" });
    expect(isNewArticle(a, ctx({ readIds: new Set(["a"]) }))).toBe(false);
  });

  it("does not flag an article published before the last visit", () => {
    const a = makeArticle({ id: "a", published_at: "2026-06-14T10:00:00Z" });
    expect(isNewArticle(a, ctx())).toBe(false);
  });

  it("treats the exact last-visit instant as not new (strictly newer)", () => {
    const a = makeArticle({ id: "a", published_at: "2026-06-15T00:00:00Z" });
    expect(isNewArticle(a, ctx())).toBe(false);
  });

  it("returns false for an unparseable published_at", () => {
    const a = makeArticle({ id: "a", published_at: "not-a-date" });
    expect(isNewArticle(a, ctx())).toBe(false);
  });
});

describe("unread counting and view filters", () => {
  const articles = [
    makeArticle({
      id: "a1",
      feed_id: "f1",
      published_at: "2026-06-16T00:00:00Z",
    }),
    makeArticle({
      id: "a2",
      feed_id: "f1",
      published_at: "2026-06-16T00:00:00Z",
    }),
    makeArticle({
      id: "a3",
      feed_id: "f2",
      published_at: "2026-06-16T00:00:00Z",
    }),
  ];

  it("counts unread over loaded articles", () => {
    expect(unreadCount(articles, new Set(["a1"]))).toBe(2);
  });

  it("counts unread per feed", () => {
    const m = unreadCountByFeed(articles, new Set(["a1"]));
    expect(m.get("f1")).toBe(1);
    expect(m.get("f2")).toBe(1);
  });

  it("filters the unread view", () => {
    const out = filterArticles(
      articles,
      "unread",
      ctx({ readIds: new Set(["a1"]) }),
    );
    expect(out.map((a) => a.id)).toEqual(["a2", "a3"]);
  });

  it("filters the starred view", () => {
    const out = filterArticles(
      articles,
      "starred",
      ctx({ starredIds: new Set(["a3"]) }),
    );
    expect(out.map((a) => a.id)).toEqual(["a3"]);
  });

  it("returns everything for the all view", () => {
    expect(filterArticles(articles, "all", ctx())).toHaveLength(3);
  });

  it("counts new unread articles", () => {
    // all three are published after the last visit and unread
    expect(newCount(articles, ctx())).toBe(3);
    // marking one read drops the count
    expect(newCount(articles, ctx({ readIds: new Set(["a1"]) }))).toBe(2);
  });
});
