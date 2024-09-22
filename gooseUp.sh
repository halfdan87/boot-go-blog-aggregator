cd sql/schema
goose postgres "postgres://postgres:postgres@localhost:5432/blogator?sslmode=disable" up
cd ../..


