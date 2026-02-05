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

func TestChatAccessDenied(t *testing.T) {
	pool := SetupTestDB(t)
	defer pool.Close()
	repo := pgdb.New(pool)

	userService := service.NewUserService(repo, secret_token)
	userHandler := handler.NewUserHandler(userService, secret_token, nil, nil)
	chatService := service.NewChatService(repo, pool)
	chatHandler := handler.NewChatHandler(chatService)

	r := chi.NewRouter()
	r.Post("/users/register", userHandler.Register)
	r.Post("/users/login", userHandler.Login)
	r.Group(func(r chi.Router) {
		r.Use(userHandler.AuthMiddleware)
		r.Post("/chats", chatHandler.CreateChat)
		r.Get("/chats/{chat_id}/messages", chatHandler.GetMessages)
	})

	tokenAlice := RegisterAndLogin(t, userHandler, "Alice", "alice@sec.com")
	RegisterAndLogin(t, userHandler, "Bob", "bob@sec.com")
	tokenEve := RegisterAndLogin(t, userHandler, "Eve", "eve@hacker.com") // Хакер

	createBody, _ := json.Marshal(map[string]string{"username": "Bob"})
	req := httptest.NewRequest("POST", "/chats", bytes.NewBuffer(createBody))
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	chatID := resp["chat_id"].(string)

	t.Run("Eve Tries To Read", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/chats/"+chatID+"/messages", nil)
		req.Header.Set("Authorization", "Bearer "+tokenEve)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code, "Security Breach! Eve read the chat!")
		assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound, http.StatusUnauthorized}, w.Code)
	})
}
