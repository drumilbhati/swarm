package coordinator

import (
	"encoding/json"
	"net/http"

	"github.com/drumilbhati/swarm/cmd/internal/worker/connection"
	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
	"github.com/go-chi/chi"
)

type Controller struct {
	coordinator *Coordinator
}

func (c *Controller) SubmitResult(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	var result TaskResult
	if taskID == "" || json.NewDecoder(r.Body).Decode(&result) != nil {
		http.Error(w, "invalid task result", http.StatusBadRequest)
		return
	}
	c.coordinator.StoreResult(taskID, result)
	w.WriteHeader(http.StatusAccepted)
}

func (c *Controller) GetResult(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	result, ok := c.coordinator.Result(taskID)
	if !ok {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func NewController() *Controller {
	return &Controller{
		coordinator: NewCoordinator(),
	}
}

func (c *Controller) GetCoordinator() *Coordinator {
	return c.coordinator
}

func (c *Controller) ReceiveHeartBeat(w http.ResponseWriter, r *http.Request) {
	c.coordinator.ReceiveHeartBeat(w, r)
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

	workerID := r.Header.Get("X-Worker-ID")
	if workerID == "" {
		workerID = r.Header.Get("Worker-ID")
	}

	task, found := c.coordinator.MatchTask(workerHeadroom, workerID)
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
