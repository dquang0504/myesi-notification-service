package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"myesi-notification-service/internal/api"
	"myesi-notification-service/internal/domain"

	"github.com/gofiber/fiber/v2"
)

type stubTemplates struct {
	listErr   error
	upsertErr error
}

func (s *stubTemplates) List(ctx domain.Context, limit, offset int) ([]domain.NotificationTemplate, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return []domain.NotificationTemplate{{ID: 1, Name: "t", EventType: "x", Channel: "email"}}, nil
}
func (s *stubTemplates) Upsert(ctx domain.Context, tpl domain.NotificationTemplate) (domain.NotificationTemplate, error) {
	if s.upsertErr != nil {
		return domain.NotificationTemplate{}, s.upsertErr
	}
	tpl.ID = 99
	return tpl, nil
}
func (s *stubTemplates) FindByEventAndChannel(ctx domain.Context, eventType, channel string) (*domain.NotificationTemplate, error) {
	return nil, nil
}

type stubPrefs struct {
	listErr error
	saveErr error
}

func (s *stubPrefs) List(ctx domain.Context, orgID int64, userID *int64, eventType string) ([]domain.NotificationPreference, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return []domain.NotificationPreference{{ID: 1, OrganizationID: orgID, EventType: eventType}}, nil
}
func (s *stubPrefs) Save(ctx domain.Context, pref domain.NotificationPreference) (domain.NotificationPreference, error) {
	if s.saveErr != nil {
		return domain.NotificationPreference{}, s.saveErr
	}
	if pref.ID == 0 {
		pref.ID = 123
	}
	return pref, nil
}

type stubLogs struct {
	listErr error
}

func (s *stubLogs) Insert(ctx domain.Context, log domain.NotificationLog) error { return nil }
func (s *stubLogs) List(ctx domain.Context, orgID int64, eventType, status, channel string, limit, offset int) ([]domain.NotificationLog, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return []domain.NotificationLog{{ID: 1, OrganizationID: orgID}}, nil
}

type stubInbox struct {
	listErr error
}

func (s *stubInbox) Save(ctx domain.Context, n domain.UserNotification) (domain.UserNotification, error) {
	return n, nil
}
func (s *stubInbox) List(ctx domain.Context, userID int64, orgID int64, unreadOnly bool, limit, offset int) ([]domain.UserNotification, int, int, error) {
	if s.listErr != nil {
		return nil, 0, 0, s.listErr
	}
	return []domain.UserNotification{{ID: 1, UserID: userID}}, 1, 1, nil
}
func (s *stubInbox) MarkRead(ctx domain.Context, id int64, userID int64) error { return nil }
func (s *stubInbox) MarkAllRead(ctx domain.Context, userID int64) error        { return nil }
func (s *stubInbox) Delete(ctx domain.Context, id int64, userID int64) error   { return nil }

type stubNotifier struct {
	err  error
	last domain.NotificationEvent
}

func (s *stubNotifier) HandleEvent(ctx domain.Context, evt domain.NotificationEvent) error {
	s.last = evt
	return s.err
}

func newApp(deps api.HandlerDeps) *fiber.App {
	app := fiber.New()
	api.RegisterRoutes(app, deps)
	return app
}

func readJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}

func TestHealthz(t *testing.T) {
	app := newApp(api.HandlerDeps{})
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
}

