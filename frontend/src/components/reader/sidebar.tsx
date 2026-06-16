"use client";

import { Circle, List, Plus, Star, Trash2 } from "lucide-react";
import type { Feed } from "@/lib/api/schemas";
import { feedAvatarColor, feedInitial } from "@/lib/format";
import { useTheme } from "@/lib/theme";
import type { ReaderView } from "./reader-app";
import { RssWave } from "./rss-wave";

type NavKey = "all" | "unread" | "starred";

export function Sidebar({
  feeds,
  activeView,
  navCounts,
  unreadByFeed,
  feedsLoading,
  onSelectNav,
  onSelectFeed,
  onAddFeed,
  onDeleteFeed,
}: {
  feeds: Feed[];
  activeView: ReaderView;
  navCounts: { all: number; unread: number; starred: number };
  unreadByFeed: Map<string, number>;
  feedsLoading: boolean;
  onSelectNav: (key: NavKey) => void;
  onSelectFeed: (feedId: string) => void;
  onAddFeed: () => void;
  onDeleteFeed: (feedId: string) => void;
}) {
  const { theme } = useTheme();
  const dark = theme === "dark";

  return (
    <aside className="flex w-[252px] shrink-0 flex-col gap-[15px] border-r border-line bg-tint p-[18px_14px]">
      {/* Brand */}
      <div className="flex items-center gap-[11px] px-1">
        <div className="flex size-[38px] shrink-0 items-center justify-center rounded-[10px] bg-cyan text-orange shadow-[0_3px_10px_rgba(23,162,214,0.25)] dark:bg-[#0a1117] dark:shadow-[0_3px_14px_rgba(0,0,0,0.45)]">
          <RssWave size={20} />
        </div>
        <span className="leading-none">
          <span className="block text-[23px] font-bold tracking-[-0.02em] text-ink">
            FeedGo
          </span>
          <span className="mt-[3px] block text-[9.5px] font-semibold tracking-[0.22em] text-cyan">
            RSS READER
          </span>
        </span>
      </div>

      {/* Add feed */}
      <button
        type="button"
        onClick={onAddFeed}
        className="flex items-center justify-center gap-2 rounded-[10px] bg-orange px-3 py-[11px] text-[13.5px] font-bold text-white shadow-[0_5px_14px_rgba(247,147,30,0.32)]"
      >
        <Plus size={16} strokeWidth={2.5} />
        フィードを追加
      </button>

      {/* Main nav */}
      <nav className="flex flex-col gap-[2px]">
        <NavItem
          icon={<List size={16} />}
          label="すべての記事"
          count={navCounts.all}
          active={activeView.kind === "all"}
          onClick={() => onSelectNav("all")}
        />
        <NavItem
          icon={<Circle size={14} />}
          label="未読"
          count={navCounts.unread}
          active={activeView.kind === "unread"}
          onClick={() => onSelectNav("unread")}
        />
        <NavItem
          icon={<Star size={15} />}
          label="スター付き"
          count={navCounts.starred}
          active={activeView.kind === "starred"}
          onClick={() => onSelectNav("starred")}
        />
      </nav>

      {/* Feeds */}
      <div className="px-[11px] py-1 text-[10.5px] font-bold uppercase tracking-[0.1em] text-mut2">
        フィード
      </div>
      <div className="flex min-h-0 flex-1 flex-col gap-px overflow-y-auto">
        {feedsLoading && feeds.length === 0
          ? Array.from({ length: 6 }).map((_, i) => (
              <div
                key={i}
                className="flex animate-pulse items-center gap-[10px] px-[11px] py-[6px]"
              >
                <div className="size-[22px] shrink-0 rounded-[7px] bg-line" />
                <div className="h-3 w-28 rounded bg-line" />
              </div>
            ))
          : feeds.map((feed) => {
              const unread = unreadByFeed.get(feed.id) ?? 0;
              const active =
                activeView.kind === "feed" && activeView.feedId === feed.id;
              return (
                <div
                  key={feed.id}
                  className={`group/feed flex items-center gap-[8px] rounded-[9px] pl-[11px] pr-[5px] hover:bg-panel ${
                    active ? "bg-panel" : ""
                  }`}
                >
                  <button
                    type="button"
                    data-testid="feed-select"
                    onClick={() => onSelectFeed(feed.id)}
                    className={`flex min-w-0 flex-1 items-center gap-[10px] py-[6px] text-[13px] text-ink ${
                      active ? "font-bold" : ""
                    }`}
                  >
                    <span
                      className="flex size-[22px] shrink-0 items-center justify-center rounded-[7px] text-[11px] font-bold text-white"
                      style={{ background: feedAvatarColor(feed, dark) }}
                    >
                      {feedInitial(feed.title)}
                    </span>
                    <span className="flex-1 truncate text-left">
                      {feed.title}
                    </span>
                  </button>
                  {unread > 0 && (
                    <span className="min-w-[18px] rounded-[10px] bg-orange px-[6px] text-center text-[11px] font-bold text-white">
                      {unread}
                    </span>
                  )}
                  <button
                    type="button"
                    data-testid="feed-delete"
                    aria-label={`${feed.title} を削除`}
                    onClick={() => {
                      if (window.confirm(`「${feed.title}」を削除しますか？`)) {
                        onDeleteFeed(feed.id);
                      }
                    }}
                    className="shrink-0 rounded-md p-1 text-mut2 opacity-0 transition-opacity hover:text-orange focus-visible:opacity-100 group-hover/feed:opacity-100"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              );
            })}
      </div>
    </aside>
  );
}

function NavItem({
  icon,
  label,
  count,
  active,
  onClick,
}: {
  icon: React.ReactNode;
  label: string;
  count: number;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={active ? "page" : undefined}
      className={`relative flex items-center gap-[10px] rounded-[9px] px-[11px] py-[9px] text-[13.5px] ${
        active
          ? "bg-panel font-bold text-cyan-d shadow-[0_2px_8px_rgba(23,90,120,0.07)] dark:shadow-[0_2px_10px_rgba(0,0,0,0.4)]"
          : "text-mut"
      }`}
    >
      {active && (
        <span className="absolute inset-y-2 left-0 w-[3px] rounded-[3px] bg-cyan" />
      )}
      {icon}
      <span className="flex-1 text-left">{label}</span>
      <span
        className={`text-[12px] font-semibold ${
          active ? "text-orange" : "text-mut2"
        }`}
      >
        {count}
      </span>
    </button>
  );
}
