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
				if payload, ok := generic["payload"].(map[string]interface{}); ok {
					evt.Payload = payload
				}
			}
			evt = hydrateFromEnvelope(evt, generic)
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
	if payload, ok := generic["payload"].(map[string]interface{}); ok {
		evt.Payload = payload
	} else if dataMap, ok := generic["data"].(map[string]interface{}); ok {
		if pl, ok := dataMap["payload"].(map[string]interface{}); ok {
			evt.Payload = pl
		} else {
			evt.Payload = dataMap
		}
		if evt.Severity == "" {
			if sev, ok := dataMap["severity"]; ok {
				evt.Severity = fmt.Sprintf("%v", sev)
			}
		}
		if evt.TargetEmails == nil || len(evt.TargetEmails) == 0 {
			if rawEmails, ok := dataMap["emails"]; ok {
				evt.TargetEmails = toStringSlice(rawEmails)
			}
		}
		if evt.UserID == nil {
			if uid, ok := toInt64(dataMap["user_id"]); ok {
				evt.UserID = &uid
			}
		}
	} else {
		evt.Payload = generic
	}
	return evt, nil
}

func hydrateFromEnvelope(evt domain.NotificationEvent, generic map[string]interface{}) domain.NotificationEvent {
	dataField, ok := generic["data"].(map[string]interface{})
	if !ok {
		return evt
	}

	if evt.Severity == "" {
		if sev, ok := dataField["severity"]; ok {
			evt.Severity = fmt.Sprintf("%v", sev)
		}
	}
	if evt.Payload == nil {
		if payload, ok := dataField["payload"].(map[string]interface{}); ok {
			evt.Payload = payload
		} else {
			evt.Payload = dataField
		}
	}
	if evt.TargetEmails == nil || len(evt.TargetEmails) == 0 {
		if rawEmails, ok := dataField["emails"]; ok {
			evt.TargetEmails = toStringSlice(rawEmails)
		}
	}
	if evt.UserID == nil {
		if uid, ok := toInt64(dataField["user_id"]); ok {
			evt.UserID = &uid
		}
	}
	if evt.WebhookURL == "" {
		if webhook, ok := dataField["webhook_url"].(string); ok {
			evt.WebhookURL = webhook
		}
	}
	if evt.SlackWebhook == "" {
		if slack, ok := dataField["slack_webhook"].(string); ok {
			evt.SlackWebhook = slack
		}
	}
	return evt
}

func toStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprintf("%v", item))
		}
		return out
	case string:
		return []string{v}
	default:
		return nil
	}
}

func toInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i, true
		}
	case string:
		var parsed int64
		if _, err := fmt.Sscan(v, &parsed); err == nil {
			return parsed, true
		}
	}
	return 0, false
}
