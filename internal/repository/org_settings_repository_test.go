package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func resetOrgSettingsCacheForTest() {
	orgSettingsMu.Lock()
	defer orgSettingsMu.Unlock()
	orgSettingsCache = map[int64]orgSettingsCacheEntry{}
	orgSettingsTTL = 5 * time.Minute
}

func TestOrgSettingsRepositoryPG_Get_Caches(t *testing.T) {
	resetOrgSettingsCacheForTest()

	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := &OrgSettingsRepositoryPG{DB: db}

	mock.ExpectQuery("SELECT organization_id").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"organization_id", "email_notifications", "vulnerability_alerts", "weekly_reports", "user_activity_alerts", "admin_email",
		}).AddRow(int64(1), true, true, true, false, "admin@x.com"))

	// first call hits db
	st1, err := repo.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if st1 == nil || st1.AdminEmail != "admin@x.com" {
		t.Fatalf("unexpected st1: %#v", st1)
	}

	// second call should be cached => no new expectation, should not query DB
	st2, err := repo.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if st2 == nil || st2.AdminEmail != "admin@x.com" {
		t.Fatalf("unexpected st2: %#v", st2)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
