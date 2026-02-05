package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Adopten123/go-messenger/internal/repo/pgdb"
	"github.com/Adopten123/go-messenger/internal/service"
	"github.com/go-chi/chi/v5"
)

type ChatHandler struct {
	service *service.ChatService
}

func NewChatHandler(service *service.ChatService) *ChatHandler {
	return &ChatHandler{service: service}
}

type CreateChatRequest struct {
	Name    string   `json:"name"`
	UserIDs []string `json:"user_ids"`
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	// 1. Decoding
	var req CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// 2. Getting ID
	creatorID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "user not found in context", http.StatusUnauthorized)
		return
	}

	// 3. Calling server
	chat, err := h.service.CreateChat(r.Context(), req.Name, creatorID, req.UserIDs)
	if err != nil {
		http.Error(w, "failed to create chat", http.StatusInternalServerError)
		return
	}

	// 4. Sending response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]string{
		"chat_id": chat.ID.String(),
		"name":    chat.Name.String,
	})
}

func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	// 1. Getting chat_id from URL
	chatID := chi.URLParam(r, "chat_id")
	if chatID == "" {
		http.Error(w, "chat_id is required", http.StatusBadRequest)
		return
	}
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "user not found in context", http.StatusUnauthorized)
		return
	}

	// 2. Parse query params
	limit := 50
	offset := 0

	queryLimit := r.URL.Query().Get("limit")
	if queryLimit != "" {
		if l, err := strconv.Atoi(queryLimit); err == nil && l > 0 {
			limit = l
		}
	}

	queryOffset := r.URL.Query().Get("offset")
	if queryOffset != "" {
		if o, err := strconv.Atoi(queryOffset); err == nil && o > 0 {
			offset = o
		}
	}

	// 3. Calling service
	messages, err := h.service.GetMessages(r.Context(), chatID, userID, limit, offset)
	if err != nil {
		if err.Error() == "access denied" {
			http.Error(w, "access denied", http.StatusForbidden)
			return
		}
		http.Error(w, "failed to fetch messages", http.StatusInternalServerError)
		return
	}

	if messages == nil {
		messages = []pgdb.ListMessagesRow{}
	}

	// 4. Response JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
