# Currency Exchange Feature Plan — Backend (go-dutch)

**Date:** 2026-04-21  
**Feature:** Currency Convert — new `Conversion` expense type, exchange rate caching, and balance logic to support cross-currency conversions

---

## Inputs

- Repo: `/Users/raytoh/code/go/src/github.com/raythx98/go-dutch`
- Exchange rate API: `https://v6.exchangerate-api.com/v6/{API_KEY}/latest/USD`
- API key: env var `EXCHANGE_RATE_API_KEY` (non-required; graceful degradation when absent)
- Frontend counterpart plan: `../go/src/github.com/raythx98/go-dutch` (consumed by `AddConversionModal`)

## Constraints

- At most one API call per day; results cached in `exchange_rate_snapshots` table
- On API failure (any HTTP error, network timeout, rate limit), fall back to existing stale snapshot silently
- No new Go module dependencies — use only packages already in `go.mod`
- Follow all existing code conventions: lowercase SQL, `timestamp default timezone('UTC', now())`, named `fk_` constraints, `shopspring/decimal` for amounts, `sqlc`-generated DB layer, `gqlgen` for GraphQL
- Do not commit or push without explicit approval

## Success Criteria

- `addConversion` mutation atomically creates two linked expense rows (source Repayment leg + target Conversion leg)
- `expenses` query hides the source leg from the list but includes it in balance calculations
- `deleteExpense` on a Conversion expense atomically soft-deletes both legs
- `exchangeRates` query returns fresh rates (≤24h old), refreshing from API if needed
- Concurrent `exchangeRates` calls across multiple server instances do not produce duplicate API calls (distributed advisory lock)
- Unsupported currencies (in DB but missing from API response) are surfaced in `unsupportedCurrencies` field

---

## files_to_change

| File | Action |
|------|--------|
| `migrations/000004_add_conversions.up.sql` | **New** |
| `migrations/000004_add_conversions.down.sql` | **New** |
| `sqlc/query.sql` | Add 10 new queries |
| `tools/config/config.go` | Add `ExchangeRateApiKey` to `Specification` |
| `tools/resources/resources.go` | Add `ExchangeRateKey string` to `Tools` struct |
| `tools/exchangerate/service.go` | **New** — rate fetch/cache service |
| `graphql/resolver.go` | Add `ExchangeRateSvc` field; update `NewResolver` |
| `graphql/expensetype.go` | Add `ExpenseTypeConversion = 2`; update string helpers |
| `graphql/schema/types.graphqls` | Add `ConversionDetails`, `ExchangeRate`, `ExchangeRateSnapshot`; add `conversionDetails` to `Expense` |
| `graphql/schema/inputs.graphqls` | Add `ConversionInput` |
| `graphql/schema/endpoint.graphqls` | Add `addConversion` mutation + `exchangeRates` query |
| `graphql/endpoint.resolvers.go` | New `AddConversion`, `ExchangeRates`; modify `DeleteExpense`, `Expenses` |

## new_files

- `migrations/000004_add_conversions.up.sql`
- `migrations/000004_add_conversions.down.sql`
- `tools/exchangerate/service.go`

---

## Phase 1 — Database Migration

### `migrations/000004_add_conversions.up.sql`

Follow existing style exactly: lowercase, `timestamp default timezone('UTC', now())`, named FK constraints, partial indexes.

```sql
create table if not exists exchange_rate_snapshots
(
    id                 bigserial primary key,
    base_currency_code varchar(3)                               not null,
    rates              jsonb                                    not null,
    fetched_at         timestamp default timezone('UTC', now()) not null,
    unique (base_currency_code)
);

create table if not exists conversions
(
    id                bigserial primary key,
    source_expense_id bigint                                   not null,
    target_expense_id bigint                                   not null,
    rate              decimal(20, 6)                           not null,
    created_at        timestamp default timezone('UTC', now()) not null,
    unique (source_expense_id),
    unique (target_expense_id),
    constraint fk_source_expense_id foreign key (source_expense_id) references expenses (id)
        on delete cascade on update cascade,
    constraint fk_target_expense_id foreign key (target_expense_id) references expenses (id)
        on delete cascade on update cascade
);

create index idx_conversions_target_expense_id on conversions (target_expense_id);
```

