-- name: GetCurrencies :many
select *
from currencies
where is_deleted = false
order by code asc;

-- name: GetUserGroup :one
select *
from user_group
where user_id = $1
  and group_id = $2
  and is_deleted = false;

-- name: GetCurrenciesByIds :many
select *
from currencies
where is_deleted = false
  and id = ANY ($1:: bigint[]);

-- name: CreateUser :one
insert into users (username, email, password)
values ($1, $2, $3)
returning *;

-- name: GetUserByUsername :one
select *
from users
where username = $1
    and is_deleted = false;

-- name: GetUserByEmail :one
select *
from users
where email = $1
  and is_deleted = false;

-- name: GetUserByUsernameOrEmail :one
SELECT *
FROM users
WHERE (username = $1 OR email = $1)
  AND is_deleted = false;

-- name: GetUsersByIds :many
select *
from users
where id = ANY ($1:: bigint[])
  and is_deleted = false;

-- name: GetGroupMembers :many
select u.*
from users u
         join user_group ug on u.id = ug.user_id
where ug.group_id = $1
  and u.is_deleted = false
  and ug.is_deleted = false;

-- name: CreateGroup :one
insert into groups (name, invite_token)
values ($1, $2)
returning *;

-- name: GetGroup :one
select *
from groups
where id = $1
  and is_deleted = false;

-- name: GetGroupsByUser :many
select g.*
from groups g
         join user_group ug on g.id = ug.group_id
where ug.user_id = $1
  and g.is_deleted = false
  and ug.is_deleted = false;

-- name: GetGroupByInviteToken :one
select *
from groups
where invite_token = $1
  and is_deleted = false;

-- name: DeleteGroup :exec
update groups
set is_deleted = true
where id = $1;

-- name: AddUserToGroup :exec
insert into user_group (user_id, group_id)
values ($1, $2);

-- name: CreateExpense :one
insert into expenses (group_id, type, name, description, amount, currency_id, expense_at, created_at)
values ($1, $2, $3, $4, $5, $6, $7, $8)
returning *;

-- name: CreateExpensePayer :one
insert into expense_payers (expense_id, user_id, amount)
values ($1, $2, $3)
returning *;

-- name: CreateExpenseShare :one
insert into expense_shares (expense_id, user_id, amount)
values ($1, $2, $3)
returning *;

-- name: GetExpense :one
select *
from expenses
where id = $1
  and is_deleted = false;

-- name: GetExpenses :many
select *
from expenses
where group_id = $1
  and is_deleted = false
order by expense_at desc, created_at desc;

-- name: GetExpensesPayers :many
SELECT *
FROM expense_payers
WHERE expense_id = ANY ($1:: bigint[]);

-- name: GetExpensesShares :many
SELECT *
FROM expense_shares
WHERE expense_id = ANY ($1:: bigint[]);

-- name: DeleteExpense :exec
update expenses
set is_deleted = true
where id = $1;

-- name: GetGroupCurrencies :many
SELECT c.id,
       c.code,
       c.name,
       c.symbol,
       COUNT(e.id) as usage_count
FROM currencies c
         JOIN expenses e ON c.id = e.currency_id
WHERE e.group_id = $1
  AND e.is_deleted = false
  AND c.is_deleted = false
GROUP BY c.id
ORDER BY usage_count DESC, c.name ASC;

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