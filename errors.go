package traceroute

import "errors"

var (
	ErrPermission  = errors.New("traceroute: permission denied")
	ErrUnsupported = errors.New("traceroute: unsupported platform or method")
	ErrNoAddress   = errors.New("traceroute: no usable destination address")
	ErrTimeout     = errors.New("traceroute: timeout")
)

// PermissionError reports a failed privileged operation.
type PermissionError struct {
	Operation string
	Method    Method
	Cause     error
}

func (e *PermissionError) Error() string {
	return "traceroute: permission denied while opening " + e.Operation
}

func (e *PermissionError) Unwrap() error {
	return e.Cause
}

func (e *PermissionError) Is(target error) bool {
	return target == ErrPermission
}
