package db

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
)

// Needs a reachable PostgreSQL; read-only (it only opens a pool), but kept to
// the same _test database as the other integration tests.
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if name := strings.TrimPrefix(u.Path, "/"); !strings.HasSuffix(name, "_test") {
		t.Fatalf("refusing to use database %q: its name must end in _test", name)
	}

	return dsn
}

func TestConnectSizesThePool(t *testing.T) {
	dsn := testDSN(t)
	for _, tc := range []struct {
		in   int32
		want int32
	}{{3, 3}, {0, DefaultMaxConns}, {-1, DefaultMaxConns}} {
		pool, err := Connect(context.Background(), dsn, tc.in)
		if err != nil {
			t.Fatal(err)
		}
		if got := pool.Config().MaxConns; got != tc.want {
			t.Errorf("Connect(max=%d): MaxConns = %d, want %d", tc.in, got, tc.want)
		}
		pool.Close()
	}
}

func TestQuerierPrefersTheTransactionInContext(t *testing.T) {
	dsn := testDSN(t)
	pool, err := Connect(context.Background(), dsn, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	ctx := context.Background()
	if q := Querier(ctx, pool); q != Queryable(pool) {
		t.Fatal("without a transaction the fallback must be used")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	inTx := WithTx(ctx, tx)
	if got, ok := TxFrom(inTx); !ok || got != tx {
		t.Fatal("TxFrom must return the transaction stored by WithTx")
	}
	if q := Querier(inTx, pool); q != Queryable(tx) {
		t.Fatal("with a transaction in the context it must be used instead of the pool")
	}
	if _, ok := TxFrom(ctx); ok {
		t.Fatal("a plain context carries no transaction")
	}
}
