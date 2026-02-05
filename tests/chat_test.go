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

func TestChatFlow(t *testing.T) {
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

	tokenA := RegisterAndLogin(t, userHandler, "Alice", "alice@example.com")

	RegisterAndLogin(t, userHandler, "Bob", "bob@example.com")

	var chatID string

	t.Run("Create Chat", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "Bob",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/chats", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenA)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusCreated, rr.Code, "Expected status 201, got %d. Body: %s", rr.Code, rr.Body.String())

		var resp map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)

		if _, ok := resp["chat_id"]; !ok {
			t.Fatalf("Response JSON does not contain 'id'. Got: %v", resp)
		}

		chatID = resp["chat_id"].(string)
	})

	t.Run("Get Messages Empty", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/chats/"+chatID+"/messages", nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)

		var messages []interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &messages)
		require.NoError(t, err)

		assert.Len(t, messages, 0)
	})
}
