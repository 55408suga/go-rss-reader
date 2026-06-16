// Read / starred / "new" state is NOT provided by the backend (see the API
// spec — there is no read flag). It is owned entirely by the client. This
// module holds the pure predicates so the rules live in one testable place,
// separate from React state plumbing (see reader-store.tsx).

import type { Article } from "./api/schemas";

export type ReadingContext = {
  /** Article IDs the user has opened. */
  readIds: Set<string>;
  /** Article IDs the user has starred. */
  starredIds: Set<string>;
  /**
   * Epoch ms of the user's *previous* visit. Articles published after this are
   * candidates for the "new since you were last here" indicator.
   */
  lastVisitedAt: number;
};

export type ArticleStatus = {
  read: boolean;
  starred: boolean;
  isNew: boolean;
};

export function isRead(article: Article, readIds: Set<string>): boolean {
  return readIds.has(article.id);
}

export function isStarred(article: Article, starredIds: Set<string>): boolean {
  return starredIds.has(article.id);
}

/**
 * Decides whether an article should show the orange "新着" RSS-wave indicator
 * (in the timeline header badge and on the article avatar).
 */
export function isNewArticle(article: Article, ctx: ReadingContext): boolean {
  const publishedAt = new Date(article.published_at).getTime();
  if (Number.isNaN(publishedAt)) return false;
  // Reading an article clears its "new" flag.
  if (ctx.readIds.has(article.id)) return false;
  // New = published strictly after the user's previous visit.
  return publishedAt > ctx.lastVisitedAt;
}

export function articleStatus(
  article: Article,
  ctx: ReadingContext,
): ArticleStatus {
  return {
    read: isRead(article, ctx.readIds),
    starred: isStarred(article, ctx.starredIds),
    isNew: isNewArticle(article, ctx),
  };
}

/** Unread count over the supplied (loaded) articles. */
export function unreadCount(
  articles: Article[],
  readIds: Set<string>,
): number {
  let n = 0;
  for (const a of articles) if (!readIds.has(a.id)) n++;
  return n;
}

/** Unread count per feed_id over the supplied (loaded) articles. */
export function unreadCountByFeed(
  articles: Article[],
  readIds: Set<string>,
): Map<string, number> {
  const counts = new Map<string, number>();
  for (const a of articles) {
    if (readIds.has(a.id)) continue;
    counts.set(a.feed_id, (counts.get(a.feed_id) ?? 0) + 1);
  }
  return counts;
}

/** Count of articles considered "new" over the supplied (loaded) articles. */
export function newCount(articles: Article[], ctx: ReadingContext): number {
  let n = 0;
  for (const a of articles) if (isNewArticle(a, ctx)) n++;
  return n;
}

export type ArticleView = "all" | "unread" | "starred";

/** Client-side filter for the All / Unread / Starred nav views. */
export function filterArticles(
  articles: Article[],
  view: ArticleView,
  ctx: ReadingContext,
): Article[] {
  switch (view) {
    case "unread":
      return articles.filter((a) => !ctx.readIds.has(a.id));
    case "starred":
      return articles.filter((a) => ctx.starredIds.has(a.id));
    case "all":
    default:
      return articles;
  }
}
