package repository

import (
	"context"
	"database/sql"

	"myesi-notification-service/internal/domain"
)

// TemplateRepositoryPG persists templates in PostgreSQL.
type TemplateRepositoryPG struct {
	DB *sql.DB
}

func (r *TemplateRepositoryPG) List(ctx context.Context, limit, offset int) ([]domain.NotificationTemplate, error) {
	if limit == 0 {
		limit = 50
	}
	rows, err := r.DB.QueryContext(ctx, `
        SELECT id, name, event_type, channel, subject, body, is_default, created_at, updated_at
        FROM notification_templates
        ORDER BY updated_at DESC
        LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := make([]domain.NotificationTemplate, 0)
	for rows.Next() {
		var t domain.NotificationTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.EventType, &t.Channel, &t.Subject, &t.Body, &t.IsDefault, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, nil
}

func (r *TemplateRepositoryPG) Upsert(ctx context.Context, tpl domain.NotificationTemplate) (domain.NotificationTemplate, error) {
	row := r.DB.QueryRowContext(ctx, `
        INSERT INTO notification_templates (name, event_type, channel, subject, body, is_default)
        VALUES ($1,$2,$3,$4,$5,$6)
        ON CONFLICT (event_type, channel)
        DO UPDATE SET name=EXCLUDED.name, subject=EXCLUDED.subject, body=EXCLUDED.body, is_default=EXCLUDED.is_default, updated_at=NOW()
        RETURNING id, name, event_type, channel, subject, body, is_default, created_at, updated_at
    `, tpl.Name, tpl.EventType, tpl.Channel, tpl.Subject, tpl.Body, tpl.IsDefault)

	var saved domain.NotificationTemplate
	err := row.Scan(&saved.ID, &saved.Name, &saved.EventType, &saved.Channel, &saved.Subject, &saved.Body, &saved.IsDefault, &saved.CreatedAt, &saved.UpdatedAt)
	return saved, err
}

func (r *TemplateRepositoryPG) FindByEventAndChannel(ctx context.Context, eventType, channel string) (*domain.NotificationTemplate, error) {
	row := r.DB.QueryRowContext(ctx, `
        SELECT id, name, event_type, channel, subject, body, is_default, created_at, updated_at
        FROM notification_templates
        WHERE event_type=$1 AND channel=$2
        ORDER BY is_default DESC, updated_at DESC
        LIMIT 1
    `, eventType, channel)

	var tpl domain.NotificationTemplate
	if err := row.Scan(&tpl.ID, &tpl.Name, &tpl.EventType, &tpl.Channel, &tpl.Subject, &tpl.Body, &tpl.IsDefault, &tpl.CreatedAt, &tpl.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &tpl, nil
}
