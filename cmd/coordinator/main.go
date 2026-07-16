package main

import (
	"log"
	"net/http"
	"os"

	"github.com/drumilbhati/swarm/cmd/internal/coordinator"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	c := coordinator.NewController()

	r.Post("/tasks", c.SubmitTask)
	r.Post("/tasks/poll", c.MatchTask)

	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	} else if port[0] != ':' {
		port = ":" + port
	}

	log.Printf("Starting Swarm Coordinator on port %s...", port)
	log.Fatal(http.ListenAndServe(port, r))
}
