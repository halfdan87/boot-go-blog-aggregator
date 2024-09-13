-- +goose Up
CREATE TABLE "users" (
    id uuid primary key,
    created_at timestamp,
    updated_at timestamp,
    name varchar(255) not null
);

-- +goose Down
DROP TABLE "users";


