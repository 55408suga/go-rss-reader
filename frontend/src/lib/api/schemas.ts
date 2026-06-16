import { z } from "zod";

// Zod schemas for the /api/v1 contract (see docs/specifications/v1-front.md).
//
// The backend wraps every response in a common envelope:
//   success: { data, meta }
//   failure: { error, meta }
// Validating the envelope at the network boundary turns silent backend drift
// into a loud, early error instead of an "undefined is not an object" deep in
// the UI.

// Backend IDs are UUIDv7. z.string().uuid() in older zod versions rejects the
// v7 variant, so we use a permissive RFC 4122 shape that accepts any version.
const uuid = z
  .string()
  .regex(
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i,
    "must be a UUID",
  );

// RFC3339 timestamps may carry a "Z" or a numeric offset.
const rfc3339 = z.string().datetime({ offset: true });

export const FeedSchema = z.object({
  id: uuid,
  title: z.string(),
  feed_url: z.string().url(),
  website_url: z.string(),
  description: z.string(),
  registered_at: rfc3339,
  updated_at: rfc3339,
  language: z.string(),
});

export const ArticleSchema = z.object({
  id: uuid,
  title: z.string(),
  description: z.string(),
  content: z.string(),
  website_url: z.string().url(),
  published_at: rfc3339,
  feed_id: uuid,
  external_id: z.string(),
});

// FeedCandidate is a value object returned by POST /feeds/discover; it is never
// persisted. mime_type is one of rss+xml / atom+xml / feed+json.
export const FeedCandidateSchema = z.object({
  feed_url: z.string().url(),
  title: z.string(),
  mime_type: z.string(),
});

export const PaginationSchema = z.object({
  next_cursor: z.string().nullable(),
  has_more: z.boolean(),
});

export const MetaSchema = z.object({
  request_id: z.string(),
  pagination: PaginationSchema.optional(),
});

// Generic success-envelope factory: envelope(<payload schema>) validates the
// whole { data, meta } shape in one place for every endpoint.
export const envelope = <T extends z.ZodTypeAny>(data: T) =>
  z.object({ data, meta: MetaSchema });

// error.code mirrors the backend apperror.Code set exactly.
export const ErrorCodeSchema = z.enum([
  "invalid_argument",
  "not_found",
  "conflict",
  "external_unavailable",
  "internal",
]);

export const ErrorDetailSchema = z.object({
  field: z.string(),
  reason: z.string(),
});

export const ErrorEnvelopeSchema = z.object({
  error: z.object({
    code: ErrorCodeSchema,
    message: z.string(),
    details: z.array(ErrorDetailSchema).optional(),
  }),
  meta: MetaSchema,
});

// Per-endpoint payload envelopes.
export const FeedsEnvelope = envelope(z.object({ feeds: z.array(FeedSchema) }));
export const FeedEnvelope = envelope(z.object({ feed: FeedSchema }));
export const ArticlesEnvelope = envelope(
  z.object({ articles: z.array(ArticleSchema) }),
);
export const FeedWithArticlesEnvelope = envelope(
  z.object({ feed: FeedSchema, articles: z.array(ArticleSchema) }),
);
export const DiscoverEnvelope = envelope(
  z.object({
    feed: FeedSchema,
    articles: z.array(ArticleSchema),
    candidates: z.array(FeedCandidateSchema),
  }),
);

// Types are derived from the schemas so there is a single source of truth.
export type Feed = z.infer<typeof FeedSchema>;
export type Article = z.infer<typeof ArticleSchema>;
export type FeedCandidate = z.infer<typeof FeedCandidateSchema>;
export type Pagination = z.infer<typeof PaginationSchema>;
export type ErrorCode = z.infer<typeof ErrorCodeSchema>;
export type ErrorDetail = z.infer<typeof ErrorDetailSchema>;

// The cursor is an opaque token: the client never inspects its contents, it
// just echoes meta.pagination.next_cursor back as the next ?cursor=.
export type Cursor = string;
