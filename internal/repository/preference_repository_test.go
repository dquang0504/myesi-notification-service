package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"myesi-notification-service/internal/domain"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPreferenceRepositoryPG_List(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := &PreferenceRepositoryPG{DB: db}
	now := time.Now()
	user := sql.NullInt64{Int64: 9, Valid: true}

	mock.ExpectQuery("SELECT id, organization_id, user_id, event_type, channel, target, enabled, severity_min, created_at, updated_at").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "user_id", "event_type", "channel", "target", "enabled", "severity_min", "created_at", "updated_at",
		}).AddRow(int64(1), int64(2), user, "payment.success", "email", "a@b.com", true, "low", now, now))

	out, err := repo.List(context.Background(), 2, nil, "payment.success")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(out) != 1 || out[0].ID != 1 {
		t.Fatalf("unexpected out: %#v", out)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPreferenceRepositoryPG_Save_UpdateByID(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := &PreferenceRepositoryPG{DB: db}
	now := time.Now()
	user := sql.NullInt64{Int64: 9, Valid: true}

	mock.ExpectQuery("UPDATE notification_preferences").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "user_id", "event_type", "channel", "target", "enabled", "severity_min", "created_at", "updated_at",
		}).AddRow(int64(7), int64(1), user, "x", "email", "t", true, "low", now, now))

	uid := int64(9)
	out, err := repo.Save(context.Background(), domain.NotificationPreference{
		ID: 7, OrganizationID: 1, UserID: &uid, EventType: "x", Channel: "email", Target: "t", Enabled: true, SeverityMin: "low",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.ID != 7 || out.OrganizationID != 1 {
		t.Fatalf("unexpected out: %#v", out)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPreferenceRepositoryPG_Save_InsertOnConflict(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := &PreferenceRepositoryPG{DB: db}
	now := time.Now()
	user := sql.NullInt64{Valid: false}

	mock.ExpectQuery("INSERT INTO notification_preferences").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "user_id", "event_type", "channel", "target", "enabled", "severity_min", "created_at", "updated_at",
		}).AddRow(int64(10), int64(1), user, "x", "email", "t", true, "low", now, now))

	out, err := repo.Save(context.Background(), domain.NotificationPreference{
		OrganizationID: 1, UserID: nil, EventType: "x", Channel: "email", Target: "t", Enabled: true, SeverityMin: "low",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.ID != 10 {
		t.Fatalf("expected id 10 got %d", out.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
