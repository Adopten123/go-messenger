package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Adopten123/go-messenger/internal/handler"
	"github.com/Adopten123/go-messenger/internal/repo/pgdb"
	"github.com/Adopten123/go-messenger/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

const (
	secret_token = "test_secret"
)

func TestRegisterUser(t *testing.T) {
	pool := SetupTestDB(t)
	defer pool.Close()

	repo := pgdb.New(pool)

	userService := service.NewUserService(repo, secret_token)
	userHandler := handler.NewUserHandler(userService, secret_token, nil, nil)

	r := chi.NewRouter()
	r.Post("/register", userHandler.Register)

	reqBody := map[string]string{
		"username": "test_user",
		"email":    "test@example.com",
		"password": "password123",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var response map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.NotEmpty(t, response["id"], "ID пользователя не должен быть пустым")
	assert.Equal(t, "test_user", response["username"])
}
