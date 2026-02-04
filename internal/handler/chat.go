package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Adopten123/go-messenger/internal/service"
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
