package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"myesi-notification-service/internal/domain"
)

// InboxRepositoryPG stores per-user notifications for the bell UI.
type InboxRepositoryPG struct {
	DB *sql.DB
}

func (r *InboxRepositoryPG) Save(ctx context.Context, n domain.UserNotification) (domain.UserNotification, error) {
	payloadJSON, _ := json.Marshal(n.Payload)

	row := r.DB.QueryRowContext(ctx, `
        INSERT INTO user_notifications (user_id, organization_id, title, message, type, severity, action_url, payload, read)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
        RETURNING id, created_at, read_at
    `, n.UserID, n.OrganizationID, n.Title, n.Message, n.Type, n.Severity, n.ActionURL, payloadJSON, n.Read)

	if err := row.Scan(&n.ID, &n.CreatedAt, &n.ReadAt); err != nil {
		return n, err
	}
	return n, nil
}

func (r *InboxRepositoryPG) List(ctx context.Context, userID int64, orgID int64, unreadOnly bool, limit, offset int) ([]domain.UserNotification, int, int, error) {
	if limit == 0 {
		limit = 50
	}

	rows, err := r.DB.QueryContext(ctx, `
        SELECT id, user_id, organization_id, title, message, type, severity, action_url, payload, read, created_at, read_at
        FROM user_notifications
        WHERE user_id=$1 AND ($2 = 0 OR organization_id=$2) AND ($3::bool = false OR read = false)
        ORDER BY created_at DESC
        LIMIT $4 OFFSET $5
    `, userID, orgID, unreadOnly, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	list := make([]domain.UserNotification, 0)
	for rows.Next() {
		var n domain.UserNotification
		var payload []byte
		var readAt sql.NullTime
		if err := rows.Scan(&n.ID, &n.UserID, &n.OrganizationID, &n.Title, &n.Message, &n.Type, &n.Severity, &n.ActionURL, &payload, &n.Read, &n.CreatedAt, &readAt); err != nil {
			return nil, 0, 0, err
		}
		if len(payload) > 0 {
			_ = json.Unmarshal(payload, &n.Payload)
		}
		if readAt.Valid {
			n.ReadAt = &readAt.Time
		}
		list = append(list, n)
	}

	// counts
	var total, unread int
	_ = r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_notifications WHERE user_id=$1 AND ($2=0 OR organization_id=$2)`, userID, orgID).Scan(&total)
	_ = r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_notifications WHERE user_id=$1 AND ($2=0 OR organization_id=$2) AND read=false`, userID, orgID).Scan(&unread)

	return list, total, unread, nil
}

func (r *InboxRepositoryPG) MarkRead(ctx context.Context, id int64, userID int64) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE user_notifications SET read=true, read_at=NOW() WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *InboxRepositoryPG) MarkAllRead(ctx context.Context, userID int64) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE user_notifications SET read=true, read_at=NOW() WHERE user_id=$1 AND read=false`, userID)
	return err
}

func (r *InboxRepositoryPG) Delete(ctx context.Context, id int64, userID int64) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM user_notifications WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}
