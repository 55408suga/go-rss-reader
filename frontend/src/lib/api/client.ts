import { z } from "zod";
import {
  ArticlesEnvelope,
  DiscoverEnvelope,
  ErrorEnvelopeSchema,
  FeedEnvelope,
  FeedsEnvelope,
  FeedWithArticlesEnvelope,
  type Article,
  type Cursor,
  type ErrorCode,
  type ErrorDetail,
  type Feed,
  type FeedCandidate,
} from "./schemas";

// Dev default matches the backend's CORS allow-list origin pairing.
export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

/**
 * ApiError mirrors the backend failure envelope. `code` is machine-readable and
 * always accurate; `details` map directly onto form fields (details[].field is
 * the request field name the client sent, e.g. "feed_url").
 */
export class ApiError extends Error {
  readonly code: ErrorCode;
  readonly status: number;
  readonly details: ErrorDetail[];
  readonly requestId?: string;

  constructor(
    code: ErrorCode,
    message: string,
    status: number,
    details: ErrorDetail[] = [],
    requestId?: string,
  ) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
    this.details = details;
    this.requestId = requestId;
  }

  /** Collapse details into a { field: reason } map for per-input error display. */
  fieldErrors(): Record<string, string> {
    const out: Record<string, string> = {};
    for (const d of this.details) {
      if (!(d.field in out)) out[d.field] = d.reason;
    }
    return out;
  }
}

/** Thrown when a 2xx body does not match the expected schema (= backend drift). */
export class SchemaError extends Error {
  readonly issues: z.ZodIssue[];
  constructor(issues: z.ZodIssue[]) {
    super("unexpected response shape from server");
    this.name = "SchemaError";
    this.issues = issues;
  }
}

async function toApiError(res: Response): Promise<ApiError> {
  let body: unknown = null;
  try {
    body = await res.json();
  } catch {
    // body was empty or not JSON (e.g. a proxy error page)
  }
  const parsed = ErrorEnvelopeSchema.safeParse(body);
  if (parsed.success) {
    const { error, meta } = parsed.data;
    return new ApiError(
      error.code,
      error.message,
      res.status,
      error.details ?? [],
      meta.request_id,
    );
  }
  // Failure that does not match the contract: synthesize a code from status so
  // callers can still branch on .code.
  const code: ErrorCode = res.status >= 500 ? "internal" : "invalid_argument";
  return new ApiError(code, `request failed (HTTP ${res.status})`, res.status);
}

const jsonHeaders = { "Content-Type": "application/json" };

async function requestJson<T extends z.ZodTypeAny>(
  path: string,
  schema: T,
  init?: RequestInit,
): Promise<z.infer<T>> {
  const res = await fetch(`${API_BASE_URL}/api/v1${path}`, init);
  if (!res.ok) throw await toApiError(res);
  const parsed = schema.safeParse(await res.json());
  if (!parsed.success) throw new SchemaError(parsed.error.issues);
  return parsed.data;
}

async function requestNoContent(
  path: string,
  init?: RequestInit,
): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1${path}`, init);
  if (!res.ok) throw await toApiError(res);
  // 204 No Content — nothing to read.
}

// ---------------------------------------------------------------------------
// Cursor pagination
// ---------------------------------------------------------------------------

export type PageParams = { cursor?: Cursor | null; limit?: number };

/** A normalized page: the opaque next token plus a terminal flag. */
export type Page<T> = {
  items: T[];
  nextCursor: Cursor | null;
  hasMore: boolean;
};

function pageQuery(params?: PageParams): string {
  const q = new URLSearchParams();
  if (params?.cursor) q.set("cursor", params.cursor);
  if (params?.limit != null) q.set("limit", String(params.limit));
  const s = q.toString();
  return s ? `?${s}` : "";
}

function toPage<T>(
  items: T[],
  pagination?: { next_cursor: string | null; has_more: boolean },
): Page<T> {
  return {
    items,
    nextCursor: pagination?.next_cursor ?? null,
    hasMore: pagination?.has_more ?? false,
  };
}

// ---------------------------------------------------------------------------
// Endpoints (see docs/specifications/v1-front.md)
// ---------------------------------------------------------------------------

export async function listFeeds(
  params?: PageParams,
  signal?: AbortSignal,
): Promise<Page<Feed>> {
  const { data, meta } = await requestJson(
    `/feeds${pageQuery(params)}`,
    FeedsEnvelope,
    { signal },
  );
  return toPage(data.feeds, meta.pagination);
}

export async function getFeed(id: string, signal?: AbortSignal): Promise<Feed> {
  const { data } = await requestJson(`/feeds/${id}`, FeedEnvelope, { signal });
  return data.feed;
}

export async function listArticles(
  params?: PageParams,
  signal?: AbortSignal,
): Promise<Page<Article>> {
  const { data, meta } = await requestJson(
    `/articles${pageQuery(params)}`,
    ArticlesEnvelope,
    { signal },
  );
  return toPage(data.articles, meta.pagination);
}

export async function listFeedArticles(
  feedId: string,
  params?: PageParams,
  signal?: AbortSignal,
): Promise<Page<Article>> {
  const { data, meta } = await requestJson(
    `/feeds/${feedId}/articles${pageQuery(params)}`,
    ArticlesEnvelope,
    { signal },
  );
  return toPage(data.articles, meta.pagination);
}

export async function registerFeed(
  feedUrl: string,
): Promise<{ feed: Feed; articles: Article[] }> {
  const { data } = await requestJson(`/feeds`, FeedWithArticlesEnvelope, {
    method: "POST",
    headers: jsonHeaders,
    body: JSON.stringify({ feed_url: feedUrl }),
  });
  return data;
}

export async function discoverFeed(
  websiteUrl: string,
): Promise<{ feed: Feed; articles: Article[]; candidates: FeedCandidate[] }> {
  const { data } = await requestJson(`/feeds/discover`, DiscoverEnvelope, {
    method: "POST",
    headers: jsonHeaders,
    body: JSON.stringify({ website_url: websiteUrl }),
  });
  return data;
}

export async function refreshFeed(id: string): Promise<void> {
  await requestNoContent(`/feeds/${id}/refresh`, { method: "POST" });
}

export async function deleteFeed(id: string): Promise<void> {
  await requestNoContent(`/feeds/${id}`, { method: "DELETE" });
}
