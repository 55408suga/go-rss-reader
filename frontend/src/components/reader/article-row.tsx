"use client";

import { Star } from "lucide-react";
import type { ArticleStatus } from "@/lib/article-status";
import type { Article, Feed } from "@/lib/api/schemas";
import {
  feedAvatarColor,
  feedInitial,
  hostFromUrl,
  readingMinutes,
  relativeTime,
} from "@/lib/format";
import { useTheme } from "@/lib/theme";
import { RssWave } from "./rss-wave";

/**
 * Renders an interactive row for displaying an article with metadata and actions.
 *
 * @returns A React element representing the article row
 */
export function ArticleRow({
  article,
  feed,
  status,
  onOpen,
  onToggleStar,
}: {
  article: Article;
  feed?: Feed;
  status: ArticleStatus;
  onOpen: () => void;
  onToggleStar: () => void;
}) {
  const { theme } = useTheme();
  const dark = theme === "dark";

  const feedTitle = feed?.title ?? hostFromUrl(article.website_url);
  const color = feedAvatarColor({ id: feed?.id ?? article.feed_id }, dark);
  const minutes = readingMinutes(article);
  const host = hostFromUrl(article.website_url);

  return (
    <article
      role="link"
      tabIndex={0}
      aria-label={article.title}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onOpen();
        }
      }}
      className="group relative flex cursor-pointer gap-[14px] border-b border-line px-6 py-4 hover:bg-tint focus:outline-none focus-visible:bg-tint"
    >
      <span
        className="relative flex size-9 shrink-0 items-center justify-center rounded-[10px] text-[15px] font-bold text-white"
        style={{ background: color }}
      >
        {feedInitial(feedTitle)}
        {status.isNew && (
          <span className="absolute -right-1 -top-1 text-orange">
            <RssWave size={13} />
          </span>
        )}
      </span>

      <div className="min-w-0 flex-1">
        <div className="mb-[3px] flex items-center gap-2 overflow-hidden whitespace-nowrap text-xs text-mut">
          <b className="font-bold text-ink">{feedTitle}</b>
          <span>·</span>
          <span>{relativeTime(article.published_at)}</span>
        </div>

        <h3
          className={`mb-1 font-serif text-[16.5px] font-semibold leading-[1.42] ${
            status.read ? "text-mut" : "text-ink"
          }`}
        >
          {article.title}
        </h3>

        {article.description && (
          <p className="line-clamp-2 font-serif text-[13.5px] leading-[1.62] text-mut [text-wrap:pretty]">
            {article.description}
          </p>
        )}

        <div className="mt-[9px] flex items-center gap-[9px]">
          <span className="rounded-full bg-cyan-t px-[9px] py-[3px] text-[11px] font-bold text-cyan-d">
            {minutes} 分で読める
          </span>
          <span className="text-[11.5px] text-cyan-d">{host}</span>
        </div>
      </div>

      <button
        type="button"
        aria-label={status.starred ? "スターを外す" : "スターを付ける"}
        aria-pressed={status.starred}
        onClick={(e) => {
          e.stopPropagation();
          onToggleStar();
        }}
        className={`self-start rounded-md p-1 transition-opacity ${
          status.starred
            ? "text-orange opacity-100"
            : "text-mut2 opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
        }`}
      >
        <Star size={17} fill={status.starred ? "currentColor" : "none"} />
      </button>
    </article>
  );
}
