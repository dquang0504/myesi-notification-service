package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"myesi-notification-service/internal/domain"

	"github.com/segmentio/kafka-go"
)

// StartConsumer begins consuming events and delegates to the notification service.
func StartConsumer(ctx context.Context, svc *domain.NotificationService, brokers, topic, group string) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  strings.Split(brokers, ","),
		GroupID:  group,
		Topic:    topic,
		MinBytes: 1e3,
		MaxBytes: 10e6,
	})

	log.Printf("[KAFKA] Consumer listening on topic=%s group=%s", topic, group)

	go func() {
		defer reader.Close()
		for {
			m, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[KAFKA] read error: %v", err)
				time.Sleep(time.Second)
				continue
			}

			evt, err := parseEvent(m.Value)
			if err != nil {
				log.Printf("[KAFKA] decode error: %v", err)
				continue
			}

			if evt.EventType == "" {
				log.Printf("[KAFKA] skipped message with missing event_type")
				continue
			}

			if err := svc.HandleEvent(ctx, evt); err != nil {
				log.Printf("[NOTIFY] handle event failed: %v", err)
			}
		}
	}()
}

func parseEvent(data []byte) (domain.NotificationEvent, error) {
	var evt domain.NotificationEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return parseLooseEvent(data)
	}

	// Attempt to backfill fields if the payload uses different keys.
	if evt.EventType == "" || evt.Payload == nil {
		generic := make(map[string]interface{})
		if err := json.Unmarshal(data, &generic); err == nil {
			if evt.EventType == "" {
				if v, ok := generic["event_type"]; ok {
					evt.EventType = fmt.Sprintf("%v", v)
				} else if v, ok := generic["type"]; ok {
					evt.EventType = fmt.Sprintf("%v", v)
				}
			}
			if evt.OrganizationID == 0 {
				if org, ok := generic["organization_id"]; ok {
					if f, ok := org.(float64); ok {
						evt.OrganizationID = int64(f)
					}
				}
			}
			if evt.Severity == "" {
				if sev, ok := generic["severity"]; ok {
					evt.Severity = fmt.Sprintf("%v", sev)
				}
			}
			if evt.Payload == nil {
				evt.Payload = generic
			}
		}
	}

	if evt.Payload == nil {
		evt.Payload = map[string]interface{}{}
	}

	return evt, nil
}

// parseLooseEvent attempts to extract minimal information from arbitrary payloads.
func parseLooseEvent(data []byte) (domain.NotificationEvent, error) {
	var evt domain.NotificationEvent
	var generic map[string]interface{}
	if err := json.Unmarshal(data, &generic); err != nil {
		return evt, err
	}

	if v, ok := generic["event_type"]; ok {
		evt.EventType = fmt.Sprintf("%v", v)
	} else if v, ok := generic["type"]; ok {
		evt.EventType = fmt.Sprintf("%v", v)
	}

	if org, ok := generic["organization_id"]; ok {
		if f, ok := org.(float64); ok {
			evt.OrganizationID = int64(f)
		}
	}
	if sev, ok := generic["severity"]; ok {
		evt.Severity = fmt.Sprintf("%v", sev)
	}
	evt.Payload = generic
	return evt, nil
}
