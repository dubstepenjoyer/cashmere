package internal

import (
	"context"

	"github.com/dubstepenjoyer/cashmere/internal/cache"
)

type Internal struct {
}

var _ cache.Provider = (*Internal)(nil)

func (i *Internal) Fetch(ctx context.Context, key string) ([]byte, error) {
	emptyResult := make([]byte, 0)
	return emptyResult, cache.ErrNotFound
}
