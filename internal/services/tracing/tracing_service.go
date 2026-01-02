package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Service provides OpenTelemetry tracing functionality
type Service struct {
	tracer trace.Tracer
}

// NewService creates a new tracing service
func NewService(serviceName string) *Service {
	tracer := otel.Tracer(serviceName)
	return &Service{
		tracer: tracer,
	}
}

// StartSpan starts a new span
func (s *Service) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return s.tracer.Start(ctx, name, opts...)
}

// StartRootSpan starts a root span (no parent)
func (s *Service) StartRootSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// Create a new context without parent span
	newCtx := context.Background()
	return s.tracer.Start(newCtx, name, opts...)
}

// StartChildSpan starts a child span from parent context
func (s *Service) StartChildSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return s.tracer.Start(ctx, name, opts...)
}

// AddSpanAttributes adds attributes to a span
func AddSpanAttributes(span trace.Span, attrs map[string]string) {
	attributes := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		attributes = append(attributes, attribute.String(k, v))
	}
	span.SetAttributes(attributes...)
}

// RecordError records an error on a span
func RecordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// ExtractTraceID extracts trace ID from context
func ExtractTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// ExtractSpanID extracts span ID from context
func ExtractSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// SpanFromContext gets span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// WithSpan wraps a function with span creation
func WithSpan(ctx context.Context, tracer trace.Tracer, name string, fn func(context.Context) error) error {
	ctx, span := tracer.Start(ctx, name)
	defer span.End()
	return fn(ctx)
}

// WithSpanAndAttributes wraps a function with span creation and attributes
func WithSpanAndAttributes(ctx context.Context, tracer trace.Tracer, name string, attrs map[string]string, fn func(context.Context) error) error {
	ctx, span := tracer.Start(ctx, name)
	defer span.End()

	AddSpanAttributes(span, attrs)
	return fn(ctx)
}
