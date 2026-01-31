package observability

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

type Config struct {
	Enabled      bool
	ServiceName  string
	OTLPEndpoint string
	SampleRatio  float64
}

func Setup(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}
	if cfg.OTLPEndpoint == "" {
		return nil, errors.New("OTLP endpoint is empty")
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "webapp"
	}
	if cfg.SampleRatio <= 0 {
		cfg.SampleRatio = 1.0
	}

	endpoint := strings.TrimSpace(cfg.OTLPEndpoint)
	var opts []otlptracehttp.Option
	opts = append(opts, otlptracehttp.WithTimeout(5*time.Second))

	if parsed, err := url.Parse(endpoint); err == nil && parsed.Host != "" {
		opts = append(opts, otlptracehttp.WithEndpoint(parsed.Host))
		if strings.EqualFold(parsed.Scheme, "http") {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if parsed.Path != "" && parsed.Path != "/" {
			opts = append(opts, otlptracehttp.WithURLPath(parsed.Path))
		}
	} else {
		// Accept plain "host:port" too.
		opts = append(opts, otlptracehttp.WithEndpoint(endpoint))
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func WrapHandler(next http.Handler, enabled bool) http.Handler {
	if !enabled {
		return next
	}
	return otelhttp.NewHandler(next, "http",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)
}
