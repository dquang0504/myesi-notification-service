package domain

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"myesi-notification-service/internal/metrics"
	"myesi-notification-service/internal/templates"
)

// Defaults holds fallback destinations when preferences are absent.
type Defaults struct {
	Emails       []string
	SlackWebhook string
	WebhookURL   string
}

// NotificationService orchestrates routing, rendering, and delivery.
type NotificationService struct {
	Templates   TemplateRepository
	Preferences PreferenceRepository
	Logs        LogRepository
	Inbox       InboxRepository
	OrgUsers    OrgUserRepository
	Email       EmailProvider
	Slack       SlackProvider
	Webhook     WebhookProvider
	Renderer    templates.Renderer
	Metrics     *metrics.Collector
	Defaults    Defaults
}

// HandleEvent processes a single domain event and dispatches notifications.
func (s *NotificationService) HandleEvent(ctx context.Context, evt NotificationEvent) error {
	if evt.EventType == "" {
		return nil
	}
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now().UTC()
	}

	data := buildTemplateData(evt)

	baseTpl := s.resolveTemplate(ctx, evt.EventType, "")
	baseSubject, baseBody := s.renderTemplate(baseTpl, data)

	// Store in-app inbox for targeted user, independent of outbound channels.
	if s.Inbox != nil && evt.UserID != nil && *evt.UserID != 0 {
		actionURL, _ := evt.Payload["action_url"].(string)
		_, _ = s.Inbox.Save(ctx, UserNotification{
			UserID:         *evt.UserID,
			OrganizationID: evt.OrganizationID,
			Title:          baseSubject,
			Message:        baseBody,
			Type:           evt.EventType,
			Severity:       evt.Severity,
			ActionURL:      actionURL,
			Read:           false,
			Payload:        evt.Payload,
			CreatedAt:      time.Now().UTC(),
		})
	} else if s.Inbox != nil && s.OrgUsers != nil && evt.OrganizationID != 0 {
		// Org-wide notifications for payment/scan/vuln summaries when no explicit user.
		targetRole := ""
		if r, ok := evt.Payload["target_role"].(string); ok {
			targetRole = r
		}
		var userIDs []int64
		var err error
		switch {
		case strings.HasPrefix(evt.EventType, "payment."):
			userIDs, err = s.OrgUsers.ListUserIDsByOrg(ctx, evt.OrganizationID)
		case strings.HasPrefix(evt.EventType, "project.scan.") || strings.HasPrefix(evt.EventType, "sbom.scan."):
			// Default to developers unless target_role overrides.
			role := targetRole
			if role == "" {
				role = "developer"
			}
			userIDs, err = s.OrgUsers.ListUserIDsByOrgWithRole(ctx, evt.OrganizationID, role)
		default:
			// Fallback broadcast to all if no specific handler.
			userIDs, err = s.OrgUsers.ListUserIDsByOrg(ctx, evt.OrganizationID)
		}
		if err != nil {
			log.Printf("[NOTIFY] cannot load org users for event %s: %v", evt.EventType, err)
		}
		actionURL, _ := evt.Payload["action_url"].(string)
		for _, uid := range userIDs {
			_, _ = s.Inbox.Save(ctx, UserNotification{
				UserID:         uid,
				OrganizationID: evt.OrganizationID,
				Title:          baseSubject,
				Message:        baseBody,
				Type:           evt.EventType,
				Severity:       evt.Severity,
				ActionURL:      actionURL,
				Read:           false,
				Payload:        evt.Payload,
				CreatedAt:      time.Now().UTC(),
			})
		}
	}

	targets := s.resolveTargets(ctx, evt)
	if len(targets) == 0 {
		log.Printf("[NOTIFY] No targets resolved for event %s", evt.EventType)
		return nil
	}

	for _, target := range targets {
		tpl := s.resolveTemplate(ctx, evt.EventType, target.Channel)
		subject, body := s.renderTemplate(tpl, data)

		start := time.Now()
		status := "success"
		var sendErr error

		switch target.Channel {
		case ChannelEmail:
			recipients := strings.Split(target.Target, ",")
			sendErr = s.Email.SendEmail(ctx, filterNonEmpty(recipients), subject, body)
		case ChannelSlack:
			sendErr = s.Slack.SendSlackMessage(ctx, target.Target, body)
		case ChannelWebhook:
			payload := map[string]interface{}{
				"event":            evt,
				"rendered_subject": subject,
				"rendered_body":    body,
			}
			sendErr = s.Webhook.SendWebhook(ctx, target.Target, payload)
		default:
			continue
		}

		if sendErr != nil {
			status = "failed"
			log.Printf("[NOTIFY][%s] send failed: %v", target.Channel, sendErr)
		} else {
			log.Printf("[NOTIFY][%s] dispatched to %s", target.Channel, target.Target)
		}

		if s.Metrics != nil {
			s.Metrics.ObserveSend(ctx, target.Channel, status, time.Since(start))
		}

		_ = s.logAttempt(ctx, evt, target, status, sendErr)
	}

	return nil
}

