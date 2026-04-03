CREATE TABLE feeds (
    id UUID PRIMARY KEY,
    title varchar(100) NOT NULL,
    registered_at timestamp WITH time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp WITH time zone NOT NULL,
    feed_url varchar(2048) NOT NULL UNIQUE,
    website_url varchar(2048) NOT NULL UNIQUE,
    description text,
    language text
);

CREATE TABLE articles (
    id UUID PRIMARY KEY,
    title varchar(100) NOT NULL,
    description text,
    published_at timestamp WITH time zone NOT NULL,
    website_url varchar(2048) NOT NULL,
    content text,
    feed_id UUID REFERENCES feeds(id) ON DELETE CASCADE NOT NULL,
    external_id text NOT NULL,
    UNIQUE(feed_id, external_id)
);

-- create table feed_fetch_status (
--     feed_id UUID primary key REFERENCES feeds(id) ON DELETE CASCADE NOT NULL,
--     last_fetched_at timestamp WITH time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
--     next_fetch_at timestamp WITH time zone,
--     status_code int NOT NULL,
--     error_message text,
--     last_modified timestamp with time zone,
--     fetch_interval_hours int NOT NULL DEFAULT 24,
--     failure_count int NOT NULL DEFAULT 0,
-- );

-- CREATE INDEX idx_articles_feed_id ON articles(feed_id);