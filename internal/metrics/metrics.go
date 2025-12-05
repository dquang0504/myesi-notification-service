package metrics

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// Collector holds counters used by the service.
type Collector struct {
	sent     metric.Int64Counter
	duration metric.Float64Histogram
}

// Init initializes OpenTelemetry metrics with a basic SDK provider.
func Init() (*Collector, error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("notification-service"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("resource init failed: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res))
	otel.SetMeterProvider(mp)

	meter := otel.Meter("notification-service")

	sent, err := meter.Int64Counter("notifications_sent_total")
	if err != nil {
		return nil, fmt.Errorf("counter init failed: %w", err)
	}

	duration, err := meter.Float64Histogram("notification_send_duration_seconds")
	if err != nil {
		return nil, fmt.Errorf("histogram init failed: %w", err)
	}

	log.Println("[METRICS] Collector initialized")
	return &Collector{sent: sent, duration: duration}, nil
}

// ObserveSend records the result of a send attempt.
func (c *Collector) ObserveSend(ctx context.Context, channel, status string, took time.Duration) {
	if c == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("channel", channel),
		attribute.String("status", status),
	)
	c.sent.Add(ctx, 1, attrs)
	c.duration.Record(ctx, took.Seconds(), metric.WithAttributes(attribute.String("channel", channel)))
}
