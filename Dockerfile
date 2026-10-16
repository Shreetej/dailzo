# syntax=docker/dockerfile:1

# ── build ────────────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS build
WORKDIR /src

# Dependencies first, so a code-only change doesn't re-download the module
# cache on every deploy.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO off → a static binary that runs on the scratch-like runtime below.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server . \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ── runtime ──────────────────────────────────────────────────────────────────
FROM alpine:3.20
# ca-certificates: outbound TLS (SMTP, payment gateways). tzdata: order
# timestamps and operating hours are local-time sensitive.
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 dailzo
WORKDIR /app

COPY --from=build /out/server /out/migrate /app/
# cmd/migrate reads migrations/ relative to the working directory, so the SQL
# ships with the image rather than being fetched at deploy time.
COPY migrations /app/migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

USER dailzo
EXPOSE 3000
ENTRYPOINT ["/app/docker-entrypoint.sh"]