func TestListTemplates_DefaultPaging(t *testing.T) {
	app := newApp(api.HandlerDeps{Templates: &stubTemplates{}})
	req, _ := http.NewRequest(http.MethodGet, "/api/notification/templates", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
}

func TestListTemplates_RepoError(t *testing.T) {
	app := newApp(api.HandlerDeps{Templates: &stubTemplates{listErr: errors.New("db down")}})
	req, _ := http.NewRequest(http.MethodGet, "/api/notification/templates", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 500 {
		t.Fatalf("expected 500 got %d", resp.StatusCode)
	}
}

func TestUpsertTemplate_InvalidBody(t *testing.T) {
	app := newApp(api.HandlerDeps{Templates: &stubTemplates{}})
	req, _ := http.NewRequest(http.MethodPost, "/api/notification/templates", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestUpsertTemplate_Success(t *testing.T) {
	app := newApp(api.HandlerDeps{Templates: &stubTemplates{}})
	body := bytes.NewBufferString(`{"name":"x","event_type":"payment.success","channel":"email","subject":"s","body":"b","is_default":true}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/notification/templates", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	data := readJSON(t, resp)
	if data["id"].(float64) != 99 {
		t.Fatalf("expected id 99 got %v", data["id"])
	}
}

func TestListPreferences_Success(t *testing.T) {
	app := newApp(api.HandlerDeps{Preferences: &stubPrefs{}})
	req, _ := http.NewRequest(http.MethodGet, "/api/notification/preferences?organization_id=12&event_type=payment.success&user_id=9", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
}

func TestUpdatePreference_InvalidID(t *testing.T) {
	app := newApp(api.HandlerDeps{Preferences: &stubPrefs{}})
	req, _ := http.NewRequest(http.MethodPut, "/api/notification/preferences/abc", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestUpdatePreference_InvalidBody(t *testing.T) {
	app := newApp(api.HandlerDeps{Preferences: &stubPrefs{}})
	req, _ := http.NewRequest(http.MethodPut, "/api/notification/preferences/1", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestUpdatePreference_Success(t *testing.T) {
	app := newApp(api.HandlerDeps{Preferences: &stubPrefs{}})
	req, _ := http.NewRequest(http.MethodPut, "/api/notification/preferences/7",
		bytes.NewBufferString(`{"organization_id":1,"event_type":"payment.success","channel":"email","target":"a@b.com","enabled":true,"severity_min":"low"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	data := readJSON(t, resp)
	if data["id"].(float64) != 7 {
		t.Fatalf("expected id 7 got %v", data["id"])
	}
}

func TestListLogs_Success(t *testing.T) {
	app := newApp(api.HandlerDeps{Logs: &stubLogs{}})
	req, _ := http.NewRequest(http.MethodGet, "/api/notification/logs?organization_id=1&limit=10&offset=0", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
}

func TestInbox_NotEnabled(t *testing.T) {
	app := newApp(api.HandlerDeps{Inbox: nil})
	req, _ := http.NewRequest(http.MethodGet, "/api/notification/inbox", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 501 {
		t.Fatalf("expected 501 got %d", resp.StatusCode)
	}
}

func TestInbox_MissingUserID(t *testing.T) {
	app := newApp(api.HandlerDeps{Inbox: &stubInbox{}})
	req, _ := http.NewRequest(http.MethodGet, "/api/notification/inbox", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestInbox_ListOk(t *testing.T) {
	app := newApp(api.HandlerDeps{Inbox: &stubInbox{}})
	req, _ := http.NewRequest(http.MethodGet, "/api/notification/inbox?organization_id=1", nil)
	req.Header.Set("X-User-Id", "11")
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	data := readJSON(t, resp)
	d := data["data"].(map[string]any)
	if d["total"].(float64) != 1 || d["unread_count"].(float64) != 1 {
		t.Fatalf("unexpected counts: %v", d)
	}
}

func TestIngestEvent_ServiceUnavailable(t *testing.T) {
	app := newApp(api.HandlerDeps{Svc: nil})
	req, _ := http.NewRequest(http.MethodPost, "/api/notification/events", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 503 {
		t.Fatalf("expected 503 got %d", resp.StatusCode)
	}
}

func TestIngestEvent_TokenRequired(t *testing.T) {
	app := newApp(api.HandlerDeps{
		Svc:          &stubNotifier{},
		ServiceToken: "secret",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/notification/events", bytes.NewBufferString(`{"type":"payment.success","organization_id":1}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401 got %d", resp.StatusCode)
	}
}

func TestIngestEvent_InvalidBody(t *testing.T) {
	app := newApp(api.HandlerDeps{Svc: &stubNotifier{}})
	req, _ := http.NewRequest(http.MethodPost, "/api/notification/events", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestIngestEvent_SetsOccurredAtAndAccepted(t *testing.T) {
	n := &stubNotifier{}
	app := newApp(api.HandlerDeps{
		Svc:          n,
		ServiceToken: "secret",
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/notification/events",
		bytes.NewBufferString(`{"type":"payment.success","organization_id":1,"payload":{"x":1}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Token", "secret")

	before := time.Now().UTC().Add(-2 * time.Second)
	resp, _ := app.Test(req)
	after := time.Now().UTC().Add(2 * time.Second)

	if resp.StatusCode != 202 {
		t.Fatalf("expected 202 got %d", resp.StatusCode)
	}
	if n.last.EventType != "payment.success" || n.last.OrganizationID != 1 {
		t.Fatalf("unexpected event captured: %+v", n.last)
	}
	if n.last.OccurredAt.IsZero() {
		t.Fatalf("expected OccurredAt to be set")
	}
	if n.last.OccurredAt.Before(before) || n.last.OccurredAt.After(after) {
		t.Fatalf("OccurredAt out of expected range: %v", n.last.OccurredAt)
	}
}

func TestIngestEvent_PropagatesServiceError(t *testing.T) {
	app := newApp(api.HandlerDeps{
		Svc:          &stubNotifier{err: errors.New("boom")},
		ServiceToken: "",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/notification/events",
		bytes.NewBufferString(`{"type":"payment.success","organization_id":1}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 500 {
		t.Fatalf("expected 500 got %d", resp.StatusCode)
	}
}
