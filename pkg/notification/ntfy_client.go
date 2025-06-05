package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// NtfyClient sends notifications to ntfy.sh
type NtfyClient struct {
	server     string
	topic      string
	httpClient *http.Client
}

// NewNtfyClient creates a new ntfy.sh client
func NewNtfyClient(server, topic string) *NtfyClient {
	return &NtfyClient{
		server: server,
		topic:  topic,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send sends a notification to ntfy.sh
func (c *NtfyClient) Send(notification Notification) error {
	if c.topic == "" {
		return fmt.Errorf("ntfy topic not configured")
	}

	// Create the request payload
	payload := map[string]interface{}{
		"topic":   c.topic,
		"title":   notification.Title,
		"message": notification.Message,
		"tags":    []string{"claude-code", notification.Pattern},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Create the request
	url := fmt.Sprintf("%s/", c.server)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ntfy returned status %d", resp.StatusCode)
	}

	return nil
}
