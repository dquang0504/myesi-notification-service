package repository

import (
	"context"
	"testing"
	"time"

	"myesi-notification-service/internal/domain"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestTemplateRepositoryPG_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	repo := &TemplateRepositoryPG{DB: db}

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "name", "event_type", "channel", "subject", "body", "is_default", "created_at", "updated_at",
	}).AddRow(int64(1), "tpl", "payment.success", "email", "sub", "body", true, now, now)

	mock.ExpectQuery("SELECT id, name, event_type, channel, subject, body, is_default, created_at, updated_at").
		WithArgs(50, 0).
		WillReturnRows(rows)

	res, err := repo.List(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(res) != 1 || res[0].ID != 1 {
		t.Fatalf("unexpected res: %#v", res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTemplateRepositoryPG_Upsert(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := &TemplateRepositoryPG{DB: db}
	now := time.Now()

	mock.ExpectQuery("INSERT INTO notification_templates").
		WithArgs("n", "e", "email", "s", "b", true).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "event_type", "channel", "subject", "body", "is_default", "created_at", "updated_at",
		}).AddRow(int64(7), "n", "e", "email", "s", "b", true, now, now))

	out, err := repo.Upsert(context.Background(), domain.NotificationTemplate{
		Name: "n", EventType: "e", Channel: "email", Subject: "s", Body: "b", IsDefault: true,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.ID != 7 {
		t.Fatalf("expected id 7 got %d", out.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
