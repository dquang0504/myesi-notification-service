package repository

import (
	"context"
	"database/sql"
)

// OrgUserRepositoryPG resolves users for an organization for broadcast notifications.
type OrgUserRepositoryPG struct {
	DB *sql.DB
}

func (r *OrgUserRepositoryPG) ListUserIDsByOrg(ctx context.Context, orgID int64) ([]int64, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id FROM users WHERE organization_id=$1 AND is_active=true`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *OrgUserRepositoryPG) ListUserIDsByOrgWithRole(ctx context.Context, orgID int64, role string) ([]int64, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id FROM users WHERE organization_id=$1 AND is_active=true AND role=$2`, orgID, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
