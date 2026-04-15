# Architecture

## Overview

go-dutch is a GraphQL API server for group expense management — the backend for the Dutch SPA frontend. It handles user authentication, group management, expense tracking, and balance calculation. Built in Go using `gqlgen` for schema-first GraphQL and `sqlc` for type-safe SQL.

## Layer Structure

```
┌──────────────────────────────────────────────────────────────┐
│  HTTP Server  (server.go)                                     │
│  Middleware chain: CORS · rate limit · request ID            │
│  JWT extraction · structured logging                         │
├──────────────────────────────────────────────────────────────┤
│  GraphQL Layer  (graphql/)                                    │
│  gqlgen-generated handler + schema + resolver implementations│
├──────────────────────────────────────────────────────────────┤
│  Resolver Layer  (graphql/*.resolvers.go)                     │
│  Business logic: group management, expense ops, auth         │
├──────────────────────────────────────────────────────────────┤
│  Database Layer  (sqlc/db/)                                   │
│  SQLC-generated type-safe query functions                    │
├──────────────────────────────────────────────────────────────┤
│  Infrastructure (tools/)                                      │
│  config · postgres · crypto · zerologger · resources        │
└──────────────────────────────────────────────────────────────┘
```

## Request Data Flow

```
HTTP POST /query
  │
  ├── Middleware chain (server.go)
  │     ├── CORS
  │     ├── Rate limiting (token bucket, per YAML config)
  │     ├── Add request ID
  │     ├── Request context setup (reqctx)
  │     ├── JWT subject extraction (from Authorization header)
  │     └── Structured logging (zerolog)
  │
  ├── gqlgen GraphQL handler
  │     ├── @auth directive check (resolver-level authorization)
  │     └── Dispatch to resolver method
  │
  ├── Resolver
  │     ├── Validate inputs
  │     ├── Call sqlc query functions (db.Queries)
  │     └── Return typed response
  │
  └── GraphQL response serialized and returned
```

## Dependency Wiring

`tools/resources/resources.go` is the composition root. It initializes and wires:
- `Logger` (zerolog)
- `Postgres` (pgx connection pool)
- `JWT` helper
- `Crypto` helper (password hashing)
- `Resolver` struct embedding all tools + `db.Queries`

The `Resolver` struct is the only thing the GraphQL handler needs.

## Authentication

- JWT tokens are issued on `login` and `register` mutations.
- The `@auth` directive on schema fields enforces authentication.
- JWT subject (user ID) is extracted from the `Authorization: Bearer <token>` header by the middleware and stored in request context via `reqctx`.
- Resolvers read the user ID from context — they never re-validate the token.

## Balance Calculation

Balance settlement uses a greedy algorithm:
1. Aggregate all expense shares to compute net balances per user.
2. Separate users into creditors (owed money) and debtors (owe money).
3. Greedily match the largest debtor to the largest creditor until all balances are settled.

This minimizes the number of transactions needed to settle all debts.

## Code Generation

| Tool | Config | Input | Output |
|------|--------|-------|--------|
| `gqlgen` | `gqlgen.yml` | `graphql/schema/*.graphqls` | `graphql/generated.go`, `graphql/model/` |
| `sqlc` | `sqlc/sqlc.yaml` | `sqlc/query.sql` | `sqlc/db/` |

**Never manually edit generated files.** Regenerate after schema or query changes.

## Rate Limiting

Per-operation token bucket rate limiting is configured in `ratelimit.yaml`. The middleware extracts the GraphQL operation name from the request body to look up the per-operation limit. Anonymous requests fall back to IP-based limiting.
