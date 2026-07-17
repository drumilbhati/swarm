package coordinator

import (
	"encoding/json"
	"net/http"

	"github.com/drumilbhati/swarm/cmd/internal/worker/connection"
	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
)

type Controller struct {
	coordinator *Coordinator
}

func NewController() *Controller {
	return &Controller{
		coordinator: NewCoordinator(),
	}
}

func (c *Controller) SubmitTask(w http.ResponseWriter, r *http.Request) {
	var task executor.Task

	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := c.coordinator.SubmitTask(task); err != nil {
		http.Error(w, "Task exceeds Coordinator capacity limits: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode("Successfully submitted task")
}

func (c *Controller) MatchTask(w http.ResponseWriter, r *http.Request) {
	var workerHeadroom connection.Headroom
	if err := json.NewDecoder(r.Body).Decode(&workerHeadroom); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	task, found := c.coordinator.MatchTask(workerHeadroom)
	if !found {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
