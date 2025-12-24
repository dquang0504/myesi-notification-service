package api_test

import (
	"net/http"
	"testing"

	"myesi-notification-service/internal/api"
	"myesi-notification-service/internal/domain"

	"github.com/gofiber/fiber/v2"
)

type inboxMock struct {
	markReadCalled    int
	markAllReadCalled int
	deleteCalled      int
	markReadErr       error
	markAllReadErr    error
	deleteErr         error
	lastID            int64
	lastUserID        int64
}

func (m *inboxMock) Save(ctx domain.Context, n domain.UserNotification) (domain.UserNotification, error) {
	return n, nil
}
func (m *inboxMock) List(ctx domain.Context, userID int64, orgID int64, unreadOnly bool, limit, offset int) ([]domain.UserNotification, int, int, error) {
	return nil, 0, 0, nil
}
func (m *inboxMock) MarkRead(ctx domain.Context, id int64, userID int64) error {
	m.markReadCalled++
	m.lastID = id
	m.lastUserID = userID
	return m.markReadErr
}
func (m *inboxMock) MarkAllRead(ctx domain.Context, userID int64) error {
	m.markAllReadCalled++
	m.lastUserID = userID
	return m.markAllReadErr
}
func (m *inboxMock) Delete(ctx domain.Context, id int64, userID int64) error {
	m.deleteCalled++
	m.lastID = id
	m.lastUserID = userID
	return m.deleteErr
}

func newInboxApp(inbox domain.InboxRepository) *fiber.App {
	app := fiber.New()
	api.RegisterRoutes(app, api.HandlerDeps{
		Inbox: inbox,
	})
	return app
}

// ========== markInboxRead ==========
func TestInbox_MarkRead_NotEnabled(t *testing.T) {
	app := newInboxApp(nil)

	req, _ := http.NewRequest(http.MethodPatch, "/api/notification/inbox/1/read", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 501 {
		t.Fatalf("expected 501 got %d", resp.StatusCode)
	}
}

func TestInbox_MarkRead_MissingUserID(t *testing.T) {
	app := newInboxApp(&inboxMock{})

	req, _ := http.NewRequest(http.MethodPatch, "/api/notification/inbox/1/read", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestInbox_MarkRead_InvalidID(t *testing.T) {
	app := newInboxApp(&inboxMock{})

	req, _ := http.NewRequest(http.MethodPatch, "/api/notification/inbox/abc/read", nil)
	req.Header.Set("X-User-Id", "11")
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestInbox_MarkRead_DBError(t *testing.T) {
	m := &inboxMock{markReadErr: fiber.ErrInternalServerError}
	app := newInboxApp(m)

	req, _ := http.NewRequest(http.MethodPatch, "/api/notification/inbox/9/read", nil)
	req.Header.Set("X-User-Id", "11")
	resp, _ := app.Test(req)

	if resp.StatusCode != 500 {
		t.Fatalf("expected 500 got %d", resp.StatusCode)
	}
	if m.markReadCalled != 1 || m.lastID != 9 || m.lastUserID != 11 {
		t.Fatalf("unexpected calls=%d id=%d user=%d", m.markReadCalled, m.lastID, m.lastUserID)
	}
}

func TestInbox_MarkRead_Success(t *testing.T) {
	m := &inboxMock{}
	app := newInboxApp(m)

	req, _ := http.NewRequest(http.MethodPatch, "/api/notification/inbox/9/read", nil)
	req.Header.Set("X-User-Id", "11")
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	if m.markReadCalled != 1 || m.lastID != 9 || m.lastUserID != 11 {
		t.Fatalf("unexpected calls=%d id=%d user=%d", m.markReadCalled, m.lastID, m.lastUserID)
	}
}

// ========== markAllInboxRead ==========
func TestInbox_MarkAllRead_NotEnabled(t *testing.T) {
	app := newInboxApp(nil)

	req, _ := http.NewRequest(http.MethodPatch, "/api/notification/inbox/read-all", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 501 {
		t.Fatalf("expected 501 got %d", resp.StatusCode)
	}
}

func TestInbox_MarkAllRead_MissingUserID(t *testing.T) {
	app := newInboxApp(&inboxMock{})

	req, _ := http.NewRequest(http.MethodPatch, "/api/notification/inbox/read-all", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestInbox_MarkAllRead_DBError(t *testing.T) {
	m := &inboxMock{markAllReadErr: fiber.ErrInternalServerError}
	app := newInboxApp(m)

	req, _ := http.NewRequest(http.MethodPatch, "/api/notification/inbox/read-all", nil)
	req.Header.Set("X-User-Id", "11")
	resp, _ := app.Test(req)

	if resp.StatusCode != 500 {
		t.Fatalf("expected 500 got %d", resp.StatusCode)
	}
	if m.markAllReadCalled != 1 || m.lastUserID != 11 {
		t.Fatalf("unexpected calls=%d user=%d", m.markAllReadCalled, m.lastUserID)
	}
}

func TestInbox_MarkAllRead_Success(t *testing.T) {
	m := &inboxMock{}
	app := newInboxApp(m)

	req, _ := http.NewRequest(http.MethodPatch, "/api/notification/inbox/read-all", nil)
	req.Header.Set("X-User-Id", "11")
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	if m.markAllReadCalled != 1 || m.lastUserID != 11 {
		t.Fatalf("unexpected calls=%d user=%d", m.markAllReadCalled, m.lastUserID)
	}
}

// ========== deleteInbox ==========
func TestInbox_Delete_NotEnabled(t *testing.T) {
	app := newInboxApp(nil)

	req, _ := http.NewRequest(http.MethodDelete, "/api/notification/inbox/1", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 501 {
		t.Fatalf("expected 501 got %d", resp.StatusCode)
	}
}

func TestInbox_Delete_MissingUserID(t *testing.T) {
	app := newInboxApp(&inboxMock{})

	req, _ := http.NewRequest(http.MethodDelete, "/api/notification/inbox/1", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestInbox_Delete_InvalidID(t *testing.T) {
	app := newInboxApp(&inboxMock{})

	req, _ := http.NewRequest(http.MethodDelete, "/api/notification/inbox/abc", nil)
	req.Header.Set("X-User-Id", "11")
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestInbox_Delete_DBError(t *testing.T) {
	m := &inboxMock{deleteErr: fiber.ErrInternalServerError}
	app := newInboxApp(m)

	req, _ := http.NewRequest(http.MethodDelete, "/api/notification/inbox/9", nil)
	req.Header.Set("X-User-Id", "11")
	resp, _ := app.Test(req)

	if resp.StatusCode != 500 {
		t.Fatalf("expected 500 got %d", resp.StatusCode)
	}
	if m.deleteCalled != 1 || m.lastID != 9 || m.lastUserID != 11 {
		t.Fatalf("unexpected calls=%d id=%d user=%d", m.deleteCalled, m.lastID, m.lastUserID)
	}
}

func TestInbox_Delete_Success(t *testing.T) {
	m := &inboxMock{}
	app := newInboxApp(m)

	req, _ := http.NewRequest(http.MethodDelete, "/api/notification/inbox/9", nil)
	req.Header.Set("X-User-Id", "11")
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	if m.deleteCalled != 1 || m.lastID != 9 || m.lastUserID != 11 {
		t.Fatalf("unexpected calls=%d id=%d user=%d", m.deleteCalled, m.lastID, m.lastUserID)
	}
}
