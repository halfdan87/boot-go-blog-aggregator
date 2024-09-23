-- +goose Up
CREATE TABLE posts (
    id uuid primary key,
    created_at timestamp,
    updated_at timestamp,
    title varchar(255) not null,
    url varchar(512) not null unique,
    description varchar(1024) not null,
    published_at timestamp,
    feed_id uuid not null references feeds(id) on delete cascade
);

-- +goose Down
DROP TABLE posts;
