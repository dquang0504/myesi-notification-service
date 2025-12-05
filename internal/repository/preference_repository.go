package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"myesi-notification-service/internal/domain"
)

// PreferenceRepositoryPG persists preferences in PostgreSQL.
type PreferenceRepositoryPG struct {
	DB *sql.DB
}

func (r *PreferenceRepositoryPG) List(ctx context.Context, orgID int64, userID *int64, eventType string) ([]domain.NotificationPreference, error) {
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

	if userID != nil {
		conditions = append(conditions, fmt.Sprintf("(user_id IS NULL OR user_id=$%d)", len(args)+1))
		args = append(args, *userID)
	}

	query := fmt.Sprintf(`
        SELECT id, organization_id, user_id, event_type, channel, target, enabled, severity_min, created_at, updated_at
        FROM notification_preferences
        WHERE %s
        ORDER BY updated_at DESC
    `, strings.Join(conditions, " AND "))

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]domain.NotificationPreference, 0)
	for rows.Next() {
		var pref domain.NotificationPreference
		var user sql.NullInt64
		if err := rows.Scan(&pref.ID, &pref.OrganizationID, &user, &pref.EventType, &pref.Channel, &pref.Target, &pref.Enabled, &pref.SeverityMin, &pref.CreatedAt, &pref.UpdatedAt); err != nil {
			return nil, err
		}
		if user.Valid {
			val := user.Int64
			pref.UserID = &val
		}
		results = append(results, pref)
	}
	return results, nil
}

func (r *PreferenceRepositoryPG) Save(ctx context.Context, pref domain.NotificationPreference) (domain.NotificationPreference, error) {
	var userID interface{}
	if pref.UserID != nil {
		userID = pref.UserID
	} else {
		userID = nil
	}

	// When an ID is provided, update that record directly to honor the REST contract.
	if pref.ID > 0 {
		row := r.DB.QueryRowContext(ctx, `
            UPDATE notification_preferences
            SET organization_id=$2, user_id=$3, event_type=$4, channel=$5, target=$6, enabled=$7, severity_min=$8, updated_at=NOW()
            WHERE id=$1
            RETURNING id, organization_id, user_id, event_type, channel, target, enabled, severity_min, created_at, updated_at
        `, pref.ID, pref.OrganizationID, userID, pref.EventType, pref.Channel, pref.Target, pref.Enabled, pref.SeverityMin)

		var saved domain.NotificationPreference
		var user sql.NullInt64
		err := row.Scan(&saved.ID, &saved.OrganizationID, &user, &saved.EventType, &saved.Channel, &saved.Target, &saved.Enabled, &saved.SeverityMin, &saved.CreatedAt, &saved.UpdatedAt)
		if user.Valid {
			val := user.Int64
			saved.UserID = &val
		}
		return saved, err
	}

	row := r.DB.QueryRowContext(ctx, `
        INSERT INTO notification_preferences (organization_id, user_id, event_type, channel, target, enabled, severity_min)
        VALUES ($1,$2,$3,$4,$5,$6,$7)
        ON CONFLICT ON CONSTRAINT ux_notification_preferences_scope
        DO UPDATE SET target=EXCLUDED.target, enabled=EXCLUDED.enabled, severity_min=EXCLUDED.severity_min, updated_at=NOW()
        RETURNING id, organization_id, user_id, event_type, channel, target, enabled, severity_min, created_at, updated_at
    `, pref.OrganizationID, userID, pref.EventType, pref.Channel, pref.Target, pref.Enabled, pref.SeverityMin)

	var saved domain.NotificationPreference
	var user sql.NullInt64
	err := row.Scan(&saved.ID, &saved.OrganizationID, &user, &saved.EventType, &saved.Channel, &saved.Target, &saved.Enabled, &saved.SeverityMin, &saved.CreatedAt, &saved.UpdatedAt)
	if user.Valid {
		val := user.Int64
		saved.UserID = &val
	}
	return saved, err
}
