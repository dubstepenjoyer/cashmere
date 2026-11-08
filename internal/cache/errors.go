package cache

import "errors"

var (
	ErrInvalidKey      = errors.New("invalid cache key")
	ErrUnknownProvider = errors.New("unknown provider")
	ErrInvalidEntry    = errors.New("invalid cache entry")
	ErrStorageFull     = errors.New("cache storage is full")
	ErrNotFound        = errors.New("key not found")
	ErrUnavailable     = errors.New("provider unavailable")
)