### `migrations/000004_add_conversions.down.sql`

```sql
drop table if exists conversions;
drop table if exists exchange_rate_snapshots;
```

**Decisions:**
- `decimal(20, 6)` for rate: 6 decimal places handle small cross-rates (e.g. VND/JPY ≈ 0.003) while following the `decimal(p, s)` convention from existing tables
- `unique (base_currency_code)` allows upsert-on-conflict to keep only the latest snapshot
- `unique` on both expense IDs in `conversions` prevents double-linking in concurrent creation

---

## Phase 2 — SQLC Queries

Append to `sqlc/query.sql`, then run `make sqlc`:

```sql
-- name: UpsertExchangeRateSnapshot :one
INSERT INTO exchange_rate_snapshots (base_currency_code, rates, fetched_at)
VALUES ($1, $2, NOW())
ON CONFLICT (base_currency_code)
DO UPDATE SET rates = EXCLUDED.rates, fetched_at = EXCLUDED.fetched_at
RETURNING *;

-- name: GetExchangeRateSnapshot :one
SELECT * FROM exchange_rate_snapshots WHERE base_currency_code = $1;

-- name: AcquireExchangeRateLock :exec
SELECT pg_advisory_lock($1);

-- name: ReleaseExchangeRateLock :exec
SELECT pg_advisory_unlock($1);

-- name: CreateConversion :one
INSERT INTO conversions (source_expense_id, target_expense_id, rate)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetConversionByTargetExpenseId :one
SELECT * FROM conversions WHERE target_expense_id = $1;

-- name: GetConversionBySourceExpenseId :one
SELECT * FROM conversions WHERE source_expense_id = $1;

-- name: GetConversionsByExpenseIds :many
SELECT * FROM conversions
WHERE target_expense_id = ANY($1::bigint[]);

-- name: GetSourceExpenseIdsForGroup :many
SELECT c.source_expense_id
FROM conversions c
JOIN expenses e ON e.id = c.source_expense_id
WHERE e.group_id = $1 AND e.is_deleted = false;

-- name: SoftDeleteConversionLegs :exec
UPDATE expenses SET is_deleted = true
WHERE id IN (
  SELECT source_expense_id FROM conversions WHERE target_expense_id = $1
  UNION ALL
  SELECT $1
);
```

After `make sqlc`, generated types for `ExchangeRateSnapshot` and `Conversion` will appear in `sqlc/db/models.go`. The `rates` column will be `pgtype.Text` or `[]byte` (JSONB) — marshal/unmarshal with `encoding/json`.

---

## Phase 3 — Config & Resources

### `tools/config/config.go`

Add to `Specification` struct:
```go
ExchangeRateApiKey string `envconfig:"EXCHANGE_RATE_API_KEY" required:"false"`
```

### `tools/resources/resources.go`

Add to `Tools` struct:
```go
ExchangeRateKey string
```

In `CreateTools`, populate it:
```go
ExchangeRateKey: cfg.ExchangeRateApiKey,
```

---

## Phase 4 — Exchange Rate Service

### `tools/exchangerate/service.go`

**Package:** `exchangerate`

**Dependencies (all already in go.mod):**
- `github.com/jackc/pgx/v5/pgxpool` — pool for pinned connections
- `github.com/raythx98/go-dutch/sqlc/db` — generated queries
- `encoding/json`, `net/http`, `sync`, `time`

