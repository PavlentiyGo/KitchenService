package postgres

import (
	core_logger "avitoKitchen/internal/core/logger"
	"avitoKitchen/internal/repository"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type txCtxKey struct{}
type TxManager struct {
	pool           *pgxpool.Pool
	requestTimeout time.Duration
}

func NewTxManager(pool *pgxpool.Pool, timeout time.Duration) *TxManager {
	return &TxManager{
		pool:           pool,
		requestTimeout: timeout,
	}
}

func (m *TxManager) WithinTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			logger := core_logger.FromContext(ctx)
			logger.Error("failed to rollback transaction", zap.Error(rollbackErr))
		}
	}()
	txCtx := context.WithValue(ctx, txCtxKey{}, tx)
	err = fn(txCtx)
	if err != nil {
		return fmt.Errorf("error during transaction: %w", err)
	}
	err = tx.Commit(txCtx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	return nil
}
func (m *TxManager) GetExecutor(ctx context.Context) repository.Executor {
	if tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx); ok {
		return tx
	}
	return m.pool
}
func (m *TxManager) GetTimeout() time.Duration {
	return m.requestTimeout
}
