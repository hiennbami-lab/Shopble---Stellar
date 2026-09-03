package comsocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Envelope struct {
	Ns   string `json:"namespace"`
	Room string `json:"room,omitempty"`
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type Client struct {
	Conn     *websocket.Conn
	Send     chan Envelope
	Rooms    map[string]bool
	ClientId string
}

type Hub struct {
	mu        sync.RWMutex
	namespace string
	rooms     map[string]map[*Client]bool
}

func NewHub(
	namespace string,
) *Hub {
	return &Hub{
		namespace: namespace,
		rooms:     make(map[string]map[*Client]bool),
	}
}

// join a client into ns/room
func (h *Hub) Join(c *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients, ok := h.rooms[room]
	if !ok {
		clients = make(map[*Client]bool)
	}
	c.Rooms[room] = true
	clients[c] = true
	h.rooms[room] = clients
}

// leave a room
func (h *Hub) Leave(c *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.rooms[room]; ok {
		delete(clients, c)
	}
	delete(c.Rooms, room)
}

// broadcast to a ns/room
func (h *Hub) Broadcast(room string, msg Envelope) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.rooms[room]; ok {
		for c := range clients {
			select {
			case c.Send <- msg:
			default:
				// drop on full channel
			}
		}
	}
}
