package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"myesi-notification-service/internal/domain"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestInboxRepositoryPG_Save(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := &InboxRepositoryPG{DB: db}
	now := time.Now()

	mock.ExpectQuery("INSERT INTO user_notifications").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "read_at"}).
			AddRow(int64(5), now, nil))

	n, err := repo.Save(context.Background(), domain.UserNotification{
		UserID: 1, OrganizationID: 2, Title: "t", Message: "m", Type: "x", Severity: "low", ActionURL: "", Read: false,
		Payload: map[string]interface{}{"hello": "world"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if n.ID != 5 {
		t.Fatalf("expected id 5 got %d", n.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestInboxRepositoryPG_List_ScansPayloadAndCounts(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := &InboxRepositoryPG{DB: db}
	now := time.Now()
	payload, _ := json.Marshal(map[string]interface{}{"k": "v"})

	// list query
	mock.ExpectQuery("SELECT id, user_id, organization_id, title, message, type, severity, action_url, payload, read, created_at, read_at").
		WithArgs(int64(11), int64(0), false, 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "organization_id", "title", "message", "type", "severity", "action_url", "payload", "read", "created_at", "read_at",
		}).AddRow(int64(1), int64(11), int64(0), "t", "m", "x", "low", "", payload, false, now, nil))

	// total count
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM user_notifications").
		WithArgs(int64(11), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// unread count
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM user_notifications").
		WithArgs(int64(11), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	items, total, unread, err := repo.List(context.Background(), 11, 0, false, 0, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if total != 1 || unread != 1 || len(items) != 1 {
		t.Fatalf("unexpected results total=%d unread=%d items=%d", total, unread, len(items))
	}
	if items[0].Payload["k"] != "v" {
		t.Fatalf("expected payload k=v got %v", items[0].Payload)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
