package coordinator

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/drumilbhati/swarm/cmd/internal/heartbeat"
)

func TestReceiveHeartBeatRecordsWorker(t *testing.T) {
	coordinator := NewCoordinator()
	payload := heartbeat.Payload{
		Worker:    "worker-1",
		Timestamp: time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	resp := httptest.NewRecorder()
	coordinator.ReceiveHeartBeat(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, resp.Code)
	}
	if got := coordinator.workers[payload.Worker]; got.IsZero() || time.Since(got) > 5*time.Second {
		t.Fatalf("expected recent timestamp, got %s", got)
	}
}

func TestReceiveHeartBeatRejectsMissingWorker(t *testing.T) {
	coordinator := NewCoordinator()
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewBufferString(`{"timestamp":"2026-08-16T12:00:00Z"}`))
	resp := httptest.NewRecorder()

	coordinator.ReceiveHeartBeat(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}
}