func (s *NotificationService) resolveTargets(ctx context.Context, evt NotificationEvent) []DeliveryTarget {
	prefs, err := s.Preferences.List(ctx, evt.OrganizationID, evt.UserID, evt.EventType)
	if err != nil {
		log.Printf("[NOTIFY] preference lookup failed: %v", err)
	}

	resolved := make([]DeliveryTarget, 0)
	for _, pref := range prefs {
		if !pref.Enabled {
			continue
		}
		if !shouldSendForSeverity(pref.SeverityMin, evt.Severity) {
			continue
		}
		resolved = append(resolved, DeliveryTarget{Channel: pref.Channel, Target: pref.Target})
	}

	// Fallback: use event-provided targets or defaults if no preferences.
	// If a specific user is provided, avoid broadcasting to org defaults.
	if len(resolved) == 0 {
		if evt.UserID != nil && *evt.UserID != 0 {
			return resolved
		}

		if len(evt.TargetEmails) > 0 {
			resolved = append(resolved, DeliveryTarget{Channel: ChannelEmail, Target: strings.Join(evt.TargetEmails, ",")})
		} else if len(s.Defaults.Emails) > 0 {
			resolved = append(resolved, DeliveryTarget{Channel: ChannelEmail, Target: strings.Join(s.Defaults.Emails, ",")})
		}

		slackURL := evt.SlackWebhook
		if slackURL == "" {
			slackURL = s.Defaults.SlackWebhook
		}
		if slackURL != "" {
			resolved = append(resolved, DeliveryTarget{Channel: ChannelSlack, Target: slackURL})
		}

		webhookURL := evt.WebhookURL
		if webhookURL == "" {
			webhookURL = s.Defaults.WebhookURL
		}
		if webhookURL != "" {
			resolved = append(resolved, DeliveryTarget{Channel: ChannelWebhook, Target: webhookURL})
		}
	}

	return resolved
}

