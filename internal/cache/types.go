package cache

import (
	"time"
)

type Key struct {
	Provider string
	ID       string
}

type Entry struct {
	Value     []byte
	FetchedAt time.Time
	NotFound  bool
}

type Result struct {
	Value     []byte
	FetchedAt time.Time
	Stale     bool
}

type Policy struct {
	FreshTTL    time.Duration
	MaxAge      time.Duration
	NegativeTTL time.Duration
	LoadTimeout time.Duration
}
