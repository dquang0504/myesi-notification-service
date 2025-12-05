package domain

import (
	"context"
	"time"
)

const (
	ChannelEmail   = "email"
	ChannelSlack   = "slack"
	ChannelWebhook = "webhook"
)

// NotificationEvent represents an inbound domain event (usually from Kafka).
type NotificationEvent struct {
	EventType      string                 `json:"type"`
	OrganizationID int64                  `json:"organization_id"`
	UserID         *int64                 `json:"user_id,omitempty"`
	Severity       string                 `json:"severity,omitempty"`
	TargetEmails   []string               `json:"emails,omitempty"`
	SlackWebhook   string                 `json:"slack_webhook,omitempty"`
	WebhookURL     string                 `json:"webhook_url,omitempty"`
	Payload        map[string]interface{} `json:"payload,omitempty"`
	OccurredAt     time.Time              `json:"occurred_at"`
}

// NotificationTemplate is the rendering blueprint for outbound messages.
type NotificationTemplate struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	EventType string    `json:"event_type"`
	Channel   string    `json:"channel"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NotificationPreference allows orgs/users to customize routing.
type NotificationPreference struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	UserID         *int64    `json:"user_id,omitempty"`
	EventType      string    `json:"event_type"`
	Channel        string    `json:"channel"`
	Target         string    `json:"target"`
	Enabled        bool      `json:"enabled"`
	SeverityMin    string    `json:"severity_min"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NotificationLog captures delivery attempts for auditing.
type NotificationLog struct {
	ID             int64                  `json:"id"`
	OrganizationID int64                  `json:"organization_id"`
	UserID         *int64                 `json:"user_id,omitempty"`
	EventType      string                 `json:"event_type"`
	Channel        string                 `json:"channel"`
	Target         string                 `json:"target"`
	Status         string                 `json:"status"`
	Error          string                 `json:"error,omitempty"`
	Payload        map[string]interface{} `json:"payload,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// DeliveryTarget is a resolved destination for an event.
type DeliveryTarget struct {
	Channel string
	Target  string
}

// UserNotification represents a notification stored for in-app bell.
type UserNotification struct {
	ID             int64                  `json:"id"`
	UserID         int64                  `json:"user_id"`
	OrganizationID int64                  `json:"organization_id"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Type           string                 `json:"type"`
	Severity       string                 `json:"severity"`
	ActionURL      string                 `json:"action_url,omitempty"`
	Read           bool                   `json:"read"`
	Payload        map[string]interface{} `json:"payload,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	ReadAt         *time.Time             `json:"read_at,omitempty"`
}

// TemplateRepository abstracts persistence for templates.
type TemplateRepository interface {
	List(ctx Context, limit, offset int) ([]NotificationTemplate, error)
	Upsert(ctx Context, tpl NotificationTemplate) (NotificationTemplate, error)
	FindByEventAndChannel(ctx Context, eventType, channel string) (*NotificationTemplate, error)
}

// PreferenceRepository abstracts persistence for preferences.
type PreferenceRepository interface {
	List(ctx Context, orgID int64, userID *int64, eventType string) ([]NotificationPreference, error)
	Save(ctx Context, pref NotificationPreference) (NotificationPreference, error)
}

// LogRepository abstracts auditing persistence.
type LogRepository interface {
	Insert(ctx Context, log NotificationLog) error
	List(ctx Context, orgID int64, eventType, status, channel string, limit, offset int) ([]NotificationLog, error)
}

// InboxRepository stores per-user in-app notifications.
type InboxRepository interface {
	Save(ctx Context, n UserNotification) (UserNotification, error)
	List(ctx Context, userID int64, orgID int64, unreadOnly bool, limit, offset int) ([]UserNotification, int, int, error)
	MarkRead(ctx Context, id int64, userID int64) error
	MarkAllRead(ctx Context, userID int64) error
	Delete(ctx Context, id int64, userID int64) error
}

// OrgUserRepository resolves user IDs belonging to an organization.
type OrgUserRepository interface {
	ListUserIDsByOrg(ctx Context, orgID int64) ([]int64, error)
	ListUserIDsByOrgWithRole(ctx Context, orgID int64, role string) ([]int64, error)
}

// EmailProvider dispatches email notifications.
type EmailProvider interface {
	SendEmail(ctx Context, to []string, subject string, body string) error
}

// SlackProvider dispatches Slack messages via webhook.
type SlackProvider interface {
	SendSlackMessage(ctx Context, webhookURL string, text string) error
}

// WebhookProvider dispatches generic webhooks.
type WebhookProvider interface {
	SendWebhook(ctx Context, url string, payload any) error
}

// Context is aliased to context.Context for convenience while keeping the domain package decoupled.
type Context = context.Context
