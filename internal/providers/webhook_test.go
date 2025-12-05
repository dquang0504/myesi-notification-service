package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenericWebhookProvider(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := GenericWebhookProvider{}
	if err := p.SendWebhook(context.Background(), server.URL, map[string]string{"hello": "world"}); err != nil {
		t.Fatalf("send webhook returned error: %v", err)
	}

	if received["hello"] != "world" {
		t.Fatalf("payload mismatch, got %v", received)
	}
}
