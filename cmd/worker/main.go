package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/drumilbhati/swarm/cmd/internal/worker/connection"
	decisionengine "github.com/drumilbhati/swarm/cmd/internal/worker/decision_engine"
	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
	"github.com/drumilbhati/swarm/cmd/internal/worker/telemetry"
)

func main() {
	monitor, err := telemetry.NewMonitor()
	if err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}

	engine := &decisionengine.DecisionEngineData{
		ConcurrencyLimit: 4, // Max 4 concurrent tasks
		Threshold: telemetry.UsageStats{
			SystemCPUUsage:     80.0, // Maintain system CPU usage below 80%
			SystemMemoryUsage:  80.0, // Maintain system RAM usage below 80%
			ProcessCPUUsage:    60.0, // Limit worker container CPU to 60%
			ProcessMemoryUsage: 60.0, // Limit worker container Memory to 60%
		},
	}

	exec, err := executor.NewDockerExecutor()
	if err != nil {
		log.Fatalf("Failed to initialize Docker executor: %v", err)
	}

	var urls []string
	if env := os.Getenv("COORDINATOR_URLS"); env != "" {
		urls = strings.Split(env, ",")
	} else if env := os.Getenv("COORDINATOR_URL"); env != "" {
		urls = strings.Split(env, ",")
	} else {
		urls = []string{"http://localhost:8081"}
	}

	pollInterval := 2 * time.Second
	conn := connection.NewConnection(urls, pollInterval, monitor, engine, exec)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down worker gracefully...")
		cancel()
	}()

	log.Printf("Starting Swarm Worker connecting to %v...", urls)
	conn.Start(ctx)
}
