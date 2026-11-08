package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"
)

type Storage interface {
	Get(ctx context.Context, key Key) (Entry, bool, error)
	Set(ctx context.Context, key Key, entry Entry, expiresAt time.Time) error
	Delete(ctx context.Context, key Key) error
}

type ProviderBinding struct {
	Provider Provider
	Policy   Policy
}

type Service struct {
	providers map[string]ProviderBinding
	storage   Storage
	loads     singleflight.Group
}

func New(storage Storage, providers map[string]ProviderBinding) (*Service, error) {
	if storage == nil {
		return nil, fmt.Errorf("storage is required")
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("at least one provider is required")
	}

	registered := make(map[string]ProviderBinding, len(providers))

	for name, binding := range providers {
		if name == "" || binding.Provider == nil {
			return nil, fmt.Errorf("provider %q: name and implementation are required", name)
		}

		policy := binding.Policy

		if policy.MaxAge <= 0 || policy.FreshTTL <= 0 || policy.FreshTTL > policy.MaxAge {
			return nil, fmt.Errorf("provider %q: invalid FreshTTL or MaxAge", name)
		}

		if policy.NegativeTTL <= 0 || policy.LoadTimeout <= 0 {
			return nil, fmt.Errorf("provider %q: NegativeTTL and LoadTimeout must be positive", name)
		}

		registered[name] = binding
	}

	return &Service{
		storage:   storage,
		providers: registered,
	}, nil
}

func (s *Service) Set(ctx context.Context, key Key, entry Entry) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if key.Provider == "" || key.ID == "" {
		return ErrInvalidKey
	}

	binding, ok := s.providers[key.Provider]
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownProvider, key.Provider)
	}
	if entry.FetchedAt.IsZero() {
		return fmt.Errorf("%w: FetchedAt is required", ErrInvalidEntry)
	}

	if entry.NotFound && len(entry.Value) > 0 {
		return fmt.Errorf("%w: a negative entry must contain a value", ErrInvalidEntry)
	}

	ttl := binding.Policy.MaxAge
	if entry.NotFound {
		ttl = binding.Policy.NegativeTTL
	}

	expiresAt := entry.FetchedAt.Add(ttl)

	if err := s.storage.Set(ctx, key, entry, expiresAt); err != nil {
		return fmt.Errorf("store cache entry: %w", err)
	}

	return nil
}

func (s *Service) Get(ctx context.Context, key Key) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	if key.Provider == "" || key.ID == "" {
		return Result{}, ErrInvalidKey
	}

	binding, ok := s.providers[key.Provider]
	if !ok {
		return Result{}, fmt.Errorf("%w: %q", ErrUnknownProvider, key.Provider)
	}

	policy := binding.Policy

	cached, exists, err := s.storage.Get(ctx, key)
	if err != nil {
		return Result{}, fmt.Errorf("get cache entry: %w", err)
	}

	now := time.Now()

	if exists {
		if cached.NotFound {
			negativeUntil := cached.FetchedAt.Add(policy.NegativeTTL)

			if now.Before(negativeUntil) {
				return Result{}, ErrNotFound
			}
		} else {
			freshUntil := cached.FetchedAt.Add(policy.FreshTTL)

			if now.Before(freshUntil) {
				return Result{
					Value:     cached.Value,
					FetchedAt: cached.FetchedAt,
					Stale:     false,
				}, nil
			}
		}
	}

	loadCtx, cancel := context.WithTimeout(ctx, policy.LoadTimeout)

	value, fetchErr := binding.Provider.Fetch(loadCtx, key.ID)

	fetchedAt := time.Now()

	cancel()

	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	if fetchErr != nil && !errors.Is(fetchErr, ErrNotFound) {
		unavailable := errors.Is(fetchErr, ErrUnavailable) || errors.Is(fetchErr, context.DeadlineExceeded)

		if unavailable && exists && !cached.NotFound {
			expiresAt := cached.FetchedAt.Add(policy.MaxAge)

			if time.Now().Before(expiresAt) {
				return Result{
					Value:     cached.Value,
					FetchedAt: cached.FetchedAt,
					Stale:     true,
				}, nil
			}
		}

		return Result{}, fmt.Errorf("fetch from provider %q: %w", key.Provider, fetchErr)
	}

	loaded := Entry{
		FetchedAt: fetchedAt,
		NotFound:  errors.Is(fetchErr, ErrNotFound),
	}

	if loaded.NotFound {
		loaded.Value = value
	}

	if err := s.Set(ctx, key, loaded); err != nil {
		if !errors.Is(err, ErrStorageFull) {
			return Result{}, err
		}
	}

	if loaded.NotFound {
		return Result{}, ErrNotFound
	}

	return Result{
		Value:     loaded.Value,
		FetchedAt: loaded.FetchedAt,
		Stale:     false,
	}, nil
}

func (s *Service) Delete(ctx context.Context, key Key) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if key.Provider == "" || key.ID == "" {
		return ErrInvalidKey
	}

	_, ok := s.providers[key.Provider]
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownProvider, key.Provider)
	}

	if err := s.storage.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete cache entry: %w", err)
	}

	return nil
}
