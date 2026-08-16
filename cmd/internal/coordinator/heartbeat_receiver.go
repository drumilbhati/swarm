package coordinator

import (
	"encoding/json"
	"net/http"

	"github.com/drumilbhati/swarm/cmd/internal/heartbeat"
)

func (c *Coordinator) ReceiveHeartBeat(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var hb heartbeat.Payload
	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		http.Error(w, "invalid heartbeat payload", http.StatusBadRequest)
		return
	}
	if hb.Worker == "" {
		http.Error(w, "worker is required", http.StatusBadRequest)
		return
	}

	c.mu.Lock()
	c.workers[hb.Worker] = hb.Timestamp
	c.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}
