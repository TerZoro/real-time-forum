package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"real-time-forum/models"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID string
	nick   string
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var env models.WSMessage
		if err := json.Unmarshal(raw, &env); err != nil {
			continue
		}

		switch env.Type {
		case "private_message":
			c.hub.handlePrivateMessage(c, env.Payload)
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[string]*Client // userID → client
	register   chan *Client
	unregister chan *Client
}

var GlobalHub = &Hub{
	clients:    make(map[string]*Client),
	register:   make(chan *Client, 64),
	unregister: make(chan *Client, 64),
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c.userID] = c
			h.mu.Unlock()

			// send current online list to the new client
			h.sendOnlineList(c)
			// send to everyone else
			h.broadcastPresence(c.userID, c.nick, true, c)

		case c := <-h.unregister:
			h.mu.Lock()
			if existing, ok := h.clients[c.userID]; ok && existing == c {
				delete(h.clients, c.userID)
				close(c.send)
			}
			h.mu.Unlock()
			h.broadcastPresence(c.userID, c.nick, false, nil)
		}
	}
}

func (h *Hub) sendOnlineList(target *Client) {
	h.mu.RLock()
	statuses := make([]models.UserStatus, 0, len(h.clients))
	for _, c := range h.clients {
		statuses = append(statuses, models.UserStatus{
			UserID:   c.userID,
			Nickname: c.nick,
			Online:   true,
		})
	}
	h.mu.RUnlock()

	msg := h.marshal(models.WSMessage{Type: "online_users", Payload: statuses})
	select {
	case target.send <- msg:
	default:
	}
}

func (h *Hub) broadcastPresence(userID, nick string, online bool, skip *Client) {
	msg := h.marshal(models.WSMessage{
		Type: "presence",
		Payload: models.UserStatus{
			UserID:   userID,
			Nickname: nick,
			Online:   online,
		},
	})

	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		if c == skip {
			continue
		}
		select {
		case c.send <- msg:
		default:
		}
	}
}

func (h *Hub) handlePrivateMessage(sender *Client, payload interface{}) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}

	var pm struct {
		ReceiverID string `json:"receiver_id"`
		Content    string `json:"content"`
	}
	if err := json.Unmarshal(raw, &pm); err != nil || pm.ReceiverID == "" || pm.Content == "" {
		return
	}

	msg, err := SaveMessage(sender.userID, pm.ReceiverID, pm.Content)
	if err != nil {
		log.Printf("SaveMessage error: %v", err)
		return
	}

	envelope := h.marshal(models.WSMessage{Type: "private_message", Payload: msg})

	h.mu.RLock()
	receiver, online := h.clients[pm.ReceiverID]
	h.mu.RUnlock()

	if online {
		select {
		case receiver.send <- envelope:
		default:
		}
	}

	// Echo back to sender
	select {
	case sender.send <- envelope:
	default:
	}
}

func (h *Hub) marshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

// HTTP handler
func ServeWS(w http.ResponseWriter, r *http.Request) {
	user, ok := GetSessionUser(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
	}

	client := &Client{
		hub:    GlobalHub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: user.ID,
		nick:   user.Nickname,
	}

	GlobalHub.register <- client
	go client.writePump()
	go client.readPump()
}
