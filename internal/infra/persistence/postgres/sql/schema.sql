CREATE TABLE feeds (
    id UUID PRIMARY KEY,
    title text NOT NULL,
    registered_at timestamp WITH time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp WITH time zone NOT NULL,
    feed_url varchar(2048) NOT NULL UNIQUE,
    website_url varchar(2048) NOT NULL UNIQUE,
    description text NOT NULL DEFAULT '',
    language text NOT NULL DEFAULT ''
);

CREATE TABLE articles (
    id UUID PRIMARY KEY,
    title text NOT NULL,
    description text NOT NULL DEFAULT '',
    published_at timestamp WITH time zone NOT NULL,
    website_url varchar(2048) NOT NULL,
    content text NOT NULL DEFAULT '',
    feed_id UUID REFERENCES feeds(id) ON DELETE CASCADE NOT NULL,
    external_id text NOT NULL,
    UNIQUE(feed_id, external_id)
);

create table feed_fetch_status (
    feed_id UUID primary key REFERENCES feeds(id) ON DELETE CASCADE NOT NULL,
    last_fetched_at timestamp WITH time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    next_fetch_at timestamp WITH time zone NOT NULL,
    status_code int NOT NULL,
    error_message text,
    last_modified timestamp with time zone,
    etag text,
    fetch_interval_hours int NOT NULL DEFAULT 12 CHECK (fetch_interval_hours > 0),
    failure_count int NOT NULL DEFAULT 0 CHECK (failure_count >= 0)
);

CREATE INDEX idx_feeds_registered_at_desc
ON feeds (registered_at DESC);

CREATE INDEX idx_articles_published_at_desc
ON articles (published_at DESC);

CREATE INDEX idx_articles_feed_id_published_at_desc
ON articles (feed_id, published_at DESC);

CREATE INDEX idx_feed_fetch_status_next_fetch_at
ON feed_fetch_status (next_fetch_at);