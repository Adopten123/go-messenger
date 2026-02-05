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
func TestRegister_FailCases(t *testing.T) {
	pool := SetupTestDB(t)
	defer pool.Close()

	repo := pgdb.New(pool)
	userService := service.NewUserService(repo, secret_token)
	userHandler := handler.NewUserHandler(userService, secret_token, nil, nil)

	r := chi.NewRouter()
	r.Post("/register", userHandler.Register)

	existingUser := map[string]string{
		"username": "existing",
		"email":    "busy@example.com",
		"password": "password123",
	}
	body, _ := json.Marshal(existingUser)
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	tests := []struct {
		name        string
		username    string
		email       string
		password    string
		expectedOld int
		expectedMsg string
	}{
		{
			name:        "Empty Password",
			username:    "user1",
			email:       "user1@example.com",
			password:    "",
			expectedOld: http.StatusBadRequest,
			expectedMsg: "password is required",
		},
		{
			name:        "Empty Email",
			username:    "user2",
			email:       "",
			password:    "pass123",
			expectedOld: http.StatusBadRequest,
			expectedMsg: "email is required",
		},
		{
			name:        "Duplicate Email",
			username:    "new_user",
			email:       "busy@example.com",
			password:    "pass123",
			expectedOld: http.StatusConflict,
			expectedMsg: "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := map[string]string{
				"username": tt.username,
				"email":    tt.email,
				"password": tt.password,
			}
			bodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			assert.GreaterOrEqual(t, rr.Code, 400, "Должна быть ошибка")

			if tt.expectedMsg != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedMsg)
			}
		})
	}
}