```go
package exchangerate

const exchangeRateLockKey int64 = 8200001 // unique advisory lock key for this operation

type Service struct {
    pool    *pgxpool.Pool
    poolQry *db.Queries  // for fast path reads (shared pool connection)
    apiKey  string
    log     logger.ILogger
}

func NewService(pool *pgxpool.Pool, apiKey string, log logger.ILogger) *Service {
    return &Service{
        pool:    pool,
        poolQry: db.New(pool),
        apiKey:  apiKey,
        log:     log,
    }
}

type Result struct {
    FetchedAt             time.Time
    Rates                 map[string]float64
    UnsupportedCurrencies []string
}
```

**`GetOrRefresh(ctx context.Context) (*Result, error)`** — full flow:

```
Step 1 — Fast path (no lock, uses shared pool connection):
  snap, err := s.poolQry.GetExchangeRateSnapshot(ctx, "USD")
  if err == nil && time.Since(snap.FetchedAt) < 24h:
      return s.buildResult(ctx, snap)

Step 2 — Acquire dedicated connection (advisory lock is session-scoped):
  conn, err := s.pool.Acquire(ctx)
  defer conn.Release()
  qtx := db.New(conn)

Step 3 — Acquire distributed advisory lock (blocking):
  qtx.AcquireExchangeRateLock(ctx, exchangeRateLockKey)
  defer qtx.ReleaseExchangeRateLock(ctx, exchangeRateLockKey)
  // All other instances/goroutines queue here; only one proceeds

Step 4 — Double-check inside lock (another instance may have already fetched):
  snap, err := qtx.GetExchangeRateSnapshot(ctx, "USD")
  if err == nil && time.Since(snap.FetchedAt) < 24h:
      return s.buildResult(ctx, snap)

Step 5 — Fetch from API:
  ratesJSON, err := s.fetchFromAPI(ctx)
  if err != nil:
      s.log.Warn("exchange rate fetch failed, using cached values", "error", err)
      if stale, err := qtx.GetExchangeRateSnapshot(ctx, "USD"); err == nil:
          return s.buildResult(ctx, stale)  // fall back to previous day
      return nil, err  // no snapshot at all — propagate error

Step 6 — Upsert into DB:
  stored, err := qtx.UpsertExchangeRateSnapshot(ctx, db.UpsertExchangeRateSnapshotParams{
      BaseCurrencyCode: "USD",
      Rates:            ratesJSON,
  })

Step 7 — Return result:
  return s.buildResult(ctx, stored)
```

**`fetchFromAPI(ctx) (json.RawMessage, error)`**:
- If `s.apiKey == ""` return an error (no key configured)
- `GET https://v6.exchangerate-api.com/v6/{apiKey}/latest/USD` with `http.NewRequestWithContext` and 10-second timeout via `context.WithTimeout`
- Decode JSON: `{ "result": "success", "conversion_rates": { "USD": 1.0, "SGD": 1.34, ... } }`
- On non-200 or `result != "success"`, return an error
- Return the raw `conversion_rates` JSON bytes

**`buildResult(ctx, snap) (*Result, error)`**:
- Unmarshal `snap.Rates` (JSONB) into `map[string]float64`
- Fetch app currencies: `s.poolQry.GetCurrencies(ctx)` → build set of codes
- Compute `UnsupportedCurrencies`: any currency code in DB not present as a key in the rates map
- Log each unsupported code at WARN level
- Return `&Result{FetchedAt: snap.FetchedAt, Rates: rates, UnsupportedCurrencies: unsupported}`

**Why connection pinning:**  
`pg_advisory_lock` is session-scoped in PostgreSQL. Using `pgxpool.Pool` directly could route `AcquireExchangeRateLock` and `ReleaseExchangeRateLock` to different connections, leaving the lock permanently held. Acquiring a single `pgxpool.Conn` and passing it to `db.New(conn)` pins all operations to one session.

---

## Phase 5 — GraphQL Schema

### `graphql/schema/types.graphqls` — append:

