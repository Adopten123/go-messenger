package ws

import (
	"context"
	"log/slog"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Client struct {
	UserID string
	Conn   *websocket.Conn
	Hub    *Hub
}

type Message struct {
	ChatID   string `json:"chat_id"`
	Content  string `json:"content"`
	SenderID string `json:"sender_id"`
}

// ReadPump - listens for messages from client
func (c *Client) ReadPump(ctx context.Context) {
	// Must remove the client from the Hub, if connection is broken
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		var msg Message

		err := wsjson.Read(ctx, c.Conn, &msg)
		if err != nil {
			slog.Info("websocket closed", "user_id", c.UserID, "reason", err)
			break
		}

		msg.SenderID = c.UserID
		c.Hub.broadcast <- msg // Sending message to Hub
	}
}
