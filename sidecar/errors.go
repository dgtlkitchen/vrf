package sidecar

import "errors"

var (
	ErrRoundNotAvailable  = errors.New("sidecar: round not available")
	ErrServiceUnavailable = errors.New("sidecar: service unavailable")
)
