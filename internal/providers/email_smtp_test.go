package providers

import (
	"context"
	"testing"
)

func TestSMTPProvider_EmptyRecipients_NoError(t *testing.T) {
	p := SMTPProvider{}
	if err := p.SendEmail(context.Background(), nil, "s", "b"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestSMTPProvider_MissingHost_Error(t *testing.T) {
	p := SMTPProvider{Host: "", Port: 587, From: "x@y"}
	if err := p.SendEmail(context.Background(), []string{"a@b.com"}, "s", "b"); err == nil {
		t.Fatalf("expected error")
	}
}
