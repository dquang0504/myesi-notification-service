package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration sourced from env vars.
type Config struct {
	Port                 string
	KafkaBrokers         string
	KafkaTopic           string
	KafkaConsumerGroup   string
	DatabaseURL          string
	DefaultEmails        []string
	SMTPHost             string
	SMTPPort             int
	SMTPUser             string
	SMTPPass             string
	FromAddress          string
	SlackDefaultWebhook  string
	WebhookDefaultTarget string
	ServiceToken         string
}

// LoadConfig reads configuration from environment variables with sane defaults.
func LoadConfig() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		Port:         getEnv("NOTIFICATION_SERVICE_PORT", "8006"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", getEnv("KAFKA_BROKER", "kafka:9092")),
		// TODO: align topic with actual event bus once upstream naming is finalized.
		KafkaTopic:           getEnv("KAFKA_TOPIC", "notification-events"),
		KafkaConsumerGroup:   getEnv("KAFKA_CONSUMER_GROUP", "myesi-notification-group"),
		DatabaseURL:          getEnv("POSTGRES_DSN", getEnv("DATABASE_URL", "postgres://myesi:123456789@postgres:5432/myesi_db?sslmode=disable")),
		DefaultEmails:        splitCSV(getEnv("DEFAULT_ALERT_EMAILS", "")),
		SMTPHost:             getEnv("SMTP_HOST", ""),
		SMTPPort:             getEnvInt("SMTP_PORT", 587),
		SMTPUser:             getEnv("SMTP_USER", ""),
		SMTPPass:             getEnv("SMTP_PASS", ""),
		FromAddress:          getEnv("FROM_ADDRESS", "alerts@myesi.local"),
		SlackDefaultWebhook:  getEnv("SLACK_DEFAULT_WEBHOOK", ""),
		WebhookDefaultTarget: getEnv("WEBHOOK_DEFAULT_TARGET", ""),
		ServiceToken:         getEnv("NOTIFICATION_SERVICE_TOKEN", ""),
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL/POSTGRES_DSN missing")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return fallback
}

func splitCSV(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}
