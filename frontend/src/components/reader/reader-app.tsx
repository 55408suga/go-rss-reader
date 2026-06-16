"use client";

import { useMemo, useState } from "react";
import {
  articleStatus,
  filterArticles,
  newCount as countNew,
  unreadCount,
  unreadCountByFeed,
  type ArticleView,
} from "@/lib/article-status";
import type { Article } from "@/lib/api/schemas";
import {
  useArticles,
  useDeleteFeed,
  useFeeds,
  useRefreshFeed,
  type ArticleScope,
} from "@/lib/hooks";
import { useReaderStore } from "@/lib/reader-store";
import { AddFeedDialog } from "./add-feed-dialog";
import { Sidebar } from "./sidebar";
import { EmptyState } from "./states";
import { Timeline } from "./timeline";

export type ReaderView =
  | { kind: "all" }
  | { kind: "unread" }
  | { kind: "starred" }
  | { kind: "feed"; feedId: string };

function scopeFor(view: ReaderView): ArticleScope {
  // Unread / starred are client-side filters over the full "all" stream;
  // only an explicit feed selection narrows the server query.
  return view.kind === "feed"
    ? { type: "feed", feedId: view.feedId }
    : { type: "all" };
}

export function ReaderApp() {
  const [view, setView] = useState<ReaderView>({ kind: "all" });
  const [dialogOpen, setDialogOpen] = useState(false);

  const { ctx, markRead, toggleStar } = useReaderStore();

  const feedsQuery = useFeeds();
  const feeds = feedsQuery.data?.items ?? [];

  const scope = scopeFor(view);
  const articlesQuery = useArticles(scope);
  const refreshFeed = useRefreshFeed();
  const deleteFeed = useDeleteFeed();

  const loadedArticles = useMemo(
    () => articlesQuery.data?.pages.flatMap((p) => p.items) ?? [],
    [articlesQuery.data],
  );

  // Apply the client-side view filter (no-op for all / feed).
  const viewFilter: ArticleView =
    view.kind === "unread"
      ? "unread"
      : view.kind === "starred"
        ? "starred"
        : "all";
  const visibleArticles = useMemo(
    () => filterArticles(loadedArticles, viewFilter, ctx),
    [loadedArticles, viewFilter, ctx],
  );

  const navCounts = useMemo(
    () => ({
      all: loadedArticles.length,
      unread: unreadCount(loadedArticles, ctx.readIds),
      starred: ctx.starredIds.size,
    }),
    [loadedArticles, ctx.readIds, ctx.starredIds],
  );

  const unreadByFeed = useMemo(
    () => unreadCountByFeed(loadedArticles, ctx.readIds),
    [loadedArticles, ctx.readIds],
  );

  const title =
    view.kind === "feed"
      ? (feeds.find((f) => f.id === view.feedId)?.title ?? "フィード")
      : view.kind === "unread"
        ? "未読"
        : view.kind === "starred"
          ? "スター付き"
          : "すべての記事";

  function openArticle(article: Article) {
    markRead(article.id);
    window.open(article.website_url, "_blank", "noopener,noreferrer");
  }

  function refresh() {
    if (view.kind === "feed") {
      refreshFeed.mutate(view.feedId, {
        onSettled: () => articlesQuery.refetch(),
      });
    } else {
      articlesQuery.refetch();
    }
  }

  function removeFeed(feedId: string) {
    // Leave a feed-scoped view if its feed is being deleted.
    if (view.kind === "feed" && view.feedId === feedId) {
      setView({ kind: "all" });
    }
    deleteFeed.mutate(feedId);
  }

  const empty =
    feeds.length === 0 ? (
      <EmptyState
        title="まだフィードが登録されていません"
        message="「フィードを追加」から購読したい RSS/Atom フィードの URL、またはサイトの URL を登録してください。"
        action={
          <button
            type="button"
            onClick={() => setDialogOpen(true)}
            className="mt-1 rounded-[10px] bg-orange px-4 py-2 text-sm font-bold text-white shadow-[0_5px_14px_rgba(247,147,30,0.32)]"
          >
            フィードを追加
          </button>
        }
      />
    ) : view.kind === "unread" ? (
      <EmptyState
        title="未読の記事はありません"
        message="読み込み済みの記事はすべて既読です。"
      />
    ) : view.kind === "starred" ? (
      <EmptyState
        title="スター付きの記事はありません"
        message="記事の ★ を押すとここに集まります。"
      />
    ) : (
      <EmptyState
        title="記事がありません"
        message="このフィードにはまだ記事がありません。更新すると取得できる場合があります。"
      />
    );

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar
        feeds={feeds}
        activeView={view}
        navCounts={navCounts}
        unreadByFeed={unreadByFeed}
        feedsLoading={feedsQuery.isLoading}
        onSelectNav={(key) => setView({ kind: key })}
        onSelectFeed={(feedId) => setView({ kind: "feed", feedId })}
        onAddFeed={() => setDialogOpen(true)}
        onDeleteFeed={removeFeed}
      />

      <Timeline
        title={title}
        newCount={countNew(visibleArticles, ctx)}
        feedCount={feeds.length}
        articles={visibleArticles}
        feeds={feeds}
        statusOf={(article) => articleStatus(article, ctx)}
        empty={empty}
        isLoading={articlesQuery.isLoading}
        isError={articlesQuery.isError}
        error={articlesQuery.error}
        refetch={() => articlesQuery.refetch()}
        hasNextPage={articlesQuery.hasNextPage}
        isFetchingNextPage={articlesQuery.isFetchingNextPage}
        fetchNextPage={() => articlesQuery.fetchNextPage()}
        onRefresh={refresh}
        isRefreshing={refreshFeed.isPending || articlesQuery.isRefetching}
        onOpenArticle={openArticle}
        onToggleStar={toggleStar}
      />

      <AddFeedDialog open={dialogOpen} onClose={() => setDialogOpen(false)} />
    </div>
  );
}
