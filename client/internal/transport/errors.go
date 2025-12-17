package transport

import "errors"

var (
	// ErrClientNotInitialized возникает, когда gRPC клиент не был инициализирован.
	ErrClientNotInitialized = errors.New("gRPC client not initialized")
)
