package common

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetDBConnectionString() (string, error) {
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	db := os.Getenv("POSTGRES_DB")
	host := os.Getenv("POSTGRES_HOST")

	if host == "" {
		host = "localhost"
	}

	if user == "" || password == "" || db == "" {
		return "", fmt.Errorf("missing required env vars: POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB must all be set")
	}

	return fmt.Sprintf("postgres://%s:%s@%s:5432/%s", user, password, host, db), nil
}

func ConnectToDatabase(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool : %w", err)
	}

	pingctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil

}
