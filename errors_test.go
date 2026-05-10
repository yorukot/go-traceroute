package traceroute

import (
	"context"
	"errors"
	"testing"
)

func TestPermissionErrorMatchesSentinel(t *testing.T) {
	err := &PermissionError{
		Operation: "icmp",
		Method:    MethodICMP,
		Cause:     errors.New("operation not permitted"),
	}

	if !errors.Is(err, ErrPermission) {
		t.Fatal("PermissionError should match ErrPermission")
	}
}

func TestTraceReturnsUnsupportedUntilBackendExists(t *testing.T) {
	tr, err := New(Options{Method: MethodICMP})
	if err != nil {
		t.Fatal(err)
	}

	result, err := tr.Trace(context.Background(), "127.0.0.1")
	if result != nil {
		t.Fatalf("result = %#v, want nil", result)
	}
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Trace() error = %v, want ErrUnsupported", err)
	}
}

func TestTraceStreamEmitsUnsupportedError(t *testing.T) {
	tr, err := New(Options{Method: MethodICMP})
	if err != nil {
		t.Fatal(err)
	}

	events, err := tr.TraceStream(context.Background(), "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	var sawError bool
	for event := range events {
		if event.Kind == EventError {
			sawError = true
			if !errors.Is(event.Error, ErrUnsupported) {
				t.Fatalf("EventError = %v, want ErrUnsupported", event.Error)
			}
		}
	}

	if !sawError {
		t.Fatal("missing EventError")
	}
}
