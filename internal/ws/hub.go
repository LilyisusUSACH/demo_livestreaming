package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"test_zapping/internal/auth"
	"test_zapping/internal/cache"
	"test_zapping/internal/middleware"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Message struct {
	Type      string `json:"type"` // "chat", "reaction", or "system"
	ChannelID string `json:"channel_id,omitempty"`
	Content   string `json:"content,omitempty"`
	Emoji     string `json:"emoji,omitempty"`
	Sender    string `json:"sender"`
	SenderID  string `json:"sender_id"`
	Time      string `json:"time,omitempty"`
	IsHistory bool   `json:"is_history,omitempty"`
}

type Client struct {
	Hub       *Hub
	Conn      *websocket.Conn
	Send      chan []byte
	UserName  string
	UserID    string
	ChannelID string
}

type ChannelRoom struct {
	clients map[*Client]bool
	mu      sync.RWMutex
}

type Hub struct {
	rooms      map[string]*ChannelRoom
	register   chan *Client
	unregister chan *Client
	cacheSvc   *cache.CacheService
	mu         sync.RWMutex
}

func NewHub(cacheSvc *cache.CacheService) *Hub {
	return &Hub{
		rooms:      make(map[string]*ChannelRoom),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		cacheSvc:   cacheSvc,
	}
}

func getChannelDisplayName(channelID string) string {
	switch channelID {
	case "kuspid-cinema":
		return "Kuspid Cinema 4K 🎬"
	case "kuspid-tech":
		return "Kuspid Tech & Gaming 🎮"
	default:
		return "Kuspid Sports HD ⚽"
	}
}

func (h *Hub) getOrCreateRoom(channelID string) *ChannelRoom {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[channelID]
	if !ok {
		room = &ChannelRoom{
			clients: make(map[*Client]bool),
		}
		h.rooms[channelID] = room
	}
	return room
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			room := h.getOrCreateRoom(client.ChannelID)
			room.mu.Lock()
			room.clients[client] = true
			room.mu.Unlock()

			// Send channel System Welcome message
			welcomeMsg := Message{
				Type:      "system",
				ChannelID: client.ChannelID,
				Content:   fmt.Sprintf("👋 ¡Bienvenid@ a la sala en vivo de %s!", getChannelDisplayName(client.ChannelID)),
				Sender:    "SISTEMA KUSPID",
			}
			welcomeBytes, _ := json.Marshal(welcomeMsg)
			client.Send <- welcomeBytes

			// Send recent channel chat history stored in Redis on connect
			if h.cacheSvc != nil {
				go func(c *Client) {
					history, err := h.cacheSvc.GetChatHistory(context.Background(), c.ChannelID)
					if err == nil {
						for _, msgBytes := range history {
							var m Message
							if err := json.Unmarshal(msgBytes, &m); err == nil {
								m.IsHistory = true
								out, _ := json.Marshal(m)
								c.Send <- out
							} else {
								c.Send <- msgBytes
							}
						}
					}
				}(client)
			}

		case client := <-h.unregister:
			h.mu.RLock()
			room, ok := h.rooms[client.ChannelID]
			h.mu.RUnlock()

			if ok {
				room.mu.Lock()
				if _, exists := room.clients[client]; exists {
					delete(room.clients, client)
					close(client.Send)
				}
				room.mu.Unlock()
			}
		}
	}
}

func (h *Hub) BroadcastToChannel(channelID string, message []byte) {
	h.mu.RLock()
	room, ok := h.rooms[channelID]
	h.mu.RUnlock()

	if !ok || room == nil {
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	for client := range room.clients {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			delete(room.clients, client)
		}
	}
}

func (h *Hub) ActiveClientsCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, room := range h.rooms {
		room.mu.RLock()
		total += len(room.clients)
		room.mu.RUnlock()
	}
	return total
}

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}

	channelID := r.URL.Query().Get("channel")
	if channelID == "" {
		channelID = "kuspid-sports"
	}

	userName := "Anon"
	userID := "anon"
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if ok && claims != nil {
		userName = claims.Name
		userID = claims.UserID
	}

	client := &Client{
		Hub:       hub,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		UserName:  userName,
		UserID:    userID,
		ChannelID: channelID,
	}

	client.Hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err == nil {
			msg.Sender = c.UserName
			msg.SenderID = c.UserID
			msg.ChannelID = c.ChannelID

			out, _ := json.Marshal(msg)

			// Store message in Redis chat history
			if c.Hub.cacheSvc != nil {
				_ = c.Hub.cacheSvc.SaveChatMessage(context.Background(), c.ChannelID, out)
				_ = c.Hub.cacheSvc.PublishChatMessage(context.Background(), c.ChannelID, out)
			}

			// Broadcast to channel room clients
			c.Hub.BroadcastToChannel(c.ChannelID, out)
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.Conn.Close()
	}()

	for message := range c.Send {
		w, err := c.Conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}
		w.Write(message)

		if err := w.Close(); err != nil {
			return
		}
	}
}
