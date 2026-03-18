CREATE TABLE feeds (
    id UUID PRIMARY KEY,
    title varchar(100) NOT NULL,
    registered_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone,
    feed_url varchar(2048) NOT NULL UNIQUE,
    website_url varchar(2048) NOT NULL UNIQUE,
    description text,
    language varchar(30)
);

CREATE TABLE articles (
    id UUID PRIMARY KEY,
    title varchar(100) NOT NULL UNIQUE,
    description text,
    published_at timestamp with time zone NOT NULL,
    website_url varchar(2048) NOT NULL UNIQUE,
    content text,
    feed_id UUID REFERENCES feeds(id) ON DELETE CASCADE
);