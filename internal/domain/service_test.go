package domain

import (
	"context"
	"errors"
	"testing"

	"myesi-notification-service/internal/templates"
)

type stubTemplateRepo struct{ tpl NotificationTemplate }

type stubPrefRepo struct{ prefs []NotificationPreference }

type stubLogRepo struct{ entries []NotificationLog }

type stubEmail struct {
	to      []string
	subject string
	body    string
	err     error
}

type stubSlack struct {
	msg string
	url string
	err error
}

type stubWebhook struct {
	url string
	err error
}

type stubMetrics struct{}

func (r *stubTemplateRepo) List(ctx Context, limit, offset int) ([]NotificationTemplate, error) {
	return []NotificationTemplate{r.tpl}, nil
}
func (r *stubTemplateRepo) Upsert(ctx Context, tpl NotificationTemplate) (NotificationTemplate, error) {
	r.tpl = tpl
	return tpl, nil
}
func (r *stubTemplateRepo) FindByEventAndChannel(ctx Context, eventType, channel string) (*NotificationTemplate, error) {
	return &r.tpl, nil
}

func (r *stubPrefRepo) List(ctx Context, orgID int64, userID *int64, eventType string) ([]NotificationPreference, error) {
	return r.prefs, nil
}
func (r *stubPrefRepo) Save(ctx Context, pref NotificationPreference) (NotificationPreference, error) {
	r.prefs = append(r.prefs, pref)
	return pref, nil
}

func (s *stubEmail) SendEmail(ctx Context, to []string, subject, body string) error {
	if s.err != nil {
		return s.err
	}
	s.to = append(s.to, to...)
	s.subject = subject
	s.body = body
	return nil
}

func (s *stubSlack) SendSlackMessage(ctx Context, webhookURL, text string) error {
	if s.err != nil {
		return s.err
	}
	s.url = webhookURL
	s.msg = text
	return nil
}

func (s *stubWebhook) SendWebhook(ctx Context, url string, payload any) error {
	if s.err != nil {
		return s.err
	}
	s.url = url
	return nil
}

func TestHandleEventUsesPreferences(t *testing.T) {
	tplRepo := &stubTemplateRepo{tpl: NotificationTemplate{Subject: "Alert {{.event.type}}", Body: "Hello"}}
	prefRepo := &stubPrefRepo{prefs: []NotificationPreference{{OrganizationID: 1, EventType: "vulnerability.found", Channel: ChannelEmail, Target: "a@example.com", Enabled: true}}}
	logRepo := &stubLogRepo{}
	email := &stubEmail{}

	svc := &NotificationService{
		Templates:   tplRepo,
		Preferences: prefRepo,
		Logs:        logRepo,
		Email:       email,
		Slack:       &stubSlack{},
		Webhook:     &stubWebhook{},
		Renderer:    templates.Renderer{},
	}

	evt := NotificationEvent{EventType: "vulnerability.found", OrganizationID: 1, Payload: map[string]interface{}{"id": "123"}}
	if err := svc.HandleEvent(context.Background(), evt); err != nil {
		t.Fatalf("handle event returned error: %v", err)
	}

	if len(email.to) == 0 {
		t.Fatalf("expected email to be sent via preference")
	}
	if len(logRepo.entries) == 0 {
		t.Fatalf("expected log entry")
	}
}

func TestHandleEventSkipsWhenSeverityBelowThreshold(t *testing.T) {
	prefRepo := &stubPrefRepo{prefs: []NotificationPreference{{OrganizationID: 1, EventType: "vulnerability.found", Channel: ChannelEmail, Target: "a@example.com", Enabled: true, SeverityMin: "high"}}}
	svc := &NotificationService{
		Templates:   &stubTemplateRepo{tpl: NotificationTemplate{Subject: "s", Body: "b"}},
		Preferences: prefRepo,
		Logs:        &stubLogRepo{},
		Email:       &stubEmail{},
		Slack:       &stubSlack{},
		Webhook:     &stubWebhook{},
		Renderer:    templates.Renderer{},
	}

	evt := NotificationEvent{EventType: "vulnerability.found", OrganizationID: 1, Severity: "low"}
	if err := svc.HandleEvent(context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleEventFallbackOnSendError(t *testing.T) {
	prefRepo := &stubPrefRepo{prefs: []NotificationPreference{{OrganizationID: 1, EventType: "project.scan.completed", Channel: ChannelEmail, Target: "a@example.com", Enabled: true}}}
	email := &stubEmail{err: errors.New("smtp down")}
	logRepo := &stubLogRepo{}

	svc := &NotificationService{
		Templates:   &stubTemplateRepo{tpl: NotificationTemplate{Subject: "s", Body: "b"}},
		Preferences: prefRepo,
		Logs:        logRepo,
		Email:       email,
		Slack:       &stubSlack{},
		Webhook:     &stubWebhook{},
		Renderer:    templates.Renderer{},
	}

	evt := NotificationEvent{EventType: "project.scan.completed", OrganizationID: 1}
	_ = svc.HandleEvent(context.Background(), evt)

	if len(logRepo.entries) == 0 {
		t.Fatalf("expected log entry recorded on failure")
	}
}
