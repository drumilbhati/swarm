package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/drumilbhati/swarm/cmd/internal/coordinator"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	c := coordinator.NewController()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Launch background Liveness Sweeper (checks every 3s, evicts after 10s inactivity)
	go c.GetCoordinator().StartLivenessSweeper(ctx, 3*time.Second, 10*time.Second)

	r.Post("/tasks", c.SubmitTask)
	r.Post("/tasks/poll", c.MatchTask)
	r.Post("/heartbeat", c.ReceiveHeartBeat)

	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	} else if port[0] != ':' {
		port = ":" + port
	}

	log.Printf("Starting Swarm Coordinator on port %s...", port)
	log.Fatal(http.ListenAndServe(port, r))
}
