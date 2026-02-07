package websocket

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

// Hub manages WebSocket connections for real-time delivery tracking
type Hub struct {
	// Map of order_id -> connected clients
	clients    map[string]map[*websocket.Conn]bool
	broadcast  chan *LocationUpdate
	register   chan *ClientRegistration
	unregister chan *ClientRegistration
	mu         sync.RWMutex
}

// LocationUpdate represents a delivery location update
type LocationUpdate struct {
	OrderID   string  `json:"order_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Status    string  `json:"status"`
	ETAMins   int     `json:"eta_mins"`
	Timestamp string  `json:"timestamp"`
}

// ClientRegistration represents a client connection registration
type ClientRegistration struct {
	OrderID string
	Conn    *websocket.Conn
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*websocket.Conn]bool),
		broadcast:  make(chan *LocationUpdate, 256),
		register:   make(chan *ClientRegistration),
		unregister: make(chan *ClientRegistration),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case reg := <-h.register:
			h.mu.Lock()
			if h.clients[reg.OrderID] == nil {
				h.clients[reg.OrderID] = make(map[*websocket.Conn]bool)
			}
			h.clients[reg.OrderID][reg.Conn] = true
			h.mu.Unlock()

		case reg := <-h.unregister:
			h.mu.Lock()
			if conns, ok := h.clients[reg.OrderID]; ok {
				delete(conns, reg.Conn)
				if len(conns) == 0 {
					delete(h.clients, reg.OrderID)
				}
			}
			h.mu.Unlock()

		case update := <-h.broadcast:
			h.mu.RLock()
			if conns, ok := h.clients[update.OrderID]; ok {
				for conn := range conns {
					if err := conn.WriteJSON(update); err != nil {
						// Connection error, will be cleaned up on next read
						continue
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a client to the hub
func (h *Hub) Register(orderID string, conn *websocket.Conn) {
	h.register <- &ClientRegistration{OrderID: orderID, Conn: conn}
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(orderID string, conn *websocket.Conn) {
	h.unregister <- &ClientRegistration{OrderID: orderID, Conn: conn}
}

// BroadcastLocation sends a location update to all clients tracking an order
func (h *Hub) BroadcastLocation(update *LocationUpdate) {
	h.broadcast <- update
}

// GetActiveConnections returns the number of active connections for an order
func (h *Hub) GetActiveConnections(orderID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if conns, ok := h.clients[orderID]; ok {
		return len(conns)
	}
	return 0
}

// GetTotalConnections returns the total number of active WebSocket connections
func (h *Hub) GetTotalConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, conns := range h.clients {
		total += len(conns)
	}
	return total
}
