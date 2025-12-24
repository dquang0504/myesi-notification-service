package domain

import (
	"context"
	"testing"

	"myesi-notification-service/internal/templates"
)

type stubOrgSettings struct {
	st  *OrgSettings
	err error
}

func (s *stubOrgSettings) Get(ctx Context, orgID int64) (*OrgSettings, error) { return s.st, s.err }

type stubOrgUsers struct {
	all    []int64
	byRole map[string][]int64
}

func (s *stubOrgUsers) ListUserIDsByOrg(ctx Context, orgID int64) ([]int64, error) { return s.all, nil }
func (s *stubOrgUsers) ListUserIDsByOrgWithRole(ctx Context, orgID int64, role string) ([]int64, error) {
	if s.byRole == nil {
		return nil, nil
	}
	return s.byRole[role], nil
}

type stubInboxRepo struct{ saved []UserNotification }

func (s *stubInboxRepo) Save(ctx Context, n UserNotification) (UserNotification, error) {
	s.saved = append(s.saved, n)
	return n, nil
}
func (s *stubInboxRepo) List(ctx Context, userID int64, orgID int64, unreadOnly bool, limit, offset int) ([]UserNotification, int, int, error) {
	return nil, 0, 0, nil
}
func (s *stubInboxRepo) MarkRead(ctx Context, id int64, userID int64) error { return nil }
func (s *stubInboxRepo) MarkAllRead(ctx Context, userID int64) error        { return nil }
func (s *stubInboxRepo) Delete(ctx Context, id int64, userID int64) error   { return nil }

type stubTemplateRepoAlways struct{ tpl NotificationTemplate }

func (r *stubTemplateRepoAlways) List(ctx Context, limit, offset int) ([]NotificationTemplate, error) {
	return []NotificationTemplate{r.tpl}, nil
}
func (r *stubTemplateRepoAlways) Upsert(ctx Context, tpl NotificationTemplate) (NotificationTemplate, error) {
	r.tpl = tpl
	return tpl, nil
}
func (r *stubTemplateRepoAlways) FindByEventAndChannel(ctx Context, eventType, channel string) (*NotificationTemplate, error) {
	return &r.tpl, nil
}

type stubPrefRepoStatic struct{ prefs []NotificationPreference }

func (r *stubPrefRepoStatic) List(ctx Context, orgID int64, userID *int64, eventType string) ([]NotificationPreference, error) {
	return r.prefs, nil
}
func (r *stubPrefRepoStatic) Save(ctx Context, pref NotificationPreference) (NotificationPreference, error) {
	r.prefs = append(r.prefs, pref)
	return pref, nil
}

func (r *stubLogRepo) Insert(ctx Context, log NotificationLog) error {
	r.entries = append(r.entries, log)
	return nil
}
func (r *stubLogRepo) List(ctx Context, orgID int64, eventType, status, channel string, limit, offset int) ([]NotificationLog, error) {
	return r.entries, nil
}

func TestHandleEvent_RespectsOrgSettings_DisableVulnAlerts(t *testing.T) {
	settings := &OrgSettings{OrganizationID: 1, VulnerabilityAlerts: false, EmailNotifications: true, WeeklyReports: true, UserActivityAlerts: true}
	svc := &NotificationService{
		Templates:   &stubTemplateRepoAlways{tpl: NotificationTemplate{Subject: "s", Body: "b"}},
		Preferences: &stubPrefRepoStatic{prefs: []NotificationPreference{{OrganizationID: 1, EventType: "vulnerability.critical", Channel: ChannelEmail, Target: "a@b.com", Enabled: true}}},
		Logs:        &stubLogRepo{},
		OrgSettings: &stubOrgSettings{st: settings},
		Email:       &stubEmail{},
		Slack:       &stubSlack{},
		Webhook:     &stubWebhook{},
		Renderer:    templates.Renderer{},
	}

	evt := NotificationEvent{EventType: "vulnerability.critical", OrganizationID: 1, Severity: "critical", Payload: map[string]interface{}{}}
	if err := svc.HandleEvent(context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// disabled => should skip entirely (no sends, no logs)
	if len(svc.Logs.(*stubLogRepo).entries) != 0 {
		t.Fatalf("expected no logs when event disabled")
	}
}

func TestResolveTargets_FallbackSuppressedWhenUserSpecified(t *testing.T) {
	uid := int64(9)
	svc := &NotificationService{
		Templates:   &stubTemplateRepoAlways{tpl: NotificationTemplate{Subject: "s", Body: "b"}},
		Preferences: &stubPrefRepoStatic{prefs: nil}, // none
		Logs:        &stubLogRepo{},
		Email:       &stubEmail{},
		Slack:       &stubSlack{},
		Webhook:     &stubWebhook{},
		Renderer:    templates.Renderer{},
		Defaults:    Defaults{Emails: []string{"default@x.com"}, SlackWebhook: "slack", WebhookURL: "wh"},
	}

	evt := NotificationEvent{EventType: "payment.success", OrganizationID: 1, UserID: &uid}
	targets := svc.resolveTargets(context.Background(), evt, &OrgSettings{EmailNotifications: true})
	if len(targets) != 0 {
		t.Fatalf("expected no fallback targets for user-specific event, got %v", targets)
	}
}

func TestInbox_BroadcastPaymentToAllOrgUsers(t *testing.T) {
	inbox := &stubInboxRepo{}
	orgUsers := &stubOrgUsers{all: []int64{1, 2, 3}}
	prefs := &stubPrefRepoStatic{prefs: []NotificationPreference{{OrganizationID: 1, EventType: "payment.success", Channel: ChannelSlack, Target: "slack", Enabled: true}}}

	svc := &NotificationService{
		Templates:   &stubTemplateRepoAlways{tpl: NotificationTemplate{Subject: "Payment", Body: "Body"}},
		Preferences: prefs,
		Logs:        &stubLogRepo{},
		Inbox:       inbox,
		OrgUsers:    orgUsers,
		Email:       &stubEmail{},
		Slack:       &stubSlack{},
		Webhook:     &stubWebhook{},
		Renderer:    templates.Renderer{},
	}

	evt := NotificationEvent{EventType: "payment.success", OrganizationID: 1, Payload: map[string]interface{}{}}
	if err := svc.HandleEvent(context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inbox.saved) != 3 {
		t.Fatalf("expected 3 inbox saves, got %d", len(inbox.saved))
	}
}
