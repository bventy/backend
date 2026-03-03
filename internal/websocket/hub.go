package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 // We don't expect large messages from clients (mostly auth tokens or typings)
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allowing all origins for brevity; in prod, check against allowed domains.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub            *Hub
	Conn           *websocket.Conn
	Send           chan []byte
	UserID         string
	ConversationID string
}

// Hub maintains the set of active clients and broadcasts messages to the rooms.
type Hub struct {
	// Registered clients split by ConversationID rooms.
	Rooms map[string]map[*Client]bool

	// Active users mapped to bool to emit presence updates.
	ActiveUsers map[string]bool

	// Locks
	mu sync.RWMutex

	Broadcast  chan MessageEvent
	Register   chan *Client
	Unregister chan *Client
}

// MessageEvent is strictly for PUB/SUB. Clients listen for these formats.
type MessageEvent struct {
	Type           string      `json:"type"` // "new_message", "message_read", "presence_update"
	ConversationID string      `json:"conversation_id,omitempty"`
	Payload        interface{} `json:"payload"`
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:   make(chan MessageEvent),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		Rooms:       make(map[string]map[*Client]bool),
		ActiveUsers: make(map[string]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if _, ok := h.Rooms[client.ConversationID]; !ok {
				h.Rooms[client.ConversationID] = make(map[*Client]bool)
			}
			h.Rooms[client.ConversationID][client] = true
			h.ActiveUsers[client.UserID] = true

			// Broadcast presence join to that room
			presenceMsg := MessageEvent{
				Type:           "presence_update",
				ConversationID: client.ConversationID,
				Payload: map[string]interface{}{
					"user_id": client.UserID,
					"online":  true,
				},
			}
			h.broadcastToRoom(presenceMsg)
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if connections, ok := h.Rooms[client.ConversationID]; ok {
				if _, ok := connections[client]; ok {
					delete(connections, client)
					close(client.Send)

					// If no clients left in room, delete room
					if len(connections) == 0 {
						delete(h.Rooms, client.ConversationID)
					}

					// Simple offline check - if user has no other connections, mark offline
					isOnline := false
					for _, room := range h.Rooms {
						for c := range room {
							if c.UserID == client.UserID {
								isOnline = true
								break
							}
						}
						if isOnline {
							break
						}
					}

					if !isOnline {
						delete(h.ActiveUsers, client.UserID)
					}

					// Broadcast presence leave
					presenceMsg := MessageEvent{
						Type:           "presence_update",
						ConversationID: client.ConversationID,
						Payload: map[string]interface{}{
							"user_id": client.UserID,
							"online":  isOnline,
						},
					}
					h.broadcastToRoom(presenceMsg)
				}
			}
			h.mu.Unlock()

		case message := <-h.Broadcast:
			h.mu.RLock()
			h.broadcastToRoom(message)
			h.mu.RUnlock()
		}
	}
}

// broadcastToRoom assumes lock is held by caller
func (h *Hub) broadcastToRoom(message MessageEvent) {
	connections, ok := h.Rooms[message.ConversationID]
	if !ok {
		return
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v", err)
		return
	}

	for client := range connections {
		select {
		case client.Send <- payload:
		default:
			close(client.Send)
			delete(connections, client)
		}
	}
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { _ = c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		// Based on spec, clients do NOT send database queries via socket.
		// Sockets are strictly downstream, but we could listen for typing indicators here.
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, userID string, conversationID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{
		Hub:            hub,
		Conn:           conn,
		Send:           make(chan []byte, 256),
		UserID:         userID,
		ConversationID: conversationID,
	}

	client.Hub.Register <- client

	// Allow collection of memory referenced by the caller by doing all work in new goroutines.
	go client.writePump()
	go client.readPump()
}
