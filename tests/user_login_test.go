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
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	pool := SetupTestDB(t)
	defer pool.Close()

	repo := pgdb.New(pool)
	userService := service.NewUserService(repo, secret_token)
	userHandler := handler.NewUserHandler(userService, secret_token, nil, nil)

	r := chi.NewRouter()
	r.Post("/register", userHandler.Register)
	r.Post("/login", userHandler.Login)

	validUser := map[string]string{
		"username": "tester",
		"email":    "tester@example.com",
		"password": "strongpassword",
	}
	body, _ := json.Marshal(validUser)
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/register", bytes.NewBuffer(body)))

	tests := []struct {
		name         string
		email        string
		password     string
		expectedCode int
		checkToken   bool
	}{
		{
			name:         "Success Login",
			email:        "tester@example.com",
			password:     "strongpassword",
			expectedCode: http.StatusOK,
			checkToken:   true,
		},
		{
			name:         "Wrong Password",
			email:        "tester@example.com",
			password:     "wrong123",
			expectedCode: http.StatusUnauthorized,
			checkToken:   false,
		},
		{
			name:         "User Not Found",
			email:        "ghost@example.com",
			password:     "anyPass",
			expectedCode: http.StatusUnauthorized,
			checkToken:   false,
		},
		{
			name:         "Empty Email",
			email:        "",
			password:     "123",
			expectedCode: http.StatusBadRequest,
			checkToken:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := map[string]string{
				"email":    tt.email,
				"password": tt.password,
			}
			bodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tt.expectedCode, rr.Code)

			if tt.checkToken {
				var resp map[string]string
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.NotEmpty(t, resp["token"], "Token should not be empty on success")
			}
		})
	}
}
