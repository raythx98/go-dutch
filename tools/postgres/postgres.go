package postgres

import "github.com/jackc/pgx/v5/pgxpool"

type IPostgres interface {
	Pool() *pgxpool.Pool
}
