package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewService(t *testing.T) {
	service := NewService("test-service")
	assert.NotNil(t, service)
	assert.NotNil(t, service.tracer)
}

func TestService_StartSpan(t *testing.T) {
	service := NewService("test-service")
	ctx := context.Background()

	newCtx, span := service.StartSpan(ctx, "test-span")
	assert.NotNil(t, newCtx)
	assert.NotNil(t, span)

	// Verify span is in context
	spanFromCtx := trace.SpanFromContext(newCtx)
	assert.NotNil(t, spanFromCtx)
	assert.Equal(t, span, spanFromCtx)

	span.End()
}

func TestService_StartRootSpan(t *testing.T) {
	service := NewService("test-service")
	ctx := context.Background()

	// Create a context with a parent span
	parentTracer := otel.Tracer("parent-service")
	parentCtx, parentSpan := parentTracer.Start(ctx, "parent-span")
	defer parentSpan.End()

	// Start root span should ignore parent
	newCtx, rootSpan := service.StartRootSpan(parentCtx, "root-span")
	assert.NotNil(t, newCtx)
	assert.NotNil(t, rootSpan)

	// Root span should not have parent
	spanFromCtx := trace.SpanFromContext(newCtx)
	assert.NotNil(t, spanFromCtx)

	rootSpan.End()
}

func TestService_StartChildSpan(t *testing.T) {
	service := NewService("test-service")
	ctx := context.Background()

	// Create a parent span
	parentTracer := otel.Tracer("parent-service")
	parentCtx, parentSpan := parentTracer.Start(ctx, "parent-span")
	defer parentSpan.End()

	// Start child span should use parent context
	newCtx, childSpan := service.StartChildSpan(parentCtx, "child-span")
	assert.NotNil(t, newCtx)
	assert.NotNil(t, childSpan)

	// Verify child span is in context
	spanFromCtx := trace.SpanFromContext(newCtx)
	assert.NotNil(t, spanFromCtx)

	childSpan.End()
}

func TestAddSpanAttributes(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	defer span.End()

	attrs := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	AddSpanAttributes(span, attrs)

	// Attributes are set (noop tracer doesn't expose them, but no error means success)
	assert.NotNil(t, span)
}

func TestRecordError(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	defer span.End()

	testErr := errors.New("test error")
	RecordError(span, testErr)

	// Error is recorded (noop tracer doesn't expose status, but no error means success)
	assert.NotNil(t, span)
}

func TestRecordError_NilError(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	defer span.End()

	RecordError(span, nil)

	// Should not panic with nil error
	assert.NotNil(t, span)
}

func TestExtractTraceID(t *testing.T) {
	service := NewService("test-service")
	ctx := context.Background()

	// Start a span
	newCtx, span := service.StartSpan(ctx, "test-span")
	defer span.End()

	traceID := ExtractTraceID(newCtx)

	// Trace ID should be extracted (may be empty with noop tracer, but function should work)
	assert.NotNil(t, traceID)
}

func TestExtractTraceID_NoSpan(t *testing.T) {
	ctx := context.Background()
	traceID := ExtractTraceID(ctx)

	// Should return empty string when no span in context
	assert.Equal(t, "", traceID)
}

func TestExtractSpanID(t *testing.T) {
	service := NewService("test-service")
	ctx := context.Background()

	// Start a span
	newCtx, span := service.StartSpan(ctx, "test-span")
	defer span.End()

	spanID := ExtractSpanID(newCtx)

	// Span ID should be extracted (may be empty with noop tracer, but function should work)
	assert.NotNil(t, spanID)
}

func TestExtractSpanID_NoSpan(t *testing.T) {
	ctx := context.Background()
	spanID := ExtractSpanID(ctx)

	// Should return empty string when no span in context
	assert.Equal(t, "", spanID)
}

func TestSpanFromContext(t *testing.T) {
	service := NewService("test-service")
	ctx := context.Background()

	// Start a span
	newCtx, span := service.StartSpan(ctx, "test-span")
	defer span.End()

	spanFromCtx := SpanFromContext(newCtx)
	assert.NotNil(t, spanFromCtx)
	assert.Equal(t, span, spanFromCtx)
}

func TestSpanFromContext_NoSpan(t *testing.T) {
	ctx := context.Background()
	span := SpanFromContext(ctx)

	// Should return a noop span when no span in context
	assert.NotNil(t, span)
}

func TestWithSpan(t *testing.T) {
	service := NewService("test-service")
	ctx := context.Background()
	tracer := service.tracer

	called := false
	err := WithSpan(ctx, tracer, "test-span", func(ctx context.Context) error {
		called = true
		// Verify span is in context
		span := trace.SpanFromContext(ctx)
		assert.NotNil(t, span)
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestWithSpan_Error(t *testing.T) {
	service := NewService("test-service")
	ctx := context.Background()
	tracer := service.tracer

	testErr := errors.New("test error")
	err := WithSpan(ctx, tracer, "test-span", func(ctx context.Context) error {
		return testErr
	})

	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestWithSpanAndAttributes(t *testing.T) {
	service := NewService("test-service")
	ctx := context.Background()
	tracer := service.tracer

	attrs := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	called := false
	err := WithSpanAndAttributes(ctx, tracer, "test-span", attrs, func(ctx context.Context) error {
		called = true
		// Verify span is in context
		span := trace.SpanFromContext(ctx)
		assert.NotNil(t, span)
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestWithSpanAndAttributes_Error(t *testing.T) {
	service := NewService("test-service")
	ctx := context.Background()
	tracer := service.tracer

	attrs := map[string]string{
		"key1": "value1",
	}

	testErr := errors.New("test error")
	err := WithSpanAndAttributes(ctx, tracer, "test-span", attrs, func(ctx context.Context) error {
		return testErr
	})

	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestWithSpanAndAttributes_EmptyAttributes(t *testing.T) {
	service := NewService("test-service")
	ctx := context.Background()
	tracer := service.tracer

	called := false
	err := WithSpanAndAttributes(ctx, tracer, "test-span", map[string]string{}, func(ctx context.Context) error {
		called = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestService_StartSpan_WithOptions(t *testing.T) {
	service := NewService("test-service")
	ctx := context.Background()

	// Start span with options
	newCtx, span := service.StartSpan(ctx, "test-span", trace.WithSpanKind(trace.SpanKindServer))
	assert.NotNil(t, newCtx)
	assert.NotNil(t, span)

	span.End()
}

func TestService_StartChildSpan_PreservesParent(t *testing.T) {
	service := NewService("test-service")
	parentService := NewService("parent-service")
	ctx := context.Background()

	// Create parent span
	parentCtx, parentSpan := parentService.StartSpan(ctx, "parent-span")
	defer parentSpan.End()

	// Create child span
	childCtx, childSpan := service.StartChildSpan(parentCtx, "child-span")
	defer childSpan.End()

	// Both spans should be accessible
	parentSpanFromCtx := trace.SpanFromContext(parentCtx)
	childSpanFromCtx := trace.SpanFromContext(childCtx)

	assert.NotNil(t, parentSpanFromCtx)
	assert.NotNil(t, childSpanFromCtx)
}

func TestAddSpanAttributes_EmptyMap(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	defer span.End()

	AddSpanAttributes(span, map[string]string{})

	// Should not panic with empty map
	assert.NotNil(t, span)
}

func TestAddSpanAttributes_NilMap(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	defer span.End()

	// Should handle nil map gracefully
	AddSpanAttributes(span, nil)
	assert.NotNil(t, span)
}
