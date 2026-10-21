package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultMaxConns is the pool size when DB_MAX_CONNS is not set. pgxpool's own
// default is max(4, CPUs), which is too small once a request can fan out into
// several queries while BearerAuth holds a connection per request.
const DefaultMaxConns = 10

// Connect opens the pool. maxConns <= 0 means DefaultMaxConns.
func Connect(ctx context.Context, url string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	// Disable server-side prepared statements to work with pgBouncer in transaction mode
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	if maxConns <= 0 {
		maxConns = DefaultMaxConns
	}
	cfg.MaxConns = maxConns

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

// Queryable is the part of *pgxpool.Pool and pgx.Tx that repositories use, so
// the same code runs inside or outside a transaction.
type Queryable interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

// WithTx returns a context that carries tx. Usecases never see pgx: a
// TxManager puts the transaction here and repositories pick it up via Querier.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFrom returns the transaction carried by ctx, if any.
func TxFrom(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)

	return tx, ok
}

// Querier returns the transaction carried by ctx, or fallback (normally the
// pool) when there is none.
func Querier(ctx context.Context, fallback Queryable) Queryable {
	if tx, ok := TxFrom(ctx); ok {
		return tx
	}

	return fallback
}