func (s *NotificationService) resolveTemplate(ctx context.Context, eventType, channel string) NotificationTemplate {
	// Opinionated defaults for common events to keep messages user-friendly without org IDs.
	switch eventType {
	case "vulnerability.assignment":
		return NotificationTemplate{
			EventType: eventType,
			Channel:   channel,
			Subject:   "New vulnerability assigned to you",
			Body:      "A vulnerability task for project {{.payload.project}} has been assigned to you. Priority: {{.event.severity}}.",
		}
	case "code_finding.assignment":
		return NotificationTemplate{
			EventType: eventType,
			Channel:   channel,
			Subject:   "New code finding assigned to you",
			Body:      "A code finding in project {{.payload.project}} has been assigned to you. Priority: {{.event.severity}}.",
		}
	case "project.scan.completed":
		return NotificationTemplate{
			EventType: eventType,
			Channel:   channel,
			Subject:   "Project scan completed",
			Body:      "Scan finished for {{.payload.project}}. Findings: {{.payload.vulns}} vulns, {{.payload.code_findings}} code findings.",
		}
	case "project.scan.failed":
		return NotificationTemplate{
			EventType: eventType,
			Channel:   channel,
			Subject:   "Project scan failed",
			Body:      "Scan failed for {{.payload.project}}. Error: {{.payload.error}}",
		}
	case "sbom.scan.completed":
		return NotificationTemplate{
			EventType: eventType,
			Channel:   channel,
			Subject:   "SBOM scan completed",
			Body:      "Manual SBOM scan finished for {{.payload.project}}. Findings: {{.payload.vulns}} vulns, {{.payload.code_findings}} code findings.",
		}
	case "sbom.scan.failed":
		return NotificationTemplate{
			EventType: eventType,
			Channel:   channel,
			Subject:   "SBOM scan failed",
			Body:      "Manual SBOM scan failed for {{.payload.project}}. Error: {{.payload.error}}",
		}
	case "project.scan.summary":
		return NotificationTemplate{
			EventType: eventType,
			Channel:   channel,
			Subject:   "Project scan summary",
			Body:      "{{.payload.project}} scan complete: {{.payload.vulns}} vulns, {{.payload.code_findings}} code findings.",
		}
	case "sbom.scan.summary":
		return NotificationTemplate{
			EventType: eventType,
			Channel:   channel,
			Subject:   "SBOM scan summary",
			Body:      "SBOM uploaded for {{.payload.project}}: {{.payload.components}} components, {{.payload.vulns}} vulns found.",
		}
	case "payment.success":
		return NotificationTemplate{
			EventType: eventType,
			Channel:   channel,
			Subject:   "Payment received",
			Body:      "Your payment for {{.payload.plan_name}} succeeded. Amount: {{.payload.amount}}. Thank you!",
		}
	case "payment.failed":
		return NotificationTemplate{
			EventType: eventType,
			Channel:   channel,
			Subject:   "Payment failed",
			Body:      "A payment attempt for {{.payload.plan_name}} failed. Please update billing details.",
		}
	}

	tpl, err := s.Templates.FindByEventAndChannel(ctx, eventType, channel)
	if err != nil {
		log.Printf("[NOTIFY] template lookup failed: %v", err)
	}
	if tpl != nil {
		return *tpl
	}

	// Default fallback
	return NotificationTemplate{
		Subject: "MyESI update",
		Body:    "You have a new update: {{.event.type}}.",
		Channel: channel,
	}
}

func (s *NotificationService) renderTemplate(tpl NotificationTemplate, data map[string]interface{}) (string, string) {
	subj, err := s.Renderer.Render(tpl.Subject, data)
	if err != nil {
		log.Printf("[NOTIFY] render subject failed, using fallback: %v", err)
		subj = tpl.Subject
	}
	body, err := s.Renderer.Render(tpl.Body, data)
	if err != nil {
		log.Printf("[NOTIFY] render body failed, using fallback: %v", err)
		body = tpl.Body
	}
	return subj, body
}

func (s *NotificationService) logAttempt(ctx context.Context, evt NotificationEvent, target DeliveryTarget, status string, sendErr error) error {
	payloadCopy := make(map[string]interface{}, len(evt.Payload)+2)
	for k, v := range evt.Payload {
		payloadCopy[k] = v
	}
	payloadCopy["event_type"] = evt.EventType
	payloadCopy["occurred_at"] = evt.OccurredAt

	var errMsg string
	if sendErr != nil {
		errMsg = sendErr.Error()
	}

	logEntry := NotificationLog{
		OrganizationID: evt.OrganizationID,
		UserID:         evt.UserID,
		EventType:      evt.EventType,
		Channel:        target.Channel,
		Target:         target.Target,
		Status:         status,
		Error:          errMsg,
		Payload:        payloadCopy,
		CreatedAt:      time.Now().UTC(),
	}
	return s.Logs.Insert(ctx, logEntry)
}

func buildTemplateData(evt NotificationEvent) map[string]interface{} {
	data := map[string]interface{}{
		"event": map[string]interface{}{
			"type":            evt.EventType,
			"organization_id": evt.OrganizationID,
			"user_id":         evt.UserID,
			"severity":        evt.Severity,
			"occurred_at":     evt.OccurredAt,
		},
		"payload": evt.Payload,
	}
	return data
}

func shouldSendForSeverity(min, actual string) bool {
	if min == "" || actual == "" {
		return true
	}
	order := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
	}
	minRank := order[strings.ToLower(min)]
	actualRank := order[strings.ToLower(actual)]
	if minRank == 0 || actualRank == 0 {
		return true
	}
	return actualRank >= minRank
}

func filterNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// JSONMarshal is a helper mainly for tests to ensure payloads serialize.
func JSONMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
