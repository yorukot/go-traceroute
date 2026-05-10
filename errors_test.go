package traceroute

import (
	"errors"
	"testing"
)

func TestPermissionErrorMatchesSentinel(t *testing.T) {
	err := &PermissionError{
		Operation: "icmp",
		Cause:     errors.New("operation not permitted"),
	}

	if !errors.Is(err, ErrPermission) {
		t.Fatal("PermissionError should match ErrPermission")
	}
}
