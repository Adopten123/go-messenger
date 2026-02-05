package tests

import (
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

func TestGetMe(t *testing.T) {
	pool := SetupTestDB(t)
	defer pool.Close()
	repo := pgdb.New(pool)

	userService := service.NewUserService(repo, secret_token)
	userHandler := handler.NewUserHandler(userService, secret_token, nil, nil)

	r := chi.NewRouter()
	r.Use(userHandler.AuthMiddleware)
	r.Get("/users/me", userHandler.GetMe)

	email := "me@test.com"
	username := "Myself"
	token := RegisterAndLogin(t, userHandler, username, email)

	req := httptest.NewRequest("GET", "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)

	require.NoError(t, err, "Failed to parse JSON response: %s", w.Body.String())

	assert.Equal(t, username, resp["username"])
	assert.Equal(t, email, resp["email"])
	assert.NotEmpty(t, resp["id"])
}
