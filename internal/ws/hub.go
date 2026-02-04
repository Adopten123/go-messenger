package ws

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Adopten123/go-messenger/internal/repo/pgdb"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"nhooyr.io/websocket/wsjson"
)

type HubMessage struct {
	Client *Client
	Msg    IncomingMessage
}

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
	broadcast chan *HubMessage

	// DB con
	repo *pgdb.Queries
	// Redis
	rdb *redis.Client
}

func NewHub(repo *pgdb.Queries, rdb *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *HubMessage),
		repo:       repo,
		rdb:        rdb,
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

			go func(uid string) {
				h.rdb.Set(context.Background(), "user:"+uid+":online", "true", 0)
			}(client.UserID)

			slog.Info("client registered", "user_id", client.UserID)
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)

				go func(uid string) {
					h.rdb.Del(context.Background(), "user:"+uid+":online")
				}(client.UserID)
			}
			h.mu.Unlock()
			slog.Info("client unregistered", "user_id", client.UserID)

		case hubMsg := <-h.broadcast:
			h.routeEvent(hubMsg)
		}
	}
}

func (h *Hub) routeEvent(hm *HubMessage) {
	switch hm.Msg.Type {
	case EventNewMessage:
		h.handleNewMessage(hm)
	case EventMarkRead:
		h.handleMarkRead(hm)
	case EventTyping:
		h.handleTyping(hm)
	default:
		slog.Warn("unknown event type", "type", hm.Msg.Type)
	}
}

// handleNewMessage: Access Check -> Save -> Mailing
func (h *Hub) handleNewMessage(hm *HubMessage) {
	msg := hm.Msg
	client := hm.Client
	ctx := context.Background()

	var chatUUID, senderUUID pgtype.UUID
	if err := chatUUID.Scan(msg.ChatID); err != nil {
		slog.Error("invalid chat uuid", "chat_id", msg.ChatID)
		return
	}
	senderUUID.Scan(client.UserID)

	isMember, err := h.repo.IsChatMember(ctx, pgdb.IsChatMemberParams{
		ChatID: chatUUID,
		UserID: senderUUID,
	})
	if err != nil {
		slog.Error("failed to check membership", "error", err)
		return
	}
	if !isMember {
		slog.Warn("access denied", "user_id", client.UserID, "chat_id", msg.ChatID)
		return
	}

	savedMsg, err := h.repo.CreateMessage(ctx, pgdb.CreateMessageParams{
		ChatID:   chatUUID,
		SenderID: senderUUID,
		Content:  msg.Content,
	})
	if err != nil {
		slog.Error("failed to save message", "error", err)
		return
	}

	response := OutgoingMessage{
		Type:      EventNewMessage,
		ID:        savedMsg.ID.String(),
		ChatID:    savedMsg.ChatID.String(),
		Content:   savedMsg.Content,
		SenderID:  savedMsg.SenderID.String(),
		CreatedAt: savedMsg.CreatedAt.Time.Format(time.RFC3339),
		IsRead:    false,
	}

	h.broadcastToChat(ctx, chatUUID, response)
}

// handleMarkRead - DB Update -> Send Notification
func (h *Hub) handleMarkRead(hm *HubMessage) {
	ctx := context.Background()
	var chatUUID, userUUID pgtype.UUID

	if err := chatUUID.Scan(hm.Msg.ChatID); err != nil {
		return
	}
	userUUID.Scan(hm.Client.UserID)

	err := h.repo.MarkMessagesAsRead(ctx, pgdb.MarkMessagesAsReadParams{
		ChatID:   chatUUID,
		SenderID: userUUID,
	})
	if err != nil {
		slog.Error("failed to mark messages read", "error", err)
		return
	}

	response := OutgoingMessage{
		Type:     EventMarkRead,
		ChatID:   hm.Msg.ChatID,
		SenderID: hm.Client.UserID,
	}

	h.broadcastToChat(ctx, chatUUID, response)
}

// broadcastToChat - find chat members and send message
func (h *Hub) broadcastToChat(ctx context.Context, chatUUID pgtype.UUID, msg OutgoingMessage) {
	memberIDs, err := h.repo.GetChatMembers(ctx, chatUUID)
	if err != nil {
		slog.Error("failed to get chat members", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, memberUUID := range memberIDs {
		memberID := memberUUID.String()

		if client, ok := h.clients[memberID]; ok {
			// async send
			go func(c *Client) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				wsjson.Write(ctx, c.Conn, msg)
			}(client)
		}
	}
}

func (h *Hub) handleTyping(hm *HubMessage) {
	ctx := context.Background()

	var chatUUID, userUUID pgtype.UUID
	if err := chatUUID.Scan(hm.Msg.ChatID); err != nil {
		return
	}
	if err := userUUID.Scan(hm.Client.UserID); err != nil {
		return
	}

	isMember, err := h.repo.IsChatMember(ctx, pgdb.IsChatMemberParams{
		ChatID: chatUUID,
		UserID: userUUID,
	})

	if err != nil || !isMember {
		return
	}

	response := OutgoingMessage{
		Type:     EventTyping,
		ChatID:   hm.Msg.ChatID,
		SenderID: hm.Client.UserID,
	}

	h.broadcastToChat(ctx, chatUUID, response)
}
