-- name: SaveFeed :exec
INSERT INTO feeds (
    id, title, registered_at, updated_at, feed_url, website_url, description, language
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: GetFeedByID :one
SELECT id, title, registered_at, updated_at, feed_url, website_url, description, language
FROM feeds
WHERE id = $1 LIMIT 1;

-- name: GetAllFeeds :many
SELECT id, title, registered_at, updated_at, feed_url, website_url, description, language
FROM feeds
ORDER BY registered_at DESC;

-- name: GetArticles :many
SELECT id, title, description, published_at, website_url, content, feed_id
FROM articles
WHERE feed_id = $1
ORDER BY published_at DESC;

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

-- name: RegisterArticle :exec
INSERT INTO articles (
    id, title, description, published_at, website_url, content, feed_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: GetArticleByID :one
SELECT id, title, description, published_at, website_url, content, feed_id
FROM articles
WHERE id = $1 LIMIT 1;

-- name: GetAllArticles :many
SELECT id, title, description, published_at, website_url, content, feed_id
FROM articles
ORDER BY published_at DESC;

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
