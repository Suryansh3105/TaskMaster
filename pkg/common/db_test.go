package common

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestConnectToDatabase(t *testing.T) {
	ctx := context.Background()

	dbURL := fmt.Sprintf(
		"postgres://%s:%s@localhost:5432/%s",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)

	pool, err := ConnectToDatabase(ctx, dbURL)

	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}

	defer pool.Close()
}
