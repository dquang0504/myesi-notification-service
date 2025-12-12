package repository

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"myesi-notification-service/internal/domain"
)

// OrgSettingsRepositoryPG reads organization_settings table for notification toggles.
type OrgSettingsRepositoryPG struct {
	DB *sql.DB
}

type orgSettingsCacheEntry struct {
	value   *domain.OrgSettings
	expires time.Time
}

var (
	orgSettingsTTL   = 5 * time.Minute
	orgSettingsCache = map[int64]orgSettingsCacheEntry{}
	orgSettingsMu    sync.RWMutex
)

// Get fetches organization-level notification toggles, caching for a short time.
func (r *OrgSettingsRepositoryPG) Get(ctx context.Context, orgID int64) (*domain.OrgSettings, error) {
	if orgID == 0 || r == nil || r.DB == nil {
		return nil, nil
	}

	orgSettingsMu.RLock()
	if entry, ok := orgSettingsCache[orgID]; ok && time.Now().Before(entry.expires) {
		orgSettingsMu.RUnlock()
		return entry.value, nil
	}
	orgSettingsMu.RUnlock()

	query := `
        SELECT organization_id,
               COALESCE(email_notifications, TRUE),
               COALESCE(vulnerability_alerts, TRUE),
               COALESCE(weekly_reports, TRUE),
               COALESCE(user_activity_alerts, FALSE),
               COALESCE(admin_email, '')
        FROM organization_settings
        WHERE organization_id = $1
    `

	row := r.DB.QueryRowContext(ctx, query, orgID)
	settings := domain.OrgSettings{
		OrganizationID:      orgID,
		EmailNotifications:  true,
		VulnerabilityAlerts: true,
		WeeklyReports:       true,
		UserActivityAlerts:  false,
		AdminEmail:          "",
	}

	if err := row.Scan(
		&settings.OrganizationID,
		&settings.EmailNotifications,
		&settings.VulnerabilityAlerts,
		&settings.WeeklyReports,
		&settings.UserActivityAlerts,
		&settings.AdminEmail,
	); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("[ORG_SETTINGS] query failed: %v", err)
			return nil, err
		}
	}

	orgSettingsMu.Lock()
	orgSettingsCache[orgID] = orgSettingsCacheEntry{
		value:   &settings,
		expires: time.Now().Add(orgSettingsTTL),
	}
	orgSettingsMu.Unlock()
	return &settings, nil
}