```graphql
type ConversionDetails {
    sourceCurrency: Currency!
    sourceAmount: Decimal!
    rate: Decimal!
}

type ExchangeRate {
    code: String!
    rate: Decimal!
}

type ExchangeRateSnapshot {
    base: String!
    rates: [ExchangeRate!]!
    fetchedAt: Time!
    unsupportedCurrencies: [String!]!
}
```

Add optional field to existing `Expense` type:
```graphql
conversionDetails: ConversionDetails
```

### `graphql/schema/inputs.graphqls` — append:

```graphql
input ConversionInput {
    name: String!
    description: String!
    sourceAmount: Decimal!
    sourceCurrencyId: ID!
    targetAmount: Decimal!
    targetCurrencyId: ID!
    expenseAt: Time!
    debtorId: ID!
    creditorId: ID!
}
```

### `graphql/schema/endpoint.graphqls` — add to Mutation and Query:

```graphql
# Mutation:
addConversion(groupId: ID!, input: ConversionInput!): Expense! @auth

# Query:
exchangeRates: ExchangeRateSnapshot! @auth
```

Run `make gqlgen` after all schema changes.

**`gqlgen.yml` note:** No changes needed. `conversionDetails` is a nullable struct pointer field (`*model.ConversionDetails`) on `model.Expense`. gqlgen includes it automatically and resolvers set it inline when building the response — no separate field resolver function required.

---

## Phase 6 — Expense Type

### `graphql/expensetype.go`

```go
const (
    ExpenseTypeGeneric    int16 = iota // 0
    ExpenseTypeRepayment               // 1
    ExpenseTypeConversion              // 2
)
```

Update `expenseTypeString`:
```go
case ExpenseTypeConversion:
    return "Conversion"
```

Update `expenseTypeFromString`:
```go
case "Conversion":
    return ExpenseTypeConversion
```

---

## Phase 7 — Resolver Wiring

### `graphql/resolver.go`

```go
type Resolver struct {
    resources.Tools
    DbQuery         *db.Queries
    ExchangeRateSvc *exchangerate.Service
}

func NewResolver(tools resources.Tools) *Resolver {
    return &Resolver{
        Tools:           tools,
        DbQuery:         db.New(tools.Db.Pool()),
        ExchangeRateSvc: exchangerate.NewService(tools.Db.Pool(), tools.ExchangeRateKey, tools.Log),
    }
}
```

---

## Phase 8 — Resolver Implementations

### `AddConversion` (new — stub generated by gqlgen)

Location: `graphql/endpoint.resolvers.go`

```
1. checkIsGroupMember(ctx, r.DbQuery, groupID, getActionTaker(ctx))
2. Validate input.SourceCurrencyId != input.TargetCurrencyId
3. Begin pgx transaction
4. Inside qtx:
   a. Validate both currencies exist (GetCurrenciesByIds or individual gets)
   b. Create source expense:
      - type = ExpenseTypeRepayment
      - name = "[Conversion Source]"
      - description = ""
      - currency = sourceCurrencyId
      - amount = sourceAmount
      - expense_at = input.ExpenseAt
      - Payer record: {creditorId, sourceAmount}
      - Share record: {debtorId, sourceAmount}
   c. Create target expense:
      - type = ExpenseTypeConversion
      - name = input.Name
      - description = input.Description
      - currency = targetCurrencyId
      - amount = targetAmount
      - expense_at = input.ExpenseAt
      - Payer record: {creditorId, targetAmount}
      - Share record: {debtorId, targetAmount}
   d. Compute rate using shopspring/decimal:
      rate = decimal.NewFromString(input.TargetAmount.String()).Div(
               decimal.NewFromString(input.SourceAmount.String()))
   e. CreateConversion(ctx, sourceExpense.ID, targetExpense.ID, pghelper.FromDecimal(rate))
5. Commit transaction
6. Build and return *model.Expense for target expense with ConversionDetails:
   ConversionDetails: &model.ConversionDetails{
       SourceCurrency: toCurrencyModel(sourceCurrency),
       SourceAmount:   input.SourceAmount,
       Rate:           rate,
   }
```

