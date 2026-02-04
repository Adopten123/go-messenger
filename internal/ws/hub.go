package ws

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Adopten123/go-messenger/internal/repo/pgdb"
	"github.com/jackc/pgx/v5/pgtype"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Hub struct {
	// clients: map [UserID] -> Client.
	// User has 1 connection.
	// sync.RWMutex - needed for reading/writing to the map from different goroutines.
	clients map[string]*Client
	mu      sync.RWMutex

	// Channels for reg/unreg
	register   chan *Client
	unregister chan *Client

	// Channel for incoming mess
	broadcast chan Message

	// DB con
	repo *pgdb.Queries
}

func NewHub(repo *pgdb.Queries) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message),
		repo:       repo,
	}
}

// Run - starting Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = client
			h.mu.Unlock()
			slog.Info("client registered", "user_id", client.UserID)
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)
				client.Conn.Close(websocket.StatusNormalClosure, "bye")
			}
			h.mu.Unlock()
			slog.Info("client unregistered", "user_id", client.UserID)

		case msg := <-h.broadcast:
			h.handleMessage(msg)
		}
	}
}

func (h *Hub) handleMessage(msg Message) {
	ctx := context.Background()

	var chatUUID, senderUUID pgtype.UUID
	chatUUID.Scan(msg.ChatID)
	senderUUID.Scan(msg.SenderID)

	savedMsg, err := h.repo.CreateMessage(ctx, pgdb.CreateMessageParams{
		ChatID:   chatUUID,
		SenderID: senderUUID,
		Content:  msg.Content,
	})

	if err != nil {
		slog.Error("failed to save message to db", "error", err)
		return
	}

	memberIDs, err := h.repo.GetChatMembers(ctx, chatUUID)
	if err != nil {
		slog.Error("failed to get chat members", "error", err)
		return
	}

	responseMsg := map[string]interface{}{
		"id":         savedMsg.ID.String(),
		"chat_id":    savedMsg.ChatID.String(),
		"sender_id":  savedMsg.SenderID.String(),
		"content":    savedMsg.Content,
		"created_at": savedMsg.CreatedAt.Time,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, memberIDUUID := range memberIDs {
		memberIDs := memberIDUUID.String()

		if client, ok := h.clients[memberIDs]; ok {
			go func(c *Client) {
				err := wsjson.Write(ctx, c.Conn, responseMsg)
				if err != nil {
					slog.Warn("failed to write message to client", "user_id", c.UserID, "error", err)
				}
			}(client)
		}
	}
}
