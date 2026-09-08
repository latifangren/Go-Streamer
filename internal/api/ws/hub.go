package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// EventMessage merepresentasikan payload pesan WebSocket broadcast JSON ke client.
type EventMessage struct {
	Event     string      `json:"event"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// ClientAction merepresentasikan pesan aksi dari client (misal: ping, start_stream, stop_stream).
type ClientAction struct {
	Action     string          `json:"action"`
	SlotID     int64           `json:"slot_id,omitempty"`
	SlotNumber int             `json:"slot_number,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
}

// ActionHandler mendefinisikan callback untuk memproses aksi yang dikirim oleh client WebSocket.
type ActionHandler func(client *Client, action *ClientAction)

// Client merepresentasikan koneksi aktif WebSocket satu client.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// Hub mengelola koneksi client WebSocket dan menyebarkan pesan broadcast ke seluruh subscriber.
type Hub struct {
	clients       map[*Client]bool
	broadcastChan chan []byte
	register      chan *Client
	unregister    chan *Client
	actionHandler ActionHandler
	mu            sync.RWMutex
}

// NewHub menginisialisasi Hub baru dan menjalankan loop pengelola event di background.
func NewHub(actionHandler ActionHandler) *Hub {
	h := &Hub{
		clients:       make(map[*Client]bool),
		broadcastChan: make(chan []byte, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		actionHandler: actionHandler,
	}
	go h.run()
	return h
}

// SetActionHandler mengatur handler callback untuk memproses request action dari client.
func (h *Hub) SetActionHandler(handler ActionHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.actionHandler = handler
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcastChan:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast menyebarkan data event ke semua client WebSocket yang sedang terhubung.
func (h *Hub) Broadcast(event string, data interface{}) {
	msg := EventMessage{
		Event:     event,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] Gagal marshal event %s: %v", event, err)
		return
	}

	select {
	case h.broadcastChan <- payload:
	default:
		// Drop message jika antrean penuh untuk menjaga stabilitas memori
	}
}

// ServeWS menangani HTTP upgrade ke protokol WebSocket.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

// ClientCount mengembalikan jumlah client yang sedang terhubung ke hub.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(65536)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, msgBytes, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var action ClientAction
		if err := json.Unmarshal(msgBytes, &action); err == nil {
			if action.Action == "ping" {
				pongMsg, _ := json.Marshal(EventMessage{
					Event:     "pong",
					Data:      "pong",
					Timestamp: time.Now().UnixMilli(),
				})
				select {
				case c.send <- pongMsg:
				default:
				}
			} else {
				c.hub.mu.RLock()
				handler := c.hub.actionHandler
				c.hub.mu.RUnlock()
				if handler != nil {
					handler(c, &action)
				}
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Kirim pesan yang tersisa di queue dalam batch
			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
