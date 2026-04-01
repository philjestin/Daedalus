package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/philjestin/daedalus/internal/model"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Event represents a real-time event to broadcast.
// This is an alias for model.BroadcastEvent for backwards compatibility.
type Event = model.BroadcastEvent

// Client represents a connected WebSocket client.
type Client struct {
	conn *websocket.Conn
	send chan Event
}

// Hub manages WebSocket connections and broadcasts.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan Event
	register   chan *Client
	unregister chan *Client
	stop       chan struct{}
	stopped    bool
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan struct{}),
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case <-h.stop:
			// Gracefully close all client connections
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				client.conn.Close(websocket.StatusGoingAway, "server shutting down") //nolint:errcheck // best-effort shutdown
				delete(h.clients, client)
			}
			h.stopped = true
			h.mu.Unlock()
			slog.Info("websocket hub stopped")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			slog.Info("websocket client connected", "total_clients", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			slog.Info("websocket client disconnected", "total_clients", len(h.clients))

		case event := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- event:
				default:
					// Client buffer full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends an event to all connected clients.
func (h *Hub) Broadcast(event Event) {
	select {
	case h.broadcast <- event:
	default:
		slog.Warn("broadcast channel full, dropping event", "type", event.Type)
	}
}

// HandleWebSocket handles WebSocket upgrade and connection.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		slog.Error("websocket accept error", "error", err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan Event, 256),
	}

	h.register <- client

	// Start write and read pumps
	go h.writePump(client)
	h.readPump(client)
}

// writePump sends events to the WebSocket connection.
func (h *Hub) writePump(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck // best-effort cleanup
	}()

	for {
		select {
		case event, ok := <-client.send:
			if !ok {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := wsjson.Write(ctx, client.conn, event)
			cancel()

			if err != nil {
				slog.Error("websocket write error", "error", err)
				return
			}

		case <-ticker.C:
			// Send ping
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := client.conn.Ping(ctx)
			cancel()

			if err != nil {
				return
			}
		}
	}
}

// readPump reads messages from the WebSocket connection.
func (h *Hub) readPump(client *Client) {
	defer func() {
		h.unregister <- client
		client.conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck // best-effort cleanup
	}()

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		_, message, err := client.conn.Read(ctx)
		cancel()

		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				slog.Error("websocket read error", "error", err)
			}
			return
		}

		// Handle incoming messages (e.g., subscription requests)
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			slog.Debug("received websocket message", "message", msg)
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Stop gracefully shuts down the hub, closing all client connections.
func (h *Hub) Stop() {
	h.mu.RLock()
	if h.stopped {
		h.mu.RUnlock()
		return
	}
	h.mu.RUnlock()

	close(h.stop)
}

