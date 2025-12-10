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

type Config struct {
	Enabled     bool
	Endpoint    string
	ServiceName string
	Environment string
}

type Tracer struct {
	tracer trace.Tracer
}

var globalTracer *Tracer

func InitTracing(cfg Config) (*Tracer, error) {
	if !cfg.Enabled {
		globalTracer = &Tracer{
			tracer: trace.NewNoopTracerProvider().Tracer("noop"),
		}
		return globalTracer, nil
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = "offer-eligibility-api"
	}

	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.Endpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

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

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := otel.Tracer(cfg.ServiceName)

	globalTracer = &Tracer{
		tracer: tracer,
	}

	return globalTracer, nil
}

func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

func GetTracer() *Tracer {
	if globalTracer == nil {
		return &Tracer{
			tracer: trace.NewNoopTracerProvider().Tracer("noop"),
		}
	}
	return globalTracer
}

func Shutdown(ctx context.Context) error {
	if tp, ok := otel.GetTracerProvider().(*tracesdk.TracerProvider); ok {
		return tp.Shutdown(ctx)
	}
	return nil
}
