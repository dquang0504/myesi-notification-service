package main

import (
	"context"
	"log"
	"myesi-notification-service/internal/api"
	"myesi-notification-service/internal/config"
	"myesi-notification-service/internal/db"
	"myesi-notification-service/internal/domain"
	"myesi-notification-service/internal/kafka"
	"myesi-notification-service/internal/metrics"
	"myesi-notification-service/internal/providers"
	"myesi-notification-service/internal/repository"
	"myesi-notification-service/internal/templates"
	"os/signal"
	"syscall"

	fiber "github.com/gofiber/fiber/v2"
)

func main() {
	cfg := config.LoadConfig()

	db.InitPostgres(cfg.DatabaseURL)
	defer db.CloseDB()

	collector, err := metrics.Init()
	if err != nil {
		log.Printf("[METRICS] init failed: %v", err)
	}

	tplRepo := &repository.TemplateRepositoryPG{DB: db.Conn}
	prefRepo := &repository.PreferenceRepositoryPG{DB: db.Conn}
	logRepo := &repository.LogRepositoryPG{DB: db.Conn}
	inboxRepo := &repository.InboxRepositoryPG{DB: db.Conn}
	orgUserRepo := &repository.OrgUserRepositoryPG{DB: db.Conn}

	svc := &domain.NotificationService{
		Templates:   tplRepo,
		Preferences: prefRepo,
		Logs:        logRepo,
		Inbox:       inboxRepo,
		OrgUsers:    orgUserRepo,
		Email: providers.SMTPProvider{
			Host: cfg.SMTPHost,
			Port: cfg.SMTPPort,
			User: cfg.SMTPUser,
			Pass: cfg.SMTPPass,
			From: cfg.FromAddress,
		},
		Slack:    providers.SlackWebhookProvider{},
		Webhook:  providers.GenericWebhookProvider{},
		Renderer: templates.Renderer{},
		Metrics:  collector,
		Defaults: domain.Defaults{
			Emails:       cfg.DefaultEmails,
			SlackWebhook: cfg.SlackDefaultWebhook,
			WebhookURL:   cfg.WebhookDefaultTarget,
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	kafka.StartConsumer(ctx, svc, cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaConsumerGroup)

	app := fiber.New()
	api.RegisterRoutes(app, api.HandlerDeps{
		Templates:    tplRepo,
		Preferences:  prefRepo,
		Logs:         logRepo,
		Inbox:        inboxRepo,
		Svc:          svc,
		ServiceToken: cfg.ServiceToken,
	})

	go func() {
		addr := ":" + cfg.Port
		log.Printf("[HTTP] notification service listening on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("fiber listen failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("[EXIT] notification service shutting down")
}
