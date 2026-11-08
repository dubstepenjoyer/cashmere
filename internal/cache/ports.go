package cache

import "context"

type Provider interface {
	Fetch(ctx context.Context, key string) ([]byte, error)
}
