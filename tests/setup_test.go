package tests

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const TestDBString = "postgres://postgres:postgres_password@localhost:5434/messenger_test_db?sslmode=disable"

func SetupTestDB(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, TestDBString)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}

	_, err = pool.Exec(ctx, "TRUNCATE TABLE users, chats, chat_members, messages RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	return pool
}
