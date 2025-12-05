package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GenericWebhookProvider sends a JSON payload to an HTTP endpoint.
type GenericWebhookProvider struct {
	Client *http.Client
}

func (p GenericWebhookProvider) SendWebhook(ctx context.Context, url string, payload any) error {
	if url == "" {
		return fmt.Errorf("missing webhook url")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
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
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}
