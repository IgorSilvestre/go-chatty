# go-chatty

A simple chat API built with Go, Gin, and Postgres.

## Run with Docker Compose

Prerequisites:
- Docker and Docker Compose plugin installed

Commands:
- Build and start services:
  - `docker compose up -d --build`
- View logs:
  - `docker compose logs -f api`
- Stop services and remove containers:
  - `docker compose down`

Services:
- API: http://localhost:8080
- Postgres: localhost:5432 (db=chatty, user=postgres, password=postgres)

The API container receives the DB connection string via the DB_URL environment variable (see docker-compose.yml).

## Migration

If you have the migrate CLI installed locally, you can run migrations with:

`migrate -path internal/infrastructure/database/migration -database "postgresql://postgres:postgres@localhost:5432/chatty?sslmode=disable" up`

Alternatively, you can use the official migrate docker image (example uses the default compose network name; adjust if necessary):

```
docker run --rm \
  --network go-chatty_default \
  -v $(pwd)/internal/infrastructure/database/migration:/migrations \
  migrate/migrate:4 \
  -path=/migrations \
  -database postgresql://postgres:postgres@db:5432/chatty?sslmode=disable up
```