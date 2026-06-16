"use client";

import { Moon, RefreshCw, Sun } from "lucide-react";
import { useEffect, useRef, type ReactNode } from "react";
import type { ArticleStatus } from "@/lib/article-status";
import type { Article, Feed } from "@/lib/api/schemas";
import { useTheme } from "@/lib/theme";
import { ArticleRow } from "./article-row";
import { RssWave } from "./rss-wave";
import { ErrorPanel, TimelineSkeleton } from "./states";

export function Timeline({
  title,
  newCount,
  feedCount,
  articles,
  feeds,
  statusOf,
  empty,
  isLoading,
  isError,
  error,
  refetch,
  hasNextPage,
  isFetchingNextPage,
  fetchNextPage,
  onRefresh,
  isRefreshing,
  onOpenArticle,
  onToggleStar,
}: {
  title: string;
  newCount: number;
  feedCount: number;
  articles: Article[];
  feeds: Feed[];
  statusOf: (article: Article) => ArticleStatus;
  empty: ReactNode;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  refetch: () => void;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  fetchNextPage: () => void;
  onRefresh: () => void;
  isRefreshing: boolean;
  onOpenArticle: (article: Article) => void;
  onToggleStar: (articleId: string) => void;
}) {
  const { theme, toggle } = useTheme();
  const dark = theme === "dark";
  const feedById = new Map(feeds.map((f) => [f.id, f]));

  // Infinite scroll: load the next page when the sentinel scrolls into view.
  const sentinel = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = sentinel.current;
    if (!el || !hasNextPage) return;
    const observer = new IntersectionObserver((entries) => {
      if (entries[0]?.isIntersecting && !isFetchingNextPage) fetchNextPage();
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  return (
    <main className="flex min-w-0 flex-1 flex-col">
      <div className="shrink-0 px-6 pt-[18px]">
        <div className="flex items-center gap-3">
          <h1 className="flex items-center gap-[10px] text-[21px] font-bold tracking-[-0.01em] text-ink">
            {title}
            {newCount > 0 && (
              <span className="flex items-center gap-[6px] rounded-full bg-orange-t py-[3px] pl-[7px] pr-[9px] text-[11px] font-bold text-orange">
                <RssWave size={13} />
                {newCount} 件の新着
              </span>
            )}
          </h1>
          <span className="text-[13px] text-mut">{feedCount} フィード</span>
          <div className="ml-auto flex gap-[6px]">
            <ToolButton
              label="更新"
              onClick={onRefresh}
              disabled={isRefreshing}
            >
              <RefreshCw
                size={15}
                className={isRefreshing ? "animate-spin" : ""}
              />
            </ToolButton>
            <ToolButton
              label={dark ? "ライトモードに切替" : "ダークモードに切替"}
              onClick={toggle}
            >
              {dark ? <Sun size={15} /> : <Moon size={15} />}
            </ToolButton>
          </div>
        </div>
        <div
          className="mt-[14px] h-[3px] rounded-[3px] opacity-90"
          style={{
            background:
              "linear-gradient(90deg, var(--cyan) 0%, var(--cyan) 62%, var(--orange) 100%)",
          }}
        />
      </div>

      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
        {isLoading ? (
          <TimelineSkeleton />
        ) : isError ? (
          <ErrorPanel error={error} onRetry={refetch} />
        ) : articles.length === 0 ? (
          empty
        ) : (
          <>
            {articles.map((article) => (
              <ArticleRow
                key={article.id}
                article={article}
                feed={feedById.get(article.feed_id)}
                status={statusOf(article)}
                onOpen={() => onOpenArticle(article)}
                onToggleStar={() => onToggleStar(article.id)}
              />
            ))}

            <div ref={sentinel} className="flex justify-center p-4">
              {isFetchingNextPage ? (
                <span className="text-sm text-mut">読み込み中…</span>
              ) : hasNextPage ? (
                <button
                  type="button"
                  onClick={() => fetchNextPage()}
                  className="rounded-[9px] border border-line bg-panel px-4 py-2 text-sm font-semibold text-cyan-d"
                >
                  もっと読む
                </button>
              ) : (
                <span className="text-xs text-mut2">すべて読み込みました</span>
              )}
            </div>
          </>
        )}
      </div>
    </main>
  );
}

function ToolButton({
  label,
  onClick,
  disabled,
  children,
}: {
  label: string;
  onClick: () => void;
  disabled?: boolean;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      onClick={onClick}
      disabled={disabled}
      className="flex size-[34px] items-center justify-center rounded-[9px] border border-line bg-panel text-cyan-d disabled:opacity-60"
    >
      {children}
    </button>
  );
}
