# Integrations

External services — contracts, auth, and failure modes.

---

## PostgreSQL

**Used for**: all persistent data — users, groups, members, expenses, shares, currencies.

**Auth**: username/password from environment variables (`APP_GODUTCH_DBUSERNAME`, `APP_GODUTCH_DBPASSWORD`). Connection via `pgx/v5` pool.

**Schema**: managed by golang-migrate. Tables: `users`, `groups`, `group_members`, `expenses`, `expense_payers`, `expense_shares`, `currencies`.

**Failure modes**:
- Connection failure at startup → fatal log, server does not start.
- Query timeout or transient failure → operation returns an error; GraphQL returns an error response.
- No automatic query retry at the application level.

**Config vars**: `APP_GODUTCH_DBHOST`, `APP_GODUTCH_DBPORT`, `APP_GODUTCH_DBUSERNAME`, `APP_GODUTCH_DBPASSWORD`, `APP_GODUTCH_DBDEFAULTNAME`.

---

## Dutch Frontend (SPA)

**Used as**: the sole consumer of this GraphQL API.

**Auth**: JWT Bearer token in `Authorization` header. Tokens are issued by the `login` and `register` mutations and stored in the frontend's `localStorage`.

**Contract**: the `.graphqls` schema files define the API contract. The frontend's `schema.graphql` (at the root of the `dutch` repo) must be kept in sync.

**Failure modes**:
- Token expiry → GraphQL returns a `401` error code; frontend redirects to login.
- Network partition → frontend shows a generic error toast.

---

## Docker Compose Stack

**Components**:

| Service | Purpose |
|---------|---------|
| `db` | PostgreSQL 17 database |
| `migrate` | Applies golang-migrate migrations on startup |
| `app` | The go-dutch server |
| `caddy` | Reverse proxy for HTTPS/TLS termination |

**Startup order**: `db` → `migrate` → `app` → `caddy`.

**Health checks**: `migrate` waits for `db` to accept connections before applying migrations. `app` depends on `migrate` completing successfully.

**Failure modes**:
- `db` fails to start → `migrate` and `app` never start.
- Migrations fail → `app` starts but may encounter schema errors at runtime.
- `caddy` fails → app is still reachable directly on port 8080.

---

## golang-migrate

**Used for**: managing database schema evolution.

**Contract**: two-file convention per migration — `NNNNNN_name.up.sql` and `NNNNNN_name.down.sql`.

**Failure modes**:
- Migration fails mid-run → the migration is left in a dirty state; manual intervention required (`migrate force <version>`).
- Missing down migration → rollback is impossible for that version.

**New migration**: `make create_migration name=<descriptive_name>`.

---

## Caddy (Reverse Proxy)

**Used for**: TLS termination (Let's Encrypt) and HTTP→HTTPS redirect.

**Auth**: none (Caddy handles TLS, traffic forwarded to app on port 8080).

**Config**: `Caddyfile` at the repo root. The domain is set via the `DOMAIN` environment variable (or hardcoded for the specific deployment).

**Failure modes**:
- Certificate issuance failure → HTTPS unavailable; HTTP fallback may work depending on config.
- Caddy container restart → brief connection interruption; auto-recovers.
