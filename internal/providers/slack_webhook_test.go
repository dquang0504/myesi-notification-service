package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSlackWebhookProvider_MissingURL(t *testing.T) {
	p := SlackWebhookProvider{}
	if err := p.SendSlackMessage(context.Background(), "", "hi"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSlackWebhookProvider_StatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	p := SlackWebhookProvider{Client: srv.Client()}
	if err := p.SendSlackMessage(context.Background(), srv.URL, "hi"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSlackWebhookProvider_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	p := SlackWebhookProvider{Client: srv.Client()}
	if err := p.SendSlackMessage(context.Background(), srv.URL, "hi"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
