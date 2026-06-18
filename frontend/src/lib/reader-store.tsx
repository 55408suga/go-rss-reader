"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { ReadingContext } from "./article-status";

const READ_KEY = "feedgo-read";
const STAR_KEY = "feedgo-starred";
const VISIT_KEY = "feedgo-last-visited";

function loadIds(key: string): Set<string> {
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return new Set();
    const parsed: unknown = JSON.parse(raw);
    return Array.isArray(parsed) ? new Set(parsed as string[]) : new Set();
  } catch {
    return new Set();
  }
}

function saveIds(key: string, ids: Set<string>): void {
  try {
    window.localStorage.setItem(key, JSON.stringify([...ids]));
  } catch {
    // storage unavailable — state simply won't persist this session
  }
}

type ReaderStoreValue = {
  /** Reactive reading context consumed by article-status helpers. */
  ctx: ReadingContext;
  markRead: (id: string) => void;
  toggleStar: (id: string) => void;
  /** Adds the given ids to the read set (used by "mark all as read"). */
  markAllRead: (ids: string[]) => void;
};

const ReaderStoreContext = createContext<ReaderStoreValue | null>(null);

export function ReaderStoreProvider({ children }: { children: ReactNode }) {
  const [readIds, setReadIds] = useState<Set<string>>(() => new Set());
  const [starredIds, setStarredIds] = useState<Set<string>>(() => new Set());
  // Default threshold is "now" so nothing is flagged new before the effect
  // below replaces it with the *previous* visit time.
  const [lastVisitedAt, setLastVisitedAt] = useState<number>(() => Date.now());

  // Hydrate from localStorage once, on mount (client only).
  useEffect(() => {
    setReadIds(loadIds(READ_KEY));
    setStarredIds(loadIds(STAR_KEY));

    const storedVisit = Number(window.localStorage.getItem(VISIT_KEY));
    if (Number.isFinite(storedVisit) && storedVisit > 0) {
      setLastVisitedAt(storedVisit);
    }
    // Stamp this visit so the *next* one compares against now.
    try {
      window.localStorage.setItem(VISIT_KEY, String(Date.now()));
    } catch {
      // ignore
    }
  }, []);

  const markRead = useCallback((id: string) => {
    setReadIds((prev) => {
      if (prev.has(id)) return prev;
      const next = new Set(prev);
      next.add(id);
      saveIds(READ_KEY, next);
      return next;
    });
  }, []);

  const markAllRead = useCallback((ids: string[]) => {
    setReadIds((prev) => {
      const next = new Set(prev);
      for (const id of ids) next.add(id);
      saveIds(READ_KEY, next);
      return next;
    });
  }, []);

  const toggleStar = useCallback((id: string) => {
    setStarredIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      saveIds(STAR_KEY, next);
      return next;
    });
  }, []);

  const value = useMemo<ReaderStoreValue>(
    () => ({
      ctx: { readIds, starredIds, lastVisitedAt },
      markRead,
      markAllRead,
      toggleStar,
    }),
    [readIds, starredIds, lastVisitedAt, markRead, markAllRead, toggleStar],
  );

  return (
    <ReaderStoreContext.Provider value={value}>
      {children}
    </ReaderStoreContext.Provider>
  );
}

export function useReaderStore(): ReaderStoreValue {
  const ctx = useContext(ReaderStoreContext);
  if (!ctx)
    throw new Error("useReaderStore must be used within <ReaderStoreProvider>");
  return ctx;
}
