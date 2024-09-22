-- +goose Up
CREATE TABLE feeds (
    id uuid primary key,
    created_at timestamp,
    updated_at timestamp,
    name varchar(255) not null,
    url varchar(255) not null unique,
    user_id uuid not null references users(id) on delete cascade
);

-- +goose Down
DROP TABLE feeds;

