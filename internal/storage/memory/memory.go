package memory

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dubstepenjoyer/cachmere/internal/cache"
)

type item struct {
	entry     cache.Entry
	expiresAt time.Time
}

type Storage struct {
	mu       sync.Mutex
	entries  map[cache.Key]item
	capacity int64
}

var _ cache.Storage = (*Storage)(nil)

func New(capacity int64) (*Storage, error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("capacity must be greater then 0")
	}
	return &Storage{
		entries:  make(map[cache.Key]item, capacity),
		capacity: capacity,
	}, nil
}

func (s *Storage) Get(ctx context.Context, key cache.Key) (cache.Entry, bool, error) {
	if err := ctx.Err(); err != nil {
		return cache.Entry{}, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.entries[key]
	if !ok {
		return cache.Entry{}, false, nil
	}

	if !time.Now().Before(stored.expiresAt) {
		delete(s.entries, key)
		return cache.Entry{}, false, nil
	}

	entry := stored.entry
	entry.Value = bytes.Clone(entry.Value)

	return entry, true, nil
}

func (s *Storage) Set(ctx context.Context, key cache.Key, entry cache.Entry, expiresAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !time.Now().Before(expiresAt) {
		delete(s.entries, key)
		return nil
	}

	entry.Value = bytes.Clone(entry.Value)

	s.entries[key] = item{
		entry:     entry,
		expiresAt: expiresAt,
	}

	return nil
}

func (s *Storage) Delete(ctx context.Context, key cache.Key) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.entries, key)
	return nil
}
