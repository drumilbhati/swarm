package coordinator

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/drumilbhati/swarm/cmd/internal/heartbeat"
)

func (c *Coordinator) ReceiveHeartBeat(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	defer r.Body.Close()

	var hb heartbeat.Payload
	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		http.Error(w, "invalid heartbeat payload", http.StatusBadRequest)
		return
	}
	if hb.Worker == "" || len(hb.Worker) > 128 {
		http.Error(w, "worker is required", http.StatusBadRequest)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	const maxWorkers = 10000
	_, exists := c.workers[hb.Worker]
	if !exists && len(c.workers) >= maxWorkers {
		http.Error(w, "worker registry limit reached", http.StatusBadRequest)
		return
	}

	c.workers[hb.Worker] = time.Now()

	w.WriteHeader(http.StatusNoContent)
}
