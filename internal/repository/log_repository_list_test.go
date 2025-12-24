package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLogRepositoryPG_List_NoFilters_ArgsLimitOffset(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := &LogRepositoryPG{DB: db}
	now := time.Now()

	// No filters => only args: limit, offset
	mock.ExpectQuery("FROM notification_logs").
		WithArgs(50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "user_id", "event_type", "channel", "target", "status", "error", "payload", "created_at",
		}).AddRow(int64(1), int64(0), nil, "x", "email", "t", "success", "", []byte(`{}`), now))

	_, err := repo.List(context.Background(), 0, "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLogRepositoryPG_List_AllFilters_ArgsCount6(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := &LogRepositoryPG{DB: db}
	now := time.Now()

	// Filters: orgID,eventType,status,channel => args 4 + limit + offset = 6
	mock.ExpectQuery("FROM notification_logs").
		WithArgs(int64(7), "payment.success", "failed", "email", 10, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "user_id", "event_type", "channel", "target", "status", "error", "payload", "created_at",
		}).AddRow(int64(2), int64(7), nil, "payment.success", "email", "a@b.com", "failed", "boom", []byte(`{"k":"v"}`), now))

	_, err := repo.List(context.Background(), 7, "payment.success", "failed", "email", 10, 20)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLogRepositoryPG_List_SomeFilters_ArgsCountVaries(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := &LogRepositoryPG{DB: db}
	now := time.Now()

	// Only orgID + channel => args: orgID, channel, limit, offset = 4
	mock.ExpectQuery("FROM notification_logs").
		WithArgs(int64(3), "slack", 5, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "user_id", "event_type", "channel", "target", "status", "error", "payload", "created_at",
		}).AddRow(int64(3), int64(3), nil, "x", "slack", "hook", "success", "", []byte(`{}`), now))

	_, err := repo.List(context.Background(), 3, "", "", "slack", 5, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
