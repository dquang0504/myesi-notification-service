package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackWebhookProvider posts messages to Slack webhook URLs.
type SlackWebhookProvider struct {
	Client *http.Client
}

func (p SlackWebhookProvider) SendSlackMessage(ctx context.Context, webhookURL string, text string) error {
	if webhookURL == "" {
		return fmt.Errorf("missing webhook url")
	}

	payload := map[string]string{"text": text}
	body, _ := json.Marshal(payload)

	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}
