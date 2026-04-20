-- name: SaveFeed :exec
INSERT INTO feeds (
    id, title, registered_at, updated_at, feed_url, website_url, description, language
) VALUES (
    $1, $2, CURRENT_TIMESTAMP, $3, $4, $5, $6, $7
);

-- name: GetFeedByID :one
SELECT id, title, registered_at, updated_at, feed_url, website_url, description, language
FROM feeds
WHERE id = $1 LIMIT 1;

-- カーソル付きと分割しなきゃsqlcの型変換がうまくいかない
-- name: ListFeeds :many
SELECT id, title, registered_at, updated_at, feed_url, website_url, description, language
FROM feeds
ORDER BY registered_at DESC, id DESC
LIMIT sqlc.arg('limit')::integer;

-- name: ListFeedsFromCursor :many
SELECT id, title, registered_at, updated_at, feed_url, website_url, description, language
FROM feeds
WHERE registered_at < sqlc.arg('cursor_at')
   OR (registered_at = sqlc.arg('cursor_at') AND id < sqlc.arg('cursor_id'))
ORDER BY registered_at DESC, id DESC
LIMIT sqlc.arg('limit')::integer;


-- name: ListArticlesByFeedID :many
SELECT id, title, description, published_at, website_url, content, feed_id, external_id
FROM articles
WHERE feed_id = sqlc.arg('feed_id')
ORDER BY published_at DESC, id DESC
LIMIT sqlc.arg('limit')::integer;

-- name: ListArticlesByFeedIDFromCursor :many
SELECT id, title, description, published_at, website_url, content, feed_id, external_id
FROM articles
WHERE feed_id = sqlc.arg('feed_id')
  AND (published_at < sqlc.arg('cursor_at')
       OR (published_at = sqlc.arg('cursor_at') AND id < sqlc.arg('cursor_id')))
ORDER BY published_at DESC, id DESC
LIMIT sqlc.arg('limit')::integer;

-- name: UpdateFeed :exec
UPDATE feeds
SET title = $1,
    updated_at = $2,
    feed_url = $3,
    website_url = $4,
    description = $5,
    language = $6
WHERE id = $7;

-- name: DeleteFeed :exec
DELETE FROM feeds
WHERE id = $1;

-- name: SaveArticle :exec
INSERT INTO articles (
    id, title, description, published_at, website_url, content, feed_id, external_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (feed_id, external_id) DO NOTHING;

-- name: GetArticleByID :one
SELECT id, title, description, published_at, website_url, content, feed_id, external_id
FROM articles
WHERE id = $1 LIMIT 1;

-- name: ListArticles :many
SELECT id, title, description, published_at, website_url, content, feed_id, external_id
FROM articles
ORDER BY published_at DESC, id DESC
LIMIT sqlc.arg('limit')::integer;

-- name: ListArticlesFromCursor :many
SELECT id, title, description, published_at, website_url, content, feed_id, external_id
FROM articles
WHERE published_at < sqlc.arg('cursor_at')
   OR (published_at = sqlc.arg('cursor_at') AND id < sqlc.arg('cursor_id'))
ORDER BY published_at DESC, id DESC
LIMIT sqlc.arg('limit')::integer;

-- name: UpdateArticle :exec
UPDATE articles
SET title = $1,
    description = $2,
    published_at = $3,
    website_url = $4,
    content = $5,
    feed_id = $6
WHERE id = $7;

-- name: DeleteArticle :exec
DELETE FROM articles
WHERE id = $1;

-- name: SaveFeedFetchStatus :exec
INSERT INTO feed_fetch_status (
    feed_id, last_fetched_at, next_fetch_at, status_code, error_message, last_modified, etag, fetch_interval_hours, failure_count
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (feed_id) DO UPDATE
SET last_fetched_at = EXCLUDED.last_fetched_at,
    next_fetch_at = EXCLUDED.next_fetch_at,
    status_code = EXCLUDED.status_code,
    error_message = EXCLUDED.error_message,
    last_modified = EXCLUDED.last_modified,
    etag = EXCLUDED.etag,
    fetch_interval_hours = EXCLUDED.fetch_interval_hours,
    failure_count = EXCLUDED.failure_count;

-- name: GetFeedFetchStatusByFeedID :one
SELECT feed_id, last_fetched_at, next_fetch_at, status_code, error_message, last_modified, etag, fetch_interval_hours, failure_count
FROM feed_fetch_status
WHERE feed_id = $1
LIMIT 1;

-- name: GetDueFeedFetchStatuses :many
SELECT ffs.feed_id, ffs.last_fetched_at, ffs.next_fetch_at, ffs.status_code, ffs.error_message,
       ffs.last_modified, ffs.etag, ffs.fetch_interval_hours, ffs.failure_count, f.feed_url
FROM feed_fetch_status ffs
JOIN feeds f ON ffs.feed_id = f.id
WHERE ffs.next_fetch_at <= sqlc.arg('now')
ORDER BY ffs.next_fetch_at ASC
LIMIT sqlc.arg('limit')::integer;
