package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHub_BroadcastAndPing(t *testing.T) {
	var handledAction *ClientAction
	var mu sync.Mutex

	actionHandler := func(client *Client, action *ClientAction) {
		mu.Lock()
		defer mu.Unlock()
		handledAction = action
	}

	hub := NewHub(actionHandler)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 1. Sambungkan client WebSocket pertama
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial conn1 failed: %v", err)
	}
	defer conn1.Close()

	// 2. Sambungkan client WebSocket kedua
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial conn2 failed: %v", err)
	}
	defer conn2.Close()

	// Berikan jeda untuk registrasi di goroutine hub
	time.Sleep(50 * time.Millisecond)
	if count := hub.ClientCount(); count != 2 {
		t.Errorf("expected 2 clients connected, got %d", count)
	}

	// 3. Uji Broadcast pesan JSON
	testPayload := map[string]string{"message": "hello streamers"}
	hub.Broadcast("test_event", testPayload)

	// Verifikasi conn1 menerima pesan
	_ = conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg1 EventMessage
	if err := conn1.ReadJSON(&msg1); err != nil {
		t.Fatalf("conn1 failed to read broadcast: %v", err)
	}
	if msg1.Event != "test_event" {
		t.Errorf("expected event test_event, got %s", msg1.Event)
	}

	// Verifikasi conn2 menerima pesan yang sama
	_ = conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg2 EventMessage
	if err := conn2.ReadJSON(&msg2); err != nil {
		t.Fatalf("conn2 failed to read broadcast: %v", err)
	}
	if msg2.Event != "test_event" {
		t.Errorf("expected event test_event, got %s", msg2.Event)
	}

	// 4. Uji Ping Action dari client (harus dibalas 'pong' event)
	pingAction := ClientAction{Action: "ping"}
	if err := conn1.WriteJSON(pingAction); err != nil {
		t.Fatalf("failed to send ping: %v", err)
	}

	_ = conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	var pongMsg EventMessage
	if err := conn1.ReadJSON(&pongMsg); err != nil {
		t.Fatalf("conn1 failed to read pong: %v", err)
	}
	if pongMsg.Event != "pong" {
		t.Errorf("expected pong event, got %s", pongMsg.Event)
	}

	// 5. Uji Custom Action dari client (start_stream)
	startAction := ClientAction{
		Action:     "start_stream",
		SlotID:     1,
		SlotNumber: 1,
	}
	if err := conn1.WriteJSON(startAction); err != nil {
		t.Fatalf("failed to send custom action: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	if handledAction == nil || handledAction.Action != "start_stream" {
		t.Errorf("expected handled action start_stream, got %+v", handledAction)
	}
	mu.Unlock()
}
