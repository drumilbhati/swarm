package connection

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/drumilbhati/swarm/cmd/internal/heartbeat"
)

func TestSubmitHeartBeatSendsPayload(t *testing.T) {
	t.Setenv("WORKER_ID", "worker-1")

	var received heartbeat.Payload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected method %s, got %s", http.MethodPost, r.Method)
		}
		if r.URL.Path != "/heartbeat" {
			t.Errorf("expected path /heartbeat, got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode heartbeat: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	connection := NewConnection([]string{server.URL}, time.Second, nil, nil, nil)
	heartbeatSent, delivered, err := connection.SubmitHeartBeat(context.Background())
	if err != nil {
		t.Fatalf("SubmitHeartBeat returned error: %v", err)
	}
	if !delivered {
		t.Fatal("expected heartbeat to be delivered")
	}
	if heartbeatSent.Worker != "worker-1" {
		t.Fatalf("expected worker ID worker-1, got %q", heartbeatSent.Worker)
	}
	if received.Worker != "worker-1" {
		t.Fatalf("expected received worker ID worker-1, got %q", received.Worker)
	}
	if received.Timestamp.IsZero() {
		t.Fatal("expected a non-zero timestamp")
	}
}

func TestSubmitHeartBeatReturnsFalseWhenNoCoordinatorAccepts(t *testing.T) {
	connection := NewConnection([]string{"http://127.0.0.1:1"}, time.Second, nil, nil, nil)
	_, delivered, err := connection.SubmitHeartBeat(context.Background())
	if err != nil {
		t.Fatalf("SubmitHeartBeat returned error: %v", err)
	}
	if delivered {
		t.Fatal("expected heartbeat delivery to fail")
	}
}
