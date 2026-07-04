# Dailzo Backend

The Go API server for **Dailzo**, a food + grocery delivery platform. It powers ordering, menus, restaurants, marketing/offers, delivery tracking, and admin operations, and is consumed by both mobile apps.

## Part of the Dailzo platform (3 repos)

| Repo | What it is |
| --- | --- |
| **dailzo** (this repo) | Go backend / REST API + WebSocket delivery tracking |
| [dailzo_vendor](../dailzo_vendor) | Flutter app for restaurant / grocery / delivery **partners** |
| [dailzocustomerapp](../dailzocustomerapp) | Flutter app for **customers** ordering food & grocery |

Both Flutter apps talk to this server at `http://<host>:2193/api/v1`.

## Tech stack

- **Go 1.24** (toolchain 1.24.1)
- **Fiber v2** web framework
- **pgx/v5** + **PostgreSQL** (database name: `dailzo`)
- **golang-jwt/jwt v4** for JWT auth
- **zerolog** logging, **godotenv** config
- Firebase Admin SDK (FCM push), SMTP (email/OTP), WebSockets (delivery tracking)

## Prerequisites

- Go 1.24+
- A running PostgreSQL instance with a `dailzo` database
- `psql` CLI (migrations are applied manually)
- (optional) [`air`](https://github.com/air-verse/air) for hot reload

## Setup

1. **Environment.** Create a `.env` file in the repo root. It is git-ignored — **do not commit it.**

   ```env
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=your_password
   DB_NAME=dailzo
   APP_PORT=2193
   JWT_SECRET=use-a-long-random-secret
   SMTP_HOST=smtp.example.com
   SMTP_PORT=587
   SMTP_EMAIL=you@example.com
   SMTP_PASSWORD=your_smtp_password
   ```

   Use a strong, random `JWT_SECRET` — it signs all auth tokens.

2. **Database + migrations.** Create the database, then apply migrations **in order** with `psql` (there is no automatic migrator):

   ```bash
   createdb dailzo
   psql -d dailzo -f migrations/001_new_tables.sql
   psql -d dailzo -f migrations/002_marketing_and_vendor_ops.sql
   psql -d dailzo -f migrations/003_offers_and_registration.sql
   ```

3. **Dependencies.**

   ```bash
   go mod download
   ```

## Run

```bash
go run main.go        # runs on APP_PORT (default 2193)
```

Or with hot reload:

```bash
air                   # config in .air.toml
```

## API surfaces

The server exposes **two** REST surfaces plus a WebSocket:

- **Legacy** `/api/*` — hand-written routes registered in `routes/routes.go` (`SetupRoutes`).
- **OpenAPI v1** `/api/v1/*` — generated from the spec into `internal/api/api.gen.go`, implemented by handlers in `internal/server/*_handlers.go`. This is what the mobile apps use.
- **WebSocket** — real-time delivery tracking (`internal/websocket`).

Auth endpoints under `/api/v1/auth/*` are **public** (allowlisted past the JWT middleware in `main.go`); all other v1 routes require a valid JWT. All responses use a common envelope from `pkg/response`: `{ success, message, data, error }`.

The OpenAPI spec lives at `api/openapi.yaml`.

## Project structure

```
main.go               App entry: config, DB, repos, routes, JWT middleware, server
config/               Config loading + logger setup
db/                   pgxpool connection (db.go)
routes/               Legacy /api/* route registration (routes.go)
internal/api/         OpenAPI-generated code (api.gen.go)
internal/server/      v1 handlers (auth, order, product, delivery, grocery, admin)
internal/websocket/   WebSocket delivery tracking
controllers/          Fiber HTTP handlers (auth/users, orders, food_products,
                      restaurants, offers, marketing, payments, ratings, ...)
repository/           pgx SQL data access
models/               Request/response and domain structs
middleware/           JWTMiddleware and others
pkg/response/         ApiResponse envelope
globals/              Shared globals
migrations/           Raw .sql migrations (apply via psql, in order)
utils/                Helpers
```

## Notable modules

Auth/users, orders, food products, restaurants, offers, marketing (discounts + ad campaigns + ad packs), admin (approvals / partners / complaints), delivery, and grocery.

## Gotchas

- Migrations are **not** auto-applied — run them manually with `psql`, in numeric order.
- The mobile apps target `10.0.2.2:2193` (Android emulator's alias for the host machine), so keep `APP_PORT=2193`.
- A prebuilt `dailzo.exe` and `go.sum` are checked in; rebuild from source rather than trusting the binary.
