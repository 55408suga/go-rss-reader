"use client";

import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import {
  deleteFeed,
  discoverFeed,
  listArticles,
  listFeedArticles,
  listFeeds,
  refreshFeed,
  registerFeed,
} from "./api/client";

const PAGE_LIMIT = 20;
const FEEDS_LIMIT = 100;

/** Which article collection the timeline is showing. */
export type ArticleScope = { type: "all" } | { type: "feed"; feedId: string };

export const queryKeys = {
  feeds: ["feeds"] as const,
  articles: (scope: ArticleScope) =>
    scope.type === "feed"
      ? (["articles", "feed", scope.feedId] as const)
      : (["articles", "all"] as const),
};

/** Sidebar feed list. Feeds are few, so we fetch a generous first page. */
export function useFeeds() {
  return useQuery({
    queryKey: queryKeys.feeds,
    queryFn: ({ signal }) => listFeeds({ limit: FEEDS_LIMIT }, signal),
  });
}

/** Cursor-paginated article timeline for the active scope. */
export function useArticles(scope: ArticleScope) {
  return useInfiniteQuery({
    queryKey: queryKeys.articles(scope),
    queryFn: ({ pageParam, signal }) => {
      const params = { cursor: pageParam, limit: PAGE_LIMIT };
      return scope.type === "feed"
        ? listFeedArticles(scope.feedId, params, signal)
        : listArticles(params, signal);
    },
    initialPageParam: null as string | null,
    // hasMore=false / next_cursor=null ⇒ stop paginating.
    getNextPageParam: (lastPage) =>
      lastPage.hasMore ? lastPage.nextCursor : undefined,
  });
}

function useInvalidateAll() {
  const qc = useQueryClient();
  return () => {
    qc.invalidateQueries({ queryKey: ["feeds"] });
    qc.invalidateQueries({ queryKey: ["articles"] });
  };
}

export function useRegisterFeed() {
  const invalidate = useInvalidateAll();
  return useMutation({
    mutationFn: (feedUrl: string) => registerFeed(feedUrl),
    onSuccess: invalidate,
  });
}

export function useDiscoverFeed() {
  const invalidate = useInvalidateAll();
  return useMutation({
    mutationFn: (websiteUrl: string) => discoverFeed(websiteUrl),
    onSuccess: invalidate,
  });
}

export function useRefreshFeed() {
  const invalidate = useInvalidateAll();
  return useMutation({
    mutationFn: (id: string) => refreshFeed(id),
    onSuccess: invalidate,
  });
}

export function useDeleteFeed() {
  const invalidate = useInvalidateAll();
  return useMutation({
    mutationFn: (id: string) => deleteFeed(id),
    onSuccess: invalidate,
  });
}
