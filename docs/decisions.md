# Decisions

Architecture decision records — why things are the way they are.

---

## ADR-001: GraphQL over REST

**Decision**: The API uses GraphQL (gqlgen, schema-first) rather than REST.

**Rationale**: The frontend (Dutch SPA) needs to fetch nested data structures (group with members, expenses with payers and shares, balances with owed-to relationships) in a single request. GraphQL's nested query model eliminates multiple round trips and avoids over-fetching.

**Tradeoff**: GraphQL adds complexity: schema management, code generation, field-level resolvers. Justified by the data shape requirements.

---

## ADR-002: Schema-first GraphQL with gqlgen

**Decision**: gqlgen with schema-first generation is used rather than code-first.

**Rationale**: The GraphQL schema is the API contract between frontend and backend. Defining it first in `.graphqls` files makes the contract explicit and version-controllable. Code-first approaches bury the schema in Go annotations.

**Tradeoff**: Schema changes require running `make gqlgen`. Developers must regenerate code after schema edits — forgetting this causes confusing compilation errors.

---

## ADR-003: SQLC for database access

**Decision**: Database queries use SQLC-generated code rather than an ORM (e.g., GORM) or raw SQL.

**Rationale**: SQLC generates type-safe Go functions from `.sql` files. This provides compile-time query validation, no reflection overhead, and explicit SQL visibility (no hidden queries). GORM's implicit query generation has led to N+1 issues in past projects.

**Tradeoff**: SQL must be written manually. Migrations are separate from query definitions. Slightly more verbose than ORM alternatives.

---

## ADR-004: `shopspring/decimal` for currency amounts

**Decision**: All monetary amounts use `shopspring/decimal.Decimal` rather than `float64` or `int` cents.

**Rationale**: Floating-point arithmetic is unsuitable for money. `decimal` provides exact arithmetic with configurable precision. The custom GraphQL scalar maps `Decimal` to/from string in the API.

**Tradeoff**: More verbose than `float64`. All arithmetic uses method calls (`Add`, `Mul`, etc.) rather than operators.

---

## ADR-005: Greedy algorithm for balance settlement

**Decision**: Balance settlement uses a greedy algorithm (largest debtor → largest creditor matching).

**Rationale**: Minimizes the number of transactions needed to settle all group debts. Simple to implement and understand. Optimal for small groups (< 20 members).

**Tradeoff**: Not strictly optimal for all cases — a more complex algorithm could produce fewer transactions in certain debt graphs. Not a meaningful concern for personal expense groups.

---

## ADR-006: JWT stored in client, no refresh tokens

**Decision**: Only access tokens are issued (no refresh tokens). Token expiry requires re-login.

**Rationale**: This is a personal tool. Short session management complexity is not worth the added code for refresh token rotation. Access tokens have a configurable TTL.

**Tradeoff**: Users must re-login when the token expires. Acceptable for a low-frequency personal app.

---

## ADR-007: `@auth` directive for authorization

**Decision**: GraphQL field-level authorization uses the `@auth` directive rather than middleware-only enforcement.

**Rationale**: Directive-based authorization is explicit in the schema — the contract is visible to any consumer. Middleware-only enforcement is invisible and harder to audit.

**Tradeoff**: Requires custom directive logic in `server.go`.

---

## ADR-008: citext for case-insensitive email/username

**Decision**: The `users` table uses the PostgreSQL `citext` extension for the `username` and `email` columns.

**Rationale**: Case-insensitive comparison without explicit `LOWER()` calls. Prevents duplicate accounts from `User@example.com` vs `user@example.com`. Handled at the database level — no application-level normalization needed.

---

## ADR-009: Soft deletes via `is_deleted`

**Decision**: Expense and group records use a boolean `is_deleted` column rather than hard deletion.

**Rationale**: Expense records are referenced in balance calculations. Hard deletion would corrupt historical balance data. Soft deletes preserve the audit trail.

**Tradeoff**: All queries must include `WHERE is_deleted = false` filters. SQLC-generated queries handle this consistently via function-based partial indexes.
