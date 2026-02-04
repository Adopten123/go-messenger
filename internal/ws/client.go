package ws

import (
	"context"
	"log/slog"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type EventType string

const (
	EventNewMessage EventType = "new_message"
	EventMarkRead   EventType = "mark_read"
	Event
)

type IncomingMessage struct {
	Type    EventType `json:"type"`
	ChatID  string    `json:"chat_id"`
	Content string    `json:"content,omitempty"`
}

type OutgoingMessage struct {
	Type      EventType `json:"type"`
	ID        string    `json:"id,omitempty"`
	ChatID    string    `json:"chat_id"`
	Content   string    `json:"content,omitempty"`
	SenderID  string    `json:"sender_id,omitempty"`
	CreatedAt string    `json:"created_at,omitempty"`
	IsRead    bool      `json:"is_read,omitempty"`
}

type Client struct {
	UserID string
	Conn   *websocket.Conn
	Hub    *Hub
}

// ReadPump - listens for messages from client
func (c *Client) ReadPump(ctx context.Context) {
	// Must remove the client from the Hub, if connection is broken
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		var msg IncomingMessage

		err := wsjson.Read(ctx, c.Conn, &msg)
		if err != nil {
			slog.Info("websocket closed", "user_id", c.UserID, "reason", err)
			break
		}

		c.Hub.broadcast <- &HubMessage{
			Client: c,
			Msg:    msg,
		} // Sending message to Hub
	}
}
