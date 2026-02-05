package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Adopten123/go-messenger/internal/handler"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const (
	secret_token = "test_secret"
	TestDBString = "postgres://postgres:postgres_password@localhost:5434/messenger_test_db?sslmode=disable"
)

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

// RegisterAndLogin - making user and return his token
func RegisterAndLogin(t *testing.T, handler *handler.UserHandler, username, email string) string {
	regBody := map[string]string{
		"username": username,
		"email":    email,
		"password": "password123",
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Register(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	loginBody := map[string]string{
		"email":    email,
		"password": "password123",
	}
	body, _ = json.Marshal(loginBody)
	req = httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	handler.Login(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	return resp["token"]
}
