// Command client provides an interactive terminal interface for submitting
// workloads to a Swarm coordinator.
package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type task struct {
	ID                  string        `json:"id"`
	Image               string        `json:"image"`
	Cmd                 []string      `json:"cmd"`
	Duration            time.Duration `json:"duration,omitempty"`
	ResourceRequirement struct {
		RequiredProcessCPU    float64 `json:"required_process_cpu,omitempty"`
		RequiredProcessMemory float64 `json:"required_process_memory,omitempty"`
		RequiredSystemCPU     float64 `json:"required_system_cpu"`
		RequiredSystemMemory  float64 `json:"required_system_memory"`
	} `json:"resource_requirement"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := loadDotEnv(".env"); err != nil {
		return err
	}
	base := strings.TrimSpace(os.Getenv("COORDINATOR_URL"))
	if base == "" {
		return errors.New("COORDINATOR_URL is not set; add it to .env (for example http://localhost:8081)")
	}
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("invalid COORDINATOR_URL %q", base)
	}

	in := bufio.NewReader(os.Stdin)
	fmt.Println("Swarm workload submission")
	t := task{}
	id, err := newTaskID()
	if err != nil {
		return fmt.Errorf("generate task ID: %w", err)
	}
	t.ID, err = prompt(in, "Task ID", id)
	if err != nil {
		return err
	}
	t.Image, err = prompt(in, "Container image", "alpine:latest")
	if err != nil {
		return err
	}
	command, err := prompt(in, "Command (space-separated)", "echo hello from Swarm")
	if err != nil {
		return err
	}
	t.Cmd = strings.Fields(command)
	if len(t.Cmd) == 0 {
		return errors.New("command cannot be empty")
	}
	cpu, err := promptFloat(in, "Required system CPU (cores)", "0.5")
	if err != nil {
		return err
	}
	mem, err := promptFloat(in, "Required system memory (bytes)", "52428800")
	if err != nil {
		return err
	}
	t.ResourceRequirement.RequiredSystemCPU = cpu
	t.ResourceRequirement.RequiredSystemMemory = mem

	body, err := json.Marshal(t)
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(base, "/") + "/tasks"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("could not reach coordinator: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("coordinator rejected request (%s)", resp.Status)
	}
	fmt.Printf("Workload %q submitted. Waiting for completion...\n", t.ID)
	resultURL := strings.TrimRight(base, "/") + "/tasks/" + url.PathEscape(t.ID) + "/result"
	poll := time.NewTicker(2 * time.Second)
	defer poll.Stop()
	client := &http.Client{Timeout: 15 * time.Second}
	const maxCoordinatorFailures = 3
	consecutiveFailures := 0
	for {
		select {
		case <-poll.C:
			resultResp, err := client.Get(resultURL)
			if err != nil {
				consecutiveFailures++
				fmt.Printf("Coordinator unavailable (%d/%d): %v\n", consecutiveFailures, maxCoordinatorFailures, err)
				if consecutiveFailures >= maxCoordinatorFailures {
					fmt.Println("Closing client connection gracefully; the workload may still be running.")
					return nil
				}
				continue
			}
			consecutiveFailures = 0
			if resultResp.StatusCode == http.StatusAccepted {
				resultResp.Body.Close()
				continue
			}
			if resultResp.StatusCode != http.StatusOK {
				resultResp.Body.Close()
				return fmt.Errorf("result request failed: %s", resultResp.Status)
			}
			var result struct {
				TaskID string `json:"task_id"`
				Output string `json:"output"`
				Error  string `json:"error"`
			}
			decodeErr := json.NewDecoder(io.LimitReader(resultResp.Body, 4<<20)).Decode(&result)
			resultResp.Body.Close()
			if decodeErr != nil {
				return decodeErr
			}
			fmt.Printf("\nJob %s finished.\n", result.TaskID)
			if result.Output != "" {
				fmt.Print(result.Output)
				if !strings.HasSuffix(result.Output, "\n") {
					fmt.Println()
				}
			}
			if result.Error != "" {
				return fmt.Errorf("job failed: %s", result.Error)
			}
			return nil
		}
	}
}

func newTaskID() (string, error) {
	id, err := uuid.NewRandomFromReader(rand.Reader)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func prompt(in *bufio.Reader, label, fallback string) (string, error) {
	fmt.Printf("%s [%s]: ", label, fallback)
	line, err := in.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	value := strings.TrimSpace(line)
	if value == "" {
		value = fallback
	}
	return value, nil
}

func promptFloat(in *bufio.Reader, label, fallback string) (float64, error) {
	for {
		value, err := prompt(in, label, fallback)
		if err != nil {
			return 0, err
		}
		n, err := strconv.ParseFloat(value, 64)
		if err == nil && n > 0 {
			return n, nil
		}
		fmt.Println("Enter a number greater than zero.")
	}
}

func loadDotEnv(path string) error {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	for lineNo, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid .env line %d", lineNo+1)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if key == "" {
			return fmt.Errorf("invalid .env key on line %d", lineNo+1)
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return nil
}
