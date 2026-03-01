package client

import (
	"context"
	"testing"

	wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	"google.golang.org/grpc/metadata"
)

// TestCreateNATSRequestHeaders_ModifiersRunBeforeRead verifies that context
// modifiers update metadata BEFORE it is read into NATS headers.
// Previously, metadata was captured before modifiers ran, so modifier-added
// headers (correlation ID, span) were lost.
func TestCreateNATSRequestHeaders_ModifiersRunBeforeRead(t *testing.T) {
	client := &NATSClient{
		ctxModifiers: []ContextModifier{
			// Modifier that adds a correlation ID to outgoing metadata
			func(ctx context.Context, subject string) (context.Context, error) {
				md, ok := metadata.FromOutgoingContext(ctx)
				if !ok {
					md = metadata.Pairs()
				}
				md.Set("x-test-header", "test-value")
				return metadata.NewOutgoingContext(ctx, md), nil
			},
		},
	}

	ctx := context.Background()
	headers, err := client.createNATSRequestHeaders(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val := headers.Get("x-test-header")
	if val == "" {
		t.Fatal("expected 'x-test-header' to be propagated to NATS headers after modifier ran")
	}
	if val != "test-value" {
		t.Fatalf("expected 'test-value', got '%s'", val)
	}
}

// TestCreateNATSRequestHeaders_EnsureOutboundSpanTracing verifies the built-in
// EnsureOutboundSpanTracing modifier properly sets correlation-ID and span headers.
func TestCreateNATSRequestHeaders_EnsureOutboundSpanTracing(t *testing.T) {
	client := &NATSClient{
		ctxModifiers: []ContextModifier{
			EnsureOutboundSpanTracing,
		},
	}

	ctx := context.Background()
	headers, err := client.createNATSRequestHeaders(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Correlation ID should be set
	if headers.Get(wellknown.XCorrelationIDName) == "" {
		t.Fatal("expected correlation ID header to be set")
	}

	// Span should be set
	if headers.Get(wellknown.XSpanName) == "" {
		t.Fatal("expected span header to be set")
	}
}

// TestCreateNATSRequestHeaders_ExistingMetadataPreserved verifies that metadata
// already on the context is preserved through modifiers.
func TestCreateNATSRequestHeaders_ExistingMetadataPreserved(t *testing.T) {
	client := &NATSClient{
		ctxModifiers: []ContextModifier{
			// A modifier that adds new data
			func(ctx context.Context, subject string) (context.Context, error) {
				md, ok := metadata.FromOutgoingContext(ctx)
				if !ok {
					md = metadata.Pairs()
				}
				md.Set("new-header", "new-value")
				return metadata.NewOutgoingContext(ctx, md), nil
			},
		},
	}

	// Pre-set some metadata on context
	md := metadata.Pairs("existing-header", "existing-value")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	headers, err := client.createNATSRequestHeaders(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both headers should be present
	if headers.Get("existing-header") != "existing-value" {
		t.Fatal("expected existing metadata to be preserved")
	}
	if headers.Get("new-header") != "new-value" {
		t.Fatal("expected new header from modifier to be present")
	}
}
