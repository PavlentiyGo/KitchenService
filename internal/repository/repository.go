package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
type TxManager interface {
	WithinTx(ctx context.Context, fn func(txCtx context.Context) error) error
	GetExecutor(ctx context.Context) Executor
	GetTimeout() time.Duration
}
