package coordinator

import (
	"context"
	"testing"

	"github.com/Suryansh3105/taskmaster/pkg/common"
	"github.com/jackc/pgx/v5/pgxpool"
)

func cleanTasksTable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, "TRUNCATE TABLE tasks")
	if err != nil {
		t.Fatalf("failed to truncate tasks table: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "TRUNCATE TABLE tasks")
	})
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	connString, err := common.GetDBConnectionString()
	if err != nil {
		t.Fatalf("config error: %v", err)
	}
	pool, err := common.ConnectToDatabase(context.Background(), connString)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}