Pattern reference: follow `AddRepayment` resolver (lines 220–316) for transaction structure and payer/share creation pattern.

### `DeleteExpense` (modify existing — lines 415–434)

Insert before the existing `r.DbQuery.DeleteExpense` call:

```go
// Cascade-delete both legs of a conversion
if expense.Type == ExpenseTypeConversion {
    tx, err := r.Db.Pool().Begin(ctx)
    if err != nil { return false, err }
    defer tx.Rollback(ctx)
    if err := r.DbQuery.WithTx(tx).SoftDeleteConversionLegs(ctx, expenseID); err != nil {
        return false, err
    }
    return true, tx.Commit(ctx) == nil
}

// Block direct deletion of hidden source leg
if expense.Type == ExpenseTypeRepayment {
    if _, err := r.DbQuery.GetConversionBySourceExpenseId(ctx, expenseID); err == nil {
        return false, errorhelper.NewAppError(ExpenseIsConversionSourceLeg, Messages[ExpenseIsConversionSourceLeg], nil)
    } else if !errors.Is(err, pgx.ErrNoRows) {
        return false, err
    }
}
```

Add to `graphql/errorcodes.go`:
```go
ExpenseIsConversionSourceLeg // e.g. = iota value
```
And its message: `"This expense is part of a currency conversion and cannot be deleted directly"`.

### `Expenses` (modify existing query resolver)

**After fetching raw expenses list, before building the display and balance structures:**

```go
// 1. Fetch source leg IDs to exclude from display
sourceIds, err := r.DbQuery.GetSourceExpenseIdsForGroup(ctx, groupID)
if err != nil { return nil, err }
sourceIdSet := make(map[int64]bool, len(sourceIds))
for _, id := range sourceIds { sourceIdSet[id] = true }

// 2. Fetch conversion metadata for Conversion-type expenses
var convTargetIds []int64
for _, e := range expenses {
    if e.Type == ExpenseTypeConversion { convTargetIds = append(convTargetIds, e.ID) }
}
convMap := make(map[int64]db.Conversion)
if len(convTargetIds) > 0 {
    convs, err := r.DbQuery.GetConversionsByExpenseIds(ctx, convTargetIds)
    if err != nil { return nil, err }
    for _, c := range convs { convMap[c.TargetExpenseID] = c }
}
```

**Balance computation:** Use ALL expenses (unfiltered) — source Repayment legs must contribute to the balance calculation to cancel the old-currency balance. The existing balance loop runs before the display filter.

**Display list construction:** Skip source legs; attach `ConversionDetails` to Conversion expenses:

```go
// In the expenses loop that builds model.Expense objects:
if sourceIdSet[expense.ID] { continue }  // hide source leg from display

if expense.Type == ExpenseTypeConversion {
    if conv, ok := convMap[expense.ID]; ok {
        srcExp, err := r.DbQuery.GetExpense(ctx, conv.SourceExpenseID)
        if err == nil {
            expModel.ConversionDetails = &model.ConversionDetails{
                SourceCurrency: toCurrencyModel(currencyMap[srcExp.CurrencyID]),
                SourceAmount:   pghelper.Decimal(srcExp.Amount),
                Rate:           pghelper.Decimal(conv.Rate),
            }
        }
    }
}
```

### `ExchangeRates` (new — stub generated by gqlgen)

```go
func (r *queryResolver) ExchangeRates(ctx context.Context) (*model.ExchangeRateSnapshot, error) {
    result, err := r.ExchangeRateSvc.GetOrRefresh(ctx)
    if err != nil { return nil, err }

    rates := make([]*model.ExchangeRate, 0, len(result.Rates))
    for code, rate := range result.Rates {
        rates = append(rates, &model.ExchangeRate{
            Code: code,
            Rate: decimal.NewFromFloat(rate),
        })
    }
    sort.Slice(rates, func(i, j int) bool { return rates[i].Code < rates[j].Code })

    return &model.ExchangeRateSnapshot{
        Base:                  "USD",
        Rates:                 rates,
        FetchedAt:             result.FetchedAt,
        UnsupportedCurrencies: result.UnsupportedCurrencies,
    }, nil
}
```

