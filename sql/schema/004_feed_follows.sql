-- +goose Up
CREATE TABLE feed_follows (
    id uuid primary key,
    created_at timestamp,
    updated_at timestamp,
    user_id uuid not null references users(id) on delete cascade,
    feed_id uuid not null references feeds(id) on delete cascade
);

-- +goose Down
DROP TABLE feed_follows;

