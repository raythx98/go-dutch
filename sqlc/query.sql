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

-- name: UpsertUserCurrencyPreference :one
INSERT INTO user_currency_preferences (user_id, currency_id, use_count)
VALUES ($1, $2, 1)
ON CONFLICT
    (user_id, currency_id)
    DO UPDATE
    SET use_count = user_currency_preferences.use_count + 1
RETURNING *;

-- name: GetRankedCurrencies :many
SELECT c.id,
       c.code,
       c.name,
       c.symbol,
       COALESCE(ucp.use_count, 0) as preference_count
FROM currencies c
         LEFT JOIN user_currency_preferences ucp
                   ON c.id = ucp.currency_id AND ucp.user_id = $1
WHERE c.is_deleted = false
ORDER BY COALESCE(ucp.use_count, 0) DESC, c.name ASC;

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
insert into expenses (group_id, type, name, description, amount, currency_id, expense_at)
values ($1, $2, $3, $4, $5, $6, $7)
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
order by expense_at desc;

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