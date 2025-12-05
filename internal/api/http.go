package api

import (
	"strconv"
	"time"

	"myesi-notification-service/internal/domain"

	fiber "github.com/gofiber/fiber/v2"
)

// RegisterRoutes wires HTTP endpoints.
func RegisterRoutes(app *fiber.App, deps HandlerDeps) {
	// TODO: plug in auth middleware consistent with other services (e.g., JWT or gateway headers).
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := app.Group("/api/notification")

	api.Get("/templates", deps.listTemplates)
	api.Post("/templates", deps.upsertTemplate)

	api.Get("/preferences", deps.listPreferences)
	api.Put("/preferences/:id", deps.updatePreference)

	api.Get("/logs", deps.listLogs)

	// In-app inbox endpoints
	api.Get("/inbox", deps.listInbox)
	api.Patch("/inbox/:id/read", deps.markInboxRead)
	api.Patch("/inbox/read-all", deps.markAllInboxRead)
	api.Delete("/inbox/:id", deps.deleteInbox)

	// Internal event ingress (used by services like billing)
	api.Post("/events", deps.ingestEvent)
}

// HandlerDeps groups dependencies for handlers.
type HandlerDeps struct {
	Templates    domain.TemplateRepository
	Preferences  domain.PreferenceRepository
	Logs         domain.LogRepository
	Inbox        domain.InboxRepository
	ServiceToken string
	Svc          *domain.NotificationService
}

func (h HandlerDeps) listTemplates(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	res, err := h.Templates.List(c.Context(), limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

func (h HandlerDeps) upsertTemplate(c *fiber.Ctx) error {
	var body domain.NotificationTemplate
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	saved, err := h.Templates.Upsert(c.Context(), body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(saved)
}

func (h HandlerDeps) listPreferences(c *fiber.Ctx) error {
	orgID, _ := strconv.ParseInt(c.Query("organization_id", "0"), 10, 64)
	eventType := c.Query("event_type")
	var userID *int64
	if v := c.Query("user_id"); v != "" {
		id, _ := strconv.ParseInt(v, 10, 64)
		userID = &id
	}

	prefs, err := h.Preferences.List(c.Context(), orgID, userID, eventType)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(prefs)
}

func (h HandlerDeps) updatePreference(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var body domain.NotificationPreference
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body.ID = id

	saved, err := h.Preferences.Save(c.Context(), body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(saved)
}

func (h HandlerDeps) listLogs(c *fiber.Ctx) error {
	orgID, _ := strconv.ParseInt(c.Query("organization_id", "0"), 10, 64)
	eventType := c.Query("event_type")
	status := c.Query("status")
	channel := c.Query("channel")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	logs, err := h.Logs.List(c.Context(), orgID, eventType, status, channel, limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(logs)
}

// ===== In-app inbox handlers =====
func (h HandlerDeps) listInbox(c *fiber.Ctx) error {
	if h.Inbox == nil {
		return c.Status(501).JSON(fiber.Map{"error": "inbox not enabled"})
	}

	userID := extractUserID(c)
	if userID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "missing user id"})
	}
	orgID, _ := strconv.ParseInt(c.Query("organization_id", "0"), 10, 64)
	unreadOnly := c.Query("unread_only") == "true" || c.Query("unreadOnly") == "true"
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	items, total, unread, err := h.Inbox.List(c.Context(), userID, orgID, unreadOnly, limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"notifications": items,
			"total":         total,
			"unread_count":  unread,
		},
	})
}

func (h HandlerDeps) markInboxRead(c *fiber.Ctx) error {
	if h.Inbox == nil {
		return c.Status(501).JSON(fiber.Map{"error": "inbox not enabled"})
	}
	userID := extractUserID(c)
	if userID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "missing user id"})
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.Inbox.MarkRead(c.Context(), id, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "ok"})
}

func (h HandlerDeps) markAllInboxRead(c *fiber.Ctx) error {
	if h.Inbox == nil {
		return c.Status(501).JSON(fiber.Map{"error": "inbox not enabled"})
	}
	userID := extractUserID(c)
	if userID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "missing user id"})
	}
	if err := h.Inbox.MarkAllRead(c.Context(), userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "ok"})
}

func (h HandlerDeps) deleteInbox(c *fiber.Ctx) error {
	if h.Inbox == nil {
		return c.Status(501).JSON(fiber.Map{"error": "inbox not enabled"})
	}
	userID := extractUserID(c)
	if userID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "missing user id"})
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.Inbox.Delete(c.Context(), id, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "ok"})
}

// ingestEvent allows trusted services to push events directly without Kafka.
func (h HandlerDeps) ingestEvent(c *fiber.Ctx) error {
	if h.Svc == nil {
		return c.Status(503).JSON(fiber.Map{"error": "notifier unavailable"})
	}
	token := c.Get("X-Service-Token")
	if h.ServiceToken != "" && token != h.ServiceToken {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	var evt domain.NotificationEvent
	if err := c.BodyParser(&evt); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now().UTC()
	}
	if err := h.Svc.HandleEvent(c.Context(), evt); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(202).JSON(fiber.Map{"status": "accepted"})
}

func extractUserID(c *fiber.Ctx) int64 {
	if v := c.Get("X-User-Id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			return id
		}
	}
	if v := c.Query("user_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			return id
		}
	}
	return 0
}