---

## Implementation Order

1. Write `migrations/000004_add_conversions.up.sql` + `.down.sql`
2. Append queries to `sqlc/query.sql` → run `make sqlc`
3. `tools/config/config.go` — add `ExchangeRateApiKey`
4. `tools/resources/resources.go` — add `ExchangeRateKey` to `Tools`
5. `tools/exchangerate/service.go` — new service (no gqlgen dependency; can compile standalone)
6. `graphql/schema/types.graphqls`, `inputs.graphqls`, `endpoint.graphqls` — schema edits
7. Run `make gqlgen` — generates stubs for `AddConversion` and `ExchangeRates`
8. `graphql/expensetype.go` — add `ExpenseTypeConversion = 2`
9. `graphql/errorcodes.go` — add `ExpenseIsConversionSourceLeg`
10. `graphql/resolver.go` — wire `ExchangeRateSvc`
11. `graphql/endpoint.resolvers.go` — implement `AddConversion`, `ExchangeRates`; modify `DeleteExpense`, `Expenses`
12. `go build ./...` — fix all compile errors
13. Apply migration to local DB; smoke-test with Altair or curl

---

## Edge Cases

| Case | Handling |
|------|----------|
| sourceCurrencyId == targetCurrencyId | Validate before DB; return app error |
| sourceAmount = 0 | Block — division by zero computing rate |
| API key empty | Skip API call; use stale snapshot; if none exists, return error to client |
| API returns non-200 or rate-limit (429) | Log warning; fall back to existing snapshot; if no snapshot, propagate error |
| Supported currency absent from API response | Logged at WARN; surfaced in `unsupportedCurrencies` |
| Concurrent `exchangeRates` calls (same instance) | Advisory lock serialises — double-check inside lock avoids duplicate API call |
| Concurrent `exchangeRates` calls (multiple instances) | PostgreSQL advisory lock is distributed — same double-check pattern holds |
| Direct deletion of source leg | Blocked — `GetConversionBySourceExpenseId` returns a row → return `ExpenseIsConversionSourceLeg` error |
| Balance includes source legs | Balance loop runs on unfiltered `expenses` slice; display list applies filter separately |
| JSONB rates column type from sqlc | Check generated `models.go`; may be `pgtype.Text` or `[]byte` — use `encoding/json` to marshal in and unmarshal out |
| rate decimal precision | Computed with `shopspring/decimal` (no float64); stored as `decimal(20, 6)`; displayed as 6 decimal places |
| toCurrencyModel helper | Reuse existing helper in `graphql/helper.go` or `endpoint.resolvers.go` — check exact function signature before calling |

---

## Verification

```bash
# Apply migration
migrate -path migrations -database "postgres://..." up

# Rebuild
go build ./...

# Smoke test (use Altair or curl against localhost:8080/query)
# 1. Call exchangeRates query → verify rates array populated, fetchedAt recent
# 2. Call exchangeRates again immediately → no second API call (check logs)
# 3. Add a Generic expense in SGD, verify balance appears
# 4. Call addConversion → verify two rows in expenses table, one in conversions
# 5. Call expenses(groupId) → source leg absent from list, Conversion present with conversionDetails
# 6. Verify SGD balance cleared, USD balance created
# 7. Call deleteExpense on Conversion → verify both expense rows is_deleted=true
# 8. Attempt deleteExpense on source leg ID directly → expect error
```

## Future Work

- Allow editing a Conversion expense (currently blocked; user must delete and re-create)
- Support partial conversion (convert only a portion of the outstanding balance)
- Periodic background job to pre-warm exchange rate cache at midnight rather than on first request
- Admin endpoint to manually trigger exchange rate refresh
