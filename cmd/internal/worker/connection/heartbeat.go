package connection

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/drumilbhati/swarm/cmd/internal/heartbeat"
)

func (c *Connection) SubmitHeartBeat(ctx context.Context) (heartbeat.Payload, bool, error) {
	workerID, err := workerID()
	if err != nil {
		return heartbeat.Payload{}, false, err
	}

	hb := heartbeat.Payload{
		Worker:    workerID,
		Timestamp: time.Now().UTC(),
	}
	jsonBytes, err := json.Marshal(hb)
	if err != nil {
		return heartbeat.Payload{}, false, err
	}

	if len(c.coordinatorURLs) == 0 {
		return heartbeat.Payload{}, false, fmt.Errorf("no coordinator URLs configured")
	}

	delivered := true
	anyAttempted := false

	for _, coordinatorURL := range c.coordinatorURLs {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(coordinatorURL, "/")+"/heartbeat", bytes.NewReader(jsonBytes))
		if err != nil {
			delivered = false
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			delivered = false
			continue
		}
		resp.Body.Close()

		anyAttempted = true
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			delivered = false
		}
	}

	if !anyAttempted {
		delivered = false
	}

	return hb, delivered, nil
}

func workerID() (string, error) {
	if id := os.Getenv("WORKER_ID"); id != "" {
		return id, nil
	}

	return os.Hostname()
}
