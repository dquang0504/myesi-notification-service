package config

import (
	"os"
	"testing"
)

func withEnv(t *testing.T, key, val string, fn func()) {
	t.Helper()
	old := os.Getenv(key)
	if val == "" {
		_ = os.Unsetenv(key)
	} else {
		_ = os.Setenv(key, val)
	}
	t.Cleanup(func() {
		if old == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, old)
		}
	})
	fn()
}

func TestSplitCSV(t *testing.T) {
	out := splitCSV(" a@b.com,  ,c@d.com  ")
	if len(out) != 2 || out[0] != "a@b.com" || out[1] != "c@d.com" {
		t.Fatalf("unexpected: %#v", out)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// clear relevant env to force defaults
	keys := []string{
		"NOTIFICATION_SERVICE_PORT", "KAFKA_BROKERS", "KAFKA_BROKER", "KAFKA_TOPIC", "KAFKA_CONSUMER_GROUP",
		"POSTGRES_DSN", "DATABASE_URL", "DEFAULT_ALERT_EMAILS",
		"SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASS", "FROM_ADDRESS",
		"SLACK_DEFAULT_WEBHOOK", "WEBHOOK_DEFAULT_TARGET", "NOTIFICATION_SERVICE_TOKEN",
	}
	for _, k := range keys {
		k := k
		withEnv(t, k, "", func() {})
	}

	cfg := LoadConfig()
	if cfg.Port != "8006" {
		t.Fatalf("expected default port 8006 got %s", cfg.Port)
	}
	if cfg.KafkaBrokers == "" {
		t.Fatalf("expected kafka brokers default")
	}
	if cfg.KafkaTopic != "notification-events" {
		t.Fatalf("expected default topic notification-events got %s", cfg.KafkaTopic)
	}
	if cfg.KafkaConsumerGroup != "myesi-notification-group" {
		t.Fatalf("expected default group myesi-notification-group got %s", cfg.KafkaConsumerGroup)
	}
	if cfg.SMTPPort != 587 {
		t.Fatalf("expected default smtp port 587 got %d", cfg.SMTPPort)
	}
}

func TestLoadConfig_PrefersKAFKA_BROKERSOverKAFKA_BROKER(t *testing.T) {
	withEnv(t, "KAFKA_BROKER", "kafka:1111", func() {
		withEnv(t, "KAFKA_BROKERS", "k1:9092,k2:9092", func() {
			cfg := LoadConfig()
			if cfg.KafkaBrokers != "k1:9092,k2:9092" {
				t.Fatalf("unexpected brokers: %s", cfg.KafkaBrokers)
			}
		})
	})
}
