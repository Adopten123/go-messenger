package ws

import (
	"net/http"

	"github.com/Adopten123/go-messenger/internal/handler"
	"nhooyr.io/websocket"
)

type WSHandler struct {
	hub *Hub
}

func NewWSHandler(hub *Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

func (h *WSHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	// 1. Getting user from ctx
	userID, ok := r.Context().Value(handler.UserIDKey).(string)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. HTTP to WebSocket
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // TODO: del in prod
	})
	if err != nil {
		return
	}

	// 3. Making client
	client := &Client{
		UserID: userID,
		Conn:   c,
		Hub:    h.hub,
	}

	// 4. Reg in hub
	h.hub.register <- client

	// 5. Run message reading
	client.ReadPump(r.Context()) // blocks this go while the connection is open

}
