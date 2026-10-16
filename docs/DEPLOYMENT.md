# Deploying the backend (Railway)

The container builds from `Dockerfile`, applies pending migrations on start,
then serves. Data comes from a one-time restore of a dump — the schema
migrates itself, but rows do not.

## Why the first deploy crash-looped

The container had no database variables, so `config.LoadConfig` fell back to
its local defaults and the app dialled `127.0.0.1:5432` *inside its own
container*, where nothing listens:

```
dial tcp 127.0.0.1:5432: connect: connection refused
Unable to ping the database: failed to connect to `user=postgres database=dailzo`
```

Three things made that possible, all fixed:

- nothing read `DATABASE_URL`, which is the only connection detail Railway
  gives you;
- nothing read `PORT`, so the app bound 3000 while Railway routed elsewhere;
- the error printed the driver's message but never said which host it had
  tried, so a misconfigured deploy looked like a dead database.

## One-time setup

1. **Add Postgres.** In the Railway project: *New → Database → PostgreSQL*.

2. **Point the service at it.** On the backend service's *Variables* tab:

   | Variable | Value |
   | --- | --- |
   | `DATABASE_URL` | `${{Postgres.DATABASE_URL}}` — reference the DB service, don't paste a literal |
   | `JWT_SECRET` | 32+ random characters: `openssl rand -base64 48` |
   | `APP_ENV` | `production` |
   | `CORS_ORIGINS` | `none` — the two Flutter apps send no `Origin` header, so CORS never applies to them; use real origins only when a browser frontend calls this API |
   | `SMTP_HOST` / `SMTP_PORT` / `SMTP_EMAIL` / `SMTP_PASSWORD` | as in `.env.example` |

   Do **not** set `PORT` — Railway injects it. Do not set the `DB_*` variables
   either; `DATABASE_URL` wins and mixing them invites disagreement.

   Railway's Postgres needs TLS. `DATABASE_URL` already carries the right
   `sslmode`, which is another reason to prefer it over the parts.

3. **Deploy.** Push to the branch the service tracks. Railway reads
   `railway.json`, builds the Dockerfile, and health-checks `/api/v1/health`.
   The first boot creates `schema_migrations` and applies all 28 migrations,
   starting from `000_base_schema.sql`.

   That base migration exists because the core schema — `users`,
   `restaurants`, `food_products`, `orders` — used to live only in
   `DatabaseScripts&ERD/DailzoScript.sql` and was applied by hand to each
   developer's database. An empty managed Postgres had none of it, so
   `001_new_tables.sql` died on its first foreign key:
   `relation "public.users" does not exist`. It is generated from that ERD
   script by `scripts/generate_base_migration.py`; regenerate it there rather
   than editing it, and it is safe to apply to a database that was already
   set up by hand.

4. **Load the data, once.**

   ```sh
   ./scripts/db_dump.sh                     # writes dailzo-<date>.dump.sql
   ./scripts/db_restore.sh dailzo-<date>.dump.sql "<DATABASE_PUBLIC_URL>"
   ```

   Take `DATABASE_PUBLIC_URL` from the Postgres service's *Variables* tab —
   the internal `DATABASE_URL` only resolves from inside Railway's network.
   The restore asks you to retype the host before it touches anything, and
   re-checks the vendor-attribution invariants afterwards.

   The dump holds real rows, including bcrypt password hashes. `.gitignore`
   excludes `*.dump.sql`; keep it that way.

## Every deploy after that

Push. The build runs, migrations apply, the health check gates the cutover.
Adding a migration means dropping a numbered `.sql` into `migrations/` — the
runner applies what it has not already recorded, so redeploying unchanged code
re-runs nothing.

## Checking a deploy

```sh
curl https://<your-service>.up.railway.app/api/v1/health
# {"service":"dailzo-backend","status":"ok"}
```

Then point the apps at it: `baseUrl` in the customer app and
`DioClient` in the Partner app, replacing `http://10.0.2.2:2193`.

## Running the image locally

Exactly what Railway runs, against your own Postgres:

```sh
docker build -t dailzo-backend .
docker run --rm -p 8080:8080 \
  -e DATABASE_URL="postgres://postgres:<pw>@host.docker.internal:5432/dailzo?sslmode=disable" \
  -e PORT=8080 \
  -e JWT_SECRET="$(openssl rand -base64 48)" \
  dailzo-backend
```

## Startup guards

The app refuses to start, rather than running misconfigured, when:

- `JWT_SECRET` is unset. Under 16 characters it starts but warns — a short
  signing key is worth fixing before it signs anything you care about
  (`openssl rand -base64 48`). Changing it invalidates every issued token.
- `APP_ENV=production` and `CORS_ORIGINS` is unset or `*`. A wide-open origin
  combined with `Authorization`-header requests is the combination worth
  refusing. Set real origins, or `none` for a native-app-only API.

Both are deliberate: a container that boots and quietly serves with a weak
secret or an open CORS policy is worse than one that will not boot.

## Secrets

`.env` was committed to this repository and pushed to GitHub before it was
ignored, so the database password, the SMTP password and the old `JWT_SECRET`
are in the git history and must be treated as public. Rotate all three, and
use a new `JWT_SECRET` for the deployment rather than the one from `.env` —
`git rm --cached` stops tracking the file but does not remove it from history.
