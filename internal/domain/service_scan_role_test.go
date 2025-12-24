package domain

import (
	"context"
	"testing"

	"myesi-notification-service/internal/templates"
)

// ---- stubs ----
type tplRepoStub struct{ tpl NotificationTemplate }

func (r *tplRepoStub) List(ctx Context, limit, offset int) ([]NotificationTemplate, error) {
	return []NotificationTemplate{r.tpl}, nil
}
func (r *tplRepoStub) Upsert(ctx Context, tpl NotificationTemplate) (NotificationTemplate, error) {
	r.tpl = tpl
	return tpl, nil
}
func (r *tplRepoStub) FindByEventAndChannel(ctx Context, eventType, channel string) (*NotificationTemplate, error) {
	return &r.tpl, nil
}

type prefRepoNone struct{}

func (r *prefRepoNone) List(ctx Context, orgID int64, userID *int64, eventType string) ([]NotificationPreference, error) {
	return nil, nil
}
func (r *prefRepoNone) Save(ctx Context, pref NotificationPreference) (NotificationPreference, error) {
	return pref, nil
}

type logRepoNoop struct{}

func (r *logRepoNoop) Insert(ctx Context, log NotificationLog) error { return nil }
func (r *logRepoNoop) List(ctx Context, orgID int64, eventType, status, channel string, limit, offset int) ([]NotificationLog, error) {
	return nil, nil
}

type emailNoop struct{}

func (e *emailNoop) SendEmail(ctx Context, to []string, subject, body string) error { return nil }

type slackNoop struct{}

func (s *slackNoop) SendSlackMessage(ctx Context, webhookURL, text string) error { return nil }

type webhookNoop struct{}

func (w *webhookNoop) SendWebhook(ctx Context, url string, payload any) error { return nil }

type inboxSpy struct{ saved []UserNotification }

func (s *inboxSpy) Save(ctx Context, n UserNotification) (UserNotification, error) {
	s.saved = append(s.saved, n)
	return n, nil
}
func (s *inboxSpy) List(ctx Context, userID int64, orgID int64, unreadOnly bool, limit, offset int) ([]UserNotification, int, int, error) {
	return nil, 0, 0, nil
}
func (s *inboxSpy) MarkRead(ctx Context, id int64, userID int64) error { return nil }
func (s *inboxSpy) MarkAllRead(ctx Context, userID int64) error        { return nil }
func (s *inboxSpy) Delete(ctx Context, id int64, userID int64) error   { return nil }

type orgUsersSpy struct {
	allCalls   int
	roleCalls  int
	lastRole   string
	allReturn  []int64
	roleReturn map[string][]int64
}

func (o *orgUsersSpy) ListUserIDsByOrg(ctx Context, orgID int64) ([]int64, error) {
	o.allCalls++
	return o.allReturn, nil
}
func (o *orgUsersSpy) ListUserIDsByOrgWithRole(ctx Context, orgID int64, role string) ([]int64, error) {
	o.roleCalls++
	o.lastRole = role
	if o.roleReturn == nil {
		return nil, nil
	}
	return o.roleReturn[role], nil
}

// ---- tests ----

func TestInboxBroadcast_ProjectScan_DefaultRoleDeveloper(t *testing.T) {
	inbox := &inboxSpy{}
	orgUsers := &orgUsersSpy{
		roleReturn: map[string][]int64{
			"developer": {101, 102},
		},
	}
	svc := &NotificationService{
		Templates:   &tplRepoStub{tpl: NotificationTemplate{Subject: "Scan", Body: "Done"}},
		Preferences: &prefRepoNone{},
		Logs:        &logRepoNoop{},
		Inbox:       inbox,
		OrgUsers:    orgUsers,
		Email:       &emailNoop{},
		Slack:       &slackNoop{},
		Webhook:     &webhookNoop{},
		Renderer:    templates.Renderer{},
	}

	evt := NotificationEvent{
		EventType:      "project.scan.completed",
		OrganizationID: 1,
		// UserID nil => triggers org-wide inbox broadcast path
		Payload: map[string]interface{}{
			"project": "myproj",
		},
	}

	if err := svc.HandleEvent(context.Background(), evt); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if orgUsers.roleCalls != 1 {
		t.Fatalf("expected roleCalls=1 got %d", orgUsers.roleCalls)
	}
	if orgUsers.lastRole != "developer" {
		t.Fatalf("expected default role developer got %s", orgUsers.lastRole)
	}
	if len(inbox.saved) != 2 {
		t.Fatalf("expected 2 inbox items got %d", len(inbox.saved))
	}
	if inbox.saved[0].UserID != 101 || inbox.saved[1].UserID != 102 {
		t.Fatalf("unexpected recipients: %#v", inbox.saved)
	}
}

func TestInboxBroadcast_ProjectScan_TargetRoleOverride(t *testing.T) {
	inbox := &inboxSpy{}
	orgUsers := &orgUsersSpy{
		roleReturn: map[string][]int64{
			"admin": {201},
		},
	}
	svc := &NotificationService{
		Templates:   &tplRepoStub{tpl: NotificationTemplate{Subject: "Scan", Body: "Done"}},
		Preferences: &prefRepoNone{},
		Logs:        &logRepoNoop{},
		Inbox:       inbox,
		OrgUsers:    orgUsers,
		Email:       &emailNoop{},
		Slack:       &slackNoop{},
		Webhook:     &webhookNoop{},
		Renderer:    templates.Renderer{},
	}

	evt := NotificationEvent{
		EventType:      "project.scan.failed",
		OrganizationID: 1,
		Payload: map[string]interface{}{
			"target_role": "admin",
			"project":     "myproj",
		},
	}

	if err := svc.HandleEvent(context.Background(), evt); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if orgUsers.roleCalls != 1 {
		t.Fatalf("expected roleCalls=1 got %d", orgUsers.roleCalls)
	}
	if orgUsers.lastRole != "admin" {
		t.Fatalf("expected role admin got %s", orgUsers.lastRole)
	}
	if len(inbox.saved) != 1 || inbox.saved[0].UserID != 201 {
		t.Fatalf("expected inbox saved to user 201, got %#v", inbox.saved)
	}
}
