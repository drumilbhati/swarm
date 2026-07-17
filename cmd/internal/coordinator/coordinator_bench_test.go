package coordinator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/drumilbhati/swarm/cmd/internal/worker/connection"
	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
)

// Configure global client pool to support high concurrent loopback connections
func init() {
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 500
	http.DefaultTransport.(*http.Transport).MaxIdleConns = 500
	http.DefaultTransport.(*http.Transport).IdleConnTimeout = 90 * time.Second
}

// BenchmarkHTTPE2EParallelMatrix runs a matrix of jobs vs workers E2E over loopback HTTP sockets.
func BenchmarkHTTPE2EParallelMatrix(b *testing.B) {
	rng := rand.New(rand.NewSource(42))

	sizes := []int{1000, 10000, 100000, 1000000}
	workers := []int{10, 50, 100, 200}

	for _, size := range sizes {
		for _, wCount := range workers {
			name := fmt.Sprintf("Jobs-%d/Workers-%d", size, wCount)
			b.Run(name, func(b *testing.B) {
				coord := NewCoordinator()

				// Populate queue with N tasks
				for i := 0; i < size; i++ {
					task := executor.Task{
						ID:    fmt.Sprintf("task-%d", i),
						Image: "alpine",
						ResourceRequirement: executor.ResourceRequirement{
							RequiredSystemCPU:    rng.Float64() * 4.0,
							RequiredSystemMemory: float64(rng.Intn(1024*1024*1024) + 10*1024*1024),
							RequiredProcessCPU:   0.1,
							RequiredProcessMemory: 5 * 1024 * 1024,
						},
					}
					coord.SubmitTask(task)
				}

				controller := &Controller{coordinator: coord}

				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/tasks/poll" {
						controller.MatchTask(w, r)
					}
				}))
				defer server.Close()

				client := &http.Client{
					Transport: http.DefaultTransport,
				}

				payload, _ := json.Marshal(connection.Headroom{
					AvailableSystemCPU:    2.0,
					AvailableSystemMemory: 512 * 1024 * 1024,
					AvailableProcessCPU:   1.0,
					AvailableProcessMemory: 100 * 1024 * 1024,
				})

				cores := runtime.GOMAXPROCS(0)
				parallelism := wCount / cores
				if parallelism < 1 {
					parallelism = 1
				}
				b.SetParallelism(parallelism)

				b.ResetTimer()

				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						resp, err := client.Post(server.URL+"/tasks/poll", "application/json", bytes.NewReader(payload))
						if err != nil {
							b.Fatal(err)
						}
						_, _ = io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}
				})
			})
		}
	}
}

// BenchmarkHTTPE2EClusterComparison compares single coordinator vs 5 work-stealing coordinators.
func BenchmarkHTTPE2EClusterComparison(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	size := 1000000
	wCount := 100

	payload, _ := json.Marshal(connection.Headroom{
		AvailableSystemCPU:    2.0,
		AvailableSystemMemory: 512 * 1024 * 1024,
		AvailableProcessCPU:   1.0,
		AvailableProcessMemory: 100 * 1024 * 1024,
	})

	// 1. Single Coordinator Baseline
	b.Run("Single-Coordinator", func(b *testing.B) {
		coord := NewCoordinator()

		for i := 0; i < size; i++ {
			task := executor.Task{
				ID:    fmt.Sprintf("task-%d", i),
				Image: "alpine",
				ResourceRequirement: executor.ResourceRequirement{
					RequiredSystemCPU:    rng.Float64() * 4.0,
					RequiredSystemMemory: float64(rng.Intn(1024*1024*1024) + 10*1024*1024),
					RequiredProcessCPU:   0.1,
					RequiredProcessMemory: 5 * 1024 * 1024,
				},
			}
			coord.SubmitTask(task)
		}

		controller := &Controller{coordinator: coord}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/tasks/poll" {
				controller.MatchTask(w, r)
			}
		}))
		defer server.Close()

		cores := runtime.GOMAXPROCS(0)
		parallelism := wCount / cores
		if parallelism < 1 {
			parallelism = 1
		}
		b.SetParallelism(parallelism)

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req, _ := http.NewRequest("POST", server.URL+"/tasks/poll", bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					b.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	})

	// 2. Multi-Coordinator Cluster (5 Coordinators with Work Stealing)
	b.Run("Cluster-5-Coordinators", func(b *testing.B) {
		const numCoords = 5
		coords := make([]*Coordinator, numCoords)
		servers := make([]*httptest.Server, numCoords)
		urls := make([]string, numCoords)

		for i := 0; i < numCoords; i++ {
			coords[i] = NewCoordinator()
			c := coords[i]
			controller := &Controller{coordinator: c}
			servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/tasks/poll" {
					controller.MatchTask(w, r)
				}
			}))
			urls[i] = servers[i].URL
		}
		defer func() {
			for _, s := range servers {
				s.Close()
			}
		}()

		// Distribute 1M tasks across the 5 coordinators
		for i := 0; i < size; i++ {
			task := executor.Task{
				ID:    fmt.Sprintf("task-%d", i),
				Image: "alpine",
				ResourceRequirement: executor.ResourceRequirement{
					RequiredSystemCPU:    rng.Float64() * 4.0,
					RequiredSystemMemory: float64(rng.Intn(1024*1024*1024) + 10*1024*1024),
					RequiredProcessCPU:   0.1,
					RequiredProcessMemory: 5 * 1024 * 1024,
				},
			}
			coords[i%numCoords].SubmitTask(task)
		}

		cores := runtime.GOMAXPROCS(0)
		parallelism := wCount / cores
		if parallelism < 1 {
			parallelism = 1
		}
		b.SetParallelism(parallelism)

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				offset := rand.Intn(numCoords)
				for i := 0; i < numCoords; i++ {
					targetIdx := (offset + i) % numCoords
					url := urls[targetIdx]

					req, _ := http.NewRequest("POST", url+"/tasks/poll", bytes.NewReader(payload))
					req.Header.Set("Content-Type", "application/json")
					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						continue
					}
					
					statusCode := resp.StatusCode
					_, _ = io.Copy(io.Discard, resp.Body)
					resp.Body.Close()

					if statusCode == http.StatusNoContent {
						continue
					}
					if statusCode == http.StatusOK {
						break
					}
				}
			}
		})
	})
}
