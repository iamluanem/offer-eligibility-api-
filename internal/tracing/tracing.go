package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds tracing configuration.
type Config struct {
	Enabled     bool
	Endpoint    string // Jaeger endpoint (e.g., "http://localhost:14268/api/traces")
	ServiceName string
	Environment string
}

// Tracer wraps OpenTelemetry tracer functionality.
type Tracer struct {
	tracer trace.Tracer
}

var globalTracer *Tracer

// InitTracing initializes OpenTelemetry tracing.
func InitTracing(cfg Config) (*Tracer, error) {
	if !cfg.Enabled {
		// Return a no-op tracer
		globalTracer = &Tracer{
			tracer: trace.NewNoopTracerProvider().Tracer("noop"),
		}
		return globalTracer, nil
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = "offer-eligibility-api"
	}

	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.Endpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String("1.0.0"),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(res),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create tracer
	tracer := otel.Tracer(cfg.ServiceName)

	globalTracer = &Tracer{
		tracer: tracer,
	}

	return globalTracer, nil
}

// StartSpan starts a new span with the given name.
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// GetTracer returns the global tracer instance.
func GetTracer() *Tracer {
	if globalTracer == nil {
		// Return no-op tracer if not initialized
		return &Tracer{
			tracer: trace.NewNoopTracerProvider().Tracer("noop"),
		}
	}
	return globalTracer
}

// Shutdown shuts down the tracer provider.
func Shutdown(ctx context.Context) error {
	if tp, ok := otel.GetTracerProvider().(*tracesdk.TracerProvider); ok {
		return tp.Shutdown(ctx)
	}
	return nil
}
