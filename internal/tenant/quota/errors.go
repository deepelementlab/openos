package quota

import "errors"

// ErrResourceExhausted signals hard quota / cgroup exhaustion (maps to HTTP 429 / gRPC RESOURCE_EXHAUSTED).
var ErrResourceExhausted = errors.New("RESOURCE_EXHAUSTED")
