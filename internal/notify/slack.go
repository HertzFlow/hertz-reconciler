package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Slack struct {
	webhookURL string
	client     *http.Client
}

func NewSlack(webhookURL string) *Slack {
	return &Slack{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Enabled returns true when a webhook URL is configured.
func (s *Slack) Enabled() bool { return s.webhookURL != "" }

type slackPayload struct {
	Text string `json:"text"`
}

// Send posts a plain-text message to the configured Slack incoming webhook.
// No-op when webhook URL is empty.
func (s *Slack) Send(ctx context.Context, text string) error {
	if !s.Enabled() {
		return nil
	}
	body, _ := json.Marshal(slackPayload{Text: text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}
