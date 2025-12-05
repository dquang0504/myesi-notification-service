package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"myesi-notification-service/internal/domain"
)

// LogRepositoryPG stores notification logs.
type LogRepositoryPG struct {
	DB *sql.DB
}

func (r *LogRepositoryPG) Insert(ctx context.Context, logEntry domain.NotificationLog) error {
	payloadJSON, _ := json.Marshal(logEntry.Payload)

	_, err := r.DB.ExecContext(ctx, `
        INSERT INTO notification_logs (organization_id, user_id, event_type, channel, target, status, error, payload)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
    `, logEntry.OrganizationID, logEntry.UserID, logEntry.EventType, logEntry.Channel, logEntry.Target, logEntry.Status, logEntry.Error, payloadJSON)
	return err
}

func (r *LogRepositoryPG) List(ctx context.Context, orgID int64, eventType, status, channel string, limit, offset int) ([]domain.NotificationLog, error) {
	conditions := []string{"1=1"}
	args := []interface{}{}

	if orgID > 0 {
		conditions = append(conditions, fmt.Sprintf("organization_id=$%d", len(args)+1))
		args = append(args, orgID)
	}
	if eventType != "" {
		conditions = append(conditions, fmt.Sprintf("event_type=$%d", len(args)+1))
		args = append(args, eventType)
	}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status=$%d", len(args)+1))
		args = append(args, status)
	}
	if channel != "" {
		conditions = append(conditions, fmt.Sprintf("channel=$%d", len(args)+1))
		args = append(args, channel)
	}

	if limit == 0 {
		limit = 50
	}

	query := fmt.Sprintf(`
        SELECT id, organization_id, user_id, event_type, channel, target, status, error, payload, created_at
        FROM notification_logs
        WHERE %s
        ORDER BY created_at DESC
        LIMIT $%d OFFSET $%d
    `, strings.Join(conditions, " AND "), len(args)+1, len(args)+2)

	args = append(args, limit, offset)

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]domain.NotificationLog, 0)
	for rows.Next() {
		var entry domain.NotificationLog
		var user sql.NullInt64
		var payload []byte
		if err := rows.Scan(&entry.ID, &entry.OrganizationID, &user, &entry.EventType, &entry.Channel, &entry.Target, &entry.Status, &entry.Error, &payload, &entry.CreatedAt); err != nil {
			return nil, err
		}
		if user.Valid {
			val := user.Int64
			entry.UserID = &val
		}
		_ = json.Unmarshal(payload, &entry.Payload)
		logs = append(logs, entry)
	}
	return logs, nil
}
