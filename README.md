# Chirpy

A small HTTP API for a Twitter-like service where users post short messages ("chirps"). Written in Go, backed by PostgreSQL.

## Features

- User accounts with argon2id password hashing
- JWT access tokens (1 hour) plus long-lived refresh tokens
- Create, list, fetch, and delete chirps (140 character limit, basic profanity filter)
- Filter chirps by `author_id` and sort by creation date
- "Chirpy Red" membership upgrades via a Polka webhook (API-key protected)
- A static file server under `/app/` and admin metrics/reset endpoints

## Requirements

- Go 1.26+
- PostgreSQL
- [`goose`](https://github.com/pressly/goose) for migrations
- [`sqlc`](https://sqlc.dev) if you want to regenerate the database layer

## Setup

1. Create a Postgres database (default name `chirpy`).

2. Create a `.env` file in the project root:

   ```
   DB_URL="postgres://user:password@localhost:5432/chirpy?sslmode=disable"
   PLATFORM="dev"
   SECRET="<random base64 string used to sign JWTs>"
   POLKA_KEY="<api key expected from the Polka webhook>"
   ```

3. Run the migrations:

   ```bash
   goose -dir sql/schema postgres "$DB_URL" up
   ```

4. Build and run:

   ```bash
   go build -o chirpy && ./chirpy
   ```

The server listens on `:8080`.

## Development

Regenerate the database code after changing SQL in `sql/queries` or `sql/schema`:

```bash
sqlc generate
```

Run the tests:

```bash
go test ./...
```

## API

### Health & admin

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/api/healthz` | Liveness check, returns `OK` |
| `GET` | `/admin/metrics` | HTML page showing file server hit count |
| `POST` | `/admin/reset` | Resets hit count and deletes all users (only when `PLATFORM=dev`) |

### Users & auth

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/api/users` | Create a user. Body: `{ "email", "password" }` |
| `PUT` | `/api/users` | Update email/password. Requires `Authorization: Bearer <jwt>` |
| `POST` | `/api/login` | Log in, returns user plus access and refresh tokens |
| `POST` | `/api/refresh` | Exchange a valid refresh token (Bearer) for a new access token |
| `POST` | `/api/revoke` | Revoke a refresh token (Bearer) |

### Chirps

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/api/chirps` | Create a chirp. Requires Bearer JWT. Body: `{ "body" }` |
| `GET` | `/api/chirps` | List chirps. Optional `?author_id=<uuid>` and `?sort=asc\|desc` |
| `GET` | `/api/chirps/{chirpID}` | Fetch a single chirp |
| `DELETE` | `/api/chirps/{chirpID}` | Delete a chirp. Requires Bearer JWT; only the author may delete |

### Webhooks

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/api/polka/webhooks` | Marks a user as Chirpy Red on `user.upgraded`. Requires `Authorization: ApiKey <POLKA_KEY>` |

## Project layout

```
main.go                  HTTP handlers and server wiring
internal/auth/           password hashing, JWTs, refresh/API-key helpers
internal/database/       sqlc-generated queries and models
sql/schema/              goose migrations
sql/queries/             sqlc query definitions
index.html, assets/      static content served at /app/
```
