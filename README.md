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

## Background task queue (Asynq)

A generic task queue port and an Asynq adapter are available:
- Port: internal/infrastructure/taskqueue/port
- Adapter: internal/infrastructure/taskqueue/adapter

Environment variables:
- REDIS_URL: Redis connection string (e.g., redis://redis:6379/0) used by cache and Asynq.
- ASYNQ_CONCURRENCY: Optional worker concurrency (default: 10).
- ASYNQ_QUEUES: Optional queue weights, e.g., "critical=6,default=3,low=1" (default: "default=1").

Example (client):
```
cli, _ := adapter.NewAsynqClientFromEnv()

defer cli.Close()
_, _ = cli.Enqueue(ctx, port.Task{Type: "email:send", Payload: payloadBytes}, port.EnqueueOption{Queue: "default"})
```

Example (server):
```
srv, _ := adapter.NewAsynqServerFromEnv()

srv.Register("email:send", func(ctx context.Context, t port.Task) error {
    // handle t.Payload
    return nil
})

// This call blocks until the context is canceled
_ = srv.Run(ctx)
```

- get the web UI binary:
- https://github.com/hibiken/asynqmon

` ./asynqmon --redis-url <REDIS_URL> --port 8080`

## Websocket Chat Usage

- Endpoint: `GET /api/v1/chat/ws?userId=<uuid>` upgrades to a websocket; the `userId` query parameter identifies the session. No request body is sent during the upgrade.
- Handshake response: the server emits `{"type":"connected"}` once the socket is ready. Ping frames are sent every 30s; standard websocket clients reply automatically.
- Join or leave a conversation by sending JSON frames after the connection is open:
  - Join: `{"type":"join","conversationId":"<uuid>"}` → server replies `{"type":"joined","conversationId":"<uuid>"}`.
  - Leave: `{"type":"leave","conversationId":"<uuid>"}` → server replies `{"type":"left","conversationId":"<uuid>"}`.
- Send a message while joined to the conversation:
  ```
  {
    "type": "message",
    "conversationId": "<uuid>",
    "body": "hello world",
    "msgType": 0,
    "attachmentUrl": null,
    "attachmentMeta": null,
    "dedupeKey": null
  }
  ```
  The payload is persisted via the regular send-message use case and broadcast back as:
  ```
  {
    "type": "message",
    "conversationId": "<uuid>",
    "message": {
      "id": "<message-id>",
      "conversationId": "<uuid>",
      "senderId": "<user-id>",
      "createdAt": "2025-01-01T12:00:00Z",
      "body": "hello world",
      "msgType": 0,
      "attachmentUrl": null,
      "attachmentMeta": null,
      "dedupeKey": null
    }
  }
  ```
- Error frames use `{"type":"error","code":"bad_request|forbidden|internal_error","error":"..."}`. For example, attempting to join a conversation you are not part of yields `code="forbidden"`.
- Disconnecting the socket removes the user from all rooms; reconnect with the same `userId` to resume and re-issue `join` frames as needed.
